package update

import (
	"context"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/dto"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/version"
)

func isMetadataPilot(id manager.ManagerID) bool {
	switch id {
	case manager.ManagerNPM, manager.ManagerPacman:
		return true
	default:
		return false
	}
}

func (uc *UseCase) preUpdateSnapshot(
	ctx context.Context,
	adapter adapterm.Adapter,
	mgr *manager.Manager,
	dryRun bool,
) map[string]manager.Package {
	// Dry-run update must not invoke extra package-manager commands
	// (TASK-120 apt/pacman dry-run executor-zero contract).
	if dryRun || !isMetadataPilot(mgr.ID) {
		return nil
	}

	packages, err := adapter.ListPackages(ctx)
	if err != nil {
		uc.logger.Warn(ctx, "Pre-update package snapshot unavailable",
			output.Field{Key: managerFieldKey, Value: mgr.Name},
			output.Field{Key: "error", Value: err.Error()},
		)
		return nil
	}

	snapshot := make(map[string]manager.Package, len(packages))
	for _, pkg := range packages {
		snapshot[pkg.Name] = pkg
	}
	return snapshot
}

func applyUpdateMetadata(
	result *dto.ManagerUpdateResult,
	id manager.ManagerID,
	snapshot map[string]manager.Package,
	updateResult *adapterm.UpdateResult,
) {
	if !isMetadataPilot(id) {
		result.PackageCorrelation = dto.CorrelationOutOfPilot
		for _, name := range updateResult.UpdatedPackages {
			result.UpdatedPackages = append(result.UpdatedPackages, dto.UnavailablePackageUpdate(name))
		}
		return
	}

	if len(updateResult.UpdatedPackages) == 0 {
		result.PackageCorrelation = dto.CorrelationUnsupported
		return
	}

	joined, partial, unobserved := 0, 0, 0
	for _, name := range updateResult.UpdatedPackages {
		pkg, ok := snapshot[name]
		if !ok {
			unobserved++
			result.UpdatedPackages = append(result.UpdatedPackages, dto.UnavailablePackageUpdate(name))
			continue
		}

		update := packageUpdateFromSnapshot(name, pkg)
		result.UpdatedPackages = append(result.UpdatedPackages, update)
		switch classifyPackageMetadata(update) {
		case dto.CorrelationJoined:
			joined++
		case dto.CorrelationPartial:
			partial++
		default:
			unobserved++
		}
	}

	result.PackageCorrelation = correlationFromCounts(joined, partial, unobserved)
}

func packageUpdateFromSnapshot(name string, pkg manager.Package) dto.PackageUpdate {
	update := dto.UnavailablePackageUpdate(name)
	if pkg.CurrentVersion != "" {
		update.OldVersion = pkg.CurrentVersion
		update.OldVersionPresence = dto.PresenceObserved
	}
	if pkg.AvailableVersion != "" {
		update.NewVersion = pkg.AvailableVersion
		update.NewVersionPresence = dto.PresenceObserved
	}
	if update.OldVersionPresence == dto.PresenceObserved && update.NewVersionPresence == dto.PresenceObserved {
		update.UpdateType = version.DetermineUpdateType(update.OldVersion, update.NewVersion)
		update.UpdateTypePresence = dto.PresenceDerived
	}
	return update
}

func classifyPackageMetadata(update dto.PackageUpdate) dto.PackageCorrelation {
	oldObserved := update.OldVersionPresence == dto.PresenceObserved
	newObserved := update.NewVersionPresence == dto.PresenceObserved
	switch {
	case oldObserved && newObserved:
		return dto.CorrelationJoined
	case oldObserved || newObserved:
		return dto.CorrelationPartial
	default:
		return dto.CorrelationUnobserved
	}
}

func correlationFromCounts(joined, partial, unobserved int) dto.PackageCorrelation {
	switch {
	case partial == 0 && unobserved == 0 && joined > 0:
		return dto.CorrelationJoined
	case joined > 0 || partial > 0:
		return dto.CorrelationPartial
	default:
		return dto.CorrelationUnobserved
	}
}
