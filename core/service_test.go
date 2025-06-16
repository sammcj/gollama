package core

import (
	"context"
	"testing"
	"time"
)

func TestNewGollamaService(t *testing.T) {
	tests := []struct {
		name    string
		config  ServiceConfig
		wantErr bool
	}{
		{
			name: "Valid service configuration",
			config: ServiceConfig{
				OllamaAPIURL: "http://localhost:11434",
				LogLevel:     "info",
				Context:      context.Background(),
			},
			wantErr: false,
		},
		{
			name: "Empty configuration uses defaults",
			config: ServiceConfig{
				Context: context.Background(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewGollamaService(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewGollamaService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if service == nil {
					t.Error("NewGollamaService() returned nil service")
					return
				}

				// Test that service components are initialised
				if service.GetConfig() == nil {
					t.Error("Service config is nil")
				}

				if service.GetLogger() == nil {
					t.Error("Service logger is nil")
				}

				if service.GetEventBus() == nil {
					t.Error("Service event bus is nil")
				}

				if service.Context() == nil {
					t.Error("Service context is nil")
				}

				// Test service close
				if err := service.Close(); err != nil {
					t.Errorf("Service.Close() error = %v", err)
				}
			}
		})
	}
}

func TestGollamaService_UpdateConfig(t *testing.T) {
	service, err := NewGollamaService(ServiceConfig{
		Context: context.Background(),
	})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	defer service.Close()

	// Get original config
	originalConfig := service.GetConfig()

	// Create updated config
	updatedConfig := *originalConfig
	updatedConfig.LogLevel = "debug"
	updatedConfig.GuiEnabled = true
	updatedConfig.WindowWidth = 1600

	// Update config
	err = service.UpdateConfig(&updatedConfig)
	if err != nil {
		t.Errorf("UpdateConfig() error = %v", err)
		return
	}

	// Verify config was updated
	currentConfig := service.GetConfig()
	if currentConfig.LogLevel != "debug" {
		t.Errorf("Expected LogLevel to be 'debug', got '%s'", currentConfig.LogLevel)
	}
	if !currentConfig.GuiEnabled {
		t.Error("Expected GuiEnabled to be true")
	}
	if currentConfig.WindowWidth != 1600 {
		t.Errorf("Expected WindowWidth to be 1600, got %d", currentConfig.WindowWidth)
	}
}

func TestGollamaService_EventBus(t *testing.T) {
	service, err := NewGollamaService(ServiceConfig{
		Context: context.Background(),
	})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	defer service.Close()

	eventBus := service.GetEventBus()
	if eventBus == nil {
		t.Fatal("Event bus is nil")
	}

	// Test event subscription and emission
	testEventType := "test_event"

	// Subscribe to events
	eventChannel := eventBus.Subscribe(testEventType)

	// Emit a test event
	testEvent := Event{
		Type: testEventType,
		Data: "test_data",
		Time: time.Now(),
	}
	eventBus.Emit(testEvent)

	// Wait for event with timeout
	select {
	case receivedEvent := <-eventChannel:
		if receivedEvent.Type != testEvent.Type {
			t.Errorf("Expected event type '%s', got '%s'", testEvent.Type, receivedEvent.Type)
		}
		if receivedEvent.Data != testEvent.Data {
			t.Errorf("Expected event data '%v', got '%v'", testEvent.Data, receivedEvent.Data)
		}
	case <-time.After(1 * time.Second):
		t.Error("Event was not received within timeout")
	}
}

func TestGollamaService_HealthCheck(t *testing.T) {
	service, err := NewGollamaService(ServiceConfig{
		Context: context.Background(),
	})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	defer service.Close()

	// Note: This test will likely fail if Ollama is not running locally
	// In a real test environment, you might want to mock the client
	err = service.HealthCheck()
	// We don't assert on the error here since Ollama might not be running
	// In production tests, you would mock the client or ensure Ollama is available
	t.Logf("HealthCheck result: %v", err)
}
