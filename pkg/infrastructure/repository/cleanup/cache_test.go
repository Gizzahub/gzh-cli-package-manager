package cleanup

import (
	"context"
	"testing"
	"time"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/cleanup"
)

func TestCacheRepository_GetInfo(t *testing.T) {
	repo := NewCacheRepository()
	ctx := context.Background()

	// Get non-existent manager returns empty info
	info, err := repo.GetInfo(ctx, "homebrew")
	if err != nil {
		t.Fatalf("GetInfo failed: %v", err)
	}

	if info.ManagerID != "homebrew" {
		t.Errorf("got ManagerID %s, want homebrew", info.ManagerID)
	}
}

func TestCacheRepository_UpdateAndGetInfo(t *testing.T) {
	repo := NewCacheRepository()
	ctx := context.Background()

	info := &cleanup.CacheInfo{
		ManagerID:   "homebrew",
		CachePath:   "~/Library/Caches/Homebrew",
		TotalSizeMB: 1024.5,
		EntryCount:  150,
		OldestEntry: time.Now().AddDate(0, -3, 0),
		NewestEntry: time.Now(),
	}

	err := repo.UpdateInfo(ctx, info)
	if err != nil {
		t.Fatalf("UpdateInfo failed: %v", err)
	}

	got, err := repo.GetInfo(ctx, "homebrew")
	if err != nil {
		t.Fatalf("GetInfo failed: %v", err)
	}

	if got.TotalSizeMB != info.TotalSizeMB {
		t.Errorf("got TotalSizeMB %f, want %f", got.TotalSizeMB, info.TotalSizeMB)
	}

	if got.EntryCount != info.EntryCount {
		t.Errorf("got EntryCount %d, want %d", got.EntryCount, info.EntryCount)
	}
}

func TestCacheRepository_ListAll(t *testing.T) {
	repo := NewCacheRepository()
	ctx := context.Background()

	caches := []*cleanup.CacheInfo{
		{ManagerID: "homebrew", TotalSizeMB: 100},
		{ManagerID: testNPMManagerID, TotalSizeMB: 200},
		{ManagerID: "pip", TotalSizeMB: 50},
	}

	for _, cache := range caches {
		_ = repo.UpdateInfo(ctx, cache)
	}

	got, err := repo.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}

	if len(got) != 3 {
		t.Errorf("got %d caches, want 3", len(got))
	}

	// Calculate total size
	var total float64
	for _, c := range got {
		total += c.TotalSizeMB
	}

	if total != 350 {
		t.Errorf("got total %f MB, want 350 MB", total)
	}
}

func TestCacheRepository_UpdateExisting(t *testing.T) {
	repo := NewCacheRepository()
	ctx := context.Background()

	// First update
	info := &cleanup.CacheInfo{
		ManagerID:   testNPMManagerID,
		TotalSizeMB: 100,
	}
	_ = repo.UpdateInfo(ctx, info)

	// Second update (same manager)
	info2 := &cleanup.CacheInfo{
		ManagerID:   testNPMManagerID,
		TotalSizeMB: 200,
	}
	_ = repo.UpdateInfo(ctx, info2)

	// Should have only one entry with updated value
	all, _ := repo.ListAll(ctx)
	if len(all) != 1 {
		t.Errorf("got %d entries, want 1", len(all))
	}

	got, _ := repo.GetInfo(ctx, testNPMManagerID)
	if got.TotalSizeMB != 200 {
		t.Errorf("got TotalSizeMB %f, want 200", got.TotalSizeMB)
	}
}
