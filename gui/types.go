package main

import (
	"time"

	"github.com/sammcj/gollama/core"
)

// GuiModel represents a model with GUI-friendly formatting
type GuiModel struct {
	Name              string    `json:"name"`
	ID                string    `json:"id"`
	Size              float64   `json:"size"`
	SizeFormatted     string    `json:"sizeFormatted"`
	QuantizationLevel string    `json:"quantizationLevel"`
	Modified          time.Time `json:"modified"`
	ModifiedFormatted string    `json:"modifiedFormatted"`
	Family            string    `json:"family"`
	ParameterSize     string    `json:"parameterSize"`
	Selected          bool      `json:"selected"`
	IsRunning         bool      `json:"isRunning"`
	Status            string    `json:"status"`
}

// GuiRunningModel represents a running model with GUI-friendly formatting
type GuiRunningModel struct {
	Name            string    `json:"name"`
	VRAMUsage       float64   `json:"vramUsage"`
	VRAMFormatted   string    `json:"vramFormatted"`
	LoadedAt        time.Time `json:"loadedAt"`
	LoadedAtFormatted string  `json:"loadedAtFormatted"`
	ExpiresAt       time.Time `json:"expiresAt"`
	ExpiresAtFormatted string `json:"expiresAtFormatted"`
}

// PageData represents data passed to templates
type PageData struct {
	Title         string            `json:"title"`
	Models        []GuiModel        `json:"models"`
	RunningModels []GuiRunningModel `json:"runningModels"`
	CurrentView   string            `json:"currentView"`
	Config        interface{}       `json:"config"`
	Error         string            `json:"error,omitempty"`
	Success       string            `json:"success,omitempty"`
	Query         string            `json:"query,omitempty"`
	SortBy        string            `json:"sortBy,omitempty"`
	SortOrder     string            `json:"sortOrder,omitempty"`
}

// ModelDetailsData represents detailed model information for modals
type ModelDetailsData struct {
	Model   GuiModel                `json:"model"`
	Details *core.EnhancedModelInfo `json:"details"`
}

// VRAMEstimateRequest represents a vRAM estimation request
type VRAMEstimateRequest struct {
	ModelName     string  `json:"modelName"`
	VRAMAvailable float64 `json:"vramAvailable"`
	ContextLength int     `json:"contextLength"`
	Quantization  string  `json:"quantization"`
}

// VRAMEstimateResponse represents a vRAM estimation response
type VRAMEstimateResponse struct {
	ModelName     string                        `json:"modelName"`
	Estimation    *core.VRAMEstimation         `json:"estimation"`
	Recommendations []VRAMRecommendation       `json:"recommendations"`
	Error         string                       `json:"error,omitempty"`
}

// VRAMRecommendation represents a quantization recommendation
type VRAMRecommendation struct {
	Quantization  string  `json:"quantization"`
	VRAMRequired  float64 `json:"vramRequired"`
	ContextLength int     `json:"contextLength"`
	Fits          bool    `json:"fits"`
}

// SettingsData represents settings form data
type SettingsData struct {
	OllamaAPIURL    string `json:"ollamaApiUrl"`
	Theme           string `json:"theme"`
	AutoRefresh     bool   `json:"autoRefresh"`
	RefreshInterval int    `json:"refreshInterval"`
	WindowWidth     int    `json:"windowWidth"`
	WindowHeight    int    `json:"windowHeight"`
	DefaultView     string `json:"defaultView"`
	ShowSystemTray  bool   `json:"showSystemTray"`
}

// ToastMessage represents a notification message
type ToastMessage struct {
	Type    string `json:"type"`    // success, error, warning, info
	Message string `json:"message"`
	Title   string `json:"title,omitempty"`
}
