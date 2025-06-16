package core

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sammcj/gollama/vramestimator"
)

// ListModels retrieves all available models from Ollama
func (s *GollamaService) ListModels() ([]Model, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	models, err := s.client.ListModels(s.ctx)
	if err != nil {
		s.logger.Errorf("Failed to list models: %v", err)
		return nil, err
	}

	// Emit event
	s.eventBus.Emit(Event{
		Type: "models_listed",
		Data: models,
		Time: time.Now(),
	})

	return models, nil
}

// GetModel retrieves detailed information about a specific model
func (s *GollamaService) GetModel(name string) (*Model, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	info, err := s.client.GetModel(s.ctx, name)
	if err != nil {
		s.logger.Errorf("Failed to get model %s: %v", name, err)
		return nil, err
	}

	return &info.Model, nil
}

// GetModelInfo retrieves comprehensive information about a specific model
func (s *GollamaService) GetModelInfo(name string) (*EnhancedModelInfo, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	info, err := s.client.GetModel(s.ctx, name)
	if err != nil {
		s.logger.Errorf("Failed to get model info for %s: %v", name, err)
		return nil, err
	}

	return info, nil
}

// DeleteModel removes a model from Ollama
func (s *GollamaService) DeleteModel(name string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	err := s.client.DeleteModel(s.ctx, name)
	if err != nil {
		s.logger.Errorf("Failed to delete model %s: %v", name, err)
		return err
	}

	s.logger.Infof("Successfully deleted model: %s", name)

	// Emit event
	s.eventBus.Emit(Event{
		Type: EventModelDeleted,
		Data: map[string]string{"name": name},
		Time: time.Now(),
	})

	return nil
}

// RunModel starts running a model
func (s *GollamaService) RunModel(name string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	err := s.client.RunModel(s.ctx, name)
	if err != nil {
		s.logger.Errorf("Failed to run model %s: %v", name, err)
		return err
	}

	s.logger.Infof("Successfully started model: %s", name)

	// Emit event
	s.eventBus.Emit(Event{
		Type: EventModelRunning,
		Data: map[string]string{"name": name},
		Time: time.Now(),
	})

	return nil
}

// UnloadModel unloads a specific model
func (s *GollamaService) UnloadModel(name string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	err := s.client.UnloadModel(s.ctx, name)
	if err != nil {
		s.logger.Errorf("Failed to unload model %s: %v", name, err)
		return err
	}

	s.logger.Infof("Successfully unloaded model: %s", name)

	// Emit event
	s.eventBus.Emit(Event{
		Type: EventModelStopped,
		Data: map[string]string{"name": name},
		Time: time.Now(),
	})

	return nil
}

// UnloadAllModels unloads all currently running models
func (s *GollamaService) UnloadAllModels() error {
	runningModels, err := s.GetRunningModels()
	if err != nil {
		return fmt.Errorf("failed to get running models: %w", err)
	}

	for _, model := range runningModels {
		if err := s.UnloadModel(model.Name); err != nil {
			s.logger.Errorf("Failed to unload model %s: %v", model.Name, err)
			// Continue with other models even if one fails
		}
	}

	return nil
}

// CopyModel creates a copy of a model with a new name
func (s *GollamaService) CopyModel(source, dest string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	err := s.client.CopyModel(s.ctx, source, dest)
	if err != nil {
		s.logger.Errorf("Failed to copy model from %s to %s: %v", source, dest, err)
		return err
	}

	s.logger.Infof("Successfully copied model from %s to %s", source, dest)

	// Emit event
	s.eventBus.Emit(Event{
		Type: EventModelCreated,
		Data: map[string]string{"source": source, "destination": dest},
		Time: time.Now(),
	})

	return nil
}

// PushModel uploads a model to a registry
func (s *GollamaService) PushModel(name string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	err := s.client.PushModel(s.ctx, name)
	if err != nil {
		s.logger.Errorf("Failed to push model %s: %v", name, err)
		return err
	}

	s.logger.Infof("Successfully pushed model: %s", name)
	return nil
}

// PullModel downloads a model from a registry
func (s *GollamaService) PullModel(name string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	err := s.client.PullModel(s.ctx, name)
	if err != nil {
		s.logger.Errorf("Failed to pull model %s: %v", name, err)
		return err
	}

	s.logger.Infof("Successfully pulled model: %s", name)

	// Emit event
	s.eventBus.Emit(Event{
		Type: EventModelCreated,
		Data: map[string]string{"name": name},
		Time: time.Now(),
	})

	return nil
}

// GetRunningModels retrieves currently running models
func (s *GollamaService) GetRunningModels() ([]RunningModel, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	models, err := s.client.GetRunningModels(s.ctx)
	if err != nil {
		s.logger.Errorf("Failed to get running models: %v", err)
		return nil, err
	}

	return models, nil
}

// EditModelfile opens the Modelfile for editing (placeholder for now)
func (s *GollamaService) EditModelfile(name string) error {
	s.logger.Infof("Edit Modelfile requested for model: %s", name)
	// This would integrate with the existing editor functionality
	// For now, just return success
	return nil
}

// EstimateVRAM calculates vRAM usage for a model
func (s *GollamaService) EstimateVRAM(model string, constraints VRAMConstraints) (*VRAMEstimation, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Use the existing vramestimator package
	// Generate quantisation table for the model
	table, err := vramestimator.GenerateQuantTable(model, constraints.AvailableVRAM, nil, constraints.ContextLength)
	if err != nil {
		s.logger.Errorf("Failed to estimate vRAM for model %s: %v", model, err)
		return nil, err
	}

	// Find the best quantisation for the given constraints
	var bestResult *vramestimator.QuantResult
	if constraints.Quantization != "" {
		// Look for specific quantisation
		for _, result := range table.Results {
			if strings.EqualFold(result.QuantType, constraints.Quantization) {
				bestResult = &result
				break
			}
		}
	}

	// If no specific quantisation found or requested, use the first one that fits
	if bestResult == nil && constraints.AvailableVRAM > 0 {
		for _, result := range table.Results {
			if contextVRAM, ok := result.Contexts[constraints.ContextLength]; ok {
				if contextVRAM.VRAM <= constraints.AvailableVRAM {
					bestResult = &result
					break
				}
			}
		}
	}

	// If still no result, use the first available
	if bestResult == nil && len(table.Results) > 0 {
		bestResult = &table.Results[0]
	}

	if bestResult == nil {
		return nil, fmt.Errorf("no vRAM estimation results available for model %s", model)
	}

	// Get vRAM usage for the specified context length
	contextVRAM, ok := bestResult.Contexts[constraints.ContextLength]
	if !ok {
		// Use the closest context length
		closestContext := 0
		for ctx := range bestResult.Contexts {
			if closestContext == 0 || abs(ctx-constraints.ContextLength) < abs(closestContext-constraints.ContextLength) {
				closestContext = ctx
			}
		}
		if closestContext > 0 {
			contextVRAM = bestResult.Contexts[closestContext]
		}
	}

	// Convert result to our VRAMEstimation format
	estimation := &VRAMEstimation{
		ModelSize:     contextVRAM.VRAM * 0.7, // Rough estimate: 70% for model weights
		ContextSize:   contextVRAM.VRAM * 0.3, // Rough estimate: 30% for context
		TotalSize:     contextVRAM.VRAM,
		Quantization:  bestResult.QuantType,
		ContextLength: constraints.ContextLength,
		Breakdown: VRAMBreakdown{
			ModelWeights: contextVRAM.VRAM * 0.7,
			KVCache:      contextVRAM.VRAM * 0.3,
			Total:        contextVRAM.VRAM,
		},
		Estimates: make(map[string]VRAMEstimateRow),
	}

	// Add all quantisation estimates
	for _, result := range table.Results {
		if contextVRAM, ok := result.Contexts[constraints.ContextLength]; ok {
			estimation.Estimates[result.QuantType] = VRAMEstimateRow{
				Quantization:  result.QuantType,
				BitsPerWeight: result.BPW,
				Contexts: map[string]float64{
					fmt.Sprintf("%dk", constraints.ContextLength/1024): contextVRAM.VRAM,
				},
			}
		}
	}

	return estimation, nil
}

// abs returns the absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// LinkToLMStudio creates a link to LM Studio (placeholder for now)
func (s *GollamaService) LinkToLMStudio(model string) error {
	s.logger.Infof("LM Studio link requested for model: %s", model)
	// This would integrate with the existing LM Studio linking functionality
	// For now, just return success
	return nil
}

// SearchModels searches for models based on filters
func (s *GollamaService) SearchModels(filter ModelFilter, sort ModelSort, pagination PaginationOptions) (*ModelSearchResult, error) {
	models, err := s.ListModels()
	if err != nil {
		return nil, err
	}

	// Apply filters
	filteredModels := s.applyFilters(models, filter)

	// Apply sorting
	s.applySorting(filteredModels, sort)

	// Apply pagination
	total := len(filteredModels)
	start := (pagination.Page - 1) * pagination.PageSize
	end := start + pagination.PageSize

	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	paginatedModels := filteredModels[start:end]

	return &ModelSearchResult{
		Models:    paginatedModels,
		Total:     total,
		Page:      pagination.Page,
		PageSize:  pagination.PageSize,
		Query:     filter.Query,
		SortBy:    sort.Field,
		SortOrder: sort.Order,
	}, nil
}

// applyFilters applies search filters to models
func (s *GollamaService) applyFilters(models []Model, filter ModelFilter) []Model {
	var filtered []Model

	for _, model := range models {
		// Apply query filter
		if filter.Query != "" {
			query := strings.ToLower(filter.Query)
			name := strings.ToLower(model.Name)
			family := strings.ToLower(model.Details.Family)

			if !strings.Contains(name, query) && !strings.Contains(family, query) {
				continue
			}
		}

		// Apply family filter
		if len(filter.Family) > 0 {
			found := false
			for _, family := range filter.Family {
				if strings.EqualFold(model.Details.Family, family) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Apply quantization filter
		if len(filter.Quantization) > 0 {
			found := false
			for _, quant := range filter.Quantization {
				if strings.EqualFold(model.Details.QuantizationLevel, quant) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Apply size filters
		if filter.SizeMin > 0 && model.Size < filter.SizeMin {
			continue
		}
		if filter.SizeMax > 0 && model.Size > filter.SizeMax {
			continue
		}

		// Apply date filters
		if filter.ModifiedAfter != nil && model.ModifiedAt.Before(*filter.ModifiedAfter) {
			continue
		}
		if filter.ModifiedBefore != nil && model.ModifiedAt.After(*filter.ModifiedBefore) {
			continue
		}

		// Apply status filter
		if len(filter.Status) > 0 {
			found := false
			for _, status := range filter.Status {
				if strings.EqualFold(model.Status, status) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		filtered = append(filtered, model)
	}

	return filtered
}

// applySorting applies sorting to models
func (s *GollamaService) applySorting(models []Model, sortBy ModelSort) {
	switch strings.ToLower(sortBy.Field) {
	case "name":
		if strings.ToLower(sortBy.Order) == "desc" {
			sort.Slice(models, func(i, j int) bool {
				return models[i].Name > models[j].Name
			})
		} else {
			sort.Slice(models, func(i, j int) bool {
				return models[i].Name < models[j].Name
			})
		}
	case "size":
		if strings.ToLower(sortBy.Order) == "desc" {
			sort.Slice(models, func(i, j int) bool {
				return models[i].Size > models[j].Size
			})
		} else {
			sort.Slice(models, func(i, j int) bool {
				return models[i].Size < models[j].Size
			})
		}
	case "modified":
		if strings.ToLower(sortBy.Order) == "desc" {
			sort.Slice(models, func(i, j int) bool {
				return models[i].ModifiedAt.After(models[j].ModifiedAt)
			})
		} else {
			sort.Slice(models, func(i, j int) bool {
				return models[i].ModifiedAt.Before(models[j].ModifiedAt)
			})
		}
	case "family":
		if strings.ToLower(sortBy.Order) == "desc" {
			sort.Slice(models, func(i, j int) bool {
				return models[i].Details.Family > models[j].Details.Family
			})
		} else {
			sort.Slice(models, func(i, j int) bool {
				return models[i].Details.Family < models[j].Details.Family
			})
		}
	case "quantization":
		if strings.ToLower(sortBy.Order) == "desc" {
			sort.Slice(models, func(i, j int) bool {
				return models[i].Details.QuantizationLevel > models[j].Details.QuantizationLevel
			})
		} else {
			sort.Slice(models, func(i, j int) bool {
				return models[i].Details.QuantizationLevel < models[j].Details.QuantizationLevel
			})
		}
	}
}

// HealthCheck verifies that the service and Ollama API are accessible
func (s *GollamaService) HealthCheck() error {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.client.HealthCheck(s.ctx)
}
