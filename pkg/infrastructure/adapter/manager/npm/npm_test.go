package npm

import (
	"context"
	"errors"
	"maps"
	"slices"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/testutil"
)

const (
	npmCommand         = "npm"
	whichCommand       = "which"
	doctorCommand      = "doctor"
	testNPMListCommand = "list"
	testNPMGlobalFlag  = "-g"
	testNPMDepthFlag   = "--depth=0"
	testNPMJSONFlag    = "--json"
	testNPMOutdated    = "outdated"
	testNPMTypeScript  = "typescript"
)

var errNPMListPackages = errors.New("npm list executor failed")

type npmListPackagesCall struct {
	args   []string
	result *output.ExecutionResult
	err    error
}

type npmListPackagesExpectation struct {
	packages map[string]manager.Package
	err      error
	wantErr  bool
}

func newNPMListPackagesExecutor(t *testing.T, calls []npmListPackagesCall) (executor output.CommandExecutor, assertCalls func()) {
	t.Helper()

	callIndex := 0
	executor = testutil.NewMockExecutor(func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		if callIndex >= len(calls) {
			t.Fatalf("Execute() received unexpected call %q %q", command, args)
		}

		call := calls[callIndex]
		callIndex++
		if command != npmCommand || !slices.Equal(args, call.args) {
			t.Errorf("Execute() = %q %q, want %q %q", command, args, npmCommand, call.args)
		}

		return call.result, call.err
	})
	assertCalls = func() {
		if callIndex != len(calls) {
			t.Errorf("Execute() calls = %d, want %d", callIndex, len(calls))
		}
	}
	return executor, assertCalls
}

func assertNPMListPackages(t *testing.T, got []manager.Package, err error, want npmListPackagesExpectation) {
	t.Helper()

	switch {
	case want.err != nil:
		if !errors.Is(err, want.err) {
			t.Fatalf("ListPackages() error = %v, want errors.Is(_, %v)", err, want.err)
		}
		return
	case want.wantErr:
		if err == nil {
			t.Fatal("ListPackages() error = nil, want an error")
		}
		return
	case err != nil:
		t.Fatalf("ListPackages() unexpected error = %v", err)
	}

	if got == nil {
		t.Fatal("ListPackages() returned a nil package slice")
	}

	gotByName := make(map[string]manager.Package, len(got))
	for _, pkg := range got {
		if _, exists := gotByName[pkg.Name]; exists {
			t.Fatalf("ListPackages() returned duplicate package %q", pkg.Name)
		}
		gotByName[pkg.Name] = pkg
	}
	if !maps.Equal(gotByName, want.packages) {
		t.Errorf("ListPackages() packages = %#v, want %#v", gotByName, want.packages)
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
			name: "npm installed",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == whichCommand && len(args) == 1 && args[0] == npmCommand {
					return &output.ExecutionResult{
						ExitCode: 0,
						Stdout:   "/usr/bin/npm\n",
					}, nil
				}
				return nil, errors.New("unexpected command")
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "npm not installed",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == whichCommand && len(args) == 1 && args[0] == npmCommand {
					return &output.ExecutionResult{
						ExitCode: 1,
						Stderr:   "npm not found",
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
				if command == npmCommand && len(args) == 1 && args[0] == "--version" {
					return &output.ExecutionResult{
						ExitCode: 0,
						Stdout:   "10.5.0\n",
					}, nil
				}
				return nil, errors.New("unexpected command")
			},
			want:    "10.5.0",
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
				if command == whichCommand && len(args) == 1 && args[0] == npmCommand {
					return &output.ExecutionResult{
						ExitCode: 0,
						Stdout:   "/usr/bin/npm\n",
					}, nil
				}
				return nil, errors.New("unexpected command")
			},
			want:    "/usr/bin/npm",
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
	tests := []struct {
		name     string
		execFunc func(ctx context.Context, command string, args ...string) (*output.ExecutionResult, error)
		want     string
		wantErr  bool
	}{
		{
			name: "config path found",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == npmCommand && len(args) == 3 && args[0] == "config" {
					return &output.ExecutionResult{
						ExitCode: 0,
						Stdout:   "/usr/local\n",
					}, nil
				}
				return nil, errors.New("unexpected command")
			},
			want:    "/usr/local",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter(testutil.NewMockExecutor(tt.execFunc), testutil.NewMockLogger())

			got, err := adapter.GetConfigPath(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("GetConfigPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetConfigPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAdapter_ListPackages(t *testing.T) {
	tests := []struct {
		name  string
		calls []npmListPackagesCall
		want  npmListPackagesExpectation
	}{
		{
			name: "preserves complete package details and updates",
			calls: []npmListPackagesCall{
				{
					args: []string{testNPMListCommand, testNPMGlobalFlag, testNPMDepthFlag, testNPMJSONFlag},
					result: testutil.SuccessResult(`{
						"dependencies": {
							"react": {"version": "18.0.0"},
							"typescript": {"version": "5.0.0"},
							"vue": {"version": "3.0.0"}
						}
					}`),
				},
				{
					args: []string{testNPMOutdated, testNPMGlobalFlag, testNPMJSONFlag},
					result: &output.ExecutionResult{
						ExitCode: 1, // npm outdated returns 1 when packages are outdated.
						Stdout: `{
							"react": {"latest": "19.0.0"},
							"vue": {"latest": "3.4.0"}
						}`,
					},
				},
			},
			want: npmListPackagesExpectation{packages: map[string]manager.Package{
				"react": {
					Name:             "react",
					CurrentVersion:   "18.0.0",
					AvailableVersion: "19.0.0",
					IsGlobal:         true,
					UpdateType:       manager.UpdateMajor,
				},
				testNPMTypeScript: {
					Name:           testNPMTypeScript,
					CurrentVersion: "5.0.0",
					IsGlobal:       true,
					UpdateType:     manager.UpdateNone,
				},
				"vue": {
					Name:             "vue",
					CurrentVersion:   "3.0.0",
					AvailableVersion: "3.4.0",
					IsGlobal:         true,
					UpdateType:       manager.UpdateMinor,
				},
			}},
		},
		{
			name: "returns packages when outdated output is invalid",
			calls: []npmListPackagesCall{
				{
					args:   []string{testNPMListCommand, testNPMGlobalFlag, testNPMDepthFlag, testNPMJSONFlag},
					result: testutil.SuccessResult(`{"dependencies":{"typescript":{"version":"5.0.0"}}}`),
				},
				{
					args:   []string{testNPMOutdated, testNPMGlobalFlag, testNPMJSONFlag},
					result: testutil.SuccessResult("not valid json"),
				},
			},
			want: npmListPackagesExpectation{packages: map[string]manager.Package{
				testNPMTypeScript: {
					Name:           testNPMTypeScript,
					CurrentVersion: "5.0.0",
					IsGlobal:       true,
					UpdateType:     manager.UpdateNone,
				},
			}},
		},
		{
			name: "returns a non-nil empty slice when no dependencies exist",
			calls: []npmListPackagesCall{
				{
					args:   []string{testNPMListCommand, testNPMGlobalFlag, testNPMDepthFlag, testNPMJSONFlag},
					result: testutil.SuccessResult(`{"dependencies":{}}`),
				},
				{
					args:   []string{testNPMOutdated, testNPMGlobalFlag, testNPMJSONFlag},
					result: testutil.SuccessResult("{}"),
				},
			},
			want: npmListPackagesExpectation{packages: map[string]manager.Package{}},
		},
		{
			name: "wraps the list executor error without running outdated",
			calls: []npmListPackagesCall{
				{
					args: []string{testNPMListCommand, testNPMGlobalFlag, testNPMDepthFlag, testNPMJSONFlag},
					err:  errNPMListPackages,
				},
			},
			want: npmListPackagesExpectation{err: errNPMListPackages},
		},
		{
			name: "rejects a non-zero list result without running outdated",
			calls: []npmListPackagesCall{
				{
					args:   []string{testNPMListCommand, testNPMGlobalFlag, testNPMDepthFlag, testNPMJSONFlag},
					result: testutil.FailureResult(1, "npm list failed"),
				},
			},
			want: npmListPackagesExpectation{wantErr: true},
		},
		{
			name: "rejects invalid list JSON without running outdated",
			calls: []npmListPackagesCall{
				{
					args:   []string{testNPMListCommand, testNPMGlobalFlag, testNPMDepthFlag, testNPMJSONFlag},
					result: testutil.SuccessResult("not valid json"),
				},
			},
			want: npmListPackagesExpectation{wantErr: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor, assertCalls := newNPMListPackagesExecutor(t, tt.calls)
			defer assertCalls()
			adapter := NewAdapter(executor, testutil.NewMockLogger())

			got, err := adapter.ListPackages(context.Background())
			assertNPMListPackages(t, got, err, tt.want)
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
				if command == npmCommand && len(args) == 1 && args[0] == doctorCommand {
					return &output.ExecutionResult{
						ExitCode: 0,
						Stdout:   "Everything is ok\n",
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
				if command == npmCommand && len(args) == 1 && args[0] == doctorCommand {
					return &output.ExecutionResult{
						ExitCode: 1,
						Stdout:   "Warning: some checks failed\n",
					}, nil
				}
				return nil, errors.New("unexpected command")
			},
			want:    manager.StatusDegraded,
			wantErr: false,
		},
		{
			name: "error state",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == npmCommand && len(args) == 1 && args[0] == doctorCommand {
					return &output.ExecutionResult{
						ExitCode: 1,
						Stderr:   "Error: critical failure\n",
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

func TestAdapter_GetVersion_NonZeroExitCode(t *testing.T) {
	execFunc := func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
		return &output.ExecutionResult{
			ExitCode: 1,
			Stderr:   "npm: command not found",
		}, nil
	}
	adapter := NewAdapter(testutil.NewMockExecutor(execFunc), testutil.NewMockLogger())

	_, err := adapter.GetVersion(context.Background())
	if err == nil {
		t.Error("Expected error for non-zero exit code")
	}
}

func TestAdapter_GetBinaryPath_Error(t *testing.T) {
	tests := []struct {
		name     string
		execFunc func(ctx context.Context, command string, args ...string) (*output.ExecutionResult, error)
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
			name: "non-zero exit code",
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return &output.ExecutionResult{
					ExitCode: 1,
					Stderr:   "npm not found",
				}, nil
			},
			wantErr: true,
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

func TestAdapter_GetConfigPath_Error(t *testing.T) {
	tests := []struct {
		name     string
		execFunc func(ctx context.Context, command string, args ...string) (*output.ExecutionResult, error)
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
			name: "non-zero exit code",
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return &output.ExecutionResult{
					ExitCode: 1,
					Stderr:   "npm config failed",
				}, nil
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

func TestAdapter_CheckHealth_ExecutorError(t *testing.T) {
	execFunc := func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
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

func TestAdapter_Detect_EmptyOutput(t *testing.T) {
	execFunc := func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
		return &output.ExecutionResult{
			ExitCode: 0,
			Stdout:   "   ", // whitespace only
		}, nil
	}
	adapter := NewAdapter(testutil.NewMockExecutor(execFunc), testutil.NewMockLogger())

	detected, err := adapter.Detect(context.Background())
	if err != nil {
		t.Errorf("Detect() error = %v", err)
	}

	if detected {
		t.Error("Detect() should return false for empty output")
	}
}
