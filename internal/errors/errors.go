// Package errors provides package-manager-specific error types and utilities.
package errors

import (
	"errors"
	"fmt"
)

// Common sentinel errors for package manager operations.
var (
	// ErrManagerNotFound indicates a package manager was not found on the system.
	ErrManagerNotFound = errors.New("package manager not found")

	// ErrManagerNotInstalled indicates a package manager is not installed.
	ErrManagerNotInstalled = errors.New("package manager not installed")

	// ErrManagerNotSupported indicates the package manager is not supported.
	ErrManagerNotSupported = errors.New("package manager not supported")

	// ErrManagerUnavailable indicates the package manager is temporarily unavailable.
	ErrManagerUnavailable = errors.New("package manager unavailable")

	// ErrPackageNotFound indicates a package was not found.
	ErrPackageNotFound = errors.New("package not found")

	// ErrPackageAlreadyInstalled indicates a package is already installed.
	ErrPackageAlreadyInstalled = errors.New("package already installed")

	// ErrInvalidPackageName indicates an invalid package name.
	ErrInvalidPackageName = errors.New("invalid package name")

	// ErrInvalidVersion indicates an invalid version string.
	ErrInvalidVersion = errors.New("invalid version")

	// ErrUpdateFailed indicates a package manager update failed.
	ErrUpdateFailed = errors.New("update failed")

	// ErrInstallFailed indicates a package installation failed.
	ErrInstallFailed = errors.New("installation failed")

	// ErrUninstallFailed indicates a package uninstallation failed.
	ErrUninstallFailed = errors.New("uninstallation failed")

	// ErrUpgradeFailed indicates a package upgrade failed.
	ErrUpgradeFailed = errors.New("upgrade failed")

	// ErrCommandNotFound indicates a required command was not found.
	ErrCommandNotFound = errors.New("command not found")

	// ErrCommandFailed indicates a command execution failed.
	ErrCommandFailed = errors.New("command execution failed")

	// ErrPermissionDenied indicates insufficient permissions.
	ErrPermissionDenied = errors.New("permission denied")

	// ErrNetworkError indicates a network-related error.
	ErrNetworkError = errors.New("network error")

	// ErrTimeoutExceeded indicates an operation timeout.
	ErrTimeoutExceeded = errors.New("timeout exceeded")

	// ErrInvalidConfiguration indicates invalid configuration.
	ErrInvalidConfiguration = errors.New("invalid configuration")

	// ErrLockFileExists indicates a lock file already exists.
	ErrLockFileExists = errors.New("lock file exists")

	// ErrDependencyConflict indicates conflicting dependencies.
	ErrDependencyConflict = errors.New("dependency conflict")

	// ErrChecksumMismatch indicates a checksum verification failure.
	ErrChecksumMismatch = errors.New("checksum mismatch")
)

// ManagerError represents an error related to a specific package manager.
type ManagerError struct {
	Manager string // Name of the package manager (e.g., "brew", "apt", "pacman")
	Op      string // Operation that failed (e.g., "update", "install", "detect")
	Err     error  // Underlying error
}

// Error returns the error message.
func (e *ManagerError) Error() string {
	if e.Op != "" {
		return fmt.Sprintf("%s: %s: %v", e.Manager, e.Op, e.Err)
	}
	return fmt.Sprintf("%s: %v", e.Manager, e.Err)
}

// Unwrap returns the underlying error.
func (e *ManagerError) Unwrap() error {
	return e.Err
}

// PackageError represents an error related to a specific package.
type PackageError struct {
	Manager string // Package manager name
	Package string // Package name
	Op      string // Operation that failed
	Err     error  // Underlying error
}

// Error returns the error message.
func (e *PackageError) Error() string {
	if e.Op != "" {
		return fmt.Sprintf("%s: package '%s': %s: %v", e.Manager, e.Package, e.Op, e.Err)
	}
	return fmt.Sprintf("%s: package '%s': %v", e.Manager, e.Package, e.Err)
}

// Unwrap returns the underlying error.
func (e *PackageError) Unwrap() error {
	return e.Err
}

// CommandError represents an error from executing a system command.
type CommandError struct {
	Command  string // Command that failed
	ExitCode int    // Exit code of the command
	Stderr   string // Standard error output
	Err      error  // Underlying error
}

// Error returns the error message.
func (e *CommandError) Error() string {
	if e.Stderr != "" {
		return fmt.Sprintf("command '%s' failed (exit %d): %s: %v", e.Command, e.ExitCode, e.Stderr, e.Err)
	}
	return fmt.Sprintf("command '%s' failed (exit %d): %v", e.Command, e.ExitCode, e.Err)
}

// Unwrap returns the underlying error.
func (e *CommandError) Unwrap() error {
	return e.Err
}

// Wrap wraps an error with additional context.
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// Wrapf wraps an error with a formatted message.
func Wrapf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err)
}

// WithManager wraps an error with manager context.
func WithManager(manager string, op string, err error) error {
	if err == nil {
		return nil
	}
	return &ManagerError{
		Manager: manager,
		Op:      op,
		Err:     err,
	}
}

// WithPackage wraps an error with package context.
func WithPackage(manager string, pkg string, op string, err error) error {
	if err == nil {
		return nil
	}
	return &PackageError{
		Manager: manager,
		Package: pkg,
		Op:      op,
		Err:     err,
	}
}

// WithCommand wraps an error with command execution context.
func WithCommand(command string, exitCode int, stderr string, err error) error {
	if err == nil {
		return nil
	}
	return &CommandError{
		Command:  command,
		ExitCode: exitCode,
		Stderr:   stderr,
		Err:      err,
	}
}

// IsManagerError checks if an error is a ManagerError.
func IsManagerError(err error) bool {
	var mErr *ManagerError
	return errors.As(err, &mErr)
}

// IsPackageError checks if an error is a PackageError.
func IsPackageError(err error) bool {
	var pErr *PackageError
	return errors.As(err, &pErr)
}

// IsCommandError checks if an error is a CommandError.
func IsCommandError(err error) bool {
	var cErr *CommandError
	return errors.As(err, &cErr)
}

// IsNotFound checks if an error indicates a "not found" condition.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrManagerNotFound) ||
		errors.Is(err, ErrPackageNotFound) ||
		errors.Is(err, ErrCommandNotFound)
}

// IsPermissionError checks if an error is a permission error.
func IsPermissionError(err error) bool {
	return errors.Is(err, ErrPermissionDenied)
}

// IsNetworkError checks if an error is a network error.
func IsNetworkError(err error) bool {
	return errors.Is(err, ErrNetworkError)
}

// IsTimeoutError checks if an error is a timeout error.
func IsTimeoutError(err error) bool {
	return errors.Is(err, ErrTimeoutExceeded)
}
