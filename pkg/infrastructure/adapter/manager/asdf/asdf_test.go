package asdf

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

// Test-specific constants
const (
	versionArg = "version"
)

// errMockExecution is a sentinel error for testing executor failures.
var errMockExecution = errors.New("mock execution error")

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
				if command == asdfCommand && args[0] == listArg && args[1] == "nodejs" {
					return &output.ExecutionResult{
						Stdout: ` 18.0.0
*20.11.0
 21.0.0
`,
						ExitCode: 0,
					}, nil
				}
				// asdf list python
				if command == asdfCommand && args[0] == listArg && args[1] == "python" {
					return &output.ExecutionResult{
						Stdout: `*3.11.7
 3.12.0
`,
						ExitCode: 0,
					}, nil
				}
				// asdf list ruby
				if command == asdfCommand && args[0] == listArg && args[1] == "ruby" {
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
		execFunc func(ctx context.Context, command string, args ...string) (*output.ExecutionResult, error)
		want     string
		wantErr  bool
	}{
		{
			name: "execute error",
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return nil, errMockExecution
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "version without v prefix",
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return &output.ExecutionResult{
					Stdout:   "0.14.0\n",
					ExitCode: 0,
				}, nil
			},
			want:    "0.14.0",
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

func TestAdapter_GetBinaryPath_Error(t *testing.T) {
	tests := []struct {
		name     string
		execFunc func(ctx context.Context, command string, args ...string) (*output.ExecutionResult, error)
		want     string
		wantErr  bool
	}{
		{
			name: "execute error",
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return nil, errMockExecution
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

func TestAdapter_GetConfigPath(t *testing.T) {
	homeErr := errors.New("home directory unavailable")

	t.Run("resolved home directory", func(t *testing.T) {
		homeDir := t.TempDir()
		adapter := NewAdapter(testutil.NewMockExecutor(nil), testutil.NewMockLogger())
		adapter.homeDirResolver = &homeDirResolver{resolve: func() (string, error) {
			return homeDir, nil
		}}

		got, err := adapter.GetConfigPath(context.Background())
		if err != nil {
			t.Fatalf("GetConfigPath() error = %v", err)
		}
		if want := filepath.Join(homeDir, ".asdfrc"); got != want {
			t.Errorf("GetConfigPath() = %q, want %q", got, want)
		}
	})

	t.Run("home directory error", func(t *testing.T) {
		adapter := NewAdapter(testutil.NewMockExecutor(nil), testutil.NewMockLogger())
		adapter.homeDirResolver = &homeDirResolver{resolve: func() (string, error) {
			return "", homeErr
		}}

		got, err := adapter.GetConfigPath(context.Background())
		if !errors.Is(err, homeErr) {
			t.Errorf("GetConfigPath() error = %v, want cause %v", err, homeErr)
		}
		if got != "" {
			t.Errorf("GetConfigPath() = %q, want empty path", got)
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
	if want := filepath.Join(homeDir, ".asdfrc"); got != want {
		t.Errorf("GetConfigPath() = %q, want %q", got, want)
	}
}

func TestAdapter_ListPackages_Error(t *testing.T) {
	tests := []struct {
		name      string
		execFunc  func(ctx context.Context, command string, args ...string) (*output.ExecutionResult, error)
		wantCount int
		wantErr   bool
	}{
		{
			name: "plugin list error",
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return nil, errMockExecution
			},
			wantCount: 0,
			wantErr:   true,
		},
		{
			name: "version list error for one plugin",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == asdfCommand && len(args) == 2 && args[0] == pluginArg && args[1] == listArg {
					return &output.ExecutionResult{
						Stdout:   "nodejs\npython\n",
						ExitCode: 0,
					}, nil
				}
				// Version list fails
				if command == asdfCommand && args[0] == listArg {
					return nil, errMockExecution
				}
				return &output.ExecutionResult{ExitCode: 0}, nil
			},
			wantCount: 0, // No packages because version listing failed
			wantErr:   false,
		},
		{
			name: "empty version output",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == asdfCommand && len(args) == 2 && args[0] == pluginArg && args[1] == listArg {
					return &output.ExecutionResult{
						Stdout:   "nodejs\n",
						ExitCode: 0,
					}, nil
				}
				if command == asdfCommand && args[0] == listArg {
					return &output.ExecutionResult{
						Stdout:   "",
						ExitCode: 0,
					}, nil
				}
				return &output.ExecutionResult{ExitCode: 0}, nil
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
		})
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
	execFunc := func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
		return nil, errMockExecution
	}
	adapter := NewAdapter(testutil.NewMockExecutor(execFunc), testutil.NewMockLogger())

	detected, err := adapter.Detect(context.Background())
	// Should not return error, just return false
	if err != nil {
		t.Errorf("Detect() should not return error, got %v", err)
	}

	if detected {
		t.Error("Detect() should return false when executor fails")
	}
}

func TestAdapter_CheckHealth_ExecutorError(t *testing.T) {
	execFunc := func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
		return nil, errMockExecution
	}
	adapter := NewAdapter(testutil.NewMockExecutor(execFunc), testutil.NewMockLogger())

	status, err := adapter.CheckHealth(context.Background())
	// Should not return error, just return degraded
	if err != nil {
		t.Errorf("CheckHealth() should not return error, got %v", err)
	}

	if status != manager.StatusDegraded {
		t.Errorf("CheckHealth() should return StatusDegraded, got %v", status)
	}
}
