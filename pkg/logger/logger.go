package logger

import (
	"log/slog"
	"os"
	"strings"
)

// LoggerConfig holds logger configuration
type LoggerConfig struct {
	Level  string // "debug", "info", "warn", "error"
	Format string // "text", "json"
	Output string // "stdout", "stderr"
}

// DefaultConfig returns default logger configuration
func DefaultConfig() LoggerConfig {
	return LoggerConfig{
		Level:  getEnvOrDefault("S3C_LOG_LEVEL", "info"),
		Format: getEnvOrDefault("S3C_LOG_FORMAT", "json"), // Default to JSON
		Output: getEnvOrDefault("S3C_LOG_OUTPUT", "stdout"),
	}
}

// NewLogger creates a new structured logger with the given configuration
func NewLogger(config LoggerConfig) *slog.Logger {
	var handler slog.Handler
	level := parseLevel(config.Level)

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: level == slog.LevelDebug, // Add source info for debug level
	}

	output := os.Stdout
	if config.Output == "stderr" {
		output = os.Stderr
	}

	if config.Format == "text" {
		handler = slog.NewTextHandler(output, opts)
	} else {
		// Default to JSON format
		handler = slog.NewJSONHandler(output, opts)
	}

	return slog.New(handler)
}

// NewDefaultLogger creates a logger with default configuration
func NewDefaultLogger() *slog.Logger {
	return NewLogger(DefaultConfig())
}

// parseLevel converts string level to slog.Level
func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// WithComponent adds a component field to the logger for better categorization
func WithComponent(logger *slog.Logger, component string) *slog.Logger {
	return logger.With("component", component)
}

// WithRequestID adds a request ID field to the logger
func WithRequestID(logger *slog.Logger, requestID string) *slog.Logger {
	return logger.With("requestId", requestID)
}

// MaskSensitiveValue masks sensitive information in logs
func MaskSensitiveValue(value string) string {
	if len(value) <= 4 {
		return "****"
	}
	return value[:2] + "****" + value[len(value)-2:]
}
