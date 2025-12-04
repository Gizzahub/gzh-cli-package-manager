// Package pacman provides an adapter for Pacman package manager.
// It implements the manager.Adapter interface for detecting, querying,
// and managing packages through Pacman (Arch Linux package manager).
package pacman

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/cmdutil"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/version"
)

// Adapter implements the manager.Adapter interface for Pacman.
type Adapter struct {
	executor output.CommandExecutor
	logger   output.Logger
}

// NewAdapter creates a new Pacman adapter.
func NewAdapter(executor output.CommandExecutor, logger output.Logger) *Adapter {
	return &Adapter{
		executor: executor,
		logger:   logger,
	}
}

// Detect checks if Pacman is installed on the system.
func (a *Adapter) Detect(ctx context.Context) (bool, error) {
	result, err := a.executor.Execute(ctx, "which", "pacman")
	return cmdutil.IsCommandAvailable(result, err), nil
}

// GetVersion retrieves the version of the installed Pacman.
func (a *Adapter) GetVersion(ctx context.Context) (string, error) {
	result, err := a.executor.Execute(ctx, "pacman", "--version")
	if err := cmdutil.CheckResult(result, err, "get pacman version"); err != nil {
		return "", err
	}

	// Output format: " .--.                  Pacman v7.0.0 - libalpm v15.0.0"
	// Extract version using regex
	re := regexp.MustCompile(`Pacman v(\d+\.\d+\.\d+)`)
	matches := re.FindStringSubmatch(result.Stdout)
	if len(matches) < 2 {
		return "", fmt.Errorf("unexpected version format: %s", result.Stdout)
	}

	return matches[1], nil
}

// GetBinaryPath returns the path to the Pacman binary.
func (a *Adapter) GetBinaryPath(ctx context.Context) (string, error) {
	result, err := a.executor.Execute(ctx, "which", "pacman")
	if err := cmdutil.CheckResult(result, err, "find pacman binary"); err != nil {
		return "", err
	}
	return cmdutil.ExtractStdout(result), nil
}

// GetConfigPath returns the path to the Pacman configuration.
func (a *Adapter) GetConfigPath(_ context.Context) (string, error) {
	// Pacman configuration is always at /etc/pacman.conf
	return "/etc/pacman.conf", nil
}

// ListPackages retrieves all packages managed by Pacman.
func (a *Adapter) ListPackages(ctx context.Context) ([]manager.Package, error) {
	// Get list of installed packages with versions
	result, err := a.executor.Execute(ctx, "pacman", "-Q")
	if err := cmdutil.CheckResult(result, err, "list pacman packages"); err != nil {
		return nil, err
	}

	// Get list of packages with available updates
	updatesResult, err := a.executor.Execute(ctx, "pacman", "-Qu")
	if err != nil {
		// pacman -Qu returns exit code 1 if no updates, which is not an error
		updatesResult = &output.ExecutionResult{Stdout: ""}
	}

	// Parse update information into a map
	updateMap := make(map[string]string)
	if updatesResult.ExitCode == 0 {
		updateLines := strings.Split(strings.TrimSpace(updatesResult.Stdout), "\n")
		for _, line := range updateLines {
			if line == "" {
				continue
			}
			// Format: "package current -> available"
			parts := strings.Fields(line)
			if len(parts) >= 4 && parts[2] == "->" {
				updateMap[parts[0]] = parts[3]
			}
		}
	}

	// Parse installed packages
	lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	packages := make([]manager.Package, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}

		// Format: "package-name version"
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		pkgName := parts[0]
		pkgVersion := parts[1]

		pkg := manager.Package{
			Name:           pkgName,
			CurrentVersion: pkgVersion,
			Description:    "", // Pacman -Q doesn't provide descriptions
			IsGlobal:       true,
			UpdateType:     manager.UpdateNone,
		}

		// Check if update is available
		if availableVersion, hasUpdate := updateMap[pkgName]; hasUpdate {
			pkg.AvailableVersion = availableVersion
			pkg.UpdateType = version.DetermineUpdateType(pkgVersion, availableVersion)
		}

		packages = append(packages, pkg)
	}

	return packages, nil
}

// CheckHealth performs health checks on Pacman.
func (a *Adapter) CheckHealth(ctx context.Context) (manager.Status, error) {
	// Check if pacman database is accessible
	result, err := a.executor.Execute(ctx, "pacman", "-Qq")
	if err != nil {
		a.logger.Warn(ctx, "Failed to query pacman database",
			output.Field{Key: "error", Value: err.Error()})
		return manager.StatusDegraded, nil
	}

	if result.ExitCode != 0 {
		return manager.StatusError, nil
	}

	// Additional check: verify database lock is not present
	// (could indicate pacman is already running or crashed)
	lockCheckResult, err := a.executor.Execute(ctx, "test", "-f", "/var/lib/pacman/db.lck")
	if err == nil && lockCheckResult.ExitCode == 0 {
		a.logger.Warn(ctx, "Pacman database lock file exists",
			output.Field{Key: "lockfile", Value: "/var/lib/pacman/db.lck"})
		return manager.StatusDegraded, nil
	}

	return manager.StatusHealthy, nil
}

// Update performs update operations (stub implementation).
func (a *Adapter) Update(ctx context.Context, opts adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
	a.logger.Warn(ctx, "Update method not yet implemented for this adapter")
	return &adapterm.UpdateResult{
		Success: false,
		Message: "Update not yet implemented for this package manager",
	}, fmt.Errorf("update not yet implemented")
}
