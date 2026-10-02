package diagnostics

import (
	"os"
	"path/filepath"
	"testing"

	domaindiagnostics "github.com/gizzahub/gzh-cli-package-manager/pkg/domain/diagnostics"
)

const (
	commandNode       = "node"
	providerMise      = "mise"
	providerHomebrew  = "homebrew"
	providerMiseShims = "mise-shims"
	providerNpm       = "npm"
	version20         = "20.0.0"
	version22         = "22.0.0"

	labelSameFileSymlink = "same-file-symlink"
	labelDistinctShim    = "distinct-shim"
	labelExecutedFile    = "executed-file"

	realNodeScript = "#!/bin/sh\nexec node \"$@\"\n"
	shimScript     = "#!/bin/sh\nexec mise exec -- node \"$@\"\n"
)

// resolveFixture is one temporary install tree holding every file the
// resolver scenarios need.
type resolveFixture struct {
	toolsBin string // directory that directly holds the real node file.
	realNode string // the mise-installed node binary, a regular file.
	brewBin  string // homebrew bin directory.
	brewNode string // symlink name for the real node inside the homebrew bin.
	shimBin  string // directory holding the mise shim.
	shimNode string // the shim, a regular file distinct from the real node.
	npmBin   string // directory holding the npm node copy.
	npmNode  string // a second, distinct regular node file.
}

// TestResolvePathIdentity covers the card labels: a search-path symlink that
// names another record's file collapses into one executed install, a shim
// that is a different file stays visible and not executed, and only the
// search-path hit is marked executed.
func TestResolvePathIdentity(t *testing.T) {
	tests := []struct {
		name  string
		run   func(t *testing.T) (resolveFixture, []domaindiagnostics.InstallRecord)
		check func(t *testing.T, fixture *resolveFixture, resolved []domaindiagnostics.InstallRecord)
	}{
		{name: labelSameFileSymlink, run: runSameFileSymlink, check: checkSameFileSymlink},
		{name: labelDistinctShim, run: runDistinctShim, check: checkDistinctShim},
		{name: labelExecutedFile, run: runExecutedFile, check: checkExecutedFile},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture, resolved := tt.run(t)
			tt.check(t, &fixture, resolved)
		})
	}
}

// newResolveFixture builds the temporary install tree. The root is resolved
// through symlinks first so the macOS /var -> /private/var indirection never
// leaks into path assertions.
func newResolveFixture(t *testing.T) resolveFixture {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("resolve temp dir: %v", err)
	}
	fixture := resolveFixture{
		toolsBin: filepath.Join(root, "tools", "node", "20", "bin"),
		brewBin:  filepath.Join(root, "homebrew", "bin"),
		shimBin:  filepath.Join(root, "mise", "shims"),
		npmBin:   filepath.Join(root, "npm", "lib", "node_modules", "node", "bin"),
	}
	fixture.realNode = filepath.Join(fixture.toolsBin, commandNode)
	fixture.brewNode = filepath.Join(fixture.brewBin, commandNode)
	fixture.shimNode = filepath.Join(fixture.shimBin, commandNode)
	fixture.npmNode = filepath.Join(fixture.npmBin, commandNode)

	writeFixtureFile(t, fixture.realNode, realNodeScript)
	writeFixtureFile(t, fixture.shimNode, shimScript)
	writeFixtureFile(t, fixture.npmNode, realNodeScript)
	if err := os.MkdirAll(fixture.brewBin, 0o750); err != nil {
		t.Fatalf("create %s: %v", fixture.brewBin, err)
	}
	if err := os.Symlink(fixture.realNode, fixture.brewNode); err != nil {
		t.Fatalf("link %s to %s: %v", fixture.brewNode, fixture.realNode, err)
	}
	return fixture
}

// writeFixtureFile writes one fixture file. The resolver never reads or runs
// these files, so the content only has to look like a binary or shim script.
func writeFixtureFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("create %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// runSameFileSymlink resolves a mise record naming the real file and a
// homebrew record naming only a symlink to it, with the symlink's directory
// as the sole search path.
func runSameFileSymlink(t *testing.T) (resolveFixture, []domaindiagnostics.InstallRecord) {
	t.Helper()
	fixture := newResolveFixture(t)
	records := []domaindiagnostics.InstallRecord{
		{Command: commandNode, ProviderID: providerMise, Kind: domaindiagnostics.KindVersionManager, Version: version20, RealPath: fixture.realNode, Active: true},
		{Command: commandNode, ProviderID: providerHomebrew, Kind: domaindiagnostics.KindSystemOrLanguage, Version: version20, RealPath: fixture.brewNode, Active: true},
	}
	return fixture, ResolvePathIdentity(records, []string{fixture.brewBin})
}

// checkSameFileSymlink asserts the two names for one file became one executed
// install carrying the real file's provider facts.
func checkSameFileSymlink(t *testing.T, fixture *resolveFixture, resolved []domaindiagnostics.InstallRecord) {
	t.Helper()
	if len(resolved) != 1 {
		t.Fatalf("expected the symlink name and the real file to become one install, got %d: %+v", len(resolved), resolved)
	}
	install := resolved[0]
	if install.ProviderID != providerMise {
		t.Errorf("expected the record that names the real file to keep provider %q, got %q", providerMise, install.ProviderID)
	}
	if install.Kind != domaindiagnostics.KindVersionManager {
		t.Errorf("expected kind %q, got %q", domaindiagnostics.KindVersionManager, install.Kind)
	}
	if install.Version != version20 {
		t.Errorf("expected version %q, got %q", version20, install.Version)
	}
	if install.RealPath != fixture.realNode {
		t.Errorf("expected real path %q, got %q", fixture.realNode, install.RealPath)
	}
	if !install.Executed {
		t.Errorf("expected the collapsed install of %q to be executed", install.RealPath)
	}
}

// runDistinctShim resolves the real file and a regular-file shim, with the
// shim directory searched after the real file's directory so the shim is
// shadowed but present.
func runDistinctShim(t *testing.T) (resolveFixture, []domaindiagnostics.InstallRecord) {
	t.Helper()
	fixture := newResolveFixture(t)
	records := []domaindiagnostics.InstallRecord{
		{Command: commandNode, ProviderID: providerMise, Kind: domaindiagnostics.KindVersionManager, Version: version20, RealPath: fixture.realNode, Active: true},
		{Command: commandNode, ProviderID: providerMiseShims, Kind: domaindiagnostics.KindVersionManager, RealPath: fixture.shimNode, Active: true},
	}
	return fixture, ResolvePathIdentity(records, []string{fixture.toolsBin, fixture.shimBin})
}

// checkDistinctShim asserts the shim stays its own visible record at its own
// path and is not executed.
func checkDistinctShim(t *testing.T, fixture *resolveFixture, resolved []domaindiagnostics.InstallRecord) {
	t.Helper()
	if len(resolved) != 2 {
		t.Fatalf("expected the shim and the real file to stay two records, got %d: %+v", len(resolved), resolved)
	}
	shim := findRecord(t, resolved, providerMiseShims)
	if shim.RealPath != fixture.shimNode {
		t.Errorf("expected the shim to stay its own record at %q, got %q", fixture.shimNode, shim.RealPath)
	}
	if shim.Executed {
		t.Errorf("expected the shim %q to stay not executed", shim.RealPath)
	}
}

// runExecutedFile resolves the full install tree with the homebrew bin dir
// searched first, so the symlink hit shadows the shim directory.
func runExecutedFile(t *testing.T) (resolveFixture, []domaindiagnostics.InstallRecord) {
	t.Helper()
	fixture := newResolveFixture(t)
	records := []domaindiagnostics.InstallRecord{
		{Command: commandNode, ProviderID: providerMise, Kind: domaindiagnostics.KindVersionManager, Version: version20, RealPath: fixture.realNode, Active: true},
		{Command: commandNode, ProviderID: providerHomebrew, Kind: domaindiagnostics.KindSystemOrLanguage, Version: version20, RealPath: fixture.brewNode, Active: true},
		{Command: commandNode, ProviderID: providerMiseShims, Kind: domaindiagnostics.KindVersionManager, RealPath: fixture.shimNode, Active: true},
		{Command: commandNode, ProviderID: providerNpm, Kind: domaindiagnostics.KindSystemOrLanguage, Version: version22, RealPath: fixture.npmNode, Active: false},
	}
	return fixture, ResolvePathIdentity(records, []string{fixture.brewBin, fixture.shimBin})
}

// checkExecutedFile asserts exactly one record is executed, that it is the
// search-path hit's file, and that the shim and the npm copy stay not
// executed.
func checkExecutedFile(t *testing.T, fixture *resolveFixture, resolved []domaindiagnostics.InstallRecord) {
	t.Helper()
	if len(resolved) != 3 {
		t.Fatalf("expected one install per distinct file, got %d: %+v", len(resolved), resolved)
	}
	executed := 0
	for i := range resolved {
		if !resolved[i].Executed {
			continue
		}
		executed++
		if resolved[i].RealPath != fixture.realNode {
			t.Errorf("expected the executed file to be %q, got %q", fixture.realNode, resolved[i].RealPath)
		}
		if resolved[i].ProviderID != providerMise {
			t.Errorf("expected the executed install to keep provider %q, got %q", providerMise, resolved[i].ProviderID)
		}
	}
	if executed != 1 {
		t.Errorf("expected exactly one executed record, got %d: %+v", executed, resolved)
	}
	if shim := findRecord(t, resolved, providerMiseShims); shim.Executed {
		t.Errorf("expected the shim %q to stay not executed", shim.RealPath)
	}
	if npm := findRecord(t, resolved, providerNpm); npm.Executed {
		t.Errorf("expected the npm copy %q to stay not executed", npm.RealPath)
	}
}

// findRecord returns the resolved record for one provider, failing the test
// when no record carries that provider id.
func findRecord(t *testing.T, resolved []domaindiagnostics.InstallRecord, providerID string) *domaindiagnostics.InstallRecord {
	t.Helper()
	for i := range resolved {
		if resolved[i].ProviderID == providerID {
			return &resolved[i]
		}
	}
	t.Fatalf("no resolved record for provider %q in %+v", providerID, resolved)
	return nil
}
