// helpers.go contains various helper functions used in the main application.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sammcj/gollama/config"
	"github.com/sammcj/gollama/logging"
	"github.com/sammcj/gollama/styles"

	"github.com/ollama/ollama/api"
	"golang.org/x/term"
)

func parseAPIResponse(resp *api.ListResponse) []Model {
	logging.DebugLogger.Println("Fetching models from API")

	models := make([]Model, len(resp.Models))
	for i, modelResp := range resp.Models {
		models[i] = Model{
			Name:              modelResp.Name,
			ID:                truncate(modelResp.Digest, 7),
			Size:              float64(modelResp.Size) / (1024 * 1024 * 1024), // Convert bytes to GB
			QuantizationLevel: modelResp.Details.QuantizationLevel,
			Family:            modelResp.Details.Family,
			Modified:          modelResp.ModifiedAt,
			ParameterSize:     modelResp.Details.ParameterSize,
		}
	}
	logging.DebugLogger.Println("Models:", models)
	return models
}

func normalizeSize(size float64) float64 {
	return size // Sizes are already in GB in the API response
}

// Constant for parameter size column width
const minParamSizeWidth = 10

func calculateColumnWidths(totalWidth int) (nameWidth, sizeWidth, quantWidth, modifiedWidth, idWidth, familyWidth, paramSizeWidth int) {
	// Calculate column widths
	nameWidth = int(0.40 * float64(totalWidth))
	sizeWidth = int(0.05 * float64(totalWidth))
	quantWidth = int(0.05 * float64(totalWidth))
	familyWidth = int(0.05 * float64(totalWidth))
	modifiedWidth = int(0.05 * float64(totalWidth))
	idWidth = int(0.02 * float64(totalWidth))
	paramSizeWidth = int(0.05 * float64(totalWidth))

	// Set the absolute minimum width for each column
	if nameWidth < minNameWidth {
		nameWidth = minNameWidth
	}
	if sizeWidth < minSizeWidth {
		sizeWidth = minSizeWidth
	}
	if quantWidth < minQuantWidth {
		quantWidth = minQuantWidth
	}
	if modifiedWidth < minModifiedWidth {
		modifiedWidth = minModifiedWidth
	}
	if idWidth < minIDWidth {
		idWidth = minIDWidth
	}
	if familyWidth < minFamilyWidth {
		familyWidth = minFamilyWidth
	}
	if paramSizeWidth < minParamSizeWidth {
		paramSizeWidth = minParamSizeWidth
	}

	// If the total width is less than the sum of the minimum column widths, adjust the name column width and make sure all columns are aligned
	if totalWidth < nameWidth+sizeWidth+quantWidth+familyWidth+modifiedWidth+idWidth+paramSizeWidth {
		nameWidth = totalWidth - sizeWidth - quantWidth - familyWidth - modifiedWidth - idWidth - paramSizeWidth
	}

	return
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

func calculateColumnWidthsTerminal() (nameWidth, sizeWidth, quantWidth, modifiedWidth, idWidth, familyWidth, paramSizeWidth int) {
	// use the terminal width to calculate column widths
	minWidth := 120

	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		logging.ErrorLogger.Println("Error getting terminal size:", err)
		width = minWidth
	}
	// make sure there's at least minWidth characters for each column
	if width < minWidth {
		width = minWidth
	}

	return calculateColumnWidths(width)
}

func listModels(models []Model) {
	// read the config file to see if the user wants to strip a string from the model name
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Println("Error loading config:", err)
		os.Exit(1)
	}

	if len(models) == 0 {
		fmt.Println("No models available to display.")
		return
	}

	stripString := cfg.StripString
	nameWidth, sizeWidth, quantWidth, modifiedWidth, idWidth, familyWidth, paramSizeWidth := calculateColumnWidthsTerminal()

	// Add extra spacing between columns
	colSpacing := 2
	longestNameAllowed := 60

	// Create the header with proper padding and alignment
	header := fmt.Sprintf("%-*s%-*s%-*s%-*s%-*s%-*s%-*s",
		nameWidth, "Name",
		sizeWidth+colSpacing, "Size",
		paramSizeWidth+colSpacing, "Params",
		quantWidth+colSpacing, "Quant",
		familyWidth+colSpacing, "Family",
		modifiedWidth+colSpacing, "Modified",
		idWidth, "ID")

	// if stripString is set, replace the model name with the stripped string
	if stripString != "" {
		for i, model := range models {
			models[i].Name = strings.Replace(model.Name, stripString, "", 1)
		}
	}

	// Prepare columns for padding
	var names, sizes, quants, families, modified, ids, paramSizes []string
	var longestName int
	for _, model := range models {
		if len(model.Name) > longestName {
			longestName = len(model.Name)
		}
		// truncate long names
		if len(model.Name) > longestNameAllowed {
			model.Name = model.Name[:longestNameAllowed] + "..."
		}
		names = append(names, model.Name)
		sizes = append(sizes, fmt.Sprintf("%.2fGB", model.Size))
		paramSizes = append(paramSizes, model.ParameterSize)
		quants = append(quants, model.QuantizationLevel)
		families = append(families, model.Family)
		modified = append(modified, model.Modified.Format("2006-01-02"))
		ids = append(ids, model.ID)
	}

	// Calculate maximum width for each column
	maxNameWidth := nameWidth
	maxSizeWidth := sizeWidth + colSpacing
	maxParamSizeWidth := paramSizeWidth + colSpacing
	maxQuantWidth := quantWidth + colSpacing
	maxFamilyWidth := familyWidth + colSpacing
	maxModifiedWidth := modifiedWidth + colSpacing
	maxIdWidth := idWidth

	// Pad columns to ensure alignment with calculated widths
	for i := range names {
		names[i] = fmt.Sprintf("%-*s", maxNameWidth, names[i])
		sizes[i] = fmt.Sprintf("%-*s", maxSizeWidth, sizes[i])
		paramSizes[i] = fmt.Sprintf("%-*s", maxParamSizeWidth, paramSizes[i])
		quants[i] = fmt.Sprintf("%-*s", maxQuantWidth, quants[i])
		families[i] = fmt.Sprintf("%-*s", maxFamilyWidth, families[i])
		modified[i] = fmt.Sprintf("%-*s", maxModifiedWidth, modified[i])
		// if the longest name is more than longestNameAllowed characters, don't display the model sha
		if longestName > longestNameAllowed {
			ids[i] = ""
			// remove the ID header
			header = fmt.Sprintf("%-*s%-*s%-*s%-*s%-*s%-*s",
				nameWidth, "Name",
				sizeWidth+colSpacing, "Size",
				paramSizeWidth+colSpacing, "Params",
				quantWidth+colSpacing, "Quant",
				familyWidth+colSpacing, "Family",
				modifiedWidth, "Modified")
		} else {
			ids[i] = fmt.Sprintf("%-*s", maxIdWidth, ids[i])
		}
	}

	// Print the header
	fmt.Println(styles.HeaderStyle().Render(header))

	modelList := []string{}

	for index, model := range models {
		name := styles.ItemNameStyle(index).Render(names[index])
		id := styles.ItemIDStyle().Render(ids[index])
		size := styles.SizeStyle(model.Size).Render(sizes[index])
		// Apply direct color based on parameter size
		var paramSize string
		if paramSizes[index] != "" {
			// Format the string first
			formattedParamSize := fmt.Sprintf("%-*s", maxParamSizeWidth, paramSizes[index])
			// Apply color directly using paramSizeColour
			paramSize = lipgloss.NewStyle().Foreground(paramSizeColour(paramSizes[index])).Render(formattedParamSize)
		} else {
			paramSize = fmt.Sprintf("%-*s", maxParamSizeWidth, paramSizes[index])
		}
		family := styles.FamilyStyle(model.Family).Render(families[index])
		quant := styles.QuantStyle(model.QuantizationLevel).Render(quants[index])
		modified := styles.ItemIDStyle().Render(modified[index])

		row := fmt.Sprintf("%-*s%-*s%-*s%-*s%-*s%-*s%-*s",
			maxNameWidth, name,
			maxSizeWidth, size,
			maxParamSizeWidth, paramSize,
			maxQuantWidth, quant,
			maxFamilyWidth, family,
			maxModifiedWidth, modified,
			maxIdWidth, id)
		modelList = append(modelList, row)
	}

	// Print the models with proper spacing
	for _, row := range modelList {
		fmt.Printf("%s\n", row)
	}
}
