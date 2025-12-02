// Package npm provides an adapter for NPM package manager.
// It implements the manager.Adapter interface for detecting, querying,
// and managing packages through NPM (Node Package Manager).
package npm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/version"
)

// Adapter implements the manager.Adapter interface for NPM.
type Adapter struct {
	executor output.CommandExecutor
	logger   output.Logger
}

// NewAdapter creates a new NPM adapter.
func NewAdapter(executor output.CommandExecutor, logger output.Logger) *Adapter {
	return &Adapter{
		executor: executor,
		logger:   logger,
	}
}

// Detect checks if NPM is installed on the system.
func (a *Adapter) Detect(ctx context.Context) (bool, error) {
	result, err := a.executor.Execute(ctx, "which", "npm")
	if err != nil {
		return false, nil // which command failed, npm not found
	}

	return result.ExitCode == 0 && strings.TrimSpace(result.Stdout) != "", nil
}

// GetVersion retrieves the version of the installed NPM.
func (a *Adapter) GetVersion(ctx context.Context) (string, error) {
	result, err := a.executor.Execute(ctx, "npm", "--version")
	if err != nil {
		return "", fmt.Errorf("failed to get npm version: %w", err)
	}

	if result.ExitCode != 0 {
		return "", fmt.Errorf("npm --version failed: %s", result.Stderr)
	}

	return strings.TrimSpace(result.Stdout), nil
}

// GetBinaryPath returns the path to the NPM binary.
func (a *Adapter) GetBinaryPath(ctx context.Context) (string, error) {
	result, err := a.executor.Execute(ctx, "which", "npm")
	if err != nil {
		return "", fmt.Errorf("failed to find npm binary: %w", err)
	}

	if result.ExitCode != 0 {
		return "", fmt.Errorf("npm binary not found")
	}

	return strings.TrimSpace(result.Stdout), nil
}

// GetConfigPath returns the path to the NPM configuration.
func (a *Adapter) GetConfigPath(ctx context.Context) (string, error) {
	// Get npm config prefix (global install location)
	result, err := a.executor.Execute(ctx, "npm", "config", "get", "prefix")
	if err != nil {
		return "", fmt.Errorf("failed to get npm config: %w", err)
	}

	if result.ExitCode != 0 {
		return "", fmt.Errorf("npm config get prefix failed: %s", result.Stderr)
	}

	return strings.TrimSpace(result.Stdout), nil
}

// ListPackages retrieves all globally installed packages managed by NPM.
func (a *Adapter) ListPackages(ctx context.Context) ([]manager.Package, error) {
	// Get list of globally installed packages
	result, err := a.executor.Execute(ctx, "npm", "list", "-g", "--depth=0", "--json")
	if err != nil {
		return nil, fmt.Errorf("failed to list npm packages: %w", err)
	}

	if result.ExitCode != 0 {
		return nil, fmt.Errorf("npm list failed: %s", result.Stderr)
	}

	// Parse JSON output
	var npmList struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}

	if err := json.Unmarshal([]byte(result.Stdout), &npmList); err != nil {
		return nil, fmt.Errorf("failed to parse npm list output: %w", err)
	}

	// Get outdated packages
	outdatedResult, err := a.executor.Execute(ctx, "npm", "outdated", "-g", "--json")
	outdatedMap := make(map[string]outdatedPackage)

	// npm outdated returns exit code 1 when packages are outdated, which is expected
	if err == nil && outdatedResult.ExitCode == 0 || outdatedResult.ExitCode == 1 {
		var outdatedData map[string]outdatedPackage
		if err := json.Unmarshal([]byte(outdatedResult.Stdout), &outdatedData); err == nil {
			outdatedMap = outdatedData
		}
	}

	// Convert to domain packages
	packages := make([]manager.Package, 0, len(npmList.Dependencies))

	for name, dep := range npmList.Dependencies {
		pkg := manager.Package{
			Name:           name,
			CurrentVersion: dep.Version,
			Description:    "", // npm list doesn't provide descriptions
			IsGlobal:       true,
			UpdateType:     manager.UpdateNone,
		}

		// Check if update is available
		if outdated, hasUpdate := outdatedMap[name]; hasUpdate {
			pkg.AvailableVersion = outdated.Latest
			pkg.UpdateType = version.DetermineUpdateType(dep.Version, outdated.Latest)
		}

		packages = append(packages, pkg)
	}

	return packages, nil
}

// CheckHealth performs health checks on NPM.
func (a *Adapter) CheckHealth(ctx context.Context) (manager.Status, error) {
	// Run npm doctor to check for issues
	result, err := a.executor.Execute(ctx, "npm", "doctor")
	if err != nil {
		a.logger.Warn(ctx, "Failed to run npm doctor",
			output.Field{Key: "error", Value: err.Error()})
		return manager.StatusDegraded, nil
	}

	if result.ExitCode == 0 {
		return manager.StatusHealthy, nil
	}

	// npm doctor returns non-zero on warnings or errors
	output := strings.ToLower(result.Stdout + result.Stderr)
	if strings.Contains(output, "error") {
		return manager.StatusError, nil
	}

	return manager.StatusDegraded, nil
}

// outdatedPackage represents an outdated npm package.
type outdatedPackage struct {
	Current  string `json:"current"`
	Wanted   string `json:"wanted"`
	Latest   string `json:"latest"`
	Location string `json:"location"`
}

// Update performs update operations (stub implementation).
func (a *Adapter) Update(ctx context.Context, opts adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
	a.logger.Warn(ctx, "Update method not yet implemented for npm adapter")
	return &adapterm.UpdateResult{
		Success: false,
		Message: "Update not yet implemented for npm package manager",
	}, fmt.Errorf("update not yet implemented")
}
