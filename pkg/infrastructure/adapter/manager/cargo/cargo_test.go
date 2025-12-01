package cargo

import (
	"context"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
)

// Test-specific constants
const (
	versionFlag = "--version"
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
			name: "cargo installed",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == "which" && len(args) == 1 && args[0] == cargoCommand {
					return &output.ExecutionResult{
						Stdout:   "/usr/bin/cargo\n",
						ExitCode: 0,
					}, nil
				}
				return &output.ExecutionResult{ExitCode: 1}, nil
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "cargo not installed",
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
				if command == cargoCommand && len(args) == 1 && args[0] == versionFlag {
					return &output.ExecutionResult{
						Stdout:   "cargo 1.75.0 (1d8b05cdd 2023-11-20)\n",
						ExitCode: 0,
					}, nil
				}
				return &output.ExecutionResult{ExitCode: 1}, nil
			},
			want:    "1.75.0",
			wantErr: false,
		},
		{
			name: "older cargo version",
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return &output.ExecutionResult{
					Stdout:   "cargo 1.70.0 (38c6b0c90 2023-08-01)\n",
					ExitCode: 0,
				}, nil
			},
			want:    "1.70.0",
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
				if command == "which" && args[0] == cargoCommand {
					return &output.ExecutionResult{
						Stdout:   "/home/user/.cargo/bin/cargo\n",
						ExitCode: 0,
					}, nil
				}
				return &output.ExecutionResult{ExitCode: 1}, nil
			},
			want:    "/home/user/.cargo/bin/cargo",
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

func TestAdapter_ListPackages(t *testing.T) {
	tests := []struct {
		name      string
		execFunc  func(ctx context.Context, command string, args ...string) (*output.ExecutionResult, error)
		wantCount int
		wantErr   bool
	}{
		{
			name: "multiple installed packages",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == cargoCommand && len(args) == 2 && args[0] == installFlag && args[1] == listFlag {
					return &output.ExecutionResult{
						Stdout: `ripgrep v14.0.3:
    rg
cargo-watch v8.4.1:
    cargo-watch
bat v0.24.0:
    bat
`,
						ExitCode: 0,
					}, nil
				}
				return &output.ExecutionResult{ExitCode: 1}, nil
			},
			wantCount: 3,
			wantErr:   false,
		},
		{
			name: "package with local path",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == cargoCommand && args[0] == installFlag && args[1] == listFlag {
					return &output.ExecutionResult{
						Stdout: `ripgrep v14.0.3:
    rg
my-tool v0.1.0 (/home/user/projects/my-tool):
    my-tool
`,
						ExitCode: 0,
					}, nil
				}
				return &output.ExecutionResult{ExitCode: 1}, nil
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name: "no packages installed",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == cargoCommand && args[0] == installFlag && args[1] == listFlag {
					return &output.ExecutionResult{
						Stdout:   "",
						ExitCode: 0,
					}, nil
				}
				return &output.ExecutionResult{ExitCode: 1}, nil
			},
			wantCount: 0,
			wantErr:   false,
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
		execFunc func(ctx context.Context, command string, args ...string) (*output.ExecutionResult, error)
		want     manager.Status
		wantErr  bool
	}{
		{
			name: "healthy system",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == cargoCommand && len(args) == 1 && args[0] == versionFlag {
					return &output.ExecutionResult{
						Stdout:   "cargo 1.75.0 (1d8b05cdd 2023-11-20)\n",
						ExitCode: 0,
					}, nil
				}
				return &output.ExecutionResult{ExitCode: 0}, nil
			},
			want:    manager.StatusHealthy,
			wantErr: false,
		},
		{
			name: "degraded system",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == cargoCommand && args[0] == versionFlag {
					return &output.ExecutionResult{
						Stderr:   "error: failed to run cargo\n",
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
