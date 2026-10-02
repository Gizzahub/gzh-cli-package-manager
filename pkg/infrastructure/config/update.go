// Package config loads persisted gz-pm update preferences.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/dto"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
)

// Policy uses pointers to distinguish omitted values from explicit false.
type Policy struct {
	Strategy  dto.UpdateStrategy `yaml:"strategy"`
	Bump      *bool              `yaml:"bump"`
	Directory *string            `yaml:"directory"`
	Local     *bool              `yaml:"local"`
	Tools     []string           `yaml:"tools"`
}

// UpdateConfig stores defaults and per-manager update policies.
type UpdateConfig struct {
	Version  int                          `yaml:"version"`
	Defaults Policy                       `yaml:"defaults"`
	Managers map[manager.ManagerID]Policy `yaml:"managers"`
}

// DefaultPath follows XDG_CONFIG_HOME, including on macOS.
func DefaultPath() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("locate gz-pm configuration: %w", err)
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "gz-pm", "config.yaml"), nil
}

// Load reads strict YAML. Only a missing implicit default file is optional.
func Load(path string, explicit bool) (*UpdateConfig, error) {
	// #nosec G304 -- The CLI-selected or XDG config path is intentionally user-controlled; contents are decoded as strict data-only YAML, never executed.
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && !explicit {
			return &UpdateConfig{}, nil
		}
		return nil, fmt.Errorf("read update config %s: %w", path, err)
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var cfg UpdateConfig
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode update config %s: %w", path, err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("update config must contain one YAML document")
	}
	if cfg.Version != 1 {
		return nil, fmt.Errorf("unsupported update config version %d (expected 1)", cfg.Version)
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (cfg *UpdateConfig) validate() error {
	if err := dto.ValidateUpdateStrategy(cfg.Defaults.Strategy); err != nil {
		return fmt.Errorf("update config defaults: %w", err)
	}
	if cfg.Defaults.Bump != nil && *cfg.Defaults.Bump {
		return fmt.Errorf("set bump in managers.mise, not in global defaults")
	}
	if cfg.Defaults.Directory != nil || cfg.Defaults.Local != nil || cfg.Defaults.Tools != nil {
		return fmt.Errorf("directory, local and tools belong in managers.mise")
	}
	ids := make([]string, 0, len(cfg.Managers))
	for id := range cfg.Managers {
		ids = append(ids, string(id))
	}
	sort.Strings(ids)
	for _, key := range ids {
		id := manager.ManagerID(key)
		policy := cfg.Managers[id]
		if err := validateManagerPolicy(id, policy); err != nil {
			return err
		}
	}
	return nil
}

func validateManagerPolicy(id manager.ManagerID, policy Policy) error {
	if policy.Tools != nil && len(policy.Tools) == 0 {
		return fmt.Errorf("%s: tools selection must not be empty; omit tools to select all", id)
	}
	if !knownManager(id) {
		return fmt.Errorf("unknown manager %q in update config", id)
	}
	if err := dto.ValidateUpdateStrategy(policy.Strategy); err != nil {
		return fmt.Errorf("update config manager %s: %w", id, err)
	}
	if id != manager.ManagerMise && (policy.Bump != nil || policy.Directory != nil || policy.Local != nil || policy.Tools != nil) {
		return fmt.Errorf("%s: bump, directory, local and tools are mise-only settings", id)
	}
	return nil
}

func knownManager(id manager.ManagerID) bool {
	switch id {
	case manager.ManagerHomebrew, manager.ManagerASDF, manager.ManagerMise, manager.ManagerNPM,
		manager.ManagerPip, manager.ManagerCargo, manager.ManagerApt, manager.ManagerPacman,
		manager.ManagerWinget, manager.ManagerScoop, manager.ManagerChocolatey:
		return true
	default:
		return false
	}
}

// Resolve applies defaults followed by the requested manager's overrides.
func (cfg *UpdateConfig) Resolve(id manager.ManagerID) dto.UpdatePolicy {
	policy := dto.UpdatePolicy{Strategy: dto.StrategyStable}
	apply(&policy, cfg.Defaults)
	apply(&policy, cfg.Managers[id])
	return policy
}

func apply(target *dto.UpdatePolicy, source Policy) {
	if source.Strategy != "" {
		target.Strategy = source.Strategy
	}
	if source.Bump != nil {
		target.Bump = *source.Bump
	}
	if source.Directory != nil {
		target.MiseDir = *source.Directory
	}
	if source.Local != nil {
		target.MiseLocal = *source.Local
	}
	if source.Tools != nil {
		target.MiseTools = source.Tools
	}
}
