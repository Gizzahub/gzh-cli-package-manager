package command

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/chocolatey"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/scoop"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/testutil"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/winget"
	"github.com/spf13/cobra"
)

type failingWriter struct {
	err error
}

const packagesOutputContext = "write packages output"

func (w failingWriter) Write(_ []byte) (int, error) {
	return 0, w.err
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
		case command == "winget" && len(args) == 1 && args[0] == testVersionFlag:
			return testutil.SuccessResult("v1.6.0\n"), nil
		case command == "winget" && len(args) > 0 && args[0] == listCommand:
			return testutil.SuccessResult(listOutput), nil
		default:
			return nil, errors.New("unexpected command: " + command + " " + strings.Join(args, " "))
		}
	})

	out, err := executePerManagerCmd(t, "winget", listCommand)
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
		case command == "winget" && len(args) == 1 && args[0] == testVersionFlag:
			return testutil.SuccessResult("v1.6.0\n"), nil
		case command == "winget" && len(args) >= 2 && args[0] == testSearchCommand && args[1] == testGitPackageName:
			return testutil.SuccessResult(searchOutput), nil
		default:
			return nil, errors.New("unexpected command: " + command + " " + strings.Join(args, " "))
		}
	})

	out, err := executePerManagerCmd(t, "winget", testSearchCommand, testGitPackageName, "--output", "json")
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
		case command == "scoop" && len(args) == 1 && args[0] == testVersionFlag:
			return testutil.SuccessResult("v0.3.1\n"), nil
		case command == "scoop" && len(args) == 1 && args[0] == listCommand:
			return testutil.SuccessResult(listOutput), nil
		default:
			return nil, errors.New("unexpected command: " + command + " " + strings.Join(args, " "))
		}
	})

	out, err := executePerManagerCmd(t, "scoop", listCommand)
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
		case command == "scoop" && len(args) == 1 && args[0] == testVersionFlag:
			return testutil.SuccessResult("v0.3.1\n"), nil
		case command == "scoop" && len(args) == 2 && args[0] == testSearchCommand && args[1] == testGitPackageName:
			return testutil.SuccessResult(searchOutput), nil
		default:
			return nil, errors.New("unexpected command: " + command + " " + strings.Join(args, " "))
		}
	})

	out, err := executePerManagerCmd(t, "scoop", testSearchCommand, testGitPackageName)
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
		case command == "choco" && len(args) == 1 && args[0] == testVersionFlag:
			return testutil.SuccessResult("2.2.2\n"), nil
		case command == "choco" && len(args) == 2 && args[0] == listCommand && args[1] == "-r":
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
		case command == "choco" && len(args) == 1 && args[0] == testVersionFlag:
			return testutil.SuccessResult("2.2.2\n"), nil
		case command == "choco" && len(args) == 3 && args[0] == testSearchCommand && args[1] == testGitPackageName && args[2] == "-r":
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

	for _, name := range []string{"winget", "scoop", "chocolatey"} {
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
	_, err := executePerManagerCmd(t, "winget", listCommand)
	if err == nil {
		t.Fatal("expected error when adapters not initialized")
	}
	if !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("error = %v, want 'not initialized'", err)
	}
}

func TestPerManager_UnknownOutputFormat(t *testing.T) {
	installTestAdapters(t, func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		if command == "winget" && len(args) == 1 && args[0] == testVersionFlag {
			return testutil.SuccessResult("v1.6.0\n"), nil
		}
		if command == "winget" && len(args) > 0 && args[0] == listCommand {
			return testutil.SuccessResult(""), nil
		}
		return nil, errors.New("unexpected")
	})

	_, err := executePerManagerCmd(t, "winget", listCommand, "--output", "yaml")
	if err == nil {
		t.Fatal("expected unknown format error")
	}
	if !strings.Contains(err.Error(), "unknown output format") {
		t.Errorf("error = %v", err)
	}
}

func TestPerManager_WingetInstallDryRun(t *testing.T) {
	installTestAdapters(t, func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		if command == "winget" && len(args) == 1 && args[0] == testVersionFlag {
			return testutil.SuccessResult("v1.6.0\n"), nil
		}
		return nil, errors.New("unexpected install call on dry-run: " + strings.Join(args, " "))
	})

	out, err := executePerManagerCmd(t, "winget", "install", "Git.Git", "--dry-run")
	if err != nil {
		t.Fatalf("winget install dry-run: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Dry-run") || !strings.Contains(out, "Git.Git") {
		t.Errorf("output = %q", out)
	}
}

func TestPerManager_WingetUninstall(t *testing.T) {
	var uninstallCalled bool
	installTestAdapters(t, func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		switch {
		case command == "winget" && len(args) == 1 && args[0] == testVersionFlag:
			return testutil.SuccessResult("v1.6.0\n"), nil
		case command == "winget" && len(args) > 0 && args[0] == "uninstall":
			uninstallCalled = true
			return testutil.SuccessResult("ok"), nil
		default:
			return nil, errors.New("unexpected: " + strings.Join(args, " "))
		}
	})

	out, err := executePerManagerCmd(t, "winget", "uninstall", "Git.Git")
	if err != nil {
		t.Fatalf("winget uninstall: %v\n%s", err, out)
	}
	if !uninstallCalled {
		t.Fatal("expected uninstall native call")
	}
	if !strings.Contains(out, "Uninstalled") {
		t.Errorf("output = %q", out)
	}
}

func TestPerManager_WingetUpgradeAllDryRun(t *testing.T) {
	installTestAdapters(t, func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		if command == "winget" && len(args) == 1 && args[0] == testVersionFlag {
			return testutil.SuccessResult("v1.6.0\n"), nil
		}
		// Update dry-run does not call executor
		return nil, errors.New("unexpected: " + strings.Join(args, " "))
	})

	out, err := executePerManagerCmd(t, "winget", "upgrade", "--all", "--dry-run")
	if err != nil {
		t.Fatalf("winget upgrade: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Dry-run") {
		t.Errorf("output = %q", out)
	}
}

func TestPerManager_WingetSourceList(t *testing.T) {
	sourceOutput := `Name    Argument
---------------------------------------------------
winget  https://cdn.winget.microsoft.com/cache
`

	installTestAdapters(t, func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		switch {
		case command == "winget" && len(args) == 1 && args[0] == testVersionFlag:
			return testutil.SuccessResult("v1.6.0\n"), nil
		case command == "winget" && len(args) == 2 && args[0] == "source" && args[1] == "list":
			return testutil.SuccessResult(sourceOutput), nil
		default:
			return nil, errors.New("unexpected: " + strings.Join(args, " "))
		}
	})

	out, err := executePerManagerCmd(t, "winget", "source", "list")
	if err != nil {
		t.Fatalf("winget source list: %v\n%s", err, out)
	}
	if !strings.Contains(out, "winget") {
		t.Errorf("output = %q", out)
	}
}

func TestPerManager_ScoopInstallUninstall(t *testing.T) {
	installTestAdapters(t, func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		switch {
		case command == "scoop" && len(args) == 1 && args[0] == testVersionFlag:
			return testutil.SuccessResult("v0.3.1\n"), nil
		case command == "scoop" && len(args) == 2 && args[0] == "install" && args[1] == testGitPackageName:
			return testutil.SuccessResult("Installing 'git'"), nil
		case command == "scoop" && len(args) == 2 && args[0] == "uninstall" && args[1] == testGitPackageName:
			return testutil.SuccessResult("Uninstalling 'git'"), nil
		default:
			return nil, errors.New("unexpected: " + strings.Join(args, " "))
		}
	})

	out, err := executePerManagerCmd(t, "scoop", "install", testGitPackageName)
	if err != nil {
		t.Fatalf("scoop install: %v\n%s", err, out)
	}
	out, err = executePerManagerCmd(t, "scoop", "uninstall", testGitPackageName, "--dry-run")
	if err != nil {
		t.Fatalf("scoop uninstall dry-run: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Dry-run") {
		t.Errorf("output = %q", out)
	}
}

func TestPerManager_ScoopBucket(t *testing.T) {
	bucketOutput := `Name Source
---- ------
main https://github.com/ScoopInstaller/Main
`

	installTestAdapters(t, func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		switch {
		case command == "scoop" && len(args) == 1 && args[0] == testVersionFlag:
			return testutil.SuccessResult("v0.3.1\n"), nil
		case command == "scoop" && len(args) == 2 && args[0] == "bucket" && args[1] == "list":
			return testutil.SuccessResult(bucketOutput), nil
		case command == "scoop" && len(args) >= 3 && args[0] == "bucket" && args[1] == "add":
			return testutil.SuccessResult("ok"), nil
		case command == "scoop" && len(args) == 3 && args[0] == "bucket" && args[1] == "rm":
			return testutil.SuccessResult("ok"), nil
		default:
			return nil, errors.New("unexpected: " + strings.Join(args, " "))
		}
	})

	out, err := executePerManagerCmd(t, "scoop", "bucket", "list")
	if err != nil {
		t.Fatalf("bucket list: %v\n%s", err, out)
	}
	if !strings.Contains(out, "main") {
		t.Errorf("output = %q", out)
	}

	out, err = executePerManagerCmd(t, "scoop", "bucket", "add", "extras")
	if err != nil {
		t.Fatalf("bucket add: %v\n%s", err, out)
	}
	if !strings.Contains(out, "extras") {
		t.Errorf("output = %q", out)
	}

	out, err = executePerManagerCmd(t, "scoop", "bucket", "remove", "extras")
	if err != nil {
		t.Fatalf("bucket remove: %v\n%s", err, out)
	}
}

func TestPerManager_ChocolateyInstallElevation(t *testing.T) {
	installTestAdapters(t, func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		switch {
		case command == "choco" && len(args) == 1 && args[0] == testVersionFlag:
			return testutil.SuccessResult("2.2.2\n"), nil
		case command == "choco" && len(args) >= 1 && args[0] == "install":
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
		if command == "choco" && len(args) == 1 && args[0] == testVersionFlag {
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
