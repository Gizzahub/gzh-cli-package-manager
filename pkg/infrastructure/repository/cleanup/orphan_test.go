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
	pkgs := []manager.Package{
		{Name: "git", CurrentVersion: "2.43.0"},
		{Name: "", CurrentVersion: "1.0"},
		{Name: "unknown", CurrentVersion: "0.1"},
		{Name: "broken", CurrentVersion: ""},
		{Name: "-", CurrentVersion: "1"},
	}

	orphans := DetectOrphansFromPackages("npm", pkgs)
	if len(orphans) != 4 {
		t.Fatalf("orphans = %d, want 4", len(orphans))
	}

	reasons := map[string]string{}
	for _, o := range orphans {
		reasons[o.Name] = o.Reason
		if o.ManagerID != "npm" {
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
		"scoop": &stubLister{packages: []manager.Package{
			{Name: "git", CurrentVersion: "2.43.0"},
			{Name: "", CurrentVersion: "1"},
		}},
	})

	orphans, err := d.Detect(context.Background(), "scoop")
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
		"npm": &stubLister{err: errors.New("boom")},
	})
	_, err := d.DetectAll(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}
