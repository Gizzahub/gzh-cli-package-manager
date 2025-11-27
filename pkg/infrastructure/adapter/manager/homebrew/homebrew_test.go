package homebrew

import (
	"context"
	"errors"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
)

const brewCommand = "brew"

// mockExecutor implements output.CommandExecutor for testing.
type mockExecutor struct {
	execFunc func(ctx context.Context, command string, args ...string) (*output.ExecutionResult, error)
}

func (m *mockExecutor) Execute(ctx context.Context, command string, args ...string) (*output.ExecutionResult, error) {
	if m.execFunc != nil {
		return m.execFunc(ctx, command, args...)
	}
	return &output.ExecutionResult{ExitCode: 0}, nil
}

func (m *mockExecutor) ExecuteWithInput(_ context.Context, _ string, _ string, _ ...string) (*output.ExecutionResult, error) {
	return &output.ExecutionResult{ExitCode: 0}, nil
}

// mockLogger implements output.Logger for testing.
type mockLogger struct{}

func (m *mockLogger) Debug(_ context.Context, _ string, _ ...output.Field)      {}
func (m *mockLogger) Info(_ context.Context, _ string, _ ...output.Field)       {}
func (m *mockLogger) Warn(_ context.Context, _ string, _ ...output.Field)       {}
func (m *mockLogger) Error(_ context.Context, _ string, _ error, _ ...output.Field) {}

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
			executor := &mockExecutor{execFunc: tt.execFunc}
			logger := &mockLogger{}
			adapter := NewAdapter(executor, logger)

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
				if command == brewCommand && len(args) == 1 && args[0] == "--version" {
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
			executor := &mockExecutor{execFunc: tt.execFunc}
			logger := &mockLogger{}
			adapter := NewAdapter(executor, logger)

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
			executor := &mockExecutor{execFunc: tt.execFunc}
			logger := &mockLogger{}
			adapter := NewAdapter(executor, logger)

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &mockExecutor{execFunc: tt.execFunc}
			logger := &mockLogger{}
			adapter := NewAdapter(executor, logger)

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
