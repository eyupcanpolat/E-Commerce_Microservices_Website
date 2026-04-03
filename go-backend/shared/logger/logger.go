// Package logger provides a structured logging utility for all services.
package logger

import (
	"log/slog"
	"os"
)

var Log *slog.Logger

func init() {
	// Use JSON handler for structured/machine-readable logs in production
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	Log = slog.New(handler)
}

// Info logs an informational message with optional key-value pairs.
func Info(msg string, args ...any) {
	Log.Info(msg, args...)
}

// Error logs an error message with optional key-value pairs.
func Error(msg string, args ...any) {
	Log.Error(msg, args...)
}

// Warn logs a warning message.
func Warn(msg string, args ...any) {
	Log.Warn(msg, args...)
}

// Debug logs a debug message.
func Debug(msg string, args ...any) {
	Log.Debug(msg, args...)
}

// Fatal logs an error and exits the application.
func Fatal(msg string, args ...any) {
	Log.Error(msg, args...)
	os.Exit(1)
}
