package vramestimator

import (
	"fmt"
	"strings"
)

func EstimateVRAM(
	modelName string,
	contextSize int,
	kvCacheQuant KVCacheQuantisation,
	availableVRAM float64,
	quantLevel string,
) (*VRAMEstimation, error) {
	var ollamaModelInfo *OllamaModelInfo
	var err error

	// Check if the modelName is an Ollama model
	if strings.Contains(modelName, ":") {
		ollamaModelInfo, err = FetchOllamaModelInfo("", modelName) // Assuming default API URL
		if err != nil {
			return nil, fmt.Errorf("error fetching Ollama model info: %v", err)
		}
	}

	bpw, err := ParseBPWOrQuant(quantLevel)
	if err != nil {
		return nil, err
	}

	if contextSize == 0 {
		if ollamaModelInfo != nil {
			contextSize = ollamaModelInfo.ModelInfo.ContextLength
		} else {
			config, err := GetModelConfig(modelName, "")
			if err != nil {
				return nil, err
			}
			contextSize = config.MaxPositionEmbeddings
		}
	}

	estimatedVRAM, err := CalculateVRAM(modelName, bpw, contextSize, kvCacheQuant, "", ollamaModelInfo)
	if err != nil {
		return nil, err
	}

	maxContextSize, err := CalculateContext(modelName, availableVRAM, bpw, kvCacheQuant, "", ollamaModelInfo)
	if err != nil {
		return nil, err
	}

	recommendedQuant, err := CalculateBPW(modelName, availableVRAM, contextSize, kvCacheQuant, "gguf", "", ollamaModelInfo)
	if err != nil {
		return nil, err
	}

	return &VRAMEstimation{
		ModelName:        modelName,
		ContextSize:      contextSize,
		KVCacheQuant:     kvCacheQuant,
		AvailableVRAM:    availableVRAM,
		QuantLevel:       quantLevel,
		EstimatedVRAM:    estimatedVRAM,
		FitsAvailable:    estimatedVRAM <= availableVRAM,
		MaxContextSize:   maxContextSize,
		RecommendedQuant: recommendedQuant.(string),
	}, nil
}
