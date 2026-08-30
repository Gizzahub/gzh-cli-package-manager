package homebrew

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
	brewCommand     = "brew"
	brewVersionFlag = "--version"
)

func TestAdapter_GetConfigPath(t *testing.T) {
	execFunc := func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		if command == brewCommand && args[0] == "--prefix" {
			return &output.ExecutionResult{
				Stdout:   "/usr/local\n",
				ExitCode: 0,
			}, nil
		}
		return &output.ExecutionResult{ExitCode: 1}, nil
	}

	adapter := NewAdapter(testutil.NewMockExecutor(execFunc), testutil.NewMockLogger())
	path, err := adapter.GetConfigPath(context.Background())
	if err != nil {
		t.Errorf("GetConfigPath() unexpected error: %v", err)
	}

	// Should return Homebrew prefix path
	if path == "" {
		t.Error("GetConfigPath() returned empty path")
	}
}

func TestAdapter_Detect(t *testing.T) {
	tests := []struct {
		name     string
		execFunc func(ctx context.Context, command string, args ...string) (*output.ExecutionResult, error)
		want     bool
		wantErr  bool
	}{
		{
			name: "brew installed",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == "which" && len(args) == 1 && args[0] == brewCommand {
					return &output.ExecutionResult{
						ExitCode: 0,
						Stdout:   "/usr/local/bin/brew\n",
					}, nil
				}
				return nil, errors.New("unexpected command")
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "brew not installed",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == "which" && len(args) == 1 && args[0] == brewCommand {
					return &output.ExecutionResult{
						ExitCode: 1,
						Stderr:   "brew not found",
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
				if command == brewCommand && len(args) == 1 && args[0] == brewVersionFlag {
					return &output.ExecutionResult{
						ExitCode: 0,
						Stdout:   "Homebrew 4.2.1\nHomebrew/homebrew-core (git revision abc123; last commit 2024-01-15)\n",
					}, nil
				}
				return nil, errors.New("unexpected command")
			},
			want:    "4.2.1",
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

func TestAdapter_ListPackages(t *testing.T) {
	tests := []struct {
		name     string
		execFunc func(ctx context.Context, command string, args ...string) (*output.ExecutionResult, error)
		wantLen  int
		wantErr  bool
	}{
		{
			name: "valid packages output",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == brewCommand && len(args) == 3 {
					return &output.ExecutionResult{
						ExitCode: 0,
						Stdout: `{
							"formulae": [
								{
									"name": "git",
									"full_name": "git",
									"desc": "Distributed revision control system",
									"version": "2.43.0",
									"installed_on_demand": false,
									"versions": {
										"stable": "2.43.0"
									}
								}
							],
							"casks": [
								{
									"token": "visual-studio-code",
									"name": ["Visual Studio Code"],
									"desc": "Code editor",
									"version": "1.85.0"
								}
							]
						}`,
					}, nil
				}
				return nil, errors.New("unexpected command")
			},
			wantLen: 2, // 1 formula + 1 cask
			wantErr: false,
		},
		{
			name: "empty packages",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == brewCommand && len(args) == 3 {
					return &output.ExecutionResult{
						ExitCode: 0,
						Stdout:   `{"formulae": [], "casks": []}`,
					}, nil
				}
				return nil, errors.New("unexpected command")
			},
			wantLen: 0,
			wantErr: false,
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
				if command == brewCommand && len(args) == 1 && args[0] == "doctor" {
					return &output.ExecutionResult{
						ExitCode: 0,
						Stdout:   "Your system is ready to brew.\n",
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
				if command == brewCommand && len(args) == 1 && args[0] == "doctor" {
					return &output.ExecutionResult{
						ExitCode: 1,
						Stdout:   "Warning: Some kegs are not writable\n",
					}, nil
				}
				return nil, errors.New("unexpected command")
			},
			want:    manager.StatusDegraded,
			wantErr: false,
		},
		{
			name: "command execution error",
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return nil, errors.New("command failed")
			},
			want:    manager.StatusDegraded,
			wantErr: false,
		},
		{
			name: "error status without warning keyword",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == brewCommand && len(args) == 1 && args[0] == "doctor" {
					return &output.ExecutionResult{
						ExitCode: 1,
						Stdout:   "Some critical issue found\n",
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

func TestAdapter_GetBinaryPath(t *testing.T) {
	tests := []struct {
		name     string
		execFunc func(ctx context.Context, command string, args ...string) (*output.ExecutionResult, error)
		want     string
		wantErr  bool
	}{
		{
			name: "brew found",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == "which" && len(args) == 1 && args[0] == brewCommand {
					return &output.ExecutionResult{
						ExitCode: 0,
						Stdout:   "/usr/local/bin/brew\n",
					}, nil
				}
				return nil, errors.New("unexpected command")
			},
			want:    "/usr/local/bin/brew",
			wantErr: false,
		},
		{
			name: "brew not found - exit code 1",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == "which" && len(args) == 1 && args[0] == brewCommand {
					return &output.ExecutionResult{
						ExitCode: 1,
						Stderr:   "brew not found",
					}, nil
				}
				return nil, errors.New("unexpected command")
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "command execution error",
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return nil, errors.New("command failed")
			},
			want:    "",
			wantErr: true,
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

func TestAdapter_GetVersion_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		execFunc func(ctx context.Context, command string, args ...string) (*output.ExecutionResult, error)
		wantErr  bool
	}{
		{
			name: "non-zero exit code",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == brewCommand && len(args) == 1 && args[0] == brewVersionFlag {
					return &output.ExecutionResult{
						ExitCode: 1,
						Stderr:   "brew not configured properly",
					}, nil
				}
				return nil, errors.New("unexpected command")
			},
			wantErr: true,
		},
		{
			name: "empty output",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == brewCommand && len(args) == 1 && args[0] == brewVersionFlag {
					return &output.ExecutionResult{
						ExitCode: 0,
						Stdout:   "",
					}, nil
				}
				return nil, errors.New("unexpected command")
			},
			wantErr: true,
		},
		{
			name: "unexpected format - single word",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == brewCommand && len(args) == 1 && args[0] == brewVersionFlag {
					return &output.ExecutionResult{
						ExitCode: 0,
						Stdout:   "Homebrew\n",
					}, nil
				}
				return nil, errors.New("unexpected command")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter(testutil.NewMockExecutor(tt.execFunc), testutil.NewMockLogger())

			_, err := adapter.GetVersion(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("GetVersion() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAdapter_GetConfigPath_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		execFunc func(ctx context.Context, command string, args ...string) (*output.ExecutionResult, error)
		wantErr  bool
	}{
		{
			name: "command execution error",
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return nil, errors.New("command failed")
			},
			wantErr: true,
		},
		{
			name: "non-zero exit code",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == brewCommand && args[0] == "--prefix" {
					return &output.ExecutionResult{
						ExitCode: 1,
						Stderr:   "brew prefix failed",
					}, nil
				}
				return nil, errors.New("unexpected command")
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

func TestAdapter_ListPackages_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		execFunc func(ctx context.Context, command string, args ...string) (*output.ExecutionResult, error)
		wantErr  bool
	}{
		{
			name: "command execution error",
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return nil, errors.New("command failed")
			},
			wantErr: true,
		},
		{
			name: "non-zero exit code",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == brewCommand && len(args) == 3 {
					return &output.ExecutionResult{
						ExitCode: 1,
						Stderr:   "brew info failed",
					}, nil
				}
				return nil, errors.New("unexpected command")
			},
			wantErr: true,
		},
		{
			name: "invalid JSON output",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == brewCommand && len(args) == 3 {
					return &output.ExecutionResult{
						ExitCode: 0,
						Stdout:   "not valid json",
					}, nil
				}
				return nil, errors.New("unexpected command")
			},
			wantErr: true,
		},
		{
			name: "package with update available",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == brewCommand && len(args) == 3 {
					return &output.ExecutionResult{
						ExitCode: 0,
						Stdout: `{
							"formulae": [
								{
									"name": "git",
									"version": "2.42.0",
									"versions": {
										"stable": "2.43.0"
									}
								}
							],
							"casks": []
						}`,
					}, nil
				}
				return nil, errors.New("unexpected command")
			},
			wantErr: false,
		},
		{
			name: "cask without name array",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == brewCommand && len(args) == 3 {
					return &output.ExecutionResult{
						ExitCode: 0,
						Stdout: `{
							"formulae": [],
							"casks": [
								{
									"token": "test-app",
									"name": [],
									"desc": "Test application",
									"version": "1.0.0"
								}
							]
						}`,
					}, nil
				}
				return nil, errors.New("unexpected command")
			},
			wantErr: false,
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

func TestAdapter_Update(t *testing.T) {
	tests := []struct {
		name        string
		dryRun      bool
		strategy    string
		execFunc    func(ctx context.Context, command string, args ...string) (*output.ExecutionResult, error)
		wantSuccess bool
		wantErr     bool
	}{
		{
			name:   "dry run mode",
			dryRun: true,
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return &output.ExecutionResult{ExitCode: 0}, nil
			},
			wantSuccess: true,
			wantErr:     false,
		},
		{
			name:     "update and upgrade success",
			dryRun:   false,
			strategy: "stable",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == brewCommand {
					if len(args) == 1 && args[0] == "update" {
						return &output.ExecutionResult{
							ExitCode: 0,
							Stdout:   "Updated Homebrew\n",
						}, nil
					}
					if len(args) == 1 && args[0] == "upgrade" {
						return &output.ExecutionResult{
							ExitCode: 0,
							Stdout:   "Upgrading git\n==> Upgrading 1 outdated package:\ngit 2.42.0 -> 2.43.0\n",
						}, nil
					}
				}
				return &output.ExecutionResult{ExitCode: 0}, nil
			},
			wantSuccess: true,
			wantErr:     false,
		},
		{
			name:     "update fails",
			dryRun:   false,
			strategy: "stable",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == brewCommand && len(args) == 1 && args[0] == "update" {
					return nil, errors.New("network error")
				}
				return &output.ExecutionResult{ExitCode: 0}, nil
			},
			wantSuccess: false,
			wantErr:     true,
		},
		{
			name:     "update non-zero exit code",
			dryRun:   false,
			strategy: "stable",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == brewCommand && len(args) == 1 && args[0] == "update" {
					return &output.ExecutionResult{
						ExitCode: 1,
						Stderr:   "update failed",
					}, nil
				}
				return &output.ExecutionResult{ExitCode: 0}, nil
			},
			wantSuccess: false,
			wantErr:     true,
		},
		{
			name:     "fixed strategy skips upgrade",
			dryRun:   false,
			strategy: "fixed",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == brewCommand && len(args) == 1 && args[0] == "update" {
					return &output.ExecutionResult{
						ExitCode: 0,
						Stdout:   "Updated Homebrew\n",
					}, nil
				}
				return &output.ExecutionResult{ExitCode: 0}, nil
			},
			wantSuccess: true,
			wantErr:     false,
		},
		{
			name:     "upgrade fails with error",
			dryRun:   false,
			strategy: "stable",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == brewCommand {
					if len(args) == 1 && args[0] == "update" {
						return &output.ExecutionResult{ExitCode: 0}, nil
					}
					if len(args) == 1 && args[0] == "upgrade" {
						return nil, errors.New("upgrade error")
					}
				}
				return &output.ExecutionResult{ExitCode: 0}, nil
			},
			wantSuccess: true, // Update succeeded, upgrade failed but not fatal.
			wantErr:     false,
		},
		{
			name:     "upgrade non-zero exit code",
			dryRun:   false,
			strategy: "stable",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == brewCommand {
					if len(args) == 1 && args[0] == "update" {
						return &output.ExecutionResult{ExitCode: 0}, nil
					}
					if len(args) == 1 && args[0] == "upgrade" {
						return &output.ExecutionResult{
							ExitCode: 1,
							Stderr:   "some packages failed",
						}, nil
					}
				}
				return &output.ExecutionResult{ExitCode: 0}, nil
			},
			wantSuccess: true,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter(testutil.NewMockExecutor(tt.execFunc), testutil.NewMockLogger())

			opts := adapterm.UpdateOptions{
				DryRun:   tt.dryRun,
				Strategy: adapterm.UpdateStrategy(tt.strategy),
			}

			result, err := adapter.Update(context.Background(), opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result != nil && result.Success != tt.wantSuccess {
				t.Errorf("Update() success = %v, want %v", result.Success, tt.wantSuccess)
			}
		})
	}
}

func TestAdapter_Detect_EmptyOutput(t *testing.T) {
	execFunc := func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		if command == "which" && len(args) == 1 && args[0] == brewCommand {
			return &output.ExecutionResult{
				ExitCode: 0,
				Stdout:   "",
			}, nil
		}
		return nil, errors.New("unexpected command")
	}

	adapter := NewAdapter(testutil.NewMockExecutor(execFunc), testutil.NewMockLogger())
	got, err := adapter.Detect(context.Background())
	if err != nil {
		t.Errorf("Detect() unexpected error: %v", err)
	}
	if got {
		t.Error("Detect() = true, want false for empty output")
	}
}
