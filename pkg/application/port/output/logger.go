package output

import "context"

// Logger provides structured logging capabilities.
// This port is implemented by infrastructure layer.
type Logger interface {
	// Debug logs a debug-level message.
	Debug(ctx context.Context, msg string, fields ...Field)

	// Info logs an info-level message.
	Info(ctx context.Context, msg string, fields ...Field)

	// Warn logs a warning-level message.
	Warn(ctx context.Context, msg string, fields ...Field)

	// Error logs an error-level message.
	Error(ctx context.Context, msg string, err error, fields ...Field)
}

// Field represents a structured logging field.
type Field struct {
	Key   string
	Value any
}
