// Package homebrew provides an adapter for Homebrew package manager.
// It implements the manager.Adapter interface for detecting, querying,
// and managing packages through Homebrew.
package homebrew

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
)

// Adapter implements the manager.Adapter interface for Homebrew.
type Adapter struct {
	executor output.CommandExecutor
	logger   output.Logger
}

// NewAdapter creates a new Homebrew adapter.
func NewAdapter(executor output.CommandExecutor, logger output.Logger) *Adapter {
	return &Adapter{
		executor: executor,
		logger:   logger,
	}
}

// Detect checks if Homebrew is installed on the system.
func (a *Adapter) Detect(ctx context.Context) (bool, error) {
	result, err := a.executor.Execute(ctx, "which", "brew")
	if err != nil {
		return false, nil // which command failed, brew not found
	}

	return result.ExitCode == 0 && strings.TrimSpace(result.Stdout) != "", nil
}

// GetVersion retrieves the version of the installed Homebrew.
func (a *Adapter) GetVersion(ctx context.Context) (string, error) {
	result, err := a.executor.Execute(ctx, "brew", "--version")
	if err != nil {
		return "", fmt.Errorf("failed to get brew version: %w", err)
	}

	if result.ExitCode != 0 {
		return "", fmt.Errorf("brew --version failed: %s", result.Stderr)
	}

	// Output format: "Homebrew 4.2.1\n..."
	lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	if len(lines) == 0 {
		return "", fmt.Errorf("unexpected brew --version output")
	}

	// Extract version from "Homebrew X.Y.Z"
	parts := strings.Fields(lines[0])
	if len(parts) < 2 {
		return "", fmt.Errorf("unexpected version format: %s", lines[0])
	}

	return parts[1], nil
}

// GetBinaryPath returns the path to the Homebrew binary.
func (a *Adapter) GetBinaryPath(ctx context.Context) (string, error) {
	result, err := a.executor.Execute(ctx, "which", "brew")
	if err != nil {
		return "", fmt.Errorf("failed to find brew binary: %w", err)
	}

	if result.ExitCode != 0 {
		return "", fmt.Errorf("brew binary not found")
	}

	return strings.TrimSpace(result.Stdout), nil
}

// GetConfigPath returns the path to the Homebrew configuration.
func (a *Adapter) GetConfigPath(ctx context.Context) (string, error) {
	result, err := a.executor.Execute(ctx, "brew", "--prefix")
	if err != nil {
		return "", fmt.Errorf("failed to get brew prefix: %w", err)
	}

	if result.ExitCode != 0 {
		return "", fmt.Errorf("brew --prefix failed: %s", result.Stderr)
	}

	return strings.TrimSpace(result.Stdout), nil
}

// ListPackages retrieves all packages managed by Homebrew.
func (a *Adapter) ListPackages(ctx context.Context) ([]manager.Package, error) {
	// Use brew info --json=v2 --installed for detailed package information
	result, err := a.executor.Execute(ctx, "brew", "info", "--json=v2", "--installed")
	if err != nil {
		return nil, fmt.Errorf("failed to list brew packages: %w", err)
	}

	if result.ExitCode != 0 {
		return nil, fmt.Errorf("brew info failed: %s", result.Stderr)
	}

	// Parse JSON output
	var brewInfo struct {
		Formulae []struct {
			Name            string   `json:"name"`
			FullName        string   `json:"full_name"`
			Desc            string   `json:"desc"`
			Version         string   `json:"version"`
			InstalledOnDemand bool   `json:"installed_on_demand"`
			Versions        struct {
				Stable string `json:"stable"`
			} `json:"versions"`
		} `json:"formulae"`
		Casks []struct {
			Token   string `json:"token"`
			Name    []string `json:"name"`
			Desc    string `json:"desc"`
			Version string `json:"version"`
		} `json:"casks"`
	}

	if err := json.Unmarshal([]byte(result.Stdout), &brewInfo); err != nil {
		return nil, fmt.Errorf("failed to parse brew info output: %w", err)
	}

	packages := make([]manager.Package, 0, len(brewInfo.Formulae)+len(brewInfo.Casks))

	// Add formulae (regular packages)
	for _, formula := range brewInfo.Formulae {
		pkg := manager.Package{
			Name:           formula.Name,
			CurrentVersion: formula.Version,
			Description:    formula.Desc,
			IsGlobal:       !formula.InstalledOnDemand,
		}

		// Check if update is available
		if formula.Versions.Stable != "" && formula.Versions.Stable != formula.Version {
			pkg.AvailableVersion = formula.Versions.Stable
			pkg.UpdateType = manager.UpdateMinor // Simplified: assume minor update
		} else {
			pkg.UpdateType = manager.UpdateNone
		}

		packages = append(packages, pkg)
	}

	// Add casks (GUI applications)
	for _, cask := range brewInfo.Casks {
		displayName := cask.Token
		if len(cask.Name) > 0 {
			displayName = cask.Name[0]
		}

		pkg := manager.Package{
			Name:           displayName,
			CurrentVersion: cask.Version,
			Description:    cask.Desc,
			IsGlobal:       true, // Casks are typically installed explicitly
			UpdateType:     manager.UpdateNone,
		}

		packages = append(packages, pkg)
	}

	return packages, nil
}

// CheckHealth performs health checks on Homebrew.
func (a *Adapter) CheckHealth(ctx context.Context) (manager.Status, error) {
	// Run brew doctor to check for issues
	result, err := a.executor.Execute(ctx, "brew", "doctor")
	if err != nil {
		a.logger.Warn(ctx, "Failed to run brew doctor",
			output.Field{Key: "error", Value: err.Error()})
		return manager.StatusDegraded, nil
	}

	if result.ExitCode == 0 {
		return manager.StatusHealthy, nil
	}

	// Check if there are warnings (exit code 1) or errors
	output := strings.ToLower(result.Stdout + result.Stderr)
	if strings.Contains(output, "warning") {
		return manager.StatusDegraded, nil
	}

	return manager.StatusError, nil
}
