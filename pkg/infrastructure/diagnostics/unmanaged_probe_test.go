package diagnostics

import (
	"os"
	"path/filepath"
	"testing"

	domaindiagnostics "github.com/gizzahub/gzh-cli-package-manager/pkg/domain/diagnostics"
)

const (
	commandClaude        = "claude"
	providerClaudeNative = "claude-native"
	claudeRelativePath   = ".local/bin/claude"
	claudeVersion        = "2.1.287"

	labelPresentRoot     = "present-root"
	labelMissingRoot     = "missing-root"
	labelNoHomeWalk      = "no-home-walk"
	labelResolvedSymlink = "resolved-symlink"
)

// probeFixture is one temporary home root with the absolute paths the probe
// scenarios assert against. The root is resolved through symlinks first so
// the macOS /var -> /private/var indirection never leaks into path
// assertions.
type probeFixture struct {
	root          string // home root handed to the probe.
	claudeInstall string // absolute path of the table row's file under the root.
	claudeTarget  string // absolute path of the versioned file behind the launcher.
}

// TestUnmanagedInstallerProbe covers the card labels: the table row's file is
// reported with exact install facts when present under the supplied root, a
// missing root or missing file reports nothing, a claude file outside the
// table row is never reported because the probe does not walk the home root,
// and a launcher symlink resolves to its versioned target with the version
// extracted.
func TestUnmanagedInstallerProbe(t *testing.T) {
	t.Run(labelPresentRoot, testPresentRoot)
	t.Run(labelMissingRoot, testMissingRoot)
	t.Run(labelNoHomeWalk, testNoHomeWalk)
	t.Run(labelResolvedSymlink, testResolvedSymlink)
}

// testPresentRoot creates the table row's file and asserts the one record the
// probe reports for it, field by field.
func testPresentRoot(t *testing.T) {
	t.Helper()
	fixture := newProbeFixture(t)
	writeProbeFile(t, fixture.claudeInstall)

	records := ProbeUnmanagedInstalls(fixture.root, nil)

	if len(records) != 1 {
		t.Fatalf("expected exactly one record for the table row, got %d: %+v", len(records), records)
	}
	record := &records[0]
	if record.RealPath != fixture.claudeInstall {
		t.Errorf("expected real path %q, got %q", fixture.claudeInstall, record.RealPath)
	}
	if record.Version != "" {
		t.Errorf("expected an empty version for a plain launcher file, got %q", record.Version)
	}
	assertUnmanagedRecord(t, record)
}

// testMissingRoot asserts that a missing home root and a present root without
// the table row's file both report nothing.
func testMissingRoot(t *testing.T) {
	t.Helper()
	fixture := newProbeFixture(t)

	missingRoot := ProbeUnmanagedInstalls(filepath.Join(fixture.root, "absent-home"), nil)
	if len(missingRoot) != 0 {
		t.Fatalf("expected no record from a missing home root, got %d: %+v", len(missingRoot), missingRoot)
	}

	missingFile := ProbeUnmanagedInstalls(fixture.root, nil)
	if len(missingFile) != 0 {
		t.Fatalf("expected no record when the table row's file is absent, got %d: %+v", len(missingFile), missingFile)
	}
}

// testNoHomeWalk places a claude file at the home root and under the fixed
// .claude/local path, neither of which is a table row, and asserts the probe
// reports neither because it never walks the root.
func testNoHomeWalk(t *testing.T) {
	t.Helper()
	fixture := newProbeFixture(t)
	writeProbeFile(t, filepath.Join(fixture.root, commandClaude))
	writeProbeFile(t, filepath.Join(fixture.root, ".claude", "local", commandClaude))

	records := ProbeUnmanagedInstalls(fixture.root, nil)

	if len(records) != 0 {
		t.Fatalf("expected no record for files outside the table row, got %d: %+v", len(records), records)
	}
}

// testResolvedSymlink points the table row's file at a versioned target and
// asserts the record resolves to that target and carries its version.
func testResolvedSymlink(t *testing.T) {
	t.Helper()
	fixture := newProbeFixture(t)
	writeProbeFile(t, fixture.claudeTarget)
	binDir := filepath.Dir(fixture.claudeInstall)
	if err := os.MkdirAll(binDir, 0o750); err != nil {
		t.Fatalf("create %s: %v", binDir, err)
	}
	if err := os.Symlink(fixture.claudeTarget, fixture.claudeInstall); err != nil {
		t.Fatalf("link %s to %s: %v", fixture.claudeInstall, fixture.claudeTarget, err)
	}

	records := ProbeUnmanagedInstalls(fixture.root, nil)

	if len(records) != 1 {
		t.Fatalf("expected the launcher and its target to be one install, got %d: %+v", len(records), records)
	}
	record := &records[0]
	if record.RealPath != fixture.claudeTarget {
		t.Errorf("expected real path %q, got %q", fixture.claudeTarget, record.RealPath)
	}
	if record.Version != claudeVersion {
		t.Errorf("expected version %q, got %q", claudeVersion, record.Version)
	}
	assertUnmanagedRecord(t, record)
}

// assertUnmanagedRecord asserts the install facts every reported record
// carries: the table row's command, provider, kind, active flag, and the
// not-executed state that holds without search paths.
func assertUnmanagedRecord(t *testing.T, record *domaindiagnostics.InstallRecord) {
	t.Helper()
	if record.Command != commandClaude {
		t.Errorf("expected command %q, got %q", commandClaude, record.Command)
	}
	if record.ProviderID != providerClaudeNative {
		t.Errorf("expected provider %q, got %q", providerClaudeNative, record.ProviderID)
	}
	if record.Kind != KindUnmanaged {
		t.Errorf("expected kind %q, got %q", KindUnmanaged, record.Kind)
	}
	if !record.Active {
		t.Error("expected the present file to be active")
	}
	if record.Executed {
		t.Error("expected the record to stay not executed without search paths")
	}
}

// newProbeFixture builds the temporary home root and the absolute paths the
// scenarios share.
func newProbeFixture(t *testing.T) probeFixture {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("resolve temp dir: %v", err)
	}
	return probeFixture{
		root:          root,
		claudeInstall: filepath.Join(root, claudeRelativePath),
		claudeTarget:  filepath.Join(root, ".local", "share", "claude", "versions", claudeVersion, commandClaude),
	}
}

// writeProbeFile creates a regular file at path. The probe never reads or
// executes fixture files, so the content is irrelevant.
func writeProbeFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("create %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
