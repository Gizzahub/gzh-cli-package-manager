package npm

import (
	"context"
	"errors"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/testutil"
)

const (
	npmCommand         = "npm"
	whichCommand       = "which"
	doctorCommand      = "doctor"
	testNPMListCommand = "list"
)

func TestAdapter_Detect(t *testing.T) {
	tests := []struct {
		name     string
		execFunc func(ctx context.Context, command string, args ...string) (*output.ExecutionResult, error)
		want     bool
		wantErr  bool
	}{
		{
			name: "npm installed",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == whichCommand && len(args) == 1 && args[0] == npmCommand {
					return &output.ExecutionResult{
						ExitCode: 0,
						Stdout:   "/usr/bin/npm\n",
					}, nil
				}
				return nil, errors.New("unexpected command")
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "npm not installed",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == whichCommand && len(args) == 1 && args[0] == npmCommand {
					return &output.ExecutionResult{
						ExitCode: 1,
						Stderr:   "npm not found",
					}, errors.New("exit code 1")
				}
				return nil, errors.New("unexpected command")
			},
			want:    false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter(testutil.NewMockExecutor(tt.execFunc), testutil.NewMockLogger())

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
				if command == npmCommand && len(args) == 1 && args[0] == "--version" {
					return &output.ExecutionResult{
						ExitCode: 0,
						Stdout:   "10.5.0\n",
					}, nil
				}
				return nil, errors.New("unexpected command")
			},
			want:    "10.5.0",
			wantErr: false,
		},
		{
			name: "command execution error",
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return nil, errors.New("command not found")
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter(testutil.NewMockExecutor(tt.execFunc), testutil.NewMockLogger())

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
			name: "binary found",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == whichCommand && len(args) == 1 && args[0] == npmCommand {
					return &output.ExecutionResult{
						ExitCode: 0,
						Stdout:   "/usr/bin/npm\n",
					}, nil
				}
				return nil, errors.New("unexpected command")
			},
			want:    "/usr/bin/npm",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter(testutil.NewMockExecutor(tt.execFunc), testutil.NewMockLogger())

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
	tests := []struct {
		name     string
		execFunc func(ctx context.Context, command string, args ...string) (*output.ExecutionResult, error)
		want     string
		wantErr  bool
	}{
		{
			name: "config path found",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == npmCommand && len(args) == 3 && args[0] == "config" {
					return &output.ExecutionResult{
						ExitCode: 0,
						Stdout:   "/usr/local\n",
					}, nil
				}
				return nil, errors.New("unexpected command")
			},
			want:    "/usr/local",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter(testutil.NewMockExecutor(tt.execFunc), testutil.NewMockLogger())

			got, err := adapter.GetConfigPath(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("GetConfigPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetConfigPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAdapter_ListPackages(t *testing.T) {
	tests := []struct {
		name        string
		execFunc    func(ctx context.Context, command string, args ...string) (*output.ExecutionResult, error)
		wantLen     int
		wantUpdates int
		wantErr     bool
	}{
		{
			name: "packages with updates",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == npmCommand && len(args) >= 2 && args[0] == testNPMListCommand {
					return &output.ExecutionResult{
						ExitCode: 0,
						Stdout: `{
							"dependencies": {
								"react": {"version": "18.0.0"},
								"vue": {"version": "3.0.0"},
								"typescript": {"version": "5.0.0"}
							}
						}`,
					}, nil
				}
				if command == npmCommand && len(args) >= 2 && args[0] == "outdated" {
					return &output.ExecutionResult{
						ExitCode: 1, // npm outdated returns 1 when packages are outdated
						Stdout: `{
							"react": {
								"current": "18.0.0",
								"wanted": "18.2.0",
								"latest": "18.3.0"
							},
							"vue": {
								"current": "3.0.0",
								"wanted": "3.4.0",
								"latest": "3.4.0"
							}
						}`,
					}, nil
				}
				return nil, errors.New("unexpected command")
			},
			wantLen:     3, // 3 packages total
			wantUpdates: 2, // 2 with updates
			wantErr:     false,
		},
		{
			name: "packages without updates",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == npmCommand && len(args) >= 2 && args[0] == testNPMListCommand {
					return &output.ExecutionResult{
						ExitCode: 0,
						Stdout: `{
							"dependencies": {
								"typescript": {"version": "5.0.0"}
							}
						}`,
					}, nil
				}
				if command == npmCommand && len(args) >= 2 && args[0] == "outdated" {
					return &output.ExecutionResult{
						ExitCode: 0, // No updates
						Stdout:   "{}",
					}, nil
				}
				return nil, errors.New("unexpected command")
			},
			wantLen:     1,
			wantUpdates: 0,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter(testutil.NewMockExecutor(tt.execFunc), testutil.NewMockLogger())

			got, err := adapter.ListPackages(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("ListPackages() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != tt.wantLen {
				t.Errorf("ListPackages() returned %d packages, want %d", len(got), tt.wantLen)
			}

			// Count packages with updates
			updatesCount := 0
			for _, pkg := range got {
				if pkg.IsUpdateAvailable() {
					updatesCount++
				}
			}
			if updatesCount != tt.wantUpdates {
				t.Errorf("ListPackages() has %d packages with updates, want %d", updatesCount, tt.wantUpdates)
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
				if command == npmCommand && len(args) == 1 && args[0] == doctorCommand {
					return &output.ExecutionResult{
						ExitCode: 0,
						Stdout:   "Everything is ok\n",
					}, nil
				}
				return nil, errors.New("unexpected command")
			},
			want:    manager.StatusHealthy,
			wantErr: false,
		},
		{
			name: "degraded with warnings",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == npmCommand && len(args) == 1 && args[0] == doctorCommand {
					return &output.ExecutionResult{
						ExitCode: 1,
						Stdout:   "Warning: some checks failed\n",
					}, nil
				}
				return nil, errors.New("unexpected command")
			},
			want:    manager.StatusDegraded,
			wantErr: false,
		},
		{
			name: "error state",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == npmCommand && len(args) == 1 && args[0] == doctorCommand {
					return &output.ExecutionResult{
						ExitCode: 1,
						Stderr:   "Error: critical failure\n",
					}, nil
				}
				return nil, errors.New("unexpected command")
			},
			want:    manager.StatusError,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter(testutil.NewMockExecutor(tt.execFunc), testutil.NewMockLogger())

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

func TestAdapter_GetVersion_NonZeroExitCode(t *testing.T) {
	execFunc := func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
		return &output.ExecutionResult{
			ExitCode: 1,
			Stderr:   "npm: command not found",
		}, nil
	}
	adapter := NewAdapter(testutil.NewMockExecutor(execFunc), testutil.NewMockLogger())

	_, err := adapter.GetVersion(context.Background())
	if err == nil {
		t.Error("Expected error for non-zero exit code")
	}
}

func TestAdapter_GetBinaryPath_Error(t *testing.T) {
	tests := []struct {
		name     string
		execFunc func(ctx context.Context, command string, args ...string) (*output.ExecutionResult, error)
		wantErr  bool
	}{
		{
			name: "executor error",
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return nil, errors.New("execution failed")
			},
			wantErr: true,
		},
		{
			name: "non-zero exit code",
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return &output.ExecutionResult{
					ExitCode: 1,
					Stderr:   "npm not found",
				}, nil
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter(testutil.NewMockExecutor(tt.execFunc), testutil.NewMockLogger())
			_, err := adapter.GetBinaryPath(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("GetBinaryPath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAdapter_GetConfigPath_Error(t *testing.T) {
	tests := []struct {
		name     string
		execFunc func(ctx context.Context, command string, args ...string) (*output.ExecutionResult, error)
		wantErr  bool
	}{
		{
			name: "executor error",
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return nil, errors.New("execution failed")
			},
			wantErr: true,
		},
		{
			name: "non-zero exit code",
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return &output.ExecutionResult{
					ExitCode: 1,
					Stderr:   "npm config failed",
				}, nil
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter(testutil.NewMockExecutor(tt.execFunc), testutil.NewMockLogger())
			_, err := adapter.GetConfigPath(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("GetConfigPath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAdapter_ListPackages_Error(t *testing.T) {
	tests := []struct {
		name     string
		execFunc func(ctx context.Context, command string, args ...string) (*output.ExecutionResult, error)
		wantErr  bool
	}{
		{
			name: "list executor error",
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return nil, errors.New("execution failed")
			},
			wantErr: true,
		},
		{
			name: "list non-zero exit code",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == "npm" && len(args) >= 1 && args[0] == testNPMListCommand {
					return &output.ExecutionResult{
						ExitCode: 1,
						Stderr:   "npm list failed",
					}, nil
				}
				return &output.ExecutionResult{ExitCode: 0}, nil
			},
			wantErr: true,
		},
		{
			name: "invalid JSON",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == "npm" && len(args) >= 1 && args[0] == testNPMListCommand {
					return &output.ExecutionResult{
						ExitCode: 0,
						Stdout:   "not valid json",
					}, nil
				}
				return &output.ExecutionResult{ExitCode: 0}, nil
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter(testutil.NewMockExecutor(tt.execFunc), testutil.NewMockLogger())
			_, err := adapter.ListPackages(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("ListPackages() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAdapter_CheckHealth_ExecutorError(t *testing.T) {
	execFunc := func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
		return nil, errors.New("execution failed")
	}
	adapter := NewAdapter(testutil.NewMockExecutor(execFunc), testutil.NewMockLogger())

	status, err := adapter.CheckHealth(context.Background())
	if err != nil {
		t.Errorf("CheckHealth() should not return error, got %v", err)
	}

	if status != manager.StatusDegraded {
		t.Errorf("CheckHealth() = %v, want StatusDegraded", status)
	}
}

func TestAdapter_Update(t *testing.T) {
	adapter := NewAdapter(testutil.NewMockExecutor(func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
		return testutil.SuccessResult(""), nil
	}), testutil.NewMockLogger())
	result, err := adapter.Update(context.Background(), adapterm.UpdateOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Update dry-run unexpected error: %v", err)
	}
	if result == nil || !result.Success {
		t.Fatalf("Update dry-run expected success result, got %#v", result)
	}
}

func TestAdapter_Detect_EmptyOutput(t *testing.T) {
	execFunc := func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
		return &output.ExecutionResult{
			ExitCode: 0,
			Stdout:   "   ", // whitespace only
		}, nil
	}
	adapter := NewAdapter(testutil.NewMockExecutor(execFunc), testutil.NewMockLogger())

	detected, err := adapter.Detect(context.Background())
	if err != nil {
		t.Errorf("Detect() error = %v", err)
	}

	if detected {
		t.Error("Detect() should return false for empty output")
	}
}
