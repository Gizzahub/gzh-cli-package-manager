package npm

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/diagnostics"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/testutil"
)

const (
	// inventoryFixtureRoot is the global node_modules prefix the fixture
	// trees report; it only exists inside the stubbed executor answers.
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

	// geminiScopedName is a real npm 11 --long sample: a scoped package
	// whose bin file lives outside a bin directory.
	geminiScopedName = "@google/gemini-cli"
	geminiCommand    = "gemini"
	geminiVersion    = "0.62.0"

	// corepackName is a real npm 11 --long sample: an unscoped package that
	// ships several commands, one of which shares the package name.
	corepackName    = "corepack"
	corepackVersion = "0.36.0"

	// typeScriptSampleVersion is the version of the real typescript sample.
	typeScriptSampleVersion = "7.0.2"

	// legacyScopedName accepts a string-form bin defensively; the command is
	// the package name without its scope.
	legacyScopedName = "@openai/codex-legacy"
	legacyCommand    = "codex-legacy"
	legacyVersion    = "0.1.0"

	// slashBinPackage reports a bin map key that contains a slash; npm links
	// it under the key's basename.
	slashBinPackage = "weird"
	slashBinKey     = "@scope/tools-cli"
	slashCommand    = "tools-cli"
	slashVersion    = "1.2.3"
)

// inventoryListArgs is the one executor call Commands may make: a single
// npm ls of the global tree in long JSON form.
var inventoryListArgs = []string{lsSubcommand, globalFlag, depthFlag, jsonFlag, longFlag}

// Executor fixtures. Each body is the stubbed npm ls -g --depth=0 --json
// --long answer, in the shape npm 11 actually prints.
const (
	codexTreeBody = `{
  "name": "gz-global",
  "path": "/usr/local/lib/node_modules",
  "dependencies": {
    "@openai/codex": {
      "name": "@openai/codex",
      "version": "0.42.0",
      "path": "/usr/local/lib/node_modules/@openai/codex",
      "bin": {"codex": "bin/codex.js"}
    }
  }
}`

	noBinTreeBody = `{
  "name": "gz-global",
  "path": "/usr/local/lib/node_modules",
  "dependencies": {
    "fsevents": {
      "name": "fsevents",
      "version": "2.3.3",
      "path": "/usr/local/lib/node_modules/fsevents"
    },
    "left-pad": {
      "name": "left-pad",
      "version": "1.3.0",
      "path": "/usr/local/lib/node_modules/left-pad",
      "bin": null
    }
  }
}`

	vueCliTreeBody = `{
  "name": "gz-global",
  "path": "/usr/local/lib/node_modules",
  "dependencies": {
    "@vue/cli": {
      "name": "@vue/cli",
      "version": "5.0.0",
      "path": "/usr/local/lib/node_modules/@vue/cli",
      "bin": {"vue": "bin/vue.js", "cli-service": "bin/cli-service.js"}
    }
  }
}`

	samplesTreeBody = `{
  "name": "gz-global",
  "path": "/usr/local/lib/node_modules",
  "dependencies": {
    "@google/gemini-cli": {
      "name": "@google/gemini-cli",
      "version": "0.62.0",
      "path": "/usr/local/lib/node_modules/@google/gemini-cli",
      "bin": {"gemini": "bundle/gemini.js"}
    },
    "corepack": {
      "name": "corepack",
      "version": "0.36.0",
      "path": "/usr/local/lib/node_modules/corepack",
      "bin": {
        "corepack": "dist/corepack.js",
        "pnpm": "dist/pnpm.js",
        "pnpx": "dist/pnpx.js",
        "yarn": "dist/yarn.js",
        "yarnpkg": "dist/yarnpkg.js"
      }
    },
    "typescript": {
      "name": "typescript",
      "version": "7.0.2",
      "path": "/usr/local/lib/node_modules/typescript",
      "bin": {"tsc": "bin/tsc"}
    }
  }
}`

	legacyTreeBody = `{
  "name": "gz-global",
  "path": "/usr/local/lib/node_modules",
  "dependencies": {
    "@openai/codex-legacy": {
      "name": "@openai/codex-legacy",
      "version": "0.1.0",
      "path": "/usr/local/lib/node_modules/@openai/codex-legacy",
      "bin": "bin/codex.js"
    }
  }
}`

	slashBinTreeBody = `{
  "name": "gz-global",
  "path": "/usr/local/lib/node_modules",
  "dependencies": {
    "weird": {
      "name": "weird",
      "version": "1.2.3",
      "path": "/usr/local/lib/node_modules/weird",
      "bin": {"@scope/tools-cli": "bin/cli.js"}
    }
  }
}`

	emptyTreeBody = `{"name":"gz-global","path":"/usr/local/lib/node_modules","dependencies":{}}`

	brokenBinTreeBody = `{
  "name": "gz-global",
  "path": "/usr/local/lib/node_modules",
  "dependencies": {
    "broken": {
      "name": "broken",
      "version": "9.9.9",
      "path": "/usr/local/lib/node_modules/broken",
      "bin": ["not", "a", "map"]
    },
    "typescript": {
      "name": "typescript",
      "version": "7.0.2",
      "path": "/usr/local/lib/node_modules/typescript",
      "bin": {"tsc": "bin/tsc"}
    }
  }
}`
)

// inventoryFixture is the stubbed answer to the one executor call Commands
// may make.
type inventoryFixture struct {
	treeJSON string
	treeErr  error
}

// newInventoryExecutor stubs the executor with the fixture's answer and
// fails the test on any command other than the single npm ls query. A
// cleanup asserts that Commands made exactly one executor call.
func newInventoryExecutor(t *testing.T, fixture *inventoryFixture) output.CommandExecutor {
	t.Helper()

	calls := 0
	t.Cleanup(func() {
		if calls != 1 {
			t.Errorf("Commands() made %d executor calls, want exactly 1", calls)
		}
	})
	return testutil.NewMockExecutor(func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		calls++
		if command != npmExecutable || !slices.Equal(args, inventoryListArgs) {
			t.Fatalf("Commands() sent unexpected executor call %q %q", command, args)
		}
		if fixture.treeErr != nil {
			return nil, fixture.treeErr
		}
		return testutil.SuccessResult(fixture.treeJSON), nil
	})
}

// runInventoryCommands drives Commands once over the fixture and requires
// success; the executor stub fails the test on any unexpected call.
func runInventoryCommands(t *testing.T, fixture *inventoryFixture) []diagnostics.InstallRecord {
	t.Helper()

	adapter := NewAdapter(newInventoryExecutor(t, fixture), testutil.NewMockLogger())

	records, err := adapter.Commands(context.Background())
	if err != nil {
		t.Fatalf("Commands() unexpected error = %v", err)
	}
	return records
}

// runInventoryError drives Commands once over a fixture whose executor call
// fails.
func runInventoryError(t *testing.T, fixture *inventoryFixture) ([]diagnostics.InstallRecord, error) {
	t.Helper()

	adapter := NewAdapter(newInventoryExecutor(t, fixture), testutil.NewMockLogger())
	return adapter.Commands(context.Background())
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

// assertInventoryRecords requires the exact record slice, order included:
// the inventory promises package-name-then-command order.
func assertInventoryRecords(t *testing.T, got, want []diagnostics.InstallRecord) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Commands() records = %#v, want %#v", got, want)
	}
}

// assertInventoryError requires an error wrapping wantErr (any error when
// wantErr is nil) and no records.
func assertInventoryError(t *testing.T, got []diagnostics.InstallRecord, err, wantErr error) {
	t.Helper()

	if err == nil {
		t.Fatal("Commands() error = nil, want an error")
	}
	if wantErr != nil && !errors.Is(err, wantErr) {
		t.Fatalf("Commands() error = %v, want errors.Is(_, %v)", err, wantErr)
	}
	if got != nil {
		t.Errorf("Commands() records = %#v on error, want nil", got)
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

func TestNpmCommandInventory(t *testing.T) {
	t.Run("package-bin", func(t *testing.T) {
		records := runInventoryCommands(t, &inventoryFixture{treeJSON: codexTreeBody})

		assertInventoryRecords(t, records, []diagnostics.InstallRecord{
			npmRecord(codexCommand, codexVersion,
				filepath.Join(inventoryFixtureRoot, codexScopedName, "bin", "codex.js")),
		})
		assertNoCommand(t, records, codexScopedName)
	})

	t.Run("no-bin", func(t *testing.T) {
		records := runInventoryCommands(t, &inventoryFixture{treeJSON: noBinTreeBody})

		assertInventoryRecords(t, records, []diagnostics.InstallRecord{})
		assertNoCommand(t, records, "fsevents")
		assertNoCommand(t, records, "left-pad")
	})

	t.Run("scoped-name-is-not-command", func(t *testing.T) {
		records := runInventoryCommands(t, &inventoryFixture{treeJSON: vueCliTreeBody})

		assertInventoryRecords(t, records, []diagnostics.InstallRecord{
			npmRecord(cliServiceCommand, vueCliVersion,
				filepath.Join(inventoryFixtureRoot, vueCliScopedName, "bin", "cli-service.js")),
			npmRecord(vueCommand, vueCliVersion,
				filepath.Join(inventoryFixtureRoot, vueCliScopedName, "bin", "vue.js")),
		})
		assertNoCommand(t, records, vueCliScopedName)
		assertNoCommand(t, records, unscopedCliName)
	})

	t.Run("long-json-samples", func(t *testing.T) {
		records := runInventoryCommands(t, &inventoryFixture{treeJSON: samplesTreeBody})

		assertInventoryRecords(t, records, []diagnostics.InstallRecord{
			npmRecord(geminiCommand, geminiVersion,
				filepath.Join(inventoryFixtureRoot, geminiScopedName, "bundle", "gemini.js")),
			npmRecord(corepackName, corepackVersion,
				filepath.Join(inventoryFixtureRoot, corepackName, "dist", "corepack.js")),
			npmRecord("pnpm", corepackVersion,
				filepath.Join(inventoryFixtureRoot, corepackName, "dist", "pnpm.js")),
			npmRecord("pnpx", corepackVersion,
				filepath.Join(inventoryFixtureRoot, corepackName, "dist", "pnpx.js")),
			npmRecord("yarn", corepackVersion,
				filepath.Join(inventoryFixtureRoot, corepackName, "dist", "yarn.js")),
			npmRecord("yarnpkg", corepackVersion,
				filepath.Join(inventoryFixtureRoot, corepackName, "dist", "yarnpkg.js")),
			npmRecord("tsc", typeScriptSampleVersion,
				filepath.Join(inventoryFixtureRoot, testNPMTypeScript, "bin", "tsc")),
		})
		assertNoCommand(t, records, geminiScopedName)
		assertNoCommand(t, records, testNPMTypeScript)
	})

	t.Run("string-bin", func(t *testing.T) {
		records := runInventoryCommands(t, &inventoryFixture{treeJSON: legacyTreeBody})

		assertInventoryRecords(t, records, []diagnostics.InstallRecord{
			npmRecord(legacyCommand, legacyVersion,
				filepath.Join(inventoryFixtureRoot, legacyScopedName, "bin", "codex.js")),
		})
		assertNoCommand(t, records, legacyScopedName)
	})

	t.Run("bin-key-with-slash", func(t *testing.T) {
		records := runInventoryCommands(t, &inventoryFixture{treeJSON: slashBinTreeBody})

		assertInventoryRecords(t, records, []diagnostics.InstallRecord{
			npmRecord(slashCommand, slashVersion,
				filepath.Join(inventoryFixtureRoot, slashBinPackage, "bin", "cli.js")),
		})
		assertNoCommand(t, records, slashBinKey)
	})

	t.Run("empty-global-tree", func(t *testing.T) {
		records := runInventoryCommands(t, &inventoryFixture{treeJSON: emptyTreeBody})

		assertInventoryRecords(t, records, []diagnostics.InstallRecord{})
	})

	t.Run("undecodable-bin-skipped", func(t *testing.T) {
		records := runInventoryCommands(t, &inventoryFixture{treeJSON: brokenBinTreeBody})

		assertInventoryRecords(t, records, []diagnostics.InstallRecord{
			npmRecord("tsc", typeScriptSampleVersion,
				filepath.Join(inventoryFixtureRoot, testNPMTypeScript, "bin", "tsc")),
		})
	})
}

func TestNpmCommandInventoryFailures(t *testing.T) {
	t.Run("list executor error", func(t *testing.T) {
		records, err := runInventoryError(t, &inventoryFixture{treeErr: errNPMListPackages})

		assertInventoryError(t, records, err, errNPMListPackages)
	})

	t.Run("list output not json", func(t *testing.T) {
		records, err := runInventoryError(t, &inventoryFixture{treeJSON: "npm ERR! not json"})

		assertInventoryError(t, records, err, nil)
	})
}
