package command

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/dto"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/input"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	"github.com/spf13/cobra"
)

var (
	updateAll           bool
	updateDryRun        bool
	updateManagers      string
	updateStrategy      string
	updateOutput        string
	updatePipAllowConda bool
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
	Run: func(_ *cobra.Command, _ []string) {
		if updateUseCase == nil {
			fmt.Println("❌ Error: Update use case not initialized")
			return
		}

		ctx := context.Background()

		// Parse manager IDs if specified
		var managerIDs []manager.ManagerID
		if updateManagers != "" {
			parts := strings.Split(updateManagers, ",")
			managerIDs = make([]manager.ManagerID, 0, len(parts))
			for _, part := range parts {
				managerIDs = append(managerIDs, manager.ManagerID(strings.TrimSpace(part)))
			}
		}

		// Convert strategy string to DTO type
		strategy := dto.StrategyStable // default
		switch updateStrategy {
		case "latest":
			strategy = dto.StrategyLatest
		case "stable":
			strategy = dto.StrategyStable
		case "minor":
			strategy = dto.StrategyMinor
		case "fixed":
			strategy = dto.StrategyFixed
		default:
			fmt.Printf("⚠️  Warning: Unknown strategy '%s', using 'stable'\n", updateStrategy)
		}

		// Build request
		req := &dto.UpdateRequest{
			All:           updateAll,
			ManagerIDs:    managerIDs,
			DryRun:        updateDryRun,
			Strategy:      strategy,
			PipAllowConda: updatePipAllowConda,
		}

		// Execute update
		resp, err := updateUseCase.Update(ctx, req)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			os.Exit(1)
		}

		// Display results
		switch updateOutput {
		case "json":
			displayUpdateJSON(resp)
		case "text":
			displayUpdateText(resp)
		default:
			fmt.Printf("❌ Error: Unknown output format: %s\n", updateOutput)
			fmt.Println("Supported formats: text, json")
		}

		// Exit with appropriate code
		if resp.Summary.FailedManagers > 0 {
			if resp.Summary.SuccessfulManagers > 0 {
				os.Exit(1) // Partial failure
			} else {
				os.Exit(2) // Complete failure
			}
		}
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
		if result.Skipped {
			fmt.Printf("   Skipped: %s\n", result.SkipReason)
		} else if result.Success {
			fmt.Printf("   Duration: %.1fs\n", result.Duration)
			if len(result.UpdatedPackages) > 0 {
				fmt.Printf("   Updated: %d packages\n", len(result.UpdatedPackages))
				for _, pkg := range result.UpdatedPackages {
					fmt.Printf("      • %s\n", pkg.Name)
				}
			} else {
				fmt.Println("   No packages updated")
			}
			if result.SpaceFreed > 0 {
				fmt.Printf("   Space freed: %.1f MB\n", float64(result.SpaceFreed)/(1024*1024))
			}
		} else {
			fmt.Printf("   Error: %s\n", result.Error)
		}
		fmt.Println()
	}

	// Display summary
	fmt.Println("📊 Summary")
	fmt.Printf("   Total Managers: %d\n", resp.Summary.TotalManagers)
	fmt.Printf("   Successful: %d\n", resp.Summary.SuccessfulManagers)
	fmt.Printf("   Failed: %d\n", resp.Summary.FailedManagers)
	if resp.Summary.SkippedManagers > 0 {
		fmt.Printf("   Skipped: %d\n", resp.Summary.SkippedManagers)
	}
	fmt.Printf("   Total Packages Updated: %d\n", resp.Summary.TotalPackagesUpdated)
	if resp.Summary.TotalBytesDownloaded > 0 {
		fmt.Printf("   Total Downloaded: %.1f MB\n", float64(resp.Summary.TotalBytesDownloaded)/(1024*1024))
	}
	if resp.Summary.TotalSpaceFreed > 0 {
		fmt.Printf("   Total Space Freed: %.1f MB\n", float64(resp.Summary.TotalSpaceFreed)/(1024*1024))
	}
	fmt.Printf("   Total Duration: %.1fs\n", resp.Summary.TotalDuration)
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
	updateCmd.Flags().StringVar(&updateStrategy, "strategy", "stable", "Update strategy (latest|stable|minor|fixed)")
	updateCmd.Flags().StringVarP(&updateOutput, "output", "o", "text", "Output format (text|json|simple)")
	updateCmd.Flags().BoolVar(&updatePipAllowConda, "pip-allow-conda", false, "Allow pip updates in conda environments")
}
