package command

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/chocolatey"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/scoop"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/testutil"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/winget"
)

type failingWriter struct {
	err error
}

const packagesOutputContext = "write packages output"

func (w failingWriter) Write(_ []byte) (int, error) {
	return 0, w.err
}

type perManagerListAdapterBase struct {
	detectErr error
	detected  bool
}

func (a *perManagerListAdapterBase) Detect(context.Context) (bool, error) {
	return a.detected, a.detectErr
}

func (*perManagerListAdapterBase) GetVersion(context.Context) (string, error) { return "", nil }

func (*perManagerListAdapterBase) GetBinaryPath(context.Context) (string, error) { return "", nil }

func (*perManagerListAdapterBase) GetConfigPath(context.Context) (string, error) { return "", nil }

func (*perManagerListAdapterBase) ListPackages(context.Context) ([]manager.Package, error) {
	return nil, nil
}

func (*perManagerListAdapterBase) CheckHealth(context.Context) (manager.Status, error) {
	return manager.StatusHealthy, nil
}

func (*perManagerListAdapterBase) Update(context.Context, adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
	return &adapterm.UpdateResult{}, nil
}

type sourceListTestAdapter struct {
	*perManagerListAdapterBase
	err error
}

func (a *sourceListTestAdapter) ListSources(context.Context) ([]adapterm.Source, error) {
	return nil, a.err
}

type bucketListTestAdapter struct {
	*perManagerListAdapterBase
	err error
}

func (a *bucketListTestAdapter) ListBuckets(context.Context) ([]adapterm.Bucket, error) {
	return nil, a.err
}

func (*bucketListTestAdapter) AddBucket(context.Context, string, string) error { return nil }

func (*bucketListTestAdapter) RemoveBucket(context.Context, string) error { return nil }

func assertRunWithDetectedAdapter(
	t *testing.T,
	adapters map[manager.ManagerID]adapterm.Adapter,
	spec perManagerSpec,
	wantError string,
	wantCause error,
) {
	t.Helper()
	SetManagerAdapters(adapters)
	t.Cleanup(func() { SetManagerAdapters(nil) })

	called := false
	err := runWithDetectedAdapter(context.Background(), spec, func(adapter adapterm.Adapter) error {
		called = true
		if adapter == nil {
			t.Error("callback adapter = nil")
		}
		return nil
	})

	if wantError == "" {
		if err != nil {
			t.Fatalf("run with detected adapter: %v", err)
		}
		if !called {
			t.Error("callback was not called")
		}
		return
	}
	if err == nil || err.Error() != wantError {
		t.Fatalf("error = %v, want %q", err, wantError)
	}
	if wantCause != nil && !errors.Is(err, wantCause) {
		t.Errorf("error = %v, want errors.Is(..., %v)", err, wantCause)
	}
	if called {
		t.Error("callback was called after failed preflight")
	}
}

// installTestAdapters wires real adapters backed by a mock executor for CLI tests.
func installTestAdapters(t *testing.T, execFunc testutil.ExecutorFunc) {
	t.Helper()
	executor := testutil.NewMockExecutor(execFunc)
	logger := testutil.NewMockLogger()
	SetManagerAdapters(map[manager.ManagerID]adapterm.Adapter{
		manager.ManagerWinget:     winget.NewAdapter(executor, logger),
		manager.ManagerScoop:      scoop.NewAdapter(executor, logger),
		manager.ManagerChocolatey: chocolatey.NewAdapter(executor, logger),
	})
	t.Cleanup(func() {
		SetManagerAdapters(nil)
	})
}

func executePerManagerCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()

	// Build a fresh command tree so package-level root flags stay isolated.
	cmd := &cobra.Command{Use: "gz-pm"}
	for _, spec := range perManagerSpecs {
		cmd.AddCommand(newPerManagerCmd(spec))
	}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return buf.String(), err
}

func TestPerManager_WingetList(t *testing.T) {
	listOutput := `Name       Id        Version Source
---------------------------------------
Git        Git.Git   2.43.0  winget
`

	installTestAdapters(t, func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		switch {
		case command == testWingetExecutable && len(args) == 1 && args[0] == testVersionFlag:
			return testutil.SuccessResult("v1.6.0\n"), nil
		case command == testWingetExecutable && len(args) > 0 && args[0] == listCommand:
			return testutil.SuccessResult(listOutput), nil
		default:
			return nil, errors.New("unexpected command: " + command + " " + strings.Join(args, " "))
		}
	})

	out, err := executePerManagerCmd(t, testWingetCLICommand, listCommand)
	if err != nil {
		t.Fatalf("winget list failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "Git") {
		t.Errorf("expected Git in output, got: %s", out)
	}
	if !strings.Contains(out, "winget list") {
		t.Errorf("expected header in output, got: %s", out)
	}
}

func TestPerManager_WingetSearch(t *testing.T) {
	searchOutput := `Name       Id        Version Source
---------------------------------------
Git        Git.Git   2.43.0  winget
`

	installTestAdapters(t, func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		switch {
		case command == testWingetExecutable && len(args) == 1 && args[0] == testVersionFlag:
			return testutil.SuccessResult("v1.6.0\n"), nil
		case command == testWingetExecutable && len(args) >= 2 && args[0] == testSearchCommand && args[1] == testGitPackageName:
			return testutil.SuccessResult(searchOutput), nil
		default:
			return nil, errors.New("unexpected command: " + command + " " + strings.Join(args, " "))
		}
	})

	out, err := executePerManagerCmd(t, testWingetCLICommand, testSearchCommand, testGitPackageName, "--output", "json")
	if err != nil {
		t.Fatalf("winget search failed: %v\noutput: %s", err, out)
	}

	var resp packagesResponse
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}
	if resp.Manager != "winget" || resp.Action != "search" {
		t.Errorf("unexpected response meta: %+v", resp)
	}
	if resp.Count < 1 {
		t.Errorf("expected packages, got count=%d", resp.Count)
	}
}

func TestPerManager_ScoopList(t *testing.T) {
	listOutput := `Name  Version Source Updated             Info
----  ------- ------ -------             ----
git   2.43.0  main   2024-01-01 00:00:00
7zip  23.01   main   2024-01-01 00:00:00
`

	installTestAdapters(t, func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		switch {
		case command == testScoopExecutable && len(args) == 1 && args[0] == testVersionFlag:
			return testutil.SuccessResult(testScoopVersionOutput), nil
		case command == testScoopExecutable && len(args) == 1 && args[0] == listCommand:
			return testutil.SuccessResult(listOutput), nil
		default:
			return nil, errors.New("unexpected command: " + command + " " + strings.Join(args, " "))
		}
	})

	out, err := executePerManagerCmd(t, testScoopCLICommand, listCommand)
	if err != nil {
		t.Fatalf("scoop list failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, testGitPackageName) || !strings.Contains(out, "7zip") {
		t.Errorf("expected packages in output, got: %s", out)
	}
}

func TestPerManager_ScoopSearch(t *testing.T) {
	searchOutput := `Name Version Source
---- ------- ------
git  2.43.0  main
`

	installTestAdapters(t, func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		switch {
		case command == testScoopExecutable && len(args) == 1 && args[0] == testVersionFlag:
			return testutil.SuccessResult(testScoopVersionOutput), nil
		case command == testScoopExecutable && len(args) == 2 && args[0] == testSearchCommand && args[1] == testGitPackageName:
			return testutil.SuccessResult(searchOutput), nil
		default:
			return nil, errors.New("unexpected command: " + command + " " + strings.Join(args, " "))
		}
	})

	out, err := executePerManagerCmd(t, testScoopCLICommand, testSearchCommand, testGitPackageName)
	if err != nil {
		t.Fatalf("scoop search failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, testGitPackageName) {
		t.Errorf("expected git in output, got: %s", out)
	}
}

func TestPerManager_ChocolateyList(t *testing.T) {
	listOutput := "git|2.43.0\nnodejs|20.10.0\n"

	installTestAdapters(t, func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		switch {
		case command == testChocoExecutable && len(args) == 1 && args[0] == testVersionFlag:
			return testutil.SuccessResult("2.2.2\n"), nil
		case command == testChocoExecutable && len(args) == 2 && args[0] == listCommand && args[1] == "-r":
			return testutil.SuccessResult(listOutput), nil
		default:
			return nil, errors.New("unexpected command: " + command + " " + strings.Join(args, " "))
		}
	})

	out, err := executePerManagerCmd(t, "chocolatey", listCommand, "-o", "json")
	if err != nil {
		t.Fatalf("chocolatey list failed: %v\noutput: %s", err, out)
	}

	var resp packagesResponse
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}
	if resp.Count != 2 {
		t.Errorf("count = %d, want 2", resp.Count)
	}
	if resp.Manager != "chocolatey" {
		t.Errorf("manager = %q, want chocolatey", resp.Manager)
	}
}

func TestPerManager_ChocolateySearch(t *testing.T) {
	searchOutput := "git|2.43.0\ngit.install|2.43.0\n"

	installTestAdapters(t, func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		switch {
		case command == testChocoExecutable && len(args) == 1 && args[0] == testVersionFlag:
			return testutil.SuccessResult("2.2.2\n"), nil
		case command == testChocoExecutable && len(args) == 3 && args[0] == testSearchCommand && args[1] == testGitPackageName && args[2] == "-r":
			return testutil.SuccessResult(searchOutput), nil
		default:
			return nil, errors.New("unexpected command: " + command + " " + strings.Join(args, " "))
		}
	})

	out, err := executePerManagerCmd(t, "chocolatey", testSearchCommand, testGitPackageName)
	if err != nil {
		t.Fatalf("chocolatey search failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, testGitPackageName) {
		t.Errorf("expected git in output, got: %s", out)
	}
}

func TestPerManager_NotDetected(t *testing.T) {
	installTestAdapters(t, func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		// Detect uses --version; return failure for all managers.
		if len(args) == 1 && args[0] == testVersionFlag {
			return testutil.FailureResult(1, command+": not found"), errors.New("not found")
		}
		return nil, errors.New("unexpected command")
	})

	for _, name := range []string{testWingetCLICommand, testScoopCLICommand, "chocolatey"} {
		t.Run(name, func(t *testing.T) {
			_, err := executePerManagerCmd(t, name, listCommand)
			if err == nil {
				t.Fatal("expected error when manager not detected")
			}
			if !strings.Contains(err.Error(), "not available") {
				t.Errorf("error = %v, want 'not available'", err)
			}
		})
	}
}

func TestPerManager_AdapterNotInitialized(t *testing.T) {
	SetManagerAdapters(nil)
	_, err := executePerManagerCmd(t, testWingetCLICommand, listCommand)
	if err == nil {
		t.Fatal("expected error when adapters not initialized")
	}
	if !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("error = %v, want 'not initialized'", err)
	}
}

func TestNewPerManagerPackageActionCmd(t *testing.T) {
	spec := perManagerSpec{Use: "test-manager"}
	tests := []struct {
		name       string
		newCommand func(perManagerSpec) *cobra.Command
		use        string
		short      string
		long       string
		dryRunHelp string
	}{
		{
			name:       perManagerInstallAction,
			newCommand: newPerManagerInstallCmd,
			use:        perManagerInstallAction + " <id>",
			short:      "Install a package via test-manager",
			long:       "Install a package by ID/name using test-manager. Use --dry-run to preview.",
			dryRunHelp: "Show what would be installed without making changes",
		},
		{
			name:       perManagerUninstallAction,
			newCommand: newPerManagerUninstallCmd,
			use:        perManagerUninstallAction + " <id>",
			short:      "Uninstall a package via test-manager",
			long:       "Uninstall a package by ID/name using test-manager. Use --dry-run to preview.",
			dryRunHelp: "Show what would be uninstalled without making changes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.newCommand(spec)
			if cmd.Use != tt.use || cmd.Short != tt.short || cmd.Long != tt.long {
				t.Errorf("command = {%q, %q, %q}, want {%q, %q, %q}", cmd.Use, cmd.Short, cmd.Long, tt.use, tt.short, tt.long)
			}
			if flag := cmd.Flags().Lookup("dry-run"); flag == nil || flag.Usage != tt.dryRunHelp {
				t.Errorf("dry-run flag = %#v, want usage %q", flag, tt.dryRunHelp)
			}
		})
	}
}

func TestPerManager_UnknownOutputFormat(t *testing.T) {
	installTestAdapters(t, func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		if command == testWingetExecutable && len(args) == 1 && args[0] == testVersionFlag {
			return testutil.SuccessResult("v1.6.0\n"), nil
		}
		if command == testWingetExecutable && len(args) > 0 && args[0] == listCommand {
			return testutil.SuccessResult(""), nil
		}
		return nil, errors.New("unexpected")
	})

	_, err := executePerManagerCmd(t, testWingetCLICommand, listCommand, "--output", "yaml")
	if err == nil {
		t.Fatal("expected unknown format error")
	}
	if !strings.Contains(err.Error(), "unknown output format") {
		t.Errorf("error = %v", err)
	}
}

func TestPerManager_WingetInstallDryRun(t *testing.T) {
	installTestAdapters(t, func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		if command == testWingetExecutable && len(args) == 1 && args[0] == testVersionFlag {
			return testutil.SuccessResult("v1.6.0\n"), nil
		}
		return nil, errors.New("unexpected install call on dry-run: " + strings.Join(args, " "))
	})

	out, err := executePerManagerCmd(t, testWingetCLICommand, "install", "Git.Git", "--dry-run")
	if err != nil {
		t.Fatalf("winget install dry-run: %v\n%s", err, out)
	}
	if want := "Dry-run: would install \"Git.Git\" via winget\n"; out != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}

func TestPerManager_WingetUninstall(t *testing.T) {
	var uninstallCalled bool
	installTestAdapters(t, func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		switch {
		case command == testWingetExecutable && len(args) == 1 && args[0] == testVersionFlag:
			return testutil.SuccessResult("v1.6.0\n"), nil
		case command == testWingetExecutable && len(args) > 0 && args[0] == "uninstall":
			uninstallCalled = true
			return testutil.SuccessResult("ok"), nil
		default:
			return nil, errors.New("unexpected: " + strings.Join(args, " "))
		}
	})

	out, err := executePerManagerCmd(t, testWingetCLICommand, "uninstall", "Git.Git")
	if err != nil {
		t.Fatalf("winget uninstall: %v\n%s", err, out)
	}
	if !uninstallCalled {
		t.Fatal("expected uninstall native call")
	}
	if want := "Uninstalled \"Git.Git\" via winget\n"; out != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}

func TestPerManager_PackageActionErrorsPreserveCause(t *testing.T) {
	sentinel := errors.New("native package action failed")
	tests := []struct {
		action string
	}{
		{action: perManagerInstallAction},
		{action: perManagerUninstallAction},
	}

	for _, tt := range tests {
		t.Run(tt.action, func(t *testing.T) {
			installTestAdapters(t, func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == testWingetExecutable && len(args) == 1 && args[0] == testVersionFlag {
					return testutil.SuccessResult("v1.6.0\n"), nil
				}
				if command == testWingetExecutable && len(args) > 0 && args[0] == tt.action {
					return nil, sentinel
				}
				return nil, errors.New("unexpected: " + strings.Join(args, " "))
			})

			_, err := executePerManagerCmd(t, testWingetCLICommand, tt.action, "Git.Git")
			if !errors.Is(err, sentinel) {
				t.Errorf("error = %v, want cause %v", err, sentinel)
			}
			if want := "winget " + tt.action + " failed"; !strings.Contains(err.Error(), want) {
				t.Errorf("error = %q, want context %q", err, want)
			}
		})
	}
}

func TestPerManager_WingetUpgradeAllDryRun(t *testing.T) {
	installTestAdapters(t, func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		if command == testWingetExecutable && len(args) == 1 && args[0] == testVersionFlag {
			return testutil.SuccessResult("v1.6.0\n"), nil
		}
		// Update dry-run does not call executor
		return nil, errors.New("unexpected: " + strings.Join(args, " "))
	})

	out, err := executePerManagerCmd(t, testWingetCLICommand, "upgrade", "--all", "--dry-run")
	if err != nil {
		t.Fatalf("winget upgrade: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Dry-run") {
		t.Errorf("output = %q", out)
	}
}

func TestRunWithDetectedAdapterPreservesPreflightContract(t *testing.T) {
	detectErr := errors.New("detect failed")
	wingetSpec := perManagerSpec{ID: manager.ManagerWinget, Use: testWingetCLICommand}
	tests := []struct {
		name      string
		adapters  map[manager.ManagerID]adapterm.Adapter
		wantError string
		wantCause error
	}{
		{
			name:      "adapters not initialized",
			wantError: "winget: adapters not initialized",
		},
		{
			name:      "adapter not registered",
			adapters:  map[manager.ManagerID]adapterm.Adapter{},
			wantError: "winget: adapter not registered",
		},
		{
			name: "detect error",
			adapters: map[manager.ManagerID]adapterm.Adapter{
				manager.ManagerWinget: &perManagerListAdapterBase{detectErr: detectErr},
			},
			wantError: "winget: detect failed: detect failed",
			wantCause: detectErr,
		},
		{
			name: "not detected",
			adapters: map[manager.ManagerID]adapterm.Adapter{
				manager.ManagerWinget: &perManagerListAdapterBase{},
			},
			wantError: "winget is not available on this system (not installed or not in PATH)",
		},
		{
			name: "detected",
			adapters: map[manager.ManagerID]adapterm.Adapter{
				manager.ManagerWinget: &perManagerListAdapterBase{detected: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertRunWithDetectedAdapter(t, tt.adapters, wingetSpec, tt.wantError, tt.wantCause)
		})
	}
}

func TestPerManagerNamedCollectionListsPreserveErrorContracts(t *testing.T) {
	listErr := errors.New("list unavailable")
	wingetSpec := perManagerSpec{ID: manager.ManagerWinget, Use: testWingetCLICommand}
	scoopSpec := perManagerSpec{ID: manager.ManagerScoop, Use: testScoopCLICommand}
	tests := []struct {
		name      string
		spec      perManagerSpec
		adapter   adapterm.Adapter
		run       func(context.Context, perManagerSpec, string, io.Writer) error
		wantError string
		wantCause error
	}{
		{
			name:      "source unsupported",
			spec:      wingetSpec,
			adapter:   &perManagerListAdapterBase{detected: true},
			run:       runWingetSourceList,
			wantError: "winget does not support source list",
		},
		{
			name: "source list failure",
			spec: wingetSpec,
			adapter: &sourceListTestAdapter{
				perManagerListAdapterBase: &perManagerListAdapterBase{detected: true},
				err:                       listErr,
			},
			run:       runWingetSourceList,
			wantError: "winget source list failed: list unavailable",
			wantCause: listErr,
		},
		{
			name:      "bucket unsupported",
			spec:      scoopSpec,
			adapter:   &perManagerListAdapterBase{detected: true},
			run:       runScoopBucketList,
			wantError: "scoop does not support bucket management",
		},
		{
			name: "bucket list failure",
			spec: scoopSpec,
			adapter: &bucketListTestAdapter{
				perManagerListAdapterBase: &perManagerListAdapterBase{detected: true},
				err:                       listErr,
			},
			run:       runScoopBucketList,
			wantError: "scoop bucket list failed: list unavailable",
			wantCause: listErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetManagerAdapters(map[manager.ManagerID]adapterm.Adapter{tt.spec.ID: tt.adapter})
			t.Cleanup(func() { SetManagerAdapters(nil) })

			err := tt.run(context.Background(), tt.spec, outputFormatText, io.Discard)
			if err == nil || err.Error() != tt.wantError {
				t.Fatalf("error = %v, want %q", err, tt.wantError)
			}
			if tt.wantCause != nil && !errors.Is(err, tt.wantCause) {
				t.Errorf("error = %v, want errors.Is(..., %v)", err, tt.wantCause)
			}
		})
	}
}

func TestPerManager_WingetSourceList(t *testing.T) {
	sourceOutput := `Name    Argument
---------------------------------------------------
winget  https://cdn.winget.microsoft.com/cache
`

	installTestAdapters(t, func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		switch {
		case command == testWingetExecutable && len(args) == 1 && args[0] == testVersionFlag:
			return testutil.SuccessResult("v1.6.0\n"), nil
		case command == testWingetExecutable && len(args) == 2 && args[0] == "source" && args[1] == "list":
			return testutil.SuccessResult(sourceOutput), nil
		default:
			return nil, errors.New("unexpected: " + strings.Join(args, " "))
		}
	})

	out, err := executePerManagerCmd(t, testWingetCLICommand, "source", "list")
	if err != nil {
		t.Fatalf("winget source list: %v\n%s", err, out)
	}
	const want = "winget sources — 1\n  winget  https://cdn.winget.microsoft.com/cache\n"
	if out != want {
		t.Errorf("source list output = %q, want %q", out, want)
	}
}

func TestPerManager_ScoopInstall(t *testing.T) {
	installTestAdapters(t, func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		switch {
		case command == testScoopExecutable && len(args) == 1 && args[0] == testVersionFlag:
			return testutil.SuccessResult(testScoopVersionOutput), nil
		case command == testScoopExecutable && len(args) == 2 && args[0] == "install" && args[1] == testGitPackageName:
			return testutil.SuccessResult("Installing 'git'"), nil
		default:
			return nil, errors.New("unexpected: " + strings.Join(args, " "))
		}
	})

	out, err := executePerManagerCmd(t, testScoopCLICommand, "install", testGitPackageName)
	if err != nil {
		t.Fatalf("scoop install: %v\n%s", err, out)
	}
	if want := "Installed \"git\" via scoop\n"; out != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}

func TestPerManager_ScoopUninstallDryRun(t *testing.T) {
	installTestAdapters(t, func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		if command == testScoopExecutable && len(args) == 1 && args[0] == testVersionFlag {
			return testutil.SuccessResult(testScoopVersionOutput), nil
		}
		return nil, errors.New("unexpected: " + strings.Join(args, " "))
	})

	out, err := executePerManagerCmd(t, testScoopCLICommand, "uninstall", testGitPackageName, "--dry-run")
	if err != nil {
		t.Fatalf("scoop uninstall dry-run: %v\n%s", err, out)
	}
	if want := "Dry-run: would uninstall \"git\" via scoop\n"; out != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}

func TestPerManager_ScoopBucketList(t *testing.T) {
	bucketOutput := `Name Source
---- ------
main https://github.com/ScoopInstaller/Main
`

	installTestAdapters(t, func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		switch {
		case command == testScoopExecutable && len(args) == 1 && args[0] == testVersionFlag:
			return testutil.SuccessResult(testScoopVersionOutput), nil
		case command == testScoopExecutable && len(args) == 2 && args[0] == testBucketCommand && args[1] == listCommand:
			return testutil.SuccessResult(bucketOutput), nil
		default:
			return nil, errors.New("unexpected: " + strings.Join(args, " "))
		}
	})

	out, err := executePerManagerCmd(t, testScoopCLICommand, testBucketCommand, listCommand)
	if err != nil {
		t.Fatalf("bucket list: %v\n%s", err, out)
	}
	const want = "scoop buckets — 1\n  main  https://github.com/ScoopInstaller/Main\n"
	if out != want {
		t.Errorf("bucket list output = %q, want %q", out, want)
	}
}

func TestPerManager_ScoopBucketAdd(t *testing.T) {
	installTestAdapters(t, func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		switch {
		case command == testScoopExecutable && len(args) == 1 && args[0] == testVersionFlag:
			return testutil.SuccessResult(testScoopVersionOutput), nil
		case command == testScoopExecutable && len(args) == 3 && args[0] == testBucketCommand && args[1] == "add" && args[2] == testExtrasBucketName:
			return testutil.SuccessResult("ok"), nil
		default:
			return nil, errors.New("unexpected: " + strings.Join(args, " "))
		}
	})

	out, err := executePerManagerCmd(t, testScoopCLICommand, testBucketCommand, "add", testExtrasBucketName)
	if err != nil {
		t.Fatalf("bucket add: %v\n%s", err, out)
	}
	if want := "Added scoop bucket \"extras\"\n"; out != want {
		t.Errorf("bucket add output = %q, want %q", out, want)
	}
}

func TestPerManager_ScoopBucketRemove(t *testing.T) {
	installTestAdapters(t, func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		switch {
		case command == testScoopExecutable && len(args) == 1 && args[0] == testVersionFlag:
			return testutil.SuccessResult(testScoopVersionOutput), nil
		case command == testScoopExecutable && len(args) == 3 && args[0] == testBucketCommand && args[1] == "rm" && args[2] == testExtrasBucketName:
			return testutil.SuccessResult("ok"), nil
		default:
			return nil, errors.New("unexpected: " + strings.Join(args, " "))
		}
	})

	out, err := executePerManagerCmd(t, testScoopCLICommand, testBucketCommand, "remove", testExtrasBucketName)
	if err != nil {
		t.Fatalf("bucket remove: %v\n%s", err, out)
	}
	if want := "Removed scoop bucket \"extras\"\n"; out != want {
		t.Errorf("bucket remove output = %q, want %q", out, want)
	}
}

func TestPerManager_ChocolateyInstallElevation(t *testing.T) {
	installTestAdapters(t, func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		switch {
		case command == testChocoExecutable && len(args) == 1 && args[0] == testVersionFlag:
			return testutil.SuccessResult("2.2.2\n"), nil
		case command == testChocoExecutable && len(args) >= 1 && args[0] == "install":
			return testutil.FailureResult(1, "Access is denied. Requires elevation."), nil
		default:
			return nil, errors.New("unexpected")
		}
	})

	_, err := executePerManagerCmd(t, "chocolatey", "install", testGitPackageName)
	if err == nil {
		t.Fatal("expected elevation error")
	}
	if !strings.Contains(err.Error(), "Administrator") {
		t.Errorf("error = %v", err)
	}
}

func TestPerManager_ChocolateyUpgradeDryRun(t *testing.T) {
	installTestAdapters(t, func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		if command == testChocoExecutable && len(args) == 1 && args[0] == testVersionFlag {
			return testutil.SuccessResult("2.2.2\n"), nil
		}
		return nil, errors.New("unexpected on dry-run")
	})

	out, err := executePerManagerCmd(t, "chocolatey", "upgrade", "--all", "--dry-run")
	if err != nil {
		t.Fatalf("chocolatey upgrade: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Dry-run") {
		t.Errorf("output = %q", out)
	}
}

func TestPerManagerOutputWriteErrorsPreserveCauseAndContext(t *testing.T) {
	writerErr := errors.New("writer failed")
	tests := []struct {
		write   func(io.Writer) error
		name    string
		context string
	}{
		{
			name:    "sources JSON",
			context: "write sources output",
			write: func(out io.Writer) error {
				return writeSources(out, "json", []adapterm.Source{{Name: "winget"}})
			},
		},
		{
			name:    "sources empty text",
			context: "write sources output",
			write: func(out io.Writer) error {
				return writeSources(out, "text", nil)
			},
		},
		{
			name:    "buckets JSON",
			context: "write buckets output",
			write: func(out io.Writer) error {
				return writeBuckets(out, "json", []adapterm.Bucket{{Name: "main"}})
			},
		},
		{
			name:    "buckets empty text",
			context: "write buckets output",
			write: func(out io.Writer) error {
				return writeBuckets(out, "text", nil)
			},
		},
		{
			name:    "packages JSON",
			context: packagesOutputContext,
			write: func(out io.Writer) error {
				return writePackages(out, "json", "winget", "list", []manager.Package{{Name: testGitPackageName}})
			},
		},
		{
			name:    "packages empty text",
			context: packagesOutputContext,
			write: func(out io.Writer) error {
				return writePackages(out, "text", "winget", "list", nil)
			},
		},
		{
			name:    "packages item text",
			context: packagesOutputContext,
			write: func(out io.Writer) error {
				return writePackages(out, "text", "winget", "list", []manager.Package{{Name: testGitPackageName}})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.write(failingWriter{err: writerErr})
			if !errors.Is(err, writerErr) {
				t.Fatalf("error = %v, want errors.Is(..., writerErr)", err)
			}
			if !strings.Contains(err.Error(), tt.context) {
				t.Errorf("error = %q, want context %q", err, tt.context)
			}
		})
	}
}

func TestPerManagerNamedCollectionWritersExactOutput(t *testing.T) {
	sources := []adapterm.Source{
		{Name: testWingetCLICommand, Arg: testWingetSourceURL},
		{Name: testScoopCLICommand},
	}
	buckets := []adapterm.Bucket{
		{Name: testExtrasBucketName, Source: testScoopBucketURL},
		{Name: testWingetCLICommand},
	}
	tests := []struct {
		name  string
		write func(io.Writer) error
		want  string
	}{
		{
			name: "sources JSON",
			write: func(out io.Writer) error {
				return writeSources(out, outputFormatJSON, sources)
			},
			want: "{\n  \"count\": 2,\n  \"sources\": [\n    {\n      \"name\": \"" + testWingetCLICommand + "\",\n      \"arg\": \"" + testWingetSourceURL + "\"\n    },\n    {\n      \"name\": \"" + testScoopCLICommand + "\"\n    }\n  ]\n}\n",
		},
		{
			name: "sources text",
			write: func(out io.Writer) error {
				return writeSources(out, outputFormatText, sources)
			},
			want: "winget sources — 2\n  " + testWingetCLICommand + "  " + testWingetSourceURL + "\n  " + testScoopCLICommand + "\n",
		},
		{
			name: "sources empty default format",
			write: func(out io.Writer) error {
				return writeSources(out, "", nil)
			},
			want: "No sources configured.\n",
		},
		{
			name: "buckets JSON",
			write: func(out io.Writer) error {
				return writeBuckets(out, outputFormatJSON, buckets)
			},
			want: "{\n  \"count\": 2,\n  \"buckets\": [\n    {\n      \"name\": \"" + testExtrasBucketName + "\",\n      \"source\": \"" + testScoopBucketURL + "\"\n    },\n    {\n      \"name\": \"" + testWingetCLICommand + "\"\n    }\n  ]\n}\n",
		},
		{
			name: "buckets text",
			write: func(out io.Writer) error {
				return writeBuckets(out, outputFormatText, buckets)
			},
			want: "scoop buckets — 2\n  " + testExtrasBucketName + "  " + testScoopBucketURL + "\n  " + testWingetCLICommand + "\n",
		},
		{
			name: "buckets empty text",
			write: func(out io.Writer) error {
				return writeBuckets(out, outputFormatText, nil)
			},
			want: "No buckets configured.\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := tt.write(&out); err != nil {
				t.Fatalf("write collection: %v", err)
			}
			if got := out.String(); got != tt.want {
				t.Errorf("output = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPerManagerNamedCollectionWritersPreserveTextWriteBehavior(t *testing.T) {
	if err := writeSources(failingWriter{err: errors.New("source writer failed")}, outputFormatText, []adapterm.Source{{Name: testWingetCLICommand}}); err != nil {
		t.Errorf("write sources = %v, want nil", err)
	}
	if err := writeBuckets(failingWriter{err: errors.New("bucket writer failed")}, outputFormatText, []adapterm.Bucket{{Name: testExtrasBucketName}}); err != nil {
		t.Errorf("write buckets = %v, want nil", err)
	}
}

func TestPerManagerNamedCollectionWritersRejectUnknownFormat(t *testing.T) {
	err := writeSources(io.Discard, "yaml", nil)
	if want := "unknown output format \"yaml\" (supported: text, json)"; err == nil || err.Error() != want {
		t.Errorf("error = %v, want %q", err, want)
	}
}
