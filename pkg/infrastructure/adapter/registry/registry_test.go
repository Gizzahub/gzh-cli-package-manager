package registry

import (
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
)

func TestNewRegistersOnlySupportedManagers(t *testing.T) {
	adapters := New(nil, nil)
	expected := map[manager.ManagerID]struct{}{
		manager.ManagerApt:        {},
		manager.ManagerHomebrew:   {},
		manager.ManagerPacman:     {},
		manager.ManagerNPM:        {},
		manager.ManagerPip:        {},
		manager.ManagerCargo:      {},
		manager.ManagerASDF:       {},
		manager.ManagerWinget:     {},
		manager.ManagerScoop:      {},
		manager.ManagerChocolatey: {},
	}

	if len(adapters) != len(expected) {
		t.Fatalf("registry has %d adapters, want %d", len(adapters), len(expected))
	}
	for id := range expected {
		if adapters[id] == nil {
			t.Errorf("registry is missing non-nil adapter for %q", id)
		}
	}
	if adapters[manager.ManagerSDKMan] != nil {
		t.Error("sdkman must remain unsupported")
	}
	if adapters[manager.ManagerYay] != nil {
		t.Error("yay must remain unsupported")
	}
}
