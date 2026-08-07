package cleanup

import (
	"context"
	"fmt"
	"time"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/cleanup"
)

// QuarantinePurger permanently removes expired quarantined package records.
type QuarantinePurger struct {
	repo cleanup.QuarantineRepository
}

// NewQuarantinePurger creates a purger over the given repository.
func NewQuarantinePurger(repo cleanup.QuarantineRepository) *QuarantinePurger {
	if repo == nil {
		repo = NewQuarantineRepository()
	}
	return &QuarantinePurger{repo: repo}
}

// PurgeExpired finds packages older than retentionDays and deletes them.
// When dryRun is true, packages are listed in the summary but not deleted.
func (p *QuarantinePurger) PurgeExpired(ctx context.Context, retentionDays int, dryRun bool) (*cleanup.Summary, error) {
	start := time.Now()
	expired, err := p.repo.FindExpired(ctx, retentionDays)
	if err != nil {
		return nil, err
	}

	summary := &cleanup.Summary{
		Errors: make([]string, 0),
	}

	for _, pkg := range expired {
		summary.PackagesRemoved++
		summary.SpaceFreedMB += pkg.SizeMB
		if dryRun {
			continue
		}
		if err := p.repo.Delete(ctx, pkg.Name, pkg.Version, pkg.ManagerID); err != nil {
			summary.Errors = append(summary.Errors,
				fmt.Sprintf("%s@%s (%s): %v", pkg.Name, pkg.Version, pkg.ManagerID, err))
			continue
		}
		// Best-effort status update if re-saved is desired — Delete removes the record.
	}

	summary.Duration = time.Since(start)
	return summary, nil
}
