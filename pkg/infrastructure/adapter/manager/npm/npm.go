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
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/cmdutil"
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
	return cmdutil.IsCommandAvailable(result, err), nil
}

// GetVersion retrieves the version of the installed NPM.
func (a *Adapter) GetVersion(ctx context.Context) (string, error) {
	result, err := a.executor.Execute(ctx, "npm", "--version")
	if resultErr := cmdutil.CheckResult(result, err, "get npm version"); resultErr != nil {
		return "", resultErr
	}
	return cmdutil.ExtractStdout(result), nil
}

// GetBinaryPath returns the path to the NPM binary.
func (a *Adapter) GetBinaryPath(ctx context.Context) (string, error) {
	result, err := a.executor.Execute(ctx, "which", "npm")
	if resultErr := cmdutil.CheckResult(result, err, "find npm binary"); resultErr != nil {
		return "", resultErr
	}
	return cmdutil.ExtractStdout(result), nil
}

// GetConfigPath returns the path to the NPM configuration.
func (a *Adapter) GetConfigPath(ctx context.Context) (string, error) {
	// Get npm config prefix (global install location)
	result, err := a.executor.Execute(ctx, "npm", "config", "get", "prefix")
	if resultErr := cmdutil.CheckResult(result, err, "get npm config prefix"); resultErr != nil {
		return "", resultErr
	}
	return cmdutil.ExtractStdout(result), nil
}

// ListPackages retrieves all globally installed packages managed by NPM.
func (a *Adapter) ListPackages(ctx context.Context) ([]manager.Package, error) {
	// Get list of globally installed packages
	result, err := a.executor.Execute(ctx, "npm", "list", "-g", "--depth=0", "--json")
	if resultErr := cmdutil.CheckResult(result, err, "list npm packages"); resultErr != nil {
		return nil, resultErr
	}

	// Parse JSON output
	var npmList struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}

	if resultErr := cmdutil.UnmarshalJSON(result, &npmList, "parse npm packages"); resultErr != nil {
		return nil, resultErr
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
// Update updates global npm packages (npm update -g).
func (a *Adapter) Update(ctx context.Context, opts adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
	a.logger.Info(ctx, "Starting npm update",
		output.Field{Key: "dry_run", Value: opts.DryRun},
		output.Field{Key: "strategy", Value: string(opts.Strategy)})

	result := &adapterm.UpdateResult{
		Success:         true,
		UpdatedPackages: []string{},
		FailedPackages:  []string{},
	}

	if opts.DryRun {
		result.Message = "Dry-run: would run npm update -g"
		return result, nil
	}
	if opts.Strategy == adapterm.StrategyFixed {
		result.Message = "Strategy 'fixed': npm update skipped"
		return result, nil
	}

	args := []string{"update", "-g"}
	if len(opts.Packages) > 0 {
		args = append(args, opts.Packages...)
	}
	execResult, err := a.executor.Execute(ctx, "npm", args...)
	if err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("npm update failed: %v", err)
		return result, fmt.Errorf("npm update failed: %w", err)
	}
	if execResult.ExitCode != 0 {
		result.Success = false
		result.Message = fmt.Sprintf("npm update failed: %s", execResult.Stderr)
		return result, nil
	}
	result.Message = "npm global packages updated"
	return result, nil
}
