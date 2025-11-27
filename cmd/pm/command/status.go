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
Use --verbose for detailed information including health checks.`,
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
