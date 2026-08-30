package memory

import (
	"context"
	"errors"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterpkg "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
)

// Test-specific constants.
const (
	npmCommand   = "npm"
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

type findAllManagerExpectation struct {
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
	npm            findAllManagerExpectation
	commandSet     []string
	npmSequence    []string
}

func currentPlatformManagerCount() int {
	if runtime.GOOS == "linux" {
		return 7
	}
	return 5
}

func assertFindAllManager(t *testing.T, got *manager.Manager, want *findAllManagerExpectation) {
	t.Helper()
	if got.Installed != want.installed {
		t.Errorf("NPM installed = %t, want %t", got.Installed, want.installed)
	}
	if got.Status != want.status {
		t.Errorf("NPM status = %q, want %q", got.Status, want.status)
	}
	if got.Version != want.version || got.BinaryPath != want.binaryPath || got.ConfigPath != want.configPath {
		t.Errorf("NPM details = version %q, binary %q, config %q", got.Version, got.BinaryPath, got.ConfigPath)
	}
	if !slices.Equal(got.Packages, want.packages) {
		t.Errorf("NPM packages = %#v, want %#v", got.Packages, want.packages)
	}
	if got.LastChecked.IsZero() {
		t.Error("NPM LastChecked is zero")
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
	}
	if installedCount != want.installedCount {
		t.Errorf("FindAll() installed count = %d, want %d", installedCount, want.installedCount)
	}
	if npmManager == nil {
		t.Fatal("FindAll() did not return NPM")
	}

	assertFindAllManager(t, npmManager, &want.npm)
	assertExecutedCommands(t, executor.calls, want.commandSet)
	assertCommandSubsequence(t, executor.calls, want.npmSequence)
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
				npm: findAllManagerExpectation{
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
				npm: findAllManagerExpectation{
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
				repo.adapters[manager.ManagerNPM] = detectErrorAdapter{err: errFindAllDetect}
			},
			want: findAllExpectation{
				managerCount:   currentPlatformManagerCount(),
				installedCount: 0,
				npm: findAllManagerExpectation{
					status: manager.StatusError,
				},
				commandSet: []string{
					scriptCommand(whichCommand, pip3Command),
				},
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
	tests := []struct {
		name          string
		execFunc      func(ctx context.Context, command string, args ...string) (*output.ExecutionResult, error)
		wantInstalled int
	}{
		{
			name: "no managers installed",
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return &output.ExecutionResult{ExitCode: 1}, nil
			},
			wantInstalled: 0,
		},
		{
			name: "pip detected",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == whichCommand && len(args) == 1 && args[0] == pip3Command {
					return &output.ExecutionResult{
						Stdout:   "/usr/bin/pip3\n",
						ExitCode: 0,
					}, nil
				}
				if command == pip3Command && args[0] == versionFlag {
					return &output.ExecutionResult{
						Stdout:   "pip 23.0.1 from /usr/lib/python3.11/site-packages/pip (python 3.11)\n",
						ExitCode: 0,
					}, nil
				}
				if command == pip3Command && args[0] == listCommand {
					return &output.ExecutionResult{
						Stdout:   `[{"name": "pip", "version": "23.0.1"}]`,
						ExitCode: 0,
					}, nil
				}
				if command == pip3Command && args[0] == "check" {
					return &output.ExecutionResult{
						Stdout:   "No broken requirements found.\n",
						ExitCode: 0,
					}, nil
				}
				return &output.ExecutionResult{ExitCode: 1}, nil
			},
			wantInstalled: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &mockExecutor{execFunc: tt.execFunc}
			logger := &mockLogger{}
			repo := NewDetectingManagerRepository(executor, logger)

			managers, err := repo.FindInstalled(context.Background())
			if err != nil {
				t.Errorf("FindInstalled() error = %v", err)
				return
			}

			if len(managers) != tt.wantInstalled {
				t.Errorf("FindInstalled() count = %d, want %d", len(managers), tt.wantInstalled)
			}

			// Verify all returned managers are actually installed
			for _, mgr := range managers {
				if !mgr.Installed {
					t.Errorf("FindInstalled() returned non-installed manager: %s", mgr.ID)
				}
			}
		})
	}
}

func TestDetectingManagerRepository_FindByID(t *testing.T) {
	tests := []struct {
		name          string
		managerID     manager.ManagerID
		execFunc      func(ctx context.Context, command string, args ...string) (*output.ExecutionResult, error)
		wantInstalled bool
		wantErr       bool
	}{
		{
			name:      "npm not installed",
			managerID: manager.ManagerNPM,
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return &output.ExecutionResult{ExitCode: 1}, nil
			},
			wantInstalled: false,
			wantErr:       false,
		},
		{
			name:      "npm installed",
			managerID: manager.ManagerNPM,
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == whichCommand && args[0] == npmCommand {
					return &output.ExecutionResult{
						Stdout:   "/usr/bin/npm\n",
						ExitCode: 0,
					}, nil
				}
				if command == npmCommand && args[0] == versionFlag {
					return &output.ExecutionResult{
						Stdout:   "10.0.0\n",
						ExitCode: 0,
					}, nil
				}
				if command == npmCommand && args[0] == "config" {
					return &output.ExecutionResult{
						Stdout:   "/usr/local\n",
						ExitCode: 0,
					}, nil
				}
				if command == npmCommand && args[0] == listCommand {
					return &output.ExecutionResult{
						Stdout:   `{"dependencies": {}}`,
						ExitCode: 0,
					}, nil
				}
				if command == npmCommand && args[0] == "outdated" {
					return &output.ExecutionResult{
						Stdout:   "{}",
						ExitCode: 0,
					}, nil
				}
				if command == npmCommand && args[0] == "doctor" {
					return &output.ExecutionResult{
						Stdout:   "ok\n",
						ExitCode: 0,
					}, nil
				}
				return &output.ExecutionResult{ExitCode: 1}, nil
			},
			wantInstalled: true,
			wantErr:       false,
		},
		{
			name:      "invalid manager ID",
			managerID: "invalid-manager",
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return &output.ExecutionResult{ExitCode: 1}, nil
			},
			wantInstalled: false,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &mockExecutor{execFunc: tt.execFunc}
			logger := &mockLogger{}
			repo := NewDetectingManagerRepository(executor, logger)

			mgr, err := repo.FindByID(context.Background(), tt.managerID)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if mgr.Installed != tt.wantInstalled {
					t.Errorf("FindByID() installed = %v, want %v", mgr.Installed, tt.wantInstalled)
				}
				if mgr.ID != tt.managerID {
					t.Errorf("FindByID() ID = %v, want %v", mgr.ID, tt.managerID)
				}
			}
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
