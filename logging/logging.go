package logging

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/natefinch/lumberjack"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	DebugLogger zerolog.Logger
	InfoLogger  zerolog.Logger
	ErrorLogger zerolog.Logger
)

func Init(logLevel string, logFilePath string) error {
	// Set default log file path if none is provided
	if logFilePath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		logFilePath = filepath.Join(homeDir, ".config", "gollama", "gollama.log")
	}

	// Expand the ~ to the user's home directory
	if strings.HasPrefix(logFilePath, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		logFilePath = filepath.Join(homeDir, logFilePath[1:])
	}

	// Create the directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(logFilePath), 0755); err != nil {
		return err
	}

	// Configure log rotation with lumberjack
	rotate := &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    2,     // megabytes
		MaxBackups: 3,     // number of files
		MaxAge:     60,    // days
		Compress:   false, // disabled by default
	}

	// Set the log level
	level, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		return err
	}
	zerolog.SetGlobalLevel(level)

	// Use lumberjack for logging to a file
	fileWriter := zerolog.MultiLevelWriter(rotate)

	// Initialise loggers
	log.Logger = zerolog.New(fileWriter).With().Timestamp().Logger()
	DebugLogger = log.Logger.Level(zerolog.DebugLevel)
	InfoLogger = log.Logger.Level(zerolog.InfoLevel)
	ErrorLogger = log.Logger.Level(zerolog.ErrorLevel)

	if logLevel == "debug" {
		DebugLogger.Printf("Logging to: %s\n", logFilePath)
	}

	return nil
}
