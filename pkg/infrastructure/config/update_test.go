package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/dto"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
)

func TestLoadAndResolveUpdatePreferences(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	data := "version: 1\ndefaults:\n  strategy: stable\nmanagers:\n  mise:\n    strategy: micro\n    bump: true\n    directory: /workspace\n    local: true\n  brew:\n    strategy: fixed\n"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path, true)
	if err != nil {
		t.Fatal(err)
	}
	mise := cfg.Resolve(manager.ManagerMise)
	if mise.Strategy != dto.StrategyMicro || !mise.Bump || mise.MiseDir != "/workspace" || !mise.MiseLocal {
		t.Fatalf("mise policy = %#v", mise)
	}
	if cfg.Resolve(manager.ManagerHomebrew).Strategy != dto.StrategyFixed || cfg.Resolve(manager.ManagerNPM).Strategy != dto.StrategyStable {
		t.Fatal("manager overrides must preserve defaults for other managers")
	}
}

func TestLoadRejectsInvalidOrAmbiguousPreferences(t *testing.T) {
	tests := []string{
		"version: 2\n",
		"version: 1\nunknown: true\n",
		"version: 1\ndefaults:\n  strategy: typo\n",
		"version: 1\ndefaults:\n  bump: true\n",
		"version: 1\nmanagers:\n  typo:\n    strategy: fixed\n",
		"version: 1\nmanagers:\n  npm:\n    bump: false\n",
		"version: 1\n---\nversion: 1\n",
		"version: 1\ndefaults:\n  local: true\n",
		"version: 1\nversion: 1\n",
		"version: 1\nmanagers:\n  mise:\n    tools: []\n",
	}
	for _, data := range tests {
		t.Run(data, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(path, true); err == nil {
				t.Fatal("invalid configuration was accepted")
			}
		})
	}
}

func TestMissingConfigAndXDGPath(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", base)
	path, err := DefaultPath()
	if err != nil || path != filepath.Join(base, "gz-pm", "config.yaml") {
		t.Fatalf("path=%q err=%v", path, err)
	}
	cfg, err := Load(path, false)
	if err != nil || cfg.Resolve(manager.ManagerMise).Strategy != dto.StrategyStable {
		t.Fatalf("implicit missing config: %#v %v", cfg, err)
	}
	if _, err := Load(path, true); err == nil {
		t.Fatal("explicit missing config must fail")
	}
}
