// Package asdf provides an adapter for ASDF (extendable version manager).
// It implements the manager.Adapter interface for detecting, querying,
// and managing multiple runtime versions through asdf.
package asdf

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
	asdfCommand  = "asdf"
	whichCommand = "which"
	pluginArg    = "plugin"
	listArg      = "list"
	currentArg   = "current"
)

// Adapter implements the manager.Adapter interface for ASDF.
type Adapter struct {
	executor    output.CommandExecutor
	logger      output.Logger
	userHomeDir func() (string, error)
}

// NewAdapter creates a new ASDF adapter.
func NewAdapter(executor output.CommandExecutor, logger output.Logger) *Adapter {
	return &Adapter{
		executor:    executor,
		logger:      logger,
		userHomeDir: os.UserHomeDir,
	}
}

// Detect checks if ASDF is installed on the system.
func (a *Adapter) Detect(ctx context.Context) (bool, error) {
	result, err := a.executor.Execute(ctx, whichCommand, asdfCommand)
	return cmdutil.IsCommandAvailable(result, err), nil
}

// GetVersion retrieves the ASDF version.
func (a *Adapter) GetVersion(ctx context.Context) (string, error) {
	result, err := a.executor.Execute(ctx, asdfCommand, "version")
	if err != nil {
		return "", fmt.Errorf("failed to get asdf version: %w", err)
	}

	// Parse version from output like "v0.13.1" or "v0.13.1-abc1234"
	version := strings.TrimSpace(result.Stdout)
	version = strings.TrimPrefix(version, "v")

	// Remove git commit hash if present
	if idx := strings.Index(version, "-"); idx > 0 {
		version = version[:idx]
	}

	return version, nil
}

// GetBinaryPath returns the path to the asdf binary.
func (a *Adapter) GetBinaryPath(ctx context.Context) (string, error) {
	result, err := a.executor.Execute(ctx, whichCommand, asdfCommand)
	if checkErr := cmdutil.CheckResult(result, err, "locate asdf binary"); checkErr != nil {
		return "", checkErr
	}
	return cmdutil.ExtractStdout(result), nil
}

// GetConfigPath returns the path to ASDF configuration.
func (a *Adapter) GetConfigPath(_ context.Context) (string, error) {
	// ASDF config is typically in ~/.asdfrc
	homeDir, err := a.userHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory for ASDF configuration: %w", err)
	}

	return filepath.Join(homeDir, ".asdfrc"), nil
}

// ListPackages retrieves all installed plugins and their versions.
func (a *Adapter) ListPackages(ctx context.Context) ([]manager.Package, error) {
	// Get list of installed plugins
	pluginResult, err := a.executor.Execute(ctx, asdfCommand, pluginArg, listArg)
	if err != nil {
		return nil, fmt.Errorf("failed to list plugins: %w", err)
	}

	plugins := strings.Split(strings.TrimSpace(pluginResult.Stdout), "\n")
	packages := make([]manager.Package, 0)

	for _, plugin := range plugins {
		plugin = strings.TrimSpace(plugin)
		if plugin == "" {
			continue
		}

		// Get installed versions for this plugin
		versionResult, err := a.executor.Execute(ctx, asdfCommand, listArg, plugin)
		if err != nil {
			a.logger.Warn(ctx, "Failed to list versions for plugin",
				output.Field{Key: "plugin", Value: plugin},
				output.Field{Key: "error", Value: err.Error()})
			continue
		}

		// Parse versions (output format: " 1.2.3" or "*1.2.3" for current)
		lines := strings.SplitSeq(strings.TrimSpace(versionResult.Stdout), "\n")
		for line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			// Check if this is the current version
			isCurrent := strings.HasPrefix(line, "*")
			version := strings.TrimPrefix(line, "*")
			version = strings.TrimSpace(version)

			// Create package entry for each installed version
			pkg := manager.Package{
				Name:           fmt.Sprintf("%s@%s", plugin, version),
				CurrentVersion: version,
				IsGlobal:       isCurrent, // Use IsGlobal to indicate if it's the current version
				UpdateType:     manager.UpdateNone,
				Description:    fmt.Sprintf("ASDF plugin: %s", plugin),
			}

			packages = append(packages, pkg)
		}
	}

	return packages, nil
}

// CheckHealth verifies ASDF system health.
func (a *Adapter) CheckHealth(ctx context.Context) (manager.Status, error) {
	// Check if asdf can execute basic commands
	result, err := a.executor.Execute(ctx, asdfCommand, "version")
	if err != nil || result.ExitCode != 0 {
		a.logger.Warn(ctx, "Failed to run asdf version", output.Field{Key: "error", Value: err})
		return manager.StatusDegraded, nil
	}

	// Check if asdf data directory exists
	homeDir, err := os.UserHomeDir()
	if err == nil {
		asdfDir := filepath.Join(homeDir, ".asdf")
		if info, err := os.Stat(asdfDir); err != nil || !info.IsDir() {
			a.logger.Warn(ctx, "ASDF directory not accessible",
				output.Field{Key: "path", Value: asdfDir})
			// Don't return degraded, might be installed via package manager
		}
	}

	return manager.StatusHealthy, nil
}

// Update performs update operations (stub implementation).
// Update refreshes asdf plugins and optionally updates installed tools.
func (a *Adapter) Update(ctx context.Context, opts adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
	a.logger.Info(ctx, "Starting asdf update",
		output.Field{Key: "dry_run", Value: opts.DryRun},
		output.Field{Key: "strategy", Value: string(opts.Strategy)})

	result := &adapterm.UpdateResult{
		Success:         true,
		UpdatedPackages: []string{},
		FailedPackages:  []string{},
	}

	if opts.DryRun {
		result.Message = "Dry-run: would run asdf plugin update --all"
		return result, nil
	}

	pluginRes, err := a.executor.Execute(ctx, "asdf", "plugin", "update", "--all")
	if err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("asdf plugin update failed: %v", err)
		return result, fmt.Errorf("asdf plugin update failed: %w", err)
	}
	if pluginRes.ExitCode != 0 {
		result.Success = false
		result.Message = fmt.Sprintf("asdf plugin update failed: %s", pluginRes.Stderr)
		return result, nil
	}

	if opts.Strategy == adapterm.StrategyFixed {
		result.Message = "asdf plugins updated; tool upgrades skipped (strategy fixed)"
		return result, nil
	}

	// Update installed tools to latest when possible
	updateRes, err := a.executor.Execute(ctx, "asdf", "update")
	if err == nil && updateRes.ExitCode == 0 {
		result.Message = "asdf plugins and runtime updated"
		return result, nil
	}
	result.Message = "asdf plugins updated"
	return result, nil
}
