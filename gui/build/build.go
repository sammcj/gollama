//go:build ignore

package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func main() {
	fmt.Println("Building GUI assets...")

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Working directory: %s\n", cwd)

	// Download Tailwind CSS CLI if not exists
	err = ensureTailwindCSS()
	if err != nil {
		fmt.Printf("Error ensuring Tailwind CSS: %v\n", err)
		os.Exit(1)
	}

	// Build Tailwind CSS
	err = buildTailwind()
	if err != nil {
		fmt.Printf("Error building Tailwind: %v\n", err)
		os.Exit(1)
	}

	// Download HTMX if not exists
	err = ensureHTMX()
	if err != nil {
		fmt.Printf("Error downloading HTMX: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Build completed successfully!")
}

func ensureTailwindCSS() error {
	// Determine the correct binary name for the platform
	var binaryName string
	switch runtime.GOOS {
	case "darwin":
		if runtime.GOARCH == "arm64" {
			binaryName = "tailwindcss-macos-arm64"
		} else {
			binaryName = "tailwindcss-macos-x64"
		}
	case "linux":
		if runtime.GOARCH == "arm64" {
			binaryName = "tailwindcss-linux-arm64"
		} else {
			binaryName = "tailwindcss-linux-x64"
		}
	case "windows":
		if runtime.GOARCH == "arm64" {
			binaryName = "tailwindcss-windows-arm64.exe"
		} else {
			binaryName = "tailwindcss-windows-x64.exe"
		}
	default:
		return fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	localBinary := filepath.Join("build", "tailwindcss")
	if runtime.GOOS == "windows" {
		localBinary += ".exe"
	}

	// Check if binary already exists
	if _, err := os.Stat(localBinary); err == nil {
		fmt.Println("Tailwind CSS binary already exists")
		return nil
	}

	fmt.Printf("Downloading Tailwind CSS CLI for %s/%s...\n", runtime.GOOS, runtime.GOARCH)

	// Download the binary (use v3.4.0 for compatibility)
	url := fmt.Sprintf("https://github.com/tailwindlabs/tailwindcss/releases/download/v3.4.0/%s", binaryName)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download Tailwind CSS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download Tailwind CSS: HTTP %d", resp.StatusCode)
	}

	// Create the binary file
	file, err := os.Create(localBinary)
	if err != nil {
		return fmt.Errorf("failed to create binary file: %w", err)
	}
	defer file.Close()

	// Copy the downloaded content
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write binary file: %w", err)
	}

	// Make it executable
	err = os.Chmod(localBinary, 0755)
	if err != nil {
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	fmt.Println("Tailwind CSS CLI downloaded successfully")
	return nil
}

func buildTailwind() error {
	fmt.Println("Building Tailwind CSS...")

	binaryPath := filepath.Join("build", "tailwindcss")
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}

	cmd := exec.Command(binaryPath,
		"-i", "build/input.css",
		"-o", "static/css/tailwind.css",
		"--config", "build/tailwind.config.js",
		"--minify")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tailwind build failed: %w\nOutput: %s", err, output)
	}

	fmt.Println("Tailwind CSS built successfully")
	return nil
}

func ensureHTMX() error {
	htmxPath := filepath.Join("static", "js", "htmx.min.js")

	// Check if HTMX already exists
	if _, err := os.Stat(htmxPath); err == nil {
		fmt.Println("HTMX already exists")
		return nil
	}

	fmt.Println("Downloading HTMX...")

	// Download HTMX
	resp, err := http.Get("https://unpkg.com/htmx.org@1.9.10/dist/htmx.min.js")
	if err != nil {
		return fmt.Errorf("failed to download HTMX: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download HTMX: HTTP %d", resp.StatusCode)
	}

	// Create the file
	file, err := os.Create(htmxPath)
	if err != nil {
		return fmt.Errorf("failed to create HTMX file: %w", err)
	}
	defer file.Close()

	// Copy the downloaded content
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write HTMX file: %w", err)
	}

	fmt.Println("HTMX downloaded successfully")
	return nil
}
