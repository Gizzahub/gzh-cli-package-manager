// Package apt provides an adapter for APT (Advanced Package Tool) package manager.
// It implements the manager.Adapter interface for detecting, querying,
// and managing packages through APT on Debian/Ubuntu systems.
package apt

import (
	"context"
	"fmt"
	"strings"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
)

const (
	aptCommand     = "apt"
	aptGetCommand  = "apt-get"
	whichCommand   = "which"
	checkCommand   = "check"
	listArg        = "list"
	installedFlag  = "--installed"
	upgradableFlag = "--upgradable"
)

// Adapter implements the manager.Adapter interface for APT.
type Adapter struct {
	executor output.CommandExecutor
	logger   output.Logger
}

// NewAdapter creates a new APT adapter.
func NewAdapter(executor output.CommandExecutor, logger output.Logger) *Adapter {
	return &Adapter{
		executor: executor,
		logger:   logger,
	}
}

// Detect checks if APT is installed on the system.
func (a *Adapter) Detect(ctx context.Context) (bool, error) {
	result, err := a.executor.Execute(ctx, whichCommand, aptCommand)
	if err != nil || result.ExitCode != 0 {
		return false, nil
	}
	return true, nil
}

// GetVersion retrieves the APT version.
func (a *Adapter) GetVersion(ctx context.Context) (string, error) {
	result, err := a.executor.Execute(ctx, aptCommand, "--version")
	if err != nil {
		return "", fmt.Errorf("failed to get apt version: %w", err)
	}

	// Parse version from output like "apt 2.4.11 (amd64)"
	lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	if len(lines) == 0 {
		return "", fmt.Errorf("empty version output")
	}

	firstLine := lines[0]
	parts := strings.Fields(firstLine)
	if len(parts) < 2 {
		return "", fmt.Errorf("unexpected version format: %s", firstLine)
	}

	return parts[1], nil
}

// GetBinaryPath returns the path to the apt binary.
func (a *Adapter) GetBinaryPath(ctx context.Context) (string, error) {
	result, err := a.executor.Execute(ctx, whichCommand, aptCommand)
	if err != nil {
		return "", fmt.Errorf("failed to locate apt binary: %w", err)
	}

	return strings.TrimSpace(result.Stdout), nil
}

// GetConfigPath returns the path to APT configuration.
func (a *Adapter) GetConfigPath(_ context.Context) (string, error) {
	// APT configuration is in /etc/apt
	return "/etc/apt", nil
}

// ListPackages retrieves all installed packages.
func (a *Adapter) ListPackages(ctx context.Context) ([]manager.Package, error) {
	// Get installed packages
	result, err := a.executor.Execute(ctx, aptCommand, listArg, installedFlag)
	if err != nil {
		return nil, fmt.Errorf("failed to list installed packages: %w", err)
	}

	// Parse installed packages
	installedMap := make(map[string]string)
	lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Listing...") {
			continue
		}

		// Format: "package/suite version architecture [status]"
		// Example: "vim/jammy-updates,now 2:8.2.3995-1ubuntu2.12 amd64 [installed]"
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		// Extract package name (before /)
		nameWithSuite := parts[0]
		nameParts := strings.Split(nameWithSuite, "/")
		if len(nameParts) == 0 {
			continue
		}
		pkgName := nameParts[0]

		// Extract version
		pkgVersion := parts[1]

		installedMap[pkgName] = pkgVersion
	}

	// Get upgradable packages
	upgradeResult, err := a.executor.Execute(ctx, aptCommand, listArg, upgradableFlag)
	upgradableMap := make(map[string]string)
	if err == nil && upgradeResult.ExitCode == 0 {
		upgradeLines := strings.Split(strings.TrimSpace(upgradeResult.Stdout), "\n")
		for _, line := range upgradeLines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "Listing...") {
				continue
			}

			parts := strings.Fields(line)
			if len(parts) < 2 {
				continue
			}

			nameWithSuite := parts[0]
			nameParts := strings.Split(nameWithSuite, "/")
			if len(nameParts) == 0 {
				continue
			}
			pkgName := nameParts[0]

			// Available version
			availableVersion := parts[1]
			upgradableMap[pkgName] = availableVersion
		}
	}

	// Build package list
	packages := make([]manager.Package, 0, len(installedMap))
	for pkgName, currentVersion := range installedMap {
		pkg := manager.Package{
			Name:           pkgName,
			CurrentVersion: currentVersion,
			IsGlobal:       true, // APT packages are always system-wide
			UpdateType:     manager.UpdateNone,
		}

		// Check if update available
		if availableVersion, hasUpdate := upgradableMap[pkgName]; hasUpdate {
			pkg.AvailableVersion = availableVersion
			pkg.UpdateType = manager.UpdateMinor // APT doesn't distinguish update types
		}

		packages = append(packages, pkg)
	}

	return packages, nil
}

// CheckHealth verifies APT system health.
func (a *Adapter) CheckHealth(ctx context.Context) (manager.Status, error) {
	// Check for broken packages or lock files
	checkResult, err := a.executor.Execute(ctx, aptGetCommand, checkCommand)
	if err != nil {
		a.logger.Warn(ctx, "Failed to run apt-get check", output.Field{Key: "error", Value: err.Error()})
		return manager.StatusDegraded, nil
	}

	if checkResult.ExitCode != 0 {
		return manager.StatusDegraded, nil
	}

	// Check for dpkg lock file
	lockCheckResult, _ := a.executor.Execute(ctx, "test", "-f", "/var/lib/dpkg/lock-frontend")
	if lockCheckResult != nil && lockCheckResult.ExitCode == 0 {
		a.logger.Warn(ctx, "APT lock file exists", output.Field{Key: "lock_file", Value: "/var/lib/dpkg/lock-frontend"})
		return manager.StatusDegraded, nil
	}

	return manager.StatusHealthy, nil
}
