package homebrew

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/diagnostics"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/testutil"
)

// Subtest labels. The first three are the card-pinned labels.
const (
	labelFormulaBinary         = "formula-binary"
	labelCaskAppOnly           = "cask-app-only"
	labelCaskTokenIsNotCommand = "cask-token-is-not-command"
	labelCaskMixedArtifacts    = "cask-mixed-artifacts-decode"
	labelFormulaUnlinkedKeg    = "formula-unlinked-keg-inactive"
	labelFormulaNestedBinPaths = "formula-nested-bin-paths-ignored"
	labelNoFormulaeNoListQuery = "no-formulae-skips-list-query"
	labelFormulaListError      = "formula-list-error-propagates"
	labelFormulaListBatches    = "formula-list-batches-names"
)

// Fixture identity and real values. The formula is named claude-code and
// its linked keg ships the binary claude; the casks antigravity-ide and
// claude-code@latest rename and bare-name their binaries; only a binary
// basename can ever be a command.
const (
	testBrewProviderID = "brew"

	testFormulaName       = "claude-code"
	testFormulaKegVersion = "2.1.287"
	testFormulaCommand    = "claude"
	testFormulaBinaryPath = "/opt/homebrew/Cellar/claude-code/2.1.287/bin/claude"

	testCaskToken       = "claude"
	testCaskDisplayName = "Claude"

	testAntigravityToken         = "antigravity-ide"
	testAntigravityDisplayName   = "Antigravity IDE"
	testAntigravitySourceCommand = "antigravity-ide"
	testRenameCommand            = "agy-ide"
	testRenameLinkPath           = "/opt/homebrew/bin/agy-ide"
	testAntigravityVersion       = "1.0.0"

	testClaudeLatestToken       = "claude-code@latest"
	testClaudeLatestDisplayName = "Claude Code"
	testBareNameCommand         = "claude"
	testClaudeLatestLinkPath    = "/opt/homebrew/bin/claude"

	testCodexToken       = "codex"
	testCodexVersion     = "0.42.0"
	testCodexLinkPath    = "/opt/homebrew/bin/codex"
	testCodexDisplayName = "Codex"

	testMixedCommand  = "cmux"
	testMixedVersion  = "0.3.1"
	testMixedLinkPath = "/opt/homebrew/bin/cmux"

	testUnlinkedName       = "jq"
	testUnlinkedVersion    = "1.8.2"
	testUnlinkedBinaryPath = "/opt/homebrew/Cellar/jq/1.8.2/bin/jq"

	testNestedName = "node"
)

// formulaBinaryDocument is the brew info document of formula claude-code in
// the real entry shape: no binaries key, an installed keg list, a linked
// keg, and the keg_only flag.
const formulaBinaryDocument = `{"formulae": [{
	"name": "claude-code", "full_name": "claude-code",
	"installed": [{"version": "2.1.287", "time": 1767000000}],
	"linked_keg": "2.1.287", "keg_only": false}], "casks": []}`

// formulaBinaryListOutput is the brew list --formula claude-code answer:
// the one command binary plus library and man files that are not commands.
const formulaBinaryListOutput = "/opt/homebrew/Cellar/claude-code/2.1.287/bin/claude\n" +
	"/opt/homebrew/Cellar/claude-code/2.1.287/lib/libclaude-core.a\n" +
	"/opt/homebrew/Cellar/claude-code/2.1.287/share/man/man1/claude.1"

// caskAppOnlyDocument is the brew info document of cask token claude, whose
// artifacts contain an app and no binary stanza.
const caskAppOnlyDocument = `{"formulae": [], "casks": [{
	"token": "claude", "name": ["Claude"], "version": "1.1.0", "installed": "1.1.0",
	"artifacts": [{"app": ["Claude.app"]}]}]}`

// caskBinariesDocument carries the three real cask binary shapes the card
// pins: a rename stanza, a bare source name, and a token equal to its
// binary basename.
const caskBinariesDocument = `{"formulae": [], "casks": [
	{"token": "antigravity-ide", "name": ["Antigravity IDE"], "version": "1.0.0", "installed": "1.0.0",
		"artifacts": [{"binary": ["/Applications/Antigravity IDE.app/Contents/Resources/app/bin/antigravity-ide", {"target": "agy-ide"}], "target": "/opt/homebrew/bin/agy-ide"}]},
	{"token": "claude-code@latest", "name": ["Claude Code"], "version": "2.1.287", "installed": "2.1.287",
		"artifacts": [{"binary": ["claude"], "target": "/opt/homebrew/bin/claude"}]},
	{"token": "codex", "name": ["Codex"], "version": "0.42.0", "installed": "0.42.0",
		"artifacts": [{"binary": ["bin/codex"], "target": "/opt/homebrew/bin/codex"}]}
]}`

// caskMixedArtifactsDocument mixes the artifact kinds real casks carry: a
// plain string, an app, an uninstall, and a binary stanza whose mixed array
// holds the source, the rename options, and an element that is neither.
const caskMixedArtifactsDocument = `{"formulae": [], "casks": [{
	"token": "cmux", "name": ["cmux"], "version": "0.3.1", "installed": "0.3.1",
	"artifacts": [
		"preflight do |installer|; end",
		{"app": ["cmux.app"]},
		{"uninstall": [{"delete": "/Applications/cmux.app"}]},
		{"binary": ["/Applications/cmux.app/Contents/Resources/bin/cmux", {"target": "cmux"}, {"unknown": true}],
			"target": "/opt/homebrew/bin/cmux"}
	]}]}`

// unlinkedKegDocument is the brew info document of keg-only formula jq: the
// keg is installed but no keg is linked, so linked_keg is null.
const unlinkedKegDocument = `{"formulae": [{
	"name": "jq", "full_name": "jq",
	"installed": [{"version": "1.8.2"}],
	"linked_keg": null, "keg_only": true}], "casks": []}`

// unlinkedKegListOutput is the brew list --formula jq answer: the jq binary
// of the unlinked keg plus a library file.
const unlinkedKegListOutput = "/opt/homebrew/Cellar/jq/1.8.2/bin/jq\n" +
	"/opt/homebrew/Cellar/jq/1.8.2/lib/libjq.a"

// nestedBinDocument is the brew info document of formula node.
const nestedBinDocument = `{"formulae": [{
	"name": "node", "full_name": "node",
	"installed": [{"version": "24.10.0"}],
	"linked_keg": "24.10.0", "keg_only": false}], "casks": []}`

// nestedBinListOutput is a brew list answer whose bin-looking files all sit
// deeper than the keg's own bin directory: libexec tooling and a bin
// subdirectory.
const nestedBinListOutput = "/opt/homebrew/Cellar/node/24.10.0/libexec/bin/node\n" +
	"/opt/homebrew/Cellar/node/24.10.0/bin/corepack/npm"

// batchedFormulaeDocument holds two installed formulae so the fixture can
// pin the single brew list call for all names in document order.
const batchedFormulaeDocument = `{"formulae": [
	{"name": "claude-code", "full_name": "claude-code",
		"installed": [{"version": "2.1.287"}],
		"linked_keg": "2.1.287", "keg_only": false},
	{"name": "jq", "full_name": "jq",
		"installed": [{"version": "1.8.2"}],
		"linked_keg": null, "keg_only": true}
], "casks": []}`

// batchedFormulaeListOutput interleaves the two formulae's lines so the
// records follow the list output order.
const batchedFormulaeListOutput = "/opt/homebrew/Cellar/claude-code/2.1.287/bin/claude\n" +
	"/opt/homebrew/Cellar/jq/1.8.2/bin/jq\n" +
	"/opt/homebrew/Cellar/jq/1.8.2/lib/libjq.a"

// brewInfoArgs is the exact read-only query A argv the inventory must run.
var brewInfoArgs = []string{"info", "--json=v2", "--installed"}

// errTestListFailed is the executor failure query B must propagate.
var errTestListFailed = errors.New("brew list: exited with status 1")

// stubQuery is one read-only brew query the stub answers, with the stdout
// it answers or the executor error it fails with.
type stubQuery struct {
	args   []string
	stdout string
	err    error
}

// newInventoryStub returns a stubbed executor answering exactly the
// scripted queries and failing any unexpected command, plus a verify
// asserting each scripted query ran exactly once and nothing else did.
func newInventoryStub(t *testing.T, queries ...stubQuery) (executor testutil.ExecutorFunc, verify func()) {
	t.Helper()
	made := make(map[int]int, len(queries))
	executor = func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		t.Helper()
		if command != brewCommand {
			t.Fatalf("unexpected command %q %v", command, args)
		}
		for i := range queries {
			if !slices.Equal(args, queries[i].args) {
				continue
			}
			made[i]++
			if queries[i].err != nil {
				return nil, queries[i].err
			}
			return testutil.SuccessResult(queries[i].stdout), nil
		}
		t.Fatalf("unexpected brew %v call", args)
		return nil, nil
	}
	verify = func() {
		t.Helper()
		for i := range queries {
			if made[i] != 1 {
				t.Errorf("brew %v called %d times, want 1", queries[i].args, made[i])
			}
		}
	}
	return executor, verify
}

// newVerifiedInventoryAdapter builds an adapter over a stubbed executor and
// returns the verify function to call after Commands ran.
func newVerifiedInventoryAdapter(t *testing.T, queries ...stubQuery) (inventory *Adapter, verify func()) {
	t.Helper()
	var stubExecutor testutil.ExecutorFunc
	stubExecutor, verify = newInventoryStub(t, queries...)
	inventory = NewAdapter(testutil.NewMockExecutor(stubExecutor), testutil.NewMockLogger())
	return inventory, verify
}

// brewListArgs builds the exact query B argv for the given formula names.
func brewListArgs(names ...string) []string {
	return slices.Concat([]string{"list", "--formula"}, names)
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

// assertNoCommandWithBasename guards the card rule that package identity is
// never reported as a command: no record may carry a forbidden basename.
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

// wantRecord builds the full install record a fixture expects: the brew
// provider, the system-or-language kind, and a false Executed flag are
// common to every Homebrew record.
func wantRecord(command, version, realPath string, active bool) diagnostics.InstallRecord {
	return diagnostics.InstallRecord{
		Command:    command,
		ProviderID: testBrewProviderID,
		Kind:       diagnostics.KindSystemOrLanguage,
		Version:    version,
		RealPath:   realPath,
		Active:     active,
		Executed:   false,
	}
}

// TestHomebrewCommandInventory pins the Homebrew command-inventory contract
// through executor fixtures copied from real brew output: a formula command
// comes from its keg's bin file, an app-only cask provides no command, a
// cask token or display name is never a command, and unknown artifact and
// list shapes stay tolerated.
func TestHomebrewCommandInventory(t *testing.T) {
	t.Run(labelFormulaBinary, testFormulaBinary)
	t.Run(labelCaskAppOnly, testCaskAppOnly)
	t.Run(labelCaskTokenIsNotCommand, testCaskTokenIsNotCommand)
	t.Run(labelCaskMixedArtifacts, testCaskMixedArtifacts)
	t.Run(labelFormulaUnlinkedKeg, testFormulaUnlinkedKeg)
	t.Run(labelFormulaNestedBinPaths, testFormulaNestedBinPaths)
	t.Run(labelNoFormulaeNoListQuery, testNoFormulaeSkipsListQuery)
	t.Run(labelFormulaListError, testFormulaListError)
	t.Run(labelFormulaListBatches, testFormulaListBatches)
}

// testFormulaBinary asserts formula claude-code reports its linked keg's
// bin file claude, and never the formula name.
func testFormulaBinary(t *testing.T) {
	t.Helper()
	inventory, verify := newVerifiedInventoryAdapter(t,
		stubQuery{args: brewInfoArgs, stdout: formulaBinaryDocument},
		stubQuery{args: brewListArgs(testFormulaName), stdout: formulaBinaryListOutput},
	)
	records := runInventoryCommands(t, inventory)
	verify()

	want := []diagnostics.InstallRecord{
		wantRecord(testFormulaCommand, testFormulaKegVersion, testFormulaBinaryPath, true),
	}
	assertInventoryRecords(t, records, want)
	assertNoCommandWithBasename(t, records, []string{testFormulaName})
}

// testCaskAppOnly asserts cask claude with an app artifact and no binary
// artifact reports no command.
func testCaskAppOnly(t *testing.T) {
	t.Helper()
	inventory, verify := newVerifiedInventoryAdapter(t,
		stubQuery{args: brewInfoArgs, stdout: caskAppOnlyDocument},
	)
	records := runInventoryCommands(t, inventory)
	verify()

	assertInventoryRecords(t, records, []diagnostics.InstallRecord{})
	assertNoCommandWithBasename(t, records, []string{testCaskToken, testCaskDisplayName})
}

// testCaskTokenIsNotCommand asserts the three real cask binary shapes report
// only binary basenames: the rename wins over the source path and the
// token, the bare name wins over the token, and a token that equals its
// binary basename is still just that basename.
func testCaskTokenIsNotCommand(t *testing.T) {
	t.Helper()
	inventory, verify := newVerifiedInventoryAdapter(t,
		stubQuery{args: brewInfoArgs, stdout: caskBinariesDocument},
	)
	records := runInventoryCommands(t, inventory)
	verify()

	want := []diagnostics.InstallRecord{
		wantRecord(testRenameCommand, testAntigravityVersion, testRenameLinkPath, true),
		wantRecord(testBareNameCommand, testFormulaKegVersion, testClaudeLatestLinkPath, true),
		wantRecord(testCodexToken, testCodexVersion, testCodexLinkPath, true),
	}
	assertInventoryRecords(t, records, want)
	assertNoCommandWithBasename(t, records, []string{
		testAntigravityToken, testAntigravityDisplayName, testAntigravitySourceCommand,
		testClaudeLatestToken, testClaudeLatestDisplayName, testCodexDisplayName,
	})
}

// testCaskMixedArtifacts asserts a cask whose artifacts mix plain strings,
// apps, uninstalls, and a mixed binary array decodes without error and
// reports exactly the one renamed command.
func testCaskMixedArtifacts(t *testing.T) {
	t.Helper()
	inventory, verify := newVerifiedInventoryAdapter(t,
		stubQuery{args: brewInfoArgs, stdout: caskMixedArtifactsDocument},
	)
	records := runInventoryCommands(t, inventory)
	verify()

	want := []diagnostics.InstallRecord{
		wantRecord(testMixedCommand, testMixedVersion, testMixedLinkPath, true),
	}
	assertInventoryRecords(t, records, want)
}

// testFormulaUnlinkedKeg asserts the bin file of an installed but unlinked
// keg reports Active false.
func testFormulaUnlinkedKeg(t *testing.T) {
	t.Helper()
	inventory, verify := newVerifiedInventoryAdapter(t,
		stubQuery{args: brewInfoArgs, stdout: unlinkedKegDocument},
		stubQuery{args: brewListArgs(testUnlinkedName), stdout: unlinkedKegListOutput},
	)
	records := runInventoryCommands(t, inventory)
	verify()

	want := []diagnostics.InstallRecord{
		wantRecord(testUnlinkedName, testUnlinkedVersion, testUnlinkedBinaryPath, false),
	}
	assertInventoryRecords(t, records, want)
}

// testFormulaNestedBinPaths asserts bin-looking files deeper than the keg's
// own bin directory, libexec tooling and bin subdirectories, are ignored.
func testFormulaNestedBinPaths(t *testing.T) {
	t.Helper()
	inventory, verify := newVerifiedInventoryAdapter(t,
		stubQuery{args: brewInfoArgs, stdout: nestedBinDocument},
		stubQuery{args: brewListArgs(testNestedName), stdout: nestedBinListOutput},
	)
	records := runInventoryCommands(t, inventory)
	verify()

	assertInventoryRecords(t, records, []diagnostics.InstallRecord{})
}

// testNoFormulaeSkipsListQuery asserts a document without formulae never
// runs the brew list query: the stub scripts only the info query and fails
// on anything else.
func testNoFormulaeSkipsListQuery(t *testing.T) {
	t.Helper()
	inventory, verify := newVerifiedInventoryAdapter(t,
		stubQuery{args: brewInfoArgs, stdout: caskAppOnlyDocument},
	)
	records := runInventoryCommands(t, inventory)
	verify()

	assertInventoryRecords(t, records, []diagnostics.InstallRecord{})
}

// testFormulaListError asserts a failing brew list query fails the whole
// inventory instead of silently dropping the formulae.
func testFormulaListError(t *testing.T) {
	t.Helper()
	inventory, verify := newVerifiedInventoryAdapter(t,
		stubQuery{args: brewInfoArgs, stdout: formulaBinaryDocument},
		stubQuery{args: brewListArgs(testFormulaName), err: errTestListFailed},
	)
	_, err := inventory.Commands(context.Background())
	verify()

	if err == nil {
		t.Fatal("Commands() error = nil, want the brew list failure")
	}
	if !strings.Contains(err.Error(), errTestListFailed.Error()) {
		t.Errorf("Commands() error %q does not carry the brew list failure %q", err, errTestListFailed)
	}
}

// testFormulaListBatches asserts the inventory asks about every installed
// formula in one brew list call and reports each formula's records from the
// shared answer.
func testFormulaListBatches(t *testing.T) {
	t.Helper()
	inventory, verify := newVerifiedInventoryAdapter(t,
		stubQuery{args: brewInfoArgs, stdout: batchedFormulaeDocument},
		stubQuery{args: brewListArgs(testFormulaName, testUnlinkedName), stdout: batchedFormulaeListOutput},
	)
	records := runInventoryCommands(t, inventory)
	verify()

	want := []diagnostics.InstallRecord{
		wantRecord(testFormulaCommand, testFormulaKegVersion, testFormulaBinaryPath, true),
		wantRecord(testUnlinkedName, testUnlinkedVersion, testUnlinkedBinaryPath, false),
	}
	assertInventoryRecords(t, records, want)
}
