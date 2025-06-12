// model.go contains the Model struct which is used to represent the data for each item in the list view.
package main

import (
	"fmt"
	"time"
)

type Model struct {
	Name              string
	ID                string
	Size              float64
	QuantizationLevel string
	Modified          time.Time
	Selected          bool
	Family            string
	ParameterSize     string
}

// EnhancedModelInfo contains detailed information from the Ollama show API
type EnhancedModelInfo struct {
	ParameterSize      string   `json:"parameter_size"`
	QuantizationLevel  string   `json:"quantization_level"`
	Format             string   `json:"format"`
	Family             string   `json:"family"`
	ContextLength      int64    `json:"context_length"`
	EmbeddingLength    int64    `json:"embedding_length"`
	RopeDimensionCount int64    `json:"rope_dimension_count"`
	RopeFreqBase       float64  `json:"rope_freq_base"`
	VocabSize          int64    `json:"vocab_size"`
	Capabilities       []string `json:"capabilities"`
}

func (m Model) SelectedStr() string {
	if m.Selected {
		return "X"
	}
	return ""
}

func (m Model) Description() string {
	paramSizeStr := ""
	if m.ParameterSize != "" {
		paramSizeStr = fmt.Sprintf(", Parameters: %s", m.ParameterSize)
	}
	return fmt.Sprintf("ID: %s, Size: %.2f GB, Quant: %s%s, Modified: %s", m.ID, m.Size, m.QuantizationLevel, paramSizeStr, m.Modified.Format("2006-01-02"))
}

func (m Model) FilterValue() string {
	return m.Name
}
