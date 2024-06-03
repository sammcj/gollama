package main

import (
  "fmt"
  "math"

  "github.com/charmbracelet/lipgloss"
)

var (
  // Define neon colours for different model families
  familyColours = map[string]lipgloss.Color{
    "alpaca":     lipgloss.Color("#FF00FF"),
    "bert":       lipgloss.Color("#FF40CB"),
    "command-r":  lipgloss.Color("#FF69B4"),
    "gemma":      lipgloss.Color("#FFB6C1"),
    "llama":      lipgloss.Color("#FF1493"),
    "nomic-bert": lipgloss.Color("#FF8C00"),
    "phi2":       lipgloss.Color("#554AAF"),
    "phi3":       lipgloss.Color("#554FFF"),
    "qwen":       lipgloss.Color("#7FFF00"),
    "qwen2":      lipgloss.Color("#AAE"),
    "starcoder":  lipgloss.Color("#DDA0DD"),
    "starcoder2": lipgloss.Color("#EE82EE"),
    "vicuna":     lipgloss.Color("#00CED1"),
    "granite":    lipgloss.Color("#00BFFF"),
  }

  // Define colour gradients
  synthGradient = []string{
    "#DDA0DD", "#DA70D6", "#BA55D3", "#9932CC", "#9400D3", "#8A2BE2",
    "#9400D3", "#9932CC", "#BA55D3", "#DA70D6", "#DDA0DD", "#EE82EE",
    "#FF00FF", "#FF0000",
  }
)

func quantColour(quant string) lipgloss.Color {
  quantMap := map[string]int{
    "IQ1_XXS": 0, "IQ1_XS": 0, "IQ1_S": 0, "IQ1_NL": 0,
    "Q2_K": 0, "Q2_K_S": 0, "Q2_K_M": 0, "Q2_K_L": 0,
    "Q3_0": 1, "IQ2_XXS": 2, "Q3_K_S": 2,
    "IQ2_XS": 3, "IQ2_S": 3, "IQ2_NL": 3,
    "Q3_K_M": 4, "Q3_K_L": 4,
    "Q4_0": 5, "IQ3_XXS": 5, "IQ3_XS": 5, "IQ3_NL": 5, "IQ3_S": 6,
    "Q4_K_S": 6, "Q4_1": 6, "IQ4_XXS": 6, "Q4_K_M": 7,
    "IQ4_XS": 7, "IQ4_S": 8, "IQ4_NL": 7, "Q4_K_L": 8,
    "Q5_K_S": 8, "Q5_K_M": 9, "Q5_1": 9, "Q5_K_L": 10,
    "Q6_0": 11, "Q6_1": 11, "Q6_K": 11,
    "Q8": 12, "Q8_0": 12, "Q8_K": 12,
    "FP16": 13, "F16": 13,
  }

  index, exists := quantMap[quant]
  if !exists {
    index = 0 // Default to lightest if unknown quant
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

func familyColour(family string, index int) lipgloss.Color {
  colour, exists := familyColours[family]
  if !exists {
    colour = lipgloss.Color(fmt.Sprintf("#%02X%02X%02X", 10+index%190, 10+index%190, 10+index%190))
  }
  return colour
}
