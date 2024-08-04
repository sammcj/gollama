package vramestimator

import (
	"fmt"
	"math"
	"os"
)

func init() {
	for i := 6.0; i >= 2.0; i -= 0.05 {
		EXL2Options = append(EXL2Options, math.Round(i*100)/100)
	}

  ExampleUsage()
}

func ExampleUsage() {
	model := "llama3.1:8b-instruct-q6_K"
	availableVram := 12.0

	estimation, err := EstimateVRAM(
		model,
		0, // Use default context size
		KVCacheFP16,
		availableVram,
		"", // Use default quantisation level
	)
	if err != nil {
		fmt.Printf("Error estimating VRAM: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Model: %s\n", estimation.ModelName)
	fmt.Printf("Context Size: %d\n", estimation.ContextSize)
	fmt.Printf("Estimated VRAM: %.2f GB\n", estimation.EstimatedVRAM)
	fmt.Printf("Fits Available VRAM: %v\n", estimation.FitsAvailable)
	fmt.Printf("Max Context Size: %d\n", estimation.MaxContextSize)
	fmt.Printf("Recommended Quantization: %s\n", estimation.RecommendedQuant)
}
