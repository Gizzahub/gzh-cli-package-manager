package cleanup

import (
	"context"
	"errors"
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
	home := "/home/user"
	paths := ResolveCachePaths(home, "linux", nil, DefaultKnownCachePaths())

	if got := paths["npm"]; got != filepath.Join(home, ".npm") {
		t.Fatalf("npm path: got %q", got)
	}
	if got := paths["homebrew"]; got != filepath.Join(home, ".cache/Homebrew") {
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
	if paths["npm"] != "/custom/npm-cache" {
		t.Fatalf("npm AbsEnv: got %q", paths["npm"])
	}

	// darwin homebrew
	paths = ResolveCachePaths(home, "darwin", nil, DefaultKnownCachePaths())
	if paths["homebrew"] != filepath.Join(home, "Library/Caches/Homebrew") {
		t.Fatalf("homebrew darwin: got %q", paths["homebrew"])
	}
}

func TestCacheScanner_Scan(t *testing.T) {
	t.Parallel()
	home := "/home/user"
	npmCache := filepath.Join(home, ".npm")
	pipCache := filepath.Join(home, ".cache/pip")

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
		if r.ManagerID == "npm" {
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
	stored, err := repo.GetInfo(context.Background(), "npm")
	if err != nil {
		t.Fatalf("GetInfo: %v", err)
	}
	if stored.EntryCount != 2 {
		t.Fatalf("repo npm EntryCount: %d", stored.EntryCount)
	}
}

func TestCacheScanner_ScanManagerFilter(t *testing.T) {
	t.Parallel()
	home := "/home/user"
	fsys := NewMapFileSystem()
	fsys.AddFile(filepath.Join(home, ".npm", "a"), 100, time.Now())
	fsys.AddFile(filepath.Join(home, ".cache/pip", "b"), 100, time.Now())

	scanner := NewCacheScanner(
		WithFileSystem(fsys),
		WithHomeDir(home),
		WithGOOS("linux"),
		WithEnvLookup(func(string) string { return "" }),
		WithCacheRepository(NewCacheRepository()),
	)

	results, err := scanner.Scan(context.Background(), "npm")
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(results) != 1 || results[0].ManagerID != "npm" {
		t.Fatalf("filter failed: %+v", results)
	}
}

func TestCacheScanner_CleanDryRun(t *testing.T) {
	t.Parallel()
	home := "/home/user"
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

	summary, err := scanner.Clean(context.Background(), "npm", true)
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
	home := "/home/user"
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

	summary, err := scanner.Clean(context.Background(), "npm", false)
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

	home := "/home/user"
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

	summary, err := scanner.Clean(context.Background(), "npm", false)
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
