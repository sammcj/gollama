package main

import (
	"os"
	"testing"

	"github.com/sammcj/gollama/config"
)

func TestRunModel(t *testing.T) {
	tests := []struct {
		name         string
		model        string
		cfg          *config.Config
		expectDocker bool
		expectError  bool
		skipInCI     bool // Skip this test in CI
	}{
		{
			name:         "Run with Docker",
			model:        "test-model",
			cfg:          &config.Config{DockerContainer: "test-container"},
			expectDocker: true,
			expectError:  false,
			skipInCI:     true,
		},
		{
			name:         "Run without Docker",
			model:        "test-model",
			cfg:          &config.Config{DockerContainer: ""},
			expectDocker: false,
			expectError:  false,
			skipInCI:     true,
		},
		{
			name:         "Run with Docker set to false",
			model:        "test-model",
			cfg:          &config.Config{DockerContainer: "false"},
			expectDocker: false,
			expectError:  false,
			skipInCI:     true,
		},
	}

	for _, tt := range tests {
		if tt.skipInCI && os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" {
			t.Skip("Skipping test in CI environment")
		} else {
			t.Run(tt.name, func(t *testing.T) {
				cmd := runModel(tt.model, tt.cfg)
				if (cmd == nil) != tt.expectError {
					t.Errorf("runModel() error = %v, expectError %v", cmd == nil, tt.expectError)
					t.Logf("cmd: %v", cmd)
					return
				}
				// Further assertions can be added based on how you want to validate the `tea.Cmd` returned
			})
		}
	}
}
