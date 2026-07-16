package apt

import (
	"context"
	"errors"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/testutil"
)

// Test-specific constants
const (
	testCommand = "test"
)

func TestAdapter_Detect(t *testing.T) {
	tests := []struct {
		name     string
		execFunc testutil.ExecutorFunc
		want     bool
		wantErr  bool
	}{
		{
			name: "apt installed",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == "which" && len(args) == 1 && args[0] == aptCommand {
					return testutil.SuccessResult("/usr/bin/apt\n"), nil
				}
				return testutil.FailureResult(1, ""), nil
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "apt not installed",
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return testutil.FailureResult(1, ""), nil
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
				if command == aptCommand && len(args) == 1 && args[0] == "--version" {
					return testutil.SuccessResult("apt 2.4.11 (amd64)\n"), nil
				}
				return testutil.FailureResult(1, ""), nil
			},
			want:    "2.4.11",
			wantErr: false,
		},
		{
			name: "version with extra info",
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return testutil.SuccessResult("apt 2.0.6 (arm64)\nCompiled on Jul 15 2023\n"), nil
			},
			want:    "2.0.6",
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
				if command == "which" && args[0] == aptCommand {
					return testutil.SuccessResult("/usr/bin/apt\n"), nil
				}
				return testutil.FailureResult(1, ""), nil
			},
			want:    "/usr/bin/apt",
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
	adapter := NewAdapter(testutil.NewMockExecutor(nil), testutil.NewMockLogger())
	got, err := adapter.GetConfigPath(context.Background())
	if err != nil {
		t.Errorf("GetConfigPath() error = %v", err)
	}
	if got != "/etc/apt" {
		t.Errorf("GetConfigPath() = %v, want /etc/apt", got)
	}
}

func TestAdapter_ListPackages(t *testing.T) {
	tests := []struct {
		name         string
		execFunc     testutil.ExecutorFunc
		wantCount    int
		wantUpgrades int
		wantErr      bool
	}{
		{
			name: "packages with updates",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				// apt list --installed
				if command == aptCommand && len(args) == 2 && args[0] == listArg && args[1] == "--installed" {
					return testutil.SuccessResult(`Listing...
vim/jammy-updates,now 2:8.2.3995-1ubuntu2.12 amd64 [installed]
curl/jammy-security,now 7.81.0-1ubuntu1.15 amd64 [installed,automatic]
git/jammy-updates,now 1:2.34.1-1ubuntu1.10 amd64 [installed]
`), nil
				}
				// apt list --upgradable
				if command == aptCommand && len(args) == 2 && args[0] == listArg && args[1] == "--upgradable" {
					return testutil.SuccessResult(`Listing...
curl/jammy-security 7.81.0-1ubuntu1.16 amd64 [upgradable from: 7.81.0-1ubuntu1.15]
git/jammy-updates 1:2.34.1-1ubuntu1.11 amd64 [upgradable from: 1:2.34.1-1ubuntu1.10]
`), nil
				}
				return testutil.FailureResult(1, ""), nil
			},
			wantCount:    3,
			wantUpgrades: 2,
			wantErr:      false,
		},
		{
			name: "no updates available",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == aptCommand && args[0] == listArg && args[1] == "--installed" {
					return testutil.SuccessResult(`Listing...
vim/jammy-updates,now 2:8.2.3995-1ubuntu2.12 amd64 [installed]
`), nil
				}
				if command == aptCommand && args[0] == listArg && args[1] == "--upgradable" {
					return testutil.SuccessResult("Listing...\n"), nil
				}
				return testutil.FailureResult(1, ""), nil
			},
			wantCount:    1,
			wantUpgrades: 0,
			wantErr:      false,
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
					t.Error("APT packages should be global")
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
				if command == aptGetCommand && len(args) == 1 && args[0] == checkCommand {
					return testutil.SuccessResult("Reading package lists... Done\nBuilding dependency tree... Done\n"), nil
				}
				if command == testCommand {
					return testutil.FailureResult(1, ""), nil // Lock file doesn't exist
				}
				return testutil.SuccessResult(""), nil
			},
			want:    manager.StatusHealthy,
			wantErr: false,
		},
		{
			name: "degraded with lock file",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == aptGetCommand && args[0] == checkCommand {
					return testutil.SuccessResult("Reading package lists... Done\n"), nil
				}
				if command == testCommand {
					return testutil.SuccessResult(""), nil // Lock file exists
				}
				return testutil.SuccessResult(""), nil
			},
			want:    manager.StatusDegraded,
			wantErr: false,
		},
		{
			name: "degraded with broken packages",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == aptGetCommand && args[0] == checkCommand {
					return testutil.FailureResult(100, "E: Broken packages\n"), nil
				}
				if command == testCommand {
					return testutil.FailureResult(1, ""), nil
				}
				return testutil.SuccessResult(""), nil
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

func TestAdapter_GetVersion_Error(t *testing.T) {
	tests := []struct {
		name     string
		execFunc testutil.ExecutorFunc
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
			name: "empty version output",
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return testutil.SuccessResult(""), nil
			},
			wantErr: true,
		},
		{
			name: "unexpected version format",
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return testutil.SuccessResult("apt"), nil
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

func TestAdapter_GetBinaryPath_Error(t *testing.T) {
	adapter := NewAdapter(testutil.NewMockExecutor(func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
		return nil, errors.New("execution failed")
	}), testutil.NewMockLogger())

	_, err := adapter.GetBinaryPath(context.Background())
	if err == nil {
		t.Error("Expected error for executor failure")
	}
}

func TestAdapter_ListPackages_Error(t *testing.T) {
	adapter := NewAdapter(testutil.NewMockExecutor(func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
		return nil, errors.New("execution failed")
	}), testutil.NewMockLogger())

	_, err := adapter.ListPackages(context.Background())
	if err == nil {
		t.Error("Expected error for executor failure")
	}
}

func TestAdapter_CheckHealth_ExecutorError(t *testing.T) {
	adapter := NewAdapter(testutil.NewMockExecutor(func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
		return nil, errors.New("execution failed")
	}), testutil.NewMockLogger())

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

func TestAdapter_Detect_ExecutorError(t *testing.T) {
	adapter := NewAdapter(testutil.NewMockExecutor(func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
		return nil, errors.New("execution failed")
	}), testutil.NewMockLogger())

	detected, err := adapter.Detect(context.Background())
	if err != nil {
		t.Errorf("Detect() should not return error, got %v", err)
	}

	if detected {
		t.Error("Detect() should return false when executor fails")
	}
}
