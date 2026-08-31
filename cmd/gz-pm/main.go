package main

import (
	"github.com/gizzahub/gzh-cli-package-manager/cmd/gz-pm/command"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/usecase/bootstrap"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/usecase/status"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/usecase/update"
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
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/detector"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/executor"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/logger"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/repository/memory"
)

func main() {
	// Initialize infrastructure layer (outer layer)
	log := logger.NewStructuredLogger("gz-pm")

	// Initialize command executor
	exec := executor.NewShellExecutor(log)

	// Initialize repository with real detection
	managerRepo := memory.NewDetectingManagerRepository(exec, log)

	// Initialize adapters for each package manager
	adapters := newManagerAdapters(exec, log)

	// Initialize environment detector
	envDetector := detector.NewDetector(exec, log)

	// Initialize use cases (application layer)
	statusUC := status.NewUseCase(managerRepo, log)
	updateUC := update.NewUseCase(managerRepo, log, adapters, envDetector)
	bootstrapUC := bootstrap.NewUseCase(managerRepo, log)

	// Inject dependencies into commands (presentation layer)
	command.SetStatusUseCase(statusUC)
	command.SetUpdateUseCase(updateUC)
	command.SetBootstrapUseCase(bootstrapUC)
	command.SetManagerAdapters(adapters)

	// Execute CLI
	command.Execute()
}

// newManagerAdapters constructs the complete adapter registry used by the CLI.
// Keeping registration in one factory makes omissions visible in a focused test.
func newManagerAdapters(exec output.CommandExecutor, log output.Logger) map[manager.ManagerID]adapterm.Adapter {
	return map[manager.ManagerID]adapterm.Adapter{
		manager.ManagerApt:        apt.NewAdapter(exec, log),
		manager.ManagerHomebrew:   homebrew.NewAdapter(exec, log),
		manager.ManagerASDF:       asdf.NewAdapter(exec, log),
		manager.ManagerNPM:        npm.NewAdapter(exec, log),
		manager.ManagerCargo:      cargo.NewAdapter(exec, log),
		manager.ManagerPip:        pip.NewAdapter(exec, log),
		manager.ManagerWinget:     winget.NewAdapter(exec, log),
		manager.ManagerScoop:      scoop.NewAdapter(exec, log),
		manager.ManagerChocolatey: chocolatey.NewAdapter(exec, log),
		manager.ManagerPacman:     pacman.NewAdapter(exec, log),
	}
}
