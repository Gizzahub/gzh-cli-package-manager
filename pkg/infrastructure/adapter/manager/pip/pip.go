// Package pip provides an adapter for Pip (Python Package Installer).
// It implements the manager.Adapter interface for detecting, querying,
// and managing Python packages through pip.
package pip

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/cmdutil"
)

const (
	pipCommand   = "pip"
	pip3Command  = "pip3"
	whichCommand = "which"
	listArg      = "list"
	formatFlag   = "--format=json"
	outdatedFlag = "--outdated"
	checkCommand = "check"
)

// Adapter implements the manager.Adapter interface for Pip.
type Adapter struct {
	executor output.CommandExecutor
	logger   output.Logger
}

// NewAdapter creates a new Pip adapter.
func NewAdapter(executor output.CommandExecutor, logger output.Logger) *Adapter {
	return &Adapter{
		executor: executor,
		logger:   logger,
	}
}

// Detect checks if Pip is installed on the system.
func (a *Adapter) Detect(ctx context.Context) (bool, error) {
	// Try pip3 first (more common on modern systems)
	result, err := a.executor.Execute(ctx, whichCommand, pip3Command)
	if cmdutil.IsCommandAvailable(result, err) {
		return true, nil
	}

	// Fallback to pip
	result, err = a.executor.Execute(ctx, whichCommand, pipCommand)
	return cmdutil.IsCommandAvailable(result, err), nil
}

// GetVersion retrieves the Pip version.
func (a *Adapter) GetVersion(ctx context.Context) (string, error) {
	cmd := a.getPipCommand(ctx)
	result, err := a.executor.Execute(ctx, cmd, "--version")
	if err != nil {
		return "", fmt.Errorf("get pip version: %w", err)
	}

	// Parse version from output like "pip 23.0.1 from /usr/lib/python3.11/site-packages/pip (python 3.11)"
	return cmdutil.ExtractVersionField(result.Stdout, 1, "get pip version")
}

// GetBinaryPath returns the path to the pip binary.
func (a *Adapter) GetBinaryPath(ctx context.Context) (string, error) {
	cmd := a.getPipCommand(ctx)
	result, err := a.executor.Execute(ctx, whichCommand, cmd)
	if err != nil {
		return "", fmt.Errorf("locate pip binary: %w", err)
	}
	return cmdutil.ExtractStdout(result), nil
}

// GetConfigPath returns the path to Pip configuration.
func (a *Adapter) GetConfigPath(ctx context.Context) (string, error) {
	// pip config list shows config file locations
	cmd := a.getPipCommand(ctx)
	result, err := a.executor.Execute(ctx, cmd, "config", "list", "--user")
	if err == nil && result.ExitCode == 0 {
		// Config exists, return user config path
		return "~/.pip/pip.conf", nil
	}

	// Return default config location
	return "~/.pip/pip.conf", nil
}

// ListPackages retrieves all installed packages.
func (a *Adapter) ListPackages(ctx context.Context) ([]manager.Package, error) {
	cmd := a.getPipCommand(ctx)

	// Get installed packages
	result, err := a.executor.Execute(ctx, cmd, listArg, formatFlag)
	if err != nil {
		return nil, fmt.Errorf("list pip packages: %w", err)
	}

	// Parse JSON output
	type pipPackage struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}

	var installed []pipPackage
	if err := cmdutil.UnmarshalJSON(result, &installed, "parse pip packages"); err != nil {
		return nil, err
	}

	// Get outdated packages
	outdatedResult, err := a.executor.Execute(ctx, cmd, listArg, outdatedFlag, formatFlag)
	outdatedMap := make(map[string]string)
	if err == nil && outdatedResult.ExitCode == 0 {
		type outdatedPackage struct {
			Name           string `json:"name"`
			Version        string `json:"version"`
			LatestVersion  string `json:"latest_version"`
			LatestFileType string `json:"latest_filetype"`
		}

		var outdated []outdatedPackage
		if err := json.Unmarshal([]byte(outdatedResult.Stdout), &outdated); err == nil {
			for _, pkg := range outdated {
				outdatedMap[pkg.Name] = pkg.LatestVersion
			}
		}
	}

	// Build package list
	packages := make([]manager.Package, 0, len(installed))
	for _, pkg := range installed {
		p := manager.Package{
			Name:           pkg.Name,
			CurrentVersion: pkg.Version,
			IsGlobal:       true, // pip packages are user or system-wide
			UpdateType:     manager.UpdateNone,
		}

		// Check if update available
		if latestVersion, hasUpdate := outdatedMap[pkg.Name]; hasUpdate {
			p.AvailableVersion = latestVersion
			p.UpdateType = manager.UpdateMinor // pip doesn't distinguish update types
		}

		packages = append(packages, p)
	}

	return packages, nil
}

// CheckHealth verifies Pip system health.
func (a *Adapter) CheckHealth(ctx context.Context) (manager.Status, error) {
	cmd := a.getPipCommand(ctx)

	// Run pip check to verify package dependencies
	checkResult, err := a.executor.Execute(ctx, cmd, checkCommand)
	if err != nil {
		a.logger.Warn(ctx, "Failed to run pip check", output.Field{Key: "error", Value: err.Error()})
		return manager.StatusDegraded, nil
	}

	if checkResult.ExitCode != 0 {
		// pip check returns non-zero if there are dependency issues
		a.logger.Warn(ctx, "Pip check found dependency issues", output.Field{Key: "stderr", Value: checkResult.Stderr})
		return manager.StatusDegraded, nil
	}

	return manager.StatusHealthy, nil
}

// getPipCommand returns the appropriate pip command (pip3 or pip).
func (a *Adapter) getPipCommand(ctx context.Context) string {
	// Try pip3 first
	result, err := a.executor.Execute(ctx, whichCommand, pip3Command)
	if cmdutil.IsCommandAvailable(result, err) {
		return pip3Command
	}
	return pipCommand
}

// Update upgrades outdated pip packages (pip list --outdated then pip install -U).
func (a *Adapter) Update(ctx context.Context, opts adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
	a.logger.Info(ctx, "Starting pip update",
		output.Field{Key: "dry_run", Value: opts.DryRun},
		output.Field{Key: "strategy", Value: string(opts.Strategy)})

	result := &adapterm.UpdateResult{
		Success:         true,
		UpdatedPackages: []string{},
		FailedPackages:  []string{},
	}

	if opts.DryRun {
		result.Message = "Dry-run: would upgrade outdated pip packages"
		return result, nil
	}
	if opts.Strategy == adapterm.StrategyFixed {
		result.Message = "Strategy 'fixed': pip upgrade skipped"
		return result, nil
	}

	packages := opts.Packages
	if len(packages) == 0 {
		listRes, err := a.executor.Execute(ctx, "pip", "list", "--outdated", "--format=freeze")
		if err == nil && listRes.ExitCode == 0 && strings.TrimSpace(listRes.Stdout) != "" {
			for _, line := range strings.Split(listRes.Stdout, "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				name := strings.Split(line, "==")[0]
				if name != "" {
					packages = append(packages, name)
				}
			}
		}
	}
	if len(packages) == 0 {
		result.Message = "No outdated pip packages found"
		return result, nil
	}

	args := append([]string{"install", "--upgrade"}, packages...)
	execResult, err := a.executor.Execute(ctx, "pip", args...)
	if err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("pip install --upgrade failed: %v", err)
		return result, fmt.Errorf("pip update failed: %w", err)
	}
	if execResult.ExitCode != 0 {
		result.Success = false
		result.Message = fmt.Sprintf("pip upgrade failed: %s", execResult.Stderr)
		return result, nil
	}
	result.UpdatedPackages = packages
	result.Message = "pip packages upgraded"
	return result, nil
}
