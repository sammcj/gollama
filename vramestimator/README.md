# LLM vRAM Estimator

This Golang package provides functionality to estimate vRAM requirements for Large Language Models (LLMs). It calculates vRAM usage based on model parameters, quantisation settings, and context length.

## Features

- Calculate vRAM usage for a given model configuration
- Determine maximum context length for a given vRAM constraint
- Find the best quantisation setting for a given vRAM and context constraint
- Support for different k/v cache quantisation options (fp16, q8_0, q4_0)

## Installation

To use this package in your Golang project, you can install it using `go get`:

```shell
go get github.com/sammcj/gollama/vramestimator
```

## Usage as a Package

Here's an example of how to use the vRAM estimator in your Golang application:

```go
package main

import (
  "fmt"
  "log"
  "github.com/sammcj/gollama/vramestimator"
)

func main() {
  modelID := "NousResearch/Hermes-2-Theta-Llama-3-8B"
  accessToken := "" // Your Hugging Face access token, if needed

  // Calculate VRAM usage with q4_0 k/v cache quantisation
  vram, err := vramestimator.CalculateVRAM(modelID, 5.0, 0, vramestimator.KVCacheQ4_0, accessToken)
  if err != nil {
    log.Fatalf("Error calculating VRAM: %v", err)
  }
  fmt.Printf("VRAM required (q4_0 k/v cache): %.2f GB\n", vram)

  // Calculate maximum context with q8_0 k/v cache quantisation
  context, err := vramestimator.CalculateContext(modelID, 6, 8, vramestimator.KVCacheQ8_0, accessToken)
  if err != nil {
    log.Fatalf("Error calculating context: %v", err)
  }
  fmt.Printf("Maximum context (q8_0 k/v cache): %d\n", context)

  // Calculate best BPW with full fp16 k/v cache
  bpw, err := vramestimator.CalculateBPW(modelID, 6, 0, vramestimator.KVCacheFP16, "gguf", accessToken)
  if err != nil {
    log.Fatalf("Error calculating BPW: %v", err)
  }
  fmt.Printf("Best BPW (fp16 k/v cache): %v\n", bpw)
}
```

## Command-line Usage

To use the vRAM estimator as a command-line tool, you'll need to create a `main.go` file that uses the package and provides a command-line interface. Here's an example of how to implement this:

```go
package main

import (
  "flag"
  "fmt"
  "log"
  "github.com/sammcj/gollama/vramestimator"
)

func main() {
  modelID := flag.String("model", "", "Model ID (required)")
  mode := flag.String("mode", "vram", "Mode: vram, context, or bpw")
  bpw := flag.Float64("bpw", 5.0, "Bits per weight")
  memory := flag.Float64("memory", 0, "Available memory in GB")
  context := flag.Int("context", 0, "Context length")
  kvCache := flag.String("kvcache", "fp16", "KV cache quantisation: fp16, q8_0, or q4_0")
  quantType := flag.String("quanttype", "gguf", "Quantisation type: gguf or exl2")
  accessToken := flag.String("token", "", "Hugging Face access token")

  flag.Parse()

  if *modelID == "" {
    log.Fatal("Model ID is required")
  }

  var kvCacheQuant vramestimator.KVCacheQuantisation
  switch *kvCache {
  case "fp16":
    kvCacheQuant = vramestimator.KVCacheFP16
  case "q8_0":
    kvCacheQuant = vramestimator.KVCacheQ8_0
  case "q4_0":
    kvCacheQuant = vramestimator.KVCacheQ4_0
  default:
    log.Fatalf("Invalid KV cache quantisation: %s", *kvCache)
  }

  switch *mode {
  case "vram":
    vram, err := vramestimator.CalculateVRAM(*modelID, *bpw, *context, kvCacheQuant, *accessToken)
    if err != nil {
      log.Fatalf("Error calculating VRAM: %v", err)
    }
    fmt.Printf("VRAM required: %.2f GB\n", vram)
  case "context":
    maxContext, err := vramestimator.CalculateContext(*modelID, *memory, *bpw, kvCacheQuant, *accessToken)
    if err != nil {
      log.Fatalf("Error calculating context: %v", err)
    }
    fmt.Printf("Maximum context: %d\n", maxContext)
  case "bpw":
    bestBPW, err := vramestimator.CalculateBPW(*modelID, *memory, *context, kvCacheQuant, *quantType, *accessToken)
    if err != nil {
      log.Fatalf("Error calculating BPW: %v", err)
    }
    fmt.Printf("Best BPW: %v\n", bestBPW)
  default:
    log.Fatalf("Invalid mode: %s", *mode)
  }
}
```

After creating this file, you can build the command-line tool:

```shell
go build -o vram-estimator
```

Now you can use the tool from the command line:

```shell
# Calculate VRAM usage
./vram-estimator -model NousResearch/Hermes-2-Theta-Llama-3-8B -mode vram -bpw 5.0 -kvcache q4_0

# Calculate maximum context
./vram-estimator -model NousResearch/Hermes-2-Theta-Llama-3-8B -mode context -memory 6 -bpw 8 -kvcache q8_0

# Calculate best BPW
./vram-estimator -model NousResearch/Hermes-2-Theta-Llama-3-8B -mode bpw -memory 6 -kvcache fp16 -quanttype gguf
```

## How It Works

The vRAM estimator works by:

1. Fetching the model configuration from Hugging Face (if not cached locally)
2. Calculating the memory requirements for model parameters, activations, and KV cache
3. Adjusting calculations based on the specified quantisation settings
4. Performing binary and linear searches to optimize for context length or quantisation settings

The package uses various formulas and heuristics to estimate vRAM usage based on model architecture, quantisation, and runtime behavior.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

- Copyright Sam McLeod
- This project is licensed under the MIT License.
