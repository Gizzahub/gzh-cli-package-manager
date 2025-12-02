package pip

import (
	"context"
	"errors"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
)

// Test-specific constants
const (
	pipConfigPath = "~/.pip/pip.conf"
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
		ExitCode: 0,
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

func TestAdapter_Detect(t *testing.T) {
	tests := []struct {
		name     string
		execFunc func(ctx context.Context, command string, args ...string) (*output.ExecutionResult, error)
		want     bool
		wantErr  bool
	}{
		{
			name: "pip3 installed",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == whichCommand && len(args) == 1 && args[0] == pip3Command {
					return &output.ExecutionResult{
						Stdout:   "/usr/bin/pip3\n",
						ExitCode: 0,
					}, nil
				}
				return &output.ExecutionResult{ExitCode: 1}, nil
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "pip installed (fallback)",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == whichCommand && len(args) == 1 {
					if args[0] == pip3Command {
						return &output.ExecutionResult{ExitCode: 1}, nil
					}
					if args[0] == "pip" {
						return &output.ExecutionResult{
							Stdout:   "/usr/bin/pip\n",
							ExitCode: 0,
						}, nil
					}
				}
				return &output.ExecutionResult{ExitCode: 1}, nil
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "pip not installed",
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return &output.ExecutionResult{ExitCode: 1}, nil
			},
			want:    false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter(&mockExecutor{execFunc: tt.execFunc}, &mockLogger{})
			got, err := adapter.Detect(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("Detect() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Detect() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAdapter_GetVersion(t *testing.T) {
	tests := []struct {
		name     string
		execFunc func(ctx context.Context, command string, args ...string) (*output.ExecutionResult, error)
		want     string
		wantErr  bool
	}{
		{
			name: "valid version output",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == whichCommand && args[0] == pip3Command {
					return &output.ExecutionResult{Stdout: "/usr/bin/pip3", ExitCode: 0}, nil
				}
				if command == pip3Command && args[0] == "--version" {
					return &output.ExecutionResult{
						Stdout:   "pip 23.0.1 from /usr/lib/python3.11/site-packages/pip (python 3.11)\n",
						ExitCode: 0,
					}, nil
				}
				return &output.ExecutionResult{ExitCode: 1}, nil
			},
			want:    "23.0.1",
			wantErr: false,
		},
		{
			name: "pip version (not pip3)",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == whichCommand && args[0] == pip3Command {
					return &output.ExecutionResult{ExitCode: 1}, nil
				}
				if command == "pip" && args[0] == "--version" {
					return &output.ExecutionResult{
						Stdout:   "pip 22.3.1 from /usr/lib/python3.10/site-packages/pip (python 3.10)\n",
						ExitCode: 0,
					}, nil
				}
				return &output.ExecutionResult{ExitCode: 0}, nil
			},
			want:    "22.3.1",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter(&mockExecutor{execFunc: tt.execFunc}, &mockLogger{})
			got, err := adapter.GetVersion(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("GetVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAdapter_GetBinaryPath(t *testing.T) {
	tests := []struct {
		name     string
		execFunc func(ctx context.Context, command string, args ...string) (*output.ExecutionResult, error)
		want     string
		wantErr  bool
	}{
		{
			name: "pip3 binary found",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == whichCommand {
					if args[0] == pip3Command {
						return &output.ExecutionResult{
							Stdout:   "/usr/bin/pip3\n",
							ExitCode: 0,
						}, nil
					}
				}
				return &output.ExecutionResult{ExitCode: 1}, nil
			},
			want:    "/usr/bin/pip3",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter(&mockExecutor{execFunc: tt.execFunc}, &mockLogger{})
			got, err := adapter.GetBinaryPath(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("GetBinaryPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetBinaryPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAdapter_GetConfigPath(t *testing.T) {
	adapter := NewAdapter(&mockExecutor{}, &mockLogger{})
	got, err := adapter.GetConfigPath(context.Background())

	if err != nil {
		t.Errorf("GetConfigPath() error = %v", err)
	}
	if got != pipConfigPath {
		t.Errorf("GetConfigPath() = %v, want ~/.pip/pip.conf", got)
	}
}

func TestAdapter_ListPackages(t *testing.T) {
	tests := []struct {
		name         string
		execFunc     func(ctx context.Context, command string, args ...string) (*output.ExecutionResult, error)
		wantCount    int
		wantUpgrades int
		wantErr      bool
	}{
		{
			name: "packages with updates",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == whichCommand && args[0] == pip3Command {
					return &output.ExecutionResult{Stdout: "/usr/bin/pip3", ExitCode: 0}, nil
				}
				// pip3 list --format=json
				if command == pip3Command && len(args) == 2 && args[0] == listArg && args[1] == "--format=json" {
					return &output.ExecutionResult{
						Stdout: `[
							{"name": "pip", "version": "23.0.1"},
							{"name": "requests", "version": "2.28.0"},
							{"name": "numpy", "version": "1.24.0"}
						]`,
						ExitCode: 0,
					}, nil
				}
				// pip3 list --outdated --format=json
				if command == pip3Command && len(args) == 3 && args[0] == listArg && args[1] == "--outdated" {
					return &output.ExecutionResult{
						Stdout: `[
							{"name": "requests", "version": "2.28.0", "latest_version": "2.31.0", "latest_filetype": "wheel"},
							{"name": "numpy", "version": "1.24.0", "latest_version": "1.26.4", "latest_filetype": "wheel"}
						]`,
						ExitCode: 0,
					}, nil
				}
				return &output.ExecutionResult{ExitCode: 1}, nil
			},
			wantCount:    3,
			wantUpgrades: 2,
			wantErr:      false,
		},
		{
			name: "no updates available",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == whichCommand && args[0] == pip3Command {
					return &output.ExecutionResult{Stdout: "/usr/bin/pip3", ExitCode: 0}, nil
				}
				if command == pip3Command && args[0] == listArg && args[1] == "--format=json" {
					return &output.ExecutionResult{
						Stdout:   `[{"name": "pip", "version": "23.0.1"}]`,
						ExitCode: 0,
					}, nil
				}
				if command == pip3Command && args[0] == listArg && args[1] == "--outdated" {
					return &output.ExecutionResult{
						Stdout:   "[]",
						ExitCode: 0,
					}, nil
				}
				return &output.ExecutionResult{ExitCode: 1}, nil
			},
			wantCount:    1,
			wantUpgrades: 0,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter(&mockExecutor{execFunc: tt.execFunc}, &mockLogger{})
			packages, err := adapter.ListPackages(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("ListPackages() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(packages) != tt.wantCount {
				t.Errorf("ListPackages() package count = %d, want %d", len(packages), tt.wantCount)
			}

			// Count upgradable packages
			upgradeCount := 0
			for _, pkg := range packages {
				if pkg.UpdateType != manager.UpdateNone {
					upgradeCount++
				}
			}

			if upgradeCount != tt.wantUpgrades {
				t.Errorf("ListPackages() upgrade count = %d, want %d", upgradeCount, tt.wantUpgrades)
			}

			// Verify package properties
			if len(packages) > 0 {
				pkg := packages[0]
				if pkg.Name == "" {
					t.Error("Package name is empty")
				}
				if pkg.CurrentVersion == "" {
					t.Error("Package current version is empty")
				}
				if !pkg.IsGlobal {
					t.Error("Pip packages should be global")
				}
			}
		})
	}
}

func TestAdapter_CheckHealth(t *testing.T) {
	tests := []struct {
		name     string
		execFunc func(ctx context.Context, command string, args ...string) (*output.ExecutionResult, error)
		want     manager.Status
		wantErr  bool
	}{
		{
			name: "healthy system",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == whichCommand && args[0] == pip3Command {
					return &output.ExecutionResult{Stdout: "/usr/bin/pip3", ExitCode: 0}, nil
				}
				if command == pip3Command && len(args) == 1 && args[0] == "check" {
					return &output.ExecutionResult{
						Stdout:   "No broken requirements found.\n",
						ExitCode: 0,
					}, nil
				}
				return &output.ExecutionResult{ExitCode: 0}, nil
			},
			want:    manager.StatusHealthy,
			wantErr: false,
		},
		{
			name: "degraded with dependency issues",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == whichCommand && args[0] == pip3Command {
					return &output.ExecutionResult{Stdout: "/usr/bin/pip3", ExitCode: 0}, nil
				}
				if command == pip3Command && args[0] == "check" {
					return &output.ExecutionResult{
						Stderr:   "package-a 1.0 has requirement package-b>=2.0, but you have package-b 1.5.\n",
						ExitCode: 1,
					}, nil
				}
				return &output.ExecutionResult{ExitCode: 0}, nil
			},
			want:    manager.StatusDegraded,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter(&mockExecutor{execFunc: tt.execFunc}, &mockLogger{})
			got, err := adapter.CheckHealth(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("CheckHealth() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CheckHealth() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAdapter_GetVersion_Error(t *testing.T) {
	adapter := NewAdapter(&mockExecutor{
		execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
			return nil, errors.New("execution failed")
		},
	}, &mockLogger{})

	_, err := adapter.GetVersion(context.Background())
	if err == nil {
		t.Error("Expected error for executor failure")
	}
}

func TestAdapter_GetVersion_ParseError(t *testing.T) {
	adapter := NewAdapter(&mockExecutor{
		execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
			if command == whichCommand {
				return &output.ExecutionResult{Stdout: "/usr/bin/pip3", ExitCode: 0}, nil
			}
			// Return output with less than 2 fields (only one word)
			return &output.ExecutionResult{
				Stdout:   "pip",
				ExitCode: 0,
			}, nil
		},
	}, &mockLogger{})

	_, err := adapter.GetVersion(context.Background())
	if err == nil {
		t.Error("Expected error for parse failure")
	}
}

func TestAdapter_GetBinaryPath_Error(t *testing.T) {
	adapter := NewAdapter(&mockExecutor{
		execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
			return nil, errors.New("execution failed")
		},
	}, &mockLogger{})

	_, err := adapter.GetBinaryPath(context.Background())
	if err == nil {
		t.Error("Expected error for executor failure")
	}
}

func TestAdapter_ListPackages_Error(t *testing.T) {
	adapter := NewAdapter(&mockExecutor{
		execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
			if command == whichCommand {
				return &output.ExecutionResult{Stdout: "/usr/bin/pip3", ExitCode: 0}, nil
			}
			return nil, errors.New("execution failed")
		},
	}, &mockLogger{})

	_, err := adapter.ListPackages(context.Background())
	if err == nil {
		t.Error("Expected error for executor failure")
	}
}

func TestAdapter_ListPackages_InvalidJSON(t *testing.T) {
	adapter := NewAdapter(&mockExecutor{
		execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
			if command == whichCommand {
				return &output.ExecutionResult{Stdout: "/usr/bin/pip3", ExitCode: 0}, nil
			}
			if command == pip3Command && args[0] == "list" {
				return &output.ExecutionResult{
					Stdout:   "not valid json",
					ExitCode: 0,
				}, nil
			}
			return &output.ExecutionResult{ExitCode: 0}, nil
		},
	}, &mockLogger{})

	_, err := adapter.ListPackages(context.Background())
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestAdapter_CheckHealth_ExecutorError(t *testing.T) {
	adapter := NewAdapter(&mockExecutor{
		execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
			if command == whichCommand {
				return &output.ExecutionResult{Stdout: "/usr/bin/pip3", ExitCode: 0}, nil
			}
			return nil, errors.New("execution failed")
		},
	}, &mockLogger{})

	status, err := adapter.CheckHealth(context.Background())

	if err != nil {
		t.Errorf("CheckHealth() should not return error, got %v", err)
	}

	if status != manager.StatusDegraded {
		t.Errorf("CheckHealth() = %v, want StatusDegraded", status)
	}
}

func TestAdapter_Update(t *testing.T) {
	adapter := NewAdapter(&mockExecutor{
		execFunc: func(_ context.Context, command string, _ ...string) (*output.ExecutionResult, error) {
			if command == whichCommand {
				return &output.ExecutionResult{Stdout: "/usr/bin/pip3", ExitCode: 0}, nil
			}
			return &output.ExecutionResult{ExitCode: 0}, nil
		},
	}, &mockLogger{})

	result, err := adapter.Update(context.Background(), adapterm.UpdateOptions{})

	if err == nil {
		t.Error("Expected error from Update (not implemented)")
	}

	if result.Success {
		t.Error("Expected Success to be false")
	}
}

func TestAdapter_Detect_ExecutorError(t *testing.T) {
	adapter := NewAdapter(&mockExecutor{
		execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
			return nil, errors.New("execution failed")
		},
	}, &mockLogger{})

	detected, err := adapter.Detect(context.Background())

	if err != nil {
		t.Errorf("Detect() should not return error, got %v", err)
	}

	if detected {
		t.Error("Detect() should return false when executor fails")
	}
}
