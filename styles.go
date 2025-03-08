// styles.go contains the styles used to render the list view.
package main

import (
	"math"
	"strconv"

	"github.com/charmbracelet/lipgloss"
)

const (
	// Define minimum column widths
	minNameWidth     = 14
	minSizeWidth     = 10
	minQuantWidth    = 10
	minModifiedWidth = 10
	minIDWidth       = 10
	minFamilyWidth   = 14
)

var (
	// Define neon colours for different model families
	familyColours = map[string]lipgloss.Color{
		"llama":       lipgloss.Color("#FF1493"),
		"alpaca":      lipgloss.Color("#FF00FF"),
		"command-r":   lipgloss.Color("#FB79B4"),
		"starcoder2":  lipgloss.Color("#EE82EE"),
		"starcoder":   lipgloss.Color("#DD40DD"),
		"gemma":       lipgloss.Color("#A224AA"),
		"qwen2":       lipgloss.Color("#AAE"),
		"phi":         lipgloss.Color("#554FFF"),
		"granite":     lipgloss.Color("#BFBBBB"),
		"deepseek":    lipgloss.Color("#06AFFF"),
		"deepseek2":   lipgloss.Color("#60BFFF"),
		"vicuna":      lipgloss.Color("#00CED1"),
		"bert":        lipgloss.Color("#FF7A00"),
		"nomic-bert":  lipgloss.Color("#FF8C00"),
		"nomic":       lipgloss.Color("#FFD700"),
		"qwen":        lipgloss.Color("#7FFF00"),
		"placeholder": lipgloss.Color("#554AAF"),
	}

	// Define colour gradients
	synthGradient = []string{
		"#DDA0DD", "#DA70D6", "#BA55D3", "#9932CC", "#9400D3", "#8A2BE2",
		"#9400D3", "#9932CC", "#BA48D3", "#DA70D6", "#DDA0DD", "#EE82EE",
		"#FF00FF", "#FF0000",
	}
)

func quantColour(quant string) lipgloss.Color {
	quantMap := map[string]int{
		"IQ1_XXS": 0, "IQ1_XS": 0, "IQ1_S": 0, "IQ1_NL": 0,
		"IQ1_M": 0, "IQ1_L": 0, "Q2_K": 0,
		"Q2_K_S": 0, "Q2_K_M": 0, "Q2_0": 0, "Q2_K_L": 1,
		"Q3_0": 1, "IQ2_XXS": 2, "Q3_K_S": 2, "Q2_L": 1,
		"IQ2_XS": 3, "IQ2_S": 3, "IQ2_NL": 3, "IQ2_M": 3,
		"Q3_K_M": 4, "Q3_K_L": 4, "Q4_0": 5,
		"IQ3_XXS": 5, "IQ3_XS": 5, "IQ3_NL": 5, "IQ3_S": 6,
		"Q4_K_S": 6, "Q4_1": 6, "IQ4_XXS": 6, "Q4_K_M": 7,
		"IQ4_XS": 7, "IQ4_S": 8, "IQ4_NL": 7, "Q4_K_L": 8,
		"Q5_K_S": 8, "Q5_K_M": 9, "Q5_1": 9, "Q5_K_L": 10,
		"Q6_0": 11, "Q6_1": 11, "Q6_K": 11, "Q6_K_L": 11,
		"Q8": 12, "Q8_0": 12, "Q8_K": 12, "Q8_K_L": 12,
		"FP16": 13, "F16": 13, "F32": 15, "FP32": 15,
	}

	index, exists := quantMap[quant]
	if !exists {
		index = 0 // Default to lightest if unknown quant
	}
	if index >= len(synthGradient) {
		index = len(synthGradient) - 1 // Use the last valid index
	}
	return lipgloss.Color(synthGradient[index])
}

func sizeColour(size float64) lipgloss.Color {
	index := int(math.Log10(size+1) * 2.5)
	if index >= len(synthGradient) {
		index = len(synthGradient) - 1
	}
	return lipgloss.Color(synthGradient[index])
}

func paramSizeColour(paramSize string) lipgloss.Color {
	// Extract the numeric part from parameter size strings like "7.6B", "32B", etc.
	if paramSize == "" {
		return lipgloss.Color(synthGradient[0])
	}

	// Remove the "B" suffix if present
	numStr := paramSize
	if paramSize[len(paramSize)-1] == 'B' {
		numStr = paramSize[:len(paramSize)-1]
	}

	// Parse the numeric part
	size, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		// Default to first color if parsing fails
		return lipgloss.Color(synthGradient[0])
	}

	// Use logarithmic scale similar to sizeColour but adjusted for parameter sizes
	// Parameter sizes typically range from 1B to 100B+
	index := int(math.Log10(size+1) * 3)
	if index >= len(synthGradient) {
		index = len(synthGradient) - 1
	}
	return lipgloss.Color(synthGradient[index])
}

func familyColour(family string, index int) lipgloss.Color {
	colour, exists := familyColours[family]
	if !exists {
		// Pick the colour closest matching part of the family name
		for i := 0; i < len(family); i++ {
			if colour, exists = familyColours[family[i:]]; exists {
				break
			}
			if colour, exists = familyColours[family[:len(family)-i]]; exists {
				break
			}
		}
		// If no colour found, default to synthGradient
		if !exists {
			colour = lipgloss.Color(synthGradient[index%len(synthGradient)])
		}
	}
	return colour
}
