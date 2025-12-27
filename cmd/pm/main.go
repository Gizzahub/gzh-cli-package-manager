package main

import (
	"github.com/gizzahub/gzh-cli-package-manager/cmd/pm/command"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/usecase/bootstrap"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/usecase/status"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/usecase/update"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/asdf"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/cargo"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/chocolatey"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/homebrew"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/npm"
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
	adapters := map[manager.ManagerID]adapterm.Adapter{
		manager.ManagerHomebrew:   homebrew.NewAdapter(exec, log),
		manager.ManagerASDF:       asdf.NewAdapter(exec, log),
		manager.ManagerNPM:        npm.NewAdapter(exec, log),
		manager.ManagerCargo:      cargo.NewAdapter(exec, log),
		manager.ManagerPip:        pip.NewAdapter(exec, log),
		manager.ManagerWinget:     winget.NewAdapter(exec, log),
		manager.ManagerScoop:      scoop.NewAdapter(exec, log),
		manager.ManagerChocolatey: chocolatey.NewAdapter(exec, log),
		// TODO: Add more adapters as needed (apt, pacman, etc.)
	}

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

	// Execute CLI
	command.Execute()
}
