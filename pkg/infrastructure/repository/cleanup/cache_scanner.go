package cleanup

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/cleanup"
)

// CacheScanner scans known package-manager cache directories.
type CacheScanner struct {
	fs        FileSystem
	home      string
	goos      string
	envLookup func(string) string
	known     []KnownCachePath
	repo      cleanup.CacheRepository
}

// CacheScannerOption configures a CacheScanner.
type CacheScannerOption func(*CacheScanner)

// WithFileSystem injects a FileSystem implementation.
func WithFileSystem(fsys FileSystem) CacheScannerOption {
	return func(s *CacheScanner) { s.fs = fsys }
}

// WithHomeDir sets the home directory used to resolve relative cache paths.
func WithHomeDir(home string) CacheScannerOption {
	return func(s *CacheScanner) { s.home = home }
}

// WithGOOS overrides runtime.GOOS for path selection.
func WithGOOS(goos string) CacheScannerOption {
	return func(s *CacheScanner) { s.goos = goos }
}

// WithEnvLookup injects environment variable lookup.
func WithEnvLookup(fn func(string) string) CacheScannerOption {
	return func(s *CacheScanner) { s.envLookup = fn }
}

// WithKnownPaths overrides the default known cache path table.
func WithKnownPaths(paths []KnownCachePath) CacheScannerOption {
	return func(s *CacheScanner) { s.known = paths }
}

// WithCacheRepository sets the repository used to persist scan results.
func WithCacheRepository(repo cleanup.CacheRepository) CacheScannerOption {
	return func(s *CacheScanner) { s.repo = repo }
}

// NewCacheScanner creates a CacheScanner with production defaults.
func NewCacheScanner(opts ...CacheScannerOption) *CacheScanner {
	s := &CacheScanner{
		fs:        NewOSFileSystem(),
		envLookup: os.Getenv,
		known:     DefaultKnownCachePaths(),
		repo:      NewCacheRepository(),
	}
	if home, err := os.UserHomeDir(); err == nil {
		s.home = home
	}
	for _, opt := range opts {
		opt(s)
	}
	if s.fs == nil {
		s.fs = NewOSFileSystem()
	}
	return s
}

// Scan measures all known cache paths that exist and optionally updates the repo.
// If managerID is non-empty only that manager is scanned.
func (s *CacheScanner) Scan(ctx context.Context, managerID string) ([]*cleanup.CacheInfo, error) {
	paths := ResolveCachePaths(s.home, s.goos, s.envLookup, s.known)
	results := make([]*cleanup.CacheInfo, 0, len(paths))

	for id, path := range paths {
		if managerID != "" && id != managerID {
			continue
		}
		info, err := s.scanPath(ctx, id, path)
		if err != nil {
			// Skip missing paths; report other errors.
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("scan %s (%s): %w", id, path, err)
		}
		if info == nil {
			continue
		}
		results = append(results, info)
		if s.repo != nil {
			if err := s.repo.UpdateInfo(ctx, info); err != nil {
				return nil, fmt.Errorf("update cache info for %s: %w", id, err)
			}
		}
	}
	return results, nil
}

// ScanPath measures a single cache directory.
func (s *CacheScanner) scanPath(_ context.Context, managerID, path string) (*cleanup.CacheInfo, error) {
	info, err := s.fs.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", path)
	}

	var (
		totalBytes int64
		entries    int
		oldest     time.Time
		newest     time.Time
	)

	err = s.fs.WalkDir(path, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			// Skip unreadable nodes rather than failing the whole scan.
			return nil
		}
		if d == nil || d.IsDir() {
			return nil
		}
		fi, statErr := d.Info()
		if statErr != nil {
			return nil
		}
		totalBytes += fi.Size()
		entries++
		mt := fi.ModTime()
		if oldest.IsZero() || mt.Before(oldest) {
			oldest = mt
		}
		if newest.IsZero() || mt.After(newest) {
			newest = mt
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &cleanup.CacheInfo{
		ManagerID:   managerID,
		CachePath:   path,
		TotalSizeMB: float64(totalBytes) / (1024 * 1024),
		EntryCount:  entries,
		OldestEntry: oldest,
		NewestEntry: newest,
	}, nil
}

// Clean removes cache directory contents for the given manager (or all known).
// When dryRun is true, no deletions are performed; the Summary still reports
// what would be freed based on a scan.
func (s *CacheScanner) Clean(ctx context.Context, managerID string, dryRun bool) (*cleanup.Summary, error) {
	start := time.Now()
	scanned, err := s.Scan(ctx, managerID)
	if err != nil {
		return nil, err
	}

	summary := &cleanup.Summary{
		Errors: make([]string, 0),
	}

	for _, info := range scanned {
		if info.CachePath == "" {
			continue
		}
		summary.PackagesRemoved += info.EntryCount
		summary.SpaceFreedMB += info.TotalSizeMB

		if dryRun {
			continue
		}

		// Remove contents but keep the cache directory itself when possible.
		if err := s.clearDir(info.CachePath); err != nil {
			summary.Errors = append(summary.Errors, fmt.Sprintf("%s: %v", info.ManagerID, err))
			continue
		}
		// Refresh repo entry to zeros after successful clean.
		if s.repo != nil {
			_ = s.repo.UpdateInfo(ctx, &cleanup.CacheInfo{
				ManagerID: info.ManagerID,
				CachePath: info.CachePath,
			})
		}
	}

	summary.Duration = time.Since(start)
	if len(summary.Errors) > 0 {
		return summary, fmt.Errorf("%w: %d path(s) failed", cleanup.ErrCacheClearFailed, len(summary.Errors))
	}
	return summary, nil
}

func (s *CacheScanner) clearDir(path string) error {
	entries, err := s.fs.ReadDir(path)
	if err != nil {
		// Fall back to RemoveAll on the directory itself.
		return s.fs.RemoveAll(path)
	}
	var firstErr error
	for _, e := range entries {
		child := filepath.Join(path, e.Name())
		if joinErr := s.fs.RemoveAll(child); joinErr != nil && firstErr == nil {
			firstErr = joinErr
		}
	}
	return firstErr
}
