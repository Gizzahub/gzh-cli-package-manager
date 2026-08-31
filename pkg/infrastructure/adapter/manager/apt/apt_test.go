package apt

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
	testCommand     = "test"
	testFileFlag    = "-f"
	testAPTLockFile = "/var/lib/dpkg/lock-frontend"
	testVimPackage  = "vim"
)

type aptCommandResponse struct {
	command string
	args    []string
	result  *output.ExecutionResult
	err     error
}

func aptSequenceExecutor(t *testing.T, responses []aptCommandResponse) (execFunc testutil.ExecutorFunc, assertCalls func()) {
	t.Helper()
	index := 0
	return func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
			if index == len(responses) {
				t.Errorf("unexpected command: %s %v", command, args)
				return nil, errors.New("unexpected command")
			}

			response := responses[index]
			index++
			if command != response.command || !slices.Equal(args, response.args) {
				t.Errorf("command = %s %v, want %s %v", command, args, response.command, response.args)
			}
			return response.result, response.err
		}, func() {
			if index != len(responses) {
				t.Errorf("command count = %d, want %d", index, len(responses))
			}
		}
}

func testAPTPackage(name, currentVersion string) manager.Package {
	return manager.Package{
		Name:           name,
		CurrentVersion: currentVersion,
		IsGlobal:       true,
		UpdateType:     manager.UpdateNone,
	}
}

func testAPTUpdatablePackage(name, currentVersion, availableVersion string) manager.Package {
	pkg := testAPTPackage(name, currentVersion)
	pkg.AvailableVersion = availableVersion
	pkg.UpdateType = manager.UpdateMinor
	return pkg
}

func assertAPTPackageSet(t *testing.T, got []manager.Package, want map[string]manager.Package) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("ListPackages() count = %d, want %d", len(got), len(want))
	}

	seen := make(map[string]struct{}, len(got))
	for _, pkg := range got {
		if _, duplicate := seen[pkg.Name]; duplicate {
			t.Errorf("ListPackages() returned duplicate %q", pkg.Name)
			continue
		}
		seen[pkg.Name] = struct{}{}

		wantPkg, ok := want[pkg.Name]
		if !ok {
			t.Errorf("ListPackages() returned unexpected package %#v", pkg)
			continue
		}
		if pkg != wantPkg {
			t.Errorf("ListPackages()[%q] = %#v, want %#v", pkg.Name, pkg, wantPkg)
		}
	}
}

func TestAdapter_Detect(t *testing.T) {
	tests := []struct {
		name     string
		execFunc testutil.ExecutorFunc
		want     bool
		wantErr  bool
	}{
		{
			name: "apt installed",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == "which" && len(args) == 1 && args[0] == aptCommand {
					return testutil.SuccessResult("/usr/bin/apt\n"), nil
				}
				return testutil.FailureResult(1, ""), nil
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "apt not installed",
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return testutil.FailureResult(1, ""), nil
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
			name: "valid version output",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == aptCommand && len(args) == 1 && args[0] == "--version" {
					return testutil.SuccessResult("apt 2.4.11 (amd64)\n"), nil
				}
				return testutil.FailureResult(1, ""), nil
			},
			want:    "2.4.11",
			wantErr: false,
		},
		{
			name: "version with extra info",
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return testutil.SuccessResult("apt 2.0.6 (arm64)\nCompiled on Jul 15 2023\n"), nil
			},
			want:    "2.0.6",
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
		execFunc testutil.ExecutorFunc
		want     string
		wantErr  bool
	}{
		{
			name: "binary found",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == "which" && args[0] == aptCommand {
					return testutil.SuccessResult("/usr/bin/apt\n"), nil
				}
				return testutil.FailureResult(1, ""), nil
			},
			want:    "/usr/bin/apt",
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
	if got != "/etc/apt" {
		t.Errorf("GetConfigPath() = %v, want /etc/apt", got)
	}
}

func TestAdapter_ListPackages(t *testing.T) {
	installedError := errors.New("installed packages failed")
	upgradableError := errors.New("upgradable packages failed")
	tests := []struct {
		name      string
		responses []aptCommandResponse
		want      map[string]manager.Package
		wantErr   error
	}{
		{
			name: "merges installed packages and upgrades",
			responses: []aptCommandResponse{
				{
					command: aptCommand,
					args:    []string{listArg, installedFlag},
					result: testutil.SuccessResult(`Listing...
vim/jammy-updates,now 2:8.2.3995-1ubuntu2.12 amd64 [installed]
curl/jammy-security,now 7.81.0-1ubuntu1.15 amd64 [installed,automatic]
git/jammy-updates,now 1:2.34.1-1ubuntu1.10 amd64 [installed]
`),
				},
				{
					command: aptCommand,
					args:    []string{listArg, upgradableFlag},
					result: testutil.SuccessResult(`Listing...
curl/jammy-security 7.81.0-1ubuntu1.16 amd64 [upgradable from: 7.81.0-1ubuntu1.15]
git/jammy-updates 1:2.34.1-1ubuntu1.11 amd64 [upgradable from: 1:2.34.1-1ubuntu1.10]
					`),
				},
			},
			want: map[string]manager.Package{
				testVimPackage: testAPTPackage(testVimPackage, "2:8.2.3995-1ubuntu2.12"),
				"curl":         testAPTUpdatablePackage("curl", "7.81.0-1ubuntu1.15", "7.81.0-1ubuntu1.16"),
				"git":          testAPTUpdatablePackage("git", "1:2.34.1-1ubuntu1.10", "1:2.34.1-1ubuntu1.11"),
			},
		},
		{
			name: "no updates available",
			responses: []aptCommandResponse{
				{
					command: aptCommand,
					args:    []string{listArg, installedFlag},
					result: testutil.SuccessResult(`Listing...
vim/jammy-updates,now 2:8.2.3995-1ubuntu2.12 amd64 [installed]
					`),
				},
				{command: aptCommand, args: []string{listArg, upgradableFlag}, result: testutil.SuccessResult("Listing...\n")},
			},
			want: map[string]manager.Package{
				testVimPackage: testAPTPackage(testVimPackage, "2:8.2.3995-1ubuntu2.12"),
			},
		},
		{
			name: "parses relaxed rows and uses last versions",
			responses: []aptCommandResponse{
				{
					command: aptCommand,
					args:    []string{listArg, installedFlag},
					result: testutil.SuccessResult(`Listing...

vim/jammy,now 1 amd64 [installed]
incomplete
vim/jammy,now 2 amd64 [installed]
curl/jammy-security,now 3 amd64 [installed]
`),
				},
				{
					command: aptCommand,
					args:    []string{listArg, upgradableFlag},
					result: testutil.SuccessResult(`Listing...
vim/jammy 4 amd64 [upgradable from: 2]
vim/jammy 5 amd64 [upgradable from: 4]
orphan/jammy 6 amd64 [upgradable]
`),
				},
			},
			want: map[string]manager.Package{
				testVimPackage: testAPTUpdatablePackage(testVimPackage, "2", "5"),
				"curl":         testAPTPackage("curl", "3"),
			},
		},
		{
			name: "parses installed stdout even with nonzero exit",
			responses: []aptCommandResponse{
				{
					command: aptCommand,
					args:    []string{listArg, installedFlag},
					result:  &output.ExecutionResult{Stdout: "vim/jammy,now 2 amd64 [installed]\n", ExitCode: 100},
				},
				{command: aptCommand, args: []string{listArg, upgradableFlag}, result: testutil.SuccessResult("Listing...\n")},
			},
			want: map[string]manager.Package{
				testVimPackage: testAPTPackage(testVimPackage, "2"),
			},
		},
		{
			name: "ignores upgradable executor error",
			responses: []aptCommandResponse{
				{
					command: aptCommand,
					args:    []string{listArg, installedFlag},
					result:  testutil.SuccessResult("vim/jammy,now 2 amd64 [installed]\n"),
				},
				{command: aptCommand, args: []string{listArg, upgradableFlag}, err: upgradableError},
			},
			want: map[string]manager.Package{
				testVimPackage: testAPTPackage(testVimPackage, "2"),
			},
		},
		{
			name: "ignores upgradable stdout on nonzero exit",
			responses: []aptCommandResponse{
				{
					command: aptCommand,
					args:    []string{listArg, installedFlag},
					result:  testutil.SuccessResult("vim/jammy,now 2 amd64 [installed]\n"),
				},
				{
					command: aptCommand,
					args:    []string{listArg, upgradableFlag},
					result:  &output.ExecutionResult{Stdout: "vim/jammy 3 amd64 [upgradable]\n", ExitCode: 100},
				},
			},
			want: map[string]manager.Package{
				testVimPackage: testAPTPackage(testVimPackage, "2"),
			},
		},
		{
			name: "wraps installed executor error",
			responses: []aptCommandResponse{
				{command: aptCommand, args: []string{listArg, installedFlag}, err: installedError},
			},
			wantErr: installedError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			execFunc, assertCalls := aptSequenceExecutor(t, tt.responses)
			adapter := NewAdapter(testutil.NewMockExecutor(execFunc), testutil.NewMockLogger())
			packages, err := adapter.ListPackages(context.Background())
			assertCalls()

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("ListPackages() error = %v, want errors.Is(..., %v)", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ListPackages() unexpected error = %v", err)
			}
			assertAPTPackageSet(t, packages, tt.want)
		})
	}
}

func TestAdapter_CheckHealth(t *testing.T) {
	tests := []struct {
		name      string
		responses []aptCommandResponse
		want      manager.Status
	}{
		{
			name: "healthy system",
			responses: []aptCommandResponse{
				{command: aptGetCommand, args: []string{checkCommand}, result: testutil.SuccessResult("Reading package lists... Done\nBuilding dependency tree... Done\n")},
				{command: testCommand, args: []string{testFileFlag, testAPTLockFile}, result: testutil.FailureResult(1, "")},
			},
			want: manager.StatusHealthy,
		},
		{
			name: "degraded with lock file",
			responses: []aptCommandResponse{
				{command: aptGetCommand, args: []string{checkCommand}, result: testutil.SuccessResult("Reading package lists... Done\n")},
				{command: testCommand, args: []string{testFileFlag, testAPTLockFile}, result: testutil.SuccessResult("")},
			},
			want: manager.StatusDegraded,
		},
		{
			name: "degraded with broken packages",
			responses: []aptCommandResponse{
				{command: aptGetCommand, args: []string{checkCommand}, result: testutil.FailureResult(100, "E: Broken packages\n")},
			},
			want: manager.StatusDegraded,
		},
		{
			name: "healthy when lock is absent with shell exit error",
			responses: []aptCommandResponse{
				{command: aptGetCommand, args: []string{checkCommand}, result: testutil.SuccessResult("Reading package lists... Done\n")},
				{command: testCommand, args: []string{testFileFlag, testAPTLockFile}, result: testutil.FailureResult(1, ""), err: errors.New("exit status 1")},
			},
			want: manager.StatusHealthy,
		},
		{
			name: "degraded when lock probe fails",
			responses: []aptCommandResponse{
				{command: aptGetCommand, args: []string{checkCommand}, result: testutil.SuccessResult("Reading package lists... Done\n")},
				{command: testCommand, args: []string{testFileFlag, testAPTLockFile}, err: errors.New("lock check failed")},
			},
			want: manager.StatusDegraded,
		},
		{
			name: "degraded when package check executor fails",
			responses: []aptCommandResponse{
				{command: aptGetCommand, args: []string{checkCommand}, err: errors.New("execution failed")},
			},
			want: manager.StatusDegraded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			execFunc, assertCalls := aptSequenceExecutor(t, tt.responses)
			adapter := NewAdapter(testutil.NewMockExecutor(execFunc), testutil.NewMockLogger())
			got, err := adapter.CheckHealth(context.Background())
			assertCalls()

			if err != nil {
				t.Fatalf("CheckHealth() unexpected error = %v", err)
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
		execFunc testutil.ExecutorFunc
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
			name: "empty version output",
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return testutil.SuccessResult(""), nil
			},
			wantErr: true,
		},
		{
			name: "unexpected version format",
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return testutil.SuccessResult("apt"), nil
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

func TestAdapter_GetBinaryPath_Error(t *testing.T) {
	adapter := NewAdapter(testutil.NewMockExecutor(func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
		return nil, errors.New("execution failed")
	}), testutil.NewMockLogger())

	_, err := adapter.GetBinaryPath(context.Background())
	if err == nil {
		t.Error("Expected error for executor failure")
	}
}

func TestAdapter_Update(t *testing.T) {
	var calls int
	adapter := NewAdapter(testutil.NewMockExecutor(func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
		calls++
		return testutil.SuccessResult(""), nil
	}), testutil.NewMockLogger())
	result, err := adapter.Update(context.Background(), adapterm.UpdateOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Update dry-run unexpected error: %v", err)
	}
	if result == nil || !result.Success {
		t.Fatalf("Update dry-run expected success result, got %#v", result)
	}
	if calls != 0 {
		t.Fatalf("Update dry-run executed %d commands, want 0", calls)
	}
}

func TestAdapter_Detect_ExecutorError(t *testing.T) {
	adapter := NewAdapter(testutil.NewMockExecutor(func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
		return nil, errors.New("execution failed")
	}), testutil.NewMockLogger())

	detected, err := adapter.Detect(context.Background())
	if err != nil {
		t.Errorf("Detect() should not return error, got %v", err)
	}

	if detected {
		t.Error("Detect() should return false when executor fails")
	}
}
