package core

import (
	"github.com/sammcj/gollama/logging"
)

// Logger wraps the existing logging functionality for the core service
type Logger struct {
	level    string
	filePath string
}

// NewLogger creates a new logger instance
func NewLogger(level, filePath string) (*Logger, error) {
	// Initialize the existing logging system
	if err := logging.Init(level, filePath); err != nil {
		return nil, err
	}

	return &Logger{
		level:    level,
		filePath: filePath,
	}, nil
}

// Debug logs a debug message
func (l *Logger) Debug(msg string) {
	logging.DebugLogger.Debug().Msg(msg)
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, args ...interface{}) {
	logging.DebugLogger.Debug().Msgf(format, args...)
}

// Info logs an info message
func (l *Logger) Info(msg string) {
	logging.InfoLogger.Info().Msg(msg)
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...interface{}) {
	logging.InfoLogger.Info().Msgf(format, args...)
}

// Error logs an error message
func (l *Logger) Error(msg string) {
	logging.ErrorLogger.Error().Msg(msg)
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	logging.ErrorLogger.Error().Msgf(format, args...)
}

// With returns a logger with additional context fields
func (l *Logger) With(key string, value interface{}) *Logger {
	// For now, return the same logger
	// In the future, we could implement context-aware logging
	return l
}

// GetLevel returns the current log level
func (l *Logger) GetLevel() string {
	return l.level
}

// GetFilePath returns the log file path
func (l *Logger) GetFilePath() string {
	return l.filePath
}
