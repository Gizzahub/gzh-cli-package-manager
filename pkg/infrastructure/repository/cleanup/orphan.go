// Package cleanup provides infrastructure implementations for cleanup operations.
package cleanup

import (
	"context"
	"fmt"
	"strings"

	domaincleanup "github.com/gizzahub/gzh-cli-package-manager/pkg/domain/cleanup"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
)

// PackageLister lists packages for a manager (typically an adapter).
type PackageLister interface {
	ListPackages(ctx context.Context) ([]manager.Package, error)
}

// HeuristicOrphanDetector finds orphan candidates from package lists using
// simple heuristics (no dependency graph). Suitable for best-effort CLI listing.
//
// Heuristics (any match → orphan candidate):
//   - empty / whitespace-only package name
//   - name is "-" or "unknown" (case-insensitive)
//   - empty version when name is present (incomplete install metadata)
type HeuristicOrphanDetector struct {
	listers map[string]PackageLister
}

// NewHeuristicOrphanDetector creates a detector keyed by manager ID string.
func NewHeuristicOrphanDetector(listers map[string]PackageLister) *HeuristicOrphanDetector {
	if listers == nil {
		listers = map[string]PackageLister{}
	}
	return &HeuristicOrphanDetector{listers: listers}
}

// Detect finds orphan candidates for a specific manager.
func (d *HeuristicOrphanDetector) Detect(ctx context.Context, managerID string) ([]*domaincleanup.OrphanPackage, error) {
	lister, ok := d.listers[managerID]
	if !ok || lister == nil {
		return nil, fmt.Errorf("orphan detect: no package lister for manager %q", managerID)
	}
	packages, err := lister.ListPackages(ctx)
	if err != nil {
		return nil, fmt.Errorf("orphan detect %s: %w", managerID, err)
	}
	return DetectOrphansFromPackages(managerID, packages), nil
}

// DetectAll finds orphan candidates across all registered managers.
func (d *HeuristicOrphanDetector) DetectAll(ctx context.Context) ([]*domaincleanup.OrphanPackage, error) {
	var all []*domaincleanup.OrphanPackage
	for id, lister := range d.listers {
		if lister == nil {
			continue
		}
		packages, err := lister.ListPackages(ctx)
		if err != nil {
			return all, fmt.Errorf("orphan detect %s: %w", id, err)
		}
		all = append(all, DetectOrphansFromPackages(id, packages)...)
	}
	return all, nil
}

// DetectOrphansFromPackages is a pure helper: mark packages matching heuristics.
func DetectOrphansFromPackages(managerID string, packages []manager.Package) []*domaincleanup.OrphanPackage {
	var orphans []*domaincleanup.OrphanPackage
	for i := range packages {
		p := &packages[i]
		reason := orphanReason(p)
		if reason == "" {
			continue
		}
		orphans = append(orphans, &domaincleanup.OrphanPackage{
			Name:           displayName(p.Name),
			Version:        p.CurrentVersion,
			ManagerID:      managerID,
			SizeMB:         p.SizeMB,
			DependentCount: 0,
			Reason:         reason,
		})
	}
	return orphans
}

func orphanReason(p *manager.Package) string {
	name := strings.TrimSpace(p.Name)
	if name == "" {
		return "empty package name"
	}
	lower := strings.ToLower(name)
	if lower == "-" || lower == "unknown" {
		return "placeholder package name"
	}
	if strings.TrimSpace(p.CurrentVersion) == "" {
		return "missing version metadata"
	}
	return ""
}

func displayName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "(unnamed)"
	}
	return name
}
