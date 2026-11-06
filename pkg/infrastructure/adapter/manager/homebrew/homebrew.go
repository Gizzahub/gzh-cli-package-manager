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
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/cmdutil"
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
	return cmdutil.IsCommandAvailable(result, err), nil
}

// GetVersion retrieves the version of the installed Homebrew.
func (a *Adapter) GetVersion(ctx context.Context) (string, error) {
	result, err := a.executor.Execute(ctx, "brew", "--version")
	if resultErr := cmdutil.CheckResult(result, err, "get brew version"); resultErr != nil {
		return "", resultErr
	}

	// Output format: "Homebrew 4.2.1\n..."
	lines := strings.Split(cmdutil.ExtractStdout(result), "\n")
	if len(lines) == 0 {
		return "", fmt.Errorf("get brew version: unexpected empty output")
	}

	return cmdutil.ExtractVersionField(lines[0], 1, "get brew version")
}

// GetBinaryPath returns the path to the Homebrew binary.
func (a *Adapter) GetBinaryPath(ctx context.Context) (string, error) {
	result, err := a.executor.Execute(ctx, "which", "brew")
	if resultErr := cmdutil.CheckResult(result, err, "find brew binary"); resultErr != nil {
		return "", resultErr
	}
	return cmdutil.ExtractStdout(result), nil
}

// GetConfigPath returns the path to the Homebrew configuration.
func (a *Adapter) GetConfigPath(ctx context.Context) (string, error) {
	result, err := a.executor.Execute(ctx, "brew", "--prefix")
	if resultErr := cmdutil.CheckResult(result, err, "get brew prefix"); resultErr != nil {
		return "", resultErr
	}
	return cmdutil.ExtractStdout(result), nil
}

// ListPackages retrieves all packages managed by Homebrew.
func (a *Adapter) ListPackages(ctx context.Context) ([]manager.Package, error) {
	// Use brew info --json=v2 --installed for detailed package information
	result, err := a.executor.Execute(ctx, "brew", "info", "--json=v2", "--installed")
	if resultErr := cmdutil.CheckResult(result, err, "list brew packages"); resultErr != nil {
		return nil, resultErr
	}

	// Parse JSON output
	var brewInfo struct {
		Formulae []struct {
			Name              string `json:"name"`
			FullName          string `json:"full_name"`
			Desc              string `json:"desc"`
			Version           string `json:"version"`
			InstalledOnDemand bool   `json:"installed_on_demand"`
			Versions          struct {
				Stable string `json:"stable"`
			} `json:"versions"`
		} `json:"formulae"`
		Casks []struct {
			Token   string   `json:"token"`
			Name    []string `json:"name"`
			Desc    string   `json:"desc"`
			Version string   `json:"version"`
		} `json:"casks"`
	}

	if err := cmdutil.UnmarshalJSON(result, &brewInfo, "parse brew packages"); err != nil {
		return nil, err
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
		return manager.StatusDegraded, nil //nolint:nilerr // CheckHealth reports probe failures as degraded status.
	}

	if result.ExitCode == 0 {
		return manager.StatusHealthy, nil
	}

	// Check if there are warnings (exit code 1) or errors
	outputStr := strings.ToLower(result.Stdout + result.Stderr)
	if strings.Contains(outputStr, "warning") {
		return manager.StatusDegraded, nil
	}

	return manager.StatusError, nil
}

// Update performs update operations on Homebrew and its packages.
func (a *Adapter) Update(ctx context.Context, opts adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
	a.logger.Info(ctx, "Starting Homebrew update",
		output.Field{Key: "dry_run", Value: opts.DryRun},
		output.Field{Key: "strategy", Value: string(opts.Strategy)})

	result := &adapterm.UpdateResult{
		Success:         true,
		UpdatedPackages: []string{},
		FailedPackages:  []string{},
	}

	// If dry-run, just simulate
	if opts.DryRun {
		return a.preview(ctx, opts.Packages, result)
	}

	// Step 1: Update Homebrew itself
	updateResult, err := a.executor.Execute(ctx, "brew", "update")
	if err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("brew update failed: %v", err)
		return result, fmt.Errorf("brew update failed: %w", err)
	}

	if updateResult.ExitCode != 0 {
		result.Success = false
		result.Message = fmt.Sprintf("brew update failed: %s", updateResult.Stderr)
		return result, fmt.Errorf("brew update failed with exit code %d", updateResult.ExitCode)
	}

	a.logger.Info(ctx, "Homebrew updated successfully")

	// Step 2: Upgrade packages (skip if strategy is "fixed")
	if opts.Strategy == adapterm.StrategyFixed {
		result.Message = "Strategy 'fixed': Homebrew updated, packages not upgraded"
		return result, nil
	}

	if err := a.upgrade(ctx, opts.Packages, result); err != nil {
		result.Success = false
		result.Message = err.Error()
		return result, err
	}

	result.Message = fmt.Sprintf("Homebrew and %d packages updated successfully", len(result.UpdatedPackages))
	a.logger.Info(ctx, "Homebrew update completed",
		output.Field{Key: "updated_packages", Value: len(result.UpdatedPackages)})

	return result, nil
}

func (a *Adapter) upgrade(ctx context.Context, packages []string, result *adapterm.UpdateResult) error {
	if len(packages) == 0 {
		upgradeResult, err := a.executor.Execute(ctx, "brew", "upgrade")
		if resultErr := cmdutil.CheckResult(upgradeResult, err, "brew upgrade"); resultErr != nil {
			return resultErr
		}
		appendUpgraded(result, upgradeResult.Stdout)
		return nil
	}
	for _, pkg := range packages {
		upgradeResult, err := a.executor.Execute(ctx, "brew", "upgrade", pkg)
		if resultErr := cmdutil.CheckResult(upgradeResult, err, "brew upgrade "+pkg); resultErr != nil {
			result.FailedPackages = append(result.FailedPackages, pkg)
			return resultErr
		}
		appendUpgraded(result, upgradeResult.Stdout)
	}
	return nil
}

func (a *Adapter) preview(ctx context.Context, packages []string, result *adapterm.UpdateResult) (*adapterm.UpdateResult, error) {
	args := []string{"outdated", "--formula", "--json=v2"}
	r, execErr := a.executor.Execute(ctx, "brew", args...)
	if err := cmdutil.CheckResult(r, execErr, "list outdated brew formulae"); err != nil {
		result.Success, result.Message = false, err.Error()
		return result, err
	}
	var payload struct {
		Formulae []struct {
			Name    string `json:"name"`
			Current string `json:"current_version"`
		} `json:"formulae"`
	}
	if err := json.Unmarshal([]byte(r.Stdout), &payload); err != nil {
		return result, fmt.Errorf("parse outdated brew formulae: %w", err)
	}
	selected := make(map[string]bool, len(packages))
	for _, pkg := range packages {
		selected[pkg] = true
	}
	available := make([]string, 0, len(payload.Formulae))
	for _, formula := range payload.Formulae {
		if len(selected) == 0 || selected[formula.Name] {
			available = append(available, formula.Name)
		}
	}
	if len(available) == 0 {
		result.Message = "Dry-run: selected Homebrew formulae are up to date"
	} else {
		result.Message = "Dry-run: available Homebrew formula updates: " + strings.Join(available, ", ")
	}
	return result, nil
}

func appendUpgraded(result *adapterm.UpdateResult, stdout string) {
	for line := range strings.SplitSeq(stdout, "\n") {
		parts := strings.Fields(strings.TrimSpace(line))
		if len(parts) >= 2 && parts[0] == "Upgrading" {
			result.UpdatedPackages = append(result.UpdatedPackages, parts[1])
		}
	}
}
