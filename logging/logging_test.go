package logging

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInit(t *testing.T) {
	tests := []struct {
		name          string
		logLevel      string
		logFilePath   string
		expectedError bool
		expectedDebug bool
	}{
		{
			name:          "Valid debug log level with valid log file path",
			logLevel:      "debug",
			logFilePath:   "test_debug.log",
			expectedError: false,
			expectedDebug: true,
		},
		{
			name:          "Valid info log level with valid log file path",
			logLevel:      "info",
			logFilePath:   "test_info.log",
			expectedError: false,
			expectedDebug: false,
		},
		{
			name:          "Invalid log file path",
			logLevel:      "debug",
			logFilePath:   "/invalid/path/test.log",
			expectedError: true,
			expectedDebug: false,
		},
		{
			name:          "Empty log file path (use default)",
			logLevel:      "info",
			logFilePath:   "",
			expectedError: false,
			expectedDebug: false,
		},
		{
			name:          "Log file path with home directory (~)",
			logLevel:      "debug",
			logFilePath:   "~/test_home.log",
			expectedError: false,
			expectedDebug: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Determine the actual log file path
			var actualLogFilePath string
			if tt.logFilePath == "" {
				homeDir, err := os.UserHomeDir()
				if err != nil {
					t.Fatalf("Failed to get user home directory: %v", err)
				}
				actualLogFilePath = filepath.Join(homeDir, ".config", "gollama", "gollama.log")
			} else if strings.HasPrefix(tt.logFilePath, "~") {
				homeDir, err := os.UserHomeDir()
				if err != nil {
					t.Fatalf("Failed to get user home directory: %v", err)
				}
				actualLogFilePath = filepath.Join(homeDir, tt.logFilePath[1:])
			} else {
				actualLogFilePath = tt.logFilePath
			}

			// Create directory if it does not exist
			if !tt.expectedError {
				if err := os.MkdirAll(filepath.Dir(actualLogFilePath), 0755); err != nil {
					t.Fatalf("Failed to create log directory: %v", err)
				}
			}

			// Clean up any existing log file
			defer os.Remove(actualLogFilePath)

			err := Init(tt.logLevel, tt.logFilePath)
			if (err != nil) != tt.expectedError {
				t.Fatalf("Init() error = %v, expectedError %v", err, tt.expectedError)
			}

			if !tt.expectedError {
				// Log messages to each logger
				InfoLogger.Info().Msg("This is an info message")
				ErrorLogger.Error().Msg("This is an error message")
				DebugLogger.Debug().Msg("This is a debug message")

				// Check the contents of the log file
				data, err := ioutil.ReadFile(actualLogFilePath)
				if err != nil {
					t.Fatalf("Failed to read log file: %v", err)
				}
				logContent := string(data)

				// Check for presence of info and error messages
				if !strings.Contains(logContent, "This is an info message") {
					t.Errorf("Info log message not found in log file: %s", logContent)
				}
				if !strings.Contains(logContent, "This is an error message") {
					t.Errorf("Error log message not found in log file: %s", logContent)
				}

				// Check for presence of debug messages based on log level
				if tt.expectedDebug && !strings.Contains(logContent, "This is a debug message") {
					t.Errorf("Expected debug log message not found in log file: %s", logContent)
				}
				if !tt.expectedDebug && strings.Contains(logContent, "This is a debug message") {
					t.Errorf("Unexpected debug log message found in log file: %s", logContent)
				}
			}
		})
	}
}
