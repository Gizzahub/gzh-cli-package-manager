// Package cleanup provides infrastructure implementations for cleanup operations.
//
// This package contains in-memory implementations of:
//   - QuarantineRepository: manages quarantined package records
//   - CacheRepository: tracks package manager cache statistics
//
// These implementations are suitable for testing and initial development.
// Production deployments may want to use persistent storage implementations.
package cleanup
