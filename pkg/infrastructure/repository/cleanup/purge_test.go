package cleanup

import (
	"context"
	"testing"
	"time"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/cleanup"
)

func TestQuarantinePurger_DryRunAndPurge(t *testing.T) {
	t.Parallel()
	repo := NewQuarantineRepository()
	ctx := context.Background()
	now := time.Now()

	old := &cleanup.QuarantinedPackage{
		Name: "old-pkg", Version: "1.0.0", ManagerID: "homebrew",
		QuarantinedAt: now.AddDate(0, 0, -40), Status: cleanup.StatusQuarantined, SizeMB: 12.5,
	}
	recent := &cleanup.QuarantinedPackage{
		Name: "new-pkg", Version: "2.0.0", ManagerID: "homebrew",
		QuarantinedAt: now.AddDate(0, 0, -5), Status: cleanup.StatusQuarantined, SizeMB: 3.0,
	}
	if err := repo.Save(ctx, old); err != nil {
		t.Fatalf("Save old: %v", err)
	}
	if err := repo.Save(ctx, recent); err != nil {
		t.Fatalf("Save recent: %v", err)
	}

	purger := NewQuarantinePurger(repo)

	// Dry-run: no deletion
	summary, err := purger.PurgeExpired(ctx, 30, true)
	if err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if summary.PackagesRemoved != 1 || summary.SpaceFreedMB != 12.5 {
		t.Fatalf("dry-run summary: %+v", summary)
	}
	if _, lookupErr := repo.Get(ctx, "old-pkg", "1.0.0", "homebrew"); lookupErr != nil {
		t.Fatalf("dry-run should keep package: %v", lookupErr)
	}

	// Actual purge
	summary, err = purger.PurgeExpired(ctx, 30, false)
	if err != nil {
		t.Fatalf("purge: %v", err)
	}
	if summary.PackagesRemoved != 1 {
		t.Fatalf("purge count: %d", summary.PackagesRemoved)
	}
	if _, err := repo.Get(ctx, "old-pkg", "1.0.0", "homebrew"); err == nil {
		t.Fatal("expected old package purged")
	}
	if _, err := repo.Get(ctx, "new-pkg", "2.0.0", "homebrew"); err != nil {
		t.Fatalf("recent should remain: %v", err)
	}
}

func TestQuarantinePurger_InvalidRetention(t *testing.T) {
	t.Parallel()
	purger := NewQuarantinePurger(NewQuarantineRepository())
	_, err := purger.PurgeExpired(context.Background(), 0, false)
	if err == nil {
		t.Fatal("expected invalid retention error")
	}
}
