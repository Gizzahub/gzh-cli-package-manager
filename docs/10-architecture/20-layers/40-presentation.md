# 2.4 Presentation Layer (cmd/gz-pm)

> gzh-cli-package-manager 레이어 아키텍처 · [레이어 인덱스](README.md) · [아키텍처 인덱스](../README.md) · [ARCHITECTURE.md](../../../ARCHITECTURE.md)

**Responsibility**: User interface (CLI)

**Dependencies**:
- Application layer (use cases)
- Infrastructure layer (for dependency injection)
- Cobra framework

**Components**:

```go
cmd/gz-pm/
├── main.go                      # Entry point, DI setup
├── command/                     # Cobra commands
│   ├── root.go                 # Root command
│   ├── update.go               # Update command
│   ├── status.go               # Status command
│   ├── bootstrap.go            # Bootstrap command
│   ├── sync.go                 # Sync command
│   ├── export.go               # Export command
│   └── cache.go                # Cache command
├── formatter/                   # Output formatters
│   ├── text_formatter.go       # Enhanced text output
│   ├── json_formatter.go       # JSON output
│   ├── table_formatter.go      # Table output
│   └── formatter_test.go
└── validator/                   # Input validation
    ├── flag_validator.go
    └── validator_test.go
```

**Example Command**:

```go
// cmd/gz-pm/command/update.go
package command

import (
    "github.com/spf13/cobra"
    "github.com/gizzahub/gzh-cli-package-manager/pkg/application/update"
)

func NewUpdateCommand(updateUC *update.UpdateAllManagersUseCase) *cobra.Command {
    cmd := &cobra.Command{
        Use:   "update",
        Short: "Update package managers and packages",
        Long:  `Update all or specific package managers and their packages.`,
        RunE: func(cmd *cobra.Command, args []string) error {
            // 1. Parse flags
            all, _ := cmd.Flags().GetBool("all")
            dryRun, _ := cmd.Flags().GetBool("dry-run")
            strategy, _ := cmd.Flags().GetString("strategy")
            outputFormat, _ := cmd.Flags().GetString("output")

            // 2. Build request DTO
            req := &dto.UpdateAllRequest{
                All:      all,
                DryRun:   dryRun,
                Strategy: parseStrategy(strategy),
            }

            // 3. Execute use case
            resp, err := updateUC.Execute(cmd.Context(), req)
            if err != nil {
                return err
            }

            // 4. Format output
            formatter := getFormatter(outputFormat)
            return formatter.FormatUpdateResponse(resp)
        },
    }

    // Flags
    cmd.Flags().BoolP("all", "a", false, "Update all managers")
    cmd.Flags().Bool("dry-run", false, "Preview changes without executing")
    cmd.Flags().String("strategy", "stable", "Update strategy (latest|stable|minor|fixed)")
    cmd.Flags().StringP("output", "o", "text", "Output format (text|json)")

    return cmd
}
```

**Dependency Injection** (main.go):

```go
// cmd/gz-pm/main.go
package main

import (
    "github.com/spf13/cobra"
    // ... imports
)

func main() {
    // 1. Initialize infrastructure
    logger := logger.NewStructuredLogger("gz-pm")
    executor := executor.NewShellExecutor(logger)

    // 2. Initialize repositories
    configRepo := yaml.NewConfigRepository("~/.config/gz-pm", logger)
    managerRepo := repository.NewManagerRepository(executor, logger)

    // 3. Initialize adapters
    homebrewAdapter := homebrew.NewAdapter(executor, logger)
    asdfAdapter := asdf.NewAdapter(executor, logger)
    // ... more adapters

    // Register adapters with factory
    adapterFactory := adapter.NewFactory()
    adapterFactory.Register("brew", homebrewAdapter)
    adapterFactory.Register("asdf", asdfAdapter)

    // 4. Initialize use cases
    updateUC := update.NewUpdateAllManagersUseCase(
        managerRepo,
        executor,
        logger,
        update.NewStableStrategy(),
    )

    bootstrapUC := bootstrap.NewInstallManagerUseCase(
        managerRepo,
        executor,
        logger,
    )

    // 5. Build CLI
    rootCmd := command.NewRootCommand()
    rootCmd.AddCommand(command.NewUpdateCommand(updateUC))
    rootCmd.AddCommand(command.NewBootstrapCommand(bootstrapUC))
    // ... more commands

    // 6. Execute
    if err := rootCmd.Execute(); err != nil {
        logger.Fatal("Command failed", err)
        os.Exit(1)
    }
}
```
