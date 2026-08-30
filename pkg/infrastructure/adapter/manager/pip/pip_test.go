package pip

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

// Test-specific constants.
const (
	pipConfigPath      = "~/.pip/pip.conf"
	testPip3BinaryPath = "/usr/bin/pip3"
	testPipPackage     = "requests"
)

type pipUpdateCall struct {
	args    []string
	command string
	err     error
	result  *output.ExecutionResult
}

type pipUpdateExpectation struct {
	err     error
	message string
	success bool
	updated []string
}

func newPipUpdateExecutor(t *testing.T, calls []pipUpdateCall) (executor testutil.ExecutorFunc, verify func()) {
	t.Helper()
	callIndex := 0
	executor = func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		t.Helper()
		if callIndex == len(calls) {
			t.Fatalf("unexpected command %q %v", command, args)
		}
		call := calls[callIndex]
		callIndex++
		if command != call.command || !slices.Equal(args, call.args) {
			t.Errorf("command = %q %v, want %q %v", command, args, call.command, call.args)
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

func assertPipUpdateResult(t *testing.T, result *adapterm.UpdateResult, err error, want pipUpdateExpectation) {
	t.Helper()
	if want.err == nil {
		if err != nil {
			t.Fatalf("Update() error = %v, want nil", err)
		}
	} else if !errors.Is(err, want.err) {
		t.Fatalf("Update() error = %v, want errors.Is(..., %v)", err, want.err)
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

func TestAdapter_Detect(t *testing.T) {
	tests := []struct {
		name     string
		execFunc func(ctx context.Context, command string, args ...string) (*output.ExecutionResult, error)
		want     bool
		wantErr  bool
	}{
		{
			name: "pip3 installed",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == whichCommand && len(args) == 1 && args[0] == pip3Command {
					return &output.ExecutionResult{
						Stdout:   "/usr/bin/pip3\n",
						ExitCode: 0,
					}, nil
				}
				return &output.ExecutionResult{ExitCode: 1}, nil
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "pip installed (fallback)",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == whichCommand && len(args) == 1 {
					if args[0] == pip3Command {
						return &output.ExecutionResult{ExitCode: 1}, nil
					}
					if args[0] == pipCommand {
						return &output.ExecutionResult{
							Stdout:   "/usr/bin/pip\n",
							ExitCode: 0,
						}, nil
					}
				}
				return &output.ExecutionResult{ExitCode: 1}, nil
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "pip not installed",
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
				if command == whichCommand && args[0] == pip3Command {
					return &output.ExecutionResult{Stdout: testPip3BinaryPath, ExitCode: 0}, nil
				}
				if command == pip3Command && args[0] == "--version" {
					return &output.ExecutionResult{
						Stdout:   "pip 23.0.1 from /usr/lib/python3.11/site-packages/pip (python 3.11)\n",
						ExitCode: 0,
					}, nil
				}
				return &output.ExecutionResult{ExitCode: 1}, nil
			},
			want:    "23.0.1",
			wantErr: false,
		},
		{
			name: "pip version (not pip3)",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == whichCommand && args[0] == pip3Command {
					return &output.ExecutionResult{ExitCode: 1}, nil
				}
				if command == pipCommand && args[0] == "--version" {
					return &output.ExecutionResult{
						Stdout:   "pip 22.3.1 from /usr/lib/python3.10/site-packages/pip (python 3.10)\n",
						ExitCode: 0,
					}, nil
				}
				return &output.ExecutionResult{ExitCode: 0}, nil
			},
			want:    "22.3.1",
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
			name: "pip3 binary found",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == whichCommand {
					if args[0] == pip3Command {
						return &output.ExecutionResult{
							Stdout:   "/usr/bin/pip3\n",
							ExitCode: 0,
						}, nil
					}
				}
				return &output.ExecutionResult{ExitCode: 1}, nil
			},
			want:    testPip3BinaryPath,
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
	if got != pipConfigPath {
		t.Errorf("GetConfigPath() = %v, want ~/.pip/pip.conf", got)
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
				if command == whichCommand && args[0] == pip3Command {
					return &output.ExecutionResult{Stdout: testPip3BinaryPath, ExitCode: 0}, nil
				}
				// pip3 list --format=json
				if command == pip3Command && len(args) == 2 && args[0] == listArg && args[1] == "--format=json" {
					return &output.ExecutionResult{
						Stdout: `[
							{"name": "pip", "version": "23.0.1"},
							{"name": "requests", "version": "2.28.0"},
							{"name": "numpy", "version": "1.24.0"}
						]`,
						ExitCode: 0,
					}, nil
				}
				// pip3 list --outdated --format=json
				if command == pip3Command && len(args) == 3 && args[0] == listArg && args[1] == outdatedFlag {
					return &output.ExecutionResult{
						Stdout: `[
							{"name": "requests", "version": "2.28.0", "latest_version": "2.31.0", "latest_filetype": "wheel"},
							{"name": "numpy", "version": "1.24.0", "latest_version": "1.26.4", "latest_filetype": "wheel"}
						]`,
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
				if command == whichCommand && args[0] == pip3Command {
					return &output.ExecutionResult{Stdout: testPip3BinaryPath, ExitCode: 0}, nil
				}
				if command == pip3Command && args[0] == listArg && args[1] == "--format=json" {
					return &output.ExecutionResult{
						Stdout:   `[{"name": "pip", "version": "23.0.1"}]`,
						ExitCode: 0,
					}, nil
				}
				if command == pip3Command && args[0] == listArg && args[1] == outdatedFlag {
					return &output.ExecutionResult{
						Stdout:   "[]",
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
					t.Error("Pip packages should be global")
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
				if command == whichCommand && args[0] == pip3Command {
					return &output.ExecutionResult{Stdout: testPip3BinaryPath, ExitCode: 0}, nil
				}
				if command == pip3Command && len(args) == 1 && args[0] == "check" {
					return &output.ExecutionResult{
						Stdout:   "No broken requirements found.\n",
						ExitCode: 0,
					}, nil
				}
				return &output.ExecutionResult{ExitCode: 0}, nil
			},
			want:    manager.StatusHealthy,
			wantErr: false,
		},
		{
			name: "degraded with dependency issues",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == whichCommand && args[0] == pip3Command {
					return &output.ExecutionResult{Stdout: testPip3BinaryPath, ExitCode: 0}, nil
				}
				if command == pip3Command && args[0] == "check" {
					return &output.ExecutionResult{
						Stderr:   "package-a 1.0 has requirement package-b>=2.0, but you have package-b 1.5.\n",
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
	execFunc := func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
		return nil, errors.New("execution failed")
	}
	adapter := NewAdapter(testutil.NewMockExecutor(execFunc), testutil.NewMockLogger())

	_, err := adapter.GetVersion(context.Background())
	if err == nil {
		t.Error("Expected error for executor failure")
	}
}

func TestAdapter_GetVersion_ParseError(t *testing.T) {
	execFunc := func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		if command == whichCommand {
			return &output.ExecutionResult{Stdout: testPip3BinaryPath, ExitCode: 0}, nil
		}
		// Return output with less than 2 fields (only one word)
		return &output.ExecutionResult{
			Stdout:   pipCommand,
			ExitCode: 0,
		}, nil
	}
	adapter := NewAdapter(testutil.NewMockExecutor(execFunc), testutil.NewMockLogger())

	_, err := adapter.GetVersion(context.Background())
	if err == nil {
		t.Error("Expected error for parse failure")
	}
}

func TestAdapter_GetBinaryPath_Error(t *testing.T) {
	execFunc := func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
		return nil, errors.New("execution failed")
	}
	adapter := NewAdapter(testutil.NewMockExecutor(execFunc), testutil.NewMockLogger())

	_, err := adapter.GetBinaryPath(context.Background())
	if err == nil {
		t.Error("Expected error for executor failure")
	}
}

func TestAdapter_ListPackages_Error(t *testing.T) {
	execFunc := func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		if command == whichCommand {
			return &output.ExecutionResult{Stdout: testPip3BinaryPath, ExitCode: 0}, nil
		}
		return nil, errors.New("execution failed")
	}
	adapter := NewAdapter(testutil.NewMockExecutor(execFunc), testutil.NewMockLogger())

	_, err := adapter.ListPackages(context.Background())
	if err == nil {
		t.Error("Expected error for executor failure")
	}
}

func TestAdapter_ListPackages_InvalidJSON(t *testing.T) {
	execFunc := func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		if command == whichCommand {
			return &output.ExecutionResult{Stdout: testPip3BinaryPath, ExitCode: 0}, nil
		}
		if command == pip3Command && args[0] == listArg {
			return &output.ExecutionResult{
				Stdout:   "not valid json",
				ExitCode: 0,
			}, nil
		}
		return &output.ExecutionResult{ExitCode: 0}, nil
	}
	adapter := NewAdapter(testutil.NewMockExecutor(execFunc), testutil.NewMockLogger())

	_, err := adapter.ListPackages(context.Background())
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestAdapter_CheckHealth_ExecutorError(t *testing.T) {
	execFunc := func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		if command == whichCommand {
			return &output.ExecutionResult{Stdout: testPip3BinaryPath, ExitCode: 0}, nil
		}
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

func TestAdapter_UpdateNonDryRunContracts(t *testing.T) {
	discoveryErr := errors.New("discovery failed")
	installErr := errors.New("install failed")
	tests := []struct {
		calls []pipUpdateCall
		name  string
		opts  adapterm.UpdateOptions
		want  pipUpdateExpectation
	}{
		{
			name: "explicit packages skip discovery",
			opts: adapterm.UpdateOptions{Packages: []string{testPipPackage}},
			want: pipUpdateExpectation{success: true, updated: []string{testPipPackage}, message: "pip packages upgraded"},
			calls: []pipUpdateCall{
				{command: pipCommand, args: []string{installArg, upgradeFlag, testPipPackage}, result: testutil.SuccessResult("")},
			},
		},
		{
			name: "discovered packages preserve order",
			want: pipUpdateExpectation{success: true, updated: []string{testPipPackage, "flask"}, message: "pip packages upgraded"},
			calls: []pipUpdateCall{
				{command: pipCommand, args: []string{listArg, outdatedFlag, freezeFormat}, result: testutil.SuccessResult(testPipPackage + "==2.28.0\n flask==3.0.0 \n\n")},
				{command: pipCommand, args: []string{installArg, upgradeFlag, testPipPackage, "flask"}, result: testutil.SuccessResult("")},
			},
		},
		{
			name: "empty discovery has no install",
			want: pipUpdateExpectation{success: true, updated: []string{}, message: noOutdatedPackagesMessage},
			calls: []pipUpdateCall{
				{command: pipCommand, args: []string{listArg, outdatedFlag, freezeFormat}, result: testutil.SuccessResult(" \n")},
			},
		},
		{
			name: "discovery error has no install",
			want: pipUpdateExpectation{success: true, updated: []string{}, message: noOutdatedPackagesMessage},
			calls: []pipUpdateCall{
				{command: pipCommand, args: []string{listArg, outdatedFlag, freezeFormat}, err: discoveryErr},
			},
		},
		{
			name: "nonzero discovery has no install",
			want: pipUpdateExpectation{success: true, updated: []string{}, message: noOutdatedPackagesMessage},
			calls: []pipUpdateCall{
				{command: pipCommand, args: []string{listArg, outdatedFlag, freezeFormat}, result: &output.ExecutionResult{ExitCode: 1}},
			},
		},
		{
			name: "install error preserves cause",
			opts: adapterm.UpdateOptions{Packages: []string{testPipPackage}},
			want: pipUpdateExpectation{err: installErr, success: false, updated: []string{}, message: "pip install --upgrade failed: install failed"},
			calls: []pipUpdateCall{
				{command: pipCommand, args: []string{installArg, upgradeFlag, testPipPackage}, err: installErr},
			},
		},
		{
			name: "nonzero install reports stderr",
			opts: adapterm.UpdateOptions{Packages: []string{testPipPackage}},
			want: pipUpdateExpectation{success: false, updated: []string{}, message: "pip upgrade failed: permission denied"},
			calls: []pipUpdateCall{
				{command: pipCommand, args: []string{installArg, upgradeFlag, testPipPackage}, result: &output.ExecutionResult{ExitCode: 1, Stderr: "permission denied"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor, verify := newPipUpdateExecutor(t, tt.calls)
			result, err := NewAdapter(testutil.NewMockExecutor(executor), testutil.NewMockLogger()).Update(context.Background(), tt.opts)
			verify()

			assertPipUpdateResult(t, result, err, tt.want)
		})
	}
}

func TestAdapter_Detect_ExecutorError(t *testing.T) {
	execFunc := func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
		return nil, errors.New("execution failed")
	}
	adapter := NewAdapter(testutil.NewMockExecutor(execFunc), testutil.NewMockLogger())

	detected, err := adapter.Detect(context.Background())
	if err != nil {
		t.Errorf("Detect() should not return error, got %v", err)
	}

	if detected {
		t.Error("Detect() should return false when executor fails")
	}
}
