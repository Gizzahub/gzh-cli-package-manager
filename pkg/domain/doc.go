// Package domain contains the core business logic and entities.
//
// This layer is the heart of the application, containing:
//   - Entities: Core business objects (Manager, Package, Config)
//   - Value Objects: Immutable objects representing concepts
//   - Domain Services: Business logic that doesn't belong to a single entity
//   - Repository Interfaces: Contracts for data persistence (implemented in infrastructure)
//
// The domain layer has NO external dependencies - only Go stdlib.
// It defines interfaces that outer layers implement.
//
// Design Principles:
//   - Pure functions where possible
//   - No dependencies on frameworks or external libraries
//   - No knowledge of use cases, presentation, or infrastructure
//   - Repository pattern for data access abstraction
package domain
