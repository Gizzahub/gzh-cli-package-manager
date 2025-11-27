package memory

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
)

// ManagerRepository is an in-memory implementation of manager.Repository.
// This is useful for testing and initial development.
type ManagerRepository struct {
	mu       sync.RWMutex
	managers map[manager.ManagerID]*manager.Manager
}

// NewManagerRepository creates a new in-memory manager repository.
func NewManagerRepository() *ManagerRepository {
	repo := &ManagerRepository{
		managers: make(map[manager.ManagerID]*manager.Manager),
	}
	repo.initializeDefaultManagers()
	return repo
}

// FindAll returns all supported package managers for the current platform.
func (r *ManagerRepository) FindAll(_ context.Context) ([]*manager.Manager, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*manager.Manager, 0, len(r.managers))
	for _, mgr := range r.managers {
		result = append(result, mgr)
	}

	return result, nil
}

// FindInstalled returns only the installed package managers.
func (r *ManagerRepository) FindInstalled(ctx context.Context) ([]*manager.Manager, error) {
	all, err := r.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*manager.Manager, 0)
	for _, mgr := range all {
		if mgr.Installed {
			result = append(result, mgr)
		}
	}

	return result, nil
}

// FindByID returns a specific manager by its ID.
func (r *ManagerRepository) FindByID(_ context.Context, id manager.ManagerID) (*manager.Manager, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	mgr, exists := r.managers[id]
	if !exists {
		return nil, fmt.Errorf("manager not found: %s", id)
	}

	return mgr, nil
}

// Save persists a manager's state.
func (r *ManagerRepository) Save(_ context.Context, m *manager.Manager) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.managers[m.ID] = m
	return nil
}

// Delete removes a manager's persisted state.
func (r *ManagerRepository) Delete(_ context.Context, id manager.ManagerID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.managers, id)
	return nil
}

// initializeDefaultManagers populates the repository with default managers.
func (r *ManagerRepository) initializeDefaultManagers() {
	now := time.Now()
	platform := getCurrentPlatform()

	// Define all supported managers
	managers := []*manager.Manager{
		{
			ID:          manager.ManagerHomebrew,
			Name:        "Homebrew",
			Type:        manager.TypeSystem,
			Platform:    platform,
			Installed:   false,
			Version:     "",
			Status:      manager.StatusUnavailable,
			ConfigPath:  "",
			BinaryPath:  "",
			Packages:    []manager.Package{},
			LastChecked: now,
		},
		{
			ID:          manager.ManagerASDF,
			Name:        "ASDF",
			Type:        manager.TypeVersion,
			Platform:    platform,
			Installed:   false,
			Version:     "",
			Status:      manager.StatusUnavailable,
			ConfigPath:  "",
			BinaryPath:  "",
			Packages:    []manager.Package{},
			LastChecked: now,
		},
		{
			ID:          manager.ManagerNPM,
			Name:        "NPM",
			Type:        manager.TypeLanguage,
			Platform:    platform,
			Installed:   false,
			Version:     "",
			Status:      manager.StatusUnavailable,
			ConfigPath:  "",
			BinaryPath:  "",
			Packages:    []manager.Package{},
			LastChecked: now,
		},
		{
			ID:          manager.ManagerPip,
			Name:        "Pip",
			Type:        manager.TypeLanguage,
			Platform:    platform,
			Installed:   false,
			Version:     "",
			Status:      manager.StatusUnavailable,
			ConfigPath:  "",
			BinaryPath:  "",
			Packages:    []manager.Package{},
			LastChecked: now,
		},
		{
			ID:          manager.ManagerCargo,
			Name:        "Cargo",
			Type:        manager.TypeLanguage,
			Platform:    platform,
			Installed:   false,
			Version:     "",
			Status:      manager.StatusUnavailable,
			ConfigPath:  "",
			BinaryPath:  "",
			Packages:    []manager.Package{},
			LastChecked: now,
		},
	}

	// Add platform-specific managers
	if platform == manager.PlatformLinux {
		managers = append(managers, []*manager.Manager{
			{
				ID:          manager.ManagerApt,
				Name:        "APT",
				Type:        manager.TypeSystem,
				Platform:    platform,
				Installed:   false,
				Version:     "",
				Status:      manager.StatusUnavailable,
				ConfigPath:  "",
				BinaryPath:  "",
				Packages:    []manager.Package{},
				LastChecked: now,
			},
			{
				ID:          manager.ManagerPacman,
				Name:        "Pacman",
				Type:        manager.TypeSystem,
				Platform:    platform,
				Installed:   false,
				Version:     "",
				Status:      manager.StatusUnavailable,
				ConfigPath:  "",
				BinaryPath:  "",
				Packages:    []manager.Package{},
				LastChecked: now,
			},
		}...)
	}

	for _, mgr := range managers {
		r.managers[mgr.ID] = mgr
	}
}

// getCurrentPlatform returns the current platform.
func getCurrentPlatform() manager.Platform {
	switch runtime.GOOS {
	case "darwin":
		return manager.PlatformDarwin
	case "linux":
		return manager.PlatformLinux
	case "windows":
		return manager.PlatformWindows
	default:
		return manager.PlatformLinux
	}
}
