package manager

import "time"

// Manager represents a package manager installation.
// It contains metadata about the manager's state and configuration.
type Manager struct {
	// ID uniquely identifies the manager (e.g., "brew", "asdf")
	ID ManagerID

	// Name is the human-readable name ("Homebrew", "ASDF")
	Name string

	// Type categorizes the manager (system, version, language)
	Type ManagerType

	// Platform indicates the OS this manager runs on
	Platform Platform

	// Installed indicates whether the manager is currently installed
	Installed bool

	// Version is the currently installed version
	Version string

	// Status represents the health status
	Status Status

	// ConfigPath is the path to the manager's configuration file
	ConfigPath string

	// BinaryPath is the path to the manager's binary
	BinaryPath string

	// Packages is the list of packages managed by this manager
	Packages []Package

	// LastChecked is when the manager status was last verified
	LastChecked time.Time
}

// Package represents a package managed by a package manager.
type Package struct {
	// Name is the package name
	Name string

	// CurrentVersion is the currently installed version
	CurrentVersion string

	// AvailableVersion is the latest available version
	AvailableVersion string

	// Manager is the ID of the manager that installed this package
	Manager ManagerID

	// Description is a brief description of the package
	Description string

	// SizeMB is the installed size in megabytes
	SizeMB float64

	// UpdateType indicates the type of available update
	UpdateType UpdateType

	// IsGlobal indicates if this is a globally installed package
	IsGlobal bool
}

// UpdateType represents the semantic versioning update type.
type UpdateType string

// Update type classifications.
const (
	UpdateNone  UpdateType = "none"  // No update available
	UpdatePatch UpdateType = "patch" // Patch version update (x.x.X)
	UpdateMinor UpdateType = "minor" // Minor version update (x.X.x)
	UpdateMajor UpdateType = "major" // Major version update (X.x.x)
)

// IsUpdateAvailable returns true if an update is available for the package.
//
//nolint:gocritic // Preserve the exported value-receiver method set; no measured hot-path benefit justifies an API change.
func (p Package) IsUpdateAvailable() bool {
	return p.UpdateType != UpdateNone && p.AvailableVersion != "" && p.AvailableVersion != p.CurrentVersion
}

// IsHealthy returns true if the manager is in a healthy state.
//
//nolint:gocritic // Preserve the exported value-receiver method set; no measured hot-path benefit justifies an API change.
func (m Manager) IsHealthy() bool {
	return m.Status == StatusHealthy
}

// PackageCount returns the number of packages managed by this manager.
//
//nolint:gocritic // Preserve the exported value-receiver method set; no measured hot-path benefit justifies an API change.
func (m Manager) PackageCount() int {
	return len(m.Packages)
}

// UpdatableCount returns the number of packages with available updates.
//
//nolint:gocritic // Preserve the exported value-receiver method set; no measured hot-path benefit justifies an API change.
func (m Manager) UpdatableCount() int {
	count := 0
	for _, pkg := range m.Packages {
		if pkg.IsUpdateAvailable() {
			count++
		}
	}
	return count
}
