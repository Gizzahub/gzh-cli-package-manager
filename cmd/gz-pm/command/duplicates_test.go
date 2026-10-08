package command

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/diagnostics"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
)

func TestStatusDuplicates(t *testing.T) {
	t.Run("report-ok", testDuplicateReportOK)
	t.Run("text-and-json", testDuplicateTextAndJSON)
	t.Run("collection-fails", testDuplicateCollectionFails)
	t.Run("strict-warning", testDuplicateStrictWarning)
	t.Run("strict-note", testDuplicateStrictNote)
	t.Run("single-provider-omitted", testDuplicateSingleProviderOmitted)
	t.Run("summary-unchanged", testDuplicateSummaryUnchanged)
	t.Run("probe-appended", testDuplicateProbeAppended)
	t.Run("list-packages-untouched", testDuplicateListPackagesUntouched)
}

func testDuplicateReportOK(t *testing.T) {
	home, search, _, adapters := warningAdapters(t)
	useDuplicateFixtures(t, home, search, adapters)
	out, err := executeStatus(t, "duplicates")
	if err != nil {
		t.Fatalf("report-ok exit: %v", err)
	}
	assertFacts(t, out, "claude", "warning", "homebrew", "npm")
	assertNoRemovalAdvice(t, out)
}

func testDuplicateTextAndJSON(t *testing.T) {
	home, search, executed, adapters := warningAdapters(t)
	useDuplicateFixtures(t, home, search, adapters)
	text, err := executeStatus(t, "duplicates", "--output", "text")
	if err != nil {
		t.Fatalf("text exit: %v", err)
	}
	encoded, err := executeStatus(t, "duplicates", "--output", "json")
	if err != nil {
		t.Fatalf("json exit: %v", err)
	}
	assertFacts(t, text, "claude", "warning", "homebrew", "npm", "1.0.0", "2.0.0", executed, "inactive")
	assertFacts(t, encoded, "claude", "warning", "homebrew", "npm", "1.0.0", "2.0.0", executed, "inactive")
}

func testDuplicateCollectionFails(t *testing.T) {
	home := t.TempDir()
	search := []string{t.TempDir()}
	failing := &inventoryAdapter{err: errors.New("collector failed")}
	useDuplicateFixtures(t, home, search, map[manager.ManagerID]adapterm.Adapter{
		"fail": failing,
	})
	if _, err := executeStatus(t, "duplicates"); err == nil {
		t.Fatal("collection-fails: expected non-zero exit")
	}
	if _, err := executeStatus(t, "duplicates", "--strict"); err == nil {
		t.Fatal("collection-fails: expected non-zero exit with strict")
	}
}

func testDuplicateStrictWarning(t *testing.T) {
	home, search, _, adapters := warningAdapters(t)
	useDuplicateFixtures(t, home, search, adapters)
	if _, err := executeStatus(t, "duplicates"); err != nil {
		t.Fatalf("strict-warning without flag: %v", err)
	}
	out, err := executeStatus(t, "duplicates", "--strict")
	if err == nil {
		t.Fatal("strict-warning: expected non-zero exit")
	}
	assertFacts(t, out, "claude", "warning")
}

func testDuplicateStrictNote(t *testing.T) {
	home, search, adapters := noteAdapters(t)
	useDuplicateFixtures(t, home, search, adapters)
	out, err := executeStatus(t, "duplicates")
	if err != nil {
		t.Fatalf("strict-note without flag: %v", err)
	}
	assertFacts(t, out, "node", "note")
	assertNoRemovalAdvice(t, out)
	out, err = executeStatus(t, "duplicates", "--strict")
	if err != nil {
		t.Fatalf("strict-note with flag: %v", err)
	}
	assertFacts(t, out, "node", "note")
}

func testDuplicateSingleProviderOmitted(t *testing.T) {
	home, search, _, adapters := warningAdapters(t)
	useDuplicateFixtures(t, home, search, adapters)
	text, err := executeStatus(t, "duplicates")
	if err != nil {
		t.Fatalf("single-provider text: %v", err)
	}
	encoded, err := executeStatus(t, "duplicates", "--output", "json")
	if err != nil {
		t.Fatalf("single-provider json: %v", err)
	}
	if strings.Contains(text, "gemini") || strings.Contains(encoded, "gemini") {
		t.Fatalf("single-provider-omitted: gemini present\ntext=%s\njson=%s", text, encoded)
	}
}

func testDuplicateSummaryUnchanged(t *testing.T) {
	home := t.TempDir()
	search := []string{t.TempDir()}
	inventory := &inventoryAdapter{records: []diagnostics.InstallRecord{{
		Command:    "summary-marker",
		ProviderID: "homebrew",
		Kind:       diagnostics.KindSystemOrLanguage,
		RealPath:   filepath.Join(search[0], "summary-marker"),
		Active:     true,
	}}}
	useDuplicateFixtures(t, home, search, map[manager.ManagerID]adapterm.Adapter{
		manager.ManagerHomebrew: inventory,
	})
	statusUseCase = nil
	out, err := executeStatus(t)
	if err != nil {
		t.Fatalf("summary exit: %v", err)
	}
	if inventory.calls != 0 {
		t.Fatalf("summary-unchanged: collector calls = %d", inventory.calls)
	}
	if strings.Contains(out, "summary-marker") || strings.Contains(out, "command:") {
		t.Fatalf("summary-unchanged printed a duplicate report: %s", out)
	}
}

func testDuplicateProbeAppended(t *testing.T) {
	home := t.TempDir()
	searchDir := t.TempDir()
	executed := writeCommandFile(t, searchDir, "claude")
	nativeDir := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(nativeDir, 0o755); err != nil {
		t.Fatalf("probe dir: %v", err)
	}
	writeCommandFile(t, nativeDir, "claude")
	adapters := map[manager.ManagerID]adapterm.Adapter{
		manager.ManagerHomebrew: &inventoryAdapter{records: []diagnostics.InstallRecord{
			commandRecord("claude", "homebrew", diagnostics.KindSystemOrLanguage, "1.0.0", executed, true),
		}},
	}
	useDuplicateFixtures(t, home, []string{searchDir}, adapters)
	out, err := executeStatus(t, "duplicates")
	if err != nil {
		t.Fatalf("probe-appended exit: %v", err)
	}
	assertFacts(t, out, "claude", "warning", "claude-native", "homebrew", "unmanaged")
	assertNoRemovalAdvice(t, out)
}

func testDuplicateListPackagesUntouched(t *testing.T) {
	home, search, _, adapters := warningAdapters(t)
	lister := &stubAdapter{}
	adapters["lister"] = lister
	useDuplicateFixtures(t, home, search, adapters)
	out, err := executeStatus(t, "duplicates")
	if err != nil {
		t.Fatalf("list-packages exit: %v", err)
	}
	if lister.listCalls != 0 {
		t.Fatalf("list-packages-untouched: ListPackages calls = %d", lister.listCalls)
	}
	if strings.Contains(out, "listed-package") {
		t.Fatalf("list-packages-untouched reported a listed package: %s", out)
	}
}

func useDuplicateFixtures(t *testing.T, home string, search []string, adapters map[manager.ManagerID]adapterm.Adapter) {
	t.Helper()
	previousAdapters := managerAdapters
	previousUseCase := statusUseCase
	SetManagerAdapters(adapters)
	setDuplicateEnvironment(home, search)
	t.Cleanup(func() {
		SetManagerAdapters(previousAdapters)
		statusUseCase = previousUseCase
		clearDuplicateEnvironment()
		if err := duplicatesCmd.Flags().Set("strict", "false"); err != nil {
			t.Errorf("reset strict: %v", err)
		}
		if err := duplicatesCmd.Flags().Set("output", outputFormatText); err != nil {
			t.Errorf("reset output: %v", err)
		}
		statusCmd.SetArgs(nil)
		duplicatesOutput = outputFormatText
		duplicatesStrict = false
	})
}

func executeStatus(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var buf bytes.Buffer
	root := statusCmd.Root()
	previousOut := root.OutOrStdout()
	previousErr := root.ErrOrStderr()
	previousSilenceUsage := root.SilenceUsage
	previousSilenceErrors := root.SilenceErrors
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(append([]string{"status"}, args...))
	root.SilenceUsage = true
	root.SilenceErrors = true
	err := root.Execute()
	root.SetOut(previousOut)
	root.SetErr(previousErr)
	root.SetArgs(nil)
	root.SilenceUsage = previousSilenceUsage
	root.SilenceErrors = previousSilenceErrors
	return buf.String(), err
}

func warningAdapters(t *testing.T) (home string, search []string, executed string, adapters map[manager.ManagerID]adapterm.Adapter) {
	t.Helper()
	home = t.TempDir()
	searchDir := t.TempDir()
	otherDir := t.TempDir()
	executed = writeCommandFile(t, searchDir, "claude")
	other := writeCommandFile(t, otherDir, "claude")
	writeCommandFile(t, searchDir, "gemini")
	gemini := writeCommandFile(t, otherDir, "gemini")
	adapters = map[manager.ManagerID]adapterm.Adapter{
		manager.ManagerHomebrew: &inventoryAdapter{records: []diagnostics.InstallRecord{
			commandRecord("claude", "homebrew", diagnostics.KindSystemOrLanguage, "1.0.0", executed, true),
			commandRecord("gemini", "homebrew", diagnostics.KindSystemOrLanguage, "0.1.0", gemini, true),
		}},
		manager.ManagerNPM: &inventoryAdapter{records: []diagnostics.InstallRecord{
			commandRecord("claude", "npm", diagnostics.KindSystemOrLanguage, "2.0.0", other, false),
		}},
	}
	return home, []string{searchDir}, executed, adapters
}

func noteAdapters(t *testing.T) (home string, search []string, adapters map[manager.ManagerID]adapterm.Adapter) {
	t.Helper()
	home = t.TempDir()
	searchDir := t.TempDir()
	otherDir := t.TempDir()
	executed := writeCommandFile(t, searchDir, "node")
	other := writeCommandFile(t, otherDir, "node")
	adapters = map[manager.ManagerID]adapterm.Adapter{
		manager.ManagerMise: &inventoryAdapter{records: []diagnostics.InstallRecord{
			commandRecord("node", "mise", diagnostics.KindVersionManager, "20.0.0", executed, true),
		}},
		manager.ManagerHomebrew: &inventoryAdapter{records: []diagnostics.InstallRecord{
			commandRecord("node", "homebrew", diagnostics.KindSystemOrLanguage, "20.0.0", other, true),
		}},
	}
	return home, []string{searchDir}, adapters
}

func commandRecord(command, provider string, kind diagnostics.ProviderKind, version, path string, active bool) diagnostics.InstallRecord {
	return diagnostics.InstallRecord{
		Command:    command,
		ProviderID: provider,
		Kind:       kind,
		Version:    version,
		RealPath:   path,
		Active:     active,
	}
}

func writeCommandFile(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("fixture"), 0o755); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func assertFacts(t *testing.T, out string, facts ...string) {
	t.Helper()
	for _, fact := range facts {
		if !strings.Contains(out, fact) {
			t.Fatalf("output missing %q\n%s", fact, out)
		}
	}
}

func assertNoRemovalAdvice(t *testing.T, out string) {
	t.Helper()
	for _, word := range []string{"uninstall", "relink", "rewrite"} {
		if strings.Contains(out, word) {
			t.Fatalf("output recommends removal via %q\n%s", word, out)
		}
	}
}

type stubAdapter struct {
	listCalls int
}

func (s *stubAdapter) Detect(context.Context) (bool, error) { return false, nil }
func (s *stubAdapter) GetVersion(context.Context) (string, error) {
	return "", nil
}
func (s *stubAdapter) GetBinaryPath(context.Context) (string, error) { return "", nil }
func (s *stubAdapter) GetConfigPath(context.Context) (string, error) { return "", nil }
func (s *stubAdapter) ListPackages(context.Context) ([]manager.Package, error) {
	s.listCalls++
	return []manager.Package{{Name: "listed-package"}}, nil
}

func (s *stubAdapter) CheckHealth(context.Context) (manager.Status, error) {
	return "", nil
}

func (s *stubAdapter) Update(context.Context, adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
	return &adapterm.UpdateResult{}, nil
}

type inventoryAdapter struct {
	stubAdapter
	records []diagnostics.InstallRecord
	err     error
	calls   int
}

func (a *inventoryAdapter) Commands(context.Context) ([]diagnostics.InstallRecord, error) {
	a.calls++
	if a.err != nil {
		return nil, a.err
	}
	return a.records, nil
}
