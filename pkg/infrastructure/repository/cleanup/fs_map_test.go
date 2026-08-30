package cleanup

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const testCacheRoot = "/cache"

func TestMapFileSystemWalkDirPreservesWrappedSkipDir(t *testing.T) {
	t.Parallel()

	fileSystem := NewMapFileSystem()
	root := testCacheRoot
	fileSystem.AddFile(filepath.Join(root, "child", "package.tgz"), 1, time.Now())

	want := fmt.Errorf("wrapped skip: %w", fs.SkipDir)
	err := fileSystem.WalkDir(root, func(path string, _ fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == filepath.Join(root, "child") {
			return want
		}
		return nil
	})
	if !errors.Is(err, fs.SkipDir) {
		t.Fatalf("WalkDir() error = %v, want wrapped fs.SkipDir", err)
	}
}

func TestMapFileSystemWalkDirSkipsDescendantsForExactSkipDir(t *testing.T) {
	t.Parallel()

	fileSystem := NewMapFileSystem()
	root := testCacheRoot
	child := filepath.Join(root, "child")
	descendant := filepath.Join(child, "package.tgz")
	fileSystem.AddFile(descendant, 1, time.Now())

	visitedDescendant := false
	err := fileSystem.WalkDir(root, func(path string, _ fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == child {
			return fs.SkipDir
		}
		if path == descendant {
			visitedDescendant = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir() error = %v, want nil", err)
	}
	if visitedDescendant {
		t.Fatal("WalkDir() visited a descendant after exact fs.SkipDir")
	}
}

func TestMapFileSystemWalkDirPassesMissingRootErrorToCallback(t *testing.T) {
	t.Parallel()

	fileSystem := NewMapFileSystem()
	root := "/missing"
	callbackCalls := 0
	err := fileSystem.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		callbackCalls++
		if path != root || entry != nil {
			t.Errorf("callback = (%q, %#v), want (%q, nil)", path, entry, root)
		}
		if !errors.Is(walkErr, fs.ErrNotExist) {
			t.Errorf("callback error = %v, want fs.ErrNotExist", walkErr)
		}
		return walkErr
	})
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("WalkDir() error = %v, want fs.ErrNotExist", err)
	}
	if callbackCalls != 1 {
		t.Errorf("callback calls = %d, want 1", callbackCalls)
	}
}

func TestMapFileSystemWalkDirRootExactSkipDirStopsWalk(t *testing.T) {
	t.Parallel()

	fileSystem := NewMapFileSystem()
	root := testCacheRoot
	fileSystem.AddFile(filepath.Join(root, "child", "package.tgz"), 1, time.Now())
	callbackCalls := 0
	err := fileSystem.WalkDir(root, func(_ string, _ fs.DirEntry, walkErr error) error {
		callbackCalls++
		if walkErr != nil {
			return walkErr
		}
		return fs.SkipDir
	})
	if err != nil {
		t.Fatalf("WalkDir() error = %v, want nil", err)
	}
	if callbackCalls != 1 {
		t.Errorf("callback calls = %d, want 1", callbackCalls)
	}
}

func TestMapFileSystemWalkDirPropagatesSkipAll(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name       string
		stopAtRoot bool
		wantCalls  int
	}{
		{name: "root", stopAtRoot: true, wantCalls: 1},
		{name: "child", wantCalls: 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fileSystem := NewMapFileSystem()
			root := testCacheRoot
			child := filepath.Join(root, "child")
			fileSystem.AddFile(filepath.Join(child, "package.tgz"), 1, time.Now())
			callbackCalls := 0
			err := fileSystem.WalkDir(root, func(path string, _ fs.DirEntry, walkErr error) error {
				callbackCalls++
				if walkErr != nil {
					return walkErr
				}
				if (tt.stopAtRoot && path == root) || (!tt.stopAtRoot && path == child) {
					return fs.SkipAll
				}
				return nil
			})
			if !errors.Is(err, fs.SkipAll) {
				t.Fatalf("WalkDir() error = %v, want fs.SkipAll", err)
			}
			if callbackCalls != tt.wantCalls {
				t.Errorf("callback calls = %d, want %d", callbackCalls, tt.wantCalls)
			}
		})
	}
}

func TestMapFileSystemWalkDirVisitsKnownPathsWithoutMutation(t *testing.T) {
	t.Parallel()

	fileSystem := NewMapFileSystem()
	root := testCacheRoot
	topLevelFile := filepath.Join(root, "top.tgz")
	emptyDir := filepath.Join(root, "empty")
	nestedFile := filepath.Join(root, "nested", "package.tgz")
	modTime := time.Now().Add(-time.Hour)
	fileSystem.AddFile(topLevelFile, 7, modTime)
	fileSystem.AddDir(emptyDir)
	fileSystem.AddFile(nestedFile, 11, time.Now())
	filesBefore := maps.Clone(fileSystem.Files)
	dirsBefore := maps.Clone(fileSystem.Dirs)
	modTimesBefore := maps.Clone(fileSystem.ModTimes)
	removedBefore := maps.Clone(fileSystem.Removed)
	entries := map[string]fs.DirEntry{}

	err := fileSystem.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		entries[path] = entry
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir() unexpected error: %v", err)
	}
	for path, wantDir := range map[string]bool{
		root:                          true,
		topLevelFile:                  false,
		emptyDir:                      true,
		filepath.Join(root, "nested"): true,
		nestedFile:                    false,
	} {
		entry, ok := entries[path]
		if !ok || entry.IsDir() != wantDir {
			t.Errorf("entry[%q] = %#v, want directory=%t", path, entry, wantDir)
		}
	}
	if len(entries) != 5 {
		t.Errorf("visited entries = %d, want 5", len(entries))
	}
	topEntry, ok := entries[topLevelFile]
	if !ok {
		t.Fatalf("missing top-level file entry %q", topLevelFile)
	}
	info, err := topEntry.Info()
	if err != nil {
		t.Fatalf("top-level entry Info() error: %v", err)
	}
	if info.Size() != 7 || !info.ModTime().Equal(modTime) {
		t.Errorf("top-level entry info = size %d mod %s, want size 7 mod %s", info.Size(), info.ModTime(), modTime)
	}
	if !maps.Equal(fileSystem.Files, filesBefore) || !maps.Equal(fileSystem.Dirs, dirsBefore) || !maps.Equal(fileSystem.ModTimes, modTimesBefore) || !maps.Equal(fileSystem.Removed, removedBefore) {
		t.Error("WalkDir() mutated MapFileSystem state")
	}
}

func TestOSFileSystemMissingPathPreservesNotExistSemantics(t *testing.T) {
	t.Parallel()

	fileSystem := NewOSFileSystem()
	missingPath := filepath.Join(t.TempDir(), "missing")

	assertNotExist := func(name string, err error) {
		t.Helper()
		if !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("%s() error = %v, want errors.Is(err, fs.ErrNotExist)", name, err)
		}
		if !os.IsNotExist(err) {
			t.Fatalf("%s() error = %v, want os.IsNotExist(err)", name, err)
		}
	}

	_, err := fileSystem.Stat(missingPath)
	assertNotExist("Stat", err)

	err = fileSystem.WalkDir(missingPath, func(_ string, _ fs.DirEntry, walkErr error) error { return walkErr })
	assertNotExist("WalkDir", err)

	_, err = fileSystem.ReadDir(missingPath)
	assertNotExist("ReadDir", err)
}

func TestOSFileSystemRemoveAllMissingPathMatchesOS(t *testing.T) {
	t.Parallel()

	err := NewOSFileSystem().RemoveAll(filepath.Join(t.TempDir(), "missing"))
	if err != nil {
		t.Fatalf("RemoveAll() error = %v, want nil for a missing path", err)
	}
}
