// File: vramestimator/table.go

package vramestimator

import (
	"bytes"
	"fmt"
	"log"
	"sort"

	"github.com/charmbracelet/lipgloss"
	"github.com/olekukonko/tablewriter"
)

func GenerateQuantTable(modelID string, accessToken string, fitsVRAM float64, ollamaModelInfo *OllamaModelInfo) (QuantResultTable, error) {
	if fitsVRAM == 0 {
		var err error
		fitsVRAM, err = GetAvailableMemory()
		if err != nil {
			log.Printf("Failed to get available memory: %v. Using default value.", err)
			fitsVRAM = 24 // Default to 24GB if we can't determine available memory
		}
		log.Printf("Using %.2f GB as available memory for VRAM estimation", fitsVRAM)
	}

	table := QuantResultTable{ModelID: modelID, FitsVRAM: fitsVRAM}
	contextSizes := []int{2048, 8192, 16384, 32768, 49152, 65536}

	if ollamaModelInfo == nil {
		_, err := GetModelConfig(modelID, accessToken)
		if err != nil {
			return QuantResultTable{}, err
		}
	}

	for quantType, bpw := range GGUFMapping {
		var result QuantResult
		result.QuantType = quantType
		result.BPW = bpw
		result.Contexts = make(map[int]ContextVRAM)

		for _, context := range contextSizes {
			vramFP16, err := CalculateVRAM(modelID, bpw, context, KVCacheFP16, accessToken, ollamaModelInfo)
			if err != nil {
				return QuantResultTable{}, err
			}
			vramQ8_0, err := CalculateVRAM(modelID, bpw, context, KVCacheQ8_0, accessToken, ollamaModelInfo)
			if err != nil {
				return QuantResultTable{}, err
			}
			vramQ4_0, err := CalculateVRAM(modelID, bpw, context, KVCacheQ4_0, accessToken, ollamaModelInfo)
			if err != nil {
				return QuantResultTable{}, err
			}
			result.Contexts[context] = ContextVRAM{
				VRAM:     vramFP16,
				VRAMQ8_0: vramQ8_0,
				VRAMQ4_0: vramQ4_0,
			}
		}
		table.Results = append(table.Results, result)
	}

	// Sort the results from lowest BPW to highest
	sort.Slice(table.Results, func(i, j int) bool {
		return table.Results[i].BPW < table.Results[j].BPW
	})

	return table, nil
}

func PrintFormattedTable(table QuantResultTable) string {
	var buf bytes.Buffer
	tw := tablewriter.NewWriter(&buf)

	// Set table header
	tw.SetHeader([]string{"Quant|Ctx", "BPW", "2K", "8K", "16K", "32K", "49K", "64K"})

	// Set table style
	tw.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	tw.SetCenterSeparator("|")
	tw.SetColumnSeparator("|")
	tw.SetRowSeparator("-")

	// Set header colour to bright white
	headerColours := make([]tablewriter.Colors, 8)
	for i := range headerColours {
		headerColours[i] = tablewriter.Colors{tablewriter.FgHiWhiteColor}
	}
	tw.SetHeaderColor(headerColours...)
	// set header row colours to bright white

	// Prepare data rows
	for _, result := range table.Results {
		row := []string{
			result.QuantType,
			fmt.Sprintf("%.2f", result.BPW),
		}

		// Add VRAM estimates for each context size
		contextSizes := []int{2048, 8192, 16384, 32768, 49152, 65536}
		for _, context := range contextSizes {
			vram := result.Contexts[context]

			fp16Str := getColouredVRAM(vram.VRAM, fmt.Sprintf("%.1f", vram.VRAM), table.FitsVRAM)

			if context >= 16384 {
				q8Str := getColouredVRAM(vram.VRAMQ8_0, fmt.Sprintf("%.1f", vram.VRAMQ8_0), table.FitsVRAM)
				q4Str := getColouredVRAM(vram.VRAMQ4_0, fmt.Sprintf("%.1f", vram.VRAMQ4_0), table.FitsVRAM)

				combinedStr := fmt.Sprintf("%s(%s,%s)", fp16Str, q8Str, q4Str)
				row = append(row, combinedStr)
			} else {
				combinedStr := fp16Str
				row = append(row, combinedStr)
			}
		}

		tw.Append(row)
	}

	// Render the table
	tw.Render()

	return lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff")).Render(fmt.Sprintf("📊 VRAM Estimation for Model: %s\n\n%s", table.ModelID, buf.String()))
}
