package cleanup

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/cleanup"
)

func TestQuarantineRepository_SaveAndGet(t *testing.T) {
	repo := NewQuarantineRepository()
	ctx := context.Background()

	pkg := &cleanup.QuarantinedPackage{
		Name:          "test-pkg",
		Version:       "1.0.0",
		ManagerID:     "homebrew",
		QuarantinedAt: time.Now(),
		Reason:        "unused",
		Status:        cleanup.StatusQuarantined,
		SizeMB:        10.5,
	}

	// Save
	err := repo.Save(ctx, pkg)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Get
	got, err := repo.Get(ctx, pkg.Name, pkg.Version, pkg.ManagerID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if got.Name != pkg.Name || got.Version != pkg.Version {
		t.Errorf("got %s@%s, want %s@%s", got.Name, got.Version, pkg.Name, pkg.Version)
	}
}

func TestQuarantineRepository_GetNotFound(t *testing.T) {
	repo := NewQuarantineRepository()
	ctx := context.Background()

	_, err := repo.Get(ctx, "nonexistent", "1.0.0", "homebrew")
	if !errors.Is(err, cleanup.ErrPackageNotFound) {
		t.Errorf("got %v, want ErrPackageNotFound", err)
	}
}

func TestQuarantineRepository_List(t *testing.T) {
	repo := NewQuarantineRepository()
	ctx := context.Background()

	packages := []*cleanup.QuarantinedPackage{
		{Name: "pkg1", Version: "1.0", ManagerID: "homebrew", Status: cleanup.StatusQuarantined},
		{Name: "pkg2", Version: "2.0", ManagerID: testNPMManagerID, Status: cleanup.StatusQuarantined},
		{Name: "pkg3", Version: "3.0", ManagerID: "homebrew", Status: cleanup.StatusRestored},
	}

	for _, pkg := range packages {
		_ = repo.Save(ctx, pkg)
	}

	// List should only return quarantined packages
	got, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(got) != 2 {
		t.Errorf("got %d packages, want 2", len(got))
	}
}

func TestQuarantineRepository_ListByManager(t *testing.T) {
	repo := NewQuarantineRepository()
	ctx := context.Background()

	packages := []*cleanup.QuarantinedPackage{
		{Name: "pkg1", Version: "1.0", ManagerID: "homebrew", Status: cleanup.StatusQuarantined},
		{Name: "pkg2", Version: "2.0", ManagerID: testNPMManagerID, Status: cleanup.StatusQuarantined},
		{Name: "pkg3", Version: "3.0", ManagerID: "homebrew", Status: cleanup.StatusQuarantined},
	}

	for _, pkg := range packages {
		_ = repo.Save(ctx, pkg)
	}

	got, err := repo.ListByManager(ctx, "homebrew")
	if err != nil {
		t.Fatalf("ListByManager failed: %v", err)
	}

	if len(got) != 2 {
		t.Errorf("got %d packages, want 2 homebrew packages", len(got))
	}
}

func TestQuarantineRepository_Delete(t *testing.T) {
	repo := NewQuarantineRepository()
	ctx := context.Background()

	pkg := &cleanup.QuarantinedPackage{
		Name:      "test-pkg",
		Version:   "1.0.0",
		ManagerID: "homebrew",
		Status:    cleanup.StatusQuarantined,
	}

	_ = repo.Save(ctx, pkg)

	err := repo.Delete(ctx, pkg.Name, pkg.Version, pkg.ManagerID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = repo.Get(ctx, pkg.Name, pkg.Version, pkg.ManagerID)
	if !errors.Is(err, cleanup.ErrPackageNotFound) {
		t.Errorf("package should be deleted")
	}
}

func TestQuarantineRepository_FindExpired(t *testing.T) {
	repo := NewQuarantineRepository()
	ctx := context.Background()

	now := time.Now()
	packages := []*cleanup.QuarantinedPackage{
		{Name: "old", Version: "1.0", ManagerID: "brew", QuarantinedAt: now.AddDate(0, 0, -31), Status: cleanup.StatusQuarantined},
		{Name: "recent", Version: "1.0", ManagerID: "brew", QuarantinedAt: now.AddDate(0, 0, -5), Status: cleanup.StatusQuarantined},
	}

	for _, pkg := range packages {
		_ = repo.Save(ctx, pkg)
	}

	expired, err := repo.FindExpired(ctx, 30)
	if err != nil {
		t.Fatalf("FindExpired failed: %v", err)
	}

	if len(expired) != 1 {
		t.Errorf("got %d expired, want 1", len(expired))
	}

	if len(expired) > 0 && expired[0].Name != "old" {
		t.Errorf("got %s, want 'old'", expired[0].Name)
	}
}

func TestQuarantineRepository_FindExpiredInvalidDays(t *testing.T) {
	repo := NewQuarantineRepository()
	ctx := context.Background()

	_, err := repo.FindExpired(ctx, 0)
	if !errors.Is(err, cleanup.ErrInvalidRetentionDays) {
		t.Errorf("got %v, want ErrInvalidRetentionDays", err)
	}

	_, err = repo.FindExpired(ctx, -1)
	if !errors.Is(err, cleanup.ErrInvalidRetentionDays) {
		t.Errorf("got %v, want ErrInvalidRetentionDays", err)
	}
}
