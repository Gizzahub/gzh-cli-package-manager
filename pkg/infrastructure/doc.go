// Package infrastructure contains adapters and implementations for external concerns.
//
// This layer implements the ports (interfaces) defined by the domain and application layers.
// It contains:
//   - Adapters: Package manager adapters (Homebrew, ASDF, npm, pip, etc.)
//   - Repositories: Data persistence implementations (YAML, SQLite)
//   - Executors: Command execution implementations
//   - Loggers: Logging implementations
//
// The infrastructure layer depends on domain and application (to implement their interfaces)
// but the reverse is not true - domain and application are unaware of infrastructure.
//
// Design Principles:
//   - Each adapter implements one or more port interfaces
//   - Adapters contain technology-specific code
//   - No business logic in adapters (business logic belongs in domain/application)
//   - Adapters can use external libraries and frameworks
package infrastructure
