package cmdutil

import (
	"errors"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
)

func TestCheckResult(t *testing.T) {
	tests := []struct {
		name      string
		result    *output.ExecutionResult
		err       error
		operation string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "success",
			result:    &output.ExecutionResult{ExitCode: 0},
			err:       nil,
			operation: "test operation",
			wantErr:   false,
		},
		{
			name:      "executor error",
			result:    nil,
			err:       errors.New("command not found"),
			operation: "test operation",
			wantErr:   true,
			errMsg:    "test operation: command not found",
		},
		{
			name:      "non-zero exit code with stderr",
			result:    &output.ExecutionResult{ExitCode: 1, Stderr: "permission denied"},
			err:       nil,
			operation: "test operation",
			wantErr:   true,
			errMsg:    "test operation failed: permission denied",
		},
		{
			name:      "non-zero exit code without stderr",
			result:    &output.ExecutionResult{ExitCode: 127, Stderr: ""},
			err:       nil,
			operation: "test operation",
			wantErr:   true,
			errMsg:    "test operation failed with exit code 127",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckResult(tt.result, tt.err, tt.operation)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckResult() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("CheckResult() error = %q, want %q", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestExtractStdout(t *testing.T) {
	tests := []struct {
		name   string
		result *output.ExecutionResult
		want   string
	}{
		{
			name:   "normal output",
			result: &output.ExecutionResult{Stdout: "hello world\n"},
			want:   "hello world",
		},
		{
			name:   "output with whitespace",
			result: &output.ExecutionResult{Stdout: "  hello  \n"},
			want:   "hello",
		},
		{
			name:   "empty output",
			result: &output.ExecutionResult{Stdout: ""},
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractStdout(tt.result)
			if got != tt.want {
				t.Errorf("ExtractStdout() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name      string
		result    *output.ExecutionResult
		operation string
		wantErr   bool
	}{
		{
			name:      "valid JSON",
			result:    &output.ExecutionResult{Stdout: `{"name": "test", "version": "1.0"}`},
			operation: "parse package",
			wantErr:   false,
		},
		{
			name:      "invalid JSON",
			result:    &output.ExecutionResult{Stdout: "not valid json"},
			operation: "parse package",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var v map[string]string
			err := UnmarshalJSON(tt.result, &v, tt.operation)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExtractVersionField(t *testing.T) {
	tests := []struct {
		name       string
		stdout     string
		fieldIndex int
		operation  string
		want       string
		wantErr    bool
	}{
		{
			name:       "normal version output",
			stdout:     "pip 23.0.1 from /usr/lib/python3.11/site-packages/pip",
			fieldIndex: 1,
			operation:  "get pip version",
			want:       "23.0.1",
			wantErr:    false,
		},
		{
			name:       "Homebrew version",
			stdout:     "Homebrew 4.2.1\n",
			fieldIndex: 1,
			operation:  "get brew version",
			want:       "4.2.1",
			wantErr:    false,
		},
		{
			name:       "insufficient fields",
			stdout:     "pip",
			fieldIndex: 1,
			operation:  "get pip version",
			want:       "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractVersionField(tt.stdout, tt.fieldIndex, tt.operation)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractVersionField() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExtractVersionField() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsCommandAvailable(t *testing.T) {
	tests := []struct {
		name   string
		result *output.ExecutionResult
		err    error
		want   bool
	}{
		{
			name:   "command found",
			result: &output.ExecutionResult{ExitCode: 0, Stdout: "/usr/bin/brew\n"},
			err:    nil,
			want:   true,
		},
		{
			name:   "command not found - exit code",
			result: &output.ExecutionResult{ExitCode: 1, Stdout: ""},
			err:    nil,
			want:   false,
		},
		{
			name:   "command not found - empty stdout",
			result: &output.ExecutionResult{ExitCode: 0, Stdout: ""},
			err:    nil,
			want:   false,
		},
		{
			name:   "executor error",
			result: nil,
			err:    errors.New("execution failed"),
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsCommandAvailable(tt.result, tt.err)
			if got != tt.want {
				t.Errorf("IsCommandAvailable() = %v, want %v", got, tt.want)
			}
		})
	}
}
