// Package cleanup provides infrastructure implementations for cleanup operations.
package cleanup

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/cleanup"
)

// quarantineKey generates a unique key for a quarantined package.
func quarantineKey(name, version, managerID string) string {
	return fmt.Sprintf("%s:%s:%s", managerID, name, version)
}

// QuarantineRepository is an in-memory implementation of cleanup.QuarantineRepository.
type QuarantineRepository struct {
	mu       sync.RWMutex
	packages map[string]*cleanup.QuarantinedPackage
}

// NewQuarantineRepository creates a new in-memory quarantine repository.
func NewQuarantineRepository() *QuarantineRepository {
	return &QuarantineRepository{
		packages: make(map[string]*cleanup.QuarantinedPackage),
	}
}

// List returns all quarantined packages.
func (r *QuarantineRepository) List(_ context.Context) ([]*cleanup.QuarantinedPackage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*cleanup.QuarantinedPackage, 0, len(r.packages))
	for _, pkg := range r.packages {
		if pkg.Status == cleanup.StatusQuarantined {
			result = append(result, pkg)
		}
	}

	return result, nil
}

// ListByManager returns quarantined packages for a specific manager.
func (r *QuarantineRepository) ListByManager(_ context.Context, managerID string) ([]*cleanup.QuarantinedPackage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*cleanup.QuarantinedPackage, 0)
	for _, pkg := range r.packages {
		if pkg.ManagerID == managerID && pkg.Status == cleanup.StatusQuarantined {
			result = append(result, pkg)
		}
	}

	return result, nil
}

// Get returns a specific quarantined package.
func (r *QuarantineRepository) Get(_ context.Context, name, version, managerID string) (*cleanup.QuarantinedPackage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := quarantineKey(name, version, managerID)
	pkg, exists := r.packages[key]
	if !exists {
		return nil, cleanup.ErrPackageNotFound
	}

	return pkg, nil
}

// Save persists a quarantined package record.
func (r *QuarantineRepository) Save(_ context.Context, pkg *cleanup.QuarantinedPackage) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := quarantineKey(pkg.Name, pkg.Version, pkg.ManagerID)
	r.packages[key] = pkg
	return nil
}

// Delete removes a quarantined package record.
func (r *QuarantineRepository) Delete(_ context.Context, name, version, managerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := quarantineKey(name, version, managerID)
	delete(r.packages, key)
	return nil
}

// FindExpired returns packages that have exceeded the retention period.
func (r *QuarantineRepository) FindExpired(_ context.Context, retentionDays int) ([]*cleanup.QuarantinedPackage, error) {
	if retentionDays <= 0 {
		return nil, cleanup.ErrInvalidRetentionDays
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	expiry := time.Now().AddDate(0, 0, -retentionDays)
	result := make([]*cleanup.QuarantinedPackage, 0)

	for _, pkg := range r.packages {
		if pkg.Status == cleanup.StatusQuarantined && pkg.QuarantinedAt.Before(expiry) {
			result = append(result, pkg)
		}
	}

	return result, nil
}

// Ensure QuarantineRepository implements the interface.
var _ cleanup.QuarantineRepository = (*QuarantineRepository)(nil)
