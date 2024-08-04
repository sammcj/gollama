// File: vramestimator/utils.go

package vramestimator

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sammcj/gollama/logging"
	"github.com/shirou/gopsutil/mem"
)

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

// ParseBPW parses the BPW value
func ParseBPW(bpw string) float64 {
	if val, ok := GGUFMapping[bpw]; ok {
		return val
	}
	return 0
}

// bitsToGB converts bits to gigabytes
func bitsToGB(bits float64) float64 {
	return bits / math.Pow(2, 30)
}

// GetBPWValues parses the BPW values based on the input
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

func GetSystemRAM() (float64, error) {
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return 0, fmt.Errorf("failed to get system memory info: %v", err)
	}

	totalRAM := float64(vmStat.Total) / 1024 / 1024 / 1024 // Convert to GB
	return totalRAM, nil
}
