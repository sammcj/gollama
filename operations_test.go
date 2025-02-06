package main

import (
	"os"
	"testing"

	"github.com/sammcj/gollama/config"
)

func TestRunModel(t *testing.T) {
	// Determine if we're running in CI
	inCI := os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != ""

	// if we're running in CI, simply skip these tests
	if inCI {
		t.Skip("Skipping operations tests in CI")
	} else {

		tests := []struct {
			name         string
			model        string
			cfg          *config.Config
			expectDocker bool
			expectError  bool
		}{
			{
				name:         "Run with Docker",
				model:        "test-model",
				cfg:          &config.Config{DockerContainer: "test-container"},
				expectDocker: true,
				expectError:  inCI, // Expect error in CI since docker won't be available
			},
			{
				name:         "Run without Docker",
				model:        "test-model",
				cfg:          &config.Config{DockerContainer: ""},
				expectDocker: false,
				expectError:  inCI, // Expect error in CI since ollama won't be available
			},
			{
				name:         "Run with Docker set to false",
				model:        "test-model",
				cfg:          &config.Config{DockerContainer: "false"},
				expectDocker: false,
				expectError:  inCI, // Expect error in CI since ollama won't be available
			},
		}

		for _, tt := range tests {
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
