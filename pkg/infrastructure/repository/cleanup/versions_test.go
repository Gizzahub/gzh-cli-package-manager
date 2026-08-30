package cleanup

import (
	"context"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
)

func TestScanVersionsFromPackages_MultiVersion(t *testing.T) {
	pkgs := []manager.Package{
		{Name: testPythonPackageName, CurrentVersion: "3.11.0"},
		{Name: testPythonPackageName, CurrentVersion: "3.12.0"},
		{Name: testGitPackageName, CurrentVersion: testGitCurrentVersion},
		{Name: testPythonPackageName, CurrentVersion: "3.11.0"}, // duplicate ignored
	}

	got := ScanVersionsFromPackages(testASDFManagerID, pkgs)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (only multi-version package rows)", len(got))
	}

	currentCount := 0
	for _, v := range got {
		if v.Name != testPythonPackageName {
			t.Errorf("unexpected name %q", v.Name)
		}
		if v.ManagerID != testASDFManagerID {
			t.Errorf("ManagerID = %q", v.ManagerID)
		}
		if v.IsCurrent {
			currentCount++
			if v.Version != "3.12.0" {
				t.Errorf("current version = %q, want 3.12.0 (lexicographic last)", v.Version)
			}
		}
	}
	if currentCount != 1 {
		t.Errorf("currentCount = %d, want 1", currentCount)
	}
}

func TestScanVersionsFromPackages_SingleNoReport(t *testing.T) {
	pkgs := []manager.Package{
		{Name: testGitPackageName, CurrentVersion: testGitCurrentVersion},
	}
	if got := ScanVersionsFromPackages("winget", pkgs); len(got) != 0 {
		t.Fatalf("expected empty, got %d", len(got))
	}
}

func TestHeuristicVersionScanner_Scan(t *testing.T) {
	s := NewHeuristicVersionScanner(map[string]PackageLister{
		testASDFManagerID: &stubLister{packages: []manager.Package{
			{Name: testNodePackageName, CurrentVersion: "18.0.0"},
			{Name: testNodePackageName, CurrentVersion: "20.0.0"},
			{Name: testPythonPackageName, CurrentVersion: "3.12.0"},
			{Name: testPythonPackageName, CurrentVersion: "3.11.0"},
		}},
	})

	got, err := s.Scan(context.Background(), testNodePackageName, testASDFManagerID)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
}
