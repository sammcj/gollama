package logging

import (
  "io"
  "log"
  "os"
)

var (
  DebugLogger *log.Logger
  InfoLogger  *log.Logger
  ErrorLogger *log.Logger
)

func Init(logLevel, logFilePath string) error {
  f, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
  if err != nil {
    return err
  }

  DebugLogger = log.New(io.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
  InfoLogger = log.New(f, "INFO: ", log.Ldate|log.Ltime)
  ErrorLogger = log.New(f, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)

  if logLevel == "debug" {
    DebugLogger.SetOutput(f)
  }
  return nil
}
