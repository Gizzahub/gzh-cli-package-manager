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
	scoopCommand   = "scoop"
	dryRunFieldKey = "dry_run"
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
	if resultErr := cmdutil.CheckResult(result, err, "get scoop version"); resultErr != nil {
		return "", resultErr
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
	if resultErr := cmdutil.CheckResult(result, err, "find scoop binary"); resultErr != nil {
		return "", resultErr
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
	if resultErr := cmdutil.CheckResult(result, err, "list scoop packages"); resultErr != nil {
		return nil, resultErr
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

// Search finds packages matching query via `scoop search`.
func (a *Adapter) Search(ctx context.Context, query string) ([]manager.Package, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("search scoop packages: query is required")
	}

	result, err := a.executor.Execute(ctx, scoopCommand, "search", query)
	if resultErr := cmdutil.CheckResult(result, err, "search scoop packages"); resultErr != nil {
		return nil, resultErr
	}

	return a.parseSearchOutput(result), nil
}

// parseSearchOutput parses `scoop search` text output.
// Common shapes:
//
//	Name   Version Source
//	----   ------- ------
//	git    2.43.0  main
//
// or "Results from other known buckets..." sections with "'name' (version)".
func (a *Adapter) parseSearchOutput(result *output.ExecutionResult) []manager.Package {
	lines := strings.Split(result.Stdout, "\n")
	var packages []manager.Package
	seen := make(map[string]struct{})

	startIdx := 0
	for i, line := range lines {
		if strings.Contains(line, "---") || strings.Contains(line, "===") {
			startIdx = i + 1
			break
		}
	}
	if startIdx == 0 && len(lines) > 1 {
		// Skip possible header "Name Version Source"
		if strings.Contains(strings.ToLower(lines[0]), "name") {
			startIdx = 1
		}
	}

	for i := startIdx; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		// Skip section headers
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "results from") || strings.HasPrefix(lower, "name") {
			continue
		}

		// Pattern: 'name' (version)
		if strings.HasPrefix(line, "'") {
			name, version := parseScoopQuotedResult(line)
			if name == "" {
				continue
			}
			key := name + "@" + version
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			packages = append(packages, manager.Package{
				Name:           name,
				CurrentVersion: version,
				IsGlobal:       false,
				UpdateType:     manager.UpdateNone,
				Manager:        manager.ManagerScoop,
			})
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := fields[0]
		version := fields[1]
		key := name + "@" + version
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		packages = append(packages, manager.Package{
			Name:           name,
			CurrentVersion: version,
			IsGlobal:       false,
			UpdateType:     manager.UpdateNone,
			Manager:        manager.ManagerScoop,
		})
	}

	return packages
}

// parseScoopQuotedResult extracts name and version from "'name' (version) ..." lines.
func parseScoopQuotedResult(line string) (string, string) {
	// 'git' (2.43.0)
	start := strings.Index(line, "'")
	if start < 0 {
		return "", ""
	}
	rest := line[start+1:]
	end := strings.Index(rest, "'")
	if end < 0 {
		return "", ""
	}
	name := rest[:end]
	version := ""
	if open := strings.Index(rest[end:], "("); open >= 0 {
		frag := rest[end+open+1:]
		if close := strings.Index(frag, ")"); close >= 0 {
			version = strings.TrimSpace(frag[:close])
		}
	}
	return name, version
}

// Install installs a package by Scoop name.
// dryRun skips the native install.
func (a *Adapter) Install(ctx context.Context, pkgID string, dryRun bool) error {
	pkgID = strings.TrimSpace(pkgID)
	if pkgID == "" {
		return fmt.Errorf("install scoop package: package name is required")
	}

	a.logger.Info(ctx, "Scoop install",
		output.Field{Key: "package", Value: pkgID},
		output.Field{Key: dryRunFieldKey, Value: dryRun})

	if dryRun {
		return nil
	}

	result, err := a.executor.Execute(ctx, scoopCommand, "install", pkgID)
	if resultErr := cmdutil.CheckResult(result, err, "install scoop package "+pkgID); resultErr != nil {
		return resultErr
	}
	return nil
}

// Uninstall removes a package by Scoop name.
// dryRun skips the native uninstall.
func (a *Adapter) Uninstall(ctx context.Context, pkgID string, dryRun bool) error {
	pkgID = strings.TrimSpace(pkgID)
	if pkgID == "" {
		return fmt.Errorf("uninstall scoop package: package name is required")
	}

	a.logger.Info(ctx, "Scoop uninstall",
		output.Field{Key: "package", Value: pkgID},
		output.Field{Key: dryRunFieldKey, Value: dryRun})

	if dryRun {
		return nil
	}

	result, err := a.executor.Execute(ctx, scoopCommand, "uninstall", pkgID)
	if resultErr := cmdutil.CheckResult(result, err, "uninstall scoop package "+pkgID); resultErr != nil {
		return resultErr
	}
	return nil
}

// ListBuckets returns configured Scoop buckets via `scoop bucket list`.
func (a *Adapter) ListBuckets(ctx context.Context) ([]adapterm.Bucket, error) {
	result, err := a.executor.Execute(ctx, scoopCommand, "bucket", "list")
	if resultErr := cmdutil.CheckResult(result, err, "list scoop buckets"); resultErr != nil {
		return nil, resultErr
	}
	return parseScoopBuckets(result.Stdout), nil
}

// AddBucket adds a Scoop bucket. url may be empty for known buckets.
func (a *Adapter) AddBucket(ctx context.Context, name, url string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("add scoop bucket: name is required")
	}

	args := []string{"bucket", "add", name}
	if strings.TrimSpace(url) != "" {
		args = append(args, strings.TrimSpace(url))
	}

	a.logger.Info(ctx, "Scoop bucket add",
		output.Field{Key: "name", Value: name},
		output.Field{Key: "url", Value: url})

	result, err := a.executor.Execute(ctx, scoopCommand, args...)
	if resultErr := cmdutil.CheckResult(result, err, "add scoop bucket "+name); resultErr != nil {
		return resultErr
	}
	return nil
}

// RemoveBucket removes a Scoop bucket by name.
func (a *Adapter) RemoveBucket(ctx context.Context, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("remove scoop bucket: name is required")
	}

	a.logger.Info(ctx, "Scoop bucket remove",
		output.Field{Key: "name", Value: name})

	result, err := a.executor.Execute(ctx, scoopCommand, "bucket", "rm", name)
	if resultErr := cmdutil.CheckResult(result, err, "remove scoop bucket "+name); resultErr != nil {
		return resultErr
	}
	return nil
}

// parseScoopBuckets parses `scoop bucket list` text output.
// Typical columns: Name Source Updated Manifests
//
//	Name   Source                                  Updated             Manifests
//	----   ------                                  -------             ---------
//	main   https://github.com/ScoopInstaller/Main  2024-01-01 00:00:00 1234
func parseScoopBuckets(stdout string) []adapterm.Bucket {
	lines := strings.Split(stdout, "\n")
	var buckets []adapterm.Bucket

	startIdx := 0
	for i, line := range lines {
		if strings.Contains(line, "---") || strings.Contains(line, "===") {
			startIdx = i + 1
			break
		}
	}
	if startIdx == 0 && len(lines) > 0 {
		if strings.Contains(strings.ToLower(lines[0]), "name") {
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
		b := adapterm.Bucket{Name: fields[0]}
		if len(fields) >= 2 {
			b.Source = fields[1]
		}
		buckets = append(buckets, b)
	}
	return buckets
}

// CheckHealth performs health checks on Scoop.
func (a *Adapter) CheckHealth(ctx context.Context) (manager.Status, error) {
	// Run 'scoop status' to check for issues
	result, err := a.executor.Execute(ctx, scoopCommand, "status")
	if err != nil {
		a.logger.Warn(ctx, "Failed to check scoop status",
			output.Field{Key: "error", Value: err.Error()})
		return manager.StatusDegraded, nil //nolint:nilerr // CheckHealth reports probe failures as degraded status.
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
		output.Field{Key: dryRunFieldKey, Value: opts.DryRun},
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
