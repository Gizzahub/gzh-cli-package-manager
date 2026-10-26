package update

import (
	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/dto"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
)

func resolvedBoundedOptions(id manager.ManagerID, req *dto.UpdateRequest) adapterm.UpdateOptions {
	opts := (&UseCase{}).updateOptions(id, req)
	opts.DryRun = false
	return opts
}

func boundedResponse(state *boundedState, results []*dto.ManagerUpdateResult, dryRun bool) *dto.UpdateResponse {
	response := &dto.UpdateResponse{Results: results, Summary: &dto.UpdateSummary{}, DryRun: dryRun}
	managers := make(map[manager.ManagerID]bool)
	failed := make(map[manager.ManagerID]bool)
	processed := make(map[manager.ManagerID]bool)
	for _, item := range state.Items {
		managers[item.Manager] = true
		response.Progress = append(response.Progress, dto.BoundedPackageProgress{Manager: item.Manager, Package: item.Package, Status: item.Status, Error: item.Error})
		if item.Status == boundedPending || item.Status == boundedDeferred {
			response.Deferred = !dryRun
		}
	}
	for _, result := range results {
		processed[result.ID] = true
		if !result.Success {
			failed[result.ID] = true
		}
		if !dryRun {
			response.Summary.TotalPackagesUpdated += len(result.UpdatedPackages)
		}
		response.Summary.TotalDuration += result.Duration
	}
	response.Summary.TotalManagers = len(managers)
	for id := range managers {
		switch {
		case failed[id]:
			response.Summary.FailedManagers++
		case processed[id]:
			response.Summary.SuccessfulManagers++
		default:
			response.Summary.SkippedManagers++
		}
	}
	return response
}
