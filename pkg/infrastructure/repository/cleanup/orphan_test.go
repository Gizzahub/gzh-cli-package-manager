package cleanup

import (
	"context"
	"errors"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
)

type stubLister struct {
	err      error
	packages []manager.Package
}

func (s *stubLister) ListPackages(_ context.Context) ([]manager.Package, error) {
	return s.packages, s.err
}

func TestDetectOrphansFromPackages(t *testing.T) {
	pkgs := make([]manager.Package, 0, 6)
	pkgs = append(pkgs, []manager.Package{
		{Name: testGitPackageName, CurrentVersion: testGitCurrentVersion},
		{Name: "", CurrentVersion: testVersionOne, SizeMB: 1.5},
		{Name: "unknown", CurrentVersion: "0.1", SizeMB: 2.5},
		{Name: "broken", CurrentVersion: "", SizeMB: 3.5},
		{Name: "-", CurrentVersion: "1", SizeMB: 4.5},
	}...)
	before := append([]manager.Package(nil), pkgs[:cap(pkgs)]...)
	wantLength := len(pkgs)
	wantCapacity := cap(pkgs)

	orphans := DetectOrphansFromPackages(testNPMManagerID, pkgs)
	if len(orphans) != 4 {
		t.Fatalf("orphans = %d, want 4", len(orphans))
	}
	if len(pkgs) != wantLength || cap(pkgs) != wantCapacity {
		t.Fatalf("packages shape = len %d cap %d, want len %d cap %d", len(pkgs), cap(pkgs), wantLength, wantCapacity)
	}
	for i, want := range before {
		if got := pkgs[:cap(pkgs)][i]; got != want {
			t.Errorf("packages[%d] = %#v, want %#v", i, got, want)
		}
	}

	wantNames := []string{
		displayName(pkgs[1].Name),
		pkgs[2].Name,
		pkgs[3].Name,
		pkgs[4].Name,
	}
	reasons := map[string]string{}
	for i, o := range orphans {
		if o.Name != wantNames[i] {
			t.Errorf("orphans[%d].Name = %q, want %q", i, o.Name, wantNames[i])
		}
		source := pkgs[i+1]
		if o.Version != source.CurrentVersion {
			t.Errorf("orphans[%d].Version = %q, want %q", i, o.Version, source.CurrentVersion)
		}
		if o.SizeMB != source.SizeMB {
			t.Errorf("orphans[%d].SizeMB = %v, want %v", i, o.SizeMB, source.SizeMB)
		}
		reasons[o.Name] = o.Reason
		if o.ManagerID != testNPMManagerID {
			t.Errorf("ManagerID = %q, want npm", o.ManagerID)
		}
	}
	if reasons["(unnamed)"] != "empty package name" {
		t.Errorf("unnamed reason = %q", reasons["(unnamed)"])
	}
	if reasons["unknown"] != "placeholder package name" {
		t.Errorf("unknown reason = %q", reasons["unknown"])
	}
	if reasons["broken"] != "missing version metadata" {
		t.Errorf("broken reason = %q", reasons["broken"])
	}
	if reasons["-"] != "placeholder package name" {
		t.Errorf("- reason = %q", reasons["-"])
	}
}

func TestHeuristicOrphanDetector_Detect(t *testing.T) {
	d := NewHeuristicOrphanDetector(map[string]PackageLister{
		testScoopManagerID: &stubLister{packages: []manager.Package{
			{Name: testGitPackageName, CurrentVersion: testGitCurrentVersion},
			{Name: "", CurrentVersion: "1"},
		}},
	})

	orphans, err := d.Detect(context.Background(), testScoopManagerID)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(orphans) != 1 {
		t.Fatalf("len = %d, want 1", len(orphans))
	}

	_, err = d.Detect(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error for missing manager")
	}
}

func TestHeuristicOrphanDetector_DetectAll_Error(t *testing.T) {
	d := NewHeuristicOrphanDetector(map[string]PackageLister{
		testNPMManagerID: &stubLister{err: errors.New("boom")},
	})
	_, err := d.DetectAll(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}
