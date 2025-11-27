// Package status implements the status use case for querying package manager status.
// It orchestrates fetching manager information from the repository and
// converting domain entities to DTOs for presentation.
package status

import (
	"context"
	"fmt"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/dto"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
)

// UseCase implements the status use case.
type UseCase struct {
	managerRepo manager.Repository
	logger      output.Logger
}

// NewUseCase creates a new status use case.
func NewUseCase(managerRepo manager.Repository, logger output.Logger) *UseCase {
	return &UseCase{
		managerRepo: managerRepo,
		logger:      logger,
	}
}

// GetStatus retrieves the status of all package managers.
func (uc *UseCase) GetStatus(ctx context.Context, req *dto.StatusRequest) (*dto.StatusResponse, error) {
	uc.logger.Info(ctx, "Getting package manager status", output.Field{Key: "verbose", Value: req.Verbose})

	// Fetch managers from repository
	var managers []*manager.Manager
	var err error

	if len(req.ManagerIDs) > 0 {
		// Fetch specific managers
		managers = make([]*manager.Manager, 0, len(req.ManagerIDs))
		for _, id := range req.ManagerIDs {
			mgr, fetchErr := uc.managerRepo.FindByID(ctx, id)
			if fetchErr != nil {
				uc.logger.Error(ctx, "Failed to fetch manager", fetchErr, output.Field{Key: "manager_id", Value: id})
				return nil, fmt.Errorf("failed to fetch manager %s: %w", id, fetchErr)
			}
			managers = append(managers, mgr)
		}
	} else {
		// Fetch all managers
		managers, err = uc.managerRepo.FindAll(ctx)
		if err != nil {
			uc.logger.Error(ctx, "Failed to fetch all managers", err)
			return nil, fmt.Errorf("failed to fetch managers: %w", err)
		}
	}

	// Convert domain entities to DTOs
	managerStatuses := make([]*dto.ManagerStatus, 0, len(managers))
	summary := &dto.StatusSummary{
		TotalManagers: len(managers),
	}

	for _, mgr := range managers {
		status := &dto.ManagerStatus{
			ID:             mgr.ID,
			Name:           mgr.Name,
			Type:           mgr.Type,
			Installed:      mgr.Installed,
			Version:        mgr.Version,
			Status:         mgr.Status,
			PackageCount:   mgr.PackageCount(),
			UpdatableCount: mgr.UpdatableCount(),
			BinaryPath:     mgr.BinaryPath,
			ConfigPath:     mgr.ConfigPath,
		}
		managerStatuses = append(managerStatuses, status)

		// Update summary
		if mgr.Installed {
			summary.InstalledManagers++
		}
		if mgr.IsHealthy() {
			summary.HealthyManagers++
		}
		summary.TotalPackages += mgr.PackageCount()
		summary.UpdatablePackages += mgr.UpdatableCount()
	}

	uc.logger.Info(ctx, "Status retrieved successfully",
		output.Field{Key: "total_managers", Value: summary.TotalManagers},
		output.Field{Key: "installed_managers", Value: summary.InstalledManagers},
	)

	return &dto.StatusResponse{
		Managers: managerStatuses,
		Summary:  summary,
	}, nil
}
