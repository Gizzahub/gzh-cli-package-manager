package manager

import (
	"context"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
)

// Adapter defines the interface for package manager adapters.
// Each package manager (Homebrew, asdf, npm, etc.) implements this interface.
type Adapter interface {
	// Detect checks if the package manager is installed on the system.
	Detect(ctx context.Context) (bool, error)

	// GetVersion retrieves the version of the installed package manager.
	GetVersion(ctx context.Context) (string, error)

	// GetBinaryPath returns the path to the package manager binary.
	GetBinaryPath(ctx context.Context) (string, error)

	// GetConfigPath returns the path to the package manager configuration.
	GetConfigPath(ctx context.Context) (string, error)

	// ListPackages retrieves all packages managed by this manager.
	ListPackages(ctx context.Context) ([]manager.Package, error)

	// CheckHealth performs health checks on the package manager.
	CheckHealth(ctx context.Context) (manager.Status, error)

	// Update performs update operations on the package manager and its packages.
	Update(ctx context.Context, opts UpdateOptions) (*UpdateResult, error)
}

// Searcher is an optional capability for managers that support package search.
// Winget, Scoop, and Chocolatey implement this interface for per-manager CLI.
type Searcher interface {
	// Search finds packages matching query in the manager's sources/catalog.
	Search(ctx context.Context, query string) ([]manager.Package, error)
}

// Installer is an optional capability for install/uninstall operations.
// Winget, Scoop, and Chocolatey implement this for per-manager CLI.
type Installer interface {
	// Install installs a package by ID/name. When dryRun is true, no native
	// install is executed; the adapter reports what would run.
	Install(ctx context.Context, pkgID string, dryRun bool) error

	// Uninstall removes a package by ID/name. When dryRun is true, no native
	// uninstall is executed.
	Uninstall(ctx context.Context, pkgID string, dryRun bool) error
}

// Source describes a package source (e.g. winget catalog / msstore).
type Source struct {
	// Name is the source identifier (e.g. "winget", "msstore").
	Name string
	// Arg is the source argument/URL when available.
	Arg string
}

// SourceLister is an optional capability for managers that expose package sources.
type SourceLister interface {
	// ListSources returns configured package sources.
	ListSources(ctx context.Context) ([]Source, error)
}

// Bucket describes a Scoop bucket.
type Bucket struct {
	// Name is the bucket name (e.g. "main", "extras").
	Name string
	// Source is the remote URL or path when available.
	Source string
}

// BucketManager is an optional capability for Scoop-style bucket operations.
type BucketManager interface {
	// ListBuckets returns configured buckets.
	ListBuckets(ctx context.Context) ([]Bucket, error)
	// AddBucket adds a bucket by name (and optional URL).
	AddBucket(ctx context.Context, name, url string) error
	// RemoveBucket removes a bucket by name.
	RemoveBucket(ctx context.Context, name string) error
}

// UpdateOptions contains options for updating packages.
type UpdateOptions struct {
	// DryRun performs a dry run without actually updating.
	DryRun bool

	// Packages lists specific packages to update.
	// If empty, all packages are updated.
	Packages []string

	// Strategy specifies the update strategy.
	Strategy UpdateStrategy
}

// UpdateStrategy defines how packages should be updated.
type UpdateStrategy string

const (
	// StrategyLatest updates to the latest version (including pre-releases).
	StrategyLatest UpdateStrategy = "latest"

	// StrategyStable updates to the latest stable version only.
	StrategyStable UpdateStrategy = "stable"

	// StrategyMinor updates to the latest minor version (no major updates).
	StrategyMinor UpdateStrategy = "minor"

	// StrategyFixed does not update, only shows available updates.
	StrategyFixed UpdateStrategy = "fixed"
)

// UpdateResult contains the result of an update operation.
type UpdateResult struct {
	// Success indicates if the update succeeded.
	Success bool

	// UpdatedPackages lists packages that were updated.
	UpdatedPackages []string

	// FailedPackages lists packages that failed to update.
	FailedPackages []string

	// Message contains a human-readable status message.
	Message string
}
