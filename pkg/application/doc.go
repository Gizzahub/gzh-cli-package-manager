// Package application contains use cases and application-specific business rules.
//
// This layer orchestrates the flow of data to and from entities and directs
// them to use the business rules. It contains:
//   - Use Cases: Application-specific business rules (UpdateAllManagers, Bootstrap, etc.)
//   - DTOs: Data Transfer Objects for cross-layer communication
//   - Input Ports: Interfaces defining use case entry points
//   - Output Ports: Interfaces for dependencies (repositories, executors, loggers)
//
// The application layer depends on the domain layer but not on infrastructure.
// It uses dependency injection - dependencies are injected via interfaces (ports).
//
// Design Principles:
//   - Use cases are independent and focused on single responsibilities
//   - DTOs prevent leaking domain models to presentation layer
//   - Ports (interfaces) define contracts with infrastructure
//   - No knowledge of presentation or infrastructure implementations
package application
