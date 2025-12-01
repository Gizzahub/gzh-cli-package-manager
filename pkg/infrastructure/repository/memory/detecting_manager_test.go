package memory

import (
	"context"
	"runtime"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
)

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

// mockLogger implements output.Logger for testing.
type mockLogger struct{}

func (m *mockLogger) Debug(_ context.Context, _ string, _ ...output.Field)          {}
func (m *mockLogger) Info(_ context.Context, _ string, _ ...output.Field)           {}
func (m *mockLogger) Warn(_ context.Context, _ string, _ ...output.Field)           {}
func (m *mockLogger) Error(_ context.Context, _ string, _ error, _ ...output.Field) {}

func TestDetectingManagerRepository_FindAll(t *testing.T) {
	// Expected manager count varies by platform
	// Darwin/macOS: 5 (homebrew, asdf, npm, pip, cargo)
	// Linux: 7 (+ apt, pacman)
	expectedManagerCount := 5
	if runtime.GOOS == "linux" {
		expectedManagerCount = 7
	}

	tests := []struct {
		name              string
		execFunc          func(ctx context.Context, command string, args ...string) (*output.ExecutionResult, error)
		wantTotalManagers int
		wantInstalled     int
	}{
		{
			name: "no managers detected",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				// All 'which' commands fail
				return &output.ExecutionResult{ExitCode: 1}, nil
			},
			wantTotalManagers: expectedManagerCount,
			wantInstalled:     0,
		},
		{
			name: "npm detected",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == "which" && len(args) == 1 && args[0] == "npm" {
					return &output.ExecutionResult{
						Stdout:   "/usr/bin/npm\n",
						ExitCode: 0,
					}, nil
				}
				if command == "npm" && args[0] == "--version" {
					return &output.ExecutionResult{
						Stdout:   "10.0.0\n",
						ExitCode: 0,
					}, nil
				}
				if command == "npm" && args[0] == "config" {
					return &output.ExecutionResult{
						Stdout:   "/usr/local\n",
						ExitCode: 0,
					}, nil
				}
				if command == "npm" && args[0] == "list" {
					return &output.ExecutionResult{
						Stdout:   `{"dependencies": {}}`,
						ExitCode: 0,
					}, nil
				}
				if command == "npm" && args[0] == "outdated" {
					return &output.ExecutionResult{
						Stdout:   "{}",
						ExitCode: 0,
					}, nil
				}
				if command == "npm" && args[0] == "doctor" {
					return &output.ExecutionResult{
						Stdout:   "ok\n",
						ExitCode: 0,
					}, nil
				}
				return &output.ExecutionResult{ExitCode: 1}, nil
			},
			wantTotalManagers: expectedManagerCount,
			wantInstalled:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &mockExecutor{execFunc: tt.execFunc}
			logger := &mockLogger{}
			repo := NewDetectingManagerRepository(executor, logger)

			managers, err := repo.FindAll(context.Background())
			if err != nil {
				t.Errorf("FindAll() error = %v", err)
				return
			}

			if len(managers) != tt.wantTotalManagers {
				t.Errorf("FindAll() total managers = %d, want %d", len(managers), tt.wantTotalManagers)
			}

			installedCount := 0
			for _, mgr := range managers {
				if mgr.Installed {
					installedCount++
				}
			}

			if installedCount != tt.wantInstalled {
				t.Errorf("FindAll() installed count = %d, want %d", installedCount, tt.wantInstalled)
			}
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
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				return &output.ExecutionResult{ExitCode: 1}, nil
			},
			wantInstalled: 0,
		},
		{
			name: "pip detected",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == "which" && len(args) == 1 && args[0] == "pip3" {
					return &output.ExecutionResult{
						Stdout:   "/usr/bin/pip3\n",
						ExitCode: 0,
					}, nil
				}
				if command == "pip3" && args[0] == "--version" {
					return &output.ExecutionResult{
						Stdout:   "pip 23.0.1 from /usr/lib/python3.11/site-packages/pip (python 3.11)\n",
						ExitCode: 0,
					}, nil
				}
				if command == "pip3" && args[0] == "list" {
					return &output.ExecutionResult{
						Stdout:   `[{"name": "pip", "version": "23.0.1"}]`,
						ExitCode: 0,
					}, nil
				}
				if command == "pip3" && args[0] == "check" {
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
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				return &output.ExecutionResult{ExitCode: 1}, nil
			},
			wantInstalled: false,
			wantErr:       false,
		},
		{
			name:      "npm installed",
			managerID: manager.ManagerNPM,
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == "which" && args[0] == "npm" {
					return &output.ExecutionResult{
						Stdout:   "/usr/bin/npm\n",
						ExitCode: 0,
					}, nil
				}
				if command == "npm" && args[0] == "--version" {
					return &output.ExecutionResult{
						Stdout:   "10.0.0\n",
						ExitCode: 0,
					}, nil
				}
				if command == "npm" && args[0] == "config" {
					return &output.ExecutionResult{
						Stdout:   "/usr/local\n",
						ExitCode: 0,
					}, nil
				}
				if command == "npm" && args[0] == "list" {
					return &output.ExecutionResult{
						Stdout:   `{"dependencies": {}}`,
						ExitCode: 0,
					}, nil
				}
				if command == "npm" && args[0] == "outdated" {
					return &output.ExecutionResult{
						Stdout:   "{}",
						ExitCode: 0,
					}, nil
				}
				if command == "npm" && args[0] == "doctor" {
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
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
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
