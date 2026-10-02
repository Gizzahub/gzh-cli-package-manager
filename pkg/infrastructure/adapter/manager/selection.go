package manager

import (
	"fmt"
	"strings"
)

// ValidateToolSelection distinguishes an omitted selection from an empty allowlist.
func ValidateToolSelection(tools []string) error {
	if tools != nil && len(tools) == 0 {
		return fmt.Errorf("selected tools must not be empty")
	}
	seen := make(map[string]struct{}, len(tools))
	for _, name := range tools {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("selected tool name is empty")
		}
		if strings.TrimSpace(name) != name {
			return fmt.Errorf("selected tool %q contains surrounding whitespace", name)
		}
		if _, exists := seen[name]; exists {
			return fmt.Errorf("selected tool %q is duplicated", name)
		}
		seen[name] = struct{}{}
	}
	return nil
}
