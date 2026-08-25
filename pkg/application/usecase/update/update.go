// Package update implements the update use case for updating package managers.
// It orchestrates fetching managers, executing updates, and collecting results.
package update

import (
	"context"
	"fmt"
	"time"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/dto"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/detector"
)

// UseCase implements the update use case.
type UseCase struct {
	managerRepo manager.Repository
	logger      output.Logger
	adapters    map[manager.ManagerID]adapterm.Adapter
	envDetector *detector.Detector
}

// NewUseCase creates a new update use case.
func NewUseCase(
	managerRepo manager.Repository,
	logger output.Logger,
	adapters map[manager.ManagerID]adapterm.Adapter,
	envDetector *detector.Detector,
) *UseCase {
	return &UseCase{
		managerRepo: managerRepo,
		logger:      logger,
		adapters:    adapters,
		envDetector: envDetector,
	}
}

// Update performs update operations on package managers.
func (uc *UseCase) Update(ctx context.Context, req *dto.UpdateRequest) (*dto.UpdateResponse, error) {
	startTime := time.Now()

	uc.logger.Info(
		ctx, "Starting package manager update",
		output.Field{Key: "all", Value: req.All},
		output.Field{Key: "dry_run", Value: req.DryRun},
		output.Field{Key: "strategy", Value: string(req.Strategy)},
	)

	// Determine which managers to update
	var managers []*manager.Manager
	var err error

	if req.All {
		// Update all installed managers
		managers, err = uc.managerRepo.FindInstalled(ctx)
		if err != nil {
			uc.logger.Error(ctx, "Failed to find installed managers", err)
			return nil, fmt.Errorf("failed to find installed managers: %w", err)
		}
	} else if len(req.ManagerIDs) > 0 {
		// Update specific managers
		managers = make([]*manager.Manager, 0, len(req.ManagerIDs))
		for _, id := range req.ManagerIDs {
			mgr, fetchErr := uc.managerRepo.FindByID(ctx, id)
			if fetchErr != nil {
				uc.logger.Error(ctx, "Failed to fetch manager", fetchErr, output.Field{Key: "manager_id", Value: id})
				return nil, fmt.Errorf("failed to fetch manager %s: %w", id, fetchErr)
			}
			if !mgr.Installed {
				uc.logger.Warn(ctx, "Manager not installed, skipping", output.Field{Key: "manager_id", Value: id})
				continue
			}
			managers = append(managers, mgr)
		}
	} else {
		return nil, fmt.Errorf("either --all flag or --managers must be specified")
	}

	if len(managers) == 0 {
		uc.logger.Info(ctx, "No managers to update")
		return &dto.UpdateResponse{
			Results: []*dto.ManagerUpdateResult{},
			Summary: &dto.UpdateSummary{},
			DryRun:  req.DryRun,
		}, nil
	}

	// Convert DTO strategy to adapter strategy
	adapterStrategy := uc.convertStrategy(req.Strategy)

	// Detect environment for pip safety checks
	var env *detector.Environment
	if uc.envDetector != nil {
		env = uc.envDetector.Detect(ctx)
		if len(env.Warnings) > 0 {
			for _, warning := range env.Warnings {
				uc.logger.Warn(ctx, warning)
			}
		}
	}

	// Update each manager
	results := make([]*dto.ManagerUpdateResult, 0, len(managers))
	summary := &dto.UpdateSummary{
		TotalManagers: len(managers),
	}

	for _, mgr := range managers {
		// Check if pip should be skipped in conda environment
		if mgr.ID == manager.ManagerPip && env != nil && !env.IsPipSafe && !req.PipAllowConda {
			uc.logger.Warn(
				ctx, "Skipping pip update in conda environment",
				output.Field{Key: "env_type", Value: string(env.Type)},
				output.Field{Key: "env_name", Value: env.Name},
			)
			result := &dto.ManagerUpdateResult{
				ID:              mgr.ID,
				Name:            mgr.Name,
				Success:         true, // Not a failure, intentional skip
				Skipped:         true,
				SkipReason:      fmt.Sprintf("Conda environment detected (%s). Use --pip-allow-conda to override.", env.Name),
				UpdatedPackages: []dto.PackageUpdate{},
				SkippedPackages: []string{},
			}
			results = append(results, result)
			summary.SkippedManagers++
			continue
		}

		uc.logger.Info(ctx, "Updating manager", output.Field{Key: "manager", Value: mgr.Name})

		result := uc.updateManager(ctx, mgr, adapterStrategy, req.DryRun)
		results = append(results, result)

		// Update summary
		if result.Skipped {
			summary.SkippedManagers++
		} else if result.Success {
			summary.SuccessfulManagers++
		} else {
			summary.FailedManagers++
		}
		summary.TotalPackagesUpdated += len(result.UpdatedPackages)
		summary.TotalBytesDownloaded += result.BytesDownloaded
		summary.TotalSpaceFreed += result.SpaceFreed
	}

	summary.TotalDuration = time.Since(startTime).Seconds()

	uc.logger.Info(
		ctx, "Update completed",
		output.Field{Key: "successful", Value: summary.SuccessfulManagers},
		output.Field{Key: "failed", Value: summary.FailedManagers},
		output.Field{Key: "skipped", Value: summary.SkippedManagers},
		output.Field{Key: "packages_updated", Value: summary.TotalPackagesUpdated},
		output.Field{Key: "duration_seconds", Value: summary.TotalDuration},
	)

	return &dto.UpdateResponse{
		Results: results,
		Summary: summary,
		DryRun:  req.DryRun,
	}, nil
}

// updateManager updates a single manager.
func (uc *UseCase) updateManager(
	ctx context.Context,
	mgr *manager.Manager,
	strategy adapterm.UpdateStrategy,
	dryRun bool,
) *dto.ManagerUpdateResult {
	startTime := time.Now()

	result := &dto.ManagerUpdateResult{
		ID:              mgr.ID,
		Name:            mgr.Name,
		UpdatedPackages: []dto.PackageUpdate{},
		SkippedPackages: []string{},
	}

	// Get the adapter for this manager
	adapter, exists := uc.adapters[mgr.ID]
	if !exists {
		result.Success = false
		errMsg := fmt.Sprintf("no adapter found for manager: %s", mgr.ID)
		result.Error = errMsg
		uc.logger.Error(ctx, "No adapter found", fmt.Errorf("%s", errMsg), output.Field{Key: "manager_id", Value: mgr.ID})
		return result
	}

	// Execute update
	opts := adapterm.UpdateOptions{
		DryRun:   dryRun,
		Strategy: strategy,
		Packages: []string{}, // Empty means update all packages
	}

	updateResult, err := adapter.Update(ctx, opts)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		uc.logger.Error(ctx, "Update failed", err, output.Field{Key: "manager", Value: mgr.Name})
		result.Duration = time.Since(startTime).Seconds()
		return result
	}

	result.Success = updateResult.Success
	result.Duration = time.Since(startTime).Seconds()

	// Note: For MVP, we're using simplified package update tracking.
	// The UpdateResult from adapters currently doesn't include detailed version info.
	// This will be enhanced in future iterations.
	for _, pkgName := range updateResult.UpdatedPackages {
		result.UpdatedPackages = append(result.UpdatedPackages, dto.PackageUpdate{
			Name:       pkgName,
			OldVersion: "unknown", // TODO: Track actual versions
			NewVersion: "unknown", // TODO: Track actual versions
			UpdateType: manager.UpdateMinor,
			SizeBytes:  0, // TODO: Track download sizes
		})
	}

	result.SkippedPackages = updateResult.FailedPackages

	uc.logger.Info(
		ctx, "Manager update completed",
		output.Field{Key: "manager", Value: mgr.Name},
		output.Field{Key: "success", Value: result.Success},
		output.Field{Key: "updated_packages", Value: len(result.UpdatedPackages)},
		output.Field{Key: "duration_seconds", Value: result.Duration},
	)

	return result
}

// convertStrategy converts DTO strategy to adapter strategy.
func (uc *UseCase) convertStrategy(dtoStrategy dto.UpdateStrategy) adapterm.UpdateStrategy {
	switch dtoStrategy {
	case dto.StrategyLatest:
		return adapterm.StrategyLatest
	case dto.StrategyStable:
		return adapterm.StrategyStable
	case dto.StrategyMinor:
		return adapterm.StrategyMinor
	case dto.StrategyFixed:
		return adapterm.StrategyFixed
	default:
		return adapterm.StrategyStable // Default to stable
	}
}
