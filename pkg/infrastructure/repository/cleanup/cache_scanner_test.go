package cleanup

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	domaincleanup "github.com/gizzahub/gzh-cli-package-manager/pkg/domain/cleanup"
)

type cacheRepositoryStub struct {
	updateErr    error
	updateCalls  int
	failOnUpdate int
}

type injectedWalkFileSystem struct {
	*MapFileSystem
	walkDir        func(string, fs.WalkDirFunc) error
	removeAllCalls int
}

func (f *injectedWalkFileSystem) WalkDir(root string, fn fs.WalkDirFunc) error {
	return f.walkDir(root, fn)
}

func (f *injectedWalkFileSystem) RemoveAll(path string) error {
	f.removeAllCalls++
	return f.MapFileSystem.RemoveAll(path)
}

type injectedCacheFileSystem struct {
	*MapFileSystem
	statErr        error
	readDirErr     error
	removeAllErr   error
	removeAllCalls int
}

func (f *injectedCacheFileSystem) Stat(path string) (fs.FileInfo, error) {
	if f.statErr != nil {
		return nil, f.statErr
	}
	return f.MapFileSystem.Stat(path)
}

func (f *injectedCacheFileSystem) ReadDir(path string) ([]fs.DirEntry, error) {
	if f.readDirErr != nil {
		return nil, f.readDirErr
	}
	return f.MapFileSystem.ReadDir(path)
}

func (f *injectedCacheFileSystem) RemoveAll(path string) error {
	f.removeAllCalls++
	if f.removeAllErr != nil {
		return f.removeAllErr
	}
	return f.MapFileSystem.RemoveAll(path)
}

type infoErrorDirEntry struct {
	err error
}

func (e infoErrorDirEntry) Name() string               { return "package.tgz" }
func (e infoErrorDirEntry) IsDir() bool                { return false }
func (e infoErrorDirEntry) Type() fs.FileMode          { return 0 }
func (e infoErrorDirEntry) Info() (fs.FileInfo, error) { return nil, e.err }

func (r *cacheRepositoryStub) GetInfo(context.Context, string) (*domaincleanup.CacheInfo, error) {
	return nil, nil
}

func (r *cacheRepositoryStub) ListAll(context.Context) ([]*domaincleanup.CacheInfo, error) {
	return nil, nil
}

func (r *cacheRepositoryStub) UpdateInfo(_ context.Context, _ *domaincleanup.CacheInfo) error {
	r.updateCalls++
	if r.updateCalls == r.failOnUpdate {
		return r.updateErr
	}
	return nil
}

func TestResolveCachePaths(t *testing.T) {
	t.Parallel()
	home := testHomeDir
	paths := ResolveCachePaths(home, "linux", nil, DefaultKnownCachePaths())

	if got := paths[testNPMManagerID]; got != filepath.Join(home, ".npm") {
		t.Fatalf("npm path: got %q", got)
	}
	if got := paths[testHomebrewManagerID]; got != filepath.Join(home, ".cache", "Homebrew") {
		t.Fatalf("homebrew linux: got %q", got)
	}
	if _, ok := paths["pnpm"]; !ok {
		// linux pnpm should resolve
		t.Fatalf("expected pnpm on linux")
	}

	// AbsEnv wins
	paths = ResolveCachePaths(home, "linux", func(k string) string {
		if k == "npm_config_cache" {
			return "/custom/npm-cache"
		}
		return ""
	}, DefaultKnownCachePaths())
	if paths[testNPMManagerID] != "/custom/npm-cache" {
		t.Fatalf("npm AbsEnv: got %q", paths[testNPMManagerID])
	}

	// darwin homebrew
	paths = ResolveCachePaths(home, "darwin", nil, DefaultKnownCachePaths())
	if paths[testHomebrewManagerID] != filepath.Join(home, "Library", "Caches", "Homebrew") {
		t.Fatalf("homebrew darwin: got %q", paths[testHomebrewManagerID])
	}
}

func TestCacheScanner_Scan(t *testing.T) {
	t.Parallel()
	home := testHomeDir
	npmCache := filepath.Join(home, ".npm")
	pipCache := filepath.Join(home, ".cache", "pip")

	fsys := NewMapFileSystem()
	now := time.Now()
	fsys.AddFile(filepath.Join(npmCache, "pkg-a.tgz"), 2*1024*1024, now.Add(-48*time.Hour))
	fsys.AddFile(filepath.Join(npmCache, "pkg-b.tgz"), 1*1024*1024, now)
	fsys.AddFile(filepath.Join(pipCache, "wheel.whl"), 512*1024, now)

	repo := NewCacheRepository()
	scanner := NewCacheScanner(
		WithFileSystem(fsys),
		WithHomeDir(home),
		WithGOOS("linux"),
		WithEnvLookup(func(string) string { return "" }),
		WithCacheRepository(repo),
	)

	results, err := scanner.Scan(context.Background(), "")
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(results) < 2 {
		t.Fatalf("expected at least npm+pip, got %d: %+v", len(results), results)
	}

	var npmInfo *struct {
		size  float64
		count int
	}
	for _, r := range results {
		if r.ManagerID == testNPMManagerID {
			npmInfo = &struct {
				size  float64
				count int
			}{r.TotalSizeMB, r.EntryCount}
			if r.EntryCount != 2 {
				t.Fatalf("npm entries: got %d want 2", r.EntryCount)
			}
			// ~3 MiB
			if r.TotalSizeMB < 2.9 || r.TotalSizeMB > 3.1 {
				t.Fatalf("npm size MB: got %f", r.TotalSizeMB)
			}
		}
	}
	if npmInfo == nil {
		t.Fatal("npm not in scan results")
	}

	// Repo was updated
	stored, err := repo.GetInfo(context.Background(), testNPMManagerID)
	if err != nil {
		t.Fatalf("GetInfo: %v", err)
	}
	if stored.EntryCount != 2 {
		t.Fatalf("repo npm EntryCount: %d", stored.EntryCount)
	}
}

func TestCacheScanner_ScanManagerFilter(t *testing.T) {
	t.Parallel()
	home := testHomeDir
	fsys := NewMapFileSystem()
	fsys.AddFile(filepath.Join(home, ".npm", "a"), 100, time.Now())
	fsys.AddFile(filepath.Join(home, ".cache", "pip", "b"), 100, time.Now())

	scanner := NewCacheScanner(
		WithFileSystem(fsys),
		WithHomeDir(home),
		WithGOOS("linux"),
		WithEnvLookup(func(string) string { return "" }),
		WithCacheRepository(NewCacheRepository()),
	)

	results, err := scanner.Scan(context.Background(), testNPMManagerID)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(results) != 1 || results[0].ManagerID != testNPMManagerID {
		t.Fatalf("filter failed: %+v", results)
	}
}

func TestCacheScanner_ScanWrapsStatErrorAndSkipsMissingRoot(t *testing.T) {
	t.Parallel()

	const (
		home      = testHomeDir
		managerID = testNPMManagerID
		relPath   = ".npm"
	)
	cachePath := filepath.Join(home, relPath)

	t.Run("stat error retains sentinel", func(t *testing.T) {
		wantErr := errors.New("stat unavailable")
		fileSystem := &injectedCacheFileSystem{
			MapFileSystem: NewMapFileSystem(),
			statErr:       wantErr,
		}
		scanner := NewCacheScanner(
			WithFileSystem(fileSystem),
			WithHomeDir(home),
			WithGOOS("linux"),
			WithEnvLookup(func(string) string { return "" }),
			WithKnownPaths([]KnownCachePath{{ManagerID: managerID, RelPath: relPath}}),
			WithCacheRepository(nil),
		)

		_, err := scanner.Scan(context.Background(), managerID)
		if !errors.Is(err, wantErr) {
			t.Fatalf("Scan() error = %v, want wrapped %v", err, wantErr)
		}
		if !strings.Contains(err.Error(), "stat cache path "+cachePath) {
			t.Fatalf("Scan() error = %q, want stat cache path context", err)
		}
	})

	t.Run("missing root is skipped", func(t *testing.T) {
		scanner := NewCacheScanner(
			WithFileSystem(NewMapFileSystem()),
			WithHomeDir(home),
			WithGOOS("linux"),
			WithEnvLookup(func(string) string { return "" }),
			WithKnownPaths([]KnownCachePath{{ManagerID: managerID, RelPath: relPath}}),
			WithCacheRepository(nil),
		)

		results, err := scanner.Scan(context.Background(), managerID)
		if err != nil {
			t.Fatalf("Scan() error = %v, want missing root to be skipped", err)
		}
		if len(results) != 0 {
			t.Fatalf("Scan() results = %+v, want no missing-root result", results)
		}
	})
}

func TestCacheScanner_PropagatesWalkErrorsAndPreventsClean(t *testing.T) {
	t.Parallel()

	const (
		home          = testHomeDir
		managerID     = testNPMManagerID
		secondManager = "pip"
	)
	cachePath := filepath.Join(home, ".npm")
	secondCachePath := filepath.Join(home, ".cache", "pip")

	tests := []struct {
		newErr  func() error
		walkErr func(string, fs.WalkDirFunc, error) error
		name    string
	}{
		{
			name:   "callback walk error",
			newErr: func() error { return errors.New("callback walk error") },
			walkErr: func(root string, fn fs.WalkDirFunc, wantErr error) error {
				return fn(filepath.Join(root, "unreadable"), nil, wantErr)
			},
		},
		{
			name:   "directory entry info error",
			newErr: func() error { return errors.New("directory entry info error") },
			walkErr: func(root string, fn fs.WalkDirFunc, wantErr error) error {
				return fn(filepath.Join(root, "package.tgz"), infoErrorDirEntry{err: wantErr}, nil)
			},
		},
		{
			name:   "callback nested not exist error",
			newErr: func() error { return os.ErrNotExist },
			walkErr: func(root string, fn fs.WalkDirFunc, wantErr error) error {
				return fn(filepath.Join(root, "missing"), nil, wantErr)
			},
		},
		{
			name:   "directory entry nested not exist error",
			newErr: func() error { return os.ErrNotExist },
			walkErr: func(root string, fn fs.WalkDirFunc, wantErr error) error {
				return fn(filepath.Join(root, "missing.tgz"), infoErrorDirEntry{err: wantErr}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantErr := tt.newErr()
			fileSystem := &injectedWalkFileSystem{MapFileSystem: NewMapFileSystem()}
			fileSystem.AddDir(cachePath)
			fileSystem.AddDir(secondCachePath)
			fileSystem.walkDir = func(root string, fn fs.WalkDirFunc) error {
				if root == cachePath {
					return tt.walkErr(root, fn, wantErr)
				}
				info, err := fileSystem.Stat(root)
				if err != nil {
					return fn(root, nil, err)
				}
				return fn(root, fs.FileInfoToDirEntry(info), nil)
			}
			scanner := NewCacheScanner(
				WithFileSystem(fileSystem),
				WithHomeDir(home),
				WithGOOS("linux"),
				WithEnvLookup(func(string) string { return "" }),
				WithKnownPaths([]KnownCachePath{
					{ManagerID: managerID, RelPath: ".npm"},
					{ManagerID: secondManager, RelPath: ".cache/pip"},
				}),
				WithCacheRepository(nil),
			)

			if _, err := scanner.Scan(context.Background(), managerID); !errors.Is(err, wantErr) {
				t.Fatalf("Scan() error = %v, want wrapped %v", err, wantErr)
			} else if !strings.Contains(err.Error(), "walk cache path "+cachePath) {
				t.Fatalf("Scan() error = %q, want walk cache path context", err)
			}
			if _, err := scanner.Clean(context.Background(), "", false); !errors.Is(err, wantErr) {
				t.Fatalf("Clean() error = %v, want wrapped %v", err, wantErr)
			}
			if fileSystem.removeAllCalls != 0 {
				t.Fatalf("Clean() RemoveAll calls = %d, want 0 after scan failure", fileSystem.removeAllCalls)
			}
		})
	}
}

func TestCacheScanner_ClearDirFallbackUsesRemoveAllError(t *testing.T) {
	t.Parallel()

	const cachePath = "/home/user/.npm"

	t.Run("success remains success", func(t *testing.T) {
		fileSystem := &injectedCacheFileSystem{
			MapFileSystem: NewMapFileSystem(),
			readDirErr:    errors.New("cannot read directory"),
		}
		scanner := NewCacheScanner(WithFileSystem(fileSystem))

		if err := scanner.clearDir(cachePath); err != nil {
			t.Fatalf("clearDir() error = %v, want successful RemoveAll fallback", err)
		}
		if fileSystem.removeAllCalls != 1 {
			t.Fatalf("RemoveAll calls = %d, want 1", fileSystem.removeAllCalls)
		}
	})

	t.Run("RemoveAll error replaces ReadDir error", func(t *testing.T) {
		readDirErr := errors.New("cannot read directory")
		removeAllErr := errors.New("cannot remove directory")
		fileSystem := &injectedCacheFileSystem{
			MapFileSystem: NewMapFileSystem(),
			readDirErr:    readDirErr,
			removeAllErr:  removeAllErr,
		}
		scanner := NewCacheScanner(WithFileSystem(fileSystem))

		err := scanner.clearDir(cachePath)
		if !errors.Is(err, removeAllErr) {
			t.Fatalf("clearDir() error = %v, want wrapped RemoveAll error %v", err, removeAllErr)
		}
		if errors.Is(err, readDirErr) {
			t.Fatalf("clearDir() error = %v, must not retain ReadDir error %v", err, readDirErr)
		}
		if !strings.Contains(err.Error(), "clear cache path "+cachePath+" after ReadDir failure") {
			t.Fatalf("clearDir() error = %q, want fallback clear context", err)
		}
		if fileSystem.removeAllCalls != 1 {
			t.Fatalf("RemoveAll calls = %d, want 1", fileSystem.removeAllCalls)
		}
	})
}

func TestCacheScanner_CleanFallbackReportsOnlyRemoveAllResult(t *testing.T) {
	t.Parallel()

	const (
		home      = testHomeDir
		managerID = testNPMManagerID
		relPath   = ".npm"
	)
	cachePath := filepath.Join(home, relPath)
	newScanner := func(fileSystem FileSystem) *CacheScanner {
		return NewCacheScanner(
			WithFileSystem(fileSystem),
			WithHomeDir(home),
			WithGOOS("linux"),
			WithEnvLookup(func(string) string { return "" }),
			WithKnownPaths([]KnownCachePath{{ManagerID: managerID, RelPath: relPath}}),
			WithCacheRepository(nil),
		)
	}

	t.Run("successful fallback leaves no summary error", func(t *testing.T) {
		fileSystem := &injectedCacheFileSystem{
			MapFileSystem: NewMapFileSystem(),
			readDirErr:    errors.New("cannot read directory"),
		}
		fileSystem.AddFile(filepath.Join(cachePath, "package.tgz"), 1024, time.Now())

		summary, err := newScanner(fileSystem).Clean(context.Background(), managerID, false)
		if err != nil {
			t.Fatalf("Clean() error = %v, want successful RemoveAll fallback", err)
		}
		if summary == nil || len(summary.Errors) != 0 {
			t.Fatalf("Clean() summary = %#v, want no fallback error", summary)
		}
		if fileSystem.removeAllCalls != 1 {
			t.Fatalf("RemoveAll calls = %d, want 1", fileSystem.removeAllCalls)
		}
	})

	t.Run("failed fallback reports only RemoveAll error", func(t *testing.T) {
		readDirErr := errors.New("cannot read directory")
		removeAllErr := errors.New("cannot remove directory")
		fileSystem := &injectedCacheFileSystem{
			MapFileSystem: NewMapFileSystem(),
			readDirErr:    readDirErr,
			removeAllErr:  removeAllErr,
		}
		fileSystem.AddFile(filepath.Join(cachePath, "package.tgz"), 1024, time.Now())

		summary, err := newScanner(fileSystem).Clean(context.Background(), managerID, false)
		if !errors.Is(err, domaincleanup.ErrCacheClearFailed) {
			t.Fatalf("Clean() error = %v, want ErrCacheClearFailed", err)
		}
		if errors.Is(err, readDirErr) {
			t.Fatalf("Clean() error = %v, must not retain ReadDir error %v", err, readDirErr)
		}
		if summary == nil || len(summary.Errors) != 1 {
			t.Fatalf("Clean() summary = %#v, want one fallback error", summary)
		}
		if !strings.Contains(summary.Errors[0], "clear cache path "+cachePath+" after ReadDir failure: "+removeAllErr.Error()) {
			t.Fatalf("summary error = %q, want RemoveAll context", summary.Errors[0])
		}
		if strings.Contains(summary.Errors[0], readDirErr.Error()) {
			t.Fatalf("summary error = %q, must not expose ReadDir error", summary.Errors[0])
		}
		if fileSystem.removeAllCalls != 1 {
			t.Fatalf("RemoveAll calls = %d, want 1", fileSystem.removeAllCalls)
		}
	})
}

func TestCacheScanner_CleanDryRun(t *testing.T) {
	t.Parallel()
	home := testHomeDir
	npmCache := filepath.Join(home, ".npm")
	fsys := NewMapFileSystem()
	fsys.AddFile(filepath.Join(npmCache, "a"), 1024*1024, time.Now())

	scanner := NewCacheScanner(
		WithFileSystem(fsys),
		WithHomeDir(home),
		WithGOOS("linux"),
		WithEnvLookup(func(string) string { return "" }),
		WithCacheRepository(NewCacheRepository()),
	)

	summary, err := scanner.Clean(context.Background(), testNPMManagerID, true)
	if err != nil {
		t.Fatalf("Clean dry-run: %v", err)
	}
	if summary.PackagesRemoved != 1 {
		t.Fatalf("PackagesRemoved: %d", summary.PackagesRemoved)
	}
	if summary.SpaceFreedMB < 0.9 {
		t.Fatalf("SpaceFreedMB: %f", summary.SpaceFreedMB)
	}
	// File must still exist after dry-run
	if _, err := fsys.Stat(filepath.Join(npmCache, "a")); err != nil {
		t.Fatalf("dry-run removed file: %v", err)
	}
	if len(fsys.Removed) != 0 {
		t.Fatalf("dry-run should not call RemoveAll: %+v", fsys.Removed)
	}
}

func TestCacheScanner_CleanExecutes(t *testing.T) {
	t.Parallel()
	home := testHomeDir
	npmCache := filepath.Join(home, ".npm")
	fsys := NewMapFileSystem()
	fsys.AddFile(filepath.Join(npmCache, "a"), 1024, time.Now())
	fsys.AddFile(filepath.Join(npmCache, "b"), 2048, time.Now())

	scanner := NewCacheScanner(
		WithFileSystem(fsys),
		WithHomeDir(home),
		WithGOOS("linux"),
		WithEnvLookup(func(string) string { return "" }),
		WithCacheRepository(NewCacheRepository()),
	)

	summary, err := scanner.Clean(context.Background(), testNPMManagerID, false)
	if err != nil {
		t.Fatalf("Clean: %v", err)
	}
	if summary.PackagesRemoved != 2 {
		t.Fatalf("PackagesRemoved: %d", summary.PackagesRemoved)
	}
	if _, err := fsys.Stat(filepath.Join(npmCache, "a")); err == nil {
		t.Fatal("expected file a to be removed")
	}
}

func TestCacheScanner_CleanReportsRepositoryRefreshFailure(t *testing.T) {
	t.Parallel()

	home := testHomeDir
	npmCache := filepath.Join(home, ".npm")
	failingRepo := &cacheRepositoryStub{
		failOnUpdate: 2,
		updateErr:    errors.New("metadata store unavailable"),
	}
	fileSystem := NewMapFileSystem()
	target := filepath.Join(npmCache, "package.tgz")
	fileSystem.AddFile(target, 1024, time.Now())
	scanner := NewCacheScanner(
		WithFileSystem(fileSystem),
		WithHomeDir(home),
		WithGOOS("linux"),
		WithEnvLookup(func(string) string { return "" }),
		WithCacheRepository(failingRepo),
	)

	summary, err := scanner.Clean(context.Background(), testNPMManagerID, false)
	if !errors.Is(err, domaincleanup.ErrCacheClearFailed) {
		t.Fatalf("Clean() error = %v, want ErrCacheClearFailed", err)
	}
	if summary == nil || len(summary.Errors) != 1 {
		t.Fatalf("Clean() summary = %#v, want one reported error", summary)
	}
	if !strings.Contains(summary.Errors[0], "npm: update cache info: metadata store unavailable") {
		t.Fatalf("summary error = %q", summary.Errors[0])
	}
	if _, err := fileSystem.Stat(target); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Clean() did not remove target: %v", err)
	}
	if failingRepo.updateCalls != 2 {
		t.Fatalf("UpdateInfo calls = %d, want 2", failingRepo.updateCalls)
	}
}
