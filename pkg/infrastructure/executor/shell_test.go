// Package executor provides command execution infrastructure.
package executor

import (
	"context"
	"runtime"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
)

// mockLogger implements output.Logger for testing.
type mockLogger struct{}

func (m *mockLogger) Debug(_ context.Context, _ string, _ ...output.Field)          {}
func (m *mockLogger) Info(_ context.Context, _ string, _ ...output.Field)           {}
func (m *mockLogger) Warn(_ context.Context, _ string, _ ...output.Field)           {}
func (m *mockLogger) Error(_ context.Context, _ string, _ error, _ ...output.Field) {}

func TestShellExecutor_Execute(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		args        []string
		wantSuccess bool
		wantErr     bool
	}{
		{
			name:        "successful echo command",
			command:     "echo",
			args:        []string{"hello"},
			wantSuccess: true,
			wantErr:     false,
		},
		{
			name:        "command not found",
			command:     "nonexistentcommand12345",
			args:        []string{},
			wantSuccess: false,
			wantErr:     true,
		},
		{
			name:        "false command (exit code 1)",
			command:     getFalseCommand(),
			args:        []string{},
			wantSuccess: false,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewShellExecutor(&mockLogger{})
			result, err := executor.Execute(context.Background(), tt.command, tt.args...)

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if result == nil {
				t.Error("Execute() result is nil")
				return
			}

			if result.Success != tt.wantSuccess {
				t.Errorf("Execute() success = %v, want %v", result.Success, tt.wantSuccess)
			}

			if tt.wantSuccess && result.ExitCode != 0 {
				t.Errorf("Execute() exit code = %d, want 0", result.ExitCode)
			}
		})
	}
}

func TestShellExecutor_ExecuteWithInput(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		command     string
		args        []string
		wantSuccess bool
		wantErr     bool
	}{
		{
			name:        "cat with input",
			input:       "test input",
			command:     "cat",
			args:        []string{},
			wantSuccess: true,
			wantErr:     false,
		},
		{
			name:        "command not found with input",
			input:       "test",
			command:     "nonexistentcommand12345",
			args:        []string{},
			wantSuccess: false,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewShellExecutor(&mockLogger{})
			result, err := executor.ExecuteWithInput(context.Background(), tt.input, tt.command, tt.args...)

			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteWithInput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if result == nil {
				t.Error("ExecuteWithInput() result is nil")
				return
			}

			if result.Success != tt.wantSuccess {
				t.Errorf("ExecuteWithInput() success = %v, want %v", result.Success, tt.wantSuccess)
			}

			if tt.wantSuccess && tt.name == "cat with input" {
				if result.Stdout != tt.input {
					t.Errorf("ExecuteWithInput() stdout = %q, want %q", result.Stdout, tt.input)
				}
			}
		})
	}
}

func TestShellExecutor_ExecutionResult(t *testing.T) {
	executor := NewShellExecutor(&mockLogger{})

	// Test successful command with output
	result, err := executor.Execute(context.Background(), "echo", "test output")
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	if result.Stdout == "" {
		t.Error("Execute() stdout is empty, expected output")
	}

	if result.ExitCode != 0 {
		t.Errorf("Execute() exit code = %d, want 0", result.ExitCode)
	}

	if !result.Success {
		t.Error("Execute() success = false, want true")
	}
}

func TestShellExecutor_ContextCancellation(t *testing.T) {
	executor := NewShellExecutor(&mockLogger{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Command should fail due to cancelled context
	_, err := executor.Execute(ctx, "sleep", "10")
	if err == nil {
		t.Error("Execute() with cancelled context should return error")
	}
}

// getFalseCommand returns the appropriate false command for the OS.
func getFalseCommand() string {
	if runtime.GOOS == "windows" {
		return "cmd"
	}
	return "false"
}
