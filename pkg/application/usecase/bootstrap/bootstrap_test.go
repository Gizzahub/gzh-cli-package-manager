package bootstrap_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/dto"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/usecase/bootstrap"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testBrewManagerName   = "brew"
	testMissingConfigPath = "/nonexistent/file.yaml"
)

// mockManagerRepository is a test double for manager.Repository.
type mockManagerRepository struct {
	findErr  error
	managers []*manager.Manager
}

func (m *mockManagerRepository) FindAll(ctx context.Context) ([]*manager.Manager, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.managers, nil
}

func (m *mockManagerRepository) FindInstalled(ctx context.Context) ([]*manager.Manager, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	installed := make([]*manager.Manager, 0)
	for _, mgr := range m.managers {
		if mgr.Installed {
			installed = append(installed, mgr)
		}
	}
	return installed, nil
}

func (m *mockManagerRepository) FindByID(ctx context.Context, id manager.ManagerID) (*manager.Manager, error) {
	for _, mgr := range m.managers {
		if mgr.ID == id {
			return mgr, nil
		}
	}
	return nil, nil
}

func (m *mockManagerRepository) Save(ctx context.Context, mgr *manager.Manager) error {
	return nil
}

func (m *mockManagerRepository) Delete(ctx context.Context, id manager.ManagerID) error {
	return nil
}

func TestBootstrap_InteractiveModePrefersDefaultConfig(t *testing.T) {
	// Setup
	mockRepo := &mockManagerRepository{
		managers: []*manager.Manager{
			{ID: manager.ManagerHomebrew, Name: testBrewManagerName, Installed: true},
		},
	}
	log := logger.NewStructuredLogger("test")
	uc := bootstrap.NewUseCase(mockRepo, log)

	// Execute
	req := &dto.BootstrapRequest{
		ConfigPath:  testMissingConfigPath,
		Interactive: true,
		DryRun:      true,
	}
	resp, err := uc.Bootstrap(context.Background(), req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.DryRun)
	assert.NotEmpty(t, resp.Results)
}

func TestBootstrap_WithConfigFile(t *testing.T) {
	// Setup
	mockRepo := &mockManagerRepository{
		managers: []*manager.Manager{
			{ID: manager.ManagerHomebrew, Name: testBrewManagerName, Installed: false},
			{ID: manager.ManagerNPM, Name: "npm", Installed: true},
		},
	}
	log := logger.NewStructuredLogger("test")
	uc := bootstrap.NewUseCase(mockRepo, log)

	// Create temp config file
	configContent := `managers:
  - id: brew
    enabled: true
  - id: npm
    enabled: true
preferences:
  skip_already_installed: true
  fail_on_error: false
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "bootstrap.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0o644))

	// Execute
	req := &dto.BootstrapRequest{
		ConfigPath: configPath,
		DryRun:     true,
	}
	resp, err := uc.Bootstrap(context.Background(), req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.DryRun)
	assert.Len(t, resp.Results, 2)

	// Check brew result (not installed, would install)
	brewResult := resp.Results[0]
	assert.Equal(t, manager.ManagerHomebrew, brewResult.ID)
	assert.False(t, brewResult.AlreadyInstalled)
	assert.True(t, brewResult.Success)
	assert.Contains(t, brewResult.Steps[0], "Would install")

	// Check npm result (already installed, skipped)
	npmResult := resp.Results[1]
	assert.Equal(t, manager.ManagerNPM, npmResult.ID)
	assert.True(t, npmResult.AlreadyInstalled)
	assert.True(t, npmResult.Skipped)
	assert.Equal(t, "already installed", npmResult.SkipReason)
	assert.Equal(t, 1, resp.Summary.AlreadyInstalledManagers)
	assert.Equal(t, 0, resp.Summary.SkippedManagers)
	assert.Equal(t, 1, resp.Summary.InstalledManagers)
	assert.Equal(t, 0, resp.Summary.FailedManagers)
}

func TestBootstrap_DisabledManager(t *testing.T) {
	// Setup
	mockRepo := &mockManagerRepository{
		managers: []*manager.Manager{},
	}
	log := logger.NewStructuredLogger("test")
	uc := bootstrap.NewUseCase(mockRepo, log)

	// Create temp config file with disabled manager
	configContent := `managers:
  - id: brew
    enabled: false
preferences:
  skip_already_installed: true
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "bootstrap.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0o644))

	// Execute
	req := &dto.BootstrapRequest{
		ConfigPath: configPath,
	}
	resp, err := uc.Bootstrap(context.Background(), req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Results, 1)

	result := resp.Results[0]
	assert.True(t, result.Skipped)
	assert.Equal(t, "disabled in configuration", result.SkipReason)
	assert.Equal(t, 1, resp.Summary.SkippedManagers)
}

func TestBootstrap_AlreadyInstalledWithoutSkip(t *testing.T) {
	// Setup
	mockRepo := &mockManagerRepository{
		managers: []*manager.Manager{
			{ID: manager.ManagerHomebrew, Name: testBrewManagerName, Installed: true},
		},
	}
	log := logger.NewStructuredLogger("test")
	uc := bootstrap.NewUseCase(mockRepo, log)

	// Create temp config file with skip_already_installed=false
	configContent := `managers:
  - id: brew
    enabled: true
preferences:
  skip_already_installed: false
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "bootstrap.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0o644))

	// Execute
	req := &dto.BootstrapRequest{
		ConfigPath: configPath,
		DryRun:     true,
	}
	resp, err := uc.Bootstrap(context.Background(), req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Results, 1)

	result := resp.Results[0]
	assert.True(t, result.AlreadyInstalled)
	assert.False(t, result.Skipped)
	assert.True(t, result.Success)
	assert.Contains(t, result.Steps[0], "already installed")
	assert.Equal(t, 1, resp.Summary.AlreadyInstalledManagers)
}

func TestBootstrap_InstallationNotImplemented(t *testing.T) {
	// Setup
	mockRepo := &mockManagerRepository{
		managers: []*manager.Manager{}, // No managers installed
	}
	log := logger.NewStructuredLogger("test")
	uc := bootstrap.NewUseCase(mockRepo, log)

	// Create temp config file
	configContent := `managers:
  - id: brew
    enabled: true
preferences:
  skip_already_installed: true
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "bootstrap.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0o644))

	// Execute (not dry-run, so should fail with "not implemented")
	req := &dto.BootstrapRequest{
		ConfigPath: configPath,
		DryRun:     false,
	}
	resp, err := uc.Bootstrap(context.Background(), req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Results, 1)

	result := resp.Results[0]
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "not yet implemented")
	assert.Equal(t, 1, resp.Summary.FailedManagers)
}

func TestBootstrap_NoConfigOrInteractive(t *testing.T) {
	// Setup
	mockRepo := &mockManagerRepository{}
	log := logger.NewStructuredLogger("test")
	uc := bootstrap.NewUseCase(mockRepo, log)

	// Execute
	req := &dto.BootstrapRequest{
		ConfigPath:  "",
		Interactive: false,
	}
	_, err := uc.Bootstrap(context.Background(), req)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "either --config or --interactive must be specified")
}

func TestBootstrap_InvalidConfigFile(t *testing.T) {
	// Setup
	mockRepo := &mockManagerRepository{}
	log := logger.NewStructuredLogger("test")
	uc := bootstrap.NewUseCase(mockRepo, log)

	// Execute with non-existent file
	req := &dto.BootstrapRequest{
		ConfigPath: testMissingConfigPath,
	}
	_, err := uc.Bootstrap(context.Background(), req)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load config")
}

func TestBootstrap_InvalidYAML(t *testing.T) {
	// Setup
	mockRepo := &mockManagerRepository{}
	log := logger.NewStructuredLogger("test")
	uc := bootstrap.NewUseCase(mockRepo, log)

	// Create temp config file with invalid YAML
	configContent := `managers:
  - id: brew
    enabled: true
    invalid_indent
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "bootstrap.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0o644))

	// Execute
	req := &dto.BootstrapRequest{
		ConfigPath: configPath,
	}
	_, err := uc.Bootstrap(context.Background(), req)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse YAML")
}

func TestBootstrap_RepositoryError(t *testing.T) {
	// Setup
	mockRepo := &mockManagerRepository{
		findErr: assert.AnError,
	}
	log := logger.NewStructuredLogger("test")
	uc := bootstrap.NewUseCase(mockRepo, log)

	// Execute
	req := &dto.BootstrapRequest{
		Interactive: true,
	}
	_, err := uc.Bootstrap(context.Background(), req)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check installed managers")
}

func TestBootstrap_Summary(t *testing.T) {
	// Setup
	mockRepo := &mockManagerRepository{
		managers: []*manager.Manager{
			{ID: manager.ManagerHomebrew, Name: testBrewManagerName, Installed: true},
			{ID: manager.ManagerNPM, Name: "npm", Installed: false},
		},
	}
	log := logger.NewStructuredLogger("test")
	uc := bootstrap.NewUseCase(mockRepo, log)

	// Create temp config file
	configContent := `managers:
  - id: brew
    enabled: true
  - id: npm
    enabled: true
  - id: asdf
    enabled: false
preferences:
  skip_already_installed: true
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "bootstrap.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0o644))

	// Execute
	req := &dto.BootstrapRequest{
		ConfigPath: configPath,
		DryRun:     true,
	}
	resp, err := uc.Bootstrap(context.Background(), req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, resp.Summary)
	assert.Equal(t, 3, resp.Summary.TotalManagers)
	assert.Equal(t, 1, resp.Summary.InstalledManagers)        // npm would be installed
	assert.Equal(t, 1, resp.Summary.AlreadyInstalledManagers) // brew already installed
	assert.Equal(t, 1, resp.Summary.SkippedManagers)          // asdf disabled
	assert.Equal(t, 0, resp.Summary.FailedManagers)
	assert.Greater(t, resp.Summary.TotalDuration, 0.0)
}

func TestBootstrap_FailOnError(t *testing.T) {
	// Setup
	mockRepo := &mockManagerRepository{
		managers: []*manager.Manager{}, // No managers installed
	}
	log := logger.NewStructuredLogger("test")
	uc := bootstrap.NewUseCase(mockRepo, log)

	// Create temp config file with fail_on_error=true
	configContent := `managers:
  - id: brew
    enabled: true
  - id: npm
    enabled: true
preferences:
  fail_on_error: true
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "bootstrap.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0o644))

	// Execute (not dry-run, so should fail)
	req := &dto.BootstrapRequest{
		ConfigPath: configPath,
		DryRun:     false,
	}
	resp, err := uc.Bootstrap(context.Background(), req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, resp)
	// Should stop after first failure
	assert.Equal(t, 1, resp.Summary.FailedManagers)
	assert.Len(t, resp.Results, 1) // Only brew should be processed
}
