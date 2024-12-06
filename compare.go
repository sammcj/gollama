package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ollama/ollama/api"
	"github.com/sammcj/gollama/logging"
)

type ModelfileDiff struct {
	Command string
	Current string
	Latest  string
	Type    string // "modified", "added", or "removed"
}

func fetchLatestModelfile(modelName string) (string, error) {
	// Split the model name into components
	parts := strings.Split(modelName, ":")
	name := parts[0]
	tag := "latest"
	if len(parts) > 1 {
		tag = parts[1]
	}
	// Clean the model name
	name = strings.ToLower(name)
	name = strings.TrimPrefix(name, "library/")
	// Handle special cases for common model naming patterns
	name = strings.ReplaceAll(name, ".", "-") // e.g., llama2.7b -> llama2-7b
	name = strings.ReplaceAll(name, "_", "-") // Replace underscores with hyphens
	// Normalize registry path
	path := name
	if !strings.Contains(path, "/") {
		path = "library/" + path
	}
	url := fmt.Sprintf("https://registry.ollama.ai/v2/%s/manifests/%s", path, tag)
	logging.DebugLogger.Printf("Fetching manifest from URL: %s", url)
	// First get the manifest
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	// Add required OCI registry headers
	req.Header.Set("Accept", "application/vnd.oci.image.manifest.v1+json")
	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	logging.DebugLogger.Printf("Response status code: %d for URL: %s", resp.StatusCode, url)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch manifest: status %d", resp.StatusCode)
	}
	// Read and log the manifest response
	manifestBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading manifest body: %v", err)
	}
	logging.DebugLogger.Printf("Manifest response: %s", string(manifestBody))
	// Parse the manifest to get the config blob
	var manifest struct {
		SchemaVersion int `json:"schemaVersion"`
		Config        struct {
			MediaType string `json:"mediaType"`
			Digest    string `json:"digest"`
			Size      int    `json:"size"`
		} `json:"config"`
		Layers []struct {
			MediaType string `json:"mediaType"`
			Digest    string `json:"digest"`
			Size      int    `json:"size"`
		} `json:"layers"`
	}
	if err := json.Unmarshal(manifestBody, &manifest); err != nil {
		return "", fmt.Errorf("error decoding manifest: %v", err)
	}
	logging.DebugLogger.Printf("Parsed manifest: %+v", manifest)

	// Find the TEMPLATE and PARAMETER layers
	var templateDigest, paramsDigest string
	for _, layer := range manifest.Layers {
		switch layer.MediaType {
		case "application/vnd.ollama.image.template":
			templateDigest = layer.Digest
		case "application/vnd.ollama.image.params":
			paramsDigest = layer.Digest
		}
	}

	if templateDigest == "" || paramsDigest == "" {
		return "", fmt.Errorf("template or parameter layer not found in manifest")
	}

	// Fetch the TEMPLATE layer
	templateURL := fmt.Sprintf("https://registry.ollama.ai/v2/%s/blobs/%s", path, templateDigest)
	logging.DebugLogger.Printf("Fetching template from URL: %s", templateURL)
	req, err = http.NewRequest("GET", templateURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.ollama.image.template")
	resp, err = httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch template: status %d", resp.StatusCode)
	}
	templateBody, err := io.ReadAll(resp.Body)
	if err != nil {

		return "", fmt.Errorf("error reading template body: %v", err)
	}
	logging.DebugLogger.Printf("Template response: %s", string(templateBody))

	// Fetch the PARAMETER layer
	paramsURL := fmt.Sprintf("https://registry.ollama.ai/v2/%s/blobs/%s", path, paramsDigest)
	logging.DebugLogger.Printf("Fetching parameters from URL: %s", paramsURL)
	req, err = http.NewRequest("GET", paramsURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.ollama.image.params")
	resp, err = httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch parameters: status %d", resp.StatusCode)
	}
	paramsBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading parameters body: %v", err)
	}
	logging.DebugLogger.Printf("Parameters response: %s", string(paramsBody))

	// Parse the PARAMETER layer
	var params map[string]interface{}
	if err := json.Unmarshal(paramsBody, &params); err != nil {
		return "", fmt.Errorf("error decoding parameters: %v", err)
	}

	// Construct the modelfile content
	var modelfile strings.Builder
	modelfile.WriteString(fmt.Sprintf("TEMPLATE \"\"\"%s\"\"\"\n", templateBody))
	for key, value := range params {
		switch v := value.(type) {
		case string:
			modelfile.WriteString(fmt.Sprintf("PARAMETER %s %s\n", key, v))
		case []interface{}:
			for _, item := range v {
				modelfile.WriteString(fmt.Sprintf("PARAMETER %s %s\n", key, item))
			}
		default:
			logging.DebugLogger.Printf("Unknown type for parameter %s: %v", key, value)
		}
	}

	return modelfile.String(), nil
}

func compareModelfiles(currentTemplate string, currentParams map[string]string, latestTemplate string, latestParams map[string]string) []ModelfileDiff {
	diffs := []ModelfileDiff{}

	// Compare TEMPLATE
	if currentTemplate != latestTemplate {
		diffs = append(diffs, ModelfileDiff{
			Command: "TEMPLATE",
			Current: currentTemplate,
			Latest:  latestTemplate,
			Type:    "modified",
		})
	}

	// Compare PARAMETERs
	for cmd, latestValue := range latestParams {
		if currentValue, exists := currentParams[cmd]; exists {
			if currentValue != latestValue {
				diffs = append(diffs, ModelfileDiff{
					Command: cmd,
					Current: currentValue,
					Latest:  latestValue,
					Type:    "modified",
				})
			}
		} else {
			diffs = append(diffs, ModelfileDiff{
				Command: cmd,
				Latest:  latestValue,
				Type:    "added",
			})
		}
	}

	// Check for removed PARAMETERs
	for cmd, currentValue := range currentParams {
		if _, exists := latestParams[cmd]; !exists {
			diffs = append(diffs, ModelfileDiff{
				Command: cmd,
				Current: currentValue,
				Type:    "removed",
			})
		}
	}

	return diffs
}

func (m *AppModel) handleCompareModelfile() (tea.Model, tea.Cmd) {
	if item, ok := m.list.SelectedItem().(Model); ok {
		// Get current modelfile
		_, err := m.client.Show(context.Background(), &api.ShowRequest{Name: item.Name})
		if err != nil {
			m.message = fmt.Sprintf("Error fetching current modelfile: %v", err)
			return m, nil
		}
		currentParams, currentTemplate, err := getModelParams(item.Name, m.client)
		if err != nil {
			m.message = fmt.Sprintf("Error parsing current modelfile: %v", err)
			return m, nil
		}

		// Get latest modelfile
		latestModelfile, err := fetchLatestModelfile(item.Name)
		if err != nil {
			if strings.Contains(err.Error(), "failed to fetch") {
				m.message = fmt.Sprintf("Model '%s' not found in public registry - it might be a private or custom model", item.Name)
			} else {
				m.message = fmt.Sprintf("Error fetching latest modelfile: %v", err)
			}
			return m, nil
		}

		// Parse the latest modelfile to extract TEMPLATE and PARAMETERs
		latestLines := strings.Split(latestModelfile, "\n")
		var latestTemplate string
		latestParams := make(map[string]string)

		for _, line := range latestLines {
			if strings.HasPrefix(line, "TEMPLATE") {
				latestTemplate = strings.TrimPrefix(line, "TEMPLATE ")
				latestTemplate = strings.Trim(latestTemplate, "\"")
			} else if strings.HasPrefix(line, "PARAMETER") {
				parts := strings.SplitN(line, " ", 2)
				if len(parts) == 2 {
					key := parts[1]
					value := strings.TrimSpace(parts[1])
					latestParams[key] = value
				}
			}
		}

		// Compare modelfiles
		m.modelfileDiffs = compareModelfiles(currentTemplate, currentParams, latestTemplate, latestParams)
		if len(m.modelfileDiffs) == 0 {
			m.message = "No differences found between local and registry modelfiles"
			return m, nil
		}
		m.comparingModelfile = true
	}
	return m, nil
}

func (m *AppModel) modelfileDiffView() string {
	if !m.comparingModelfile {
		return ""
	}

	// Define styles using the application's colour scheme
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF00FF")).
		MarginBottom(1).
		Padding(0, 1)

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#333333")).
		Padding(0, 1)

	commandStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9932CC")).
		Padding(0, 1)

	localStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#60BFFF")).
		Padding(0, 1)

	remoteStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00CED1")).
		Padding(0, 1)

	modifiedLocalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFF00")).
		Padding(0, 1)

	modifiedRemoteStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFA500")).
		Padding(0, 1)

	addedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FF00")).
		Padding(0, 1)

	removedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF0000")).
		Padding(0, 1)

	// Calculate column widths
	commandWidth := 20
	valueWidth := 30

	for _, diff := range m.modelfileDiffs {
		commandLen := len(diff.Command)
		if commandLen > commandWidth {
			commandWidth = commandLen
		}
		currentLen := len(diff.Current)
		latestLen := len(diff.Latest)
		if currentLen > valueWidth {
			valueWidth = currentLen
		}
		if latestLen > valueWidth {
			valueWidth = latestLen
		}
	}

	// Build the table
	var rows []string

	// Add header
	header := lipgloss.JoinHorizontal(lipgloss.Left,
		headerStyle.Width(commandWidth).Render("Command"),
		headerStyle.Width(valueWidth).Render("Local Value"),
		headerStyle.Width(valueWidth).Render("Remote Value"),
	)

	rows = append(rows, header)

	// Add separator
	separator := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#333333")).
		Render(strings.Repeat("─", commandWidth+valueWidth*2+2))
	rows = append(rows, separator)

	// Add data rows
	for _, diff := range m.modelfileDiffs {
		var current, latest string
		var currentStyle, latestStyle lipgloss.Style

		switch diff.Type {
		case "modified":
			currentStyle = modifiedLocalStyle
			latestStyle = modifiedRemoteStyle
			current = diff.Current
			latest = diff.Latest
		case "added":
			currentStyle = localStyle
			latestStyle = addedStyle
			current = "undefined"
			latest = diff.Latest
		case "removed":
			currentStyle = localStyle
			latestStyle = removedStyle
			current = diff.Current
			latest = "undefined"
		default:
			currentStyle = localStyle
			latestStyle = remoteStyle
			current = diff.Current
			latest = diff.Latest
		}

		row := lipgloss.JoinHorizontal(lipgloss.Left,
			commandStyle.Width(commandWidth).Render(diff.Command),
			currentStyle.Width(valueWidth).Render(current),
			latestStyle.Width(valueWidth).Render(latest),
		)
		rows = append(rows, row)
	}

	// Build the final view
	var b strings.Builder
	b.WriteString(titleStyle.Render("Modelfile Comparison"))
	b.WriteString("\n\n")
	b.WriteString(strings.Join(rows, "\n"))
	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Render("Press 'q' or 'esc' to return to the main view"))

	return b.String()
}
