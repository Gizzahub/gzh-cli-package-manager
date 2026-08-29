// Package cargo provides an adapter for Cargo (Rust Package Manager).
// It implements the manager.Adapter interface for detecting, querying,
// and managing Rust packages through cargo.
package cargo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
)

const (
	cargoCommand = "cargo"
	whichCommand = "which"
	installFlag  = "install"
	listFlag     = "--list"
)

// Adapter implements the manager.Adapter interface for Cargo.
type Adapter struct {
	executor        output.CommandExecutor
	logger          output.Logger
	homeDirResolver *homeDirResolver
}

type homeDirResolver struct {
	resolve func() (string, error)
}

// NewAdapter creates a new Cargo adapter.
func NewAdapter(executor output.CommandExecutor, logger output.Logger) *Adapter {
	return &Adapter{
		executor: executor,
		logger:   logger,
		homeDirResolver: &homeDirResolver{
			resolve: os.UserHomeDir,
		},
	}
}

func (a *Adapter) resolveUserHomeDir() (string, error) {
	if a.homeDirResolver == nil || a.homeDirResolver.resolve == nil {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}

		return homeDir, nil
	}

	return a.homeDirResolver.resolve()
}

// Detect checks if Cargo is installed on the system.
func (a *Adapter) Detect(ctx context.Context) (bool, error) {
	result, err := a.executor.Execute(ctx, whichCommand, cargoCommand)
	if err != nil || result.ExitCode != 0 {
		return false, nil
	}
	return true, nil
}

// GetVersion retrieves the Cargo version.
func (a *Adapter) GetVersion(ctx context.Context) (string, error) {
	result, err := a.executor.Execute(ctx, cargoCommand, "--version")
	if err != nil {
		return "", fmt.Errorf("failed to get cargo version: %w", err)
	}

	// Parse version from output like "cargo 1.75.0 (1d8b05cdd 2023-11-20)"
	parts := strings.Fields(strings.TrimSpace(result.Stdout))
	if len(parts) < 2 {
		return "", fmt.Errorf("unexpected version format: %s", result.Stdout)
	}

	return parts[1], nil
}

// GetBinaryPath returns the path to the cargo binary.
func (a *Adapter) GetBinaryPath(ctx context.Context) (string, error) {
	result, err := a.executor.Execute(ctx, whichCommand, cargoCommand)
	if err != nil {
		return "", fmt.Errorf("failed to locate cargo binary: %w", err)
	}

	return strings.TrimSpace(result.Stdout), nil
}

// GetConfigPath returns the path to Cargo configuration.
func (a *Adapter) GetConfigPath(_ context.Context) (string, error) {
	// Cargo config is in ~/.cargo/config.toml or ~/.cargo/config
	homeDir, err := a.resolveUserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory for Cargo configuration: %w", err)
	}

	configPath := filepath.Join(homeDir, ".cargo", "config.toml")
	if _, err := os.Stat(configPath); err == nil {
		return configPath, nil
	}

	// Fallback to config without .toml extension
	return filepath.Join(homeDir, ".cargo", "config"), nil
}

// ListPackages retrieves all installed packages.
func (a *Adapter) ListPackages(ctx context.Context) ([]manager.Package, error) {
	// cargo install --list shows installed binaries
	result, err := a.executor.Execute(ctx, cargoCommand, installFlag, listFlag)
	if err != nil {
		return nil, fmt.Errorf("failed to list installed packages: %w", err)
	}

	// Parse output format:
	// package-name v1.2.3:
	//     binary1
	//     binary2
	// another-package v0.5.0 (/path/to/local):
	//     binary3

	packages := make([]manager.Package, 0)
	lines := strings.SplitSeq(strings.TrimSpace(result.Stdout), "\n")

	for line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, " ") {
			// Skip binary names (indented lines)
			continue
		}

		// Package line format: "package-name v1.2.3:" or "package-name v1.2.3 (/path):"
		if !strings.HasSuffix(line, ":") {
			continue
		}

		// Remove trailing colon
		line = strings.TrimSuffix(line, ":")

		// Split by space
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		pkgName := parts[0]
		pkgVersion := strings.TrimPrefix(parts[1], "v")

		// Check if it's a local path install (contains parentheses)
		isLocal := strings.Contains(line, "(")

		pkg := manager.Package{
			Name:           pkgName,
			CurrentVersion: pkgVersion,
			IsGlobal:       !isLocal, // Local path installs are not global
			UpdateType:     manager.UpdateNone,
		}

		// Note: cargo install --list doesn't show available updates
		// Users need to manually check crates.io or use cargo-update plugin

		packages = append(packages, pkg)
	}

	return packages, nil
}

// CheckHealth verifies Cargo system health.
func (a *Adapter) CheckHealth(ctx context.Context) (manager.Status, error) {
	// Check if cargo can execute basic commands
	result, err := a.executor.Execute(ctx, cargoCommand, "--version")
	if err != nil || result.ExitCode != 0 {
		a.logger.Warn(ctx, "Failed to run cargo --version", output.Field{Key: "error", Value: err})
		return manager.StatusDegraded, nil
	}

	return manager.StatusHealthy, nil
}

// Update performs update operations (stub implementation).
// Update runs cargo install-update -a when cargo-update is available,
// otherwise cargo install --list based self-update is not attempted.
func (a *Adapter) Update(ctx context.Context, opts adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
	a.logger.Info(ctx, "Starting cargo update",
		output.Field{Key: "dry_run", Value: opts.DryRun},
		output.Field{Key: "strategy", Value: string(opts.Strategy)})

	result := &adapterm.UpdateResult{
		Success:         true,
		UpdatedPackages: []string{},
		FailedPackages:  []string{},
	}

	if opts.DryRun {
		result.Message = "Dry-run: would run cargo install-update -a"
		return result, nil
	}
	if opts.Strategy == adapterm.StrategyFixed {
		result.Message = "Strategy 'fixed': cargo update skipped"
		return result, nil
	}

	// cargo-update plugin: cargo install-update -a
	execResult, err := a.executor.Execute(ctx, "cargo", "install-update", "-a")
	if err != nil {
		// Fallback: cargo update in workspace is not global; report clear error
		result.Success = false
		result.Message = "cargo install-update not available; install cargo-update crate for global binary updates"
		return result, fmt.Errorf("cargo update failed: %w", err)
	}
	if execResult.ExitCode != 0 {
		result.Success = false
		result.Message = fmt.Sprintf("cargo install-update failed: %s", execResult.Stderr)
		return result, nil
	}
	result.Message = "cargo global packages updated via install-update"
	return result, nil
}
