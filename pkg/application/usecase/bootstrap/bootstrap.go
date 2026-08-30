// Package bootstrap implements the bootstrap use case for installing package managers.
// It orchestrates reading configuration, detecting existing managers, and installing missing ones.
package bootstrap

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/dto"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	"gopkg.in/yaml.v3"
)

const managerIDLogKey = "manager_id"

// UseCase implements the bootstrap use case.
type UseCase struct {
	managerRepo manager.Repository
	logger      output.Logger
}

// NewUseCase creates a new bootstrap use case.
func NewUseCase(
	managerRepo manager.Repository,
	logger output.Logger,
) *UseCase {
	return &UseCase{
		managerRepo: managerRepo,
		logger:      logger,
	}
}

// Bootstrap performs bootstrap operations to install and configure package managers.
func (uc *UseCase) Bootstrap(ctx context.Context, req *dto.BootstrapRequest) (*dto.BootstrapResponse, error) {
	startTime := time.Now()

	uc.logger.Info(
		ctx, "Starting package manager bootstrap",
		output.Field{Key: "config_path", Value: req.ConfigPath},
		output.Field{Key: "interactive", Value: req.Interactive},
		output.Field{Key: "dry_run", Value: req.DryRun},
	)

	// Load configuration
	var config *dto.BootstrapConfig
	var err error

	switch {
	case req.Interactive:
		// Interactive mode: use minimal default config
		config = uc.generateDefaultConfig()
		uc.logger.Info(ctx, "Using interactive mode with default configuration")
	case req.ConfigPath != "":
		// Load from file
		config, err = uc.loadConfig(req.ConfigPath)
		if err != nil {
			uc.logger.Error(ctx, "Failed to load config file", err, output.Field{Key: "path", Value: req.ConfigPath})
			return nil, fmt.Errorf("failed to load config from %s: %w", req.ConfigPath, err)
		}
	default:
		return nil, fmt.Errorf("either --config or --interactive must be specified")
	}

	// Get currently installed managers
	installedManagers, err := uc.managerRepo.FindAll(ctx)
	if err != nil {
		uc.logger.Error(ctx, "Failed to find installed managers", err)
		return nil, fmt.Errorf("failed to check installed managers: %w", err)
	}

	// Create a map for quick lookup
	installedMap := make(map[string]bool)
	for _, mgr := range installedManagers {
		if mgr.Installed {
			installedMap[string(mgr.ID)] = true
		}
	}

	// Process each manager in config
	results := make([]*dto.ManagerBootstrapResult, 0, len(config.Managers))
	summary := &dto.BootstrapSummary{
		TotalManagers: len(config.Managers),
	}

	for _, mgrConfig := range config.Managers {
		if !mgrConfig.Enabled {
			uc.logger.Debug(ctx, "Manager disabled in config, skipping", output.Field{Key: managerIDLogKey, Value: mgrConfig.ID})
			result := &dto.ManagerBootstrapResult{
				ID:         manager.ManagerID(mgrConfig.ID),
				Name:       mgrConfig.ID,
				Skipped:    true,
				SkipReason: "disabled in configuration",
			}
			results = append(results, result)
			summary.SkippedManagers++
			continue
		}

		uc.logger.Info(ctx, "Processing manager", output.Field{Key: managerIDLogKey, Value: mgrConfig.ID})

		result := uc.bootstrapManager(ctx, mgrConfig, installedMap, req.DryRun, config.Preferences)
		results = append(results, result)

		// Update summary
		if result.AlreadyInstalled {
			summary.AlreadyInstalledManagers++
		} else if result.Skipped {
			summary.SkippedManagers++
		} else if result.Success {
			summary.InstalledManagers++
		} else {
			summary.FailedManagers++
			if config.Preferences.FailOnError {
				uc.logger.Warn(ctx, "Bootstrap failed, stopping due to fail-on-error policy")
				break
			}
		}
	}

	summary.TotalDuration = time.Since(startTime).Seconds()

	uc.logger.Info(
		ctx, "Bootstrap completed",
		output.Field{Key: "installed", Value: summary.InstalledManagers},
		output.Field{Key: "failed", Value: summary.FailedManagers},
		output.Field{Key: "already_installed", Value: summary.AlreadyInstalledManagers},
		output.Field{Key: "skipped", Value: summary.SkippedManagers},
		output.Field{Key: "duration_seconds", Value: summary.TotalDuration},
	)

	return &dto.BootstrapResponse{
		Results: results,
		Summary: summary,
		DryRun:  req.DryRun,
	}, nil
}

// bootstrapManager handles the bootstrap process for a single manager.
func (uc *UseCase) bootstrapManager(
	ctx context.Context,
	mgrConfig dto.ManagerConfig,
	installedMap map[string]bool,
	dryRun bool,
	prefs dto.PreferencesConfig,
) *dto.ManagerBootstrapResult {
	startTime := time.Now()

	result := &dto.ManagerBootstrapResult{
		ID:    manager.ManagerID(mgrConfig.ID),
		Name:  mgrConfig.ID,
		Steps: []string{},
	}

	// Check if already installed
	if installedMap[mgrConfig.ID] {
		result.AlreadyInstalled = true
		if prefs.SkipAlreadyInstalled {
			result.Skipped = true
			result.SkipReason = "already installed"
			uc.logger.Info(ctx, "Manager already installed, skipping", output.Field{Key: managerIDLogKey, Value: mgrConfig.ID})
			return result
		}
		result.Success = true
		result.Steps = append(result.Steps, "Manager already installed")
		uc.logger.Info(ctx, "Manager already installed", output.Field{Key: managerIDLogKey, Value: mgrConfig.ID})
		result.Duration = time.Since(startTime).Seconds()
		return result
	}

	// Manager not installed - attempt installation
	if dryRun {
		result.Success = true
		result.Steps = append(result.Steps, fmt.Sprintf("Would install %s", mgrConfig.ID))
		uc.logger.Info(ctx, "Dry-run: would install manager", output.Field{Key: managerIDLogKey, Value: mgrConfig.ID})
	} else {
		// NOTE: For MVP, we're not implementing actual installation
		// This would require platform-specific installation scripts
		result.Success = false
		result.Error = "automatic installation not yet implemented for this manager"
		result.Steps = append(result.Steps, "Installation not implemented")
		uc.logger.Warn(ctx, "Manager installation not yet implemented", output.Field{Key: managerIDLogKey, Value: mgrConfig.ID})
	}

	result.Duration = time.Since(startTime).Seconds()
	return result
}

// loadConfig loads bootstrap configuration from a YAML file.
func (uc *UseCase) loadConfig(path string) (*dto.BootstrapConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var config dto.BootstrapConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &config, nil
}

// generateDefaultConfig creates a default configuration for interactive mode.
func (uc *UseCase) generateDefaultConfig() *dto.BootstrapConfig {
	return &dto.BootstrapConfig{
		Managers: []dto.ManagerConfig{
			{ID: "brew", Enabled: true},
			{ID: "asdf", Enabled: true},
			{ID: "npm", Enabled: true},
		},
		Preferences: dto.PreferencesConfig{
			SkipAlreadyInstalled: true,
			FailOnError:          false,
		},
	}
}
