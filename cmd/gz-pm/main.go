package main

import (
	"github.com/gizzahub/gzh-cli-package-manager/cmd/gz-pm/command"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/usecase/bootstrap"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/usecase/status"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/usecase/update"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/registry"
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

// newManagerAdapters delegates to the infrastructure registry so the CLI and
// detecting repository share one supported-manager constructor set.
func newManagerAdapters(exec output.CommandExecutor, log output.Logger) map[manager.ManagerID]adapterm.Adapter {
	return registry.New(exec, log)
}
