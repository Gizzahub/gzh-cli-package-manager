package dto

import "fmt"

// UpdatePolicy contains a fully resolved per-manager update policy.
type UpdatePolicy struct {
	Strategy  UpdateStrategy
	Bump      bool
	MiseDir   string
	MiseLocal bool
	MiseTools []string
}

// ValidateUpdateStrategy rejects unknown strategies rather than widening them.
func ValidateUpdateStrategy(strategy UpdateStrategy) error {
	switch strategy {
	case "", StrategyStable, StrategyLatest, StrategyMinor, StrategyMicro, StrategyFixed:
		return nil
	default:
		return fmt.Errorf("unknown update strategy %q (supported: latest, stable, minor, micro, fixed)", strategy)
	}
}
