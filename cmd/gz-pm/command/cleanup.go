package command

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/cleanup"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
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
- Orphans: List/remove heuristic candidates (remove defaults to --dry-run=true)
- Versions: List/remove multi-version rows (keep current; remove defaults dry-run)

Examples:
  gz-pm cleanup quarantine list
  gz-pm cleanup quarantine purge --retention 30 --dry-run
  gz-pm cleanup cache status
  gz-pm cleanup cache scan
  gz-pm cleanup cache clean --manager npm --dry-run
  gz-pm cleanup orphans list --manager scoop --dry-run
  gz-pm cleanup orphans remove --manager scoop
  gz-pm cleanup orphans remove --manager scoop --dry-run=false
  gz-pm cleanup versions list --manager asdf
  gz-pm cleanup versions remove --manager asdf`,
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
			return fmt.Errorf("list quarantined packages: %w", err)
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
			return fmt.Errorf("find expired quarantined packages: %w", err)
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
			return fmt.Errorf("purge expired quarantined packages: %w", err)
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
			return fmt.Errorf("list cache status: %w", err)
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
			return fmt.Errorf("scan caches: %w", err)
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
			return fmt.Errorf("clean caches: %w", err)
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

// orphansCmd is the parent for orphan subcommands.
var orphansCmd = &cobra.Command{
	Use:   "orphans",
	Short: "Find heuristic orphan package candidates",
	Long: `List packages that look like orphans using best-effort heuristics
(empty name, placeholder name, missing version). Does not consult dependency graphs.

Use --dry-run to print dry-run remove messages (no packages are removed).`,
}

// orphansListCmd lists orphan candidates from registered adapters.
var orphansListCmd = &cobra.Command{
	Use:   "list",
	Short: "List orphan package candidates",
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := context.Background()
		listers := packageListersFromAdapters()
		if len(listers) == 0 {
			return fmt.Errorf("orphans list: no package managers registered (adapters not initialized)")
		}

		detector := repo.NewHeuristicOrphanDetector(listers)

		var packages []*cleanup.OrphanPackage
		var err error
		if cleanupManagerID != "" {
			packages, err = detector.Detect(ctx, cleanupManagerID)
		} else {
			packages, err = detector.DetectAll(ctx)
		}
		if err != nil {
			return fmt.Errorf("detect orphan packages: %w", err)
		}

		if len(packages) == 0 {
			fmt.Println("📦 No orphan candidates found")
			return nil
		}

		fmt.Println("🧹 Orphan Candidates (heuristic)")
		fmt.Println()
		for _, pkg := range packages {
			fmt.Printf("  %s@%s (%s) — %s\n", pkg.Name, emptyDash(pkg.Version), pkg.ManagerID, pkg.Reason)
			if cleanupDryRun {
				fmt.Printf("    Dry-run: would remove %s@%s via %s\n", pkg.Name, emptyDash(pkg.Version), pkg.ManagerID)
			}
		}
		fmt.Printf("\nTotal: %d candidate(s)\n", len(packages))
		if cleanupDryRun {
			fmt.Println("No packages were removed (dry-run).")
		} else {
			fmt.Println("Listing only — remove is not performed by this command.")
		}
		return nil
	},
}

// versionsCmd is the parent for version cleanup subcommands.
var versionsCmd = &cobra.Command{
	Use:   "versions",
	Short: "Find packages with multiple installed versions",
	Long: `Best-effort multi-version detection from adapter ListPackages results.
Reports names that appear with more than one distinct CurrentVersion.`,
}

// versionsListCmd lists multi-version packages from registered adapters.
var versionsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List packages with multiple installed versions",
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := context.Background()
		listers := packageListersFromAdapters()
		if len(listers) == 0 {
			return fmt.Errorf("versions list: no package managers registered (adapters not initialized)")
		}

		scanner := repo.NewHeuristicVersionScanner(listers)

		var all []*cleanup.OldVersion
		if cleanupManagerID != "" {
			versions, err := scanner.ScanAll(ctx, cleanupManagerID)
			if err != nil {
				return fmt.Errorf("scan package versions for %s: %w", cleanupManagerID, err)
			}
			all = versions
		} else {
			for id := range listers {
				versions, err := scanner.ScanAll(ctx, id)
				if err != nil {
					return fmt.Errorf("scan package versions for %s: %w", id, err)
				}
				all = append(all, versions...)
			}
		}

		if len(all) == 0 {
			fmt.Println("📦 No multi-version packages found")
			return nil
		}

		fmt.Println("📚 Multi-version Packages (best-effort)")
		fmt.Println()
		for _, v := range all {
			marker := "old"
			if v.IsCurrent {
				marker = "current"
			}
			fmt.Printf("  %s@%s (%s) [%s]\n", v.Name, v.Version, v.ManagerID, marker)
			if cleanupDryRun && !v.IsCurrent {
				fmt.Printf("    Dry-run: would remove old version %s@%s via %s\n", v.Name, v.Version, v.ManagerID)
			}
		}
		fmt.Printf("\nTotal: %d version row(s)\n", len(all))
		return nil
	},
}

func packageListersFromAdapters() map[string]repo.PackageLister {
	if managerAdapters == nil {
		return nil
	}
	out := make(map[string]repo.PackageLister, len(managerAdapters))
	for id, adapter := range managerAdapters {
		if adapter == nil {
			continue
		}
		// adapterm.Adapter already has ListPackages — adapt via wrapper.
		out[string(id)] = adapterPackageLister{adapter: adapter}
	}
	return out
}

// adapterPackageLister adapts adapterm.Adapter to repo.PackageLister.
type adapterPackageLister struct {
	adapter adapterm.Adapter
}

func (a adapterPackageLister) ListPackages(ctx context.Context) ([]manager.Package, error) {
	//nolint:wrapcheck // Cleanup callers add operation context with %w; preserve the adapter error identity for errors.Is.
	return a.adapter.ListPackages(ctx)
}

func emptyDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// removeDryRun is the dry-run flag for orphans/versions remove (default true).
var removeDryRun bool

// adapterUninstaller routes uninstall to the registered manager adapter.
type adapterUninstaller struct{}

func (adapterUninstaller) Uninstall(ctx context.Context, managerID, pkgID string, dryRun bool) error {
	if managerAdapters == nil {
		return fmt.Errorf("no package managers registered (adapters not initialized)")
	}
	adapter, ok := managerAdapters[manager.ManagerID(managerID)]
	if !ok || adapter == nil {
		return fmt.Errorf("manager %q is not registered", managerID)
	}
	installer, ok := adapter.(adapterm.Installer)
	if !ok {
		return fmt.Errorf("manager %q does not support uninstall", managerID)
	}
	okDetected, err := adapter.Detect(ctx)
	if err != nil {
		return fmt.Errorf("detect %s: %w", managerID, err)
	}
	if !okDetected {
		return fmt.Errorf("%s is not available on this system", managerID)
	}
	//nolint:wrapcheck // The executor records package@version (manager) in Summary.Errors; preserve the installer error identity for errors.Is.
	return installer.Uninstall(ctx, pkgID, dryRun)
}

// orphansRemoveCmd removes orphan candidates (default --dry-run=true).
var orphansRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Uninstall orphan package candidates",
	Long: `Uninstall packages flagged as orphan candidates (same heuristics as list).

Safety: --dry-run defaults to true. Pass --dry-run=false to perform live uninstall
via managers that implement Installer (winget/scoop/chocolatey, …).

Packages with empty or placeholder names are skipped.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := context.Background()
		listers := packageListersFromAdapters()
		if len(listers) == 0 {
			return fmt.Errorf("orphans remove: no package managers registered (adapters not initialized)")
		}
		detector := repo.NewHeuristicOrphanDetector(listers)

		var packages []*cleanup.OrphanPackage
		var err error
		if cleanupManagerID != "" {
			packages, err = detector.Detect(ctx, cleanupManagerID)
		} else {
			packages, err = detector.DetectAll(ctx)
		}
		if err != nil {
			return fmt.Errorf("detect orphan packages for removal: %w", err)
		}
		if len(packages) == 0 {
			fmt.Println("📦 No orphan candidates to remove")
			return nil
		}

		ex := repo.NewAdapterCleanupExecutor(adapterUninstaller{})
		summary, err := ex.RemoveOrphans(ctx, packages, removeDryRun)
		if err != nil {
			return fmt.Errorf("remove orphan packages: %w", err)
		}
		printRemoveSummary("Orphan remove", summary, removeDryRun)
		return nil
	},
}

// versionsRemoveCmd removes non-current multi-version rows (default --dry-run=true).
var versionsRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Uninstall old package versions (keep current)",
	Long: `Uninstall non-current versions reported by the multi-version scanner.

Safety: --dry-run defaults to true. Pass --dry-run=false for live uninstall.
Current versions are never removed.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := context.Background()
		listers := packageListersFromAdapters()
		if len(listers) == 0 {
			return fmt.Errorf("versions remove: no package managers registered (adapters not initialized)")
		}
		scanner := repo.NewHeuristicVersionScanner(listers)

		var all []*cleanup.OldVersion
		if cleanupManagerID != "" {
			versions, err := scanner.ScanAll(ctx, cleanupManagerID)
			if err != nil {
				return fmt.Errorf("scan package versions for removal from %s: %w", cleanupManagerID, err)
			}
			all = versions
		} else {
			for id := range listers {
				versions, err := scanner.ScanAll(ctx, id)
				if err != nil {
					return fmt.Errorf("scan package versions for removal from %s: %w", id, err)
				}
				all = append(all, versions...)
			}
		}

		// Only non-current
		var old []*cleanup.OldVersion
		for _, v := range all {
			if v != nil && !v.IsCurrent {
				old = append(old, v)
			}
		}
		if len(old) == 0 {
			fmt.Println("📦 No old versions to remove")
			return nil
		}

		ex := repo.NewAdapterCleanupExecutor(adapterUninstaller{})
		summary, err := ex.RemoveOldVersions(ctx, old, removeDryRun)
		if err != nil {
			return fmt.Errorf("remove old package versions: %w", err)
		}
		printRemoveSummary("Version remove", summary, removeDryRun)
		return nil
	},
}

func printRemoveSummary(title string, summary *cleanup.Summary, dryRun bool) {
	mode := "LIVE"
	if dryRun {
		mode = "DRY-RUN"
	}
	fmt.Printf("🗑️  %s complete [%s]\n", title, mode)
	fmt.Printf("  Packages: %d\n", summary.PackagesRemoved)
	fmt.Printf("  Space:    %.2f MB\n", summary.SpaceFreedMB)
	fmt.Printf("  Duration: %s\n", summary.Duration)
	if dryRun {
		fmt.Println("  No packages were uninstalled (dry-run). Re-run with --dry-run=false to execute.")
	}
	if len(summary.Errors) > 0 {
		fmt.Fprintln(os.Stderr, "  Errors/skips:")
		for _, e := range summary.Errors {
			fmt.Fprintf(os.Stderr, "    - %s\n", e)
		}
	}
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

	// Orphans / versions
	cleanupCmd.AddCommand(orphansCmd)
	orphansCmd.AddCommand(orphansListCmd)
	orphansCmd.AddCommand(orphansRemoveCmd)
	cleanupCmd.AddCommand(versionsCmd)
	versionsCmd.AddCommand(versionsListCmd)
	versionsCmd.AddCommand(versionsRemoveCmd)

	// remove subcommands: dry-run defaults to true (destructive safety)
	orphansRemoveCmd.Flags().BoolVar(&removeDryRun, "dry-run", true, "Preview uninstalls without executing (default true)")
	versionsRemoveCmd.Flags().BoolVar(&removeDryRun, "dry-run", true, "Preview uninstalls without executing (default true)")

	// Global flags for cleanup
	cleanupCmd.PersistentFlags().IntVar(&cleanupRetentionDays, "retention", 30, "Retention period in days")
	cleanupCmd.PersistentFlags().BoolVar(&cleanupDryRun, "dry-run", false, "Show what would be done without making changes")
	cleanupCmd.PersistentFlags().StringVar(&cleanupManagerID, "manager", "", "Filter by package manager ID")
}
