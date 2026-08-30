// Package version provides shared version comparison utilities for adapters.
package version

import (
	"strings"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
)

// DetermineUpdateType compares two version strings and returns the update type.
// It follows semantic versioning conventions:
// - Different major version -> UpdateMajor
// - Different minor version (same major) -> UpdateMinor
// - Different patch version (same major/minor) -> UpdatePatch.
func DetermineUpdateType(current, latest string) manager.UpdateType {
	// Remove 'v' prefix if present
	current = strings.TrimPrefix(current, "v")
	latest = strings.TrimPrefix(latest, "v")

	currentParts := strings.Split(current, ".")
	latestParts := strings.Split(latest, ".")

	if len(currentParts) == 0 || len(latestParts) == 0 {
		return manager.UpdateMinor
	}

	// Compare major version
	if len(currentParts) > 0 && len(latestParts) > 0 {
		if currentParts[0] != latestParts[0] {
			return manager.UpdateMajor
		}
	}

	// Compare minor version
	if len(currentParts) > 1 && len(latestParts) > 1 {
		if currentParts[1] != latestParts[1] {
			return manager.UpdateMinor
		}
	}

	// Default to patch for any other difference
	return manager.UpdatePatch
}
