package command

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/dto"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/input"
	"github.com/spf13/cobra"
)

var (
	statusVerbose bool
	statusOutput  string
	statusUseCase input.StatusUseCase
)

const statusPackageDisplayLimit = 10

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

		// Handle output format
		switch statusOutput {
		case outputFormatJSON:
			displayJSON(resp)
		case outputFormatText:
			displayText(resp)
		default:
			fmt.Printf("❌ Error: Unknown output format: %s\n", statusOutput)
			fmt.Println("Supported formats: text, json")
		}
	},
}

func displayText(resp *dto.StatusResponse) {
	fmt.Println("📋 Package Manager Status")
	fmt.Println()

	for _, mgr := range resp.Managers {
		displayManagerText(mgr, statusVerbose)
		fmt.Println()
	}

	displaySummaryText(resp.Summary)
}

func displayManagerText(mgr *dto.ManagerStatus, verbose bool) {
	if !mgr.Installed {
		fmt.Printf("⛔ %s (%s)\n", mgr.Name, mgr.Type)
		return
	}

	fmt.Printf("✅ %s (%s)\n", mgr.Name, mgr.Type)
	fmt.Printf("   Version: %s\n", mgr.Version)
	fmt.Printf("   Status: %s\n", mgr.Status)
	fmt.Printf("   Packages: %d (Updates: %d)\n", mgr.PackageCount, mgr.UpdatableCount)
	displayPackagesText(mgr.Packages, verbose)
}

func displayPackagesText(packages []dto.PackageInfo, verbose bool) {
	if !verbose || len(packages) == 0 {
		return
	}

	fmt.Println("   ")
	limit := min(len(packages), statusPackageDisplayLimit)
	for i := range limit {
		displayPackageText(&packages[i])
	}
	if len(packages) > statusPackageDisplayLimit {
		fmt.Printf("   ... and %d more packages\n", len(packages)-statusPackageDisplayLimit)
	}
}

func displayPackageText(pkg *dto.PackageInfo) {
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

func displaySummaryText(summary *dto.StatusSummary) {
	fmt.Println("📊 Summary")
	fmt.Printf("   Total Managers: %d\n", summary.TotalManagers)
	fmt.Printf("   Installed: %d\n", summary.InstalledManagers)
	fmt.Printf("   Healthy: %d\n", summary.HealthyManagers)
	fmt.Printf("   Total Packages: %d\n", summary.TotalPackages)
	fmt.Printf("   Updates Available: %d\n", summary.UpdatablePackages)
}

func displayJSON(resp *dto.StatusResponse) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(resp); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error encoding JSON: %v\n", err)
	}
}

func init() {
	rootCmd.AddCommand(statusCmd)

	// Flags
	statusCmd.Flags().BoolVarP(&statusVerbose, "verbose", "v", false, "Show detailed status information")
	statusCmd.Flags().StringVarP(&statusOutput, "output", "o", outputFormatText, "Output format (text|json)")
}
