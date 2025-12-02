package detector

import (
	"context"
	"os"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
)

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

	if len(env.Warnings) == 0 {
		t.Error("Expected warnings for conda environment")
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
		name            string
		setup           func()
		cleanup         func()
		expectWarnings  bool
		expectedContain string
	}{
		{
			name: "normal_no_warnings",
			setup: func() {
				os.Unsetenv("CONDA_DEFAULT_ENV")
				os.Unsetenv("VIRTUAL_ENV")
			},
			cleanup:        func() {},
			expectWarnings: false,
		},
		{
			name: "conda_has_warnings",
			setup: func() {
				os.Setenv("CONDA_DEFAULT_ENV", "test")
			},
			cleanup: func() {
				os.Unsetenv("CONDA_DEFAULT_ENV")
			},
			expectWarnings:  true,
			expectedContain: "Conda",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			defer tt.cleanup()

			detector := NewDetector(nil, &mockLogger{})
			warnings := detector.GetEnvironmentWarnings(context.Background())

			if tt.expectWarnings && len(warnings) == 0 {
				t.Error("Expected warnings, got none")
			}

			if !tt.expectWarnings && len(warnings) > 0 {
				t.Errorf("Expected no warnings, got %v", warnings)
			}

			if tt.expectedContain != "" && len(warnings) > 0 {
				found := false
				for _, w := range warnings {
					if contains(w, tt.expectedContain) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected warning containing %q, got %v", tt.expectedContain, warnings)
				}
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

// contains checks if s contains substr (case-sensitive).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
