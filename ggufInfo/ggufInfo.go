package ggufInfo

import (
	"fmt"

	gguf "github.com/gpustack/gguf-parser-go"
)

func GgufInfo(filePath string) error {
	// Parse the GGUF file
	ggufFile, err := gguf.ParseGGUFFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to parse GGUF file: %w", err)
	}

	// Get metadata
	metadata := ggufFile.Metadata()

	// Get architecture
	architecture := ggufFile.Architecture()

	// Get tokenizer
	tokenizer := ggufFile.Tokenizer()

	// Display information
	fmt.Println("GGUF Model Information:")
	fmt.Printf("File: %s\n\n", filePath)

	fmt.Println("Metadata:")
	fmt.Printf("  Type: %s\n", metadata.Type)
	fmt.Printf("  Name: %s\n", metadata.Name)
	fmt.Printf("  Architecture: %s\n", metadata.Architecture)
	fmt.Printf("  Quantization: %s\n", metadata.FileType)
	fmt.Printf("  Size: %s\n", gguf.GGUFBytesScalar(metadata.Size))
	fmt.Printf("  Parameters: %d\n", metadata.Parameters)
	fmt.Printf("  Bits per Weight: %.2f\n", metadata.BitsPerWeight)
  fmt.Printf("  File Type: %s\n", metadata.FileType)
  fmt.Printf("  Quantization Version: %d\n\n", metadata.QuantizationVersion)

	fmt.Println("Architecture:")
	fmt.Printf("  Maximum Context Length: %d\n", architecture.MaximumContextLength)
	fmt.Printf("  Embedding Length: %d\n", architecture.EmbeddingLength)
	fmt.Printf("  Block Count: %d\n", architecture.BlockCount)
	fmt.Printf("  Feed Forward Length: %d\n", architecture.FeedForwardLength)
	fmt.Printf("  Attention Head Count: %d\n\n", architecture.AttentionHeadCount)
  fmt.Println("Type: ", architecture.Type)
  fmt.Println("ClipHasTextEncoder: ", architecture.ClipHasTextEncoder)
  fmt.Println("EmbeddingGQA: ", architecture.EmbeddingGQA)
  fmt.Println("VocabularyLength: ", architecture.VocabularyLength)


	fmt.Println("Tokenizer:")
	fmt.Printf("  Model: %s\n", tokenizer.Model)
	fmt.Printf("  Tokens Length: %d\n", tokenizer.TokensLength)
	fmt.Printf("  BOS Token ID: %d\n", tokenizer.BOSTokenID)
	fmt.Printf("  EOS Token ID: %d\n", tokenizer.EOSTokenID)

  // print the template if it has one

	return nil
}

