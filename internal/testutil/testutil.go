// Package testutil provides test utilities for package manager testing.
package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// TempDir creates a temporary directory for testing and returns the path.
// The directory is automatically cleaned up when the test completes.
func TempDir(t *testing.T) string {
	t.Helper()

	dir, err := os.MkdirTemp("", "gz-pm-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})

	return dir
}

// TempFile creates a temporary file for testing and returns the path.
// The file is automatically cleaned up when the test completes.
func TempFile(t *testing.T, pattern string) string {
	t.Helper()

	file, err := os.CreateTemp("", pattern)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	path := file.Name()
	_ = file.Close()

	t.Cleanup(func() {
		_ = os.Remove(path)
	})

	return path
}

// WriteFile writes content to a file for testing.
func WriteFile(t *testing.T, path string, content []byte) {
	t.Helper()

	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		t.Fatalf("failed to create parent directory: %v", err)
	}

	err = os.WriteFile(path, content, 0644)
	if err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
}

// ReadFile reads a file and returns its content.
func ReadFile(t *testing.T, path string) []byte {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	return content
}

// AssertEqual fails the test if got != want.
func AssertEqual(t *testing.T, got, want interface{}) {
	t.Helper()

	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// AssertNotEqual fails the test if got == want.
func AssertNotEqual(t *testing.T, got, want interface{}) {
	t.Helper()

	if got == want {
		t.Errorf("got %v, want different value", got)
	}
}

// AssertNil fails the test if v is not nil.
func AssertNil(t *testing.T, v interface{}) {
	t.Helper()

	if v != nil {
		t.Errorf("expected nil, got %v", v)
	}
}

// AssertNotNil fails the test if v is nil.
func AssertNotNil(t *testing.T, v interface{}) {
	t.Helper()

	if v == nil {
		t.Errorf("expected non-nil value")
	}
}

// AssertNoError fails the test if err is not nil.
func AssertNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// AssertError fails the test if err is nil.
func AssertError(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// AssertStringContains fails the test if s does not contain substr.
func AssertStringContains(t *testing.T, s, substr string) {
	t.Helper()

	if !contains(s, substr) {
		t.Errorf("string %q does not contain %q", s, substr)
	}
}

// AssertStringNotContains fails the test if s contains substr.
func AssertStringNotContains(t *testing.T, s, substr string) {
	t.Helper()

	if contains(s, substr) {
		t.Errorf("string %q contains %q", s, substr)
	}
}

// AssertFileExists fails the test if the file does not exist.
func AssertFileExists(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("file %q does not exist", path)
	}
}

// AssertFileNotExists fails the test if the file exists.
func AssertFileNotExists(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); err == nil {
		t.Errorf("file %q exists", path)
	}
}

// AssertDirExists fails the test if the directory does not exist.
func AssertDirExists(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		t.Errorf("directory %q does not exist", path)
		return
	}
	if err != nil {
		t.Fatalf("failed to stat directory: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("%q is not a directory", path)
	}
}

// SkipIfCI skips the test if running in CI environment.
func SkipIfCI(t *testing.T) {
	t.Helper()

	if os.Getenv("CI") != "" {
		t.Skip("Skipping test in CI environment")
	}
}

// SkipIfNotCI skips the test if not running in CI environment.
func SkipIfNotCI(t *testing.T) {
	t.Helper()

	if os.Getenv("CI") == "" {
		t.Skip("Skipping test outside CI environment")
	}
}

// SetEnv sets an environment variable for the duration of the test.
func SetEnv(t *testing.T, key, value string) {
	t.Helper()

	old := os.Getenv(key)
	_ = os.Setenv(key, value)

	t.Cleanup(func() {
		if old == "" {
			_ = os.Unsetenv(key)
		} else {
			_ = os.Setenv(key, old)
		}
	})
}

// Chdir changes the working directory for the duration of the test.
func Chdir(t *testing.T, dir string) {
	t.Helper()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}

	err = os.Chdir(dir)
	if err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})
}

// Helper function to check if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
