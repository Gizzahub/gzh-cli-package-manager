package cleanup

import (
	"context"
	"errors"
	"testing"

	domaincleanup "github.com/gizzahub/gzh-cli-package-manager/pkg/domain/cleanup"
)

type recordingUninstaller struct {
	failOn string
	calls  []uninstallCall
}

type uninstallCall struct {
	managerID string
	pkgID     string
	dryRun    bool
}

func (r *recordingUninstaller) Uninstall(_ context.Context, managerID, pkgID string, dryRun bool) error {
	r.calls = append(r.calls, uninstallCall{managerID: managerID, pkgID: pkgID, dryRun: dryRun})
	if r.failOn != "" && pkgID == r.failOn {
		return errors.New("boom")
	}
	return nil
}

func TestRemoveOrphans_DryRun(t *testing.T) {
	u := &recordingUninstaller{}
	ex := NewAdapterCleanupExecutor(u)
	summary, err := ex.RemoveOrphans(context.Background(), []*domaincleanup.OrphanPackage{
		{Name: testGitPackageName, Version: testVersionOne, ManagerID: "scoop", Reason: "missing version metadata"},
		{Name: "(unnamed)", ManagerID: "scoop", Reason: "empty package name"},
		{Name: "unknown", ManagerID: "scoop", Reason: "placeholder"},
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	if summary.PackagesRemoved != 1 {
		t.Fatalf("removed=%d want 1 (only actionable name)", summary.PackagesRemoved)
	}
	if len(u.calls) != 1 || !u.calls[0].dryRun || u.calls[0].pkgID != testGitPackageName {
		t.Fatalf("calls=%+v", u.calls)
	}
	if len(summary.Errors) != 2 {
		t.Fatalf("errors=%v want 2 skips", summary.Errors)
	}
}

func TestRemoveOrphans_Live(t *testing.T) {
	u := &recordingUninstaller{}
	ex := NewAdapterCleanupExecutor(u)
	summary, err := ex.RemoveOrphans(context.Background(), []*domaincleanup.OrphanPackage{
		{Name: "pkg-a", ManagerID: "winget"},
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	if summary.PackagesRemoved != 1 || u.calls[0].dryRun {
		t.Fatalf("want live uninstall: summary=%+v calls=%+v", summary, u.calls)
	}
}

func TestRemoveOldVersions_SkipsCurrent(t *testing.T) {
	u := &recordingUninstaller{}
	ex := NewAdapterCleanupExecutor(u)
	summary, err := ex.RemoveOldVersions(context.Background(), []*domaincleanup.OldVersion{
		{Name: testNodePackageName, Version: "18.0.0", ManagerID: testASDFManagerID, IsCurrent: false},
		{Name: testNodePackageName, Version: "20.0.0", ManagerID: testASDFManagerID, IsCurrent: true},
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	if summary.PackagesRemoved != 1 {
		t.Fatalf("removed=%d want 1", summary.PackagesRemoved)
	}
	if u.calls[0].pkgID != "node@18.0.0" {
		t.Fatalf("pkgID=%q want node@18.0.0", u.calls[0].pkgID)
	}
}

func TestRemoveOldVersions_RetryBareName(t *testing.T) {
	u := &recordingUninstaller{failOn: "git@2.0.0"}
	ex := NewAdapterCleanupExecutor(u)
	summary, err := ex.RemoveOldVersions(context.Background(), []*domaincleanup.OldVersion{
		{Name: testGitPackageName, Version: "2.0.0", ManagerID: "winget", IsCurrent: false},
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	if summary.PackagesRemoved != 1 {
		t.Fatalf("removed=%d errors=%v", summary.PackagesRemoved, summary.Errors)
	}
	if len(u.calls) != 2 || u.calls[1].pkgID != testGitPackageName {
		t.Fatalf("expected retry with bare name: %+v", u.calls)
	}
}

func TestRemoveOrphans_NoUninstaller(t *testing.T) {
	ex := NewAdapterCleanupExecutor(nil)
	_, err := ex.RemoveOrphans(context.Background(), nil, true)
	if err == nil {
		t.Fatal("expected error")
	}
}
