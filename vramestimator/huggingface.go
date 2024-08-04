package vramestimator

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/sammcj/gollama/logging"
)

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

// GetModelConfig retrieves and parses the model configuration
func GetModelConfig(modelID, accessToken string) (ModelConfig, error) {
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
