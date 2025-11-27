// Package manager provides the core domain models for package managers.
//
// This package contains:
//   - Manager: Entity representing a package manager installation
//   - Package: Entity representing a managed package
//   - Repository: Interface for data access (implemented in infrastructure layer)
//
// The domain layer has NO external dependencies (Go stdlib only) and contains
// pure business logic. This is the innermost layer in Clean Architecture.
package manager
