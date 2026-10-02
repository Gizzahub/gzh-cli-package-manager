package mise

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/diagnostics"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/testutil"
)

// Subtest labels required by the TASK-277 completion criteria.
const (
	subtestInactiveTool = "inactive-tool"
	subtestActiveFlag   = "active-flag"
	subtestUpdateList   = "update-list-stays-current"
)

// Fixture identities for the inventory scenarios.
const (
	fixtureActiveTool   = "node"
	fixtureInactiveTool = "deno"
	fixtureUninstalled  = "bun"
	activeToolVersion   = "22.14.0"
	inactiveToolVersion = "1.0.0"
)

// inventoryFixture describes one active installed tool, one inactive
// installed tool, and one tool that is not installed.
const inventoryFixture = `{"node":[{"version":"22.14.0","requested_version":"22","installed":true,"active":true}],` +
	`"deno":[{"version":"1.0.0","installed":true,"active":false}],` +
	`"bun":[{"version":"1.3.0","installed":false,"active":false}]}`

// lsSelection matches every mise ls invocation, whatever the scope flag is.
const lsSelection = " ls "

func TestMiseCommandInventory(t *testing.T) {
	t.Run(subtestInactiveTool, testInactiveTool)
	t.Run(subtestActiveFlag, testActiveFlag)
	t.Run(subtestUpdateList, testUpdateList)
}

// testInactiveTool requires the installed inactive tool to appear inactive,
// unexecuted, and carrying its version, while a not-installed tool stays out.
func testInactiveTool(t *testing.T) {
	t.Helper()
	var calls []string
	records := collectInventoryRecords(t, &calls)
	record, found := findRecord(records, fixtureInactiveTool)
	if !found {
		t.Fatalf("Commands() omitted the installed inactive tool %q; records = %#v", fixtureInactiveTool, records)
	}
	if record.Active {
		t.Errorf("installed inactive tool reported active: %#v", record)
	}
	if record.Executed {
		t.Errorf("inventory record reported executed: %#v", record)
	}
	if record.Version != inactiveToolVersion {
		t.Errorf("inactive tool version = %q, want %q", record.Version, inactiveToolVersion)
	}
	if _, found := findRecord(records, fixtureUninstalled); found {
		t.Errorf("Commands() reported %q, which is not installed", fixtureUninstalled)
	}
}

// testActiveFlag requires the installed active tool to appear active as a
// version-manager record of the mise provider with no resolved path.
func testActiveFlag(t *testing.T) {
	t.Helper()
	var calls []string
	records := collectInventoryRecords(t, &calls)
	record, found := findRecord(records, fixtureActiveTool)
	if !found {
		t.Fatalf("Commands() omitted the installed active tool %q; records = %#v", fixtureActiveTool, records)
	}
	if !record.Active {
		t.Errorf("installed active tool reported inactive: %#v", record)
	}
	if record.Kind != diagnostics.KindVersionManager {
		t.Errorf("provider kind = %q, want %q", record.Kind, diagnostics.KindVersionManager)
	}
	if record.ProviderID != command {
		t.Errorf("provider id = %q, want %q", record.ProviderID, command)
	}
	if record.Version != activeToolVersion {
		t.Errorf("active tool version = %q, want %q", record.Version, activeToolVersion)
	}
	if record.RealPath != "" {
		t.Errorf("real path = %q, want empty; shim resolution belongs to the path step", record.RealPath)
	}
}

// testUpdateList requires ListPackages to keep omitting the inactive tool
// while the active tool stays listed.
func testUpdateList(t *testing.T) {
	t.Helper()
	var calls []string
	packages, err := newInventoryAdapter(&calls).ListPackages(context.Background())
	if err != nil {
		t.Fatalf("ListPackages() error = %v", err)
	}
	for i := range packages {
		if packages[i].Name == fixtureInactiveTool {
			t.Fatalf("ListPackages() includes the inactive tool %q; the update listing must stay active-only: %#v",
				fixtureInactiveTool, packages)
		}
	}
	if !hasPackage(packages, fixtureActiveTool) {
		t.Fatalf("ListPackages() lost the active tool %q; packages = %#v", fixtureActiveTool, packages)
	}
}

// newInventoryAdapter returns an adapter whose executor serves the inventory
// fixture for every mise ls query and rejects any other command.
func newInventoryAdapter(calls *[]string) *Adapter {
	return NewAdapter(testutil.NewMockExecutor(func(_ context.Context, cmd string, args ...string) (*output.ExecutionResult, error) {
		call := cmd + " " + strings.Join(args, " ")
		*calls = append(*calls, call)
		if strings.Contains(call, lsSelection) {
			return testutil.SuccessResult(inventoryFixture), nil
		}
		return nil, errors.New("unexpected command: " + call)
	}), testutil.NewMockLogger())
}

// collectInventoryRecords runs Commands once and asserts the query stayed a
// separate full listing rather than the update listing's selection.
func collectInventoryRecords(t *testing.T, calls *[]string) []diagnostics.InstallRecord {
	t.Helper()
	records, err := newInventoryAdapter(calls).Commands(context.Background())
	if err != nil {
		t.Fatalf("Commands() error = %v", err)
	}
	assertInventoryQuery(t, *calls)
	return records
}

// assertInventoryQuery requires exactly one executor call, a full mise ls
// JSON listing without the update listing's current or local selection.
func assertInventoryQuery(t *testing.T, calls []string) {
	t.Helper()
	if len(calls) != 1 {
		t.Fatalf("inventory made %d executor calls, want exactly one: %q", len(calls), calls)
	}
	if !strings.Contains(calls[0], lsSelection) || !strings.Contains(calls[0], "--json") {
		t.Fatalf("inventory call %q is not a mise ls --json query", calls[0])
	}
	if strings.Contains(calls[0], "--current") || strings.Contains(calls[0], "--local") {
		t.Fatalf("inventory call %q reused the update listing selection; it must be a separate query", calls[0])
	}
}

// findRecord locates one install record by command basename.
func findRecord(records []diagnostics.InstallRecord, name string) (diagnostics.InstallRecord, bool) {
	for i := range records {
		if records[i].Command == name {
			return records[i], true
		}
	}
	return diagnostics.InstallRecord{}, false
}

// hasPackage reports whether the package list contains one name.
func hasPackage(packages []manager.Package, name string) bool {
	for i := range packages {
		if packages[i].Name == name {
			return true
		}
	}
	return false
}
