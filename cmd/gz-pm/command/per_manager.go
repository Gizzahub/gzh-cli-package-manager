package command

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
	"github.com/spf13/cobra"
)

// managerAdapters holds adapters for per-manager CLI commands.
// Injected from main (or tests) via SetManagerAdapters.
var managerAdapters map[manager.ManagerID]adapterm.Adapter

const (
	perManagerInstallAction   = "install"
	perManagerUninstallAction = "uninstall"
)

// SetManagerAdapters injects package manager adapters for per-manager commands.
func SetManagerAdapters(adapters map[manager.ManagerID]adapterm.Adapter) {
	managerAdapters = adapters
}

// perManagerSpec describes a registered per-manager command group.
type perManagerSpec struct {
	// CLI name (e.g. "winget", "scoop", "chocolatey")
	Use string
	// Manager ID used for adapter lookup
	ID manager.ManagerID
	// Human-readable short description
	Short string
	// Longer help text
	Long string
}

var perManagerSpecs = []perManagerSpec{
	{
		Use:   "winget",
		ID:    manager.ManagerWinget,
		Short: "Windows Package Manager (winget) operations",
		Long: `Run winget-specific package operations through the installed winget adapter.

Subcommands:
  list              List installed packages
  search            Search the winget catalog
  install           Install a package by ID
  uninstall         Uninstall a package by ID
  upgrade           Upgrade packages (use --all for all)
  source list       List configured winget sources

Examples:
  gz-pm winget list
  gz-pm winget search git
  gz-pm winget install Git.Git --dry-run
  gz-pm winget uninstall Git.Git --dry-run
  gz-pm winget upgrade --all --dry-run
  gz-pm winget source list
  gz-pm winget list --output json`,
	},
	{
		Use:   "scoop",
		ID:    manager.ManagerScoop,
		Short: "Scoop package manager operations",
		Long: `Run Scoop-specific package operations through the installed scoop adapter.

Subcommands:
  list              List installed packages
  search            Search scoop buckets
  install           Install a package
  uninstall         Uninstall a package
  upgrade           Upgrade packages (use --all for all)
  bucket list|add|remove  Manage scoop buckets

Examples:
  gz-pm scoop list
  gz-pm scoop search git
  gz-pm scoop install git --dry-run
  gz-pm scoop uninstall git --dry-run
  gz-pm scoop upgrade --all --dry-run
  gz-pm scoop bucket list
  gz-pm scoop bucket add extras
  gz-pm scoop bucket remove extras
  gz-pm scoop search 7zip --output json`,
	},
	{
		Use:   "chocolatey",
		ID:    manager.ManagerChocolatey,
		Short: "Chocolatey package manager operations",
		Long: `Run Chocolatey-specific package operations through the installed choco adapter.

Subcommands:
  list       List installed packages (local)
  search     Search the Chocolatey community feed
  install    Install a package (may require Administrator)
  uninstall  Uninstall a package (may require Administrator)
  upgrade    Upgrade packages (use --all for all)

Examples:
  gz-pm chocolatey list
  gz-pm chocolatey search git
  gz-pm chocolatey install git --dry-run
  gz-pm chocolatey uninstall git --dry-run
  gz-pm chocolatey upgrade --all --dry-run
  gz-pm chocolatey list --output json`,
	},
}

func init() {
	for _, spec := range perManagerSpecs {
		rootCmd.AddCommand(newPerManagerCmd(spec))
	}
}

func newPerManagerCmd(spec perManagerSpec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   spec.Use,
		Short: spec.Short,
		Long:  spec.Long,
	}

	cmd.AddCommand(newPerManagerListCmd(spec))
	cmd.AddCommand(newPerManagerSearchCmd(spec))
	cmd.AddCommand(newPerManagerInstallCmd(spec))
	cmd.AddCommand(newPerManagerUninstallCmd(spec))
	cmd.AddCommand(newPerManagerUpgradeCmd(spec))

	// Manager-specific subtrees
	switch spec.ID {
	case manager.ManagerWinget:
		cmd.AddCommand(newWingetSourceCmd(spec))
	case manager.ManagerScoop:
		cmd.AddCommand(newScoopBucketCmd(spec))
	}

	return cmd
}

func newPerManagerListCmd(spec perManagerSpec) *cobra.Command {
	var outputFormat string

	cmd := &cobra.Command{
		Use:   listCommand,
		Short: fmt.Sprintf("List packages installed via %s", spec.Use),
		Long:  fmt.Sprintf("List packages managed by %s on this system.", spec.Use),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runPerManagerList(cmd.Context(), spec, outputFormat, cmd.OutOrStdout())
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "output", "o", outputFormatText, "Output format (text|json)")
	return cmd
}

func newPerManagerSearchCmd(spec perManagerSpec) *cobra.Command {
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: fmt.Sprintf("Search packages available via %s", spec.Use),
		Long:  fmt.Sprintf("Search the %s catalog/sources for packages matching the query.", spec.Use),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPerManagerSearch(cmd.Context(), spec, args[0], outputFormat, cmd.OutOrStdout())
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "output", "o", outputFormatText, "Output format (text|json)")
	return cmd
}

func newPerManagerInstallCmd(spec perManagerSpec) *cobra.Command {
	return newPerManagerPackageActionCmd(spec, perManagerInstallAction, "Install", "installed", runPerManagerInstall)
}

func newPerManagerUninstallCmd(spec perManagerSpec) *cobra.Command {
	return newPerManagerPackageActionCmd(spec, perManagerUninstallAction, "Uninstall", "uninstalled", runPerManagerUninstall)
}

type perManagerPackageActionRunner func(context.Context, perManagerSpec, string, bool, io.Writer) error

func newPerManagerPackageActionCmd(spec perManagerSpec, action, displayAction, dryRunDescription string, run perManagerPackageActionRunner) *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   action + " <id>",
		Short: fmt.Sprintf("%s a package via %s", displayAction, spec.Use),
		Long:  fmt.Sprintf("%s a package by ID/name using %s. Use --dry-run to preview.", displayAction, spec.Use),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd.Context(), spec, args[0], dryRun, cmd.OutOrStdout())
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be "+dryRunDescription+" without making changes")
	return cmd
}

func newPerManagerUpgradeCmd(spec perManagerSpec) *cobra.Command {
	var dryRun bool
	var all bool

	cmd := &cobra.Command{
		Use:   "upgrade [id]",
		Short: fmt.Sprintf("Upgrade packages via %s", spec.Use),
		Long: fmt.Sprintf(`Upgrade packages managed by %s via Adapter.Update.

With --all (or no package id), upgrades all packages.
With a package id, upgrades that package only when the adapter supports package lists.
Use --dry-run to preview.`, spec.Use),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pkgID := ""
			if len(args) == 1 {
				pkgID = args[0]
			}
			if pkgID == "" {
				all = true
			}
			return runPerManagerUpgrade(cmd.Context(), spec, pkgID, all, dryRun, cmd.OutOrStdout())
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be upgraded without making changes")
	cmd.Flags().BoolVar(&all, "all", false, "Upgrade all packages")
	return cmd
}

func newWingetSourceCmd(spec perManagerSpec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "source",
		Short: "Manage winget package sources",
	}
	cmd.AddCommand(newWingetSourceListCmd(spec))
	return cmd
}

func newWingetSourceListCmd(spec perManagerSpec) *cobra.Command {
	var outputFormat string

	cmd := &cobra.Command{
		Use:   listCommand,
		Short: "List winget package sources",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runWingetSourceList(cmd.Context(), spec, outputFormat, cmd.OutOrStdout())
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "output", "o", outputFormatText, "Output format (text|json)")
	return cmd
}

func newScoopBucketCmd(spec perManagerSpec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bucket",
		Short: "Manage Scoop buckets",
	}
	cmd.AddCommand(newScoopBucketListCmd(spec))
	cmd.AddCommand(newScoopBucketAddCmd(spec))
	cmd.AddCommand(newScoopBucketRemoveCmd(spec))
	return cmd
}

func newScoopBucketListCmd(spec perManagerSpec) *cobra.Command {
	var outputFormat string

	cmd := &cobra.Command{
		Use:   listCommand,
		Short: "List Scoop buckets",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runScoopBucketList(cmd.Context(), spec, outputFormat, cmd.OutOrStdout())
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "output", "o", outputFormatText, "Output format (text|json)")
	return cmd
}

func newScoopBucketAddCmd(spec perManagerSpec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <name> [url]",
		Short: "Add a Scoop bucket",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := ""
			if len(args) == 2 {
				url = args[1]
			}
			return runScoopBucketAdd(cmd.Context(), spec, args[0], url, cmd.OutOrStdout())
		},
	}
	return cmd
}

func newScoopBucketRemoveCmd(spec perManagerSpec) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove <name>",
		Aliases: []string{"rm"},
		Short:   "Remove a Scoop bucket",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScoopBucketRemove(cmd.Context(), spec, args[0], cmd.OutOrStdout())
		},
	}
	return cmd
}

// runPerManagerList lists installed packages for a manager after Detect succeeds.
func runPerManagerList(ctx context.Context, spec perManagerSpec, format string, out io.Writer) error {
	adapter, err := requireAdapter(spec)
	if err != nil {
		return err
	}
	if detectErr := ensureDetected(ctx, adapter, spec.Use); detectErr != nil {
		return detectErr
	}

	packages, err := adapter.ListPackages(ctx)
	if err != nil {
		return fmt.Errorf("%s list failed: %w", spec.Use, err)
	}

	return writePackages(out, format, spec.Use, "list", packages)
}

// runPerManagerSearch searches packages for a manager after Detect succeeds.
func runPerManagerSearch(ctx context.Context, spec perManagerSpec, query, format string, out io.Writer) error {
	adapter, err := requireAdapter(spec)
	if err != nil {
		return err
	}
	if detectErr := ensureDetected(ctx, adapter, spec.Use); detectErr != nil {
		return detectErr
	}

	searcher, ok := adapter.(adapterm.Searcher)
	if !ok {
		return fmt.Errorf("%s does not support search", spec.Use)
	}

	packages, err := searcher.Search(ctx, query)
	if err != nil {
		return fmt.Errorf("%s search failed: %w", spec.Use, err)
	}

	return writePackages(out, format, spec.Use, "search", packages)
}

// runPerManagerInstall installs a package after Detect succeeds.
func runPerManagerInstall(ctx context.Context, spec perManagerSpec, pkgID string, dryRun bool, out io.Writer) error {
	return runPerManagerPackageAction(ctx, spec, pkgID, dryRun, out, perManagerInstallAction, "Installed", adapterm.Installer.Install)
}

// runPerManagerUninstall uninstalls a package after Detect succeeds.
func runPerManagerUninstall(ctx context.Context, spec perManagerSpec, pkgID string, dryRun bool, out io.Writer) error {
	return runPerManagerPackageAction(ctx, spec, pkgID, dryRun, out, perManagerUninstallAction, "Uninstalled", adapterm.Installer.Uninstall)
}

type perManagerPackageActionInvoker func(adapterm.Installer, context.Context, string, bool) error

func runPerManagerPackageAction(
	ctx context.Context,
	spec perManagerSpec,
	pkgID string,
	dryRun bool,
	out io.Writer,
	action string,
	completedAction string,
	invoke perManagerPackageActionInvoker,
) error {
	adapter, err := requireAdapter(spec)
	if err != nil {
		return err
	}
	if err := ensureDetected(ctx, adapter, spec.Use); err != nil {
		return err
	}

	installer, ok := adapter.(adapterm.Installer)
	if !ok {
		return fmt.Errorf("%s does not support %s", spec.Use, action)
	}

	if dryRun {
		_, _ = fmt.Fprintf(out, "Dry-run: would %s %q via %s\n", action, pkgID, spec.Use)
	}

	if err := invoke(installer, ctx, pkgID, dryRun); err != nil {
		return fmt.Errorf("%s %s failed: %w", spec.Use, action, err)
	}

	if !dryRun {
		_, _ = fmt.Fprintf(out, "%s %q via %s\n", completedAction, pkgID, spec.Use)
	}
	return nil
}

// runPerManagerUpgrade upgrades packages via Adapter.Update.
func runPerManagerUpgrade(ctx context.Context, spec perManagerSpec, pkgID string, all, dryRun bool, out io.Writer) error {
	adapter, err := requireAdapter(spec)
	if err != nil {
		return err
	}
	if detectErr := ensureDetected(ctx, adapter, spec.Use); detectErr != nil {
		return detectErr
	}

	opts := adapterm.UpdateOptions{
		DryRun:   dryRun,
		Strategy: adapterm.StrategyLatest,
	}
	if pkgID != "" && !all {
		opts.Packages = []string{pkgID}
	}

	if dryRun {
		if pkgID != "" && !all {
			_, _ = fmt.Fprintf(out, "Dry-run: would upgrade %q via %s\n", pkgID, spec.Use)
		} else {
			_, _ = fmt.Fprintf(out, "Dry-run: would upgrade all packages via %s\n", spec.Use)
		}
	}

	result, err := adapter.Update(ctx, opts)
	if err != nil {
		return fmt.Errorf("%s upgrade failed: %w", spec.Use, err)
	}

	if result != nil {
		if result.Message != "" {
			_, _ = fmt.Fprintln(out, result.Message)
		}
		if !result.Success {
			return fmt.Errorf("%s upgrade failed: %s", spec.Use, result.Message)
		}
		if len(result.UpdatedPackages) > 0 {
			_, _ = fmt.Fprintf(out, "Updated: %s\n", strings.Join(result.UpdatedPackages, ", "))
		}
	}
	return nil
}

func runWingetSourceList(ctx context.Context, spec perManagerSpec, format string, out io.Writer) error {
	adapter, err := requireAdapter(spec)
	if err != nil {
		return err
	}
	if detectErr := ensureDetected(ctx, adapter, spec.Use); detectErr != nil {
		return detectErr
	}

	lister, ok := adapter.(adapterm.SourceLister)
	if !ok {
		return fmt.Errorf("%s does not support source list", spec.Use)
	}

	sources, err := lister.ListSources(ctx)
	if err != nil {
		return fmt.Errorf("%s source list failed: %w", spec.Use, err)
	}

	return writeSources(out, format, sources)
}

func runScoopBucketList(ctx context.Context, spec perManagerSpec, format string, out io.Writer) error {
	adapter, err := requireAdapter(spec)
	if err != nil {
		return err
	}
	if detectErr := ensureDetected(ctx, adapter, spec.Use); detectErr != nil {
		return detectErr
	}

	bm, ok := adapter.(adapterm.BucketManager)
	if !ok {
		return fmt.Errorf("%s does not support bucket management", spec.Use)
	}

	buckets, err := bm.ListBuckets(ctx)
	if err != nil {
		return fmt.Errorf("%s bucket list failed: %w", spec.Use, err)
	}

	return writeBuckets(out, format, buckets)
}

func runScoopBucketAdd(ctx context.Context, spec perManagerSpec, name, url string, out io.Writer) error {
	adapter, err := requireAdapter(spec)
	if err != nil {
		return err
	}
	if err := ensureDetected(ctx, adapter, spec.Use); err != nil {
		return err
	}

	bm, ok := adapter.(adapterm.BucketManager)
	if !ok {
		return fmt.Errorf("%s does not support bucket management", spec.Use)
	}

	if err := bm.AddBucket(ctx, name, url); err != nil {
		return fmt.Errorf("%s bucket add failed: %w", spec.Use, err)
	}
	_, _ = fmt.Fprintf(out, "Added scoop bucket %q\n", name)
	return nil
}

func runScoopBucketRemove(ctx context.Context, spec perManagerSpec, name string, out io.Writer) error {
	adapter, err := requireAdapter(spec)
	if err != nil {
		return err
	}
	if err := ensureDetected(ctx, adapter, spec.Use); err != nil {
		return err
	}

	bm, ok := adapter.(adapterm.BucketManager)
	if !ok {
		return fmt.Errorf("%s does not support bucket management", spec.Use)
	}

	if err := bm.RemoveBucket(ctx, name); err != nil {
		return fmt.Errorf("%s bucket remove failed: %w", spec.Use, err)
	}
	_, _ = fmt.Fprintf(out, "Removed scoop bucket %q\n", name)
	return nil
}

func writeSources(out io.Writer, format string, sources []adapterm.Source) error {
	switch strings.ToLower(format) {
	case outputFormatJSON:
		type srcView struct {
			Name string `json:"name"`
			Arg  string `json:"arg,omitempty"`
		}
		views := make([]srcView, 0, len(sources))
		for _, s := range sources {
			views = append(views, srcView{Name: s.Name, Arg: s.Arg})
		}
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return wrapOutputError("sources", enc.Encode(map[string]any{
			"count":   len(views),
			"sources": views,
		}))
	case outputFormatText, "":
		if len(sources) == 0 {
			_, err := fmt.Fprintln(out, "No sources configured.")
			return wrapOutputError("sources", err)
		}
		_, _ = fmt.Fprintf(out, "winget sources — %d\n", len(sources))
		for _, s := range sources {
			if s.Arg != "" {
				_, _ = fmt.Fprintf(out, "  %s  %s\n", s.Name, s.Arg)
			} else {
				_, _ = fmt.Fprintf(out, "  %s\n", s.Name)
			}
		}
		return nil
	default:
		return fmt.Errorf("unknown output format %q (supported: text, json)", format)
	}
}

func writeBuckets(out io.Writer, format string, buckets []adapterm.Bucket) error {
	switch strings.ToLower(format) {
	case outputFormatJSON:
		type bucketView struct {
			Name   string `json:"name"`
			Source string `json:"source,omitempty"`
		}
		views := make([]bucketView, 0, len(buckets))
		for _, b := range buckets {
			views = append(views, bucketView{Name: b.Name, Source: b.Source})
		}
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return wrapOutputError("buckets", enc.Encode(map[string]any{
			"count":   len(views),
			"buckets": views,
		}))
	case outputFormatText, "":
		if len(buckets) == 0 {
			_, err := fmt.Fprintln(out, "No buckets configured.")
			return wrapOutputError("buckets", err)
		}
		_, _ = fmt.Fprintf(out, "scoop buckets — %d\n", len(buckets))
		for _, b := range buckets {
			if b.Source != "" {
				_, _ = fmt.Fprintf(out, "  %s  %s\n", b.Name, b.Source)
			} else {
				_, _ = fmt.Fprintf(out, "  %s\n", b.Name)
			}
		}
		return nil
	default:
		return fmt.Errorf("unknown output format %q (supported: text, json)", format)
	}
}

func requireAdapter(spec perManagerSpec) (adapterm.Adapter, error) {
	if managerAdapters == nil {
		return nil, fmt.Errorf("%s: adapters not initialized", spec.Use)
	}
	adapter, ok := managerAdapters[spec.ID]
	if !ok || adapter == nil {
		return nil, fmt.Errorf("%s: adapter not registered", spec.Use)
	}
	return adapter, nil
}

func ensureDetected(ctx context.Context, adapter adapterm.Adapter, name string) error {
	detected, err := adapter.Detect(ctx)
	if err != nil {
		return fmt.Errorf("%s: detect failed: %w", name, err)
	}
	if !detected {
		return fmt.Errorf("%s is not available on this system (not installed or not in PATH)", name)
	}
	return nil
}

// packageView is the JSON/text presentation shape for list/search results.
type packageView struct {
	Name             string `json:"name"`
	CurrentVersion   string `json:"current_version,omitempty"`
	AvailableVersion string `json:"available_version,omitempty"`
	Description      string `json:"description,omitempty"`
	Manager          string `json:"manager,omitempty"`
}

type packagesResponse struct {
	Manager  string        `json:"manager"`
	Action   string        `json:"action"`
	Count    int           `json:"count"`
	Packages []packageView `json:"packages"`
}

func writePackages(out io.Writer, format, managerName, action string, packages []manager.Package) error {
	views := make([]packageView, 0, len(packages))
	for _, p := range packages {
		views = append(views, packageView{
			Name:             p.Name,
			CurrentVersion:   p.CurrentVersion,
			AvailableVersion: p.AvailableVersion,
			Description:      p.Description,
			Manager:          string(p.Manager),
		})
	}

	switch strings.ToLower(format) {
	case outputFormatJSON:
		resp := packagesResponse{
			Manager:  managerName,
			Action:   action,
			Count:    len(views),
			Packages: views,
		}
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return wrapOutputError("packages", enc.Encode(resp))
	case outputFormatText, "":
		if len(views) == 0 {
			_, err := fmt.Fprintf(out, "No packages found (%s %s).\n", managerName, action)
			return wrapOutputError("packages", err)
		}
		_, _ = fmt.Fprintf(out, "%s %s — %d package(s)\n", managerName, action, len(views))
		for _, p := range views {
			version := p.CurrentVersion
			if version == "" {
				version = "-"
			}
			line := fmt.Sprintf("  %s  %s", p.Name, version)
			if p.AvailableVersion != "" && p.AvailableVersion != p.CurrentVersion {
				line += fmt.Sprintf(" → %s", p.AvailableVersion)
			}
			if _, err := fmt.Fprintln(out, line); err != nil {
				return wrapOutputError("packages", err)
			}
		}
		return nil
	default:
		return fmt.Errorf("unknown output format %q (supported: text, json)", format)
	}
}

func wrapOutputError(kind string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("write %s output: %w", kind, err)
}
