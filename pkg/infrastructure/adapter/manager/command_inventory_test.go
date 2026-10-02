package manager

import (
	"context"
	"reflect"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/diagnostics"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
)

// Subtest labels the card pins for TestCommandInventoryEmptyDefault.
const (
	labelEmptyAdapter          = "empty-adapter"
	labelListPackagesUntouched = "list-packages-untouched"
	labelSuppliedCommands      = "supplied-commands"
	labelAdapterMethodSet      = "adapter-method-set"
)

// adapterMethods is the exact method set of Adapter the card pins, in the
// alphabetical order reflect reports for interface methods.
var adapterMethods = []string{
	"CheckHealth",
	"Detect",
	"GetBinaryPath",
	"GetConfigPath",
	"GetVersion",
	"ListPackages",
	"Update",
}

// Compile-time proof that the fixtures satisfy the interfaces they claim and
// that the capability stays optional: a plain adapter remains a valid Adapter
// without CommandInventory.
var (
	_ Adapter          = (*stubAdapter)(nil)
	_ Adapter          = (*inventoryAdapter)(nil)
	_ CommandInventory = (*inventoryAdapter)(nil)
)

// stubAdapter is an Adapter that never touches the machine and does not opt
// in to CommandInventory. The counters let tests prove the collector never
// lists packages or updates anything.
type stubAdapter struct {
	listCalls   int
	updateCalls int
}

// Detect reports the manager as absent without probing the system.
func (s *stubAdapter) Detect(_ context.Context) (bool, error) { return false, nil }

// GetVersion returns an empty version without running anything.
func (s *stubAdapter) GetVersion(_ context.Context) (string, error) { return "", nil }

// GetBinaryPath returns an empty path without looking at the machine.
func (s *stubAdapter) GetBinaryPath(_ context.Context) (string, error) { return "", nil }

// GetConfigPath returns an empty path without looking at the machine.
func (s *stubAdapter) GetConfigPath(_ context.Context) (string, error) { return "", nil }

// ListPackages counts the call and returns no packages.
func (s *stubAdapter) ListPackages(_ context.Context) ([]manager.Package, error) {
	s.listCalls++
	return nil, nil
}

// CheckHealth reports a healthy manager without probing anything.
func (s *stubAdapter) CheckHealth(_ context.Context) (manager.Status, error) {
	return manager.StatusHealthy, nil
}

// Update counts the call and reports an unchanged manager.
func (s *stubAdapter) Update(_ context.Context, _ UpdateOptions) (*UpdateResult, error) {
	s.updateCalls++
	return &UpdateResult{}, nil
}

// inventoryAdapter is a stubAdapter that additionally opts in to
// CommandInventory and hands back fixed records.
type inventoryAdapter struct {
	stubAdapter
	records []diagnostics.InstallRecord
}

// Commands returns the fixed records without touching the machine.
func (i *inventoryAdapter) Commands(_ context.Context) ([]diagnostics.InstallRecord, error) {
	return i.records, nil
}

// TestCommandInventoryEmptyDefault pins the default-empty contract of the
// CommandInventory capability: an adapter that does not opt in contributes no
// commands, the collector never lists packages, an opted-in adapter receives
// its own records back, and the Adapter interface keeps its exact method set.
func TestCommandInventoryEmptyDefault(t *testing.T) {
	t.Run(labelEmptyAdapter, testEmptyAdapter)
	t.Run(labelListPackagesUntouched, testListPackagesUntouched)
	t.Run(labelSuppliedCommands, testSuppliedCommands)
	t.Run(labelAdapterMethodSet, testAdapterMethodSet)
}

// testEmptyAdapter asserts that an Adapter without CommandInventory yields no
// commands and a nil error.
func testEmptyAdapter(t *testing.T) {
	t.Helper()
	stub := &stubAdapter{}
	assertNotCommandInventory(t, stub)
	records, err := CollectCommands(context.Background(), stub)
	if err != nil {
		t.Fatalf("CollectCommands returned an error for an adapter without CommandInventory: %v", err)
	}
	if records == nil {
		t.Fatal("CollectCommands returned nil records; want an empty slice")
	}
	if len(records) != 0 {
		t.Fatalf("CollectCommands returned %d records; want 0", len(records))
	}
}

// assertNotCommandInventory guards the fixture premise: if the adapter
// implemented CommandInventory the empty-default path would not be exercised.
func assertNotCommandInventory(t *testing.T, mgr Adapter) {
	t.Helper()
	if _, ok := mgr.(CommandInventory); ok {
		t.Fatal("stubAdapter unexpectedly implements CommandInventory; the empty-default path is not exercised")
	}
}

// testListPackagesUntouched asserts that collecting from a non-opting adapter
// leaves its package listing untouched: the counted ListPackages and Update
// calls both stay at zero.
func testListPackagesUntouched(t *testing.T) {
	t.Helper()
	stub := &stubAdapter{}
	records, err := CollectCommands(context.Background(), stub)
	if err != nil {
		t.Fatalf("CollectCommands returned an error: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("CollectCommands returned %d records; want 0", len(records))
	}
	if stub.listCalls != 0 {
		t.Fatalf("ListPackages was called %d times; want 0", stub.listCalls)
	}
	if stub.updateCalls != 0 {
		t.Fatalf("Update was called %d times; want 0", stub.updateCalls)
	}
}

// testSuppliedCommands asserts that an opted-in adapter receives its own
// records back unchanged, with a nil error.
func testSuppliedCommands(t *testing.T) {
	t.Helper()
	supplied := []diagnostics.InstallRecord{
		{
			Command:    "claude",
			ProviderID: "npm",
			Kind:       diagnostics.KindSystemOrLanguage,
			Version:    "2.0.14",
			RealPath:   "/usr/local/lib/node_modules/@anthropic-ai/claude-code/cli.js",
			Active:     true,
			Executed:   true,
		},
		{
			Command:    "claude",
			ProviderID: "homebrew",
			Kind:       diagnostics.KindSystemOrLanguage,
			Version:    "1.9.0",
			RealPath:   "/usr/local/bin/claude",
			Active:     false,
			Executed:   false,
		},
	}
	stub := &inventoryAdapter{records: supplied}
	records, err := CollectCommands(context.Background(), stub)
	if err != nil {
		t.Fatalf("CollectCommands returned an error for an adapter with CommandInventory: %v", err)
	}
	if !reflect.DeepEqual(records, supplied) {
		t.Fatalf("CollectCommands returned %+v; want the supplied records %+v", records, supplied)
	}
}

// testAdapterMethodSet asserts that the Adapter interface still declares
// exactly the seven methods the card pins, so the capability can never grow
// into the required surface unnoticed.
func testAdapterMethodSet(t *testing.T) {
	t.Helper()
	got := adapterMethodNames()
	if !reflect.DeepEqual(got, adapterMethods) {
		t.Fatalf("Adapter method set changed: got %v; want %v", got, adapterMethods)
	}
}

// adapterMethodNames lists the method names reflect reports for the Adapter
// interface type.
func adapterMethodNames() []string {
	typ := reflect.TypeFor[Adapter]()
	names := make([]string, 0, typ.NumMethod())
	for i := 0; i < typ.NumMethod(); i++ {
		names = append(names, typ.Method(i).Name)
	}
	return names
}
