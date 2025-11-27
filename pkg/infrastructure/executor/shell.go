package executor

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
)

// ShellExecutor executes commands using the system shell.
type ShellExecutor struct {
	logger output.Logger
}

// NewShellExecutor creates a new shell executor.
func NewShellExecutor(logger output.Logger) *ShellExecutor {
	return &ShellExecutor{
		logger: logger,
	}
}

// Execute runs a command and returns the result.
func (e *ShellExecutor) Execute(ctx context.Context, command string, args ...string) (*output.ExecutionResult, error) {
	e.logger.Debug(ctx, "Executing command",
		output.Field{Key: "command", Value: command},
		output.Field{Key: "args", Value: args},
	)

	cmd := exec.CommandContext(ctx, command, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	exitCode := -1
	success := false
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
		success = cmd.ProcessState.Success()
	}

	result := &output.ExecutionResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
		Success:  success,
	}

	if err != nil {
		e.logger.Debug(ctx, "Command execution failed",
			output.Field{Key: "command", Value: command},
			output.Field{Key: "exit_code", Value: result.ExitCode},
			output.Field{Key: "stderr", Value: result.Stderr},
		)
		return result, fmt.Errorf("command execution failed: %w", err)
	}

	e.logger.Debug(ctx, "Command executed successfully",
		output.Field{Key: "command", Value: command},
		output.Field{Key: "exit_code", Value: result.ExitCode},
	)

	return result, nil
}

// ExecuteWithInput runs a command with stdin input.
func (e *ShellExecutor) ExecuteWithInput(ctx context.Context, input string, command string, args ...string) (*output.ExecutionResult, error) {
	e.logger.Debug(ctx, "Executing command with input",
		output.Field{Key: "command", Value: command},
		output.Field{Key: "args", Value: args},
	)

	cmd := exec.CommandContext(ctx, command, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = bytes.NewBufferString(input)

	err := cmd.Run()

	exitCode := -1
	success := false
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
		success = cmd.ProcessState.Success()
	}

	result := &output.ExecutionResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
		Success:  success,
	}

	if err != nil {
		e.logger.Debug(ctx, "Command execution with input failed",
			output.Field{Key: "command", Value: command},
			output.Field{Key: "exit_code", Value: result.ExitCode},
		)
		return result, fmt.Errorf("command execution with input failed: %w", err)
	}

	e.logger.Debug(ctx, "Command with input executed successfully",
		output.Field{Key: "command", Value: command},
	)

	return result, nil
}
