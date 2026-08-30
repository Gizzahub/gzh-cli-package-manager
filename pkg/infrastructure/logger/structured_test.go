// Package logger provides logging infrastructure.
package logger

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strings"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
)

const testLoggerPrefix = "test"

func TestNewStructuredLogger(t *testing.T) {
	logger := NewStructuredLogger("test-app")

	if logger == nil {
		t.Fatal("NewStructuredLogger() returned nil")
	}

	if logger.prefix != "test-app" {
		t.Errorf("prefix = %q, want %q", logger.prefix, "test-app")
	}

	if logger.logger == nil {
		t.Error("internal logger is nil")
	}
}

func TestStructuredLogger_Debug(t *testing.T) {
	var buf bytes.Buffer
	logger := &StructuredLogger{
		prefix: testLoggerPrefix,
		logger: log.New(&buf, "test: ", 0),
	}

	logger.Debug(context.Background(), "debug message")

	gotOutput := buf.String()
	if !strings.Contains(gotOutput, "[DEBUG]") {
		t.Errorf("output missing [DEBUG]: %q", gotOutput)
	}
	if !strings.Contains(gotOutput, "debug message") {
		t.Errorf("output missing message: %q", gotOutput)
	}
}

func TestStructuredLogger_DebugWithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := &StructuredLogger{
		prefix: testLoggerPrefix,
		logger: log.New(&buf, "test: ", 0),
	}

	logger.Debug(
		context.Background(), "debug with fields",
		output.Field{Key: "key1", Value: "value1"},
		output.Field{Key: "key2", Value: 42},
	)

	gotOutput := buf.String()
	if !strings.Contains(gotOutput, "[DEBUG]") {
		t.Errorf("output missing [DEBUG]: %q", gotOutput)
	}
	if !strings.Contains(gotOutput, "debug with fields") {
		t.Errorf("output missing message: %q", gotOutput)
	}
	if !strings.Contains(gotOutput, "key1=value1") {
		t.Errorf("output missing field key1: %q", gotOutput)
	}
	if !strings.Contains(gotOutput, "key2=42") {
		t.Errorf("output missing field key2: %q", gotOutput)
	}
}

func TestStructuredLogger_Info(t *testing.T) {
	var buf bytes.Buffer
	logger := &StructuredLogger{
		prefix: testLoggerPrefix,
		logger: log.New(&buf, "test: ", 0),
	}

	logger.Info(context.Background(), "info message")

	gotOutput := buf.String()
	if !strings.Contains(gotOutput, "[INFO]") {
		t.Errorf("output missing [INFO]: %q", gotOutput)
	}
	if !strings.Contains(gotOutput, "info message") {
		t.Errorf("output missing message: %q", gotOutput)
	}
}

func TestStructuredLogger_Warn(t *testing.T) {
	var buf bytes.Buffer
	logger := &StructuredLogger{
		prefix: testLoggerPrefix,
		logger: log.New(&buf, "test: ", 0),
	}

	logger.Warn(
		context.Background(), "warning message",
		output.Field{Key: "component", Value: "adapter"},
	)

	gotOutput := buf.String()
	if !strings.Contains(gotOutput, "[WARN]") {
		t.Errorf("output missing [WARN]: %q", gotOutput)
	}
	if !strings.Contains(gotOutput, "warning message") {
		t.Errorf("output missing message: %q", gotOutput)
	}
	if !strings.Contains(gotOutput, "component=adapter") {
		t.Errorf("output missing field: %q", gotOutput)
	}
}

func TestStructuredLogger_Error(t *testing.T) {
	var buf bytes.Buffer
	logger := &StructuredLogger{
		prefix: testLoggerPrefix,
		logger: log.New(&buf, "test: ", 0),
	}

	fields := make([]output.Field, 1, 2)
	fields[0] = output.Field{Key: "operation", Value: "detect"}

	testErr := errors.New("test error")
	logger.Error(
		context.Background(), "error occurred", testErr, fields...,
	)

	gotOutput := buf.String()
	if !strings.Contains(gotOutput, "[ERROR]") {
		t.Errorf("output missing [ERROR]: %q", gotOutput)
	}
	if !strings.Contains(gotOutput, "error occurred") {
		t.Errorf("output missing message: %q", gotOutput)
	}
	if !strings.Contains(gotOutput, "error=test error") {
		t.Errorf("output missing error field: %q", gotOutput)
	}
	if !strings.Contains(gotOutput, "operation=detect") {
		t.Errorf("output missing operation field: %q", gotOutput)
	}
	if strings.Index(gotOutput, "operation=detect") > strings.Index(gotOutput, "error=test error") {
		t.Errorf("output fields are out of order: %q", gotOutput)
	}

	spareField := fields[:cap(fields)][1]
	if spareField.Key != "" || spareField.Value != nil {
		t.Errorf("Error() mutated caller field capacity: %#v", spareField)
	}
}

func TestStructuredLogger_MultipleFields(t *testing.T) {
	var buf bytes.Buffer
	logger := &StructuredLogger{
		prefix: testLoggerPrefix,
		logger: log.New(&buf, "test: ", 0),
	}

	logger.Info(
		context.Background(), "multiple fields",
		output.Field{Key: "field1", Value: "value1"},
		output.Field{Key: "field2", Value: "value2"},
		output.Field{Key: "field3", Value: 123},
	)

	gotOutput := buf.String()
	if !strings.Contains(gotOutput, "field1=value1") {
		t.Errorf("output missing field1: %q", gotOutput)
	}
	if !strings.Contains(gotOutput, "field2=value2") {
		t.Errorf("output missing field2: %q", gotOutput)
	}
	if !strings.Contains(gotOutput, "field3=123") {
		t.Errorf("output missing field3: %q", gotOutput)
	}
}

func TestStructuredLogger_NoFields(t *testing.T) {
	var buf bytes.Buffer
	logger := &StructuredLogger{
		prefix: testLoggerPrefix,
		logger: log.New(&buf, "test: ", 0),
	}

	logger.Info(context.Background(), "message without fields")

	gotOutput := buf.String()
	if !strings.Contains(gotOutput, "[INFO]") {
		t.Errorf("output missing [INFO]: %q", gotOutput)
	}
	if !strings.Contains(gotOutput, "message without fields") {
		t.Errorf("output missing message: %q", gotOutput)
	}
	// Should not contain pipe separator when no fields
	if strings.Contains(gotOutput, " | ") {
		t.Errorf("output should not contain pipe separator: %q", gotOutput)
	}
}

func TestStructuredLogger_AllLevels(t *testing.T) {
	levels := []struct {
		name string
		fn   func(*StructuredLogger, context.Context, string, ...output.Field)
		want string
	}{
		{"debug", (*StructuredLogger).Debug, "[DEBUG]"},
		{"info", (*StructuredLogger).Info, "[INFO]"},
		{"warn", (*StructuredLogger).Warn, "[WARN]"},
	}

	for _, tt := range levels {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := &StructuredLogger{
				prefix: testLoggerPrefix,
				logger: log.New(&buf, "test: ", 0),
			}

			tt.fn(logger, context.Background(), "test message")

			gotOutput := buf.String()
			if !strings.Contains(gotOutput, tt.want) {
				t.Errorf("output missing %s: %q", tt.want, gotOutput)
			}
		})
	}
}
