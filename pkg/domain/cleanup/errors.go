// Package cleanup provides domain types for package cleanup operations.
package cleanup

import "errors"

// ErrPackageNotFound is returned when a quarantined package is not found.
var ErrPackageNotFound = errors.New("quarantined package not found")

// ErrAlreadyQuarantined is returned when trying to quarantine an already quarantined package.
var ErrAlreadyQuarantined = errors.New("package is already quarantined")

// ErrQuarantineFailed is returned when quarantine operation fails.
var ErrQuarantineFailed = errors.New("failed to quarantine package")

// ErrRestoreFailed is returned when restore operation fails.
var ErrRestoreFailed = errors.New("failed to restore package")

// ErrCacheClearFailed is returned when cache clear operation fails.
var ErrCacheClearFailed = errors.New("failed to clear cache")

// ErrInvalidRetentionDays is returned when retention days is invalid.
var ErrInvalidRetentionDays = errors.New("retention days must be positive")
