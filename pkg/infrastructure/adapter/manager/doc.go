// Package manager provides adapters for various package managers.
//
// Each package manager (Homebrew, asdf, npm, pip, etc.) has its own
// adapter implementation in a subdirectory. All adapters implement
// the Adapter interface.
//
// Example structure:
//   - homebrew/: Homebrew adapter
//   - asdf/: ASDF adapter
//   - npm/: NPM adapter
package manager
