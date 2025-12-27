// Package memory provides in-memory repository implementations.
// This file contains the detecting repository that uses adapters
// to detect and query real package managers on the system.
package memory

import (
	"context"
	"time"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	adapterpkg "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/apt"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/asdf"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/cargo"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/chocolatey"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/homebrew"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/npm"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/pacman"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/pip"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/scoop"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/winget"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
)

// DetectingManagerRepository extends ManagerRepository with real detection.
// It uses adapters to detect and query actual package managers on the system.
type DetectingManagerRepository struct {
	*ManagerRepository
	adapters map[manager.ManagerID]adapterpkg.Adapter
	executor output.CommandExecutor
	logger   output.Logger
}

// NewDetectingManagerRepository creates a repository that detects real managers.
func NewDetectingManagerRepository(executor output.CommandExecutor, logger output.Logger) *DetectingManagerRepository {
	base := NewManagerRepository()

	repo := &DetectingManagerRepository{
		ManagerRepository: base,
		adapters:          make(map[manager.ManagerID]adapterpkg.Adapter),
		executor:          executor,
		logger:            logger,
	}

	// Register available adapters
	repo.registerAdapters()

	return repo
}

// registerAdapters initializes adapters for supported package managers.
func (r *DetectingManagerRepository) registerAdapters() {
	// Register system package managers
	r.adapters[manager.ManagerApt] = apt.NewAdapter(r.executor, r.logger)
	r.adapters[manager.ManagerHomebrew] = homebrew.NewAdapter(r.executor, r.logger)
	r.adapters[manager.ManagerPacman] = pacman.NewAdapter(r.executor, r.logger)

	// Register language package managers
	r.adapters[manager.ManagerNPM] = npm.NewAdapter(r.executor, r.logger)
	r.adapters[manager.ManagerPip] = pip.NewAdapter(r.executor, r.logger)
	r.adapters[manager.ManagerCargo] = cargo.NewAdapter(r.executor, r.logger)

	// Register version managers
	r.adapters[manager.ManagerASDF] = asdf.NewAdapter(r.executor, r.logger)

	// Register Windows package managers
	r.adapters[manager.ManagerWinget] = winget.NewAdapter(r.executor, r.logger)
	r.adapters[manager.ManagerScoop] = scoop.NewAdapter(r.executor, r.logger)
	r.adapters[manager.ManagerChocolatey] = chocolatey.NewAdapter(r.executor, r.logger)
}

// FindAll returns all managers with detection performed.
func (r *DetectingManagerRepository) FindAll(ctx context.Context) ([]*manager.Manager, error) {
	// Get base managers
	managers, err := r.ManagerRepository.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	// Detect and update each manager
	for _, mgr := range managers {
		if err := r.detectAndUpdate(ctx, mgr); err != nil {
			r.logger.Warn(ctx, "Failed to detect manager",
				output.Field{Key: "manager", Value: string(mgr.ID)},
				output.Field{Key: "error", Value: err.Error()})
			// Continue with other managers even if one fails
		}
	}

	return managers, nil
}

// FindInstalled returns only installed managers with detection performed.
func (r *DetectingManagerRepository) FindInstalled(ctx context.Context) ([]*manager.Manager, error) {
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

// FindByID returns a specific manager with detection performed.
func (r *DetectingManagerRepository) FindByID(ctx context.Context, id manager.ManagerID) (*manager.Manager, error) {
	mgr, err := r.ManagerRepository.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := r.detectAndUpdate(ctx, mgr); err != nil {
		r.logger.Warn(ctx, "Failed to detect manager",
			output.Field{Key: "manager", Value: string(id)},
			output.Field{Key: "error", Value: err.Error()})
	}

	return mgr, nil
}

// detectAndUpdate detects a manager and updates its state.
func (r *DetectingManagerRepository) detectAndUpdate(ctx context.Context, mgr *manager.Manager) error {
	adapter, exists := r.adapters[mgr.ID]
	if !exists {
		// No adapter available, keep as unavailable
		return nil
	}

	now := time.Now()

	// Detect if manager is installed
	detected, err := adapter.Detect(ctx)
	if err != nil {
		mgr.Status = manager.StatusError
		mgr.LastChecked = now
		return err
	}

	if !detected {
		mgr.Installed = false
		mgr.Status = manager.StatusUnavailable
		mgr.LastChecked = now
		return nil
	}

	// Manager is installed, gather information
	mgr.Installed = true

	// Get version
	if version, err := adapter.GetVersion(ctx); err == nil {
		mgr.Version = version
	} else {
		r.logger.Warn(ctx, "Failed to get version",
			output.Field{Key: "manager", Value: string(mgr.ID)},
			output.Field{Key: "error", Value: err.Error()})
	}

	// Get binary path
	if binPath, err := adapter.GetBinaryPath(ctx); err == nil {
		mgr.BinaryPath = binPath
	}

	// Get config path
	if configPath, err := adapter.GetConfigPath(ctx); err == nil {
		mgr.ConfigPath = configPath
	}

	// List packages
	if packages, err := adapter.ListPackages(ctx); err == nil {
		// Set manager ID for each package
		for i := range packages {
			packages[i].Manager = mgr.ID
		}
		mgr.Packages = packages
	} else {
		r.logger.Warn(ctx, "Failed to list packages",
			output.Field{Key: "manager", Value: string(mgr.ID)},
			output.Field{Key: "error", Value: err.Error()})
	}

	// Check health
	if status, err := adapter.CheckHealth(ctx); err == nil {
		mgr.Status = status
	} else {
		mgr.Status = manager.StatusDegraded
		r.logger.Warn(ctx, "Failed to check health",
			output.Field{Key: "manager", Value: string(mgr.ID)},
			output.Field{Key: "error", Value: err.Error()})
	}

	mgr.LastChecked = now

	return nil
}
