package logger

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
)

// StructuredLogger provides structured logging using the standard library.
type StructuredLogger struct {
	prefix string
	logger *log.Logger
}

// NewStructuredLogger creates a new structured logger.
func NewStructuredLogger(prefix string) *StructuredLogger {
	return &StructuredLogger{
		prefix: prefix,
		logger: log.New(os.Stderr, prefix+": ", log.LstdFlags),
	}
}

// Debug logs a debug-level message.
func (l *StructuredLogger) Debug(_ context.Context, msg string, fields ...output.Field) {
	l.log("DEBUG", msg, fields)
}

// Info logs an info-level message.
func (l *StructuredLogger) Info(_ context.Context, msg string, fields ...output.Field) {
	l.log("INFO", msg, fields)
}

// Warn logs a warning-level message.
func (l *StructuredLogger) Warn(_ context.Context, msg string, fields ...output.Field) {
	l.log("WARN", msg, fields)
}

// Error logs an error-level message.
func (l *StructuredLogger) Error(_ context.Context, msg string, err error, fields ...output.Field) {
	allFields := append(fields, output.Field{Key: "error", Value: err.Error()})
	l.log("ERROR", msg, allFields)
}

func (l *StructuredLogger) log(level, msg string, fields []output.Field) {
	if len(fields) == 0 {
		l.logger.Printf("[%s] %s", level, msg)
		return
	}

	var fieldStr strings.Builder
	for i, field := range fields {
		if i > 0 {
			fieldStr.WriteString(" ")
		}
		fieldStr.WriteString(fmt.Sprintf("%s=%v", field.Key, field.Value))
	}

	l.logger.Printf("[%s] %s | %s", level, msg, fieldStr.String())
}
