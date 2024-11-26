package utils

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGetHomeDir(t *testing.T) {
	expected := homeDir()
	got := GetHomeDir()
	if got != expected {
		t.Errorf("GetHomeDir() = %v, want %v", got, expected)
	}
}

func TestGetConfigDir(t *testing.T) {
	expected := filepath.Join(homeDir(), ".config", "gollama")
	got := GetConfigDir()
	if got != expected {
		t.Errorf("GetConfigDir() = %v, want %v", got, expected)
	}
}

func TestGetConfigPath(t *testing.T) {
	expected := filepath.Join(homeDir(), ".config", "gollama", "config.json")
	got := GetConfigPath()
	if got != expected {
		t.Errorf("GetConfigPath() = %v, want %v", got, expected)
	}
}

func homeDir() string {
	// Get User Home directory (simplified). Refer to "os/file"
	var env string
	if runtime.GOOS == "windows" {
		env = "USERPROFILE"
	} else {
		env = "HOME"
	}
	if v := os.Getenv(env); v != "" {
		return v
	}
	return ""
}
