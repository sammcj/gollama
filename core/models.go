package core

import (
	"time"
)

// Model represents an Ollama model with enhanced information
type Model struct {
	Name         string                 `json:"name"`
	ID           string                 `json:"id"`
	Size         int64                  `json:"size"`
	Digest       string                 `json:"digest"`
	ModifiedAt   time.Time              `json:"modified_at"`
	Details      ModelDetails           `json:"details"`
	Status       string                 `json:"status"`
	ExpiresAt    *time.Time             `json:"expires_at,omitempty"`
	SizeVRAM     int64                  `json:"size_vram,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ModelDetails contains detailed information about a model
type ModelDetails struct {
	Parent            string            `json:"parent"`
	Format            string            `json:"format"`
	Family            string            `json:"family"`
	Families          []string          `json:"families"`
	ParameterSize     string            `json:"parameter_size"`
	QuantizationLevel string            `json:"quantization_level"`
	License           string            `json:"license"`
	Template          string            `json:"template"`
	System            string            `json:"system"`
	Parameters        map[string]string `json:"parameters"`
}

// RunningModel represents a currently running model
type RunningModel struct {
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	SizeVRAM  int64     `json:"size_vram"`
	LoadedAt  time.Time `json:"loaded_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// EnhancedModelInfo contains comprehensive model information
type EnhancedModelInfo struct {
	Model       Model                  `json:"model"`
	Modelfile   string                 `json:"modelfile"`
	Template    string                 `json:"template"`
	System      string                 `json:"system"`
	Parameters  map[string]interface{} `json:"parameters"`
	VRAMUsage   *VRAMEstimation        `json:"vram_usage,omitempty"`
}

// VRAMConstraints defines constraints for vRAM estimation
type VRAMConstraints struct {
	AvailableVRAM   float64 `json:"available_vram"`
	ContextLength   int     `json:"context_length"`
	Quantization    string  `json:"quantization"`
	BatchSize       int     `json:"batch_size"`
	SequenceLength  int     `json:"sequence_length"`
}

// VRAMEstimation contains vRAM usage estimation results
type VRAMEstimation struct {
	ModelSize       float64                    `json:"model_size"`
	ContextSize     float64                    `json:"context_size"`
	TotalSize       float64                    `json:"total_size"`
	Quantization    string                     `json:"quantization"`
	ContextLength   int                        `json:"context_length"`
	Recommendations []VRAMRecommendation       `json:"recommendations"`
	Breakdown       VRAMBreakdown              `json:"breakdown"`
	Estimates       map[string]VRAMEstimateRow `json:"estimates"`
}

// VRAMRecommendation provides recommendations for optimal settings
type VRAMRecommendation struct {
	Quantization  string  `json:"quantization"`
	ContextLength int     `json:"context_length"`
	VRAMUsage     float64 `json:"vram_usage"`
	Description   string  `json:"description"`
}

// VRAMBreakdown provides detailed breakdown of vRAM usage
type VRAMBreakdown struct {
	ModelWeights float64 `json:"model_weights"`
	KVCache      float64 `json:"kv_cache"`
	Activations  float64 `json:"activations"`
	Overhead     float64 `json:"overhead"`
	Total        float64 `json:"total"`
}

// VRAMEstimateRow represents a single row in the vRAM estimation table
type VRAMEstimateRow struct {
	Quantization string             `json:"quantization"`
	BitsPerWeight float64           `json:"bits_per_weight"`
	Contexts     map[string]float64 `json:"contexts"` // context length -> vRAM usage
}

// ModelOperation represents an operation that can be performed on a model
type ModelOperation struct {
	Type        string                 `json:"type"`
	ModelName   string                 `json:"model_name"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Progress    *OperationProgress     `json:"progress,omitempty"`
	Error       string                 `json:"error,omitempty"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
}

// OperationProgress represents the progress of a long-running operation
type OperationProgress struct {
	Status      string  `json:"status"`
	Completed   int64   `json:"completed"`
	Total       int64   `json:"total"`
	Percentage  float64 `json:"percentage"`
	Message     string  `json:"message"`
}

// ModelSearchResult represents a search result for models
type ModelSearchResult struct {
	Models      []Model `json:"models"`
	Total       int     `json:"total"`
	Page        int     `json:"page"`
	PageSize    int     `json:"page_size"`
	Query       string  `json:"query"`
	SortBy      string  `json:"sort_by"`
	SortOrder   string  `json:"sort_order"`
}

// ModelFilter defines filters for model listing
type ModelFilter struct {
	Query         string   `json:"query"`
	Family        []string `json:"family"`
	Quantization  []string `json:"quantization"`
	SizeMin       int64    `json:"size_min"`
	SizeMax       int64    `json:"size_max"`
	ModifiedAfter *time.Time `json:"modified_after"`
	ModifiedBefore *time.Time `json:"modified_before"`
	Status        []string `json:"status"`
}

// ModelSort defines sorting options for model listing
type ModelSort struct {
	Field string `json:"field"` // name, size, modified, family, quantization
	Order string `json:"order"` // asc, desc
}

// PaginationOptions defines pagination parameters
type PaginationOptions struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}
