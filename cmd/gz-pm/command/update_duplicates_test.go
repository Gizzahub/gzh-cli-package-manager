package command

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/dto"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
)

func TestUpdateCheckDuplicates(t *testing.T) {
	t.Run("report-printed", testUpdateReportPrinted)
	t.Run("does-not-fail-on-warning", testUpdateDoesNotFailOnWarning)
	t.Run("collection-does-not-fail", testUpdateCollectionDoesNotFail)
	t.Run("flag-off", testUpdateDuplicateFlagOff)
	t.Run("json-report", testUpdateDuplicateJSON)
}

func testUpdateReportPrinted(t *testing.T) {
	home, search, _, adapters := warningAdapters(t)
	recorder := &recordingUpdate{}
	useUpdateDuplicateFixtures(t, home, search, adapters, recorder)
	out, err := executeUpdate(t, "--all", "--check-duplicates")
	if err != nil {
		t.Fatalf("report-printed exit: %v", err)
	}
	assertFacts(t, out, "command:", "kind:", "warning", "claude")
	assertNoRemovalAdvice(t, out)
	if recorder.calls != 1 || !recorder.sawCheck {
		t.Fatalf("report-printed update calls=%d check=%v", recorder.calls, recorder.sawCheck)
	}
}

func testUpdateDoesNotFailOnWarning(t *testing.T) {
	home, search, _, adapters := warningAdapters(t)
	recorder := &recordingUpdate{}
	useUpdateDuplicateFixtures(t, home, search, adapters, recorder)
	if _, err := executeUpdate(t, "--all", "--check-duplicates"); err != nil {
		t.Fatalf("does-not-fail-on-warning: %v", err)
	}
	if recorder.calls != 1 {
		t.Fatalf("does-not-fail-on-warning calls=%d", recorder.calls)
	}
}

func testUpdateCollectionDoesNotFail(t *testing.T) {
	home := t.TempDir()
	search := []string{t.TempDir()}
	failing := &inventoryAdapter{err: errors.New("collector failed")}
	recorder := &recordingUpdate{}
	useUpdateDuplicateFixtures(t, home, search, map[manager.ManagerID]adapterm.Adapter{
		"fail": failing,
	}, recorder)
	out, err := executeUpdate(t, "--all", "--check-duplicates")
	if err != nil {
		t.Fatalf("collection-does-not-fail: %v", err)
	}
	if recorder.calls != 1 {
		t.Fatalf("collection-does-not-fail calls=%d", recorder.calls)
	}
	if !strings.Contains(out, "collect commands") {
		t.Fatalf("collection-does-not-fail missing error\n%s", out)
	}
}

func testUpdateDuplicateFlagOff(t *testing.T) {
	home := t.TempDir()
	search := []string{t.TempDir()}
	inventory := &inventoryAdapter{}
	recorder := &recordingUpdate{}
	useUpdateDuplicateFixtures(t, home, search, map[manager.ManagerID]adapterm.Adapter{
		manager.ManagerHomebrew: inventory,
	}, recorder)
	out, err := executeUpdate(t, "--all")
	if err != nil {
		t.Fatalf("flag-off exit: %v", err)
	}
	if inventory.calls != 0 || recorder.calls != 1 || recorder.sawCheck {
		t.Fatalf("flag-off inventory=%d update=%d check=%v", inventory.calls, recorder.calls, recorder.sawCheck)
	}
	if strings.Contains(out, "command:") {
		t.Fatalf("flag-off printed a report\n%s", out)
	}
}

func testUpdateDuplicateJSON(t *testing.T) {
	home, search, _, adapters := warningAdapters(t)
	recorder := &recordingUpdate{}
	useUpdateDuplicateFixtures(t, home, search, adapters, recorder)
	out, err := executeUpdate(t, "--all", "--check-duplicates", "--output", "json")
	if err != nil {
		t.Fatalf("json-report exit: %v", err)
	}
	assertFacts(t, out, `"Command"`, `"Kind"`, "warning")
	if recorder.calls != 1 {
		t.Fatalf("json-report calls=%d", recorder.calls)
	}
}

type recordingUpdate struct {
	calls    int
	sawCheck bool
}

func (r *recordingUpdate) Update(_ context.Context, req *dto.UpdateRequest) (*dto.UpdateResponse, error) {
	r.calls++
	if req != nil {
		r.sawCheck = req.CheckDuplicates
	}
	return &dto.UpdateResponse{Summary: &dto.UpdateSummary{}}, nil
}

func useUpdateDuplicateFixtures(t *testing.T, home string, search []string, adapters map[manager.ManagerID]adapterm.Adapter, uc *recordingUpdate) {
	t.Helper()
	previousAdapters := managerAdapters
	previousUseCase := updateUseCase
	previousAll := updateAll
	previousOutput := updateOutput
	previousCheck := updateCheckDuplicates
	SetManagerAdapters(adapters)
	SetUpdateUseCase(uc)
	setDuplicateEnvironment(home, search)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Cleanup(func() {
		SetManagerAdapters(previousAdapters)
		SetUpdateUseCase(previousUseCase)
		clearDuplicateEnvironment()
		updateAll = previousAll
		updateOutput = previousOutput
		updateCheckDuplicates = previousCheck
		if err := updateCmd.Flags().Set("check-duplicates", "false"); err != nil {
			t.Errorf("reset check-duplicates: %v", err)
		}
		if err := updateCmd.Flags().Set("all", "false"); err != nil {
			t.Errorf("reset all: %v", err)
		}
		if err := updateCmd.Flags().Set("output", outputFormatText); err != nil {
			t.Errorf("reset output: %v", err)
		}
		updateCmd.SetArgs(nil)
	})
}

func executeUpdate(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var buf bytes.Buffer
	root := updateCmd.Root()
	previousOut := root.OutOrStdout()
	previousErr := root.ErrOrStderr()
	previousSilenceUsage := root.SilenceUsage
	previousSilenceErrors := root.SilenceErrors
	previousStdout := os.Stdout
	discard, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("discard stdout: %v", err)
	}
	os.Stdout = discard
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(append([]string{"update"}, args...))
	root.SilenceUsage = true
	root.SilenceErrors = true
	runErr := root.Execute()
	os.Stdout = previousStdout
	if closeErr := discard.Close(); closeErr != nil {
		t.Errorf("close discard: %v", closeErr)
	}
	root.SetOut(previousOut)
	root.SetErr(previousErr)
	root.SetArgs(nil)
	root.SilenceUsage = previousSilenceUsage
	root.SilenceErrors = previousSilenceErrors
	return buf.String(), runErr
}
