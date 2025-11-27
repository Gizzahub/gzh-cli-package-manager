package output

import "context"

// CommandExecutor executes system commands.
// This port is implemented by infrastructure layer.
type CommandExecutor interface {
	// Execute runs a command and returns the result.
	Execute(ctx context.Context, command string, args ...string) (*ExecutionResult, error)

	// ExecuteWithInput runs a command with stdin input.
	ExecuteWithInput(ctx context.Context, input string, command string, args ...string) (*ExecutionResult, error)
}

// ExecutionResult contains the output of a command execution.
type ExecutionResult struct {
	// Stdout is the standard output.
	Stdout string

	// Stderr is the standard error output.
	Stderr string

	// ExitCode is the process exit code.
	ExitCode int

	// Success indicates if the command succeeded (exit code 0).
	Success bool
}
