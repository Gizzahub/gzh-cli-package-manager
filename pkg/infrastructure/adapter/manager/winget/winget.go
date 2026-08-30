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
	wingetCommand  = "winget"
	dryRunFieldKey = "dry_run"
	errorFieldKey  = "error"
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
	if resultErr := cmdutil.CheckResult(result, err, "get winget version"); resultErr != nil {
		return "", resultErr
	}

	// Output format: "v1.6.3482\n" or similar
	version := strings.TrimPrefix(cmdutil.ExtractStdout(result), "v")
	return strings.TrimSpace(version), nil
}

// GetBinaryPath returns the path to the winget binary.
func (a *Adapter) GetBinaryPath(ctx context.Context) (string, error) {
	// On Windows, use 'where' command instead of 'which'
	result, err := a.executor.Execute(ctx, "where", wingetCommand)
	if resultErr := cmdutil.CheckResult(result, err, "find winget binary"); resultErr != nil {
		return "", resultErr
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
	if resultErr := cmdutil.CheckResult(result, err, "list winget packages"); resultErr != nil {
		return nil, resultErr
	}

	// Try to parse as JSON first
	packages, err := a.parseListOutput(result)
	if err != nil {
		a.logger.Warn(ctx, "Failed to parse winget JSON output, falling back to text parsing",
			output.Field{Key: errorFieldKey, Value: err.Error()})
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
			output.Field{Key: errorFieldKey, Value: err.Error()})
		return manager.StatusDegraded, nil //nolint:nilerr // CheckHealth reports probe failures as degraded status.
	}

	if result.ExitCode == 0 {
		return manager.StatusHealthy, nil
	}

	return manager.StatusError, nil
}

// Search finds packages matching query via `winget search`.
func (a *Adapter) Search(ctx context.Context, query string) ([]manager.Package, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("search winget packages: query is required")
	}

	result, err := a.executor.Execute(ctx, wingetCommand, "search", query, "--disable-interactivity")
	if resultErr := cmdutil.CheckResult(result, err, "search winget packages"); resultErr != nil {
		return nil, resultErr
	}

	// Prefer JSON when available; fall back to text table parsing.
	packages, err := a.parseSearchJSON(result)
	if err != nil {
		a.logger.Warn(ctx, "Failed to parse winget search JSON, falling back to text parsing",
			output.Field{Key: errorFieldKey, Value: err.Error()})
		return a.parseSearchText(result), nil //nolint:nilerr // JSON is optional; a text parse is a successful fallback.
	}
	return packages, nil
}

// parseSearchJSON parses winget search JSON output (same shape as list export).
func (a *Adapter) parseSearchJSON(result *output.ExecutionResult) ([]manager.Package, error) {
	return a.parseListOutput(result)
}

// parseSearchText parses winget search table output.
// Columns: Name, Id, Version, Match, Source (variable width).
func (a *Adapter) parseSearchText(result *output.ExecutionResult) []manager.Package {
	lines := strings.Split(result.Stdout, "\n")
	var packages []manager.Package

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

		fields := strings.Fields(line)
		// Need at least Name + Id + Version
		if len(fields) < 3 {
			continue
		}

		pkg := manager.Package{
			Name:           fields[0],
			CurrentVersion: fields[2],
			IsGlobal:       true,
			UpdateType:     manager.UpdateNone,
			Manager:        manager.ManagerWinget,
		}
		// Last field is typically Source when present
		if len(fields) >= 4 {
			pkg.Description = fields[1] // Id as description context
		}
		packages = append(packages, pkg)
	}

	return packages
}

// Install installs a package by winget ID or name.
// dryRun skips the native install and returns nil after logging intent.
func (a *Adapter) Install(ctx context.Context, pkgID string, dryRun bool) error {
	pkgID = strings.TrimSpace(pkgID)
	if pkgID == "" {
		return fmt.Errorf("install winget package: package id is required")
	}

	a.logger.Info(ctx, "Winget install",
		output.Field{Key: "package", Value: pkgID},
		output.Field{Key: dryRunFieldKey, Value: dryRun})

	if dryRun {
		return nil
	}

	result, err := a.executor.Execute(
		ctx, wingetCommand,
		"install", "--id", pkgID,
		"--disable-interactivity",
		"--accept-package-agreements",
		"--accept-source-agreements",
	)
	if resultErr := cmdutil.CheckResult(result, err, "install winget package "+pkgID); resultErr != nil {
		return resultErr
	}
	return nil
}

// Uninstall removes a package by winget ID or name.
// dryRun skips the native uninstall.
func (a *Adapter) Uninstall(ctx context.Context, pkgID string, dryRun bool) error {
	pkgID = strings.TrimSpace(pkgID)
	if pkgID == "" {
		return fmt.Errorf("uninstall winget package: package id is required")
	}

	a.logger.Info(ctx, "Winget uninstall",
		output.Field{Key: "package", Value: pkgID},
		output.Field{Key: dryRunFieldKey, Value: dryRun})

	if dryRun {
		return nil
	}

	result, err := a.executor.Execute(
		ctx, wingetCommand,
		"uninstall", "--id", pkgID,
		"--disable-interactivity",
	)
	if resultErr := cmdutil.CheckResult(result, err, "uninstall winget package "+pkgID); resultErr != nil {
		return resultErr
	}
	return nil
}

// ListSources returns configured winget sources via `winget source list`.
func (a *Adapter) ListSources(ctx context.Context) ([]adapterm.Source, error) {
	result, err := a.executor.Execute(ctx, wingetCommand, "source", "list")
	if resultErr := cmdutil.CheckResult(result, err, "list winget sources"); resultErr != nil {
		return nil, resultErr
	}
	return parseWingetSources(result.Stdout), nil
}

// parseWingetSources parses text table from `winget source list`.
// Typical columns: Name, Argument
//
//	Name    Argument
//	---------------------------------------------------
//	msstore https://storeedgefd.dsx.mp.microsoft.com/v9.0
//	winget  https://cdn.winget.microsoft.com/cache
func parseWingetSources(stdout string) []adapterm.Source {
	lines := strings.Split(stdout, "\n")
	var sources []adapterm.Source

	startIdx := 0
	for i, line := range lines {
		if strings.Contains(line, "---") {
			startIdx = i + 1
			break
		}
	}
	// No separator: skip possible header
	if startIdx == 0 && len(lines) > 0 {
		lower := strings.ToLower(lines[0])
		if strings.Contains(lower, "name") {
			startIdx = 1
		}
	}

	for i := startIdx; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		src := adapterm.Source{Name: fields[0]}
		if len(fields) >= 2 {
			src.Arg = fields[1]
		}
		sources = append(sources, src)
	}
	return sources
}

// Update performs update operations on winget packages.
func (a *Adapter) Update(ctx context.Context, opts adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
	a.logger.Info(ctx, "Starting winget update",
		output.Field{Key: dryRunFieldKey, Value: opts.DryRun},
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
	lines := strings.SplitSeq(upgradeResult.Stdout, "\n")
	for line := range lines {
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
