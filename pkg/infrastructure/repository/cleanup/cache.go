package cleanup

import (
	"context"
	"sync"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/cleanup"
)

// CacheRepository is an in-memory implementation of cleanup.CacheRepository.
type CacheRepository struct {
	mu     sync.RWMutex
	caches map[string]*cleanup.CacheInfo
}

// NewCacheRepository creates a new in-memory cache repository.
func NewCacheRepository() *CacheRepository {
	return &CacheRepository{
		caches: make(map[string]*cleanup.CacheInfo),
	}
}

// GetInfo returns cache statistics for a package manager.
func (r *CacheRepository) GetInfo(_ context.Context, managerID string) (*cleanup.CacheInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	info, exists := r.caches[managerID]
	if !exists {
		// Return empty info if not tracked yet
		return &cleanup.CacheInfo{
			ManagerID: managerID,
		}, nil
	}

	return info, nil
}

// ListAll returns cache info for all managers.
func (r *CacheRepository) ListAll(_ context.Context) ([]*cleanup.CacheInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*cleanup.CacheInfo, 0, len(r.caches))
	for _, info := range r.caches {
		result = append(result, info)
	}

	return result, nil
}

// UpdateInfo updates cache statistics for a manager.
func (r *CacheRepository) UpdateInfo(_ context.Context, info *cleanup.CacheInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.caches[info.ManagerID] = info
	return nil
}

// Ensure CacheRepository implements the interface.
var _ cleanup.CacheRepository = (*CacheRepository)(nil)
