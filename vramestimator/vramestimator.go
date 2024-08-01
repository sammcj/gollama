package vramestimator

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
)

// KVCacheQuantisation represents the quantisation type for the k/v context cache
type KVCacheQuantisation string

const (
	KVCacheFP16 KVCacheQuantisation = "fp16"
	KVCacheQ8_0 KVCacheQuantisation = "q8_0"
	KVCacheQ4_0 KVCacheQuantisation = "q4_0"
)

const (
	CUDASize = 500 * 1024 * 1024 // 500 MB
)

// GGUFMapping maps GGUF quantisation types to their corresponding bits per weight
var GGUFMapping = map[string]float64{
	"Q8_0":    8.5,
	"Q6_K":    6.59,
	"Q5_K_M":  5.69,
	"Q5_K_S":  5.54,
	"Q5_0":    5.54,
	"Q4_K_M":  4.85,
	"Q4_K_S":  4.58,
	"Q4_0":    4.55,
	"IQ4_NL":  4.5,
	"Q3_K_L":  4.27,
	"IQ4_XS":  4.25,
	"Q3_K_M":  3.91,
	"IQ3_M":   3.7,
	"IQ3_S":   3.5,
	"Q3_K_S":  3.5,
	"Q2_K":    3.35,
	"IQ3_XS":  3.3,
	"IQ3_XXS": 3.06,
	"IQ2_M":   2.7,
	"IQ2_S":   2.5,
	"IQ2_XS":  2.31,
	"IQ2_XXS": 2.06,
	"IQ1_S":   1.56,
}

// EXL2Options contains the EXL2 quantisation options
var EXL2Options []float64

func init() {
	for i := 6.0; i >= 2.0; i -= 0.05 {
		EXL2Options = append(EXL2Options, math.Round(i*100)/100)
	}
}

// ModelConfig represents the configuration of a model
type ModelConfig struct {
	NumParams             float64 `json:"num_params"`
	MaxPositionEmbeddings int     `json:"max_position_embeddings"`
	NumHiddenLayers       int     `json:"num_hidden_layers"`
	HiddenSize            int     `json:"hidden_size"`
	NumKeyValueHeads      int     `json:"num_key_value_heads"`
	NumAttentionHeads     int     `json:"num_attention_heads"`
	IntermediateSize      int     `json:"intermediate_size"`
	VocabSize             int     `json:"vocab_size"`
}


// BPWValues represents the bits per weight values for different components
type BPWValues struct {
	BPW         float64
	LMHeadBPW   float64
	KVCacheBPW  float64
}

// CalculateVRAMRaw calculates the raw VRAM usage
func CalculateVRAMRaw(config ModelConfig, bpwValues BPWValues, context int, numGPUs int, gqa bool) float64 {
	cudaSize := float64(CUDASize * numGPUs)
	paramsSize := config.NumParams * 1e9 * (bpwValues.BPW / 8)

	kvCacheSize := float64(context * 2 * config.NumHiddenLayers * config.HiddenSize) * (bpwValues.KVCacheBPW / 8)
	if gqa {
		kvCacheSize *= float64(config.NumKeyValueHeads) / float64(config.NumAttentionHeads)
	}

	bytesPerParam := bpwValues.BPW / 8
	lmHeadBytesPerParam := bpwValues.LMHeadBPW / 8

	headDim := float64(config.HiddenSize) / float64(config.NumAttentionHeads)
	attentionInput := bytesPerParam * float64(context * config.HiddenSize)

	q := bytesPerParam * float64(context) * headDim * float64(config.NumAttentionHeads)
	k := bytesPerParam * float64(context) * headDim * float64(config.NumKeyValueHeads)
	v := bytesPerParam * float64(context) * headDim * float64(config.NumKeyValueHeads)

	softmaxOutput := lmHeadBytesPerParam * float64(config.NumAttentionHeads * context)
	softmaxDropoutMask := float64(config.NumAttentionHeads * context)
	dropoutOutput := lmHeadBytesPerParam * float64(config.NumAttentionHeads * context)

	outProjInput := lmHeadBytesPerParam * float64(context * config.NumAttentionHeads) * headDim
	attentionDropout := float64(context * config.HiddenSize)

	attentionBlock := attentionInput + q + k + softmaxOutput + v + outProjInput + softmaxDropoutMask + dropoutOutput + attentionDropout

	mlpInput := bytesPerParam * float64(context * config.HiddenSize)
	activationInput := bytesPerParam * float64(context * config.IntermediateSize)
	downProjInput := bytesPerParam * float64(context * config.IntermediateSize)
	dropoutMask := float64(context * config.HiddenSize)
	mlpBlock := mlpInput + activationInput + downProjInput + dropoutMask

	layerNorms := bytesPerParam * float64(context * config.HiddenSize * 2)
	activationsSize := attentionBlock + mlpBlock + layerNorms

	outputSize := lmHeadBytesPerParam * float64(context * config.VocabSize)

	vramBits := cudaSize + paramsSize + activationsSize + outputSize + kvCacheSize

	return bitsToGB(vramBits)
}

// bitsToGB converts bits to gigabytes
func bitsToGB(bits float64) float64 {
	return bits / math.Pow(2, 30)
}

// DownloadFile downloads a file from a URL and saves it to the specified path
func DownloadFile(url, filePath string, headers map[string]string) error {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// GetModelConfig retrieves and parses the model configuration
func GetModelConfig(modelID, accessToken string) (ModelConfig, error) {
	baseDir := filepath.Join("cache", modelID)
	configPath := filepath.Join(baseDir, "config.json")
	indexPath := filepath.Join(baseDir, "model.safetensors.index.json")

	configURL := fmt.Sprintf("https://huggingface.co/%s/raw/main/config.json", modelID)
	indexURL := fmt.Sprintf("https://huggingface.co/%s/raw/main/model.safetensors.index.json", modelID)

	headers := make(map[string]string)
	if accessToken != "" {
		headers["Authorization"] = "Bearer " + accessToken
	}

	if err := DownloadFile(configURL, configPath, headers); err != nil {
		return ModelConfig{}, err
	}

	if err := DownloadFile(indexURL, indexPath, headers); err != nil {
		return ModelConfig{}, err
	}

	configFile, err := os.ReadFile(configPath)
	if err != nil {
		return ModelConfig{}, err
	}

	indexFile, err := os.ReadFile(indexPath)
	if err != nil {
		return ModelConfig{}, err
	}

	var config ModelConfig
	if err := json.Unmarshal(configFile, &config); err != nil {
		return ModelConfig{}, err
	}

	var index struct {
		Metadata struct {
			TotalSize float64 `json:"total_size"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(indexFile, &index); err != nil {
		return ModelConfig{}, err
	}

	config.NumParams = index.Metadata.TotalSize / 2 / 1e9

	return config, nil
}

// ParseBPW parses the BPW value
func ParseBPW(bpw string) float64 {
	if val, ok := GGUFMapping[bpw]; ok {
		return val
	}
	return 0
}

// GetBPWValues calculates the BPW values based on the input
func GetBPWValues(bpw float64, kvCacheQuant KVCacheQuantisation) BPWValues {
	var lmHeadBPW, kvCacheBPW float64

	if bpw > 6.0 {
		lmHeadBPW = 8.0
	} else {
		lmHeadBPW = 6.0
	}

	switch kvCacheQuant {
	case KVCacheFP16:
		kvCacheBPW = 16
	case KVCacheQ8_0:
		kvCacheBPW = 8
	case KVCacheQ4_0:
		kvCacheBPW = 4
	default:
		kvCacheBPW = 16 // Default to fp16 if not specified
	}

	return BPWValues{
		BPW:        bpw,
		LMHeadBPW:  lmHeadBPW,
		KVCacheBPW: kvCacheBPW,
	}
}

// CalculateVRAM calculates the VRAM usage for a given model and configuration
func CalculateVRAM(modelID string, bpw float64, context int, kvCacheQuant KVCacheQuantisation, accessToken string) (float64, error) {
	config, err := GetModelConfig(modelID, accessToken)
	if err != nil {
		return 0, err
	}

	bpwValues := GetBPWValues(bpw, kvCacheQuant)

	if context == 0 {
		context = config.MaxPositionEmbeddings
	}

	vram := CalculateVRAMRaw(config, bpwValues, context, 1, true)
	return math.Round(vram*100) / 100, nil
}

// CalculateContext calculates the maximum context for a given memory constraint
func CalculateContext(modelID string, memory, bpw float64, kvCacheQuant KVCacheQuantisation, accessToken string) (int, error) {
	config, err := GetModelConfig(modelID, accessToken)
	if err != nil {
		return 0, err
	}

	minContext := 2048
	maxContext := config.MaxPositionEmbeddings

	low, high := minContext, maxContext
	for low < high {
		mid := (low + high + 1) / 2
		vram, err := CalculateVRAM(modelID, bpw, mid, kvCacheQuant, accessToken)
		if err != nil {
			return 0, err
		}
		if vram > memory {
			high = mid - 1
		} else {
			low = mid
		}
	}

	context := low
	for context <= maxContext {
		vram, err := CalculateVRAM(modelID, bpw, context, kvCacheQuant, accessToken)
		if err != nil {
			return 0, err
		}
		if vram >= memory {
			break
		}
		context += 100
	}

	return context - 100, nil
}

// CalculateBPW calculates the best BPW for a given memory and context constraint
func CalculateBPW(modelID string, memory float64, context int, kvCacheQuant KVCacheQuantisation, quantType string, accessToken string) (interface{}, error) {
	switch quantType {
	case "exl2":
		for _, bpw := range EXL2Options {
			vram, err := CalculateVRAM(modelID, bpw, context, kvCacheQuant, accessToken)
			if err != nil {
				return nil, err
			}
			if vram < memory {
				return bpw, nil
			}
		}
	case "gguf":
		for name, bpw := range GGUFMapping {
			vram, err := CalculateVRAM(modelID, bpw, context, kvCacheQuant, accessToken)
			if err != nil {
				return nil, err
			}
			if vram < memory {
				return name, nil
			}
		}
	default:
		return nil, fmt.Errorf("invalid quantisation type: %s", quantType)
	}
	return nil, nil
}
