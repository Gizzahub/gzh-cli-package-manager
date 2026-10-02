package homebrew

import (
	"context"
	"reflect"
	"slices"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/diagnostics"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/testutil"
)

// Subtest labels the card pins for TestHomebrewCommandInventory.
const (
	labelFormulaBinary         = "formula-binary"
	labelCaskAppOnly           = "cask-app-only"
	labelCaskTokenIsNotCommand = "cask-token-is-not-command"
)

const (
	// testBrewProviderID pins the literal provider id the inventory records
	// must carry, independent of the implementation's own constant.
	testBrewProviderID = "brew"

	// Package identity the fixtures build on. The formula is named
	// claude-code and installs the binary claude; the cask is the token
	// claude with the display name Claude. None of the names is a command
	// on its own: only a binary basename can be one.
	testFormulaName     = "claude-code"
	testFormulaVersion  = "2.0.14"
	testBinaryCommand   = "claude"
	testCaskToken       = "claude"
	testCaskDisplayName = "Claude"
	testCaskVersion     = "1.1.0"

	// The fixture binary paths the records must report as their real paths.
	testFormulaBinaryPath = "/opt/homebrew/bin/claude"
	testCaskBinaryPath    = "/opt/homebrew/bin/claude-code"
)

// formulaBinaryDocument is a brew info document whose formula claude-code
// installs the binary claude.
const formulaBinaryDocument = `{
	"formulae": [
		{
			"name": "claude-code",
			"version": "2.0.14",
			"binaries": ["/opt/homebrew/bin/claude"]
		}
	],
	"casks": []
}`

// caskAppOnlyDocument is a brew info document whose cask claude carries an
// app artifact and no binary artifact.
const caskAppOnlyDocument = `{
	"formulae": [],
	"casks": [
		{
			"token": "claude",
			"name": ["Claude"],
			"version": "1.1.0",
			"artifacts": [
				{"app": ["Claude.app"]}
			]
		}
	]
}`

// caskBinaryArtifactDocument is a brew info document whose cask claude
// carries a binary artifact installing the file claude-code.
const caskBinaryArtifactDocument = `{
	"formulae": [],
	"casks": [
		{
			"token": "claude",
			"name": ["Claude"],
			"version": "1.1.0",
			"artifacts": [
				{"binary": [{"source": "/opt/homebrew/bin/claude-code"}]}
			]
		}
	]
}`

// brewInventoryArgs is the exact read-only query the inventory must run.
var brewInventoryArgs = []string{"info", "--json=v2", "--installed"}

// newInventoryExecutor returns a stubbed executor answering the brew info
// query with document, plus a verify that fails when the inventory ran any
// other command or needed more than the single read-only query.
func newInventoryExecutor(t *testing.T, document string) (executor testutil.ExecutorFunc, verify func()) {
	t.Helper()
	calls := 0
	executor = func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		t.Helper()
		calls++
		if command != brewCommand || !slices.Equal(args, brewInventoryArgs) {
			t.Fatalf("unexpected command %q %v", command, args)
		}
		return testutil.SuccessResult(document), nil
	}
	verify = func() {
		t.Helper()
		if calls != 1 {
			t.Errorf("executor calls = %d, want 1", calls)
		}
	}
	return executor, verify
}

// newInventoryAdapter builds an adapter over one stubbed executor response.
func newInventoryAdapter(t *testing.T, document string) *Adapter {
	t.Helper()
	executor, _ := newInventoryExecutor(t, document)
	return NewAdapter(testutil.NewMockExecutor(executor), testutil.NewMockLogger())
}

// runInventoryCommands executes Commands and fails the test on any error.
func runInventoryCommands(t *testing.T, inventory *Adapter) []diagnostics.InstallRecord {
	t.Helper()
	records, err := inventory.Commands(context.Background())
	if err != nil {
		t.Fatalf("Commands() unexpected error: %v", err)
	}
	return records
}

// assertInventoryRecords compares inventory records with the exact expected
// records, so any change to how a command is derived fails the test.
func assertInventoryRecords(t *testing.T, records, want []diagnostics.InstallRecord) {
	t.Helper()
	if !reflect.DeepEqual(records, want) {
		t.Fatalf("Commands() = %+v, want %+v", records, want)
	}
}

// assertNoCommandWithBasename guards the card rule that a cask token and a
// cask display name are never reported as commands: no record may carry one
// of the forbidden basenames.
func assertNoCommandWithBasename(t *testing.T, records []diagnostics.InstallRecord, forbidden []string) {
	t.Helper()
	for i := range records {
		for _, name := range forbidden {
			if records[i].Command == name {
				t.Errorf("Commands() reported forbidden command %q", name)
			}
		}
	}
}

// TestHomebrewCommandInventory pins the Homebrew command-inventory contract
// through executor fixtures: a formula command comes from its binary, an
// app-only cask provides no command, and a cask token or display name is
// never a command.
func TestHomebrewCommandInventory(t *testing.T) {
	t.Run(labelFormulaBinary, testFormulaBinary)
	t.Run(labelCaskAppOnly, testCaskAppOnly)
	t.Run(labelCaskTokenIsNotCommand, testCaskTokenIsNotCommand)
}

// testFormulaBinary asserts that formula claude-code with binary claude
// reports command claude and never the formula name.
func testFormulaBinary(t *testing.T) {
	t.Helper()
	executor, verify := newInventoryExecutor(t, formulaBinaryDocument)
	inventory := NewAdapter(testutil.NewMockExecutor(executor), testutil.NewMockLogger())
	records := runInventoryCommands(t, inventory)
	verify()

	want := []diagnostics.InstallRecord{{
		Command:    testBinaryCommand,
		ProviderID: testBrewProviderID,
		Kind:       diagnostics.KindSystemOrLanguage,
		Version:    testFormulaVersion,
		RealPath:   testFormulaBinaryPath,
		Active:     false,
		Executed:   false,
	}}
	assertInventoryRecords(t, records, want)
}

// testCaskAppOnly asserts that cask claude with an app artifact and no
// binary artifact reports no command.
func testCaskAppOnly(t *testing.T) {
	t.Helper()
	inventory := newInventoryAdapter(t, caskAppOnlyDocument)
	records := runInventoryCommands(t, inventory)

	assertInventoryRecords(t, records, []diagnostics.InstallRecord{})
	assertNoCommandWithBasename(t, records, []string{testCaskToken, testCaskDisplayName})
}

// testCaskTokenIsNotCommand asserts that the cask token claude and the
// display name Claude are not commands: the cask whose binary artifact
// installs claude-code reports command claude-code.
func testCaskTokenIsNotCommand(t *testing.T) {
	t.Helper()
	inventory := newInventoryAdapter(t, caskBinaryArtifactDocument)
	records := runInventoryCommands(t, inventory)

	want := []diagnostics.InstallRecord{{
		Command:    testFormulaName,
		ProviderID: testBrewProviderID,
		Kind:       diagnostics.KindSystemOrLanguage,
		Version:    testCaskVersion,
		RealPath:   testCaskBinaryPath,
		Active:     false,
		Executed:   false,
	}}
	assertInventoryRecords(t, records, want)
	assertNoCommandWithBasename(t, records, []string{testCaskToken, testCaskDisplayName})
}
