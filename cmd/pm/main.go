package main

import (
	"github.com/gizzahub/gzh-cli-package-manager/cmd/pm/command"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/usecase/status"
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

	// Initialize use cases (application layer)
	statusUC := status.NewUseCase(managerRepo, log)

	// Inject dependencies into commands (presentation layer)
	command.SetStatusUseCase(statusUC)

	// Execute CLI
	command.Execute()
}
