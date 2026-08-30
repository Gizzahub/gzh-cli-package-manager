package homebrew

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/testutil"
)

const (
	brewCommand                       = "brew"
	brewVersionFlag                   = "--version"
	brewWhichCommand                  = "which"
	testBrewDoctorCommand             = "doctor"
	testBrewUpdateCommand             = "update"
	testBrewUpgradeCommand            = "upgrade"
	testNonZeroExitCodeCaseName       = "non-zero exit code"
	testCommandExecutionErrorCaseName = "command execution error"
	testBrewStableStrategy            = "stable"
)

type homebrewUpdateCall struct {
	args   []string
	err    error
	result *output.ExecutionResult
}

type homebrewUpdateExpectation struct {
	err     error
	message string
	success bool
	updated []string
	wantErr bool
}

func newHomebrewUpdateExecutor(t *testing.T, calls []homebrewUpdateCall) (executor testutil.ExecutorFunc, verify func()) {
	t.Helper()
	callIndex := 0
	executor = func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		t.Helper()
		if callIndex == len(calls) {
			t.Fatalf("unexpected command %q %v", command, args)
		}
		call := calls[callIndex]
		callIndex++
		if command != brewCommand || !slices.Equal(args, call.args) {
			t.Errorf("command = %q %v, want %q %v", command, args, brewCommand, call.args)
		}
		return call.result, call.err
	}
	verify = func() {
		t.Helper()
		if callIndex != len(calls) {
			t.Errorf("command calls = %d, want %d", callIndex, len(calls))
		}
	}
	return executor, verify
}

func assertHomebrewUpdateResult(t *testing.T, result *adapterm.UpdateResult, err error, want homebrewUpdateExpectation) {
	t.Helper()
	switch {
	case want.err != nil:
		if !errors.Is(err, want.err) {
			t.Fatalf("Update() error = %v, want errors.Is(..., %v)", err, want.err)
		}
	case want.wantErr:
		if err == nil {
			t.Fatal("Update() error = nil, want error")
		}
	default:
		if err != nil {
			t.Fatalf("Update() error = %v, want nil", err)
		}
	}
	if result == nil {
		t.Fatal("Update() result = nil")
	}
	if result.Success != want.success {
		t.Errorf("Success = %v, want %v", result.Success, want.success)
	}
	if result.Message != want.message {
		t.Errorf("Message = %q, want %q", result.Message, want.message)
	}
	if !slices.Equal(result.UpdatedPackages, want.updated) {
		t.Errorf("UpdatedPackages = %v, want %v", result.UpdatedPackages, want.updated)
	}
}

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
				if command == brewWhichCommand && len(args) == 1 && args[0] == brewCommand {
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
				if command == brewWhichCommand && len(args) == 1 && args[0] == brewCommand {
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
			name: testCommandExecutionErrorCaseName,
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
				if command == brewCommand && len(args) == 1 && args[0] == testBrewDoctorCommand {
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
				if command == brewCommand && len(args) == 1 && args[0] == testBrewDoctorCommand {
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
			name: testCommandExecutionErrorCaseName,
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return nil, errors.New("command failed")
			},
			want:    manager.StatusDegraded,
			wantErr: false,
		},
		{
			name: "error status without warning keyword",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == brewCommand && len(args) == 1 && args[0] == testBrewDoctorCommand {
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
				if command == brewWhichCommand && len(args) == 1 && args[0] == brewCommand {
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
				if command == brewWhichCommand && len(args) == 1 && args[0] == brewCommand {
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
			name: testCommandExecutionErrorCaseName,
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
			name: testNonZeroExitCodeCaseName,
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
			name: testCommandExecutionErrorCaseName,
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return nil, errors.New("command failed")
			},
			wantErr: true,
		},
		{
			name: testNonZeroExitCodeCaseName,
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
			name: testCommandExecutionErrorCaseName,
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return nil, errors.New("command failed")
			},
			wantErr: true,
		},
		{
			name: testNonZeroExitCodeCaseName,
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
	updateErr := errors.New("network error")
	upgradeErr := errors.New("upgrade error")
	tests := []struct {
		name  string
		opts  adapterm.UpdateOptions
		calls []homebrewUpdateCall
		want  homebrewUpdateExpectation
	}{
		{
			name: "dry run mode",
			opts: adapterm.UpdateOptions{DryRun: true},
			want: homebrewUpdateExpectation{
				success: true,
				updated: []string{},
				message: "Dry-run: would update Homebrew and packages",
			},
		},
		{
			name: "update and upgrade success",
			opts: adapterm.UpdateOptions{Strategy: adapterm.StrategyStable},
			calls: []homebrewUpdateCall{
				{args: []string{testBrewUpdateCommand}, result: testutil.SuccessResult("Updated Homebrew\n")},
				{args: []string{testBrewUpgradeCommand}, result: testutil.SuccessResult("Upgrading git\n==> Upgrading 1 outdated package:\ngit 2.42.0 -> 2.43.0\n")},
			},
			want: homebrewUpdateExpectation{
				success: true,
				updated: []string{"git"},
				message: "Homebrew and 1 packages updated successfully",
			},
		},
		{
			name: "update fails",
			opts: adapterm.UpdateOptions{Strategy: adapterm.StrategyStable},
			calls: []homebrewUpdateCall{
				{args: []string{testBrewUpdateCommand}, err: updateErr},
			},
			want: homebrewUpdateExpectation{
				err:     updateErr,
				success: false,
				updated: []string{},
				message: "brew update failed: network error",
			},
		},
		{
			name: "update non-zero exit code",
			opts: adapterm.UpdateOptions{Strategy: adapterm.StrategyStable},
			calls: []homebrewUpdateCall{
				{args: []string{testBrewUpdateCommand}, result: testutil.FailureResult(1, "update failed")},
			},
			want: homebrewUpdateExpectation{
				success: false,
				updated: []string{},
				message: "brew update failed: update failed",
				wantErr: true,
			},
		},
		{
			name: "fixed strategy skips upgrade",
			opts: adapterm.UpdateOptions{Strategy: adapterm.StrategyFixed},
			calls: []homebrewUpdateCall{
				{args: []string{testBrewUpdateCommand}, result: testutil.SuccessResult("Updated Homebrew\n")},
			},
			want: homebrewUpdateExpectation{
				success: true,
				updated: []string{},
				message: "Strategy 'fixed': Homebrew updated, packages not upgraded",
			},
		},
		{
			name: "upgrade fails with error",
			opts: adapterm.UpdateOptions{Strategy: adapterm.StrategyStable},
			calls: []homebrewUpdateCall{
				{args: []string{testBrewUpdateCommand}, result: testutil.SuccessResult("")},
				{args: []string{testBrewUpgradeCommand}, err: upgradeErr},
			},
			want: homebrewUpdateExpectation{
				success: true,
				updated: []string{},
				message: "Homebrew updated, but package upgrade failed",
			},
		},
		{
			name: "upgrade non-zero exit code",
			opts: adapterm.UpdateOptions{Strategy: adapterm.StrategyStable},
			calls: []homebrewUpdateCall{
				{args: []string{testBrewUpdateCommand}, result: testutil.SuccessResult("")},
				{args: []string{testBrewUpgradeCommand}, result: testutil.FailureResult(1, "some packages failed")},
			},
			want: homebrewUpdateExpectation{
				success: true,
				updated: []string{},
				message: "Homebrew updated, but some packages failed to upgrade",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor, verify := newHomebrewUpdateExecutor(t, tt.calls)
			result, err := NewAdapter(testutil.NewMockExecutor(executor), testutil.NewMockLogger()).Update(context.Background(), tt.opts)
			verify()

			assertHomebrewUpdateResult(t, result, err, tt.want)
		})
	}
}

func TestAdapter_Detect_EmptyOutput(t *testing.T) {
	execFunc := func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		if command == brewWhichCommand && len(args) == 1 && args[0] == brewCommand {
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
