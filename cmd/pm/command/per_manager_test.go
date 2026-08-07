package command

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
		case command == "winget" && len(args) == 1 && args[0] == "--version":
			return testutil.SuccessResult("v1.6.0\n"), nil
		case command == "winget" && len(args) > 0 && args[0] == "list":
			return testutil.SuccessResult(listOutput), nil
		default:
			return nil, errors.New("unexpected command: " + command + " " + strings.Join(args, " "))
		}
	})

	out, err := executePerManagerCmd(t, "winget", "list")
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
		case command == "winget" && len(args) == 1 && args[0] == "--version":
			return testutil.SuccessResult("v1.6.0\n"), nil
		case command == "winget" && len(args) >= 2 && args[0] == "search" && args[1] == "git":
			return testutil.SuccessResult(searchOutput), nil
		default:
			return nil, errors.New("unexpected command: " + command + " " + strings.Join(args, " "))
		}
	})

	out, err := executePerManagerCmd(t, "winget", "search", "git", "--output", "json")
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
		case command == "scoop" && len(args) == 1 && args[0] == "--version":
			return testutil.SuccessResult("v0.3.1\n"), nil
		case command == "scoop" && len(args) == 1 && args[0] == "list":
			return testutil.SuccessResult(listOutput), nil
		default:
			return nil, errors.New("unexpected command: " + command + " " + strings.Join(args, " "))
		}
	})

	out, err := executePerManagerCmd(t, "scoop", "list")
	if err != nil {
		t.Fatalf("scoop list failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "git") || !strings.Contains(out, "7zip") {
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
		case command == "scoop" && len(args) == 1 && args[0] == "--version":
			return testutil.SuccessResult("v0.3.1\n"), nil
		case command == "scoop" && len(args) == 2 && args[0] == "search" && args[1] == "git":
			return testutil.SuccessResult(searchOutput), nil
		default:
			return nil, errors.New("unexpected command: " + command + " " + strings.Join(args, " "))
		}
	})

	out, err := executePerManagerCmd(t, "scoop", "search", "git")
	if err != nil {
		t.Fatalf("scoop search failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "git") {
		t.Errorf("expected git in output, got: %s", out)
	}
}

func TestPerManager_ChocolateyList(t *testing.T) {
	listOutput := "git|2.43.0\nnodejs|20.10.0\n"

	installTestAdapters(t, func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		switch {
		case command == "choco" && len(args) == 1 && args[0] == "--version":
			return testutil.SuccessResult("2.2.2\n"), nil
		case command == "choco" && len(args) == 2 && args[0] == "list" && args[1] == "-r":
			return testutil.SuccessResult(listOutput), nil
		default:
			return nil, errors.New("unexpected command: " + command + " " + strings.Join(args, " "))
		}
	})

	out, err := executePerManagerCmd(t, "chocolatey", "list", "-o", "json")
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
		case command == "choco" && len(args) == 1 && args[0] == "--version":
			return testutil.SuccessResult("2.2.2\n"), nil
		case command == "choco" && len(args) == 3 && args[0] == "search" && args[1] == "git" && args[2] == "-r":
			return testutil.SuccessResult(searchOutput), nil
		default:
			return nil, errors.New("unexpected command: " + command + " " + strings.Join(args, " "))
		}
	})

	out, err := executePerManagerCmd(t, "chocolatey", "search", "git")
	if err != nil {
		t.Fatalf("chocolatey search failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "git") {
		t.Errorf("expected git in output, got: %s", out)
	}
}

func TestPerManager_NotDetected(t *testing.T) {
	installTestAdapters(t, func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		// Detect uses --version; return failure for all managers.
		if len(args) == 1 && args[0] == "--version" {
			return testutil.FailureResult(1, command+": not found"), errors.New("not found")
		}
		return nil, errors.New("unexpected command")
	})

	for _, name := range []string{"winget", "scoop", "chocolatey"} {
		t.Run(name, func(t *testing.T) {
			_, err := executePerManagerCmd(t, name, "list")
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
	_, err := executePerManagerCmd(t, "winget", "list")
	if err == nil {
		t.Fatal("expected error when adapters not initialized")
	}
	if !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("error = %v, want 'not initialized'", err)
	}
}

func TestPerManager_UnknownOutputFormat(t *testing.T) {
	installTestAdapters(t, func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		if command == "winget" && len(args) == 1 && args[0] == "--version" {
			return testutil.SuccessResult("v1.6.0\n"), nil
		}
		if command == "winget" && len(args) > 0 && args[0] == "list" {
			return testutil.SuccessResult(""), nil
		}
		return nil, errors.New("unexpected")
	})

	_, err := executePerManagerCmd(t, "winget", "list", "--output", "yaml")
	if err == nil {
		t.Fatal("expected unknown format error")
	}
	if !strings.Contains(err.Error(), "unknown output format") {
		t.Errorf("error = %v", err)
	}
}
