// Package cleanup provides domain types for package cleanup operations.
package cleanup

import "context"

// QuarantineRepository defines the interface for quarantine data access.
type QuarantineRepository interface {
	// List returns all quarantined packages.
	List(ctx context.Context) ([]*QuarantinedPackage, error)

	// ListByManager returns quarantined packages for a specific manager.
	ListByManager(ctx context.Context, managerID string) ([]*QuarantinedPackage, error)

	// Get returns a specific quarantined package.
	Get(ctx context.Context, name, version, managerID string) (*QuarantinedPackage, error)

	// Save persists a quarantined package record.
	Save(ctx context.Context, pkg *QuarantinedPackage) error

	// Delete removes a quarantined package record.
	Delete(ctx context.Context, name, version, managerID string) error

	// FindExpired returns packages that have exceeded the retention period.
	FindExpired(ctx context.Context, retentionDays int) ([]*QuarantinedPackage, error)
}

// CacheRepository defines the interface for cache metadata access.
type CacheRepository interface {
	// GetInfo returns cache statistics for a package manager.
	GetInfo(ctx context.Context, managerID string) (*CacheInfo, error)

	// ListAll returns cache info for all managers.
	ListAll(ctx context.Context) ([]*CacheInfo, error)

	// UpdateInfo updates cache statistics for a manager.
	UpdateInfo(ctx context.Context, info *CacheInfo) error
}

// OrphanDetector defines the interface for detecting orphan packages.
type OrphanDetector interface {
	// Detect finds orphan packages for a specific manager.
	Detect(ctx context.Context, managerID string) ([]*OrphanPackage, error)

	// DetectAll finds orphan packages across all managers.
	DetectAll(ctx context.Context) ([]*OrphanPackage, error)
}

// VersionScanner defines the interface for finding old package versions.
type VersionScanner interface {
	// Scan finds old versions for a specific package.
	Scan(ctx context.Context, name, managerID string) ([]*OldVersion, error)

	// ScanAll finds old versions across all packages for a manager.
	ScanAll(ctx context.Context, managerID string) ([]*OldVersion, error)
}

// Executor defines the interface for executing cleanup operations.
type Executor interface {
	// QuarantinePackage moves a package to quarantine.
	QuarantinePackage(ctx context.Context, name, version, managerID, reason string) error

	// RestorePackage restores a package from quarantine.
	RestorePackage(ctx context.Context, name, version, managerID string) error

	// PurgeQuarantined permanently removes quarantined packages.
	PurgeQuarantined(ctx context.Context, packages []*QuarantinedPackage) (*Summary, error)

	// ClearCache clears cache for a package manager.
	ClearCache(ctx context.Context, managerID string) (*Summary, error)

	// RemoveOrphans removes orphan packages.
	RemoveOrphans(ctx context.Context, packages []*OrphanPackage) (*Summary, error)

	// RemoveOldVersions removes old package versions.
	RemoveOldVersions(ctx context.Context, versions []*OldVersion) (*Summary, error)
}
