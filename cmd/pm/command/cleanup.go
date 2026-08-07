package command

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/cleanup"
	repo "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/repository/cleanup"
	"github.com/spf13/cobra"
)

var (
	cleanupRetentionDays int
	cleanupDryRun        bool
	cleanupManagerID     string

	// Process-local shared repositories so scan → status / quarantine list → purge
	// work within the same process (and unit tests).
	cleanupStateMu   sync.Mutex
	sharedCacheRepo  cleanup.CacheRepository
	sharedQuarantine cleanup.QuarantineRepository
	sharedScanner    *repo.CacheScanner
)

func getCacheRepo() cleanup.CacheRepository {
	cleanupStateMu.Lock()
	defer cleanupStateMu.Unlock()
	if sharedCacheRepo == nil {
		sharedCacheRepo = repo.NewCacheRepository()
	}
	return sharedCacheRepo
}

func getQuarantineRepo() cleanup.QuarantineRepository {
	cleanupStateMu.Lock()
	defer cleanupStateMu.Unlock()
	if sharedQuarantine == nil {
		sharedQuarantine = repo.NewQuarantineRepository()
	}
	return sharedQuarantine
}

func getCacheScanner() *repo.CacheScanner {
	cleanupStateMu.Lock()
	defer cleanupStateMu.Unlock()
	if sharedScanner == nil {
		if sharedCacheRepo == nil {
			sharedCacheRepo = repo.NewCacheRepository()
		}
		sharedScanner = repo.NewCacheScanner(
			repo.WithCacheRepository(sharedCacheRepo),
		)
	}
	return sharedScanner
}

// SetCleanupDeps injects repositories/scanner for tests. Pass nil fields to reset.
func SetCleanupDeps(cache cleanup.CacheRepository, quarantine cleanup.QuarantineRepository, scanner *repo.CacheScanner) {
	cleanupStateMu.Lock()
	defer cleanupStateMu.Unlock()
	sharedCacheRepo = cache
	sharedQuarantine = quarantine
	sharedScanner = scanner
}

// cleanupCmd represents the cleanup command group.
var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Manage package cleanup operations",
	Long: `Cleanup operations for package managers including:
- Quarantine: Safely remove packages with recovery option
- Cache: Scan and clear package manager caches
- Orphans: Find and remove unused packages (planned)

Examples:
  gz-pm cleanup quarantine list
  gz-pm cleanup quarantine purge --retention 30 --dry-run
  gz-pm cleanup cache status
  gz-pm cleanup cache scan
  gz-pm cleanup cache clean --manager npm --dry-run`,
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
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := context.Background()
		quarantineRepo := getQuarantineRepo()

		var packages []*cleanup.QuarantinedPackage
		var err error

		if cleanupManagerID != "" {
			packages, err = quarantineRepo.ListByManager(ctx, cleanupManagerID)
		} else {
			packages, err = quarantineRepo.List(ctx)
		}

		if err != nil {
			return err
		}

		if len(packages) == 0 {
			fmt.Println("📦 No quarantined packages found")
			return nil
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
		return nil
	},
}

// quarantineExpiredCmd lists expired quarantined packages.
var quarantineExpiredCmd = &cobra.Command{
	Use:   "expired",
	Short: "List expired quarantined packages",
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := context.Background()
		quarantineRepo := getQuarantineRepo()

		packages, err := quarantineRepo.FindExpired(ctx, cleanupRetentionDays)
		if err != nil {
			return err
		}

		if len(packages) == 0 {
			fmt.Printf("📦 No packages expired (retention: %d days)\n", cleanupRetentionDays)
			return nil
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
		return nil
	},
}

// quarantinePurgeCmd permanently removes expired quarantined packages.
var quarantinePurgeCmd = &cobra.Command{
	Use:   "purge",
	Short: "Permanently remove expired quarantined packages",
	Long: `Delete quarantine records older than --retention days.

Use --dry-run to preview what would be removed without deleting.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := context.Background()
		purger := repo.NewQuarantinePurger(getQuarantineRepo())

		summary, err := purger.PurgeExpired(ctx, cleanupRetentionDays, cleanupDryRun)
		if err != nil {
			return err
		}

		if cleanupDryRun {
			fmt.Println("🔍 Dry-run: no packages were deleted")
		} else {
			fmt.Println("🗑️  Quarantine purge complete")
		}
		fmt.Printf("  Packages: %d\n", summary.PackagesRemoved)
		fmt.Printf("  Space:    %.2f MB\n", summary.SpaceFreedMB)
		fmt.Printf("  Duration: %s\n", summary.Duration)
		if len(summary.Errors) > 0 {
			fmt.Println("  Errors:")
			for _, e := range summary.Errors {
				fmt.Printf("    - %s\n", e)
			}
		}
		return nil
	},
}

// cacheCmd is the parent for cache subcommands.
var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage package manager caches",
	Long: `Cache operations for scanning and clearing package manager caches.
This helps free up disk space from downloaded packages.

Known managers (path resolution): homebrew, npm, pip, cargo, yarn, go, pnpm.`,
}

// cacheStatusCmd shows cache status.
var cacheStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show cache status for all managers",
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := context.Background()
		cacheRepo := getCacheRepo()

		caches, err := cacheRepo.ListAll(ctx)
		if err != nil {
			return err
		}

		if len(caches) == 0 {
			fmt.Println("📁 No cache information available")
			fmt.Println("Run 'gz-pm cleanup cache scan' to scan caches")
			return nil
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
		return nil
	},
}

// cacheScanCmd scans known cache paths and records statistics.
var cacheScanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan known package manager cache directories",
	Long: `Walk well-known cache paths under $HOME (and npm_config_cache when set),
measure size/entry counts, and store results for 'cache status'.

Use --manager to scan a single package manager.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := context.Background()
		scanner := getCacheScanner()

		results, err := scanner.Scan(ctx, cleanupManagerID)
		if err != nil {
			return err
		}

		if len(results) == 0 {
			fmt.Println("📁 No cache directories found on this system")
			if cleanupManagerID != "" {
				fmt.Printf("(manager filter: %s)\n", cleanupManagerID)
			}
			return nil
		}

		fmt.Println("📁 Cache Scan Results")
		fmt.Println()
		var total float64
		for _, c := range results {
			fmt.Printf("  %s\n", c.ManagerID)
			fmt.Printf("    Path: %s\n", c.CachePath)
			fmt.Printf("    Size: %.2f MB (%d entries)\n", c.TotalSizeMB, c.EntryCount)
			total += c.TotalSizeMB
			fmt.Println()
		}
		fmt.Printf("Total: %.2f MB across %d manager(s)\n", total, len(results))
		return nil
	},
}

// cacheCleanCmd clears scanned cache directories (supports --dry-run).
var cacheCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clear package manager caches",
	Long: `Remove contents of known cache directories.

Always prefer --dry-run first. Use --manager to limit scope.

Examples:
  gz-pm cleanup cache clean --dry-run
  gz-pm cleanup cache clean --manager npm --dry-run
  gz-pm cleanup cache clean --manager npm`,
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := context.Background()
		scanner := getCacheScanner()

		summary, err := scanner.Clean(ctx, cleanupManagerID, cleanupDryRun)
		if err != nil {
			return err
		}

		if cleanupDryRun {
			fmt.Println("🔍 Dry-run: no files were deleted")
		} else {
			fmt.Println("🧹 Cache clean complete")
		}
		fmt.Printf("  Entries:  %d\n", summary.PackagesRemoved)
		fmt.Printf("  Space:    %.2f MB\n", summary.SpaceFreedMB)
		fmt.Printf("  Duration: %s\n", summary.Duration)
		if len(summary.Errors) > 0 {
			fmt.Fprintln(os.Stderr, "  Errors:")
			for _, e := range summary.Errors {
				fmt.Fprintf(os.Stderr, "    - %s\n", e)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(cleanupCmd)

	// Quarantine subcommands
	cleanupCmd.AddCommand(quarantineCmd)
	quarantineCmd.AddCommand(quarantineListCmd)
	quarantineCmd.AddCommand(quarantineExpiredCmd)
	quarantineCmd.AddCommand(quarantinePurgeCmd)

	// Cache subcommands
	cleanupCmd.AddCommand(cacheCmd)
	cacheCmd.AddCommand(cacheStatusCmd)
	cacheCmd.AddCommand(cacheScanCmd)
	cacheCmd.AddCommand(cacheCleanCmd)

	// Global flags for cleanup
	cleanupCmd.PersistentFlags().IntVar(&cleanupRetentionDays, "retention", 30, "Retention period in days")
	cleanupCmd.PersistentFlags().BoolVar(&cleanupDryRun, "dry-run", false, "Show what would be done without making changes")
	cleanupCmd.PersistentFlags().StringVar(&cleanupManagerID, "manager", "", "Filter by package manager ID")
}
