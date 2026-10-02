// Package mise implements the manager adapter for mise-managed tools.
package mise

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/cmdutil"
)

const command = "mise"

// Adapter manages tools already declared in mise configuration.
type Adapter struct {
	executor output.CommandExecutor
	logger   output.Logger
	home     func() (string, error)
}

// NewAdapter creates a mise adapter.
func NewAdapter(executor output.CommandExecutor, logger output.Logger) *Adapter {
	return &Adapter{executor: executor, logger: logger, home: os.UserHomeDir}
}

func locator() string {
	if runtime.GOOS == "windows" {
		return "where"
	}
	return "which"
}

// Detect checks whether mise is available on PATH without relying on a shell locator.
func (a *Adapter) Detect(ctx context.Context) (bool, error) {
	r, err := a.executor.Execute(ctx, command, "--version")
	return err == nil && r != nil && r.ExitCode == 0, nil
}

// GetVersion returns the installed mise version.
func (a *Adapter) GetVersion(ctx context.Context) (string, error) {
	r, err := a.executor.Execute(ctx, command, "--version")
	if resultErr := cmdutil.CheckResult(r, err, "get mise version"); resultErr != nil {
		return "", resultErr
	}
	value := strings.TrimPrefix(cmdutil.ExtractStdout(r), "mise ")
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return "", fmt.Errorf("get mise version: empty output")
	}
	return fields[0], nil
}

// GetBinaryPath returns the mise executable path.
func (a *Adapter) GetBinaryPath(ctx context.Context) (string, error) {
	r, err := a.executor.Execute(ctx, locator(), command)
	if resultErr := cmdutil.CheckResult(r, err, "locate mise binary"); resultErr != nil {
		return "", resultErr
	}
	return strings.TrimSpace(strings.Split(cmdutil.ExtractStdout(r), "\n")[0]), nil
}

// GetConfigPath returns the global mise config path.
func (a *Adapter) GetConfigPath(_ context.Context) (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "mise", "config.toml"), nil
	}
	home, err := a.home()
	if err != nil {
		return "", fmt.Errorf("resolve home directory for mise configuration: %w", err)
	}
	return filepath.Join(home, ".config", "mise", "config.toml"), nil
}

// ListPackages lists currently active mise tools.
func (a *Adapter) ListPackages(ctx context.Context) ([]manager.Package, error) {
	tools, err := a.list(ctx, adapterm.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	packages := make([]manager.Package, 0, len(tools))
	for _, tool := range tools {
		packages = append(packages, manager.Package{
			Name: tool.Name, CurrentVersion: tool.Version, Description: "mise-managed tool",
			IsGlobal: tool.Active, UpdateType: manager.UpdateNone,
		})
	}
	return packages, nil
}

// CheckHealth reports whether mise can run its version command.
func (a *Adapter) CheckHealth(ctx context.Context) (manager.Status, error) {
	r, err := a.executor.Execute(ctx, command, "--version")
	if err != nil || r == nil || r.ExitCode != 0 {
		return manager.StatusDegraded, nil //nolint:nilerr // Health probes report failures as degraded status.
	}
	return manager.StatusHealthy, nil
}

// ValidateUpdateOptions validates mise-specific update combinations before mutation.
func ValidateUpdateOptions(opts adapterm.UpdateOptions) error {
	if err := validatePackageSelection(opts.Packages); err != nil {
		return err
	}
	if opts.Strategy == adapterm.StrategyFixed && opts.Bump {
		return fmt.Errorf("mise update: --bump cannot be used with fixed strategy")
	}
	switch opts.Strategy {
	case "", adapterm.StrategyFixed, adapterm.StrategyStable, adapterm.StrategyMinor, adapterm.StrategyMicro, adapterm.StrategyLatest:
		return nil
	default:
		return fmt.Errorf("mise update: unsupported strategy %q", opts.Strategy)
	}
}

// Update applies an update policy to tools declared by mise. Planning policies
// resolve every target before the first upgrade command is executed.
func (a *Adapter) Update(ctx context.Context, opts adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
	result := &adapterm.UpdateResult{Success: true, UpdatedPackages: []string{}, FailedPackages: []string{}}
	if err := ValidateUpdateOptions(opts); err != nil {
		result.Success, result.Message = false, err.Error()
		return result, err
	}
	if opts.Strategy == adapterm.StrategyFixed {
		result.Message = "mise upgrades skipped (strategy fixed)"
		return result, nil
	}
	if effectiveStrategy(opts) == adapterm.StrategyStable {
		return a.nativeUpdate(ctx, opts, result)
	}
	return a.policyUpdate(ctx, opts, result)
}

func (a *Adapter) policyUpdate(ctx context.Context, opts adapterm.UpdateOptions, result *adapterm.UpdateResult) (*adapterm.UpdateResult, error) {
	tools, err := a.list(ctx, opts)
	if err != nil {
		result.Success, result.Message = false, err.Error()
		return result, err
	}
	plans, err := a.plan(ctx, opts, tools)
	if err != nil {
		result.Success, result.Message = false, err.Error()
		return result, err
	}
	if opts.DryRun {
		return a.dryRunPlans(ctx, opts, result, plans)
	}
	return a.executePlans(ctx, opts, result, plans)
}

func (a *Adapter) dryRunPlans(ctx context.Context, opts adapterm.UpdateOptions, result *adapterm.UpdateResult, plans []upgradePlan) (*adapterm.UpdateResult, error) {
	previews := make([]string, 0, len(plans))
	for _, plan := range plans {
		r, err := a.runPlan(ctx, opts, plan, true)
		if err != nil {
			return updateFailure(result, plan.Name, err)
		}
		previews = append(previews, plan.Name+" -> "+plan.Target+": "+preview(r.Stdout, r.Stderr))
	}
	result.Message = "Dry-run: " + strings.Join(previews, "; ")
	if len(plans) == 0 {
		result.Message = "Dry-run: mise tools already satisfy the selected update policy"
	}
	return result, nil
}

func (a *Adapter) executePlans(ctx context.Context, opts adapterm.UpdateOptions, result *adapterm.UpdateResult, plans []upgradePlan) (*adapterm.UpdateResult, error) {
	for _, plan := range plans {
		r, err := a.runPlan(ctx, opts, plan, false)
		if err != nil {
			return updateFailure(result, plan.Name, err)
		}
		result.UpdatedPackages = append(result.UpdatedPackages, plan.Name)
		result.Message = preview(r.Stdout, r.Stderr)
	}
	if len(plans) == 0 {
		result.Message = "mise tools already satisfy the selected update policy"
	}
	return result, nil
}

func (a *Adapter) runPlan(ctx context.Context, opts adapterm.UpdateOptions, plan upgradePlan, dryRun bool) (*output.ExecutionResult, error) {
	args := a.args(opts, "upgrade", plan.Name+"@"+plan.Target, "--no-prune")
	if dryRun {
		args = append(args, "--dry-run")
	}
	if opts.Bump {
		args = append(args, "--bump")
	}
	r, execErr := a.executor.Execute(ctx, command, args...)
	if err := cmdutil.CheckResult(r, execErr, "upgrade mise tool "+plan.Name); err != nil {
		return nil, err
	}
	return r, nil
}

func (a *Adapter) nativeUpdate(ctx context.Context, opts adapterm.UpdateOptions, result *adapterm.UpdateResult) (*adapterm.UpdateResult, error) {
	args := a.args(opts, "upgrade")
	if len(opts.Packages) != 0 {
		tools, err := a.list(ctx, opts)
		if err != nil {
			result.Success, result.Message = false, err.Error()
			return result, err
		}
		for _, tool := range tools {
			args = append(args, tool.Name)
		}
	}
	args = append(args, "--no-prune")
	if opts.Bump {
		args = append(args, "--bump")
	}
	if opts.DryRun {
		args = append(args, "--dry-run")
	}
	r, execErr := a.executor.Execute(ctx, command, args...)
	if err := cmdutil.CheckResult(r, execErr, "mise upgrade"); err != nil {
		result.Success, result.Message = false, err.Error()
		return result, err
	}
	result.Message = preview(r.Stdout, r.Stderr)
	if result.Message == "" && opts.DryRun {
		result.Message = "Dry-run: mise upgrade completed"
	}
	return result, nil
}

func (a *Adapter) args(opts adapterm.UpdateOptions, values ...string) []string {
	args := make([]string, 0, len(values)+3)
	if opts.MiseDir != "" {
		args = append(args, "-C", opts.MiseDir)
	}
	args = append(args, values...)
	if opts.MiseLocal {
		args = append(args, "--local")
	}
	return args
}

type miseTool struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Requested string `json:"requested_version"`
	Installed bool   `json:"installed"`
	Active    bool   `json:"active"`
}

func (a *Adapter) list(ctx context.Context, opts adapterm.UpdateOptions) ([]miseTool, error) {
	mode := "--current"
	if opts.MiseLocal {
		mode = "--local"
	}
	listOpts := opts
	listOpts.MiseLocal = false // --local is already the ls selection flag above.
	r, execErr := a.executor.Execute(ctx, command, a.args(listOpts, "ls", mode, "--json")...)
	if err := cmdutil.CheckResult(r, execErr, "list mise tools"); err != nil {
		return nil, err
	}
	tools, err := decodeTools(r.Stdout)
	if err != nil {
		return nil, err
	}
	filtered := tools[:0]
	for _, tool := range tools {
		if tool.Installed && tool.Active {
			filtered = append(filtered, tool)
		}
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].Name < filtered[j].Name })
	return selectTools(filtered, opts.Packages)
}

func decodeTools(data string) ([]miseTool, error) {
	var array []miseTool
	if err := json.Unmarshal([]byte(data), &array); err == nil {
		return array, nil
	}
	var grouped map[string][]miseTool
	if err := json.Unmarshal([]byte(data), &grouped); err != nil {
		return nil, fmt.Errorf("parse mise tools: %w", err)
	}
	tools := make([]miseTool, 0, len(grouped))
	for name, entries := range grouped {
		for _, entry := range entries {
			entry.Name = name
			tools = append(tools, entry)
		}
	}
	return tools, nil
}
