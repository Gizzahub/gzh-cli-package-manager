package cleanup

import (
	"context"
	"fmt"
	"sort"
	"strings"

	domaincleanup "github.com/gizzahub/gzh-cli-package-manager/pkg/domain/cleanup"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
)

// HeuristicVersionScanner reports packages that appear with multiple version
// strings in a list (best-effort; adapters rarely return multi-version rows).
// When only one row exists per name but AvailableVersion differs from
// CurrentVersion, the current install is marked as current and available is
// not treated as an installed old version (no false removals).
type HeuristicVersionScanner struct {
	listers map[string]PackageLister
}

// NewHeuristicVersionScanner creates a scanner keyed by manager ID string.
func NewHeuristicVersionScanner(listers map[string]PackageLister) *HeuristicVersionScanner {
	if listers == nil {
		listers = map[string]PackageLister{}
	}
	return &HeuristicVersionScanner{listers: listers}
}

// Scan finds old versions for a specific package name under a manager.
func (s *HeuristicVersionScanner) Scan(ctx context.Context, name, managerID string) ([]*domaincleanup.OldVersion, error) {
	all, err := s.ScanAll(ctx, managerID)
	if err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	var filtered []*domaincleanup.OldVersion
	for _, v := range all {
		if strings.EqualFold(v.Name, name) {
			filtered = append(filtered, v)
		}
	}
	return filtered, nil
}

// ScanAll finds packages with multiple installed version entries for a manager.
func (s *HeuristicVersionScanner) ScanAll(ctx context.Context, managerID string) ([]*domaincleanup.OldVersion, error) {
	lister, ok := s.listers[managerID]
	if !ok || lister == nil {
		return nil, fmt.Errorf("version scan: no package lister for manager %q", managerID)
	}
	packages, err := lister.ListPackages(ctx)
	if err != nil {
		return nil, fmt.Errorf("version scan %s: %w", managerID, err)
	}
	return ScanVersionsFromPackages(managerID, packages), nil
}

// ScanVersionsFromPackages groups packages by name; when the same name appears
// with more than one distinct CurrentVersion, all but the last-seen version are
// marked non-current (best-effort multi-version detection).
func ScanVersionsFromPackages(managerID string, packages []manager.Package) []*domaincleanup.OldVersion {
	// name -> ordered unique versions
	type entry struct {
		version string
		sizeMB  float64
	}
	grouped := make(map[string][]entry)
	order := make([]string, 0)

	for _, p := range packages {
		name := strings.TrimSpace(p.Name)
		if name == "" {
			continue
		}
		ver := strings.TrimSpace(p.CurrentVersion)
		if ver == "" {
			continue
		}
		if _, ok := grouped[name]; !ok {
			order = append(order, name)
		}
		// skip duplicate same version
		dup := false
		for _, e := range grouped[name] {
			if e.version == ver {
				dup = true
				break
			}
		}
		if !dup {
			grouped[name] = append(grouped[name], entry{version: ver, sizeMB: p.SizeMB})
		}
	}

	var results []*domaincleanup.OldVersion
	for _, name := range order {
		entries := grouped[name]
		if len(entries) < 2 {
			continue
		}
		// Sort versions lexicographically for stable "current" pick; last wins as current.
		sort.SliceStable(entries, func(i, j int) bool {
			return entries[i].version < entries[j].version
		})
		for i, e := range entries {
			results = append(results, &domaincleanup.OldVersion{
				Name:      name,
				Version:   e.version,
				ManagerID: managerID,
				SizeMB:    e.sizeMB,
				IsCurrent: i == len(entries)-1,
			})
		}
	}
	return results
}
