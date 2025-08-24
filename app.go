package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/ollama/ollama/api"
)

// App struct
type App struct {
	ctx    context.Context
	client *api.Client
}

// Model represents an Ollama model
type Model struct {
	Name              string    `json:"name"`
	ID                string    `json:"id"`
	Size              float64   `json:"size"`
	QuantizationLevel string    `json:"quantizationLevel"`
	Modified          time.Time `json:"modified"`
	Family            string    `json:"family"`
	Selected          bool      `json:"selected"`
}

// ModelDetails represents detailed information about a model
type ModelDetails struct {
	Name              string            `json:"name"`
	ID                string            `json:"id"`
	Size              float64           `json:"size"`
	QuantizationLevel string            `json:"quantizationLevel"`
	Modified          string            `json:"modified"`
	Family            string            `json:"family"`
	Parameters        map[string]string `json:"parameters"`
}

// NewApp creates a new App application struct
func NewApp() *App {
	apiURL, _ := url.Parse("http://localhost:11434")
	client := api.NewClient(apiURL, &http.Client{})
	return &App{
		client: client,
	}
}

// startup is called when the app starts. The context is saved
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// GetModels returns a list of all available models
func (a *App) GetModels() ([]Model, error) {
	resp, err := a.client.List(a.ctx)
	if err != nil {
		return nil, fmt.Errorf("error fetching models: %v", err)
	}

	var models []Model
	for _, m := range resp.Models {
		model := Model{
			Name:              m.Name,
			ID:                m.Digest,
			Size:              float64(m.Size) / 1024 / 1024 / 1024, // Convert to GB
			QuantizationLevel: m.Details.QuantizationLevel,
			Modified:          m.ModifiedAt,
			Family:            m.Details.Family,
		}
		models = append(models, model)
	}

	return models, nil
}

// DeleteModel deletes a model by name
func (a *App) DeleteModel(name string) error {
	err := a.client.Delete(a.ctx, &api.DeleteRequest{Name: name})
	if err != nil {
		return fmt.Errorf("error deleting model: %v", err)
	}
	return nil
}

// RunModel runs a model by name
func (a *App) RunModel(name string) error {
	_, err := a.client.Generate(a.ctx, &api.GenerateRequest{
		Model:  name,
		Prompt: "Hello",
	})
	if err != nil {
		return fmt.Errorf("error running model: %v", err)
	}
	return nil
}

// UnloadModel unloads a model from memory
func (a *App) UnloadModel(name string) error {
	err := a.client.Delete(a.ctx, &api.DeleteRequest{Name: name})
	if err != nil {
		return fmt.Errorf("error unloading model: %v", err)
	}
	return nil
}

// InspectModel returns detailed information about a model
func (a *App) InspectModel(name string) (*ModelDetails, error) {
	resp, err := a.client.Show(a.ctx, &api.ShowRequest{Name: name})
	if err != nil {
		return nil, fmt.Errorf("error getting model details: %v", err)
	}

	details := &ModelDetails{
		Name:              resp.Model.Name,
		ID:                resp.Model.Digest,
		Size:              float64(resp.Model.Size) / 1024 / 1024 / 1024, // Convert to GB
		QuantizationLevel: resp.Model.Details.QuantizationLevel,
		Modified:          resp.Model.ModifiedAt.Format(time.RFC3339),
		Family:            resp.Model.Details.Family,
		Parameters:        resp.Model.Details.Parameters,
	}

	return details, nil
}
