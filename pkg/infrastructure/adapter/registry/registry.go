// Package registry constructs the supported runtime manager adapters.
package registry

import (
	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/apt"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/asdf"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/cargo"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/chocolatey"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/homebrew"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/npm"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/pacman"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/pip"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/scoop"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/winget"
)

// New constructs adapters for every manager currently supported at runtime.
// sdkman and yay intentionally remain unsupported until adapters are implemented.
func New(executor output.CommandExecutor, logger output.Logger) map[manager.ManagerID]adapterm.Adapter {
	return map[manager.ManagerID]adapterm.Adapter{
		manager.ManagerApt:        apt.NewAdapter(executor, logger),
		manager.ManagerHomebrew:   homebrew.NewAdapter(executor, logger),
		manager.ManagerASDF:       asdf.NewAdapter(executor, logger),
		manager.ManagerNPM:        npm.NewAdapter(executor, logger),
		manager.ManagerCargo:      cargo.NewAdapter(executor, logger),
		manager.ManagerPip:        pip.NewAdapter(executor, logger),
		manager.ManagerWinget:     winget.NewAdapter(executor, logger),
		manager.ManagerScoop:      scoop.NewAdapter(executor, logger),
		manager.ManagerChocolatey: chocolatey.NewAdapter(executor, logger),
		manager.ManagerPacman:     pacman.NewAdapter(executor, logger),
	}
}
