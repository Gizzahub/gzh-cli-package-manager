package memory

import (
	"context"
	"errors"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterpkg "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
)

// Test-specific constants.
const (
	npmCommand   = "npm"
	pipCommand   = "pip"
	pip3Command  = "pip3"
	whichCommand = "which"
	versionFlag  = "--version"
	listCommand  = "list"
)

var errFindAllDetect = errors.New("manager probe failed")

// mockExecutor implements output.CommandExecutor for testing.
type mockExecutor struct {
	execFunc func(ctx context.Context, command string, args ...string) (*output.ExecutionResult, error)
}

func (m *mockExecutor) Execute(ctx context.Context, command string, args ...string) (*output.ExecutionResult, error) {
	if m.execFunc != nil {
		return m.execFunc(ctx, command, args...)
	}
	return &output.ExecutionResult{
		Stdout:   "",
		Stderr:   "",
		ExitCode: 1, // Default: not found
	}, nil
}

func (m *mockExecutor) ExecuteWithInput(_ context.Context, _ string, _ string, _ ...string) (*output.ExecutionResult, error) {
	return &output.ExecutionResult{ExitCode: 0}, nil
}

type scriptedCommandResult struct {
	result *output.ExecutionResult
	err    error
}

type scriptedExecutor struct {
	responses map[string]scriptedCommandResult
	calls     []string
}

func newScriptedExecutor(responses map[string]scriptedCommandResult) *scriptedExecutor {
	return &scriptedExecutor{responses: responses}
}

func scriptCommand(command string, args ...string) string {
	return command + "\x00" + strings.Join(args, "\x00")
}

func newPipScriptedExecutor() *scriptedExecutor {
	return newScriptedExecutor(map[string]scriptedCommandResult{
		scriptCommand(whichCommand, pip3Command): {
			result: &output.ExecutionResult{Stdout: "/usr/bin/pip3\n"},
		},
		scriptCommand(pip3Command, versionFlag): {
			result: &output.ExecutionResult{Stdout: "pip 23.0.1 from /usr/lib/python3.11/site-packages/pip (python 3.11)\n"},
		},
		scriptCommand(pip3Command, "config", listCommand, "--user"): {
			result: &output.ExecutionResult{},
		},
		scriptCommand(pip3Command, listCommand, "--format=json"): {
			result: &output.ExecutionResult{Stdout: `[{"name":"pip","version":"23.0.1"}]`},
		},
		scriptCommand(pip3Command, listCommand, "--outdated", "--format=json"): {
			result: &output.ExecutionResult{Stdout: "[]"},
		},
		scriptCommand(pip3Command, "check"): {
			result: &output.ExecutionResult{},
		},
	})
}

func (e *scriptedExecutor) Execute(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
	key := scriptCommand(command, args...)
	e.calls = append(e.calls, key)
	if response, found := e.responses[key]; found {
		return response.result, response.err
	}
	return &output.ExecutionResult{ExitCode: 1}, nil
}

func (e *scriptedExecutor) ExecuteWithInput(_ context.Context, _ string, _ string, _ ...string) (*output.ExecutionResult, error) {
	return &output.ExecutionResult{ExitCode: 0}, nil
}

// mockLogger implements output.Logger for testing.
type mockLogger struct{}

func (m *mockLogger) Debug(_ context.Context, _ string, _ ...output.Field)          {}
func (m *mockLogger) Info(_ context.Context, _ string, _ ...output.Field)           {}
func (m *mockLogger) Warn(_ context.Context, _ string, _ ...output.Field)           {}
func (m *mockLogger) Error(_ context.Context, _ string, _ error, _ ...output.Field) {}

type managerExpectation struct {
	installed  bool
	status     manager.Status
	version    string
	binaryPath string
	configPath string
	packages   []manager.Package
}

type findAllExpectation struct {
	managerCount   int
	installedCount int
	npm            managerExpectation
	commandSet     []string
	npmSequence    []string
	allChecked     bool
}

type findByIDTestCase struct {
	name      string
	managerID manager.ManagerID
	executor  *scriptedExecutor
	configure func(*DetectingManagerRepository)
	wantErr   string
	want      managerExpectation
	calls     []string
	noCalls   bool
}

type findInstalledTestCase struct {
	name             string
	executor         *scriptedExecutor
	configure        func(*DetectingManagerRepository)
	prepare          func(*testing.T, *DetectingManagerRepository, *scriptedExecutor)
	managerCount     int
	expectPip        bool
	pip              managerExpectation
	requiredCommands []string
	pipSequence      []string
	allUnavailable   bool
}

type managerIdentity struct {
	id          manager.ManagerID
	name        string
	managerType manager.ManagerType
	platform    manager.Platform
}

func completePipExpectation(status manager.Status) managerExpectation {
	return managerExpectation{
		installed:  true,
		status:     status,
		version:    "23.0.1",
		binaryPath: "/usr/bin/pip3",
		configPath: "~/.pip/pip.conf",
		packages: []manager.Package{
			{
				Name:           "pip",
				CurrentVersion: "23.0.1",
				Manager:        manager.ManagerPip,
				IsGlobal:       true,
				UpdateType:     manager.UpdateNone,
			},
		},
	}
}

func pipCommandSequence() []string {
	return []string{
		scriptCommand(whichCommand, pip3Command),
		scriptCommand(whichCommand, pip3Command),
		scriptCommand(pip3Command, versionFlag),
		scriptCommand(whichCommand, pip3Command),
		scriptCommand(whichCommand, pip3Command),
		scriptCommand(whichCommand, pip3Command),
		scriptCommand(pip3Command, "config", listCommand, "--user"),
		scriptCommand(whichCommand, pip3Command),
		scriptCommand(pip3Command, listCommand, "--format=json"),
		scriptCommand(pip3Command, listCommand, "--outdated", "--format=json"),
		scriptCommand(whichCommand, pip3Command),
		scriptCommand(pip3Command, "check"),
	}
}

func currentPlatformManagerCount() int {
	switch runtime.GOOS {
	case "darwin":
		return 5
	case "windows":
		return 8
	default:
		return 7
	}
}

func assertManager(t *testing.T, label string, got *manager.Manager, want *managerExpectation) {
	t.Helper()
	if got.Installed != want.installed {
		t.Errorf("%s installed = %t, want %t", label, got.Installed, want.installed)
	}
	if got.Status != want.status {
		t.Errorf("%s status = %q, want %q", label, got.Status, want.status)
	}
	if got.Version != want.version || got.BinaryPath != want.binaryPath || got.ConfigPath != want.configPath {
		t.Errorf("%s details = version %q, binary %q, config %q", label, got.Version, got.BinaryPath, got.ConfigPath)
	}
	if !slices.Equal(got.Packages, want.packages) {
		t.Errorf("%s packages = %#v, want %#v", label, got.Packages, want.packages)
	}
	if got.LastChecked.IsZero() {
		t.Errorf("%s LastChecked is zero", label)
	}
}

func assertExecutedCommands(t *testing.T, calls, want []string) {
	t.Helper()
	for _, expected := range want {
		if !slices.Contains(calls, expected) {
			t.Errorf("Execute() calls = %q, missing %q", calls, expected)
		}
	}
}

func assertCommandSubsequence(t *testing.T, calls, want []string) {
	t.Helper()

	matched := 0
	for _, call := range calls {
		if matched < len(want) && call == want[matched] {
			matched++
		}
	}
	if matched != len(want) {
		t.Errorf("Execute() calls = %q, want subsequence %q", calls, want)
	}
}

func assertFindAll(t *testing.T, managers []*manager.Manager, err error, executor *scriptedExecutor, want *findAllExpectation) {
	t.Helper()
	if err != nil {
		t.Fatalf("FindAll() error = %v", err)
	}
	if len(managers) != want.managerCount {
		t.Errorf("FindAll() manager count = %d, want %d", len(managers), want.managerCount)
	}

	installedCount := 0
	var npmManager *manager.Manager
	for _, mgr := range managers {
		if mgr.Installed {
			installedCount++
		}
		if mgr.ID == manager.ManagerNPM {
			npmManager = mgr
		}
		if want.allChecked && mgr.LastChecked.IsZero() {
			t.Errorf("FindAll() left %s unchecked after a detection failure", mgr.ID)
		}
	}
	if installedCount != want.installedCount {
		t.Errorf("FindAll() installed count = %d, want %d", installedCount, want.installedCount)
	}
	if npmManager == nil {
		t.Fatal("FindAll() did not return NPM")
	}

	assertManager(t, "NPM", npmManager, &want.npm)
	assertExecutedCommands(t, executor.calls, want.commandSet)
	assertCommandSubsequence(t, executor.calls, want.npmSequence)
}

func assertFindByIDLookupError(t *testing.T, mgr *manager.Manager, err error, executor *scriptedExecutor, wantErr string) {
	t.Helper()
	if err == nil || err.Error() != wantErr {
		t.Errorf("FindByID() error = %v, want %q", err, wantErr)
	}
	if mgr != nil {
		t.Errorf("FindByID() manager = %#v, want nil", mgr)
	}
	if len(executor.calls) != 0 {
		t.Errorf("FindByID() executed commands = %q, want none", executor.calls)
	}
}

func assertFindByIDManager(t *testing.T, mgr, stored *manager.Manager, identity managerIdentity, tt *findByIDTestCase) {
	t.Helper()
	if mgr != stored {
		t.Error("FindByID() did not return the stored manager pointer")
	}
	if mgr.ID != identity.id || mgr.Name != identity.name || mgr.Type != identity.managerType || mgr.Platform != identity.platform {
		t.Errorf("FindByID() identity = %#v, want original manager identity", mgr)
	}
	assertManager(t, "NPM", mgr, &tt.want)
	assertCommandSubsequence(t, tt.executor.calls, tt.calls)
	if tt.noCalls && len(tt.executor.calls) != 0 {
		t.Errorf("FindByID() executed commands = %q, want none", tt.executor.calls)
	}
}

func assertFindByID(t *testing.T, repo *DetectingManagerRepository, tt *findByIDTestCase) {
	t.Helper()
	if tt.wantErr != "" {
		mgr, err := repo.FindByID(context.Background(), tt.managerID)
		assertFindByIDLookupError(t, mgr, err, tt.executor, tt.wantErr)
		return
	}

	stored, err := repo.ManagerRepository.FindByID(context.Background(), tt.managerID)
	if err != nil {
		t.Fatalf("base FindByID() error = %v", err)
	}
	identity := managerIdentity{
		id:          stored.ID,
		name:        stored.Name,
		managerType: stored.Type,
		platform:    stored.Platform,
	}
	stored.LastChecked = time.Time{}

	mgr, err := repo.FindByID(context.Background(), tt.managerID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	assertFindByIDManager(t, mgr, stored, identity, tt)
}

func assertAllManagersUnavailable(t *testing.T, repo *DetectingManagerRepository) {
	t.Helper()
	for _, mgr := range repo.managers {
		if mgr.Installed || mgr.Status != manager.StatusUnavailable || mgr.LastChecked.IsZero() {
			t.Errorf("FindInstalled() left %s as installed=%t status=%q checked=%t", mgr.ID, mgr.Installed, mgr.Status, !mgr.LastChecked.IsZero())
		}
	}
}

func assertFindInstalled(t *testing.T, repo *DetectingManagerRepository, executor *scriptedExecutor, stored *manager.Manager, identity managerIdentity, tt *findInstalledTestCase) {
	t.Helper()
	managers, err := repo.FindInstalled(context.Background())
	if err != nil {
		t.Fatalf("FindInstalled() error = %v", err)
	}
	if len(managers) != tt.managerCount {
		t.Errorf("FindInstalled() manager count = %d, want %d", len(managers), tt.managerCount)
	}
	if tt.allUnavailable {
		assertAllManagersUnavailable(t, repo)
	}
	if !tt.expectPip {
		assertExecutedCommands(t, executor.calls, tt.requiredCommands)
		return
	}

	var pipManager *manager.Manager
	for _, mgr := range managers {
		if mgr.ID == manager.ManagerPip {
			pipManager = mgr
		}
	}
	if pipManager == nil {
		t.Fatal("FindInstalled() did not return Pip")
	}
	if pipManager != stored {
		t.Error("FindInstalled() did not return the stored Pip manager pointer")
	}
	if pipManager.ID != identity.id || pipManager.Name != identity.name || pipManager.Type != identity.managerType || pipManager.Platform != identity.platform {
		t.Errorf("FindInstalled() Pip identity = %#v, want original manager identity", pipManager)
	}
	assertManager(t, "Pip", pipManager, &tt.pip)
	assertExecutedCommands(t, executor.calls, tt.requiredCommands)
	assertCommandSubsequence(t, executor.calls, tt.pipSequence)
}

func TestDetectingManagerRepository_FindAll(t *testing.T) {
	tests := []struct {
		name      string
		executor  *scriptedExecutor
		configure func(*DetectingManagerRepository)
		want      findAllExpectation
	}{
		{
			name:     "leaves every manager unavailable when all probes fail",
			executor: newScriptedExecutor(nil),
			want: findAllExpectation{
				managerCount:   currentPlatformManagerCount(),
				installedCount: 0,
				npm: managerExpectation{
					status: manager.StatusUnavailable,
				},
				commandSet: []string{
					scriptCommand(whichCommand, npmCommand),
				},
			},
		},
		{
			name: "persists complete NPM details without assuming manager order",
			executor: newScriptedExecutor(map[string]scriptedCommandResult{
				scriptCommand(whichCommand, npmCommand): {
					result: &output.ExecutionResult{Stdout: "/usr/bin/npm\n"},
				},
				scriptCommand(npmCommand, versionFlag): {
					result: &output.ExecutionResult{Stdout: "10.0.0\n"},
				},
				scriptCommand(npmCommand, "config", "get", "prefix"): {
					result: &output.ExecutionResult{Stdout: "/usr/local\n"},
				},
				scriptCommand(npmCommand, listCommand, "-g", "--depth=0", "--json"): {
					result: &output.ExecutionResult{Stdout: `{"dependencies":{"typescript":{"version":"5.0.0"}}}`},
				},
				scriptCommand(npmCommand, "outdated", "-g", "--json"): {
					result: &output.ExecutionResult{Stdout: "{}"},
				},
				scriptCommand(npmCommand, "doctor"): {
					result: &output.ExecutionResult{Stdout: "ok\n"},
				},
			}),
			want: findAllExpectation{
				managerCount:   currentPlatformManagerCount(),
				installedCount: 1,
				npm: managerExpectation{
					installed:  true,
					status:     manager.StatusHealthy,
					version:    "10.0.0",
					binaryPath: "/usr/bin/npm",
					configPath: "/usr/local",
					packages: []manager.Package{
						{
							Name:           "typescript",
							CurrentVersion: "5.0.0",
							Manager:        manager.ManagerNPM,
							IsGlobal:       true,
							UpdateType:     manager.UpdateNone,
						},
					},
				},
				npmSequence: []string{
					scriptCommand(whichCommand, npmCommand),
					scriptCommand(npmCommand, versionFlag),
					scriptCommand(whichCommand, npmCommand),
					scriptCommand(npmCommand, "config", "get", "prefix"),
					scriptCommand(npmCommand, listCommand, "-g", "--depth=0", "--json"),
					scriptCommand(npmCommand, "outdated", "-g", "--json"),
					scriptCommand(npmCommand, "doctor"),
				},
			},
		},
		{
			name:     "continues after a manager detection error",
			executor: newScriptedExecutor(nil),
			configure: func(repo *DetectingManagerRepository) {
				for _, mgr := range repo.managers {
					mgr.LastChecked = time.Time{}
				}
				repo.adapters[manager.ManagerNPM] = detectErrorAdapter{err: errFindAllDetect}
			},
			want: findAllExpectation{
				managerCount:   currentPlatformManagerCount(),
				installedCount: 0,
				npm: managerExpectation{
					status: manager.StatusError,
				},
				allChecked: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewDetectingManagerRepository(tt.executor, &mockLogger{})
			if tt.configure != nil {
				tt.configure(repo)
			}
			managers, err := repo.FindAll(context.Background())
			assertFindAll(t, managers, err, tt.executor, &tt.want)
		})
	}
}

func TestDetectingManagerRepository_FindInstalled(t *testing.T) {
	tests := []findInstalledTestCase{
		{
			name:     "filters every unavailable manager after eager detection",
			executor: newScriptedExecutor(nil),
			configure: func(repo *DetectingManagerRepository) {
				for _, mgr := range repo.managers {
					mgr.LastChecked = time.Time{}
				}
			},
			allUnavailable: true,
			requiredCommands: []string{
				scriptCommand(whichCommand, pip3Command),
				scriptCommand(whichCommand, pipCommand),
			},
		},
		{
			name:         "returns complete healthy Pip details through the canonical pointer",
			executor:     newPipScriptedExecutor(),
			managerCount: 1,
			expectPip:    true,
			pip:          completePipExpectation(manager.StatusHealthy),
			pipSequence:  pipCommandSequence(),
		},
		{
			name:         "retains previously installed Pip after a detection error",
			executor:     newPipScriptedExecutor(),
			managerCount: 1,
			expectPip:    true,
			pip:          completePipExpectation(manager.StatusError),
			prepare: func(t *testing.T, repo *DetectingManagerRepository, executor *scriptedExecutor) {
				t.Helper()
				managers, err := repo.FindInstalled(context.Background())
				if err != nil || len(managers) != 1 || managers[0].ID != manager.ManagerPip {
					t.Fatalf("initial FindInstalled() managers = %#v, error = %v", managers, err)
				}
				repo.managers[manager.ManagerPip].LastChecked = time.Time{}
				repo.adapters[manager.ManagerPip] = detectErrorAdapter{err: errFindAllDetect}
				executor.calls = nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewDetectingManagerRepository(tt.executor, &mockLogger{})
			if tt.configure != nil {
				tt.configure(repo)
			}
			stored, err := repo.ManagerRepository.FindByID(context.Background(), manager.ManagerPip)
			if err != nil {
				t.Fatalf("base FindByID() error = %v", err)
			}
			identity := managerIdentity{
				id:          stored.ID,
				name:        stored.Name,
				managerType: stored.Type,
				platform:    stored.Platform,
			}
			if tt.prepare != nil {
				tt.prepare(t, repo, tt.executor)
			}
			assertFindInstalled(t, repo, tt.executor, stored, identity, &tt)
		})
	}
}

func TestDetectingManagerRepository_FindByID(t *testing.T) {
	tests := []findByIDTestCase{
		{
			name:      "returns unavailable NPM after a failed probe",
			managerID: manager.ManagerNPM,
			executor:  newScriptedExecutor(nil),
			want: managerExpectation{
				status: manager.StatusUnavailable,
			},
			calls: []string{
				scriptCommand(whichCommand, npmCommand),
			},
		},
		{
			name:      "persists complete NPM details through the canonical manager pointer",
			managerID: manager.ManagerNPM,
			executor: newScriptedExecutor(map[string]scriptedCommandResult{
				scriptCommand(whichCommand, npmCommand): {
					result: &output.ExecutionResult{Stdout: "/usr/bin/npm\n"},
				},
				scriptCommand(npmCommand, versionFlag): {
					result: &output.ExecutionResult{Stdout: "10.0.0\n"},
				},
				scriptCommand(npmCommand, "config", "get", "prefix"): {
					result: &output.ExecutionResult{Stdout: "/usr/local\n"},
				},
				scriptCommand(npmCommand, listCommand, "-g", "--depth=0", "--json"): {
					result: &output.ExecutionResult{Stdout: `{"dependencies":{"typescript":{"version":"5.0.0"}}}`},
				},
				scriptCommand(npmCommand, "outdated", "-g", "--json"): {
					result: &output.ExecutionResult{Stdout: "{}"},
				},
				scriptCommand(npmCommand, "doctor"): {
					result: &output.ExecutionResult{Stdout: "ok\n"},
				},
			}),
			want: managerExpectation{
				installed:  true,
				status:     manager.StatusHealthy,
				version:    "10.0.0",
				binaryPath: "/usr/bin/npm",
				configPath: "/usr/local",
				packages: []manager.Package{
					{
						Name:           "typescript",
						CurrentVersion: "5.0.0",
						Manager:        manager.ManagerNPM,
						IsGlobal:       true,
						UpdateType:     manager.UpdateNone,
					},
				},
			},
			calls: []string{
				scriptCommand(whichCommand, npmCommand),
				scriptCommand(npmCommand, versionFlag),
				scriptCommand(whichCommand, npmCommand),
				scriptCommand(npmCommand, "config", "get", "prefix"),
				scriptCommand(npmCommand, listCommand, "-g", "--depth=0", "--json"),
				scriptCommand(npmCommand, "outdated", "-g", "--json"),
				scriptCommand(npmCommand, "doctor"),
			},
		},
		{
			name:      "returns the lookup error without probing an invalid manager",
			managerID: "invalid-manager",
			executor:  newScriptedExecutor(nil),
			wantErr:   "manager not found: invalid-manager",
		},
		{
			name:      "keeps a detection failure as partial manager state",
			managerID: manager.ManagerNPM,
			executor:  newScriptedExecutor(nil),
			configure: func(repo *DetectingManagerRepository) {
				repo.adapters[manager.ManagerNPM] = detectErrorAdapter{err: errFindAllDetect}
			},
			want: managerExpectation{
				status: manager.StatusError,
			},
			noCalls: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewDetectingManagerRepository(tt.executor, &mockLogger{})
			if tt.configure != nil {
				tt.configure(repo)
			}
			assertFindByID(t, repo, &tt)
		})
	}
}

func TestDetectingManagerRepository_AdapterRegistration(t *testing.T) {
	executor := &mockExecutor{}
	logger := &mockLogger{}
	repo := NewDetectingManagerRepository(executor, logger)

	// Verify all adapters are registered
	expectedAdapters := []manager.ManagerID{
		manager.ManagerApt,
		manager.ManagerHomebrew,
		manager.ManagerPacman,
		manager.ManagerNPM,
		manager.ManagerPip,
		manager.ManagerCargo,
		manager.ManagerASDF,
		manager.ManagerWinget,
		manager.ManagerScoop,
		manager.ManagerChocolatey,
	}

	for _, id := range expectedAdapters {
		if _, exists := repo.adapters[id]; !exists {
			t.Errorf("Adapter not registered: %s", id)
		}
	}

	// Verify adapter count
	if len(repo.adapters) != len(expectedAdapters) {
		t.Errorf("Adapter count = %d, want %d", len(repo.adapters), len(expectedAdapters))
	}
}

type detectErrorAdapter struct {
	err error
}

func (a detectErrorAdapter) Detect(context.Context) (bool, error) { return false, a.err }
func (a detectErrorAdapter) GetVersion(context.Context) (string, error) {
	return "", nil
}

func (a detectErrorAdapter) GetBinaryPath(context.Context) (string, error) {
	return "", nil
}

func (a detectErrorAdapter) GetConfigPath(context.Context) (string, error) {
	return "", nil
}

func (a detectErrorAdapter) ListPackages(context.Context) ([]manager.Package, error) {
	return nil, nil
}

func (a detectErrorAdapter) CheckHealth(context.Context) (manager.Status, error) {
	return "", nil
}

func (a detectErrorAdapter) Update(context.Context, adapterpkg.UpdateOptions) (*adapterpkg.UpdateResult, error) {
	return nil, nil
}

func TestDetectingManagerRepository_detectAndUpdate_wrapsDetectError(t *testing.T) {
	sentinel := errors.New("probe failed")
	repo := NewDetectingManagerRepository(&mockExecutor{}, &mockLogger{})
	repo.adapters[manager.ManagerNPM] = detectErrorAdapter{err: sentinel}

	mgr := &manager.Manager{ID: manager.ManagerNPM}
	err := repo.detectAndUpdate(context.Background(), mgr)
	if err == nil {
		t.Fatal("expected detect error")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("errors.Is(sentinel) = false, err=%v", err)
	}
	const want = "detect npm: probe failed"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
	if mgr.Status != manager.StatusError {
		t.Errorf("status = %v, want %v", mgr.Status, manager.StatusError)
	}
}
