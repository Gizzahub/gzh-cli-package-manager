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
  list    List installed packages
  search  Search the winget catalog

Examples:
  gz-pm winget list
  gz-pm winget search git
  gz-pm winget list --output json`,
	},
	{
		Use:   "scoop",
		ID:    manager.ManagerScoop,
		Short: "Scoop package manager operations",
		Long: `Run Scoop-specific package operations through the installed scoop adapter.

Subcommands:
  list    List installed packages
  search  Search scoop buckets

Examples:
  gz-pm scoop list
  gz-pm scoop search git
  gz-pm scoop search 7zip --output json`,
	},
	{
		Use:   "chocolatey",
		ID:    manager.ManagerChocolatey,
		Short: "Chocolatey package manager operations",
		Long: `Run Chocolatey-specific package operations through the installed choco adapter.

Subcommands:
  list    List installed packages (local)
  search  Search the Chocolatey community feed

Examples:
  gz-pm chocolatey list
  gz-pm chocolatey search git
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
	return cmd
}

func newPerManagerListCmd(spec perManagerSpec) *cobra.Command {
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "list",
		Short: fmt.Sprintf("List packages installed via %s", spec.Use),
		Long:  fmt.Sprintf("List packages managed by %s on this system.", spec.Use),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runPerManagerList(cmd.Context(), spec, outputFormat, cmd.OutOrStdout())
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "output", "o", "text", "Output format (text|json)")
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

	cmd.Flags().StringVarP(&outputFormat, "output", "o", "text", "Output format (text|json)")
	return cmd
}

// runPerManagerList lists installed packages for a manager after Detect succeeds.
func runPerManagerList(ctx context.Context, spec perManagerSpec, format string, out io.Writer) error {
	adapter, err := requireAdapter(spec)
	if err != nil {
		return err
	}
	if err := ensureDetected(ctx, adapter, spec.Use); err != nil {
		return err
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
	if err := ensureDetected(ctx, adapter, spec.Use); err != nil {
		return err
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
	case "json":
		resp := packagesResponse{
			Manager:  managerName,
			Action:   action,
			Count:    len(views),
			Packages: views,
		}
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	case "text", "":
		if len(views) == 0 {
			_, err := fmt.Fprintf(out, "No packages found (%s %s).\n", managerName, action)
			return err
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
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unknown output format %q (supported: text, json)", format)
	}
}

