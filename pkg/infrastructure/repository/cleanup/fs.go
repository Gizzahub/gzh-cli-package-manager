// Package cleanup provides infrastructure implementations for cleanup operations.
package cleanup

import (
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// FileSystem abstracts filesystem operations used by cache scan/clean.
// Tests inject an in-memory or temp-dir implementation.
type FileSystem interface {
	// Stat returns file info or an error if the path does not exist.
	Stat(path string) (fs.FileInfo, error)
	// WalkDir walks the directory tree rooted at root.
	WalkDir(root string, fn fs.WalkDirFunc) error
	// RemoveAll removes path and any children it contains.
	RemoveAll(path string) error
	// ReadDir lists directory entries.
	ReadDir(path string) ([]fs.DirEntry, error)
}

// OSFileSystem is the production FileSystem backed by the os package.
type OSFileSystem struct{}

// NewOSFileSystem returns a FileSystem using the real OS.
func NewOSFileSystem() *OSFileSystem {
	return &OSFileSystem{}
}

// Stat implements FileSystem.
func (OSFileSystem) Stat(path string) (fs.FileInfo, error) {
	return os.Stat(path)
}

// WalkDir implements FileSystem.
func (OSFileSystem) WalkDir(root string, fn fs.WalkDirFunc) error {
	return filepath.WalkDir(root, fn)
}

// RemoveAll implements FileSystem.
func (OSFileSystem) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

// ReadDir implements FileSystem.
func (OSFileSystem) ReadDir(path string) ([]fs.DirEntry, error) {
	return os.ReadDir(path)
}

// Ensure OSFileSystem implements FileSystem.
var _ FileSystem = (*OSFileSystem)(nil)

// MapFileSystem is a minimal in-memory FS for unit tests.
// Paths are absolute-style keys; only files have sizes.
type MapFileSystem struct {
	// Files maps path -> size in bytes. Directories are implied by path prefixes
	// or listed in Dirs.
	Files map[string]int64
	// Dirs lists directory paths that exist (even if empty).
	Dirs map[string]bool
	// ModTimes optional per-path mtime.
	ModTimes map[string]time.Time
	// Removed tracks RemoveAll calls (path -> true).
	Removed map[string]bool
}

// NewMapFileSystem creates an empty MapFileSystem.
func NewMapFileSystem() *MapFileSystem {
	return &MapFileSystem{
		Files:    make(map[string]int64),
		Dirs:     make(map[string]bool),
		ModTimes: make(map[string]time.Time),
		Removed:  make(map[string]bool),
	}
}

// AddFile registers a file and ensures parent directories exist.
func (m *MapFileSystem) AddFile(path string, size int64, mod time.Time) {
	m.Files[path] = size
	if !mod.IsZero() {
		m.ModTimes[path] = mod
	}
	dir := filepath.Dir(path)
	for dir != "." && dir != "/" && dir != "" {
		m.Dirs[dir] = true
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
}

// AddDir registers an empty directory.
func (m *MapFileSystem) AddDir(path string) {
	m.Dirs[path] = true
}

type mapFileInfo struct {
	modTime time.Time
	name    string
	size    int64
	mode    fs.FileMode
	isDir   bool
}

func (i mapFileInfo) Name() string       { return i.name }
func (i mapFileInfo) Size() int64        { return i.size }
func (i mapFileInfo) Mode() fs.FileMode  { return i.mode }
func (i mapFileInfo) ModTime() time.Time { return i.modTime }
func (i mapFileInfo) IsDir() bool        { return i.isDir }
func (i mapFileInfo) Sys() any           { return nil }

// Stat implements FileSystem.
func (m *MapFileSystem) Stat(path string) (fs.FileInfo, error) {
	if size, ok := m.Files[path]; ok {
		mt := m.ModTimes[path]
		return mapFileInfo{name: filepath.Base(path), size: size, mode: 0o644, modTime: mt, isDir: false}, nil
	}
	if m.Dirs[path] {
		return mapFileInfo{name: filepath.Base(path), mode: fs.ModeDir | 0o755, isDir: true}, nil
	}
	// Prefix check: path is a dir if any file lives under it.
	prefix := path + string(filepath.Separator)
	for f := range m.Files {
		if len(f) > len(path) && (f == path || hasPathPrefix(f, prefix)) {
			return mapFileInfo{name: filepath.Base(path), mode: fs.ModeDir | 0o755, isDir: true}, nil
		}
	}
	return nil, os.ErrNotExist
}

// WalkDir implements FileSystem (depth-first over known paths under root).
func (m *MapFileSystem) WalkDir(root string, fn fs.WalkDirFunc) error {
	info, err := m.Stat(root)
	if err != nil {
		return fn(root, nil, err)
	}
	rootEntry := fs.FileInfoToDirEntry(info)
	if err := fn(root, rootEntry, nil); err != nil {
		//nolint:errorlint // filepath.WalkDir treats only the exact sentinel as SkipDir.
		if err == fs.SkipDir {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return nil
	}

	// Collect unique child paths one level under each visited dir via full listing.
	type item struct {
		path  string
		isDir bool
		size  int64
	}
	seen := map[string]bool{root: true}
	queue := []string{root}

	for len(queue) > 0 {
		dir := queue[0]
		queue = queue[1:]
		children := map[string]item{}

		prefix := dir + string(filepath.Separator)
		for f, size := range m.Files {
			if !hasPathPrefix(f, prefix) {
				continue
			}
			rel := f[len(prefix):]
			first, rest, found := splitFirst(rel)
			childPath := filepath.Join(dir, first)
			if found && rest != "" {
				// intermediate dir
				children[childPath] = item{path: childPath, isDir: true}
			} else {
				children[childPath] = item{path: childPath, isDir: false, size: size}
			}
		}
		for d := range m.Dirs {
			if !hasPathPrefix(d, prefix) {
				continue
			}
			rel := d[len(prefix):]
			first, rest, found := splitFirst(rel)
			childPath := filepath.Join(dir, first)
			if found && rest != "" {
				children[childPath] = item{path: childPath, isDir: true}
			} else if first != "" {
				children[childPath] = item{path: childPath, isDir: true}
			}
		}

		for _, child := range children {
			if seen[child.path] {
				continue
			}
			seen[child.path] = true
			var fi fs.FileInfo
			if child.isDir {
				fi = mapFileInfo{name: filepath.Base(child.path), mode: fs.ModeDir | 0o755, isDir: true}
			} else {
				mt := m.ModTimes[child.path]
				fi = mapFileInfo{name: filepath.Base(child.path), size: child.size, mode: 0o644, modTime: mt, isDir: false}
			}
			entry := fs.FileInfoToDirEntry(fi)
			if err := fn(child.path, entry, nil); err != nil {
				//nolint:errorlint // filepath.WalkDir treats only the exact sentinel as SkipDir.
				if err == fs.SkipDir {
					continue
				}
				return err
			}
			if child.isDir {
				queue = append(queue, child.path)
			}
		}
	}
	return nil
}

// RemoveAll implements FileSystem.
func (m *MapFileSystem) RemoveAll(path string) error {
	if m.Removed == nil {
		m.Removed = make(map[string]bool)
	}
	m.Removed[path] = true
	prefix := path + string(filepath.Separator)
	for f := range m.Files {
		if f == path || hasPathPrefix(f, prefix) {
			delete(m.Files, f)
		}
	}
	for d := range m.Dirs {
		if d == path || hasPathPrefix(d, prefix) {
			delete(m.Dirs, d)
		}
	}
	return nil
}

// ReadDir implements FileSystem.
func (m *MapFileSystem) ReadDir(path string) ([]fs.DirEntry, error) {
	if _, err := m.Stat(path); err != nil {
		return nil, err
	}
	var entries []fs.DirEntry
	err := m.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == path {
			return nil
		}
		// only immediate children
		if filepath.Dir(p) != path {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		entries = append(entries, d)
		if d.IsDir() {
			return fs.SkipDir
		}
		return nil
	})
	return entries, err
}

func hasPathPrefix(path, prefix string) bool {
	return len(path) >= len(prefix) && path[:len(prefix)] == prefix
}

func splitFirst(rel string) (first, rest string, found bool) {
	for i := 0; i < len(rel); i++ {
		if rel[i] == '/' || rel[i] == filepath.Separator {
			return rel[:i], rel[i+1:], true
		}
	}
	return rel, "", false
}

// Ensure MapFileSystem implements FileSystem.
var _ FileSystem = (*MapFileSystem)(nil)
