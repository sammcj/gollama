// helpers.go contains various helper functions used in the main application.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/sammcj/gollama/config"
	"github.com/sammcj/gollama/logging"

	"github.com/charmbracelet/lipgloss"
	"github.com/ollama/ollama/api"
	"golang.org/x/term"
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

func calculateColumnWidths(totalWidth int) (nameWidth, sizeWidth, quantWidth, modifiedWidth, idWidth, familyWidth int) {
	// Calculate column widths
	nameWidth = int(0.45 * float64(totalWidth))
	sizeWidth = int(0.05 * float64(totalWidth))
	quantWidth = int(0.05 * float64(totalWidth))
	familyWidth = int(0.05 * float64(totalWidth))
	modifiedWidth = int(0.05 * float64(totalWidth))
	idWidth = int(0.02 * float64(totalWidth))

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

	// If the total width is less than the sum of the minimum column widths, adjust the name column width and make sure all columns are aligned
	if totalWidth < nameWidth+sizeWidth+quantWidth+familyWidth+modifiedWidth+idWidth {
		nameWidth = totalWidth - sizeWidth - quantWidth - familyWidth - modifiedWidth - idWidth
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

func calculateColumnWidthsTerminal() (nameWidth, sizeWidth, quantWidth, modifiedWidth, idWidth, familyWidth int) {
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

	stripString := cfg.StripString
	nameWidth, sizeWidth, quantWidth, modifiedWidth, idWidth, familyWidth := calculateColumnWidthsTerminal()

	// align the header with the columns (length of stripString is subtracted from the name width to ensure alignment with the other columns)
	header := fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s %-*s", nameWidth-len(stripString), "Name", sizeWidth, "Size", quantWidth, "Quant", familyWidth, "Family", modifiedWidth, "Modified", idWidth, "ID")

	// if stripString is set, replace the model name with the stripped string
	if stripString != "" {
		for i, model := range models {
			models[i].Name = strings.Replace(model.Name, stripString, "", 1)
		}
	}

	// Prepare columns for padding
	var names, sizes, quants, families, modifieds, ids []string
	for _, model := range models {
		names = append(names, model.Name)
		sizes = append(sizes, fmt.Sprintf("%.2fGB", model.Size))
		quants = append(quants, model.QuantizationLevel)
		families = append(families, model.Family)
		modifieds = append(modifieds, model.Modified.Format("2006-01-02"))
		ids = append(ids, model.ID)
	}

	// Pad columns to ensure alignment
	names = padColumn(names)
	sizes = padColumn(sizes)
	quants = padColumn(quants)
	families = padColumn(families)
	modifieds = padColumn(modifieds)
	ids = padColumn(ids)

	// Print the header
	fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Render(header))

	for index, model := range models {
		// colourise the model properties
		nameColours := []lipgloss.Color{lipgloss.Color("#FFFFFF"), lipgloss.Color("#818BA9")}
		name := lipgloss.NewStyle().Foreground(nameColours[index%len(nameColours)]).Render(names[index])
		id := lipgloss.NewStyle().Foreground(lipgloss.Color("254")).Faint(true).Render(ids[index])
		size := lipgloss.NewStyle().Foreground(sizeColour(model.Size)).Render(sizes[index])
		family := lipgloss.NewStyle().Foreground(familyColour(model.Family, 0)).Render(families[index])
		quant := lipgloss.NewStyle().Foreground(quantColour(model.QuantizationLevel)).Render(quants[index])
		modified := lipgloss.NewStyle().Foreground(lipgloss.Color("254")).Render(modifieds[index])

		row := fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s %-*s", nameWidth, name, sizeWidth, size, quantWidth, quant, familyWidth, family, modifiedWidth, modified, idWidth, id)
		fmt.Println(row)
	}
}

// padColumn takes a slice of strings and pads them with spaces to the maximum width of all the values in that column.
func padColumn(column []string) []string {
	max := 0
	for _, value := range column {
		if len(value) > max {
			max = len(value)
		}
	}

	paddedColumn := make([]string, len(column))
	for i, value := range column {
		padding := strings.Repeat(" ", max-len(value))
		paddedColumn[i] = wrapText(value+padding, max)
	}
	return paddedColumn
}
