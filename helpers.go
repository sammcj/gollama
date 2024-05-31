package main

import (
	"gollama/logging"

	"github.com/charmbracelet/lipgloss"
	"github.com/ollama/ollama/api"
)

func parseAPIResponse(resp *api.ListResponse) []Model {
	models := make([]Model, len(resp.Models))
	for i, modelResp := range resp.Models {
		models[i] = Model{
			Name:              lipgloss.NewStyle().Foreground(lipgloss.Color("white")).Render(modelResp.Name),
			ID:                truncate(modelResp.Digest, 7),                  // Truncate the ID
			Size:              float64(modelResp.Size) / (1024 * 1024 * 1024), // Convert bytes to GB
			QuantizationLevel: modelResp.Details.QuantizationLevel,
			Family:            modelResp.Details.Family,
			Modified:          modelResp.ModifiedAt,
			Selected:          false,
		}
	}
	return models
}

func normalizeSize(size float64) float64 {
	return size // Sizes are already in GB in the API response
}

func calculateColumnWidths(totalWidth int) (nameWidth, sizeWidth, quantWidth, modifiedWidth, idWidth int, familyWidth int) {
	// Calculate column widths
	nameWidth = int(0.45 * float64(totalWidth))
	sizeWidth = int(0.05 * float64(totalWidth))
	quantWidth = int(0.05 * float64(totalWidth))
	familyWidth = int(0.05 * float64(totalWidth))
	modifiedWidth = int(0.05 * float64(totalWidth))
	idWidth = int(0.02 * float64(totalWidth))

	// Set the absolute minimum width for each column
	if nameWidth < 20 {
		nameWidth = 20
	}
	if sizeWidth < 10 {
		sizeWidth = 10
	}
	if quantWidth < 5 {
		quantWidth = 5
	}
	if modifiedWidth < 10 {
		modifiedWidth = 10
	}
	if idWidth < 10 {
		idWidth = 10
	}
	if familyWidth < 14 {
		familyWidth = 14
	}

	// If the total width is less than the sum of the minimum column widths, adjust the name column width and make sure all columns are aligned
	if totalWidth < nameWidth+sizeWidth+quantWidth+familyWidth+modifiedWidth+idWidth {
		nameWidth = totalWidth - sizeWidth - quantWidth - familyWidth - modifiedWidth - idWidth
	}

	return
}

func getSelectedModels(models []Model) []Model {
	selectedModels := make([]Model, 0)
	for _, model := range models {
		if model.Selected {
			logging.DebugLogger.Printf("Model selected for deletion: %s\n", model.Name)
			selectedModels = append(selectedModels, model)
		}
	}
	return selectedModels
}

func removeModels(models []Model, selectedModels []Model) []Model {
	result := make([]Model, 0)
	for _, model := range models {
		found := false
		for _, selectedModel := range selectedModels {
			if model.Name == selectedModel.Name {
				found = true
				break
			}
		}
		if !found {
			result = append(result, model)
		}
	}
	return result
}

// truncate ensures the string fits within the specified width
func truncate(text string, width int) string {
	if len(text) > width {
		return text[:width]
	}
	return text
}

// wrapText ensures the text wraps to the next line if it exceeds the column width
func wrapText(text string, width int) string {
	var wrapped string
	for len(text) > width {
		wrapped += text[:width]
		text = text[width:] + " "
	}
	wrapped += text
	return wrapped
}
