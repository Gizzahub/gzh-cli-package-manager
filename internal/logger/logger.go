// Package logger provides a simple structured logger for package manager operations.
package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

// Level represents the logging level.
type Level int

const (
	// DebugLevel for debug messages.
	DebugLevel Level = iota
	// InfoLevel for informational messages.
	InfoLevel
	// WarnLevel for warning messages.
	WarnLevel
	// ErrorLevel for error messages.
	ErrorLevel
)

// String returns the string representation of a Level.
func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger provides structured logging capabilities.
type Logger struct {
	mu        sync.Mutex
	out       io.Writer
	level     Level
	component string
	fields    map[string]interface{}
}

// New creates a new Logger with the given component name.
func New(component string) *Logger {
	return &Logger{
		out:       os.Stderr,
		level:     InfoLevel,
		component: component,
		fields:    make(map[string]interface{}),
	}
}

// SetOutput sets the output destination for the logger.
func (l *Logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.out = w
}

// SetLevel sets the minimum logging level.
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// WithField returns a new Logger with the specified field added.
func (l *Logger) WithField(key string, value interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	fields := make(map[string]interface{})
	for k, v := range l.fields {
		fields[k] = v
	}
	fields[key] = value

	return &Logger{
		out:       l.out,
		level:     l.level,
		component: l.component,
		fields:    fields,
	}
}

// WithFields returns a new Logger with multiple fields added.
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newFields := make(map[string]interface{})
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}

	return &Logger{
		out:       l.out,
		level:     l.level,
		component: l.component,
		fields:    newFields,
	}
}

// log writes a log message at the specified level.
func (l *Logger) log(level Level, format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if level < l.level {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf(format, args...)

	// Build fields string
	var fieldParts []string
	for k, v := range l.fields {
		fieldParts = append(fieldParts, fmt.Sprintf("%s=%v", k, v))
	}

	fieldsStr := ""
	if len(fieldParts) > 0 {
		fieldsStr = " " + strings.Join(fieldParts, " ")
	}

	logLine := fmt.Sprintf("[%s] %s [%s]%s %s\n",
		timestamp,
		level.String(),
		l.component,
		fieldsStr,
		message,
	)

	_, _ = l.out.Write([]byte(logLine))
}

// Debug logs a debug message.
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DebugLevel, format, args...)
}

// Info logs an informational message.
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(InfoLevel, format, args...)
}

// Warn logs a warning message.
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WarnLevel, format, args...)
}

// Error logs an error message.
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ErrorLevel, format, args...)
}

// Fatal logs a fatal error and exits the program.
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(ErrorLevel, format, args...)
	os.Exit(1)
}

// Standard logger for package-level functions
var std = New("gz-pm")

// SetOutput sets the output for the standard logger.
func SetOutput(w io.Writer) {
	std.SetOutput(w)
}

// SetLevel sets the level for the standard logger.
func SetLevel(level Level) {
	std.SetLevel(level)
}

// Debug logs a debug message using the standard logger.
func Debug(format string, args ...interface{}) {
	std.Debug(format, args...)
}

// Info logs an informational message using the standard logger.
func Info(format string, args ...interface{}) {
	std.Info(format, args...)
}

// Warn logs a warning message using the standard logger.
func Warn(format string, args ...interface{}) {
	std.Warn(format, args...)
}

// Error logs an error message using the standard logger.
func Error(format string, args ...interface{}) {
	std.Error(format, args...)
}

// Fatal logs a fatal error using the standard logger and exits.
func Fatal(format string, args ...interface{}) {
	std.Fatal(format, args...)
}

// WithField returns a logger with the specified field.
func WithField(key string, value interface{}) *Logger {
	return std.WithField(key, value)
}

// WithFields returns a logger with multiple fields.
func WithFields(fields map[string]interface{}) *Logger {
	return std.WithFields(fields)
}

// ParseLevel parses a string into a Level.
func ParseLevel(s string) (Level, error) {
	switch strings.ToUpper(s) {
	case "DEBUG":
		return DebugLevel, nil
	case "INFO":
		return InfoLevel, nil
	case "WARN", "WARNING":
		return WarnLevel, nil
	case "ERROR":
		return ErrorLevel, nil
	default:
		return InfoLevel, fmt.Errorf("invalid log level: %s", s)
	}
}

// init initializes the standard logger to suppress default log output.
func init() {
	log.SetOutput(io.Discard)
}
