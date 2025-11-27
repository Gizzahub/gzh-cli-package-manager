package command

import (
	"context"
	"fmt"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/dto"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/input"
	"github.com/spf13/cobra"
)

var (
	statusVerbose bool
	statusOutput  string
	statusUseCase input.StatusUseCase
)

// SetStatusUseCase injects the status use case dependency.
func SetStatusUseCase(uc input.StatusUseCase) {
	statusUseCase = uc
}

// statusCmd represents the status command.
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Display package manager status",
	Long: `Display the status of all package managers on the system.

Shows which managers are installed, their versions, and package counts.

Use --verbose (-v) to display:
  - First 10 packages from each installed manager
  - Update indicators (⬆️) for packages with available updates
  - Version transitions showing current → available versions

Example:
  gz-pm status           # Show summary only
  gz-pm status --verbose # Show packages and details
  gz-pm status -v        # Short form`,
	Run: func(_ *cobra.Command, _ []string) {
		if statusUseCase == nil {
			fmt.Println("❌ Error: Status use case not initialized")
			return
		}

		ctx := context.Background()
		req := &dto.StatusRequest{
			Verbose: statusVerbose,
			Refresh: false,
		}

		resp, err := statusUseCase.GetStatus(ctx, req)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			return
		}

		// Display results
		fmt.Println("📋 Package Manager Status")
		fmt.Println()

		for _, mgr := range resp.Managers {
			statusIcon := "⛔"
			if mgr.Installed {
				statusIcon = "✅"
			}

			fmt.Printf("%s %s (%s)\n", statusIcon, mgr.Name, mgr.Type)
			if mgr.Installed {
				fmt.Printf("   Version: %s\n", mgr.Version)
				fmt.Printf("   Status: %s\n", mgr.Status)
				fmt.Printf("   Packages: %d (Updates: %d)\n", mgr.PackageCount, mgr.UpdatableCount)

				// Show package details in verbose mode
				if statusVerbose && len(mgr.Packages) > 0 {
					fmt.Println("   ")
					// Show up to 10 packages
					limit := len(mgr.Packages)
					if limit > 10 {
						limit = 10
					}

					for i := 0; i < limit; i++ {
						pkg := mgr.Packages[i]
						updateIcon := "  "
						if pkg.UpdateType != "" && pkg.UpdateType != "none" {
							updateIcon = "⬆️ "
						}

						fmt.Printf("   %s %s %s", updateIcon, pkg.Name, pkg.CurrentVersion)
						if pkg.AvailableVersion != "" {
							fmt.Printf(" → %s", pkg.AvailableVersion)
						}
						fmt.Println()
					}

					if len(mgr.Packages) > 10 {
						fmt.Printf("   ... and %d more packages\n", len(mgr.Packages)-10)
					}
				}
			}
			fmt.Println()
		}

		// Display summary
		fmt.Println("📊 Summary")
		fmt.Printf("   Total Managers: %d\n", resp.Summary.TotalManagers)
		fmt.Printf("   Installed: %d\n", resp.Summary.InstalledManagers)
		fmt.Printf("   Healthy: %d\n", resp.Summary.HealthyManagers)
		fmt.Printf("   Total Packages: %d\n", resp.Summary.TotalPackages)
		fmt.Printf("   Updates Available: %d\n", resp.Summary.UpdatablePackages)
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)

	// Flags
	statusCmd.Flags().BoolVarP(&statusVerbose, "verbose", "v", false, "Show detailed status information")
	statusCmd.Flags().StringVarP(&statusOutput, "output", "o", "text", "Output format (text|json)")
}
