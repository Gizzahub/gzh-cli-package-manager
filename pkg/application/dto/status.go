package dto

import "github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"

// StatusRequest contains parameters for retrieving manager status.
type StatusRequest struct {
	// Verbose indicates whether to include detailed information.
	Verbose bool

	// Refresh forces a status refresh instead of using cached data.
	Refresh bool

	// ManagerIDs filters the status to specific managers.
	// If empty, all managers are included.
	ManagerIDs []manager.ManagerID
}

// StatusResponse contains the status of package managers.
type StatusResponse struct {
	// Managers is the list of package managers and their status.
	Managers []*ManagerStatus

	// Summary provides aggregate statistics.
	Summary *StatusSummary
}

// ManagerStatus represents the status of a single package manager.
type ManagerStatus struct {
	// ID is the manager identifier.
	ID manager.ManagerID

	// Name is the human-readable name.
	Name string

	// Type is the manager type (system, version, language).
	Type manager.ManagerType

	// Installed indicates if the manager is installed.
	Installed bool

	// Version is the installed version (empty if not installed).
	Version string

	// Status is the health status.
	Status manager.Status

	// PackageCount is the number of managed packages.
	PackageCount int

	// UpdatableCount is the number of packages with available updates.
	UpdatableCount int

	// BinaryPath is the path to the manager binary.
	BinaryPath string

	// ConfigPath is the path to the manager configuration.
	ConfigPath string

	// Packages is the list of packages (only populated in verbose mode).
	Packages []PackageInfo
}

// PackageInfo contains information about a package.
type PackageInfo struct {
	// Name is the package name.
	Name string

	// CurrentVersion is the installed version.
	CurrentVersion string

	// AvailableVersion is the latest available version (empty if no update).
	AvailableVersion string

	// UpdateType indicates the type of update (none, patch, minor, major).
	UpdateType manager.UpdateType

	// Description is a brief description of the package.
	Description string
}

// StatusSummary provides aggregate statistics across all managers.
type StatusSummary struct {
	// TotalManagers is the total number of supported managers.
	TotalManagers int

	// InstalledManagers is the number of installed managers.
	InstalledManagers int

	// HealthyManagers is the number of healthy managers.
	HealthyManagers int

	// TotalPackages is the total number of managed packages.
	TotalPackages int

	// UpdatablePackages is the total number of packages with updates.
	UpdatablePackages int
}
