// Package chocolatey provides an adapter for the Chocolatey package manager.
// It implements the manager.Adapter interface for detecting, querying,
// and managing packages through Chocolatey on Windows systems.
// Chocolatey is a machine-level package manager for Windows that manages
// software using .nupkg packages, typically requiring admin rights.
package chocolatey

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/cmdutil"
)

const (
	chocoCommand = "choco"
)

// Adapter implements the manager.Adapter interface for Chocolatey.
type Adapter struct {
	executor output.CommandExecutor
	logger   output.Logger
}

// NewAdapter creates a new Chocolatey adapter.
func NewAdapter(executor output.CommandExecutor, logger output.Logger) *Adapter {
	return &Adapter{
		executor: executor,
		logger:   logger,
	}
}

// Detect checks if Chocolatey is installed on the system.
func (a *Adapter) Detect(ctx context.Context) (bool, error) {
	result, err := a.executor.Execute(ctx, chocoCommand, "--version")
	return cmdutil.IsCommandAvailable(result, err), nil
}

// GetVersion retrieves the version of the installed Chocolatey.
func (a *Adapter) GetVersion(ctx context.Context) (string, error) {
	result, err := a.executor.Execute(ctx, chocoCommand, "--version")
	if err := cmdutil.CheckResult(result, err, "get chocolatey version"); err != nil {
		return "", err
	}

	// Output format: "2.2.2" or "2.2.2\n"
	version := strings.TrimSpace(cmdutil.ExtractStdout(result))
	if version == "" {
		return "unknown", nil
	}
	return version, nil
}

// GetBinaryPath returns the path to the Chocolatey binary.
func (a *Adapter) GetBinaryPath(ctx context.Context) (string, error) {
	// On Windows, use 'where' command
	result, err := a.executor.Execute(ctx, "where", chocoCommand)
	if err := cmdutil.CheckResult(result, err, "find chocolatey binary"); err != nil {
		return "", err
	}

	// 'where' may return multiple lines; take the first one
	lines := strings.Split(cmdutil.ExtractStdout(result), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return "", fmt.Errorf("find chocolatey binary: unexpected empty output")
	}
	return strings.TrimSpace(lines[0]), nil
}

// GetConfigPath returns the path to the Chocolatey installation directory.
func (a *Adapter) GetConfigPath(_ context.Context) (string, error) {
	// Check ChocolateyInstall environment variable first
	if chocoDir := os.Getenv("ChocolateyInstall"); chocoDir != "" {
		return chocoDir, nil
	}

	// Default to C:\ProgramData\chocolatey
	return filepath.Join(os.Getenv("ProgramData"), "chocolatey"), nil
}

// ListPackages retrieves all packages managed by Chocolatey.
func (a *Adapter) ListPackages(ctx context.Context) ([]manager.Package, error) {
	// Use 'choco list -r' for machine-readable output
	result, err := a.executor.Execute(ctx, chocoCommand, "list", "-r")
	if err := cmdutil.CheckResult(result, err, "list chocolatey packages"); err != nil {
		return nil, err
	}

	return a.parseListOutput(result), nil
}

// parseListOutput parses the machine-readable output from 'choco list -r'.
// Format: "package|version" per line
func (a *Adapter) parseListOutput(result *output.ExecutionResult) []manager.Package {
	lines := strings.Split(result.Stdout, "\n")
	var packages []manager.Package

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse "package|version" format
		parts := strings.Split(line, "|")
		if len(parts) >= 2 {
			pkg := manager.Package{
				Name:           parts[0],
				CurrentVersion: parts[1],
				IsGlobal:       true, // Chocolatey installs system-wide
				UpdateType:     manager.UpdateNone,
			}
			packages = append(packages, pkg)
		}
	}

	return packages
}

// Search finds packages matching query via `choco search -r`.
func (a *Adapter) Search(ctx context.Context, query string) ([]manager.Package, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("search chocolatey packages: query is required")
	}

	// Machine-readable: package|version per line
	result, err := a.executor.Execute(ctx, chocoCommand, "search", query, "-r")
	if err := cmdutil.CheckResult(result, err, "search chocolatey packages"); err != nil {
		return nil, err
	}

	packages := a.parseListOutput(result)
	for i := range packages {
		packages[i].Manager = manager.ManagerChocolatey
	}
	return packages, nil
}

// Install installs a package by Chocolatey package ID.
// dryRun skips the native install. Elevation-related errors are wrapped with
// a clear message suggesting the user re-run as administrator.
func (a *Adapter) Install(ctx context.Context, pkgID string, dryRun bool) error {
	pkgID = strings.TrimSpace(pkgID)
	if pkgID == "" {
		return fmt.Errorf("install chocolatey package: package id is required")
	}

	a.logger.Info(ctx, "Chocolatey install",
		output.Field{Key: "package", Value: pkgID},
		output.Field{Key: "dry_run", Value: dryRun})

	if dryRun {
		return nil
	}

	result, err := a.executor.Execute(ctx, chocoCommand, "install", pkgID, "-y")
	if err := cmdutil.CheckResult(result, err, "install chocolatey package "+pkgID); err != nil {
		return wrapElevationError(err)
	}
	return nil
}

// Uninstall removes a package by Chocolatey package ID.
// dryRun skips the native uninstall. Elevation-related errors are wrapped.
func (a *Adapter) Uninstall(ctx context.Context, pkgID string, dryRun bool) error {
	pkgID = strings.TrimSpace(pkgID)
	if pkgID == "" {
		return fmt.Errorf("uninstall chocolatey package: package id is required")
	}

	a.logger.Info(ctx, "Chocolatey uninstall",
		output.Field{Key: "package", Value: pkgID},
		output.Field{Key: "dry_run", Value: dryRun})

	if dryRun {
		return nil
	}

	result, err := a.executor.Execute(ctx, chocoCommand, "uninstall", pkgID, "-y")
	if err := cmdutil.CheckResult(result, err, "uninstall chocolatey package "+pkgID); err != nil {
		return wrapElevationError(err)
	}
	return nil
}

// wrapElevationError rewrites admin/UAC-related failures with a clear suggestion.
// No real UAC elevation is attempted.
func wrapElevationError(err error) error {
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	elevationHints := []string{
		"access is denied",
		"access denied",
		"requires elevation",
		"elevation required",
		"administrator",
		"run as admin",
		"not running in an elevated",
		"uac",
		"privileged",
		"permission denied",
		"must be run as an administrator",
		"this operation requires elevation",
	}
	for _, hint := range elevationHints {
		if strings.Contains(msg, hint) {
			return fmt.Errorf("%w (hint: Chocolatey typically requires an elevated shell — re-run as Administrator)", err)
		}
	}
	return err
}

// CheckHealth performs health checks on Chocolatey.
func (a *Adapter) CheckHealth(ctx context.Context) (manager.Status, error) {
	// Run 'choco outdated -r' to check for outdated packages
	result, err := a.executor.Execute(ctx, chocoCommand, "outdated", "-r")
	if err != nil {
		a.logger.Warn(ctx, "Failed to check chocolatey status",
			output.Field{Key: "error", Value: err.Error()})
		return manager.StatusDegraded, nil
	}

	if result.ExitCode == 0 {
		// Check if there are outdated packages
		outputStr := strings.TrimSpace(result.Stdout)
		if outputStr == "" {
			return manager.StatusHealthy, nil
		}
		// Has outdated packages
		lines := strings.Split(outputStr, "\n")
		outdatedCount := 0
		for _, line := range lines {
			if strings.TrimSpace(line) != "" && strings.Contains(line, "|") {
				outdatedCount++
			}
		}
		if outdatedCount > 0 {
			return manager.StatusDegraded, nil
		}
		return manager.StatusHealthy, nil
	}

	return manager.StatusError, nil
}

// Update performs update operations on Chocolatey packages.
func (a *Adapter) Update(ctx context.Context, opts adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
	a.logger.Info(ctx, "Starting Chocolatey update",
		output.Field{Key: "dry_run", Value: opts.DryRun},
		output.Field{Key: "strategy", Value: string(opts.Strategy)})

	result := &adapterm.UpdateResult{
		Success:         true,
		UpdatedPackages: []string{},
		FailedPackages:  []string{},
	}

	// If dry-run, simulate
	if opts.DryRun {
		result.Message = "Dry-run: would upgrade all chocolatey packages"
		a.logger.Info(ctx, "Dry-run mode: skipping actual update")
		return result, nil
	}

	// Fixed strategy: skip updates
	if opts.Strategy == adapterm.StrategyFixed {
		result.Message = "Strategy 'fixed': chocolatey upgrade skipped"
		return result, nil
	}

	// Run 'choco upgrade all -y' to upgrade all packages
	// -y confirms all prompts automatically
	upgradeResult, err := a.executor.Execute(ctx, chocoCommand, "upgrade", "all", "-y")
	if err != nil {
		result.Success = false
		wrapped := wrapElevationError(fmt.Errorf("chocolatey upgrade failed: %w", err))
		result.Message = wrapped.Error()
		return result, wrapped
	}

	if upgradeResult.ExitCode != 0 {
		// Check if it's just "nothing to update"
		outputLower := strings.ToLower(upgradeResult.Stdout + upgradeResult.Stderr)
		if strings.Contains(outputLower, "0 packages upgraded") ||
			strings.Contains(outputLower, "nothing to do") {
			result.Message = "All packages are up to date"
			return result, nil
		}

		result.Success = false
		result.Message = fmt.Sprintf("chocolatey upgrade failed: %s", upgradeResult.Stderr)
		return result, nil
	}

	// Parse updated packages from output
	// Chocolatey shows " package v1.0.0 has been upgraded to v2.0.0"
	lines := strings.Split(upgradeResult.Stdout, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Look for lines indicating successful upgrade
		if strings.Contains(line, "has been upgraded") ||
			strings.Contains(line, "upgraded successfully") {
			// Extract package name (first word after space)
			parts := strings.Fields(line)
			if len(parts) > 0 {
				// Clean package name
				pkgName := strings.TrimSpace(parts[0])
				if pkgName != "" && !strings.HasPrefix(pkgName, "-") {
					result.UpdatedPackages = append(result.UpdatedPackages, pkgName)
				}
			}
		}
	}

	// Also check for "X packages upgraded" pattern
	for _, line := range lines {
		if strings.Contains(line, "packages upgraded") {
			// Found summary line
			break
		}
	}

	result.Message = fmt.Sprintf("%d packages updated successfully", len(result.UpdatedPackages))
	a.logger.Info(ctx, "Chocolatey update completed",
		output.Field{Key: "updated_packages", Value: len(result.UpdatedPackages)})

	return result, nil
}
