package npm

import (
	"context"
	"errors"
	"maps"
	"path/filepath"
	"slices"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/diagnostics"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/testutil"
)

const (
	// inventoryFixtureRoot stubs the npm root -g answer.
	inventoryFixtureRoot = "/usr/local/lib/node_modules"

	// codexScopedName is the scoped package whose command differs from its name.
	codexScopedName = "@openai/codex"
	codexCommand    = "codex"
	codexVersion    = "0.42.0"

	// vueCliScopedName is a scoped package whose bin map never spells the
	// package name or its unscoped basename.
	vueCliScopedName  = "@vue/cli"
	vueCliVersion     = "5.0.0"
	vueCommand        = "vue"
	cliServiceCommand = "cli-service"
	unscopedCliName   = "cli"
)

// Manifest fixtures. The bodies are executor answers, not files.
const (
	codexManifestBody      = `{"name":"@openai/codex","version":"0.42.0","bin":"bin/codex.js"}`
	vueCliManifestBody     = `{"name":"@vue/cli","version":"5.0.0","bin":{"vue":"bin/vue.js","cli-service":"bin/cli-service.js"}}`
	typeScriptManifestBody = `{"name":"typescript","version":"5.0.0"}`
	nullBinManifestBody    = `{"name":"fsevents","version":"1.2.9","bin":null}`
)

var errInventoryManifest = errors.New("manifest read failed")

// inventoryManifest is one stubbed package.json read.
type inventoryManifest struct {
	body string
	err  error
}

// inventoryFixture stubs every executor call Commands can make: the global
// package list, the global root, and one manifest read per package.
type inventoryFixture struct {
	listJSON  string
	listErr   error
	root      string
	rootErr   error
	manifests map[string]inventoryManifest
	calls     int
}

// newInventoryExecutor stubs the executor from the fixture and returns a
// checker for the total call count.
func newInventoryExecutor(t *testing.T, fixture *inventoryFixture) (executor output.CommandExecutor, assertCalls func()) {
	t.Helper()

	calls := 0
	executor = testutil.NewMockExecutor(func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		calls++
		result, err, known := respondInventoryCall(fixture, command, args)
		if !known {
			t.Fatalf("Commands() sent unexpected executor call %q %q", command, args)
		}
		return result, err
	})
	assertCalls = func() {
		if calls != fixture.calls {
			t.Errorf("Commands() executor calls = %d, want %d", calls, fixture.calls)
		}
	}
	return executor, assertCalls
}

// respondInventoryCall answers one executor call from the fixture; known is
// false when the fixture does not expect the call at all.
func respondInventoryCall(fixture *inventoryFixture, command string, args []string) (result *output.ExecutionResult, err error, known bool) {
	switch {
	case command == npmExecutable && slices.Equal(args, []string{listSubcommand, globalFlag, depthFlag, jsonFlag}):
		if fixture.listErr != nil {
			return nil, fixture.listErr, true
		}
		return testutil.SuccessResult(fixture.listJSON), nil, true
	case command == npmExecutable && slices.Equal(args, []string{rootSubcommand, globalFlag}):
		if fixture.rootErr != nil {
			return nil, fixture.rootErr, true
		}
		return testutil.SuccessResult(fixture.root + "\n"), nil, true
	case command == manifestCommand && len(args) == 1:
		manifest, found := fixture.manifests[args[0]]
		if !found {
			return nil, nil, false
		}
		if manifest.err != nil {
			return nil, manifest.err, true
		}
		return testutil.SuccessResult(manifest.body), nil, true
	default:
		return nil, nil, false
	}
}

// manifestPath builds the executor argument a package manifest read carries,
// independently of the adapter's own path assembly.
func manifestPath(packageName string) string {
	return filepath.Join(inventoryFixtureRoot, packageName, manifestFile)
}

// npmRecord builds the expected install record for one bin of a global npm
// package: provider npm, system-or-language kind, active, never executed.
func npmRecord(command, version, realPath string) diagnostics.InstallRecord {
	return diagnostics.InstallRecord{
		Command:    command,
		ProviderID: npmProviderID,
		Kind:       diagnostics.KindSystemOrLanguage,
		Version:    version,
		RealPath:   realPath,
		Active:     true,
		Executed:   false,
	}
}

// assertInventoryRecords compares the records with the wanted ones indexed
// by command, so record order never decides the verdict.
func assertInventoryRecords(t *testing.T, got []diagnostics.InstallRecord, want map[string]diagnostics.InstallRecord) {
	t.Helper()

	gotByCommand := make(map[string]diagnostics.InstallRecord, len(got))
	for _, record := range got {
		if _, duplicate := gotByCommand[record.Command]; duplicate {
			t.Errorf("Commands() returned duplicate command %q", record.Command)
			continue
		}
		gotByCommand[record.Command] = record
	}
	if !maps.Equal(gotByCommand, want) {
		t.Errorf("Commands() records = %#v, want %#v", gotByCommand, want)
	}
}

// assertNoCommand reports a failure when any record claims command, so a
// package or scope name that leaked into a command cannot hide.
func assertNoCommand(t *testing.T, records []diagnostics.InstallRecord, command string) {
	t.Helper()

	for _, record := range records {
		if record.Command == command {
			t.Errorf("Commands() reported command %q; that name is not a package bin", record.Command)
		}
	}
}

// runInventoryCommands drives Commands once and checks the executor call
// count before the caller's assertions run.
func runInventoryCommands(t *testing.T, fixture *inventoryFixture) []diagnostics.InstallRecord {
	t.Helper()

	executor, assertCalls := newInventoryExecutor(t, fixture)
	defer assertCalls()
	adapter := NewAdapter(executor, testutil.NewMockLogger())

	records, err := adapter.Commands(context.Background())
	if err != nil {
		t.Fatalf("Commands() unexpected error = %v", err)
	}
	return records
}

func TestNpmCommandInventory(t *testing.T) {
	t.Run("package-bin", func(t *testing.T) {
		fixture := &inventoryFixture{
			calls:    3,
			listJSON: `{"dependencies":{"@openai/codex":{"version":"0.42.0"}}}`,
			root:     inventoryFixtureRoot,
			manifests: map[string]inventoryManifest{
				manifestPath(codexScopedName): {body: codexManifestBody},
			},
		}

		records := runInventoryCommands(t, fixture)
		assertInventoryRecords(t, records, map[string]diagnostics.InstallRecord{
			codexCommand: npmRecord(codexCommand, codexVersion,
				filepath.Join(inventoryFixtureRoot, codexScopedName, "bin", "codex.js")),
		})
		assertNoCommand(t, records, codexScopedName)
	})

	t.Run("no-bin", func(t *testing.T) {
		fixture := &inventoryFixture{
			calls:    3,
			listJSON: `{"dependencies":{"typescript":{"version":"5.0.0"}}}`,
			root:     inventoryFixtureRoot,
			manifests: map[string]inventoryManifest{
				manifestPath(testNPMTypeScript): {body: typeScriptManifestBody},
			},
		}

		records := runInventoryCommands(t, fixture)
		assertInventoryRecords(t, records, map[string]diagnostics.InstallRecord{})
	})

	t.Run("scoped-name-is-not-command", func(t *testing.T) {
		fixture := &inventoryFixture{
			calls:    3,
			listJSON: `{"dependencies":{"@vue/cli":{"version":"5.0.0"}}}`,
			root:     inventoryFixtureRoot,
			manifests: map[string]inventoryManifest{
				manifestPath(vueCliScopedName): {body: vueCliManifestBody},
			},
		}

		records := runInventoryCommands(t, fixture)
		assertInventoryRecords(t, records, map[string]diagnostics.InstallRecord{
			vueCommand: npmRecord(vueCommand, vueCliVersion,
				filepath.Join(inventoryFixtureRoot, vueCliScopedName, "bin", "vue.js")),
			cliServiceCommand: npmRecord(cliServiceCommand, vueCliVersion,
				filepath.Join(inventoryFixtureRoot, vueCliScopedName, "bin", "cli-service.js")),
		})
		assertNoCommand(t, records, vueCliScopedName)
		assertNoCommand(t, records, unscopedCliName)
	})

	t.Run("empty-global-tree", func(t *testing.T) {
		fixture := &inventoryFixture{
			calls:    1,
			listJSON: `{"dependencies":{}}`,
		}

		records := runInventoryCommands(t, fixture)
		assertInventoryRecords(t, records, map[string]diagnostics.InstallRecord{})
	})

	t.Run("null-bin", func(t *testing.T) {
		fixture := &inventoryFixture{
			calls:    3,
			listJSON: `{"dependencies":{"fsevents":{"version":"1.2.9"}}}`,
			root:     inventoryFixtureRoot,
			manifests: map[string]inventoryManifest{
				manifestPath("fsevents"): {body: nullBinManifestBody},
			},
		}

		records := runInventoryCommands(t, fixture)
		assertInventoryRecords(t, records, map[string]diagnostics.InstallRecord{})
	})
}

func TestNpmCommandInventoryFailures(t *testing.T) {
	t.Run("list executor error", func(t *testing.T) {
		fixture := &inventoryFixture{
			calls:   1,
			listErr: errNPMListPackages,
		}

		executor, assertCalls := newInventoryExecutor(t, fixture)
		defer assertCalls()
		adapter := NewAdapter(executor, testutil.NewMockLogger())

		records, err := adapter.Commands(context.Background())
		if !errors.Is(err, errNPMListPackages) {
			t.Fatalf("Commands() error = %v, want errors.Is(_, %v)", err, errNPMListPackages)
		}
		if records != nil {
			t.Errorf("Commands() records = %#v on error, want nil", records)
		}
	})

	t.Run("root executor error", func(t *testing.T) {
		fixture := &inventoryFixture{
			calls:    2,
			listJSON: `{"dependencies":{"@openai/codex":{"version":"0.42.0"}}}`,
			rootErr:  errNPMListPackages,
		}

		executor, assertCalls := newInventoryExecutor(t, fixture)
		defer assertCalls()
		adapter := NewAdapter(executor, testutil.NewMockLogger())

		records, err := adapter.Commands(context.Background())
		if err == nil {
			t.Fatalf("Commands() error = nil, want an error")
		}
		if records != nil {
			t.Errorf("Commands() records = %#v on error, want nil", records)
		}
	})

	t.Run("manifest read error", func(t *testing.T) {
		fixture := &inventoryFixture{
			calls:    3,
			listJSON: `{"dependencies":{"typescript":{"version":"5.0.0"}}}`,
			root:     inventoryFixtureRoot,
			manifests: map[string]inventoryManifest{
				manifestPath(testNPMTypeScript): {err: errInventoryManifest},
			},
		}

		executor, assertCalls := newInventoryExecutor(t, fixture)
		defer assertCalls()
		adapter := NewAdapter(executor, testutil.NewMockLogger())

		records, err := adapter.Commands(context.Background())
		if !errors.Is(err, errInventoryManifest) {
			t.Fatalf("Commands() error = %v, want errors.Is(_, %v)", err, errInventoryManifest)
		}
		if records != nil {
			t.Errorf("Commands() records = %#v on error, want nil", records)
		}
	})

	t.Run("unusable bin field", func(t *testing.T) {
		fixture := &inventoryFixture{
			calls:    3,
			listJSON: `{"dependencies":{"broken":{"version":"1.0.0"}}}`,
			root:     inventoryFixtureRoot,
			manifests: map[string]inventoryManifest{
				manifestPath("broken"): {body: `{"bin":[1,2]}`},
			},
		}

		executor, assertCalls := newInventoryExecutor(t, fixture)
		defer assertCalls()
		adapter := NewAdapter(executor, testutil.NewMockLogger())

		records, err := adapter.Commands(context.Background())
		if err == nil {
			t.Fatalf("Commands() error = nil, want an unusable-bin error")
		}
		if records != nil {
			t.Errorf("Commands() records = %#v on error, want nil", records)
		}
	})
}
