package chocolatey

import (
	"context"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/testutil"
)

const (
	testVersionArg       = "--version"
	testNameCommandFails = "command fails"
	testPackageGit       = "git"
	testAllPackagesArg   = "all"
	testUpgradeArg       = "upgrade"
)

type chocolateyUpdateCall struct {
	command string
	args    []string
	result  *output.ExecutionResult
	err     error
}

type chocolateyUpdateCase struct {
	name    string
	opts    adapterm.UpdateOptions
	calls   []chocolateyUpdateCall
	want    *adapterm.UpdateResult
	wantErr error
}

func TestAdapter_Detect(t *testing.T) {
	tests := []struct {
		name     string
		execFunc testutil.ExecutorFunc
		want     bool
		wantErr  bool
	}{
		{
			name: "chocolatey installed",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == chocoCommand && len(args) == 1 && args[0] == testVersionArg {
					return testutil.SuccessResult("2.2.2\n"), nil
				}
				return nil, errors.New("unexpected command")
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "chocolatey not installed",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == chocoCommand && len(args) == 1 && args[0] == testVersionArg {
					return testutil.FailureResult(1, "choco: command not found"), errors.New("exit code 1")
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
		execFunc testutil.ExecutorFunc
		want     string
		wantErr  bool
	}{
		{
			name: "version with newline",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == chocoCommand && len(args) == 1 && args[0] == testVersionArg {
					return testutil.SuccessResult("2.2.2\n"), nil
				}
				return nil, errors.New("unexpected command")
			},
			want:    "2.2.2",
			wantErr: false,
		},
		{
			name: "version without newline",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == chocoCommand && len(args) == 1 && args[0] == testVersionArg {
					return testutil.SuccessResult("2.2.2"), nil
				}
				return nil, errors.New("unexpected command")
			},
			want:    "2.2.2",
			wantErr: false,
		},
		{
			name: testNameCommandFails,
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				return nil, errors.New("command failed")
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
		execFunc testutil.ExecutorFunc
		want     string
		wantErr  bool
	}{
		{
			name: "single path returned",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == "where" && len(args) == 1 && args[0] == chocoCommand {
					return testutil.SuccessResult("C:\\ProgramData\\chocolatey\\bin\\choco.exe\n"), nil
				}
				return nil, errors.New("unexpected command")
			},
			want:    "C:\\ProgramData\\chocolatey\\bin\\choco.exe",
			wantErr: false,
		},
		{
			name: "where fails",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				return testutil.FailureResult(1, "not found"), errors.New("not found")
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
	tests := []struct {
		name    string
		envVar  string
		wantErr bool
	}{
		{
			name:    "ChocolateyInstall env var set",
			envVar:  "C:\\CustomChocolatey",
			wantErr: false,
		},
		{
			name:    "default path",
			envVar:  "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVar != "" {
				t.Setenv("ChocolateyInstall", tt.envVar)
			} else {
				os.Unsetenv("ChocolateyInstall")
			}

			adapter := NewAdapter(testutil.NewMockExecutor(nil), testutil.NewMockLogger())

			got, err := adapter.GetConfigPath(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("GetConfigPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.envVar != "" {
				if got != tt.envVar {
					t.Errorf("GetConfigPath() = %v, want %v", got, tt.envVar)
				}
			} else {
				// Should contain "chocolatey" in the path
				if got == "" {
					t.Error("GetConfigPath() returned empty path")
				}
			}
		})
	}
}

func TestAdapter_ListPackages(t *testing.T) {
	listOutput := `7zip|23.01
git|2.43.0
nodejs|20.10.0
`

	execFunc := func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		if command == chocoCommand && len(args) == 2 && args[0] == "list" && args[1] == "-r" {
			return testutil.SuccessResult(listOutput), nil
		}
		return nil, errors.New("unexpected command")
	}

	adapter := NewAdapter(testutil.NewMockExecutor(execFunc), testutil.NewMockLogger())
	packages, err := adapter.ListPackages(context.Background())
	if err != nil {
		t.Errorf("ListPackages() error = %v", err)
		return
	}

	if len(packages) != 3 {
		t.Errorf("ListPackages() returned %d packages, want 3", len(packages))
		return
	}

	// Check first package
	if packages[0].Name != "7zip" {
		t.Errorf("First package name = %v, want 7zip", packages[0].Name)
	}
	if packages[0].CurrentVersion != "23.01" {
		t.Errorf("First package version = %v, want 23.01", packages[0].CurrentVersion)
	}
	if !packages[0].IsGlobal {
		t.Error("Package IsGlobal should be true for Chocolatey")
	}
}

func TestAdapter_ListPackages_Empty(t *testing.T) {
	listOutput := ``

	execFunc := func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		if command == chocoCommand && len(args) == 2 && args[0] == "list" && args[1] == "-r" {
			return testutil.SuccessResult(listOutput), nil
		}
		return nil, errors.New("unexpected command")
	}

	adapter := NewAdapter(testutil.NewMockExecutor(execFunc), testutil.NewMockLogger())
	packages, err := adapter.ListPackages(context.Background())
	if err != nil {
		t.Errorf("ListPackages() error = %v", err)
		return
	}

	if len(packages) != 0 {
		t.Errorf("ListPackages() returned %d packages, want 0", len(packages))
	}
}

func TestAdapter_CheckHealth(t *testing.T) {
	tests := []struct {
		name     string
		execFunc testutil.ExecutorFunc
		want     manager.Status
	}{
		{
			name: "healthy - no outdated",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == chocoCommand && args[0] == "outdated" {
					return testutil.SuccessResult(""), nil
				}
				return nil, errors.New("unexpected command")
			},
			want: manager.StatusHealthy,
		},
		{
			name: "degraded with outdated packages",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == chocoCommand && args[0] == "outdated" {
					return testutil.SuccessResult("git|2.42.0|2.43.0|false\n"), nil
				}
				return nil, errors.New("unexpected command")
			},
			want: manager.StatusDegraded,
		},
		{
			name: "error",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				return testutil.FailureResult(1, "outdated check error"), nil
			},
			want: manager.StatusError,
		},
		{
			name: testNameCommandFails,
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				return nil, errors.New("network error")
			},
			want: manager.StatusDegraded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter(testutil.NewMockExecutor(tt.execFunc), testutil.NewMockLogger())

			got, err := adapter.CheckHealth(context.Background())
			if err != nil {
				t.Errorf("CheckHealth() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("CheckHealth() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAdapter_Update(t *testing.T) {
	networkErr := errors.New("network error")
	tests := []chocolateyUpdateCase{
		{
			name: "dry run",
			opts: adapterm.UpdateOptions{DryRun: true},
			want: &adapterm.UpdateResult{
				Success:         true,
				Message:         "Dry-run: would upgrade all chocolatey packages",
				UpdatedPackages: []string{},
				FailedPackages:  []string{},
			},
		},
		{
			name: "fixed strategy",
			opts: adapterm.UpdateOptions{Strategy: adapterm.StrategyFixed},
			want: &adapterm.UpdateResult{
				Success:         true,
				Message:         "Strategy 'fixed': chocolatey upgrade skipped",
				UpdatedPackages: []string{},
				FailedPackages:  []string{},
			},
		},
		{
			name: "successful upgrade",
			opts: adapterm.UpdateOptions{},
			calls: []chocolateyUpdateCall{{
				command: chocoCommand,
				args:    []string{testUpgradeArg, testAllPackagesArg, "-y"},
				result:  testutil.SuccessResult("git has been upgraded to v2.43.0\n1 packages upgraded.\n"),
			}},
			want: &adapterm.UpdateResult{
				Success:         true,
				Message:         "1 packages updated successfully",
				UpdatedPackages: []string{testPackageGit},
				FailedPackages:  []string{},
			},
		},
		{
			name: "already up to date",
			opts: adapterm.UpdateOptions{},
			calls: []chocolateyUpdateCall{{
				command: chocoCommand,
				args:    []string{testUpgradeArg, testAllPackagesArg, "-y"},
				result:  testutil.SuccessResult("Chocolatey upgraded 0 packages.\n"),
			}},
			want: &adapterm.UpdateResult{
				Success:         true,
				Message:         "0 packages updated successfully",
				UpdatedPackages: []string{},
				FailedPackages:  []string{},
			},
		},
		{
			name: "upgrade fails",
			opts: adapterm.UpdateOptions{},
			calls: []chocolateyUpdateCall{{
				command: chocoCommand,
				args:    []string{testUpgradeArg, testAllPackagesArg, "-y"},
				err:     networkErr,
			}},
			want: &adapterm.UpdateResult{
				Success:         false,
				Message:         "chocolatey upgrade failed: network error",
				UpdatedPackages: []string{},
				FailedPackages:  []string{},
			},
			wantErr: networkErr,
		},
	}

	for i := range tests {
		testCase := &tests[i]
		t.Run(testCase.name, func(t *testing.T) {
			runChocolateyUpdateCase(t, testCase)
		})
	}
}

func runChocolateyUpdateCase(t *testing.T, testCase *chocolateyUpdateCase) {
	t.Helper()

	execFunc, assertCalls := newChocolateyUpdateExecutor(t, testCase.calls)
	adapter := NewAdapter(testutil.NewMockExecutor(execFunc), testutil.NewMockLogger())

	got, err := adapter.Update(context.Background(), testCase.opts)
	assertCalls()
	if !errors.Is(err, testCase.wantErr) {
		t.Errorf("Update() error = %v, want error matching %v", err, testCase.wantErr)
		return
	}
	assertChocolateyUpdateResult(t, got, testCase.want)
}

func newChocolateyUpdateExecutor(t *testing.T, calls []chocolateyUpdateCall) (execFunc testutil.ExecutorFunc, assertCalls func()) {
	t.Helper()

	callIndex := 0
	execFunc = func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		if callIndex == len(calls) {
			t.Errorf("unexpected executor call: %s %s", command, strings.Join(args, " "))
			return nil, errors.New("unexpected executor call")
		}

		want := calls[callIndex]
		callIndex++
		if command != want.command || !slices.Equal(args, want.args) {
			t.Errorf("executor call = %s %v, want %s %v", command, args, want.command, want.args)
		}
		return want.result, want.err
	}
	assertCalls = func() {
		t.Helper()
		if callIndex != len(calls) {
			t.Errorf("executor call count = %d, want %d", callIndex, len(calls))
		}
	}
	return execFunc, assertCalls
}

func assertChocolateyUpdateResult(t *testing.T, got, want *adapterm.UpdateResult) {
	t.Helper()

	if got == nil {
		t.Fatal("Update() returned nil result")
	}
	if got.Success != want.Success {
		t.Errorf("Update() Success = %v, want %v", got.Success, want.Success)
	}
	if got.Message != want.Message {
		t.Errorf("Update() Message = %v, want %v", got.Message, want.Message)
	}
	if !slices.Equal(got.UpdatedPackages, want.UpdatedPackages) {
		t.Errorf("Update() UpdatedPackages = %v, want %v", got.UpdatedPackages, want.UpdatedPackages)
	}
	if !slices.Equal(got.FailedPackages, want.FailedPackages) {
		t.Errorf("Update() FailedPackages = %v, want %v", got.FailedPackages, want.FailedPackages)
	}
}

func TestNewAdapter(t *testing.T) {
	executor := testutil.NewMockExecutor(nil)
	logger := testutil.NewMockLogger()

	adapter := NewAdapter(executor, logger)
	if adapter == nil {
		t.Error("NewAdapter() returned nil")
	}
	if adapter.executor == nil {
		t.Error("NewAdapter() executor is nil")
	}
	if adapter.logger == nil {
		t.Error("NewAdapter() logger is nil")
	}
}

func TestAdapter_Search(t *testing.T) {
	searchOutput := `git|2.43.0
git.install|2.43.0
github-desktop|3.3.0
`

	tests := []struct {
		name      string
		query     string
		execFunc  testutil.ExecutorFunc
		wantCount int
		wantErr   bool
		wantFirst string
	}{
		{
			name:  "machine readable results",
			query: testPackageGit,
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == chocoCommand && len(args) == 3 && args[0] == "search" && args[1] == testPackageGit && args[2] == "-r" {
					return testutil.SuccessResult(searchOutput), nil
				}
				return nil, errors.New("unexpected command")
			},
			wantCount: 3,
			wantFirst: testPackageGit,
		},
		{
			name:  "empty query",
			query: " ",
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				t.Fatal("executor should not be called")
				return nil, nil
			},
			wantErr: true,
		},
		{
			name:  testNameCommandFails,
			query: testPackageGit,
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return testutil.FailureResult(1, "search error"), errors.New("exit 1")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter(testutil.NewMockExecutor(tt.execFunc), testutil.NewMockLogger())
			packages, err := adapter.Search(context.Background(), tt.query)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Search() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if len(packages) != tt.wantCount {
				t.Fatalf("Search() count = %d, want %d", len(packages), tt.wantCount)
			}
			if packages[0].Name != tt.wantFirst {
				t.Errorf("Search() first = %q, want %q", packages[0].Name, tt.wantFirst)
			}
			if packages[0].Manager != manager.ManagerChocolatey {
				t.Errorf("Search() Manager = %q, want choco", packages[0].Manager)
			}
		})
	}
}

func TestAdapter_Install(t *testing.T) {
	t.Run("dry-run", func(t *testing.T) {
		adapter := NewAdapter(testutil.NewMockExecutor(func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
			return nil, errors.New("should not be called")
		}), testutil.NewMockLogger())
		if err := adapter.Install(context.Background(), testPackageGit, true); err != nil {
			t.Fatalf("Install dry-run: %v", err)
		}
	})

	t.Run("install success", func(t *testing.T) {
		var gotArgs []string
		adapter := NewAdapter(testutil.NewMockExecutor(func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
			if command != chocoCommand {
				return nil, errors.New("unexpected command")
			}
			gotArgs = args
			return testutil.SuccessResult("Chocolatey installed git"), nil
		}), testutil.NewMockLogger())
		if err := adapter.Install(context.Background(), testPackageGit, false); err != nil {
			t.Fatalf("Install: %v", err)
		}
		if len(gotArgs) != 3 || gotArgs[0] != "install" || gotArgs[1] != testPackageGit || gotArgs[2] != "-y" {
			t.Fatalf("args = %v", gotArgs)
		}
	})

	t.Run("empty id keeps install context", func(t *testing.T) {
		adapter := NewAdapter(testutil.NewMockExecutor(nil), testutil.NewMockLogger())
		err := adapter.Install(context.Background(), " ", false)
		if err == nil {
			t.Fatal("expected empty id error")
		}
		if got, want := err.Error(), "install chocolatey package: package id is required"; got != want {
			t.Errorf("error = %q, want %q", got, want)
		}
	})

	t.Run("elevation error wrapped", func(t *testing.T) {
		adapter := NewAdapter(testutil.NewMockExecutor(func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
			return testutil.FailureResult(1, "Access is denied. This operation requires elevation."), nil
		}), testutil.NewMockLogger())
		err := adapter.Install(context.Background(), testPackageGit, false)
		if err == nil {
			t.Fatal("expected elevation error")
		}
		if !strings.Contains(err.Error(), "Administrator") {
			t.Errorf("error should suggest admin: %v", err)
		}
		if !strings.Contains(err.Error(), "install chocolatey package "+testPackageGit) {
			t.Errorf("error should retain install context: %v", err)
		}
	})
}

func TestAdapter_Uninstall(t *testing.T) {
	t.Run("uninstall success", func(t *testing.T) {
		var gotArgs []string
		adapter := NewAdapter(testutil.NewMockExecutor(func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
			if command != chocoCommand {
				return nil, errors.New("unexpected command")
			}
			gotArgs = args
			return testutil.SuccessResult("Chocolatey uninstalled git"), nil
		}), testutil.NewMockLogger())
		if err := adapter.Uninstall(context.Background(), testPackageGit, false); err != nil {
			t.Fatalf("Uninstall: %v", err)
		}
		if len(gotArgs) != 3 || gotArgs[0] != "uninstall" || gotArgs[1] != testPackageGit || gotArgs[2] != "-y" {
			t.Fatalf("args = %v", gotArgs)
		}
	})

	t.Run("dry-run skips executor", func(t *testing.T) {
		adapter := NewAdapter(testutil.NewMockExecutor(func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
			return nil, errors.New("should not be called")
		}), testutil.NewMockLogger())
		if err := adapter.Uninstall(context.Background(), testPackageGit, true); err != nil {
			t.Fatalf("Uninstall dry-run: %v", err)
		}
	})

	t.Run("empty id keeps uninstall context", func(t *testing.T) {
		adapter := NewAdapter(testutil.NewMockExecutor(nil), testutil.NewMockLogger())
		err := adapter.Uninstall(context.Background(), "", false)
		if err == nil {
			t.Fatal("expected empty id error")
		}
		if got, want := err.Error(), "uninstall chocolatey package: package id is required"; got != want {
			t.Errorf("error = %q, want %q", got, want)
		}
	})
}

func TestWrapElevationError(t *testing.T) {
	if err := wrapElevationError(nil); err != nil {
		t.Fatal("nil should stay nil")
	}
	base := errors.New("permission denied while installing")
	wrapped := wrapElevationError(base)
	if !strings.Contains(wrapped.Error(), "Administrator") {
		t.Errorf("wrapped = %v", wrapped)
	}
	normal := wrapElevationError(errors.New("package not found"))
	if strings.Contains(normal.Error(), "Administrator") {
		t.Errorf("non-elevation should not wrap: %v", normal)
	}
}
