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

	// Duration is the time taken for the update in seconds.
	Duration float64

	// BytesDownloaded is the total bytes downloaded.
	BytesDownloaded int64

	// SpaceFreed is the disk space freed in bytes (cleanup operations).
	SpaceFreed int64
}

// PackageUpdate represents a package update operation.
type PackageUpdate struct {
	// Name is the package name.
	Name string

	// OldVersion is the version before update.
	OldVersion string

	// NewVersion is the version after update.
	NewVersion string

	// UpdateType indicates the type of update.
	UpdateType manager.UpdateType

	// SizeBytes is the download size in bytes.
	SizeBytes int64
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
