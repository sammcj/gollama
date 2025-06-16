package main

import (
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:generate go run build/build.go

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Initialize the app manually since we can't use callbacks
	app.OnStartup(nil) // Pass nil context for now

	// Create application with options
	wailsApp := application.New(application.Options{
		Name:        "Gollama",
		Description: "Ollama Model Manager",
		Services: []application.Service{
			application.NewService(app),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(staticFiles),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	// Create a window
	wailsApp.NewWebviewWindow().
		SetTitle("Gollama - Ollama Model Manager").
		SetSize(1200, 800).
		SetMinSize(800, 600).
		SetURL("/").
		Show()

	err := wailsApp.Run()

	if err != nil {
		log.Fatalf("Error starting application: %v", err)
	}
}
