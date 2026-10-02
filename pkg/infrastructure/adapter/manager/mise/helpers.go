package mise

import (
	"fmt"
	"strings"

	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
)

func effectiveStrategy(opts adapterm.UpdateOptions) adapterm.UpdateStrategy {
	if opts.Strategy == "" {
		return adapterm.StrategyStable
	}
	return opts.Strategy
}

func updateFailure(result *adapterm.UpdateResult, name string, err error) (*adapterm.UpdateResult, error) {
	result.Success, result.Message = false, err.Error()
	result.FailedPackages = append(result.FailedPackages, name)
	return result, err
}

func preview(stdout, stderr string) string {
	if value := strings.TrimSpace(stdout); value != "" {
		return value
	}
	return strings.TrimSpace(stderr)
}

func validatePackageSelection(packages []string) error {
	if err := adapterm.ValidateToolSelection(packages); err != nil {
		return fmt.Errorf("mise update: %w", err)
	}
	return nil
}

func selectTools(tools []miseTool, selected []string) ([]miseTool, error) {
	if len(selected) == 0 {
		return tools, nil
	}
	wanted := make(map[string]bool, len(selected))
	for _, name := range selected {
		wanted[name] = true
	}
	filtered := make([]miseTool, 0, len(selected))
	found := make(map[string]bool, len(selected))
	for _, tool := range tools {
		if wanted[tool.Name] {
			filtered = append(filtered, tool)
			found[tool.Name] = true
		}
	}
	for _, name := range selected {
		if !found[name] {
			return nil, fmt.Errorf("mise update: selected tool %q is not active and installed; run mise install first", name)
		}
	}
	return filtered, nil
}
