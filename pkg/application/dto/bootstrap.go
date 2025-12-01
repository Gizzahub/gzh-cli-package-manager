package dto

import "github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"

// BootstrapRequest contains parameters for bootstrapping package managers.
type BootstrapRequest struct {
	// ConfigPath is the path to the bootstrap configuration file.
	ConfigPath string

	// Interactive enables interactive setup wizard.
	Interactive bool

	// DryRun indicates preview mode without executing installations.
	DryRun bool
}

// BootstrapConfig represents the bootstrap configuration file structure.
type BootstrapConfig struct {
	// Managers lists package managers to install.
	Managers []ManagerConfig `yaml:"managers"`

	// Preferences contains global preferences.
	Preferences PreferencesConfig `yaml:"preferences,omitempty"`
}

// ManagerConfig specifies a package manager to install.
type ManagerConfig struct {
	// ID is the manager identifier (e.g., "brew", "asdf").
	ID string `yaml:"id"`

	// Enabled indicates whether to install this manager.
	Enabled bool `yaml:"enabled"`

	// Version specifies the version to install (optional).
	Version string `yaml:"version,omitempty"`

	// ConfigOptions contains manager-specific configuration.
	ConfigOptions map[string]interface{} `yaml:"config,omitempty"`
}

// PreferencesConfig contains global bootstrap preferences.
type PreferencesConfig struct {
	// SkipAlreadyInstalled skips managers that are already installed.
	SkipAlreadyInstalled bool `yaml:"skip_already_installed"`

	// FailOnError stops bootstrap if any installation fails.
	FailOnError bool `yaml:"fail_on_error"`
}

// BootstrapResponse contains the results of bootstrap operations.
type BootstrapResponse struct {
	// Results contains the bootstrap result for each manager.
	Results []*ManagerBootstrapResult

	// Summary provides aggregate statistics.
	Summary *BootstrapSummary

	// DryRun indicates if this was a dry-run (preview only).
	DryRun bool
}

// ManagerBootstrapResult represents the bootstrap result for a single manager.
type ManagerBootstrapResult struct {
	// Manager ID and name.
	ID   manager.ManagerID
	Name string

	// AlreadyInstalled indicates the manager was already present.
	AlreadyInstalled bool

	// Success indicates if the installation/configuration completed successfully.
	Success bool

	// Error contains the error message if bootstrap failed.
	Error string

	// Skipped indicates the manager was skipped (e.g., platform mismatch).
	Skipped bool

	// SkipReason explains why the manager was skipped.
	SkipReason string

	// Duration is the time taken for the bootstrap in seconds.
	Duration float64

	// Steps contains the installation steps performed.
	Steps []string
}

// BootstrapSummary provides aggregate statistics for bootstrap operations.
type BootstrapSummary struct {
	// TotalManagers is the total number of managers in config.
	TotalManagers int

	// InstalledManagers is the number of managers successfully installed.
	InstalledManagers int

	// SkippedManagers is the number of managers skipped.
	SkippedManagers int

	// FailedManagers is the number of managers that failed to install.
	FailedManagers int

	// AlreadyInstalledManagers is the number of managers that were already installed.
	AlreadyInstalledManagers int

	// TotalDuration is the total time taken in seconds.
	TotalDuration float64
}
