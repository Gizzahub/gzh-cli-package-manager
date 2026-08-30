package memory

import (
	"context"
	"runtime"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
)

const testLinuxGOOS = "linux"

func TestManagerRepository_NewManagerRepository(t *testing.T) {
	repo := NewManagerRepository()

	if repo == nil {
		t.Fatal("Expected non-nil repository")
	}

	if repo.managers == nil {
		t.Fatal("Expected managers map to be initialized")
	}

	// Should have default managers initialized
	if len(repo.managers) == 0 {
		t.Error("Expected default managers to be initialized")
	}
}

func TestManagerRepository_FindAll(t *testing.T) {
	repo := NewManagerRepository()
	ctx := context.Background()

	managers, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll() error = %v", err)
	}

	if len(managers) == 0 {
		t.Error("Expected at least some managers")
	}

	// Verify managers have expected properties
	for _, mgr := range managers {
		if mgr.ID == "" {
			t.Error("Manager ID should not be empty")
		}
		if mgr.Name == "" {
			t.Error("Manager Name should not be empty")
		}
	}
}

func TestManagerRepository_FindInstalled(t *testing.T) {
	repo := NewManagerRepository()
	ctx := context.Background()

	// Initially, no managers should be installed (all are marked as not installed)
	installed, err := repo.FindInstalled(ctx)
	if err != nil {
		t.Fatalf("FindInstalled() error = %v", err)
	}

	if len(installed) != 0 {
		t.Errorf("Expected 0 installed managers initially, got %d", len(installed))
	}

	// Mark one as installed
	all, _ := repo.FindAll(ctx)
	if len(all) > 0 {
		all[0].Installed = true
		_ = repo.Save(ctx, all[0])

		installed, err = repo.FindInstalled(ctx)
		if err != nil {
			t.Fatalf("FindInstalled() error = %v", err)
		}

		if len(installed) != 1 {
			t.Errorf("Expected 1 installed manager, got %d", len(installed))
		}
	}
}

func TestManagerRepository_FindByID(t *testing.T) {
	repo := NewManagerRepository()
	ctx := context.Background()

	tests := []struct {
		name      string
		managerID manager.ManagerID
		wantErr   bool
	}{
		{
			name:      "homebrew exists",
			managerID: manager.ManagerHomebrew,
			wantErr:   false,
		},
		{
			name:      "npm exists",
			managerID: manager.ManagerNPM,
			wantErr:   false,
		},
		{
			name:      "invalid manager",
			managerID: manager.ManagerID("invalid-manager-id"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr, err := repo.FindByID(ctx, tt.managerID)

			if (err != nil) != tt.wantErr {
				t.Errorf("FindByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && mgr.ID != tt.managerID {
				t.Errorf("FindByID() ID = %v, want %v", mgr.ID, tt.managerID)
			}
		})
	}
}

func TestManagerRepository_Save(t *testing.T) {
	repo := NewManagerRepository()
	ctx := context.Background()

	// Create a new manager
	newManager := &manager.Manager{
		ID:        manager.ManagerID("custom-manager"),
		Name:      "Custom Manager",
		Type:      manager.TypeLanguage,
		Platform:  manager.PlatformDarwin,
		Installed: true,
		Version:   "1.0.0",
		Status:    manager.StatusHealthy,
	}

	// Save should succeed
	err := repo.Save(ctx, newManager)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify the manager was saved
	retrieved, err := repo.FindByID(ctx, newManager.ID)
	if err != nil {
		t.Fatalf("FindByID() after Save error = %v", err)
	}

	if retrieved.Name != "Custom Manager" {
		t.Errorf("Expected name 'Custom Manager', got %s", retrieved.Name)
	}

	if retrieved.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %s", retrieved.Version)
	}
}

func TestManagerRepository_Save_UpdateExisting(t *testing.T) {
	repo := NewManagerRepository()
	ctx := context.Background()

	// Get an existing manager
	original, err := repo.FindByID(ctx, manager.ManagerHomebrew)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	// Modify it
	original.Version = "4.0.0"
	original.Installed = true

	// Save the update
	err = repo.Save(ctx, original)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify the update
	updated, err := repo.FindByID(ctx, manager.ManagerHomebrew)
	if err != nil {
		t.Fatalf("FindByID() after update error = %v", err)
	}

	if updated.Version != "4.0.0" {
		t.Errorf("Expected version '4.0.0', got %s", updated.Version)
	}

	if !updated.Installed {
		t.Error("Expected Installed to be true")
	}
}

func TestManagerRepository_Delete(t *testing.T) {
	repo := NewManagerRepository()
	ctx := context.Background()

	// Verify manager exists
	_, err := repo.FindByID(ctx, manager.ManagerHomebrew)
	if err != nil {
		t.Fatalf("Manager should exist before delete: %v", err)
	}

	// Delete the manager
	err = repo.Delete(ctx, manager.ManagerHomebrew)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify the manager is gone
	_, err = repo.FindByID(ctx, manager.ManagerHomebrew)
	if err == nil {
		t.Error("Expected error when finding deleted manager")
	}
}

func TestManagerRepository_Delete_NonExistent(t *testing.T) {
	repo := NewManagerRepository()
	ctx := context.Background()

	// Delete a non-existent manager should not error
	err := repo.Delete(ctx, manager.ManagerID("non-existent"))
	if err != nil {
		t.Errorf("Delete() non-existent should not error, got %v", err)
	}
}

func TestGetCurrentPlatform(t *testing.T) {
	platform := getCurrentPlatform()

	switch runtime.GOOS {
	case "darwin":
		if platform != manager.PlatformDarwin {
			t.Errorf("Expected PlatformDarwin on darwin, got %s", platform)
		}
	case testLinuxGOOS:
		if platform != manager.PlatformLinux {
			t.Errorf("Expected PlatformLinux on linux, got %s", platform)
		}
	case "windows":
		if platform != manager.PlatformWindows {
			t.Errorf("Expected PlatformWindows on windows, got %s", platform)
		}
	default:
		if platform != manager.PlatformLinux {
			t.Errorf("Expected PlatformLinux on unknown, got %s", platform)
		}
	}
}

func TestManagerRepository_InitializeDefaultManagers(t *testing.T) {
	repo := NewManagerRepository()
	ctx := context.Background()

	// Verify common managers are initialized
	expectedManagers := []manager.ManagerID{
		manager.ManagerHomebrew,
		manager.ManagerASDF,
		manager.ManagerNPM,
		manager.ManagerPip,
		manager.ManagerCargo,
	}

	for _, id := range expectedManagers {
		mgr, err := repo.FindByID(ctx, id)
		if err != nil {
			t.Errorf("Expected manager %s to be initialized, but got error: %v", id, err)
			continue
		}

		// Check properties
		if mgr.Status != manager.StatusUnavailable {
			t.Errorf("Manager %s: expected StatusUnavailable, got %s", id, mgr.Status)
		}

		if mgr.Installed {
			t.Errorf("Manager %s: expected Installed=false", id)
		}
	}

	// On Linux, apt and pacman should also be present
	if runtime.GOOS == testLinuxGOOS {
		linuxManagers := []manager.ManagerID{
			manager.ManagerApt,
			manager.ManagerPacman,
		}

		for _, id := range linuxManagers {
			_, err := repo.FindByID(ctx, id)
			if err != nil {
				t.Errorf("Expected manager %s to be initialized on Linux, but got error: %v", id, err)
			}
		}
	}
}

func TestManagerRepository_ConcurrentAccess(t *testing.T) {
	repo := NewManagerRepository()
	ctx := context.Background()

	// Run multiple goroutines to test concurrent access
	done := make(chan bool)

	// Reader goroutine
	go func() {
		for range 100 {
			_, _ = repo.FindAll(ctx)
			_, _ = repo.FindByID(ctx, manager.ManagerHomebrew)
		}
		done <- true
	}()

	// Writer goroutine
	go func() {
		for range 100 {
			mgr := &manager.Manager{
				ID:   manager.ManagerID("test-concurrent"),
				Name: "Test Concurrent",
			}
			_ = repo.Save(ctx, mgr)
			_ = repo.Delete(ctx, mgr.ID)
		}
		done <- true
	}()

	// Wait for both to complete
	<-done
	<-done
}
