package apt

import (
	"context"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
)

// Test-specific constants
const (
	testCommand = "test"
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
			name: "apt installed",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == "which" && len(args) == 1 && args[0] == aptCommand {
					return &output.ExecutionResult{
						Stdout:   "/usr/bin/apt\n",
						ExitCode: 0,
					}, nil
				}
				return &output.ExecutionResult{ExitCode: 1}, nil
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "apt not installed",
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return &output.ExecutionResult{
					ExitCode: 1,
				}, nil
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
				if command == aptCommand && len(args) == 1 && args[0] == "--version" {
					return &output.ExecutionResult{
						Stdout:   "apt 2.4.11 (amd64)\n",
						ExitCode: 0,
					}, nil
				}
				return &output.ExecutionResult{ExitCode: 1}, nil
			},
			want:    "2.4.11",
			wantErr: false,
		},
		{
			name: "version with extra info",
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return &output.ExecutionResult{
					Stdout:   "apt 2.0.6 (arm64)\nCompiled on Jul 15 2023\n",
					ExitCode: 0,
				}, nil
			},
			want:    "2.0.6",
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
			name: "binary found",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == "which" && args[0] == aptCommand {
					return &output.ExecutionResult{
						Stdout:   "/usr/bin/apt\n",
						ExitCode: 0,
					}, nil
				}
				return &output.ExecutionResult{ExitCode: 1}, nil
			},
			want:    "/usr/bin/apt",
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
	if got != "/etc/apt" {
		t.Errorf("GetConfigPath() = %v, want /etc/apt", got)
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
				// apt list --installed
				if command == aptCommand && len(args) == 2 && args[0] == listArg && args[1] == "--installed" {
					return &output.ExecutionResult{
						Stdout: `Listing...
vim/jammy-updates,now 2:8.2.3995-1ubuntu2.12 amd64 [installed]
curl/jammy-security,now 7.81.0-1ubuntu1.15 amd64 [installed,automatic]
git/jammy-updates,now 1:2.34.1-1ubuntu1.10 amd64 [installed]
`,
						ExitCode: 0,
					}, nil
				}
				// apt list --upgradable
				if command == aptCommand && len(args) == 2 && args[0] == listArg && args[1] == "--upgradable" {
					return &output.ExecutionResult{
						Stdout: `Listing...
curl/jammy-security 7.81.0-1ubuntu1.16 amd64 [upgradable from: 7.81.0-1ubuntu1.15]
git/jammy-updates 1:2.34.1-1ubuntu1.11 amd64 [upgradable from: 1:2.34.1-1ubuntu1.10]
`,
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
				if command == aptCommand && args[0] == listArg && args[1] == "--installed" {
					return &output.ExecutionResult{
						Stdout: `Listing...
vim/jammy-updates,now 2:8.2.3995-1ubuntu2.12 amd64 [installed]
`,
						ExitCode: 0,
					}, nil
				}
				if command == aptCommand && args[0] == listArg && args[1] == "--upgradable" {
					return &output.ExecutionResult{
						Stdout:   "Listing...\n",
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
					t.Error("APT packages should be global")
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
				if command == aptGetCommand && len(args) == 1 && args[0] == checkCommand {
					return &output.ExecutionResult{
						Stdout:   "Reading package lists... Done\nBuilding dependency tree... Done\n",
						ExitCode: 0,
					}, nil
				}
				if command == testCommand {
					return &output.ExecutionResult{ExitCode: 1}, nil // Lock file doesn't exist
				}
				return &output.ExecutionResult{ExitCode: 0}, nil
			},
			want:    manager.StatusHealthy,
			wantErr: false,
		},
		{
			name: "degraded with lock file",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == aptGetCommand && args[0] == checkCommand {
					return &output.ExecutionResult{
						Stdout:   "Reading package lists... Done\n",
						ExitCode: 0,
					}, nil
				}
				if command == testCommand {
					return &output.ExecutionResult{ExitCode: 0}, nil // Lock file exists
				}
				return &output.ExecutionResult{ExitCode: 0}, nil
			},
			want:    manager.StatusDegraded,
			wantErr: false,
		},
		{
			name: "degraded with broken packages",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == aptGetCommand && args[0] == checkCommand {
					return &output.ExecutionResult{
						Stderr:   "E: Broken packages\n",
						ExitCode: 100,
					}, nil
				}
				if command == testCommand {
					return &output.ExecutionResult{ExitCode: 1}, nil
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
