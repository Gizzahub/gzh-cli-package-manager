package command

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/diagnostics"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
	infradiag "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/diagnostics"
)

// duplicatesCmd reports cross-provider command installs. It is a child of
// status and does not change the manager summary.
var duplicatesCmd = &cobra.Command{
	Use:   "duplicates",
	Short: "Report commands installed by more than one provider",
	Long: `Report command installs that come from more than one provider.

The report is read-only. A successful report exits 0. Collection failure
exits non-zero. --strict exits non-zero only when a warning is present.
Commands from a single provider are omitted.

Examples:
  gz-pm status duplicates
  gz-pm status duplicates --output json
  gz-pm status duplicates --strict`,
	RunE: runDuplicates,
}

var (
	duplicatesOutput = outputFormatText
	duplicatesStrict bool
)

// duplicateEnvironment is the home root and search path tests inject.
// Production leaves it unset and reads the user home and process PATH.
type duplicateEnvironment struct {
	set    bool
	home   string
	search []string
}

var injectedDuplicateEnvironment duplicateEnvironment

func init() {
	duplicatesCmd.Flags().StringVarP(&duplicatesOutput, "output", "o", outputFormatText, "Output format (text|json)")
	duplicatesCmd.Flags().BoolVar(&duplicatesStrict, "strict", false, "Exit non-zero when a warning is present")
	statusCmd.AddCommand(duplicatesCmd)
}

// setDuplicateEnvironment injects the home root and search path used by tests.
func setDuplicateEnvironment(home string, search []string) {
	copied := make([]string, len(search))
	copy(copied, search)
	injectedDuplicateEnvironment = duplicateEnvironment{set: true, home: home, search: copied}
}

// clearDuplicateEnvironment restores production environment lookup.
func clearDuplicateEnvironment() {
	injectedDuplicateEnvironment = duplicateEnvironment{}
}

func runDuplicates(cmd *cobra.Command, _ []string) error {
	home, search, err := currentDuplicateEnvironment()
	if err != nil {
		return err
	}
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	findings, err := collectDuplicateFindings(ctx, managerAdapters, home, search)
	if err != nil {
		return err
	}
	if err := writeDuplicateReport(cmd.OutOrStdout(), duplicatesOutput, findings); err != nil {
		return err
	}
	if duplicatesStrict && hasDuplicateWarning(findings) {
		return fmt.Errorf("duplicate warning present")
	}
	return nil
}

func currentDuplicateEnvironment() (home string, search []string, err error) {
	if injectedDuplicateEnvironment.set {
		return injectedDuplicateEnvironment.home, injectedDuplicateEnvironment.search, nil
	}
	home, err = os.UserHomeDir()
	if err != nil {
		return "", nil, fmt.Errorf("resolve home: %w", err)
	}
	return home, filepath.SplitList(os.Getenv("PATH")), nil
}

// collectDuplicateFindings gathers manager command records, resolves them,
// appends the unmanaged probe, and classifies with no owner policy.
func collectDuplicateFindings(ctx context.Context, adapters map[manager.ManagerID]adapterm.Adapter, home string, search []string) ([]diagnostics.CommandFinding, error) {
	var records []diagnostics.InstallRecord
	for _, adapter := range adapters {
		found, err := adapterm.CollectCommands(ctx, adapter)
		if err != nil {
			return nil, fmt.Errorf("collect commands: %w", err)
		}
		records = append(records, found...)
	}
	resolved := infradiag.ResolvePathIdentity(records, search)
	resolved = append(resolved, infradiag.ProbeUnmanagedInstalls(home, search)...)
	findings := diagnostics.ClassifyCommands(resolved, nil)
	kept := make([]diagnostics.CommandFinding, 0, len(findings))
	for i := range findings {
		if findings[i].Kind == diagnostics.FindingNone {
			continue
		}
		kept = append(kept, findings[i])
	}
	return kept, nil
}

func hasDuplicateWarning(findings []diagnostics.CommandFinding) bool {
	for i := range findings {
		if findings[i].Kind == diagnostics.FindingWarning {
			return true
		}
	}
	return false
}

func writeDuplicateReport(out io.Writer, format string, findings []diagnostics.CommandFinding) error {
	switch format {
	case outputFormatJSON:
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(findings); err != nil {
			return fmt.Errorf("encode duplicate report: %w", err)
		}
		return nil
	case outputFormatText:
		return writeDuplicateText(out, findings)
	default:
		return fmt.Errorf("unknown output format %q (supported: text, json)", format)
	}
}

func writeDuplicateText(out io.Writer, findings []diagnostics.CommandFinding) error {
	for i := range findings {
		if err := writeDuplicateFinding(out, &findings[i]); err != nil {
			return err
		}
	}
	return nil
}

func writeDuplicateFinding(out io.Writer, finding *diagnostics.CommandFinding) error {
	if err := writeDuplicateLine(out, "command: %s\nkind: %s\n", finding.Command, finding.Kind); err != nil {
		return err
	}
	if err := writeDuplicateReasons(out, finding.Reasons); err != nil {
		return err
	}
	if err := writeDuplicateProviders(out, finding.Providers); err != nil {
		return err
	}
	return writeDuplicateVersions(out, finding.Versions)
}

func writeDuplicateReasons(out io.Writer, reasons []string) error {
	for _, reason := range reasons {
		if err := writeDuplicateLine(out, "reason: %s\n", reason); err != nil {
			return err
		}
	}
	return nil
}

func writeDuplicateProviders(out io.Writer, providers []diagnostics.ProviderFacts) error {
	for i := range providers {
		if err := writeDuplicateProvider(out, &providers[i]); err != nil {
			return err
		}
	}
	return nil
}

func writeDuplicateProvider(out io.Writer, provider *diagnostics.ProviderFacts) error {
	err := writeDuplicateLine(out, "provider: %s\nprovider-kind: %s\nprovider-active: %t\nprovider-executed: %t\n",
		provider.ProviderID, provider.Kind, provider.Active, provider.Executed)
	if err != nil {
		return err
	}
	for _, version := range provider.Versions {
		if err := writeDuplicateLine(out, "provider-version: %s\n", version); err != nil {
			return err
		}
	}
	for _, path := range provider.RealPaths {
		if err := writeDuplicateLine(out, "provider-path: %s\n", path); err != nil {
			return err
		}
	}
	return nil
}

func writeDuplicateVersions(out io.Writer, versions []diagnostics.VersionFact) error {
	for i := range versions {
		version := &versions[i]
		err := writeDuplicateLine(out,
			"version-executed-provider: %s\nversion-other-provider: %s\nversion-executed: %s\nversion-other: %s\nversion-comparison: %s\n",
			version.ExecutedProviderID, version.OtherProviderID, version.ExecutedVersion, version.OtherVersion, version.Comparison)
		if err != nil {
			return err
		}
	}
	return nil
}

func writeDuplicateLine(out io.Writer, format string, args ...any) error {
	if _, err := fmt.Fprintf(out, format, args...); err != nil {
		return fmt.Errorf("write duplicate report: %w", err)
	}
	return nil
}
