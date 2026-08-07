package cleanup

import (
	"context"
	"fmt"
	"strings"
	"time"

	domaincleanup "github.com/gizzahub/gzh-cli-package-manager/pkg/domain/cleanup"
)

// PackageUninstaller uninstalls a package by ID for a given manager.
// Typically backed by adapterm.Installer; kept local to cleanup to avoid
// circular imports with adapter packages.
type PackageUninstaller interface {
	// Uninstall removes pkgID. When dryRun is true, must not mutate the system.
	Uninstall(ctx context.Context, managerID, pkgID string, dryRun bool) error
}

// AdapterCleanupExecutor removes orphan/old-version packages via uninstall.
//
// Safety:
//   - dryRun=true never calls live uninstall (still invokes Uninstall with dryRun=true
//     so adapters can log/preview when they support it).
//   - orphans with placeholder/empty names are skipped (cannot target uninstall).
//   - old versions keep IsCurrent=true rows.
type AdapterCleanupExecutor struct {
	uninstaller PackageUninstaller
}

// NewAdapterCleanupExecutor creates an executor.
func NewAdapterCleanupExecutor(u PackageUninstaller) *AdapterCleanupExecutor {
	return &AdapterCleanupExecutor{uninstaller: u}
}

// RemoveOrphans uninstalls orphan candidates. dryRun defaults should be enforced by CLI.
func (e *AdapterCleanupExecutor) RemoveOrphans(ctx context.Context, packages []*domaincleanup.OrphanPackage, dryRun bool) (*domaincleanup.Summary, error) {
	start := time.Now()
	summary := &domaincleanup.Summary{
		Errors: []string{},
	}
	if e.uninstaller == nil {
		return summary, fmt.Errorf("orphan remove: no uninstaller configured")
	}

	for _, pkg := range packages {
		if pkg == nil {
			continue
		}
		id := uninstallTarget(pkg.Name)
		if id == "" {
			summary.Errors = append(summary.Errors,
				fmt.Sprintf("%s: skip uninstall — package name not actionable (%q)", pkg.ManagerID, pkg.Name))
			continue
		}
		if err := e.uninstaller.Uninstall(ctx, pkg.ManagerID, id, dryRun); err != nil {
			summary.Errors = append(summary.Errors,
				fmt.Sprintf("%s@%s (%s): %v", id, emptyVer(pkg.Version), pkg.ManagerID, err))
			continue
		}
		summary.PackagesRemoved++
		summary.SpaceFreedMB += pkg.SizeMB
	}

	summary.Duration = time.Since(start)
	return summary, nil
}

// RemoveOldVersions uninstalls non-current version rows. Prefer name@version as ID when
// the uninstaller accepts it; otherwise name only (manager-dependent best effort).
func (e *AdapterCleanupExecutor) RemoveOldVersions(ctx context.Context, versions []*domaincleanup.OldVersion, dryRun bool) (*domaincleanup.Summary, error) {
	start := time.Now()
	summary := &domaincleanup.Summary{
		Errors: []string{},
	}
	if e.uninstaller == nil {
		return summary, fmt.Errorf("version remove: no uninstaller configured")
	}

	for _, v := range versions {
		if v == nil || v.IsCurrent {
			continue
		}
		name := strings.TrimSpace(v.Name)
		if name == "" {
			summary.Errors = append(summary.Errors,
				fmt.Sprintf("%s: skip — empty package name", v.ManagerID))
			continue
		}
		// Prefer name@version for managers that support multi-version IDs (asdf-style);
		// winget/scoop/choco typically use name only.
		pkgID := name
		if ver := strings.TrimSpace(v.Version); ver != "" {
			pkgID = name + "@" + ver
		}
		if err := e.uninstaller.Uninstall(ctx, v.ManagerID, pkgID, dryRun); err != nil {
			// Retry with bare name if name@version failed (many Windows managers).
			if strings.Contains(pkgID, "@") {
				if err2 := e.uninstaller.Uninstall(ctx, v.ManagerID, name, dryRun); err2 == nil {
					summary.PackagesRemoved++
					summary.SpaceFreedMB += v.SizeMB
					continue
				}
			}
			summary.Errors = append(summary.Errors,
				fmt.Sprintf("%s@%s (%s): %v", name, emptyVer(v.Version), v.ManagerID, err))
			continue
		}
		summary.PackagesRemoved++
		summary.SpaceFreedMB += v.SizeMB
	}

	summary.Duration = time.Since(start)
	return summary, nil
}

func uninstallTarget(name string) string {
	name = strings.TrimSpace(name)
	if name == "" || name == "(unnamed)" {
		return ""
	}
	lower := strings.ToLower(name)
	if lower == "-" || lower == "unknown" {
		return ""
	}
	return name
}

func emptyVer(v string) string {
	if strings.TrimSpace(v) == "" {
		return "-"
	}
	return v
}
