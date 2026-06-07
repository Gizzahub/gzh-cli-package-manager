// Package cmdutil provides shared utilities for command execution in adapters.
package cmdutil

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
)

// CheckResult validates command execution result and returns an error if failed.
// It handles both execution errors and non-zero exit codes.
func CheckResult(result *output.ExecutionResult, err error, operation string) error {
	if err != nil {
		return fmt.Errorf("%s: %w", operation, err)
	}
	if result.ExitCode != 0 {
		stderr := strings.TrimSpace(result.Stderr)
		if stderr == "" {
			return fmt.Errorf("%s failed with exit code %d", operation, result.ExitCode)
		}
		return fmt.Errorf("%s failed: %s", operation, stderr)
	}
	return nil
}

// ExtractStdout returns trimmed stdout from the result.
func ExtractStdout(result *output.ExecutionResult) string {
	return strings.TrimSpace(result.Stdout)
}

// UnmarshalJSON parses JSON output from command result into the given value.
func UnmarshalJSON(result *output.ExecutionResult, v any, operation string) error {
	if err := json.Unmarshal([]byte(result.Stdout), v); err != nil {
		return fmt.Errorf("%s: failed to parse output: %w", operation, err)
	}
	return nil
}

// ExtractVersionField extracts version from output with format "Name X.Y.Z".
// minFields specifies minimum number of space-separated fields expected.
func ExtractVersionField(stdout string, fieldIndex int, operation string) (string, error) {
	parts := strings.Fields(strings.TrimSpace(stdout))
	if len(parts) <= fieldIndex {
		return "", fmt.Errorf("%s: unexpected version format: %s", operation, stdout)
	}
	return parts[fieldIndex], nil
}

// IsCommandAvailable checks if command exists using "which" command result.
func IsCommandAvailable(result *output.ExecutionResult, err error) bool {
	if err != nil {
		return false
	}
	return result.ExitCode == 0 && strings.TrimSpace(result.Stdout) != ""
}
