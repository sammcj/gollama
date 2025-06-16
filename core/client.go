package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OllamaClient handles communication with the Ollama API
type OllamaClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewOllamaClient creates a new Ollama API client
func NewOllamaClient(baseURL, apiKey string) (*OllamaClient, error) {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	// Ensure baseURL doesn't end with a slash
	baseURL = strings.TrimSuffix(baseURL, "/")

	client := &OllamaClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	return client, nil
}

// makeRequest makes an HTTP request to the Ollama API
func (c *OllamaClient) makeRequest(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	url := c.baseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	return resp, nil
}

// ListModels retrieves all available models from Ollama
func (c *OllamaClient) ListModels(ctx context.Context) ([]Model, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/tags", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	var response struct {
		Models []struct {
			Name       string    `json:"name"`
			Size       int64     `json:"size"`
			Digest     string    `json:"digest"`
			ModifiedAt time.Time `json:"modified_at"`
			Details    struct {
				Parent            string            `json:"parent"`
				Format            string            `json:"format"`
				Family            string            `json:"family"`
				Families          []string          `json:"families"`
				ParameterSize     string            `json:"parameter_size"`
				QuantizationLevel string            `json:"quantization_level"`
			} `json:"details"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	models := make([]Model, len(response.Models))
	for i, m := range response.Models {
		models[i] = Model{
			Name:       m.Name,
			ID:         m.Name, // Use name as ID for now
			Size:       m.Size,
			Digest:     m.Digest,
			ModifiedAt: m.ModifiedAt,
			Details: ModelDetails{
				Parent:            m.Details.Parent,
				Format:            m.Details.Format,
				Family:            m.Details.Family,
				Families:          m.Details.Families,
				ParameterSize:     m.Details.ParameterSize,
				QuantizationLevel: m.Details.QuantizationLevel,
			},
			Status: "available",
		}
	}

	return models, nil
}

// GetModel retrieves detailed information about a specific model
func (c *OllamaClient) GetModel(ctx context.Context, name string) (*EnhancedModelInfo, error) {
	resp, err := c.makeRequest(ctx, "POST", "/api/show", map[string]string{"name": name})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	var response struct {
		Modelfile  string                 `json:"modelfile"`
		Parameters map[string]interface{} `json:"parameters"`
		Template   string                 `json:"template"`
		System     string                 `json:"system"`
		Details    struct {
			Parent            string   `json:"parent"`
			Format            string   `json:"format"`
			Family            string   `json:"family"`
			Families          []string `json:"families"`
			ParameterSize     string   `json:"parameter_size"`
			QuantizationLevel string   `json:"quantization_level"`
		} `json:"details"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	model := Model{
		Name: name,
		ID:   name,
		Details: ModelDetails{
			Parent:            response.Details.Parent,
			Format:            response.Details.Format,
			Family:            response.Details.Family,
			Families:          response.Details.Families,
			ParameterSize:     response.Details.ParameterSize,
			QuantizationLevel: response.Details.QuantizationLevel,
			Template:          response.Template,
			System:            response.System,
		},
		Status: "available",
	}

	return &EnhancedModelInfo{
		Model:      model,
		Modelfile:  response.Modelfile,
		Template:   response.Template,
		System:     response.System,
		Parameters: response.Parameters,
	}, nil
}

// DeleteModel removes a model from Ollama
func (c *OllamaClient) DeleteModel(ctx context.Context, name string) error {
	resp, err := c.makeRequest(ctx, "DELETE", "/api/delete", map[string]string{"name": name})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	return nil
}

// RunModel starts running a model
func (c *OllamaClient) RunModel(ctx context.Context, name string) error {
	resp, err := c.makeRequest(ctx, "POST", "/api/generate", map[string]interface{}{
		"model":  name,
		"prompt": "", // Empty prompt just to load the model
		"stream": false,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	return nil
}

// GetRunningModels retrieves currently running models
func (c *OllamaClient) GetRunningModels(ctx context.Context) ([]RunningModel, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/ps", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	var response struct {
		Models []struct {
			Name      string    `json:"name"`
			Size      int64     `json:"size"`
			SizeVRAM  int64     `json:"size_vram"`
			ExpiresAt time.Time `json:"expires_at"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	models := make([]RunningModel, len(response.Models))
	for i, m := range response.Models {
		models[i] = RunningModel{
			Name:      m.Name,
			Size:      m.Size,
			SizeVRAM:  m.SizeVRAM,
			LoadedAt:  time.Now(), // Ollama doesn't provide this, so we use current time
			ExpiresAt: m.ExpiresAt,
		}
	}

	return models, nil
}

// UnloadModel unloads a specific model
func (c *OllamaClient) UnloadModel(ctx context.Context, name string) error {
	resp, err := c.makeRequest(ctx, "POST", "/api/generate", map[string]interface{}{
		"model":     name,
		"keep_alive": 0, // Unload immediately
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	return nil
}

// PullModel downloads a model from a registry
func (c *OllamaClient) PullModel(ctx context.Context, name string) error {
	resp, err := c.makeRequest(ctx, "POST", "/api/pull", map[string]string{"name": name})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	return nil
}

// PushModel uploads a model to a registry
func (c *OllamaClient) PushModel(ctx context.Context, name string) error {
	resp, err := c.makeRequest(ctx, "POST", "/api/push", map[string]string{"name": name})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	return nil
}

// CopyModel creates a copy of a model with a new name
func (c *OllamaClient) CopyModel(ctx context.Context, source, destination string) error {
	resp, err := c.makeRequest(ctx, "POST", "/api/copy", map[string]string{
		"source":      source,
		"destination": destination,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	return nil
}

// CreateModel creates a new model from a Modelfile
func (c *OllamaClient) CreateModel(ctx context.Context, name, modelfile string) error {
	resp, err := c.makeRequest(ctx, "POST", "/api/create", map[string]string{
		"name":      name,
		"modelfile": modelfile,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	return nil
}

// HealthCheck verifies that the Ollama API is accessible
func (c *OllamaClient) HealthCheck(ctx context.Context) error {
	resp, err := c.makeRequest(ctx, "GET", "/api/tags", nil)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}

	return nil
}
