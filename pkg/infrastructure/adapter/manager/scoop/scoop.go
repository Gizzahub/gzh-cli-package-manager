// Package scoop provides an adapter for the Scoop package manager.
// It implements the manager.Adapter interface for detecting, querying,
// and managing packages through Scoop on Windows systems.
// Scoop is a command-line installer for Windows that installs programs
// to the user's home directory, requiring no admin rights.
package scoop

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
	scoopCommand = "scoop"
)

// Adapter implements the manager.Adapter interface for Scoop.
type Adapter struct {
	executor output.CommandExecutor
	logger   output.Logger
}

// NewAdapter creates a new Scoop adapter.
func NewAdapter(executor output.CommandExecutor, logger output.Logger) *Adapter {
	return &Adapter{
		executor: executor,
		logger:   logger,
	}
}

// Detect checks if Scoop is installed on the system.
func (a *Adapter) Detect(ctx context.Context) (bool, error) {
	// Try running scoop --version to detect if it's available
	result, err := a.executor.Execute(ctx, scoopCommand, "--version")
	return cmdutil.IsCommandAvailable(result, err), nil
}

// GetVersion retrieves the version of the installed Scoop.
func (a *Adapter) GetVersion(ctx context.Context) (string, error) {
	result, err := a.executor.Execute(ctx, scoopCommand, "--version")
	if err := cmdutil.CheckResult(result, err, "get scoop version"); err != nil {
		return "", err
	}

	// Output format varies, but typically includes version info
	// Example: "Current Scoop version:\nv0.3.1\n..."
	lines := strings.Split(cmdutil.ExtractStdout(result), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "v") || strings.Contains(line, ".") {
			// Found a version-like string
			return strings.TrimPrefix(line, "v"), nil
		}
	}

	// Fallback: return first non-empty line
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.Contains(line, ":") {
			return line, nil
		}
	}

	return "unknown", nil
}

// GetBinaryPath returns the path to the Scoop binary.
func (a *Adapter) GetBinaryPath(ctx context.Context) (string, error) {
	// On Windows, use 'where' command
	result, err := a.executor.Execute(ctx, "where", scoopCommand)
	if err := cmdutil.CheckResult(result, err, "find scoop binary"); err != nil {
		return "", err
	}

	// 'where' may return multiple lines; take the first one
	lines := strings.Split(cmdutil.ExtractStdout(result), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return "", fmt.Errorf("find scoop binary: unexpected empty output")
	}
	return strings.TrimSpace(lines[0]), nil
}

// GetConfigPath returns the path to the Scoop installation directory.
func (a *Adapter) GetConfigPath(_ context.Context) (string, error) {
	// Check SCOOP environment variable first
	if scoopDir := os.Getenv("SCOOP"); scoopDir != "" {
		return scoopDir, nil
	}

	// Default to ~/scoop
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get scoop config path: %w", err)
	}
	return filepath.Join(homeDir, "scoop"), nil
}

// ListPackages retrieves all packages managed by Scoop.
func (a *Adapter) ListPackages(ctx context.Context) ([]manager.Package, error) {
	// Use 'scoop list' to get installed packages
	result, err := a.executor.Execute(ctx, scoopCommand, "list")
	if err := cmdutil.CheckResult(result, err, "list scoop packages"); err != nil {
		return nil, err
	}

	return a.parseListOutput(result), nil
}

// parseListOutput parses the text output from 'scoop list'.
// Format: "Name    Version   Source  Updated   Info"
func (a *Adapter) parseListOutput(result *output.ExecutionResult) []manager.Package {
	lines := strings.Split(result.Stdout, "\n")
	var packages []manager.Package

	// Skip header lines (typically 2 lines: header + separator)
	startIdx := 0
	for i, line := range lines {
		if strings.Contains(line, "---") || strings.Contains(line, "===") {
			startIdx = i + 1
			break
		}
	}

	// If no separator found, skip first line (header)
	if startIdx == 0 && len(lines) > 1 {
		startIdx = 1
	}

	for i := startIdx; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// Parse columns: Name, Version, Source, Updated, Info
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			pkg := manager.Package{
				Name:           fields[0],
				CurrentVersion: fields[1],
				IsGlobal:       false, // Scoop installs to user directory
				UpdateType:     manager.UpdateNone,
			}
			packages = append(packages, pkg)
		}
	}

	return packages
}

// CheckHealth performs health checks on Scoop.
func (a *Adapter) CheckHealth(ctx context.Context) (manager.Status, error) {
	// Run 'scoop status' to check for issues
	result, err := a.executor.Execute(ctx, scoopCommand, "status")
	if err != nil {
		a.logger.Warn(ctx, "Failed to check scoop status",
			output.Field{Key: "error", Value: err.Error()})
		return manager.StatusDegraded, nil
	}

	if result.ExitCode == 0 {
		// Check output for warnings
		outputLower := strings.ToLower(result.Stdout + result.Stderr)
		if strings.Contains(outputLower, "error") {
			return manager.StatusError, nil
		}
		if strings.Contains(outputLower, "warning") || strings.Contains(outputLower, "outdated") {
			return manager.StatusDegraded, nil
		}
		return manager.StatusHealthy, nil
	}

	return manager.StatusError, nil
}

// Update performs update operations on Scoop packages.
func (a *Adapter) Update(ctx context.Context, opts adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
	a.logger.Info(ctx, "Starting Scoop update",
		output.Field{Key: "dry_run", Value: opts.DryRun},
		output.Field{Key: "strategy", Value: string(opts.Strategy)})

	result := &adapterm.UpdateResult{
		Success:         true,
		UpdatedPackages: []string{},
		FailedPackages:  []string{},
	}

	// If dry-run, simulate
	if opts.DryRun {
		result.Message = "Dry-run: would update all scoop packages"
		a.logger.Info(ctx, "Dry-run mode: skipping actual update")
		return result, nil
	}

	// Fixed strategy: skip updates
	if opts.Strategy == adapterm.StrategyFixed {
		result.Message = "Strategy 'fixed': scoop update skipped"
		return result, nil
	}

	// First, update Scoop itself
	updateResult, err := a.executor.Execute(ctx, scoopCommand, "update")
	if err != nil {
		a.logger.Warn(ctx, "Failed to update scoop",
			output.Field{Key: "error", Value: err.Error()})
		// Continue anyway, try to upgrade packages
	}

	if updateResult != nil && updateResult.ExitCode != 0 {
		a.logger.Warn(ctx, "Scoop update returned non-zero exit code",
			output.Field{Key: "exit_code", Value: updateResult.ExitCode})
	}

	// Then upgrade all packages with 'scoop update *'
	upgradeResult, err := a.executor.Execute(ctx, scoopCommand, "update", "*")
	if err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("scoop update * failed: %v", err)
		return result, fmt.Errorf("scoop update failed: %w", err)
	}

	if upgradeResult.ExitCode != 0 {
		// Check if it's just "nothing to update"
		if strings.Contains(upgradeResult.Stdout, "up to date") ||
			strings.Contains(upgradeResult.Stdout, "Latest versions") {
			result.Message = "All packages are up to date"
			return result, nil
		}

		result.Success = false
		result.Message = fmt.Sprintf("scoop update failed: %s", upgradeResult.Stderr)
		return result, nil
	}

	// Parse updated packages from output
	// Scoop shows "Updating 'package' (version -> version)"
	lines := strings.SplitSeq(upgradeResult.Stdout, "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Updating '") {
			// Extract package name between quotes
			start := strings.Index(line, "'") + 1
			end := strings.Index(line[start:], "'")
			if end > 0 {
				result.UpdatedPackages = append(result.UpdatedPackages, line[start:start+end])
			}
		}
	}

	result.Message = fmt.Sprintf("%d packages updated successfully", len(result.UpdatedPackages))
	a.logger.Info(ctx, "Scoop update completed",
		output.Field{Key: "updated_packages", Value: len(result.UpdatedPackages)})

	return result, nil
}
