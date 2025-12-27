package command

import (
	"context"
	"fmt"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/cleanup"
	repo "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/repository/cleanup"
	"github.com/spf13/cobra"
)

var (
	cleanupRetentionDays int
	cleanupDryRun        bool
	cleanupManagerID     string
)

// cleanupCmd represents the cleanup command group.
var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Manage package cleanup operations",
	Long: `Cleanup operations for package managers including:
- Quarantine: Safely remove packages with recovery option
- Cache: Clear package manager caches
- Orphans: Find and remove unused packages

Examples:
  gz-pm cleanup quarantine list
  gz-pm cleanup cache status
  gz-pm cleanup cache clear --manager homebrew`,
}

// quarantineCmd is the parent for quarantine subcommands.
var quarantineCmd = &cobra.Command{
	Use:   "quarantine",
	Short: "Manage quarantined packages",
	Long: `Quarantine operations allow safe package removal with recovery option.
Quarantined packages are moved to a holding area and can be restored if needed.`,
}

// quarantineListCmd lists quarantined packages.
var quarantineListCmd = &cobra.Command{
	Use:   "list",
	Short: "List quarantined packages",
	Run: func(_ *cobra.Command, _ []string) {
		ctx := context.Background()
		quarantineRepo := repo.NewQuarantineRepository()

		var packages []*cleanup.QuarantinedPackage
		var err error

		if cleanupManagerID != "" {
			packages, err = quarantineRepo.ListByManager(ctx, cleanupManagerID)
		} else {
			packages, err = quarantineRepo.List(ctx)
		}

		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			return
		}

		if len(packages) == 0 {
			fmt.Println("📦 No quarantined packages found")
			return
		}

		fmt.Println("🔒 Quarantined Packages")
		fmt.Println()
		for _, pkg := range packages {
			fmt.Printf("  %s@%s (%s)\n", pkg.Name, pkg.Version, pkg.ManagerID)
			fmt.Printf("    Quarantined: %s (%d days ago)\n",
				pkg.QuarantinedAt.Format("2006-01-02"), pkg.DaysSinceQuarantine())
			fmt.Printf("    Reason: %s\n", pkg.Reason)
			fmt.Printf("    Size: %.2f MB\n", pkg.SizeMB)
			fmt.Println()
		}
	},
}

// quarantineExpiredCmd lists expired quarantined packages.
var quarantineExpiredCmd = &cobra.Command{
	Use:   "expired",
	Short: "List expired quarantined packages",
	Run: func(_ *cobra.Command, _ []string) {
		ctx := context.Background()
		quarantineRepo := repo.NewQuarantineRepository()

		packages, err := quarantineRepo.FindExpired(ctx, cleanupRetentionDays)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			return
		}

		if len(packages) == 0 {
			fmt.Printf("📦 No packages expired (retention: %d days)\n", cleanupRetentionDays)
			return
		}

		fmt.Printf("⏰ Expired Packages (older than %d days)\n", cleanupRetentionDays)
		fmt.Println()

		var totalSize float64
		for _, pkg := range packages {
			fmt.Printf("  %s@%s (%s) - %d days ago, %.2f MB\n",
				pkg.Name, pkg.Version, pkg.ManagerID, pkg.DaysSinceQuarantine(), pkg.SizeMB)
			totalSize += pkg.SizeMB
		}
		fmt.Println()
		fmt.Printf("Total: %d packages, %.2f MB\n", len(packages), totalSize)
	},
}

// cacheCmd is the parent for cache subcommands.
var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage package manager caches",
	Long: `Cache operations for clearing and managing package manager caches.
This helps free up disk space from downloaded packages.`,
}

// cacheStatusCmd shows cache status.
var cacheStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show cache status for all managers",
	Run: func(_ *cobra.Command, _ []string) {
		ctx := context.Background()
		cacheRepo := repo.NewCacheRepository()

		caches, err := cacheRepo.ListAll(ctx)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			return
		}

		if len(caches) == 0 {
			fmt.Println("📁 No cache information available")
			fmt.Println("Run 'gz-pm cleanup cache scan' to scan caches")
			return
		}

		fmt.Println("📁 Cache Status")
		fmt.Println()

		var totalSize float64
		for _, cache := range caches {
			fmt.Printf("  %s\n", cache.ManagerID)
			fmt.Printf("    Path: %s\n", cache.CachePath)
			fmt.Printf("    Size: %.2f MB (%d entries)\n", cache.TotalSizeMB, cache.EntryCount)
			if cache.AgeInDays() > 0 {
				fmt.Printf("    Oldest: %d days ago\n", cache.AgeInDays())
			}
			totalSize += cache.TotalSizeMB
			fmt.Println()
		}
		fmt.Printf("Total cache: %.2f MB\n", totalSize)
	},
}

func init() {
	rootCmd.AddCommand(cleanupCmd)

	// Quarantine subcommands
	cleanupCmd.AddCommand(quarantineCmd)
	quarantineCmd.AddCommand(quarantineListCmd)
	quarantineCmd.AddCommand(quarantineExpiredCmd)

	// Cache subcommands
	cleanupCmd.AddCommand(cacheCmd)
	cacheCmd.AddCommand(cacheStatusCmd)

	// Global flags for cleanup
	cleanupCmd.PersistentFlags().IntVar(&cleanupRetentionDays, "retention", 30, "Retention period in days")
	cleanupCmd.PersistentFlags().BoolVar(&cleanupDryRun, "dry-run", false, "Show what would be done without making changes")
	cleanupCmd.PersistentFlags().StringVar(&cleanupManagerID, "manager", "", "Filter by package manager ID")
}
