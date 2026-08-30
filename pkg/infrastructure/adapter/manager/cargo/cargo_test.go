package cargo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/testutil"
)

// Test-specific constants.
const (
	versionFlag                       = "--version"
	testCommandExecutionErrorCaseName = "command execution error"
)

func TestAdapter_Detect(t *testing.T) {
	tests := []struct {
		name     string
		execFunc testutil.ExecutorFunc
		want     bool
		wantErr  bool
	}{
		{
			name: "cargo installed",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == whichCommand && len(args) == 1 && args[0] == cargoCommand {
					return testutil.SuccessResult("/usr/bin/cargo\n"), nil
				}
				return testutil.FailureResult(1, ""), nil
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "cargo not installed",
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return testutil.FailureResult(1, ""), nil
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "executor error reports cargo unavailable",
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return nil, errors.New("command failed")
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
		execFunc testutil.ExecutorFunc
		want     string
		wantErr  bool
	}{
		{
			name: "valid version output",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == cargoCommand && len(args) == 1 && args[0] == versionFlag {
					return testutil.SuccessResult("cargo 1.75.0 (1d8b05cdd 2023-11-20)\n"), nil
				}
				return testutil.FailureResult(1, ""), nil
			},
			want:    "1.75.0",
			wantErr: false,
		},
		{
			name: "older cargo version",
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return testutil.SuccessResult("cargo 1.70.0 (38c6b0c90 2023-08-01)\n"), nil
			},
			want:    "1.70.0",
			wantErr: false,
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
		execFunc testutil.ExecutorFunc
		want     string
		wantErr  bool
	}{
		{
			name: "binary found",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == whichCommand && args[0] == cargoCommand {
					return testutil.SuccessResult("/home/user/.cargo/bin/cargo\n"), nil
				}
				return testutil.FailureResult(1, ""), nil
			},
			want:    "/home/user/.cargo/bin/cargo",
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

func TestAdapter_ListPackages(t *testing.T) {
	tests := []struct {
		name      string
		execFunc  testutil.ExecutorFunc
		wantCount int
		wantErr   bool
	}{
		{
			name: "multiple installed packages",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == cargoCommand && len(args) == 2 && args[0] == installFlag && args[1] == listFlag {
					return testutil.SuccessResult(`ripgrep v14.0.3:
    rg
cargo-watch v8.4.1:
    cargo-watch
bat v0.24.0:
    bat
`), nil
				}
				return testutil.FailureResult(1, ""), nil
			},
			wantCount: 3,
			wantErr:   false,
		},
		{
			name: "package with local path",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == cargoCommand && args[0] == installFlag && args[1] == listFlag {
					return testutil.SuccessResult(`ripgrep v14.0.3:
    rg
my-tool v0.1.0 (/home/user/projects/my-tool):
    my-tool
`), nil
				}
				return testutil.FailureResult(1, ""), nil
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name: "no packages installed",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == cargoCommand && args[0] == installFlag && args[1] == listFlag {
					return testutil.SuccessResult(""), nil
				}
				return testutil.FailureResult(1, ""), nil
			},
			wantCount: 0,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter(testutil.NewMockExecutor(tt.execFunc), testutil.NewMockLogger())
			packages, err := adapter.ListPackages(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("ListPackages() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(packages) != tt.wantCount {
				t.Errorf("ListPackages() package count = %d, want %d", len(packages), tt.wantCount)
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
			}

			// Verify local package detection if present
			for _, pkg := range packages {
				if pkg.Name == "my-tool" {
					if pkg.IsGlobal {
						t.Error("Local path package should not be marked as global")
					}
				}
			}
		})
	}
}

func TestAdapter_CheckHealth(t *testing.T) {
	tests := []struct {
		name     string
		execFunc testutil.ExecutorFunc
		want     manager.Status
		wantErr  bool
	}{
		{
			name: "healthy system",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == cargoCommand && len(args) == 1 && args[0] == versionFlag {
					return testutil.SuccessResult("cargo 1.75.0 (1d8b05cdd 2023-11-20)\n"), nil
				}
				return testutil.SuccessResult(""), nil
			},
			want:    manager.StatusHealthy,
			wantErr: false,
		},
		{
			name: "degraded system",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == cargoCommand && args[0] == versionFlag {
					return testutil.FailureResult(1, "error: failed to run cargo\n"), nil
				}
				return testutil.SuccessResult(""), nil
			},
			want:    manager.StatusDegraded,
			wantErr: false,
		},
		{
			name: "executor error reports degraded status",
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return nil, errors.New("command failed")
			},
			want:    manager.StatusDegraded,
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

func TestAdapter_GetConfigPath(t *testing.T) {
	homeErr := errors.New("home directory unavailable")

	t.Run("resolved home directory", func(t *testing.T) {
		homeDir := t.TempDir()
		cargoDir := filepath.Join(homeDir, ".cargo")
		if err := os.Mkdir(cargoDir, 0o755); err != nil {
			t.Fatalf("Mkdir(%q): %v", cargoDir, err)
		}
		configTOML := filepath.Join(cargoDir, "config.toml")
		if _, err := os.Stat(configTOML); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("config.toml must be absent: stat error = %v", err)
		}

		adapter := NewAdapter(testutil.NewMockExecutor(nil), testutil.NewMockLogger())
		adapter.homeDirResolver = &homeDirResolver{resolve: func() (string, error) { return homeDir, nil }}
		path, err := adapter.GetConfigPath(context.Background())

		if err != nil {
			t.Fatalf("GetConfigPath() error = %v", err)
		}
		if want := filepath.Join(cargoDir, "config"); path != want {
			t.Errorf("GetConfigPath() = %q, want %q", path, want)
		}
	})

	t.Run("home directory error", func(t *testing.T) {
		adapter := NewAdapter(testutil.NewMockExecutor(nil), testutil.NewMockLogger())
		adapter.homeDirResolver = &homeDirResolver{resolve: func() (string, error) {
			return "", homeErr
		}}
		path, err := adapter.GetConfigPath(context.Background())

		if !errors.Is(err, homeErr) {
			t.Errorf("GetConfigPath() error = %v, want cause %v", err, homeErr)
		}
		if path != "" {
			t.Errorf("GetConfigPath() = %q, want empty path", path)
		}
	})
}

func TestAdapter_GetConfigPath_ZeroValue(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("UserHomeDir unavailable: %v", err)
	}

	got, err := (&Adapter{}).GetConfigPath(context.Background())
	if err != nil {
		t.Fatalf("GetConfigPath() error = %v", err)
	}

	configTOML := filepath.Join(homeDir, ".cargo", "config.toml")
	want := filepath.Join(homeDir, ".cargo", "config")
	if _, err := os.Stat(configTOML); err == nil {
		want = configTOML
	}
	if got != want {
		t.Errorf("GetConfigPath() = %q, want %q", got, want)
	}
}

func TestAdapter_GetVersion_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		execFunc testutil.ExecutorFunc
		wantErr  bool
	}{
		{
			name: testCommandExecutionErrorCaseName,
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return nil, errors.New("command failed")
			},
			wantErr: true,
		},
		{
			name: "non-zero exit code",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == cargoCommand && args[0] == versionFlag {
					return testutil.FailureResult(1, "cargo not found"), nil
				}
				return testutil.SuccessResult(""), nil
			},
			wantErr: true,
		},
		{
			name: "empty output",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == cargoCommand && args[0] == versionFlag {
					return testutil.SuccessResult(""), nil
				}
				return testutil.SuccessResult(""), nil
			},
			wantErr: true,
		},
		{
			name: "unexpected format",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == cargoCommand && args[0] == versionFlag {
					return testutil.SuccessResult("cargo\n"), nil
				}
				return testutil.SuccessResult(""), nil
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

func TestAdapter_GetBinaryPath_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		execFunc testutil.ExecutorFunc
		wantErr  bool
	}{
		{
			name: testCommandExecutionErrorCaseName,
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return nil, errors.New("command failed")
			},
			wantErr: true,
		},
		{
			name: "binary not found - returns empty path",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == whichCommand && args[0] == cargoCommand {
					return &output.ExecutionResult{
						ExitCode: 1,
						Stderr:   "cargo not found",
						Stdout:   "",
					}, nil
				}
				return testutil.SuccessResult(""), nil
			},
			wantErr: false, // Current implementation doesn't check exit code
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

func TestAdapter_ListPackages_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		execFunc testutil.ExecutorFunc
		wantErr  bool
	}{
		{
			name: testCommandExecutionErrorCaseName,
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return nil, errors.New("command failed")
			},
			wantErr: true,
		},
		{
			name: "non-zero exit code - returns empty list",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == cargoCommand && args[0] == installFlag && args[1] == listFlag {
					return &output.ExecutionResult{
						ExitCode: 1,
						Stderr:   "cargo install --list failed",
						Stdout:   "",
					}, nil
				}
				return testutil.SuccessResult(""), nil
			},
			wantErr: false, // Current implementation doesn't check exit code
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
		execFunc    testutil.ExecutorFunc
		wantSuccess bool
		wantErr     bool
	}{
		{
			name:   "update success",
			dryRun: false,
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return testutil.SuccessResult(""), nil
			},
			wantSuccess: true,
			wantErr:     false,
		},
		{
			name:   "dry run success",
			dryRun: true,
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return testutil.SuccessResult(""), nil
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
				Strategy: adapterm.StrategyStable,
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
