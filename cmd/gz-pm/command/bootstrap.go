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
	bootstrapConfig      string
	bootstrapInteractive bool
	bootstrapDryRun      bool
	bootstrapOutput      string
	bootstrapUseCase     input.BootstrapUseCase
)

// SetBootstrapUseCase injects the bootstrap use case dependency.
func SetBootstrapUseCase(uc input.BootstrapUseCase) {
	bootstrapUseCase = uc
}

// bootstrapCmd represents the bootstrap command.
var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Bootstrap package managers on a new system",
	Long: `Install and configure package managers from a configuration file.

This command helps set up a new development environment by installing
required package managers and their initial configuration.

Examples:
  # Bootstrap from config file
  gz-pm bootstrap --config mysetup.yaml

  # Interactive setup wizard
  gz-pm bootstrap --interactive

  # Preview what would be installed
  gz-pm bootstrap --config mysetup.yaml --dry-run`,
	Run: func(_ *cobra.Command, _ []string) {
		if bootstrapUseCase == nil {
			fmt.Println("❌ Error: Bootstrap use case not initialized")
			return
		}

		ctx := context.Background()

		// Build request
		req := &dto.BootstrapRequest{
			ConfigPath:  bootstrapConfig,
			Interactive: bootstrapInteractive,
			DryRun:      bootstrapDryRun,
		}

		// Execute bootstrap
		resp, err := bootstrapUseCase.Bootstrap(ctx, req)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			os.Exit(1)
		}

		// Display results
		switch bootstrapOutput {
		case outputFormatJSON:
			displayBootstrapJSON(resp)
		case outputFormatText:
			displayBootstrapText(resp)
		default:
			fmt.Printf("❌ Error: Unknown output format: %s\n", bootstrapOutput)
			fmt.Println("Supported formats: text, json")
		}

		// Exit with appropriate code
		if resp.Summary.FailedManagers > 0 {
			if resp.Summary.InstalledManagers > 0 {
				os.Exit(1) // Partial failure
			} else {
				os.Exit(2) // Complete failure
			}
		}
	},
}

func displayBootstrapText(resp *dto.BootstrapResponse) {
	if resp.DryRun {
		fmt.Println("🧪 Package Manager Bootstrap (DRY-RUN)")
	} else {
		fmt.Println("🚀 Package Manager Bootstrap")
	}
	fmt.Println()

	// Display individual manager results
	for _, result := range resp.Results {
		var statusIcon string
		switch {
		case result.Skipped:
			statusIcon = "⏭️ "
		case result.AlreadyInstalled:
			statusIcon = "✓"
		case result.Success:
			statusIcon = "✅"
		default:
			statusIcon = "❌"
		}

		fmt.Printf("%s %s\n", statusIcon, result.Name)

		switch {
		case result.Skipped:
			fmt.Printf("   Skipped: %s\n", result.SkipReason)
		case result.AlreadyInstalled:
			fmt.Println("   Already installed")
		case result.Success:
			fmt.Printf("   Duration: %.1fs\n", result.Duration)
			if len(result.Steps) > 0 {
				for _, step := range result.Steps {
					fmt.Printf("   • %s\n", step)
				}
			}
		default:
			fmt.Printf("   Error: %s\n", result.Error)
		}
		fmt.Println()
	}

	// Display summary
	fmt.Println("📊 Summary")
	fmt.Printf("   Total Managers: %d\n", resp.Summary.TotalManagers)
	fmt.Printf("   Installed: %d\n", resp.Summary.InstalledManagers)
	fmt.Printf("   Already Installed: %d\n", resp.Summary.AlreadyInstalledManagers)
	fmt.Printf("   Skipped: %d\n", resp.Summary.SkippedManagers)
	fmt.Printf("   Failed: %d\n", resp.Summary.FailedManagers)
	fmt.Printf("   Total Duration: %.1fs\n", resp.Summary.TotalDuration)

	if resp.DryRun {
		fmt.Println()
		fmt.Println("💡 This was a dry-run. No actual installations were performed.")
	}
}

func displayBootstrapJSON(resp *dto.BootstrapResponse) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(resp); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error encoding JSON: %v\n", err)
	}
}

func init() {
	rootCmd.AddCommand(bootstrapCmd)

	// Flags
	bootstrapCmd.Flags().StringVarP(&bootstrapConfig, "config", "c", "", "Configuration file path")
	bootstrapCmd.Flags().BoolVarP(&bootstrapInteractive, "interactive", "i", false, "Interactive setup wizard")
	bootstrapCmd.Flags().BoolVar(&bootstrapDryRun, "dry-run", false, "Preview changes without executing")
	bootstrapCmd.Flags().StringVarP(&bootstrapOutput, "output", "o", outputFormatText, "Output format (text|json)")
}
