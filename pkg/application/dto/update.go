package dto

import "github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"

// UpdateRequest contains parameters for updating package managers.
type UpdateRequest struct {
	// All indicates whether to update all detected managers.
	All bool

	// ManagerIDs specifies which managers to update.
	// If empty and All is false, no managers are updated.
	ManagerIDs []manager.ManagerID

	// DryRun indicates preview mode without executing updates.
	DryRun bool

	// Strategy specifies the update strategy to use.
	Strategy UpdateStrategy

	// CheckDuplicates enables duplicate binary detection.
	CheckDuplicates bool

	// PipAllowConda allows pip updates even in conda environments.
	// By default, pip updates in conda environments are skipped with a warning.
	PipAllowConda bool
}

// UpdateStrategy defines how packages should be updated.
type UpdateStrategy string

// Update strategies.
const (
	StrategyStable UpdateStrategy = "stable" // Latest stable release only (default)
	StrategyLatest UpdateStrategy = "latest" // Absolute latest (including beta/rc)
	StrategyMinor  UpdateStrategy = "minor"  // Latest minor/patch, no major upgrades
	StrategyFixed  UpdateStrategy = "fixed"  // Show available updates but don't install
)

// UpdateResponse contains the results of update operations.
type UpdateResponse struct {
	// Results contains the update result for each manager.
	Results []*ManagerUpdateResult

	// Summary provides aggregate statistics.
	Summary *UpdateSummary

	// DryRun indicates if this was a dry-run (preview only).
	DryRun bool
}

// ManagerUpdateResult represents the update result for a single manager.
type ManagerUpdateResult struct {
	// Manager ID and name.
	ID   manager.ManagerID
	Name string

	// Success indicates if the update completed successfully.
	Success bool

	// Skipped indicates if the manager was skipped (not an error).
	Skipped bool

	// SkipReason explains why the manager was skipped.
	SkipReason string

	// Error contains the error message if update failed.
	Error string

	// UpdatedPackages is the list of packages that were updated.
	UpdatedPackages []PackageUpdate

	// SkippedPackages is the list of packages that were skipped.
	SkippedPackages []string

	// PackageCorrelation reports whether updated package names could be joined
	// with a pre-update snapshot. Empty UpdatedPackages on a successful npm or
	// pacman update is unsupported correlation, not a fabricated package list.
	PackageCorrelation PackageCorrelation

	// MetadataPilot is true for npm and pacman, the MVP metadata-fidelity pilots.
	MetadataPilot bool

	// Duration is the time taken for the update in seconds.
	Duration float64

	// BytesDownloaded is the total bytes downloaded.
	BytesDownloaded int64

	// SpaceFreed is the disk space freed in bytes (cleanup operations).
	SpaceFreed int64
}

// FieldPresence reports whether a metadata field is an observed value,
// a value derived from observed inputs, or unavailable.
type FieldPresence string

const (
	// PresenceObserved means the value was read from command output.
	PresenceObserved FieldPresence = "observed"
	// PresenceDerived means the value was computed from observed versions.
	PresenceDerived FieldPresence = "derived"
	// PresenceUnavailable means no value was observed or reported.
	PresenceUnavailable FieldPresence = "unavailable"
)

// PackageCorrelation reports how update result names were joined with a
// pre-update ListPackages snapshot.
type PackageCorrelation string

const (
	// CorrelationJoined means every named package had both versions observed.
	CorrelationJoined PackageCorrelation = "joined"
	// CorrelationPartial means some named packages had observed version fields.
	CorrelationPartial PackageCorrelation = "partial"
	// CorrelationUnobserved means names were reported but no version metadata joined.
	CorrelationUnobserved PackageCorrelation = "unobserved"
	// CorrelationUnsupported means the update result did not report package names.
	CorrelationUnsupported PackageCorrelation = "unsupported"
	// CorrelationOutOfPilot means this manager is outside the npm/pacman MVP pilot.
	CorrelationOutOfPilot PackageCorrelation = "out_of_pilot"
	// CorrelationNotApplicable means no package metadata was expected (skip or error).
	CorrelationNotApplicable PackageCorrelation = "not_applicable"
)

// PackageUpdate represents a package update operation.
type PackageUpdate struct {
	// Name is the package name.
	Name string

	// OldVersion is the version before update.
	// Empty when OldVersionPresence is unavailable; it is not a placeholder.
	OldVersion string

	// NewVersion is the version after update.
	// Empty when NewVersionPresence is unavailable; it is not a placeholder.
	NewVersion string

	// UpdateType indicates the type of update.
	// Empty when UpdateTypePresence is unavailable; it is not a default minor update.
	UpdateType manager.UpdateType

	// SizeBytes is the download size in bytes.
	// Zero is meaningful only when SizeBytesPresence is observed.
	SizeBytes int64

	// OldVersionPresence distinguishes an observed old version from an unobserved one.
	OldVersionPresence FieldPresence

	// NewVersionPresence distinguishes an observed new version from an unobserved one.
	NewVersionPresence FieldPresence

	// UpdateTypePresence is derived when both versions were observed, otherwise unavailable.
	UpdateTypePresence FieldPresence

	// SizeBytesPresence is observed only when an adapter reports download size.
	SizeBytesPresence FieldPresence
}

// UnavailablePackageUpdate returns a named package row with no observed metadata.
func UnavailablePackageUpdate(name string) PackageUpdate {
	return PackageUpdate{
		Name:               name,
		OldVersionPresence: PresenceUnavailable,
		NewVersionPresence: PresenceUnavailable,
		UpdateTypePresence: PresenceUnavailable,
		SizeBytesPresence:  PresenceUnavailable,
	}
}

// UpdateSummary provides aggregate statistics for all updates.
type UpdateSummary struct {
	// TotalManagers is the total number of managers processed.
	TotalManagers int

	// SuccessfulManagers is the number of managers updated successfully.
	SuccessfulManagers int

	// FailedManagers is the number of managers that failed to update.
	FailedManagers int

	// SkippedManagers is the number of managers that were skipped.
	SkippedManagers int

	// TotalPackagesUpdated is the total number of packages updated.
	TotalPackagesUpdated int

	// TotalBytesDownloaded is the total bytes downloaded.
	TotalBytesDownloaded int64

	// TotalSpaceFreed is the total disk space freed in bytes.
	TotalSpaceFreed int64

	// TotalDuration is the total time taken in seconds.
	TotalDuration float64
}
