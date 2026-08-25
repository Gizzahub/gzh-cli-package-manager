// Package main is the entry point for the gz-pm CLI application.
//
// This package belongs to the presentation layer and is responsible for:
//   - CLI command definitions (using Cobra)
//   - Input validation and parsing
//   - Output formatting
//   - Dependency injection (wiring use cases with adapters)
//
// The presentation layer depends on application (use cases) and infrastructure (adapters)
// but contains minimal logic itself - it delegates to use cases.
//
// Design Principles:
//   - Thin layer - delegate to application use cases
//   - Handle only CLI-specific concerns
//   - Format output appropriately (enhanced, simple, JSON)
//   - Perform dependency injection in main()
package main
