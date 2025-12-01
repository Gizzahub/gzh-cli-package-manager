package asdf

import (
	"context"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
)

// Test-specific constants
const (
	versionArg = "version"
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
			name: "asdf installed",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == "which" && len(args) == 1 && args[0] == asdfCommand {
					return &output.ExecutionResult{
						Stdout:   "/usr/local/bin/asdf\n",
						ExitCode: 0,
					}, nil
				}
				return &output.ExecutionResult{ExitCode: 1}, nil
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "asdf not installed",
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
				if command == asdfCommand && len(args) == 1 && args[0] == versionArg {
					return &output.ExecutionResult{
						Stdout:   "v0.13.1\n",
						ExitCode: 0,
					}, nil
				}
				return &output.ExecutionResult{ExitCode: 1}, nil
			},
			want:    "0.13.1",
			wantErr: false,
		},
		{
			name: "version with git hash",
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return &output.ExecutionResult{
					Stdout:   "v0.13.1-abc1234\n",
					ExitCode: 0,
				}, nil
			},
			want:    "0.13.1",
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
				if command == "which" && args[0] == asdfCommand {
					return &output.ExecutionResult{
						Stdout:   "/home/user/.asdf/bin/asdf\n",
						ExitCode: 0,
					}, nil
				}
				return &output.ExecutionResult{ExitCode: 1}, nil
			},
			want:    "/home/user/.asdf/bin/asdf",
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
			name: "multiple plugins with versions",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				// asdf plugin list
				if command == asdfCommand && len(args) == 2 && args[0] == "plugin" && args[1] == listArg {
					return &output.ExecutionResult{
						Stdout: `nodejs
python
ruby
`,
						ExitCode: 0,
					}, nil
				}
				// asdf list nodejs
				if command == asdfCommand && args[0] == "list" && args[1] == "nodejs" {
					return &output.ExecutionResult{
						Stdout: ` 18.0.0
*20.11.0
 21.0.0
`,
						ExitCode: 0,
					}, nil
				}
				// asdf list python
				if command == asdfCommand && args[0] == "list" && args[1] == "python" {
					return &output.ExecutionResult{
						Stdout: `*3.11.7
 3.12.0
`,
						ExitCode: 0,
					}, nil
				}
				// asdf list ruby
				if command == asdfCommand && args[0] == "list" && args[1] == "ruby" {
					return &output.ExecutionResult{
						Stdout: `*3.2.2
`,
						ExitCode: 0,
					}, nil
				}
				return &output.ExecutionResult{ExitCode: 1}, nil
			},
			wantCount: 6, // 3 nodejs + 2 python + 1 ruby
			wantErr:   false,
		},
		{
			name: "no plugins installed",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == asdfCommand && args[0] == "plugin" && args[1] == listArg {
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

			// Verify current version is marked with IsGlobal
			for _, pkg := range packages {
				if pkg.Name == "nodejs@20.11.0" && !pkg.IsGlobal {
					t.Error("Current version should be marked as global")
				}
				if pkg.Name == "nodejs@18.0.0" && pkg.IsGlobal {
					t.Error("Non-current version should not be marked as global")
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
				if command == asdfCommand && len(args) == 1 && args[0] == versionArg {
					return &output.ExecutionResult{
						Stdout:   "v0.13.1\n",
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
				if command == asdfCommand && args[0] == versionArg {
					return &output.ExecutionResult{
						Stderr:   "error: asdf not properly installed\n",
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
