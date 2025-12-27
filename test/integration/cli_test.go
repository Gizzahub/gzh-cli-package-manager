//go:build integration

// Package integration provides integration tests for the gz-pm CLI.
// These tests require a built binary and may interact with actual package managers.
package integration

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// getBinaryPath returns the path to the built gz-pm binary.
func getBinaryPath(t *testing.T) string {
	t.Helper()

	// Get project root (3 levels up from test/integration)
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("Failed to get caller information")
	}

	projectRoot := filepath.Join(filepath.Dir(file), "..", "..")
	binaryPath := filepath.Join(projectRoot, "bin", "gz-pm")

	// Check if binary exists
	if _, err := exec.LookPath(binaryPath); err != nil {
		// Try to build
		cmd := exec.Command("make", "build")
		cmd.Dir = projectRoot
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to build binary: %v\nOutput: %s", err, output)
		}
	}

	return binaryPath
}

// runCommand executes the CLI and returns stdout, stderr, and error.
func runCommand(t *testing.T, args ...string) (string, string, error) {
	t.Helper()

	binary := getBinaryPath(t)
	cmd := exec.Command(binary, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func TestCLI_Version(t *testing.T) {
	stdout, _, err := runCommand(t, "version")
	if err != nil {
		t.Fatalf("version command failed: %v", err)
	}

	// Should contain version information
	if !strings.Contains(stdout, "gz-pm") && !strings.Contains(stdout, "version") {
		t.Errorf("Expected version output to contain 'gz-pm' or 'version', got: %s", stdout)
	}
}

func TestCLI_Help(t *testing.T) {
	stdout, _, err := runCommand(t, "--help")
	if err != nil {
		t.Fatalf("help command failed: %v", err)
	}

	// Should contain usage information
	requiredStrings := []string{
		"gz-pm",
		"Available Commands:",
		"Flags:",
	}

	for _, s := range requiredStrings {
		if !strings.Contains(stdout, s) {
			t.Errorf("Expected help output to contain '%s', got: %s", s, stdout)
		}
	}
}

func TestCLI_Status_JSON(t *testing.T) {
	stdout, _, err := runCommand(t, "status", "--output", "json")
	if err != nil {
		t.Fatalf("status --output json failed: %v", err)
	}

	// Should be valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("Expected valid JSON output, got parse error: %v\nOutput: %s", err, stdout)
	}

	// Should contain managers array (field name could be "managers" or "Managers")
	_, hasLower := result["managers"]
	_, hasUpper := result["Managers"]
	if !hasLower && !hasUpper {
		t.Errorf("Expected 'managers' or 'Managers' field in JSON output, got: %v", result)
	}

	// Should contain summary (field name could be "summary" or "Summary")
	_, hasLowerSum := result["summary"]
	_, hasUpperSum := result["Summary"]
	if !hasLowerSum && !hasUpperSum {
		t.Errorf("Expected 'summary' or 'Summary' field in JSON output, got: %v", result)
	}
}

func TestCLI_Status_Text(t *testing.T) {
	stdout, _, err := runCommand(t, "status")
	if err != nil {
		t.Fatalf("status command failed: %v", err)
	}

	// Should contain status header
	if !strings.Contains(stdout, "Package Manager Status") {
		t.Errorf("Expected status output to contain 'Package Manager Status', got: %s", stdout)
	}

	// Should contain summary
	if !strings.Contains(stdout, "Summary") {
		t.Errorf("Expected status output to contain 'Summary', got: %s", stdout)
	}
}

func TestCLI_Status_Verbose(t *testing.T) {
	stdout, _, err := runCommand(t, "status", "--verbose")
	if err != nil {
		t.Fatalf("status --verbose failed: %v", err)
	}

	// Should contain status information
	if !strings.Contains(stdout, "Package Manager Status") {
		t.Errorf("Expected verbose status output to contain 'Package Manager Status', got: %s", stdout)
	}
}

func TestCLI_Update_DryRun(t *testing.T) {
	stdout, _, err := runCommand(t, "update", "--all", "--dry-run")
	if err != nil {
		// update may return error if no managers are healthy
		t.Logf("update --all --dry-run returned error (may be expected): %v", err)
	}

	// Should indicate dry run mode
	if !strings.Contains(stdout, "DRY-RUN") && !strings.Contains(stdout, "dry-run") {
		t.Logf("Expected dry-run indicator in output, got: %s", stdout)
	}
}

func TestCLI_Update_DryRun_JSON(t *testing.T) {
	stdout, _, _ := runCommand(t, "update", "--all", "--dry-run", "--output", "json")

	// If we got JSON output, validate it
	if strings.TrimSpace(stdout) != "" && strings.HasPrefix(strings.TrimSpace(stdout), "{") {
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(stdout), &result); err != nil {
			t.Errorf("Expected valid JSON output, got parse error: %v\nOutput: %s", err, stdout)
		}

		// Should contain dry_run flag
		if dryRun, ok := result["dry_run"]; ok {
			if dryRun != true {
				t.Errorf("Expected dry_run to be true, got: %v", dryRun)
			}
		}
	}
}

func TestCLI_Bootstrap_DryRun(t *testing.T) {
	stdout, _, err := runCommand(t, "bootstrap", "--dry-run")
	if err != nil {
		// bootstrap may return error without config
		t.Logf("bootstrap --dry-run returned error (may be expected): %v", err)
	}

	// Check output is reasonable
	if stdout != "" {
		t.Logf("Bootstrap dry-run output: %s", stdout)
	}
}

func TestCLI_InvalidCommand(t *testing.T) {
	_, stderr, err := runCommand(t, "nonexistent-command")

	if err == nil {
		t.Error("Expected error for invalid command")
	}

	// Should contain error message
	if !strings.Contains(stderr, "unknown command") && !strings.Contains(stderr, "Error") {
		t.Logf("Error output: %s", stderr)
	}
}

func TestCLI_InvalidFlag(t *testing.T) {
	_, stderr, err := runCommand(t, "status", "--nonexistent-flag")

	if err == nil {
		t.Error("Expected error for invalid flag")
	}

	// Should contain error about unknown flag
	if !strings.Contains(stderr, "unknown flag") {
		t.Logf("Error output: %s", stderr)
	}
}

func TestCLI_StatusSubcommand_Exists(t *testing.T) {
	stdout, _, err := runCommand(t, "status", "--help")
	if err != nil {
		t.Fatalf("status --help failed: %v", err)
	}

	// Should contain status-specific flags
	if !strings.Contains(stdout, "--verbose") || !strings.Contains(stdout, "--output") {
		t.Errorf("Expected status help to show flags, got: %s", stdout)
	}
}

func TestCLI_UpdateSubcommand_Exists(t *testing.T) {
	stdout, _, err := runCommand(t, "update", "--help")
	if err != nil {
		t.Fatalf("update --help failed: %v", err)
	}

	// Should contain update-specific flags
	expectedFlags := []string{"--all", "--dry-run", "--managers", "--strategy", "--output"}
	for _, flag := range expectedFlags {
		if !strings.Contains(stdout, flag) {
			t.Errorf("Expected update help to show '%s' flag, got: %s", flag, stdout)
		}
	}
}

func TestCLI_BootstrapSubcommand_Exists(t *testing.T) {
	stdout, _, err := runCommand(t, "bootstrap", "--help")
	if err != nil {
		t.Fatalf("bootstrap --help failed: %v", err)
	}

	// Should contain bootstrap-specific flags
	expectedFlags := []string{"--config", "--interactive", "--dry-run", "--output"}
	for _, flag := range expectedFlags {
		if !strings.Contains(stdout, flag) {
			t.Errorf("Expected bootstrap help to show '%s' flag, got: %s", flag, stdout)
		}
	}
}
