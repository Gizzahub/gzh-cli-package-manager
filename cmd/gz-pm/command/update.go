package command

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/dto"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/input"
)

var (
	updateAll           bool
	updateDryRun        bool
	updateManagers      string
	updateStrategy      string
	updateOutput        string
	updatePipAllowConda bool
	updateBump          bool
	updateMiseDir       string
	updateMiseLocal     bool
	updateMiseTools     []string
	updateConfigPath    string
	updateUseCase       input.UpdateUseCase
)

// SetUpdateUseCase injects the update use case dependency.
func SetUpdateUseCase(uc input.UpdateUseCase) {
	updateUseCase = uc
}

// updateCmd represents the update command.
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update package managers and packages",
	Long: `Update package managers and their packages.

Examples:
  # Update all package managers
  gz-pm update --all

  # Preview changes without executing
  gz-pm update --all --dry-run

  # Update specific managers only
  gz-pm update --managers brew,asdf,npm

  # Use specific update strategy
  gz-pm update --all --strategy stable`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		req, err := buildUpdateRequest(cmd)
		if err != nil {
			return err
		}
		if updateUseCase == nil {
			return fmt.Errorf("update use case not initialized")
		}

		ctx := cmd.Context()

		// Execute update
		resp, err := updateUseCase.Update(ctx, req)
		if err != nil {
			return fmt.Errorf("update managers: %w", err)
		}

		// Display results
		switch updateOutput {
		case outputFormatJSON:
			displayUpdateJSON(resp)
		case outputFormatText:
			displayUpdateText(resp)
		default:
			return fmt.Errorf("unknown output format %q (supported: text, json)", updateOutput)
		}

		// Exit with appropriate code
		if resp.Summary.FailedManagers > 0 {
			if resp.Summary.SuccessfulManagers > 0 {
				os.Exit(1) // Partial failure
			} else {
				os.Exit(2) // Complete failure
			}
		}
		return nil
	},
}

func displayUpdateText(resp *dto.UpdateResponse) {
	if resp.DryRun {
		fmt.Println("🧪 Package Manager Update (DRY-RUN)")
	} else {
		fmt.Println("📦 Package Manager Update")
	}
	fmt.Println()

	// Display individual manager results
	for _, result := range resp.Results {
		statusIcon := "✅"
		if result.Skipped {
			statusIcon = "⚠️"
		} else if !result.Success {
			statusIcon = "❌"
		}

		fmt.Printf("%s %s\n", statusIcon, result.Name)
		switch {
		case result.Skipped:
			fmt.Printf("   Skipped: %s\n", result.SkipReason)
		case result.Success:
			displaySuccessfulUpdate(result)
		default:
			fmt.Printf("   Error: %s\n", result.Error)
		}
		fmt.Println()
	}

	displayUpdateSummary(resp.Summary)
}

func displaySuccessfulUpdate(result *dto.ManagerUpdateResult) {
	fmt.Printf("   Duration: %.1fs\n", result.Duration)
	if result.Message != "" {
		fmt.Printf("   %s\n", result.Message)
	}
	if len(result.UpdatedPackages) > 0 {
		fmt.Printf("   Updated: %d packages\n", len(result.UpdatedPackages))
		for i := range result.UpdatedPackages {
			fmt.Printf("      • %s\n", result.UpdatedPackages[i].Name)
		}
	} else if result.Message == "" {
		fmt.Println("   No packages updated")
	}
	if result.SpaceFreed > 0 {
		fmt.Printf("   Space freed: %.1f MB\n", float64(result.SpaceFreed)/(1024*1024))
	}
}

// displayUpdateSummary prints the aggregate update results.
func displayUpdateSummary(summary *dto.UpdateSummary) {
	fmt.Println("📊 Summary")
	fmt.Printf("   Total Managers: %d\n", summary.TotalManagers)
	fmt.Printf("   Successful: %d\n", summary.SuccessfulManagers)
	fmt.Printf("   Failed: %d\n", summary.FailedManagers)
	if summary.SkippedManagers > 0 {
		fmt.Printf("   Skipped: %d\n", summary.SkippedManagers)
	}
	fmt.Printf("   Total Packages Updated: %d\n", summary.TotalPackagesUpdated)
	if summary.TotalBytesDownloaded > 0 {
		fmt.Printf("   Total Downloaded: %.1f MB\n", float64(summary.TotalBytesDownloaded)/(1024*1024))
	}
	if summary.TotalSpaceFreed > 0 {
		fmt.Printf("   Total Space Freed: %.1f MB\n", float64(summary.TotalSpaceFreed)/(1024*1024))
	}
	fmt.Printf("   Total Duration: %.1fs\n", summary.TotalDuration)
}

func displayUpdateJSON(resp *dto.UpdateResponse) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(resp); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error encoding JSON: %v\n", err)
	}
}

func init() {
	rootCmd.AddCommand(updateCmd)

	// Flags
	updateCmd.Flags().BoolVarP(&updateAll, "all", "a", false, "Update all package managers")
	updateCmd.Flags().BoolVar(&updateDryRun, "dry-run", false, "Preview changes without executing")
	updateCmd.Flags().StringVarP(&updateManagers, "managers", "m", "", "Comma-separated list of managers to update")
	updateCmd.Flags().StringVar(&updateStrategy, "strategy", "stable", "Update strategy (latest|stable|minor|micro|fixed); minor/micro require mise")
	updateCmd.Flags().BoolVar(&updateBump, "bump", false, "Allow mise to rewrite version requests (requires --managers mise)")
	updateCmd.Flags().StringVar(&updateMiseDir, "mise-dir", "", "Load mise configuration from this directory (default: current directory)")
	updateCmd.Flags().BoolVar(&updateMiseLocal, "mise-local", false, "Restrict mise updates to project-local configuration")
	updateCmd.Flags().StringSliceVar(&updateMiseTools, "mise-tools", nil, "Comma-separated declared mise tools to update (default: all active tools)")
	updateCmd.Flags().StringVar(&updateConfigPath, "config", "", "Update preferences YAML (default: $XDG_CONFIG_HOME/gz-pm/config.yaml)")
	updateCmd.Flags().StringVarP(&updateOutput, "output", "o", outputFormatText, "Output format (text|json)")
	updateCmd.Flags().BoolVar(&updatePipAllowConda, "pip-allow-conda", false, "Allow pip updates in conda environments")
}
