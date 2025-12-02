// Package testutil provides shared mock implementations for adapter tests.
package testutil

import (
	"context"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
)

// ExecutorFunc defines the function signature for mock executor callbacks.
type ExecutorFunc func(ctx context.Context, command string, args ...string) (*output.ExecutionResult, error)

// MockExecutor implements output.CommandExecutor for testing.
type MockExecutor struct {
	ExecFunc ExecutorFunc
}

// Execute runs the mock executor function if set, otherwise returns a default result.
func (m *MockExecutor) Execute(ctx context.Context, command string, args ...string) (*output.ExecutionResult, error) {
	if m.ExecFunc != nil {
		return m.ExecFunc(ctx, command, args...)
	}
	return &output.ExecutionResult{
		Stdout:   "",
		Stderr:   "",
		ExitCode: 0,
	}, nil
}

// ExecuteWithInput returns a default successful result for testing.
func (m *MockExecutor) ExecuteWithInput(_ context.Context, _ string, _ string, _ ...string) (*output.ExecutionResult, error) {
	return &output.ExecutionResult{ExitCode: 0}, nil
}

// MockLogger implements output.Logger for testing.
type MockLogger struct{}

// Debug does nothing in tests.
func (m *MockLogger) Debug(_ context.Context, _ string, _ ...output.Field) {}

// Info does nothing in tests.
func (m *MockLogger) Info(_ context.Context, _ string, _ ...output.Field) {}

// Warn does nothing in tests.
func (m *MockLogger) Warn(_ context.Context, _ string, _ ...output.Field) {}

// Error does nothing in tests.
func (m *MockLogger) Error(_ context.Context, _ string, _ error, _ ...output.Field) {}

// NewMockExecutor creates a new MockExecutor with the given function.
func NewMockExecutor(fn ExecutorFunc) *MockExecutor {
	return &MockExecutor{ExecFunc: fn}
}

// NewMockLogger creates a new MockLogger.
func NewMockLogger() *MockLogger {
	return &MockLogger{}
}

// SuccessResult returns a successful execution result with the given stdout.
func SuccessResult(stdout string) *output.ExecutionResult {
	return &output.ExecutionResult{
		Stdout:   stdout,
		Stderr:   "",
		ExitCode: 0,
	}
}

// FailureResult returns a failed execution result with the given exit code.
func FailureResult(exitCode int, stderr string) *output.ExecutionResult {
	return &output.ExecutionResult{
		Stdout:   "",
		Stderr:   stderr,
		ExitCode: exitCode,
	}
}
