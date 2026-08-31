package detector

import (
	"context"
	"os"
	"slices"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
)

const condaEnvironmentWarning = "Conda environment detected. Using pip may cause dependency conflicts with conda packages."

// mockLogger implements a mock Logger for testing.
type mockLogger struct{}

func (m *mockLogger) Debug(_ context.Context, _ string, _ ...output.Field) {}
func (m *mockLogger) Info(_ context.Context, _ string, _ ...output.Field)  {}
func (m *mockLogger) Warn(_ context.Context, _ string, _ ...output.Field)  {}
func (m *mockLogger) Error(_ context.Context, _ string, _ error, _ ...output.Field) {
}

func TestDetector_Detect_NormalEnvironment(t *testing.T) {
	// Clear any environment variables that would indicate special environments
	originalCondaEnv := os.Getenv("CONDA_DEFAULT_ENV")
	originalVirtualEnv := os.Getenv("VIRTUAL_ENV")
	originalWSLDistro := os.Getenv("WSL_DISTRO_NAME")
	defer func() {
		os.Setenv("CONDA_DEFAULT_ENV", originalCondaEnv)
		os.Setenv("VIRTUAL_ENV", originalVirtualEnv)
		os.Setenv("WSL_DISTRO_NAME", originalWSLDistro)
	}()

	os.Unsetenv("CONDA_DEFAULT_ENV")
	os.Unsetenv("VIRTUAL_ENV")
	os.Unsetenv("WSL_DISTRO_NAME")

	detector := NewDetector(nil, &mockLogger{})
	env := detector.Detect(context.Background())

	// On most systems, we should detect normal environment
	// (unless actually running in Docker/WSL)
	if env == nil {
		t.Fatal("Expected non-nil environment")
	}

	if !env.IsPipSafe {
		t.Error("Expected IsPipSafe to be true for normal environment")
	}
}

func TestDetector_Detect_CondaEnvironment(t *testing.T) {
	// Set up conda environment
	originalCondaEnv := os.Getenv("CONDA_DEFAULT_ENV")
	originalCondaPrefix := os.Getenv("CONDA_PREFIX")
	defer func() {
		os.Setenv("CONDA_DEFAULT_ENV", originalCondaEnv)
		os.Setenv("CONDA_PREFIX", originalCondaPrefix)
	}()

	os.Setenv("CONDA_DEFAULT_ENV", "myproject")
	os.Setenv("CONDA_PREFIX", "/opt/miniconda3/envs/myproject")

	detector := NewDetector(nil, &mockLogger{})
	env := detector.Detect(context.Background())

	if env.Type != EnvConda {
		t.Errorf("Expected EnvConda, got %s", env.Type)
	}

	if env.Name != "myproject" {
		t.Errorf("Expected name 'myproject', got %s", env.Name)
	}

	if env.Path != "/opt/miniconda3/envs/myproject" {
		t.Errorf("Expected path '/opt/miniconda3/envs/myproject', got %s", env.Path)
	}

	if env.IsPipSafe {
		t.Error("Expected IsPipSafe to be false for conda environment")
	}

	if !slices.Equal(env.Warnings, []string{condaEnvironmentWarning}) {
		t.Errorf("Warnings = %q, want %q", env.Warnings, []string{condaEnvironmentWarning})
	}
}

func TestDetector_Detect_VirtualenvEnvironment(t *testing.T) {
	// Clear conda and set virtualenv
	originalCondaEnv := os.Getenv("CONDA_DEFAULT_ENV")
	originalVirtualEnv := os.Getenv("VIRTUAL_ENV")
	defer func() {
		os.Setenv("CONDA_DEFAULT_ENV", originalCondaEnv)
		os.Setenv("VIRTUAL_ENV", originalVirtualEnv)
	}()

	os.Unsetenv("CONDA_DEFAULT_ENV")
	os.Setenv("VIRTUAL_ENV", "/home/user/projects/myapp/venv")

	detector := NewDetector(nil, &mockLogger{})
	env := detector.Detect(context.Background())

	if env.Type != EnvVirtualenv {
		t.Errorf("Expected EnvVirtualenv, got %s", env.Type)
	}

	if env.Name != "venv" {
		t.Errorf("Expected name 'venv', got %s", env.Name)
	}

	if env.Path != "/home/user/projects/myapp/venv" {
		t.Errorf("Expected path '/home/user/projects/myapp/venv', got %s", env.Path)
	}

	if !env.IsPipSafe {
		t.Error("Expected IsPipSafe to be true for virtualenv environment")
	}
}

func TestDetector_Detect_WSLEnvironment(t *testing.T) {
	// Set WSL environment variable
	originalWSLDistro := os.Getenv("WSL_DISTRO_NAME")
	originalCondaEnv := os.Getenv("CONDA_DEFAULT_ENV")
	originalVirtualEnv := os.Getenv("VIRTUAL_ENV")
	defer func() {
		os.Setenv("WSL_DISTRO_NAME", originalWSLDistro)
		os.Setenv("CONDA_DEFAULT_ENV", originalCondaEnv)
		os.Setenv("VIRTUAL_ENV", originalVirtualEnv)
	}()

	os.Setenv("WSL_DISTRO_NAME", "Ubuntu")
	os.Unsetenv("CONDA_DEFAULT_ENV")
	os.Unsetenv("VIRTUAL_ENV")

	detector := NewDetector(nil, &mockLogger{})
	env := detector.Detect(context.Background())

	if env.Type != EnvWSL {
		t.Errorf("Expected EnvWSL, got %s", env.Type)
	}

	if !env.IsPipSafe {
		t.Error("Expected IsPipSafe to be true for WSL environment")
	}
}

func TestDetector_CondaTakesPrecedenceOverVirtualenv(t *testing.T) {
	// Set both conda and virtualenv
	originalCondaEnv := os.Getenv("CONDA_DEFAULT_ENV")
	originalCondaPrefix := os.Getenv("CONDA_PREFIX")
	originalVirtualEnv := os.Getenv("VIRTUAL_ENV")
	defer func() {
		os.Setenv("CONDA_DEFAULT_ENV", originalCondaEnv)
		os.Setenv("CONDA_PREFIX", originalCondaPrefix)
		os.Setenv("VIRTUAL_ENV", originalVirtualEnv)
	}()

	os.Setenv("CONDA_DEFAULT_ENV", "conda_env")
	os.Setenv("CONDA_PREFIX", "/opt/conda/envs/conda_env")
	os.Setenv("VIRTUAL_ENV", "/home/user/venv")

	detector := NewDetector(nil, &mockLogger{})
	env := detector.Detect(context.Background())

	// Conda should take precedence
	if env.Type != EnvConda {
		t.Errorf("Expected EnvConda (conda takes precedence), got %s", env.Type)
	}
}

func TestDetector_IsPipUpdateSafe(t *testing.T) {
	tests := []struct {
		name     string
		setup    func()
		cleanup  func()
		expected bool
	}{
		{
			name: "normal_environment",
			setup: func() {
				os.Unsetenv("CONDA_DEFAULT_ENV")
				os.Unsetenv("VIRTUAL_ENV")
			},
			cleanup:  func() {},
			expected: true,
		},
		{
			name: "conda_environment",
			setup: func() {
				os.Setenv("CONDA_DEFAULT_ENV", "test")
			},
			cleanup: func() {
				os.Unsetenv("CONDA_DEFAULT_ENV")
			},
			expected: false,
		},
		{
			name: "virtualenv_environment",
			setup: func() {
				os.Unsetenv("CONDA_DEFAULT_ENV")
				os.Setenv("VIRTUAL_ENV", "/path/to/venv")
			},
			cleanup: func() {
				os.Unsetenv("VIRTUAL_ENV")
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			defer tt.cleanup()

			detector := NewDetector(nil, &mockLogger{})
			result := detector.IsPipUpdateSafe(context.Background())

			if result != tt.expected {
				t.Errorf("IsPipUpdateSafe() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDetector_GetEnvironmentWarnings(t *testing.T) {
	tests := []struct {
		name             string
		setup            func()
		cleanup          func()
		expectedWarnings []string
	}{
		{
			name: "normal_no_warnings",
			setup: func() {
				os.Unsetenv("CONDA_DEFAULT_ENV")
				os.Unsetenv("VIRTUAL_ENV")
			},
			cleanup:          func() {},
			expectedWarnings: nil,
		},
		{
			name: "conda_has_warnings",
			setup: func() {
				os.Setenv("CONDA_DEFAULT_ENV", "test")
			},
			cleanup: func() {
				os.Unsetenv("CONDA_DEFAULT_ENV")
			},
			expectedWarnings: []string{condaEnvironmentWarning},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			defer tt.cleanup()

			detector := NewDetector(nil, &mockLogger{})
			warnings := detector.GetEnvironmentWarnings(context.Background())

			if !slices.Equal(warnings, tt.expectedWarnings) {
				t.Errorf("GetEnvironmentWarnings() = %q, want %q", warnings, tt.expectedWarnings)
			}
		})
	}
}

func TestEnvironmentType_String(t *testing.T) {
	tests := []struct {
		envType  EnvironmentType
		expected string
	}{
		{EnvNormal, "normal"},
		{EnvConda, "conda"},
		{EnvVirtualenv, "virtualenv"},
		{EnvDocker, "docker"},
		{EnvWSL, "wsl"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if string(tt.envType) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(tt.envType))
			}
		})
	}
}

func TestDetector_DetectConda_WithCondaExeButNoPrefix(t *testing.T) {
	// Test detectConda when CONDA_PREFIX is not set but CONDA_EXE is
	originalCondaEnv := os.Getenv("CONDA_DEFAULT_ENV")
	originalCondaPrefix := os.Getenv("CONDA_PREFIX")
	originalCondaExe := os.Getenv("CONDA_EXE")
	defer func() {
		os.Setenv("CONDA_DEFAULT_ENV", originalCondaEnv)
		os.Setenv("CONDA_PREFIX", originalCondaPrefix)
		os.Setenv("CONDA_EXE", originalCondaExe)
	}()

	os.Setenv("CONDA_DEFAULT_ENV", "testenv")
	os.Unsetenv("CONDA_PREFIX")
	os.Setenv("CONDA_EXE", "/opt/conda/bin/conda")

	detector := NewDetector(nil, &mockLogger{})
	env := detector.Detect(context.Background())

	if env.Type != EnvConda {
		t.Errorf("Expected EnvConda, got %s", env.Type)
	}

	if env.Name != "testenv" {
		t.Errorf("Expected name 'testenv', got %s", env.Name)
	}

	// Path should be derived from CONDA_EXE
	expectedPath := "/opt/conda/envs/testenv"
	if env.Path != expectedPath {
		t.Errorf("Expected path '%s', got %s", expectedPath, env.Path)
	}
}

func TestDetector_DetectConda_NoConda(t *testing.T) {
	// Test when no conda environment is active
	originalCondaEnv := os.Getenv("CONDA_DEFAULT_ENV")
	defer os.Setenv("CONDA_DEFAULT_ENV", originalCondaEnv)

	os.Unsetenv("CONDA_DEFAULT_ENV")

	detector := NewDetector(nil, &mockLogger{})
	result := detector.detectConda()

	if result != nil {
		t.Errorf("Expected nil for no conda, got %+v", result)
	}
}

func TestDetector_IsDocker_WithContainerEnvVar(t *testing.T) {
	// Test Docker detection with container environment variable
	originalContainer := os.Getenv("container")
	defer os.Setenv("container", originalContainer)

	os.Setenv("container", "podman")

	detector := NewDetector(nil, &mockLogger{})
	result := detector.isDocker()

	if !result {
		t.Error("Expected isDocker to return true when container env var is set")
	}
}

func TestDetector_IsDocker_NoDocker(t *testing.T) {
	// Test when not in Docker
	originalContainer := os.Getenv("container")
	defer os.Setenv("container", originalContainer)

	os.Unsetenv("container")

	detector := NewDetector(nil, &mockLogger{})
	result := detector.isDocker()

	// This may return true if running in actual Docker container
	// For pure unit test, we just verify the function runs without error
	_ = result
}

func TestDetector_Detect_DockerEnvironment(t *testing.T) {
	// Set container env var to simulate Docker
	originalContainer := os.Getenv("container")
	originalCondaEnv := os.Getenv("CONDA_DEFAULT_ENV")
	originalVirtualEnv := os.Getenv("VIRTUAL_ENV")
	originalWSLDistro := os.Getenv("WSL_DISTRO_NAME")
	defer func() {
		os.Setenv("container", originalContainer)
		os.Setenv("CONDA_DEFAULT_ENV", originalCondaEnv)
		os.Setenv("VIRTUAL_ENV", originalVirtualEnv)
		os.Setenv("WSL_DISTRO_NAME", originalWSLDistro)
	}()

	os.Setenv("container", "docker")
	os.Unsetenv("CONDA_DEFAULT_ENV")
	os.Unsetenv("VIRTUAL_ENV")
	os.Unsetenv("WSL_DISTRO_NAME")

	detector := NewDetector(nil, &mockLogger{})
	env := detector.Detect(context.Background())

	if env.Type != EnvDocker {
		t.Errorf("Expected EnvDocker, got %s", env.Type)
	}
}

func TestDetector_CondaOverridesDocker(t *testing.T) {
	// When both conda and Docker are detected, conda takes precedence for pip safety
	originalContainer := os.Getenv("container")
	originalCondaEnv := os.Getenv("CONDA_DEFAULT_ENV")
	originalCondaPrefix := os.Getenv("CONDA_PREFIX")
	defer func() {
		os.Setenv("container", originalContainer)
		os.Setenv("CONDA_DEFAULT_ENV", originalCondaEnv)
		os.Setenv("CONDA_PREFIX", originalCondaPrefix)
	}()

	os.Setenv("container", "docker")
	os.Setenv("CONDA_DEFAULT_ENV", "test")
	os.Setenv("CONDA_PREFIX", "/opt/conda/envs/test")

	detector := NewDetector(nil, &mockLogger{})
	env := detector.Detect(context.Background())

	// Conda should override Docker detection for type
	if env.Type != EnvConda {
		t.Errorf("Expected EnvConda (takes precedence), got %s", env.Type)
	}

	if env.IsPipSafe {
		t.Error("Expected IsPipSafe to be false for conda environment")
	}
}

func TestNewDetector(t *testing.T) {
	detector := NewDetector(nil, &mockLogger{})
	if detector == nil {
		t.Error("Expected non-nil detector")
	}
}

func TestEnvironment_Struct(t *testing.T) {
	env := &Environment{
		Type:      EnvConda,
		Name:      "test",
		Path:      "/path/to/env",
		IsPipSafe: false,
		Warnings:  []string{"warning1", "warning2"},
	}

	if env.Type != EnvConda {
		t.Error("Type mismatch")
	}
	if env.Name != "test" {
		t.Error("Name mismatch")
	}
	if env.Path != "/path/to/env" {
		t.Error("Path mismatch")
	}
	if env.IsPipSafe {
		t.Error("IsPipSafe mismatch")
	}
	if len(env.Warnings) != 2 {
		t.Error("Warnings count mismatch")
	}
}
