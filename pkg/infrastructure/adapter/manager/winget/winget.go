// Package winget provides an adapter for Windows Package Manager (winget).
// It implements the manager.Adapter interface for detecting, querying,
// and managing packages through winget on Windows systems.
package winget

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
	wingetCommand = "winget"
	// Default config path on Windows.
	defaultConfigPath = `%LOCALAPPDATA%\Packages\Microsoft.DesktopAppInstaller_8wekyb3d8bbwe\LocalState\settings.json`
)

// Adapter implements the manager.Adapter interface for winget.
type Adapter struct {
	executor output.CommandExecutor
	logger   output.Logger
}

// NewAdapter creates a new winget adapter.
func NewAdapter(executor output.CommandExecutor, logger output.Logger) *Adapter {
	return &Adapter{
		executor: executor,
		logger:   logger,
	}
}

// Detect checks if winget is installed on the system.
func (a *Adapter) Detect(ctx context.Context) (bool, error) {
	// Try running winget --version to detect if it's available
	result, err := a.executor.Execute(ctx, wingetCommand, "--version")
	return cmdutil.IsCommandAvailable(result, err), nil
}

// GetVersion retrieves the version of the installed winget.
func (a *Adapter) GetVersion(ctx context.Context) (string, error) {
	result, err := a.executor.Execute(ctx, wingetCommand, "--version")
	if err := cmdutil.CheckResult(result, err, "get winget version"); err != nil {
		return "", err
	}

	// Output format: "v1.6.3482\n" or similar
	version := strings.TrimPrefix(cmdutil.ExtractStdout(result), "v")
	return strings.TrimSpace(version), nil
}

// GetBinaryPath returns the path to the winget binary.
func (a *Adapter) GetBinaryPath(ctx context.Context) (string, error) {
	// On Windows, use 'where' command instead of 'which'
	result, err := a.executor.Execute(ctx, "where", wingetCommand)
	if err := cmdutil.CheckResult(result, err, "find winget binary"); err != nil {
		return "", err
	}

	// 'where' may return multiple lines; take the first one
	lines := strings.Split(cmdutil.ExtractStdout(result), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return "", fmt.Errorf("find winget binary: unexpected empty output")
	}
	return strings.TrimSpace(lines[0]), nil
}

// GetConfigPath returns the path to the winget configuration.
func (a *Adapter) GetConfigPath(_ context.Context) (string, error) {
	// winget settings are stored in a known location
	return defaultConfigPath, nil
}

// wingetPackage represents a package in winget JSON output.
type wingetPackage struct {
	ID               string `json:"Id"`
	Name             string `json:"Name"`
	Version          string `json:"Version"`
	AvailableVersion string `json:"AvailableVersion,omitempty"`
	Source           string `json:"Source,omitempty"`
}

// wingetListOutput represents the JSON output from 'winget list'.
type wingetListOutput struct {
	Sources []struct {
		Packages []wingetPackage `json:"Packages"`
	} `json:"Sources"`
}

// ListPackages retrieves all packages managed by winget.
func (a *Adapter) ListPackages(ctx context.Context) ([]manager.Package, error) {
	// Use winget export to get JSON list of installed packages
	result, err := a.executor.Execute(ctx, wingetCommand, "list", "--source", "winget", "--disable-interactivity")
	if err := cmdutil.CheckResult(result, err, "list winget packages"); err != nil {
		return nil, err
	}

	// Try to parse as JSON first
	packages, err := a.parseListOutput(result)
	if err != nil {
		a.logger.Warn(ctx, "Failed to parse winget JSON output, falling back to text parsing",
			output.Field{Key: "error", Value: err.Error()})
		return a.parseListOutputText(result)
	}
	return packages, nil
}

// parseListOutput parses JSON output from winget list.
func (a *Adapter) parseListOutput(result *output.ExecutionResult) ([]manager.Package, error) {
	var listOutput wingetListOutput
	if err := json.Unmarshal([]byte(result.Stdout), &listOutput); err != nil {
		return nil, err
	}

	var packages []manager.Package
	for _, source := range listOutput.Sources {
		for _, pkg := range source.Packages {
			p := manager.Package{
				Name:           pkg.Name,
				CurrentVersion: pkg.Version,
				IsGlobal:       true, // winget packages are system-wide
			}

			if pkg.AvailableVersion != "" && pkg.AvailableVersion != pkg.Version {
				p.AvailableVersion = pkg.AvailableVersion
				p.UpdateType = manager.UpdateMinor
			} else {
				p.UpdateType = manager.UpdateNone
			}

			packages = append(packages, p)
		}
	}
	return packages, nil
}

// parseListOutputText parses text output from winget list (fallback).
func (a *Adapter) parseListOutputText(result *output.ExecutionResult) ([]manager.Package, error) {
	lines := strings.Split(result.Stdout, "\n")
	var packages []manager.Package

	// Skip header lines (first 2 lines typically)
	startIdx := 0
	for i, line := range lines {
		if strings.Contains(line, "---") {
			startIdx = i + 1
			break
		}
	}

	for i := startIdx; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// Parse fixed-width columns: Name, Id, Version, Available, Source
		fields := strings.Fields(line)
		if len(fields) >= 3 {
			pkg := manager.Package{
				Name:           fields[0],
				CurrentVersion: fields[2],
				IsGlobal:       true,
				UpdateType:     manager.UpdateNone,
			}
			packages = append(packages, pkg)
		}
	}

	return packages, nil
}

// CheckHealth performs health checks on winget.
func (a *Adapter) CheckHealth(ctx context.Context) (manager.Status, error) {
	// Check if winget can list sources (basic health check)
	result, err := a.executor.Execute(ctx, wingetCommand, "source", "list")
	if err != nil {
		a.logger.Warn(ctx, "Failed to check winget sources",
			output.Field{Key: "error", Value: err.Error()})
		return manager.StatusDegraded, nil
	}

	if result.ExitCode == 0 {
		return manager.StatusHealthy, nil
	}

	return manager.StatusError, nil
}

// Update performs update operations on winget packages.
func (a *Adapter) Update(ctx context.Context, opts adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
	a.logger.Info(ctx, "Starting winget update",
		output.Field{Key: "dry_run", Value: opts.DryRun},
		output.Field{Key: "strategy", Value: string(opts.Strategy)})

	result := &adapterm.UpdateResult{
		Success:         true,
		UpdatedPackages: []string{},
		FailedPackages:  []string{},
	}

	// If dry-run, simulate
	if opts.DryRun {
		result.Message = "Dry-run: would upgrade all winget packages"
		a.logger.Info(ctx, "Dry-run mode: skipping actual update")
		return result, nil
	}

	// Run winget upgrade --all
	args := []string{"upgrade", "--all", "--disable-interactivity"}
	if opts.Strategy == adapterm.StrategyFixed {
		result.Message = "Strategy 'fixed': winget update skipped"
		return result, nil
	}

	upgradeResult, err := a.executor.Execute(ctx, wingetCommand, args...)
	if err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("winget upgrade failed: %v", err)
		return result, fmt.Errorf("winget upgrade failed: %w", err)
	}

	if upgradeResult.ExitCode != 0 {
		// Exit code 0x8A150010 means no packages to upgrade - not an error
		if strings.Contains(upgradeResult.Stderr, "8A150010") {
			result.Message = "No packages to upgrade"
			return result, nil
		}

		result.Success = false
		result.Message = fmt.Sprintf("winget upgrade failed: %s", upgradeResult.Stderr)
		return result, nil
	}

	// Parse upgraded packages from output
	lines := strings.Split(upgradeResult.Stdout, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Successfully installed") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				result.UpdatedPackages = append(result.UpdatedPackages, parts[2])
			}
		}
	}

	result.Message = fmt.Sprintf("%d packages updated successfully", len(result.UpdatedPackages))
	a.logger.Info(ctx, "Winget update completed",
		output.Field{Key: "updated_packages", Value: len(result.UpdatedPackages)})

	return result, nil
}
