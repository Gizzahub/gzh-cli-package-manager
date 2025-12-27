// Package cleanup provides domain types for package cleanup operations.
//
// This package contains domain entities and interfaces for:
//   - Quarantine management: safely isolating packages before removal
//   - Cache management: tracking and clearing package manager caches
//   - Orphan detection: finding unused dependency packages
//   - Version management: tracking old package versions
//
// The cleanup domain follows Clean Architecture principles:
//   - Types and interfaces defined here, implementations in infrastructure
//   - No external dependencies (stdlib only)
//   - Business logic in domain entities
package cleanup
