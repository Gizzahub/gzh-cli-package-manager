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
