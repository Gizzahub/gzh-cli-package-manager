package command

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/cleanup"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
	repo "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/repository/cleanup"
	"github.com/spf13/cobra"
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

func captureOutput(t *testing.T, fn func() error) (stdoutText, stderrText string, runErr error) {
	t.Helper()
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		_ = stdoutReader.Close()
		_ = stdoutWriter.Close()
		t.Fatalf("stderr pipe: %v", err)
	}
	oldStdout, oldStderr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = stdoutWriter, stderrWriter
	runErr = fn()
	_ = stdoutWriter.Close()
	_ = stderrWriter.Close()
	os.Stdout, os.Stderr = oldStdout, oldStderr

	var stdout, stderr bytes.Buffer
	_, _ = io.Copy(&stdout, stdoutReader)
	_, _ = io.Copy(&stderr, stderrReader)
	_ = stdoutReader.Close()
	_ = stderrReader.Close()
	return stdout.String(), stderr.String(), runErr
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
	if !strings.Contains(out, testNPMManagerID) {
		t.Fatalf("scan output missing npm: %q", out)
	}

	out, err = captureStdout(t, func() error {
		return cacheStatusCmd.RunE(cacheStatusCmd, nil)
	})
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if !strings.Contains(out, testNPMManagerID) {
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

	cleanupManagerID = testNPMManagerID
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
		Name: "stale", Version: testQuarantineVersion, ManagerID: testNPMManagerID,
		QuarantinedAt: now.AddDate(0, 0, -60), Status: cleanup.StatusQuarantined, SizeMB: 5,
	})
	_ = qrepo.Save(ctx, &cleanup.QuarantinedPackage{
		Name: "fresh", Version: testQuarantineVersion, ManagerID: testNPMManagerID,
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

	if _, err := qrepo.Get(ctx, "stale", testQuarantineVersion, testNPMManagerID); err == nil {
		t.Fatal("stale package should be purged")
	}
	if _, err := qrepo.Get(ctx, "fresh", testQuarantineVersion, testNPMManagerID); err != nil {
		t.Fatalf("fresh package should remain: %v", err)
	}
}

func TestCleanupOrphansList(t *testing.T) {
	SetManagerAdapters(map[manager.ManagerID]adapterm.Adapter{
		manager.ManagerScoop: &stubListAdapter{packages: []manager.Package{
			{Name: testGitPackageName, CurrentVersion: "2.43.0"},
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
			{Name: testGitPackageName, CurrentVersion: "2.43.0"},
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

func TestCleanupRunEWrapsCacheScanAndCleanFailures(t *testing.T) {
	sentinel := errors.New("cache filesystem unavailable")
	scanner := repo.NewCacheScanner(
		repo.WithFileSystem(failingCacheFileSystem{err: sentinel}),
		repo.WithHomeDir("/home/test"),
		repo.WithGOOS("linux"),
		repo.WithKnownPaths([]repo.KnownCachePath{{ManagerID: testNPMManagerID, RelPath: ".npm"}}),
	)
	SetCleanupDeps(nil, nil, scanner)
	t.Cleanup(func() { SetCleanupDeps(nil, nil, nil) })

	cleanupManagerID = testNPMManagerID
	cleanupDryRun = true
	t.Cleanup(func() {
		cleanupManagerID = ""
		cleanupDryRun = false
	})

	for name, command := range map[string]*cobra.Command{
		"scan caches":  cacheScanCmd,
		"clean caches": cacheCleanCmd,
	} {
		t.Run(name, func(t *testing.T) {
			err := command.RunE(command, nil)
			if !errors.Is(err, sentinel) {
				t.Fatalf("errors.Is(%v, sentinel) = false", err)
			}
			if !strings.Contains(err.Error(), name) {
				t.Fatalf("error %q missing context %q", err, name)
			}
		})
	}
}

func TestCleanupRunEWrapsPackageListFailures(t *testing.T) {
	sentinel := errors.New("adapter list unavailable")
	stub := &failingListAdapter{err: sentinel}
	SetManagerAdapters(map[manager.ManagerID]adapterm.Adapter{manager.ManagerScoop: stub})
	t.Cleanup(func() { SetManagerAdapters(nil) })

	cleanupManagerID = "scoop"
	removeDryRun = true
	t.Cleanup(func() {
		cleanupManagerID = ""
		removeDryRun = true
	})

	for name, command := range map[string]*cobra.Command{
		"detect orphan packages":                       orphansListCmd,
		"detect orphan packages for removal":           orphansRemoveCmd,
		"scan package versions for removal from scoop": versionsRemoveCmd,
	} {
		t.Run(name, func(t *testing.T) {
			err := command.RunE(command, nil)
			if !errors.Is(err, sentinel) {
				t.Fatalf("errors.Is(%v, sentinel) = false", err)
			}
			if !strings.Contains(err.Error(), name) {
				t.Fatalf("error %q missing context %q", err, name)
			}
		})
	}
}

func TestAdapterPackageListerPreservesAdapterError(t *testing.T) {
	sentinel := errors.New("adapter list unavailable")
	lister := adapterPackageLister{adapter: &failingListAdapter{err: sentinel}}

	_, err := lister.ListPackages(context.Background())
	if !errors.Is(err, sentinel) {
		t.Fatalf("errors.Is(%v, sentinel) = false", err)
	}
	if errors.Unwrap(err) != nil || err.Error() != sentinel.Error() {
		t.Fatalf("ListPackages() error = %v, want unchanged sentinel", err)
	}
}

func TestCleanupOrphansRemoveReportsInstallerFailureWithoutDuplicateContext(t *testing.T) {
	sentinel := errors.New("uninstall unavailable")
	stub := &failingUninstallAdapter{
		stubListAdapter: &stubListAdapter{packages: []manager.Package{{Name: "ghost"}}},
		err:             sentinel,
	}
	SetManagerAdapters(map[manager.ManagerID]adapterm.Adapter{manager.ManagerScoop: stub})
	t.Cleanup(func() { SetManagerAdapters(nil) })

	cleanupManagerID = "scoop"
	removeDryRun = false
	t.Cleanup(func() {
		cleanupManagerID = ""
		removeDryRun = true
	})

	stdout, stderr, err := captureOutput(t, func() error {
		return orphansRemoveCmd.RunE(orphansRemoveCmd, nil)
	})
	if err != nil {
		t.Fatalf("orphans remove: %v", err)
	}
	if !strings.Contains(stdout, "Orphan remove complete [LIVE]") {
		t.Fatalf("summary output missing success: %q", stdout)
	}
	want := "ghost@- (scoop): " + sentinel.Error()
	if !strings.Contains(stderr, "Errors/skips:") || !strings.Contains(stderr, want) {
		t.Fatalf("stderr = %q, want Errors/skips and %q", stderr, want)
	}
	if strings.Count(stderr, "ghost@- (scoop)") != 1 {
		t.Fatalf("stderr duplicated package context: %q", stderr)
	}
	if strings.Count(stderr, sentinel.Error()) != 1 {
		t.Fatalf("stderr duplicated installer context: %q", stderr)
	}
}

type failingCacheFileSystem struct {
	err error
}

func (f failingCacheFileSystem) Stat(string) (fs.FileInfo, error) { return nil, f.err }

func (f failingCacheFileSystem) WalkDir(string, fs.WalkDirFunc) error { return f.err }

func (f failingCacheFileSystem) RemoveAll(string) error { return f.err }

func (f failingCacheFileSystem) ReadDir(string) ([]fs.DirEntry, error) { return nil, f.err }

// stubListAdapter implements adapterm.Adapter (+Installer) with fixed ListPackages.
type stubListAdapter struct {
	packages    []manager.Package
	uninstalled []string
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

type failingListAdapter struct {
	*stubListAdapter
	err error
}

func (a *failingListAdapter) ListPackages(context.Context) ([]manager.Package, error) {
	return nil, a.err
}

type failingUninstallAdapter struct {
	*stubListAdapter
	err error
}

func (a *failingUninstallAdapter) Uninstall(context.Context, string, bool) error {
	return a.err
}

// Install/Uninstall satisfy adapterm.Installer for remove-path tests.
func (s *stubListAdapter) Install(context.Context, string, bool) error { return nil }

func (s *stubListAdapter) Uninstall(_ context.Context, pkgID string, dryRun bool) error {
	if s.uninstalled == nil {
		s.uninstalled = []string{}
	}
	if dryRun {
		s.uninstalled = append(s.uninstalled, "dry:"+pkgID)
	} else {
		s.uninstalled = append(s.uninstalled, pkgID)
	}
	return nil
}

func TestCleanupOrphansRemove_DryRunDefault(t *testing.T) {
	stub := &stubListAdapter{packages: []manager.Package{
		{Name: "ghost", CurrentVersion: ""},
		{Name: testGitPackageName, CurrentVersion: "2.0"},
	}}
	SetManagerAdapters(map[manager.ManagerID]adapterm.Adapter{
		manager.ManagerScoop: stub,
	})
	t.Cleanup(func() { SetManagerAdapters(nil) })

	cleanupManagerID = "scoop"
	removeDryRun = true
	t.Cleanup(func() {
		cleanupManagerID = ""
		removeDryRun = true
	})

	out, err := captureStdout(t, func() error {
		return orphansRemoveCmd.RunE(orphansRemoveCmd, nil)
	})
	if err != nil {
		t.Fatalf("orphans remove: %v", err)
	}
	if !strings.Contains(out, "DRY-RUN") {
		t.Fatalf("expected DRY-RUN mode: %q", out)
	}
	if len(stub.uninstalled) == 0 || !strings.HasPrefix(stub.uninstalled[0], "dry:") {
		t.Fatalf("expected dry uninstall call: %v", stub.uninstalled)
	}
}

func TestCleanupOrphansRemove_Live(t *testing.T) {
	stub := &stubListAdapter{packages: []manager.Package{
		{Name: "ghost", CurrentVersion: ""},
	}}
	SetManagerAdapters(map[manager.ManagerID]adapterm.Adapter{
		manager.ManagerScoop: stub,
	})
	t.Cleanup(func() { SetManagerAdapters(nil) })

	cleanupManagerID = "scoop"
	removeDryRun = false
	t.Cleanup(func() {
		cleanupManagerID = ""
		removeDryRun = true
	})

	out, err := captureStdout(t, func() error {
		return orphansRemoveCmd.RunE(orphansRemoveCmd, nil)
	})
	if err != nil {
		t.Fatalf("orphans remove live: %v", err)
	}
	if !strings.Contains(out, "LIVE") {
		t.Fatalf("expected LIVE mode: %q", out)
	}
	if len(stub.uninstalled) != 1 || stub.uninstalled[0] != "ghost" {
		t.Fatalf("expected live uninstall of ghost: %v", stub.uninstalled)
	}
}

func TestCleanupVersionsRemove_DryRun(t *testing.T) {
	stub := &stubListAdapter{packages: []manager.Package{
		{Name: "python", CurrentVersion: "3.11.0"},
		{Name: "python", CurrentVersion: "3.12.0"},
	}}
	SetManagerAdapters(map[manager.ManagerID]adapterm.Adapter{
		manager.ManagerWinget: stub,
	})
	t.Cleanup(func() { SetManagerAdapters(nil) })

	cleanupManagerID = "winget"
	removeDryRun = true
	t.Cleanup(func() {
		cleanupManagerID = ""
		removeDryRun = true
	})

	out, err := captureStdout(t, func() error {
		return versionsRemoveCmd.RunE(versionsRemoveCmd, nil)
	})
	if err != nil {
		t.Fatalf("versions remove: %v", err)
	}
	if !strings.Contains(out, "DRY-RUN") {
		t.Fatalf("expected DRY-RUN: %q", out)
	}
	// old version 3.11.0 should be targeted (lexicographic last 3.12.0 is current)
	found := false
	for _, c := range stub.uninstalled {
		if strings.Contains(c, "3.11.0") || strings.Contains(c, "python") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected uninstall call for old python: %v", stub.uninstalled)
	}
}
