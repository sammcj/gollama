package vramestimator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/olekukonko/tablewriter"
	"github.com/sammcj/gollama/logging"
	"github.com/shirou/gopsutil/v3/mem"
)

// KVCacheQuantisation represents the quantisation type for the k/v context cache
type KVCacheQuantisation string

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
	BPW        float64
	LMHeadBPW  float64
	KVCacheBPW float64
}

// Update the QuantResult struct
type QuantResult struct {
	QuantType string
	BPW       float64
	Contexts  map[int]ContextVRAM
}

type ContextVRAM struct {
	VRAM     float64
	VRAMQ8_0 float64
	VRAMQ4_0 float64
}

// QuantResultTable represents a table of VRAM estimation results
type QuantResultTable struct {
	ModelID  string
	Results  []QuantResult
	FitsVRAM float64
}

const (
	KVCacheFP16 KVCacheQuantisation = "fp16"
	KVCacheQ8_0 KVCacheQuantisation = "q8_0"
	KVCacheQ4_0 KVCacheQuantisation = "q4_0"
)

const (
	CUDASize = 500 * 1024 * 1024 // 500 MB
)

var colourMap = []string{
	"#ff0000", // red
	"#00ff00", // green
}

// GGUFMapping maps GGUF quantisation types to their corresponding bits per weight
var GGUFMapping = map[string]float64{
	"Q8_0":    8.5,
	"Q6_K":    6.59,
	"Q5_K_L":  5.75,
	"Q5_K_M":  5.69,
	"Q5_K_S":  5.54,
	"Q5_0":    5.54,
	"Q4_K_L":  4.9,
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

var (
	modelConfigCache = make(map[string]ModelConfig)
	cacheMutex       sync.RWMutex
)

func init() {
	for i := 6.0; i >= 2.0; i -= 0.05 {
		EXL2Options = append(EXL2Options, math.Round(i*100)/100)
	}
}

func checkNVMLAvailable() bool {
	if runtime.GOOS == "darwin" {
		return false
	}
	if _, err := os.Stat("/usr/lib/libnvidia-ml.so"); err == nil {
		return true
	}
	return false
}

func GetSystemRAM() (float64, error) {
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return 0, fmt.Errorf("failed to get system memory info: %v", err)
	}

	totalRAM := float64(vmStat.Total) / 1024 / 1024 / 1024 // Convert to GB
	return totalRAM, nil
}

func GetAvailableMemory() (float64, error) {
	// will fix this soon
	// if checkNVMLAvailable() {
	// 	// Try to get CUDA
	// 	vram, err := cuda.GetCUDAVRAM()
	// 	if err == nil {
	// 		logging.InfoLogger.Printf("Using CUDA VRAM: %.2f GB", vram)
	// 		return vram, nil
	// 	}

	// 	// If CUDA is not available, fall back to system RAM
	// 	ram, err := GetSystemRAM()
	// 	if err != nil {
	// 		return 0, fmt.Errorf("failed to get system RAM: %v", err)
	// 	}

	// 	logging.InfoLogger.Printf("Using system RAM: %.2f GB", ram)
	// 	return ram, nil
	// } else {
	ram, err := GetSystemRAM()
	if err != nil {
		return 0, fmt.Errorf("failed to get system RAM: %v", err)
	}

	logging.InfoLogger.Printf("Using system RAM: %.2f GB", ram)
	return ram, nil
	// }
}

type OllamaModelInfo struct {
	Details struct {
		ParameterSize     string   `json:"parameter_size"`
		QuantizationLevel string   `json:"quantization_level"`
		Family            string   `json:"family"`
		Families          []string `json:"families"`
	} `json:"details"`
	ModelInfo map[string]interface{} `json:"model_info"`
}

func extractModelInfo(info map[string]interface{}, key string) (float64, bool) {
	for k, v := range info {
		if strings.HasSuffix(k, key) {
			switch val := v.(type) {
			case float64:
				return val, true
			case int64:
				return float64(val), true
			case int:
				return float64(val), true
			}
		}
	}
	return 0, false
}

func FetchOllamaModelInfo(apiURL, modelName string) (*OllamaModelInfo, error) {
	url := fmt.Sprintf("%s/api/show", apiURL)
	payload := []byte(fmt.Sprintf(`{"name": "%s"}`, modelName))

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("error making request to Ollama API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama API returned non-OK status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading Ollama API response: %v", err)
	}

	logging.DebugLogger.Printf("Raw Ollama API response: %s", string(body))

	var modelInfo OllamaModelInfo
	if err := json.Unmarshal(body, &modelInfo); err != nil {
		return nil, fmt.Errorf("error decoding Ollama API response: %v", err)
	}

	return &modelInfo, nil
}

func EstimateVRAM(modelIdentifier, apiURL string, fitsVRAM float64) error {
	var ollamaModelInfo *OllamaModelInfo
	var err error

	// Check if the modelIdentifier is an Ollama model name
	if strings.Contains(modelIdentifier, ":") {
		ollamaModelInfo, err = FetchOllamaModelInfo(apiURL, modelIdentifier)
		if err != nil {
			return fmt.Errorf("error fetching Ollama model info: %v", err)
		}
	}

	// Generate the quantization table
	table, err := GenerateQuantTable(modelIdentifier, fitsVRAM, ollamaModelInfo, 65536)
	if err != nil {
		return fmt.Errorf("error generating quantization table: %v", err)
	}

	// Print the formatted table
	fmt.Println(PrintFormattedTable(table))

	return nil
}

// CalculateVRAMRaw calculates the raw VRAM usage
func CalculateVRAMRaw(config ModelConfig, bpwValues BPWValues, context int, numGPUs int, gqa bool) float64 {
	logging.DebugLogger.Println("Calculating VRAM usage...")

	cudaSize := float64(CUDASize * numGPUs)
	paramsSize := config.NumParams * 1e9 * (bpwValues.BPW / 8)

	kvCacheSize := float64(context*2*config.NumHiddenLayers*config.HiddenSize) * (bpwValues.KVCacheBPW / 8)
	if gqa {
		kvCacheSize *= float64(config.NumKeyValueHeads) / float64(config.NumAttentionHeads)
	}

	bytesPerParam := bpwValues.BPW / 8
	lmHeadBytesPerParam := bpwValues.LMHeadBPW / 8

	headDim := float64(config.HiddenSize) / float64(config.NumAttentionHeads)
	attentionInput := bytesPerParam * float64(context*config.HiddenSize)

	q := bytesPerParam * float64(context) * headDim * float64(config.NumAttentionHeads)
	k := bytesPerParam * float64(context) * headDim * float64(config.NumKeyValueHeads)
	v := bytesPerParam * float64(context) * headDim * float64(config.NumKeyValueHeads)

	softmaxOutput := lmHeadBytesPerParam * float64(config.NumAttentionHeads*context)
	softmaxDropoutMask := float64(config.NumAttentionHeads * context)
	dropoutOutput := lmHeadBytesPerParam * float64(config.NumAttentionHeads*context)

	outProjInput := lmHeadBytesPerParam * float64(context*config.NumAttentionHeads) * headDim
	attentionDropout := float64(context * config.HiddenSize)

	attentionBlock := attentionInput + q + k + softmaxOutput + v + outProjInput + softmaxDropoutMask + dropoutOutput + attentionDropout

	mlpInput := bytesPerParam * float64(context*config.HiddenSize)
	activationInput := bytesPerParam * float64(context*config.IntermediateSize)
	downProjInput := bytesPerParam * float64(context*config.IntermediateSize)
	dropoutMask := float64(context * config.HiddenSize)
	mlpBlock := mlpInput + activationInput + downProjInput + dropoutMask

	layerNorms := bytesPerParam * float64(context*config.HiddenSize*2)
	activationsSize := attentionBlock + mlpBlock + layerNorms

	outputSize := lmHeadBytesPerParam * float64(context*config.VocabSize)

	vramBits := cudaSize + paramsSize + activationsSize + outputSize + kvCacheSize

	return bitsToGB(vramBits)
}

// bitsToGB converts bits to gigabytes
func bitsToGB(bits float64) float64 {
	return bits / math.Pow(2, 30)
}

// DownloadFile downloads a file from a URL and saves it to the specified path
func DownloadFile(url, filePath string, headers map[string]string) error {
	if _, err := os.Stat(filePath); err == nil {
		logging.InfoLogger.Println("File already exists, skipping download")
		return nil
	}

	// fmt.Printf("Downloading file from: %s\n", url)
	logging.DebugLogger.Println("Downloading file from:", url)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}
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

func GetHuggingFaceToken() string {
	accessToken := os.Getenv("HUGGINGFACE_TOKEN")
	if accessToken == "" {
		accessToken = os.Getenv("HF_TOKEN")
	}
	if accessToken == "" {
		tokenPath := filepath.Join(os.Getenv("HOME"), ".huggingface/token")
		if _, err := os.Stat(tokenPath); err == nil {
			token, err := os.ReadFile(tokenPath)
			if err == nil {
				accessToken = strings.TrimSpace(string(token))
			}
		}
	}
	return accessToken
}

// GetModelConfig retrieves and parses the model configuration
func GetModelConfig(modelID string) (ModelConfig, error) {
	cacheMutex.RLock()
	if config, ok := modelConfigCache[modelID]; ok {
		cacheMutex.RUnlock()
		return config, nil
	}
	cacheMutex.RUnlock()

	baseDir := filepath.Join(os.Getenv("HOME"), ".cache/huggingface/hub", modelID)
	configPath := filepath.Join(baseDir, "config.json")
	indexPath := filepath.Join(baseDir, "model.safetensors.index.json")

	configURL := fmt.Sprintf("https://huggingface.co/%s/raw/main/config.json", modelID)
	indexURL := fmt.Sprintf("https://huggingface.co/%s/raw/main/model.safetensors.index.json", modelID)

	headers := make(map[string]string)

	accessToken := GetHuggingFaceToken()

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

	cacheMutex.Lock()
	modelConfigCache[modelID] = config
	cacheMutex.Unlock()

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
	logging.DebugLogger.Println("Calculating BPW values...")
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
func CalculateVRAM(modelID string, bpw float64, context int, kvCacheQuant KVCacheQuantisation, ollamaModelInfo *OllamaModelInfo) (float64, error) {
	logging.DebugLogger.Println("Calculating VRAM usage...")

	var config ModelConfig
	var err error

	if ollamaModelInfo != nil {
		// Use Ollama model information
		paramCount, _ := extractModelInfo(ollamaModelInfo.ModelInfo, "parameter_count")
		contextLength, _ := extractModelInfo(ollamaModelInfo.ModelInfo, "context_length")
		blockCount, _ := extractModelInfo(ollamaModelInfo.ModelInfo, "block_count")
		embeddingLength, _ := extractModelInfo(ollamaModelInfo.ModelInfo, "embedding_length")
		headCountKV, _ := extractModelInfo(ollamaModelInfo.ModelInfo, "attention.head_count_kv")
		headCount, _ := extractModelInfo(ollamaModelInfo.ModelInfo, "attention.head_count")
		feedForwardLength, _ := extractModelInfo(ollamaModelInfo.ModelInfo, "feed_forward_length")
		vocabSize, _ := extractModelInfo(ollamaModelInfo.ModelInfo, "vocab_size")

		config = ModelConfig{
			NumParams:             paramCount / 1e9, // Convert to billions
			MaxPositionEmbeddings: int(contextLength),
			NumHiddenLayers:       int(blockCount),
			HiddenSize:            int(embeddingLength),
			NumKeyValueHeads:      int(headCountKV),
			NumAttentionHeads:     int(headCount),
			IntermediateSize:      int(feedForwardLength),
			VocabSize:             int(vocabSize),
		}

		// Estimate missing values
		if config.HiddenSize == 0 {
			config.HiddenSize = int(math.Sqrt(paramCount / 1000))
		}
		if config.NumHiddenLayers == 0 {
			config.NumHiddenLayers = int(math.Round(config.NumParams * 1e9 / (12 * float64(config.HiddenSize) * float64(config.HiddenSize))))
		}
		if config.NumAttentionHeads == 0 {
			config.NumAttentionHeads = config.HiddenSize / 64 // Assuming 64 dimension per head
		}
		if config.NumKeyValueHeads == 0 {
			config.NumKeyValueHeads = config.NumAttentionHeads
		}
		if config.IntermediateSize == 0 {
			config.IntermediateSize = 4 * config.HiddenSize
		}
		if config.VocabSize == 0 {
			config.VocabSize = 32000 // A common default value
		}

		// Parse BPW from quantization level if not provided
		if bpw == 0 {
			bpw, err = ParseBPWOrQuant(ollamaModelInfo.Details.QuantizationLevel)
			if err != nil {
				return 0, fmt.Errorf("error parsing BPW from Ollama quantization level: %v", err)
			}
		}

		logging.DebugLogger.Printf("Processed Ollama Model Config: %+v", config)
	} else {
		// Use Hugging Face model information
		config, err = GetModelConfig(modelID)
		if err != nil {
			return 0, err
		}
	}

	bpwValues := GetBPWValues(bpw, kvCacheQuant)

	if context == 0 {
		if ollamaModelInfo != nil {
			contextLength, found := extractModelInfo(ollamaModelInfo.ModelInfo, "context_length")
			if found {
				context = int(contextLength)
			}
		}
		if context == 0 {
			context = config.MaxPositionEmbeddings
		}
	}
	if context == 0 {
		context = 2048 // Default context if not provided
	}

	vram := CalculateVRAMRaw(config, bpwValues, context, 1, true)
	return math.Round(vram*100) / 100, nil
}

// CalculateContext calculates the maximum context for a given memory constraint
func CalculateContext(modelID string, memory, bpw float64, kvCacheQuant KVCacheQuantisation, ollamaModelInfo *OllamaModelInfo, topContext int) (int, error) {
	logging.DebugLogger.Println("Calculating context...")

	var maxContext int
	if ollamaModelInfo != nil {
		contextLength, found := extractModelInfo(ollamaModelInfo.ModelInfo, "context_length")
		if found {
			maxContext = int(contextLength)
		} else {
			// If context_length is not found, use the provided topContext
			maxContext = topContext
		}
	} else {
		config, err := GetModelConfig(modelID)
		if err != nil {
			return 0, err
		}
		maxContext = config.MaxPositionEmbeddings
	}

	// Use the smaller of maxContext and topContext
	if topContext < maxContext {
		maxContext = topContext
	}

	minContext := 512
	low, high := minContext, maxContext
	for low < high {
		mid := (low + high + 1) / 2
		vram, err := CalculateVRAM(modelID, bpw, mid, kvCacheQuant, ollamaModelInfo)
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
		vram, err := CalculateVRAM(modelID, bpw, context, kvCacheQuant, ollamaModelInfo)
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
func CalculateBPW(modelID string, memory float64, context int, kvCacheQuant KVCacheQuantisation, quantType string, ollamaModelInfo *OllamaModelInfo) (interface{}, error) {
	logging.DebugLogger.Println("Calculating BPW...")

	switch quantType {
	case "exl2":
		for _, bpw := range EXL2Options {
			vram, err := CalculateVRAM(modelID, bpw, context, kvCacheQuant, ollamaModelInfo)
			if err != nil {
				return nil, err
			}
			if vram < memory {
				return bpw, nil
			}
		}
	case "gguf":
		for name, bpw := range GGUFMapping {
			vram, err := CalculateVRAM(modelID, bpw, context, kvCacheQuant, ollamaModelInfo)
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

	return nil, fmt.Errorf("no suitable BPW found for the given memory constraint")
}

// parseBPWOrQuant takes a string and returns a float64 BPW value
func ParseBPWOrQuant(input string) (float64, error) {
	// First, try to parse as a float64 (direct BPW value)
	bpw, err := strconv.ParseFloat(input, 64)
	if err == nil {
		return bpw, nil
	}

	// If parsing as float fails, check if it's a valid quantisation type
	input = strings.ToUpper(input) // Convert to uppercase for case-insensitive matching
	if bpw, ok := GGUFMapping[input]; ok {
		return bpw, nil
	}

	// If not found, try to find a close match
	var closestMatch string
	var minDistance int = len(input)
	for key := range GGUFMapping {
		distance := levenshteinDistance(input, key)
		if distance < minDistance {
			minDistance = distance
			closestMatch = key
		}
	}

	if closestMatch != "" {
		return 0, fmt.Errorf("invalid quantisation type: %s. Did you mean %s?", input, closestMatch)
	}

	return 0, fmt.Errorf("invalid quantisation or BPW value: %s", input)
}

// levenshteinDistance calculates the Levenshtein distance between two strings
func levenshteinDistance(s1, s2 string) int {
	s1 = strings.ToUpper(s1)
	s2 = strings.ToUpper(s2)
	m := len(s1)
	n := len(s2)
	d := make([][]int, m+1)
	for i := range d {
		d[i] = make([]int, n+1)
	}
	for i := 0; i <= m; i++ {
		d[i][0] = i
	}
	for j := 0; j <= n; j++ {
		d[0][j] = j
	}
	for j := 1; j <= n; j++ {
		for i := 1; i <= m; i++ {
			if s1[i-1] == s2[j-1] {
				d[i][j] = d[i-1][j-1]
			} else {
				min := d[i-1][j]
				if d[i][j-1] < min {
					min = d[i][j-1]
				}
				if d[i-1][j-1] < min {
					min = d[i-1][j-1]
				}
				d[i][j] = min + 1
			}
		}
	}
	return d[m][n]
}

func GenerateQuantTable(modelID string, fitsVRAM float64, ollamaModelInfo *OllamaModelInfo, topContext int) (QuantResultTable, error) {
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

	// Generate context sizes based on the topContext
	contextSizes := generateContextSizes(topContext)

	if ollamaModelInfo == nil {
		_, err := GetModelConfig(modelID)
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
			vramFP16, err := CalculateVRAM(modelID, bpw, context, KVCacheFP16, ollamaModelInfo)
			if err != nil {
				return QuantResultTable{}, err
			}
			vramQ8_0, err := CalculateVRAM(modelID, bpw, context, KVCacheQ8_0, ollamaModelInfo)
			if err != nil {
				return QuantResultTable{}, err
			}
			vramQ4_0, err := CalculateVRAM(modelID, bpw, context, KVCacheQ4_0, ollamaModelInfo)
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

// generateContextSizes generates a slice of context sizes based on the topContext
func generateContextSizes(topContext int) []int {
	sizes := []int{2048, 8192}
	current := 16384
	for current <= topContext {
		sizes = append(sizes, current)
		current *= 2
	}
	if current/2 < topContext {
		sizes = append(sizes, topContext)
	}
	return sizes
}

func PrintFormattedTable(table QuantResultTable) string {
	var buf bytes.Buffer
	tw := tablewriter.NewWriter(&buf)

	// Get context sizes from the first result (assuming all results have the same context sizes)
	var contextSizes []int
	if len(table.Results) > 0 {
		for context := range table.Results[0].Contexts {
			contextSizes = append(contextSizes, context)
		}
		sort.Ints(contextSizes)
	}

	// Set table header
	header := []string{"Quant|Ctx", "BPW"}
	for _, context := range contextSizes {
		header = append(header, fmt.Sprintf("%dK", context/1024))
	}
	tw.SetHeader(header)

	// Set table style
	tw.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	tw.SetCenterSeparator("|")
	tw.SetColumnSeparator("|")
	tw.SetRowSeparator("-")

	// Set header colour to bright white
	headerColours := make([]tablewriter.Colors, len(header))
	for i := range headerColours {
		headerColours[i] = tablewriter.Colors{tablewriter.FgHiWhiteColor}
	}
	tw.SetHeaderColor(headerColours...)

	// Prepare data rows
	for _, result := range table.Results {
		row := []string{
			result.QuantType,
			fmt.Sprintf("%.2f", result.BPW),
		}

		// Add VRAM estimates for each context size
		for _, context := range contextSizes {
			vram, ok := result.Contexts[context]
			if !ok {
				row = append(row, "-")
				continue
			}

			fp16Str := getColouredVRAM(vram.VRAM, fmt.Sprintf("%.1f", vram.VRAM), table.FitsVRAM)

			if context >= 16384 {
				q8Str := getColouredVRAM(vram.VRAMQ8_0, fmt.Sprintf("%.1f", vram.VRAMQ8_0), table.FitsVRAM)
				q4Str := getColouredVRAM(vram.VRAMQ4_0, fmt.Sprintf("%.1f", vram.VRAMQ4_0), table.FitsVRAM)

				combinedStr := fmt.Sprintf("%s(%s,%s)", fp16Str, q8Str, q4Str)
				row = append(row, combinedStr)
			} else {
				row = append(row, fp16Str)
			}
		}

		tw.Append(row)
	}

	// Render the table
	tw.Render()

	return lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff")).Render(fmt.Sprintf("📊 VRAM Estimation for Model: %s\n\n%s", table.ModelID, buf.String()))
}

func getColouredVRAM(vram float64, vramStr string, fitsVRAM float64) string {
	var colorIndex int
	if fitsVRAM > 0 {
		if vram > fitsVRAM {
			colorIndex = 0 // Red
		} else {
			colorIndex = len(colourMap) - 1 // Green
		}
	} else {
		// Calculate color index based on VRAM usage
		if vram <= 4 {
			colorIndex = len(colourMap) - 1
		} else if vram >= 24 {
			colorIndex = 0
		} else {
			// Interpolate between 4 and 24 GB
			colorIndex = len(colourMap) - 1 - int((vram-4)/(24-4)*float64(len(colourMap)-1))
		}
	}

	style := lipgloss.NewStyle().Foreground(lipgloss.Color(colourMap[colorIndex]))
	return style.Render(vramStr)
}
