package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sammcj/gollama/config"
)

// GollamaService provides the core business logic for both TUI and GUI interfaces
type GollamaService struct {
	client     *OllamaClient
	config     *config.Config
	logger     *Logger
	eventBus   *EventBus
	ctx        context.Context
	cancelFunc context.CancelFunc
	mutex      sync.RWMutex
}

// ServiceConfig holds configuration for initialising the service
type ServiceConfig struct {
	OllamaAPIURL    string
	OllamaAPIKey    string
	LogLevel        string
	ConfigPath      string
	Context         context.Context
}

// NewGollamaService creates a new instance of the core service
func NewGollamaService(cfg ServiceConfig) (*GollamaService, error) {
	// Load configuration
	config, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Override config with provided values if specified
	if cfg.OllamaAPIURL != "" {
		config.OllamaAPIURL = cfg.OllamaAPIURL
	}
	if cfg.OllamaAPIKey != "" {
		config.OllamaAPIKey = cfg.OllamaAPIKey
	}
	if cfg.LogLevel != "" {
		config.LogLevel = cfg.LogLevel
	}

	// Initialise logger
	logger, err := NewLogger(config.LogLevel, config.LogFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialise logger: %w", err)
	}

	// Create context
	ctx := cfg.Context
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)

	// Create Ollama client
	client, err := NewOllamaClient(config.OllamaAPIURL, config.OllamaAPIKey)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create Ollama client: %w", err)
	}

	// Create event bus
	eventBus := NewEventBus()

	service := &GollamaService{
		client:     client,
		config:     &config,
		logger:     logger,
		eventBus:   eventBus,
		ctx:        ctx,
		cancelFunc: cancel,
	}

	logger.Info("Gollama service initialised successfully")
	return service, nil
}

// Close gracefully shuts down the service
func (s *GollamaService) Close() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.cancelFunc != nil {
		s.cancelFunc()
	}

	if s.eventBus != nil {
		s.eventBus.Close()
	}

	s.logger.Info("Gollama service shut down")
	return nil
}

// GetConfig returns the current configuration
func (s *GollamaService) GetConfig() *config.Config {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.config
}

// UpdateConfig updates the service configuration
func (s *GollamaService) UpdateConfig(cfg *config.Config) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Save configuration
	if err := config.SaveConfig(*cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	s.config = cfg
	s.logger.Info("Configuration updated successfully")

	// Emit configuration updated event
	s.eventBus.Emit(Event{
		Type: EventConfigUpdated,
		Data: cfg,
		Time: time.Now(),
	})

	return nil
}

// GetLogger returns the service logger
func (s *GollamaService) GetLogger() *Logger {
	return s.logger
}

// GetEventBus returns the event bus for subscribing to events
func (s *GollamaService) GetEventBus() *EventBus {
	return s.eventBus
}

// Context returns the service context
func (s *GollamaService) Context() context.Context {
	return s.ctx
}
