package command

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/cleanup"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
	repo "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/repository/cleanup"
)

func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	old := os.Stdout
	os.Stdout = w
	runErr := fn()
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()
	return buf.String(), runErr
}

func TestCleanupCacheScanAndStatus(t *testing.T) {
	home := t.TempDir()
	npmCache := filepath.Join(home, ".npm")
	if err := os.MkdirAll(npmCache, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(npmCache, "pkg.tgz"), bytes.Repeat([]byte("x"), 1024), 0o644); err != nil {
		t.Fatal(err)
	}

	cacheRepo := repo.NewCacheRepository()
	scanner := repo.NewCacheScanner(
		repo.WithFileSystem(repo.NewOSFileSystem()),
		repo.WithHomeDir(home),
		repo.WithGOOS("linux"),
		repo.WithEnvLookup(func(string) string { return "" }),
		repo.WithCacheRepository(cacheRepo),
	)
	SetCleanupDeps(cacheRepo, repo.NewQuarantineRepository(), scanner)
	t.Cleanup(func() { SetCleanupDeps(nil, nil, nil) })

	// Reset flags
	cleanupManagerID = ""
	cleanupDryRun = false

	out, err := captureStdout(t, func() error {
		return cacheScanCmd.RunE(cacheScanCmd, nil)
	})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if !strings.Contains(out, "npm") {
		t.Fatalf("scan output missing npm: %q", out)
	}

	out, err = captureStdout(t, func() error {
		return cacheStatusCmd.RunE(cacheStatusCmd, nil)
	})
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if !strings.Contains(out, "npm") {
		t.Fatalf("status missing npm after scan: %q", out)
	}
}

func TestCleanupCacheCleanDryRun(t *testing.T) {
	home := t.TempDir()
	npmCache := filepath.Join(home, ".npm")
	if err := os.MkdirAll(npmCache, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(npmCache, "keep-me")
	if err := os.WriteFile(target, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	cacheRepo := repo.NewCacheRepository()
	scanner := repo.NewCacheScanner(
		repo.WithFileSystem(repo.NewOSFileSystem()),
		repo.WithHomeDir(home),
		repo.WithGOOS("linux"),
		repo.WithEnvLookup(func(string) string { return "" }),
		repo.WithCacheRepository(cacheRepo),
	)
	SetCleanupDeps(cacheRepo, nil, scanner)
	t.Cleanup(func() { SetCleanupDeps(nil, nil, nil) })

	cleanupManagerID = "npm"
	cleanupDryRun = true
	t.Cleanup(func() {
		cleanupManagerID = ""
		cleanupDryRun = false
	})

	out, err := captureStdout(t, func() error {
		return cacheCleanCmd.RunE(cacheCleanCmd, nil)
	})
	if err != nil {
		t.Fatalf("clean dry-run: %v", err)
	}
	if !strings.Contains(out, "Dry-run") {
		t.Fatalf("expected dry-run banner: %q", out)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("dry-run deleted file: %v", err)
	}
}

func TestCleanupQuarantinePurge(t *testing.T) {
	qrepo := repo.NewQuarantineRepository()
	ctx := context.Background()
	now := time.Now()
	_ = qrepo.Save(ctx, &cleanup.QuarantinedPackage{
		Name: "stale", Version: "1.0", ManagerID: "npm",
		QuarantinedAt: now.AddDate(0, 0, -60), Status: cleanup.StatusQuarantined, SizeMB: 5,
	})
	_ = qrepo.Save(ctx, &cleanup.QuarantinedPackage{
		Name: "fresh", Version: "1.0", ManagerID: "npm",
		QuarantinedAt: now.AddDate(0, 0, -1), Status: cleanup.StatusQuarantined, SizeMB: 1,
	})

	SetCleanupDeps(nil, qrepo, nil)
	t.Cleanup(func() { SetCleanupDeps(nil, nil, nil) })

	cleanupRetentionDays = 30
	cleanupDryRun = false

	out, err := captureStdout(t, func() error {
		return quarantinePurgeCmd.RunE(quarantinePurgeCmd, nil)
	})
	if err != nil {
		t.Fatalf("purge: %v", err)
	}
	if !strings.Contains(out, "Packages: 1") {
		t.Fatalf("unexpected purge output: %q", out)
	}

	if _, err := qrepo.Get(ctx, "stale", "1.0", "npm"); err == nil {
		t.Fatal("stale package should be purged")
	}
	if _, err := qrepo.Get(ctx, "fresh", "1.0", "npm"); err != nil {
		t.Fatalf("fresh package should remain: %v", err)
	}
}

func TestCleanupOrphansList(t *testing.T) {
	SetManagerAdapters(map[manager.ManagerID]adapterm.Adapter{
		manager.ManagerScoop: &stubListAdapter{packages: []manager.Package{
			{Name: "git", CurrentVersion: "2.43.0"},
			{Name: "", CurrentVersion: "1.0"},
			{Name: "unknown", CurrentVersion: "0.1"},
		}},
	})
	t.Cleanup(func() { SetManagerAdapters(nil) })

	cleanupManagerID = "scoop"
	cleanupDryRun = true
	t.Cleanup(func() {
		cleanupManagerID = ""
		cleanupDryRun = false
	})

	out, err := captureStdout(t, func() error {
		return orphansListCmd.RunE(orphansListCmd, nil)
	})
	if err != nil {
		t.Fatalf("orphans list: %v", err)
	}
	if !strings.Contains(out, "orphan") && !strings.Contains(out, "Orphan") {
		t.Fatalf("expected orphan header: %q", out)
	}
	if !strings.Contains(out, "Dry-run: would remove") {
		t.Fatalf("expected dry-run remove message: %q", out)
	}
	if !strings.Contains(out, "(unnamed)") && !strings.Contains(out, "unknown") {
		t.Fatalf("expected candidate names: %q", out)
	}
}

func TestCleanupVersionsList(t *testing.T) {
	SetManagerAdapters(map[manager.ManagerID]adapterm.Adapter{
		manager.ManagerWinget: &stubListAdapter{packages: []manager.Package{
			{Name: "python", CurrentVersion: "3.11.0"},
			{Name: "python", CurrentVersion: "3.12.0"},
			{Name: "git", CurrentVersion: "2.43.0"},
		}},
	})
	t.Cleanup(func() { SetManagerAdapters(nil) })

	cleanupManagerID = "winget"
	cleanupDryRun = true
	t.Cleanup(func() {
		cleanupManagerID = ""
		cleanupDryRun = false
	})

	out, err := captureStdout(t, func() error {
		return versionsListCmd.RunE(versionsListCmd, nil)
	})
	if err != nil {
		t.Fatalf("versions list: %v", err)
	}
	if !strings.Contains(out, "python") {
		t.Fatalf("expected python multi-version: %q", out)
	}
	if !strings.Contains(out, "Dry-run: would remove old version") {
		t.Fatalf("expected dry-run for old version: %q", out)
	}
}

// stubListAdapter implements adapterm.Adapter with fixed ListPackages results.
type stubListAdapter struct {
	packages []manager.Package
}

func (s *stubListAdapter) Detect(context.Context) (bool, error) { return true, nil }
func (s *stubListAdapter) GetVersion(context.Context) (string, error) {
	return "0", nil
}
func (s *stubListAdapter) GetBinaryPath(context.Context) (string, error) { return "", nil }
func (s *stubListAdapter) GetConfigPath(context.Context) (string, error) { return "", nil }
func (s *stubListAdapter) ListPackages(context.Context) ([]manager.Package, error) {
	return s.packages, nil
}
func (s *stubListAdapter) CheckHealth(context.Context) (manager.Status, error) {
	return manager.StatusHealthy, nil
}
func (s *stubListAdapter) Update(context.Context, adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
	return &adapterm.UpdateResult{Success: true}, nil
}
