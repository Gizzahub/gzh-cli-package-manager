package asdf

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/testutil"
)

// Test-specific constants.
const (
	versionArg       = "version"
	testNodeJSPlugin = "nodejs"
	testPythonPlugin = "python"
)

// errMockExecution is a sentinel error for testing executor failures.
var errMockExecution = errors.New("mock execution error")

type asdfListPackagesCall struct {
	args   []string
	err    error
	result *output.ExecutionResult
}

type asdfListPackagesExpectation struct {
	err      error
	errText  string
	packages []manager.Package
}

func newASDFListPackagesExecutor(t *testing.T, calls []asdfListPackagesCall) (executor testutil.ExecutorFunc, verify func()) {
	t.Helper()
	callIndex := 0
	executor = func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		t.Helper()
		if callIndex == len(calls) {
			t.Fatalf("unexpected command %q %v", command, args)
		}
		call := calls[callIndex]
		callIndex++
		if command != asdfCommand || !slices.Equal(args, call.args) {
			t.Errorf("command = %q %v, want %q %v", command, args, asdfCommand, call.args)
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

func assertASDFListPackages(t *testing.T, packages []manager.Package, err error, want asdfListPackagesExpectation) {
	t.Helper()
	if want.err != nil {
		if !errors.Is(err, want.err) {
			t.Fatalf("ListPackages() error = %v, want errors.Is(..., %v)", err, want.err)
		}
		if !strings.Contains(err.Error(), want.errText) {
			t.Errorf("ListPackages() error = %q, want context %q", err, want.errText)
		}
		return
	}
	if err != nil {
		t.Fatalf("ListPackages() error = %v, want nil", err)
	}
	if packages == nil {
		t.Fatal("ListPackages() packages = nil")
	}
	if !slices.Equal(packages, want.packages) {
		t.Errorf("ListPackages() packages = %#v, want %#v", packages, want.packages)
	}
}

func testASDFPackage(plugin, version string, isGlobal bool) manager.Package {
	return manager.Package{
		Name:           plugin + "@" + version,
		CurrentVersion: version,
		Description:    "ASDF plugin: " + plugin,
		UpdateType:     manager.UpdateNone,
		IsGlobal:       isGlobal,
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
		name  string
		calls []asdfListPackagesCall
		want  asdfListPackagesExpectation
	}{
		{
			name: "multiple plugins with versions",
			calls: []asdfListPackagesCall{
				{args: []string{pluginArg, listArg}, result: testutil.SuccessResult("nodejs\npython\nruby\n")},
				{args: []string{listArg, testNodeJSPlugin}, result: testutil.SuccessResult(" 18.0.0\n*20.11.0\n 21.0.0\n")},
				{args: []string{listArg, testPythonPlugin}, result: testutil.SuccessResult("*3.11.7\n 3.12.0\n")},
				{args: []string{listArg, "ruby"}, result: testutil.SuccessResult("*3.2.2\n")},
			},
			want: asdfListPackagesExpectation{packages: []manager.Package{
				testASDFPackage(testNodeJSPlugin, "18.0.0", false),
				testASDFPackage(testNodeJSPlugin, "20.11.0", true),
				testASDFPackage(testNodeJSPlugin, "21.0.0", false),
				testASDFPackage(testPythonPlugin, "3.11.7", true),
				testASDFPackage(testPythonPlugin, "3.12.0", false),
				testASDFPackage("ruby", "3.2.2", true),
			}},
		},
		{
			name: "no plugins installed",
			calls: []asdfListPackagesCall{
				{args: []string{pluginArg, listArg}, result: testutil.SuccessResult("")},
			},
			want: asdfListPackagesExpectation{packages: []manager.Package{}},
		},
		{
			name: "plugin list error preserves cause",
			calls: []asdfListPackagesCall{
				{args: []string{pluginArg, listArg}, err: errMockExecution},
			},
			want: asdfListPackagesExpectation{err: errMockExecution, errText: "failed to list plugins"},
		},
		{
			name: "nonzero results still parse stdout",
			calls: []asdfListPackagesCall{
				{args: []string{pluginArg, listArg}, result: &output.ExecutionResult{ExitCode: 1, Stdout: testNodeJSPlugin + "\n", Stderr: "plugin list failed"}},
				{args: []string{listArg, testNodeJSPlugin}, result: &output.ExecutionResult{ExitCode: 1, Stdout: "*20.11.0\n", Stderr: "version list failed"}},
			},
			want: asdfListPackagesExpectation{packages: []manager.Package{
				testASDFPackage(testNodeJSPlugin, "20.11.0", true),
			}},
		},
		{
			name: "version list error skips plugin and continues",
			calls: []asdfListPackagesCall{
				{args: []string{pluginArg, listArg}, result: testutil.SuccessResult(testNodeJSPlugin + "\n" + testPythonPlugin + "\n")},
				{args: []string{listArg, testNodeJSPlugin}, err: errMockExecution},
				{args: []string{listArg, testPythonPlugin}, result: testutil.SuccessResult("*3.12.0\n")},
			},
			want: asdfListPackagesExpectation{packages: []manager.Package{
				testASDFPackage(testPythonPlugin, "3.12.0", true),
			}},
		},
		{
			name: "empty version output",
			calls: []asdfListPackagesCall{
				{args: []string{pluginArg, listArg}, result: testutil.SuccessResult(testNodeJSPlugin + "\n")},
				{args: []string{listArg, testNodeJSPlugin}, result: testutil.SuccessResult("")},
			},
			want: asdfListPackagesExpectation{packages: []manager.Package{}},
		},
		{
			name: "malformed nonempty version remains package",
			calls: []asdfListPackagesCall{
				{args: []string{pluginArg, listArg}, result: testutil.SuccessResult(testNodeJSPlugin + "\n")},
				{args: []string{listArg, testNodeJSPlugin}, result: testutil.SuccessResult("*\n")},
			},
			want: asdfListPackagesExpectation{packages: []manager.Package{
				testASDFPackage(testNodeJSPlugin, "", true),
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor, verify := newASDFListPackagesExecutor(t, tt.calls)
			packages, err := NewAdapter(testutil.NewMockExecutor(executor), testutil.NewMockLogger()).ListPackages(context.Background())
			verify()

			assertASDFListPackages(t, packages, err, tt.want)
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
