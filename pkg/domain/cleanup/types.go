// Package cleanup provides domain types for package cleanup operations.
package cleanup

import "time"

// QuarantineStatus represents the state of a quarantined package.
type QuarantineStatus string

const (
	// StatusQuarantined indicates the package is in quarantine.
	StatusQuarantined QuarantineStatus = "quarantined"

	// StatusRestored indicates the package was restored from quarantine.
	StatusRestored QuarantineStatus = "restored"

	// StatusPurged indicates the package was permanently removed.
	StatusPurged QuarantineStatus = "purged"
)

// QuarantinedPackage represents a package that has been quarantined.
type QuarantinedPackage struct {
	// Name is the package name
	Name string

	// Version is the quarantined version
	Version string

	// ManagerID is the package manager that owned this package
	ManagerID string

	// QuarantinedAt is when the package was quarantined
	QuarantinedAt time.Time

	// Reason explains why the package was quarantined
	Reason string

	// Status is the current quarantine status
	Status QuarantineStatus

	// BackupPath is where the package data is stored
	BackupPath string

	// SizeMB is the size of the quarantined data in megabytes
	SizeMB float64
}

// CacheInfo represents cache statistics for a package manager.
type CacheInfo struct {
	// ManagerID is the package manager ID
	ManagerID string

	// CachePath is the path to the cache directory
	CachePath string

	// TotalSizeMB is the total cache size in megabytes
	TotalSizeMB float64

	// EntryCount is the number of cached entries
	EntryCount int

	// OldestEntry is the timestamp of the oldest cache entry
	OldestEntry time.Time

	// NewestEntry is the timestamp of the newest cache entry
	NewestEntry time.Time
}

// OrphanPackage represents a package that is no longer needed.
type OrphanPackage struct {
	// Name is the package name
	Name string

	// Version is the installed version
	Version string

	// ManagerID is the package manager ID
	ManagerID string

	// InstalledAt is when the package was installed (if known)
	InstalledAt time.Time

	// SizeMB is the package size in megabytes
	SizeMB float64

	// DependentCount is the number of packages depending on this one
	DependentCount int

	// Reason explains why this is considered an orphan
	Reason string
}

// OldVersion represents an old version of a package.
type OldVersion struct {
	// Name is the package name
	Name string

	// Version is the version string
	Version string

	// ManagerID is the package manager ID
	ManagerID string

	// InstalledAt is when this version was installed
	InstalledAt time.Time

	// SizeMB is the size of this version in megabytes
	SizeMB float64

	// IsCurrent indicates if this is the currently active version
	IsCurrent bool
}

// Summary contains statistics about a cleanup operation.
type Summary struct {
	// PackagesRemoved is the count of packages removed
	PackagesRemoved int

	// SpaceFreedMB is the disk space freed in megabytes
	SpaceFreedMB float64

	// Duration is how long the cleanup took
	Duration time.Duration

	// Errors lists any errors encountered
	Errors []string
}

// IsExpired returns true if the quarantined package has exceeded the retention period.
//
//nolint:gocritic // Preserve the exported value-receiver method set; no measured hot-path benefit justifies an API change.
func (q QuarantinedPackage) IsExpired(retentionDays int) bool {
	expiry := q.QuarantinedAt.AddDate(0, 0, retentionDays)
	return time.Now().After(expiry)
}

// DaysSinceQuarantine returns the number of days since quarantine.
//
//nolint:gocritic // Preserve the exported value-receiver method set; no measured hot-path benefit justifies an API change.
func (q QuarantinedPackage) DaysSinceQuarantine() int {
	return int(time.Since(q.QuarantinedAt).Hours() / 24)
}

// AgeInDays returns the age of the oldest cache entry in days.
//
//nolint:gocritic // Preserve the exported value-receiver method set; no measured hot-path benefit justifies an API change.
func (c CacheInfo) AgeInDays() int {
	if c.OldestEntry.IsZero() {
		return 0
	}
	return int(time.Since(c.OldestEntry).Hours() / 24)
}
