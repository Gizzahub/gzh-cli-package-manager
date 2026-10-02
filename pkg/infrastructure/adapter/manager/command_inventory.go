package manager

import (
	"context"
	"fmt"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/diagnostics"
)

// CommandInventory is an optional capability for managers that can report
// which commands they provide. A manager that does not implement it
// contributes no commands.
type CommandInventory interface {
	// Commands returns one install record per install of a command this
	// manager provides. Implementations may query their package manager
	// read-only; they must not install, update, or remove anything.
	Commands(ctx context.Context) ([]diagnostics.InstallRecord, error)
}

// CollectCommands returns the command records mgr contributes. When mgr
// implements CommandInventory its records are returned unchanged; otherwise
// the result is an empty slice with a nil error. The collector never calls
// ListPackages or Update and never reaches manager.Repository.
func CollectCommands(ctx context.Context, mgr Adapter) ([]diagnostics.InstallRecord, error) {
	if inventory, ok := mgr.(CommandInventory); ok {
		records, err := inventory.Commands(ctx)
		if err != nil {
			return nil, fmt.Errorf("collect commands: %w", err)
		}
		return records, nil
	}
	return []diagnostics.InstallRecord{}, nil
}
