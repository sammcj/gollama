package main

import (
	"context"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:generate go run build/build.go

func main() {
	log.Println("=== 🚀 GOLLAMA WAILS v3 APPLICATION STARTUP ===")

	// Create an instance of the app structure
	log.Println("📦 Creating App instance...")
	app := NewApp()
	if app == nil {
		log.Fatalf("❌ CRITICAL: Failed to create App instance")
	}
	log.Println("✅ App instance created successfully")

	// Create Wails service with enhanced logging
	log.Println("🔧 Registering App as Wails v3 service...")
	appService := application.NewService(app)
	log.Println("✅ Wails v3 service created successfully")

	// Create application with options and comprehensive logging
	log.Println("🏗️  Building Wails application with configuration...")
	wailsApp := application.New(application.Options{
		Name:        "Gollama",
		Description: "Ollama Model Manager",
		Services: []application.Service{
			appService,
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(staticFiles),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	log.Println("✅ Wails application created successfully")

	// Log service registration details
	log.Println("=== 🔍 WAILS v3 SERVICE REGISTRATION DETAILS ===")
	log.Printf("✓ Service Name: %s\n", "Gollama")
	log.Printf("✓ Service Description: %s\n", "Ollama Model Manager")
	log.Printf("✓ Services Registered: %d\n", 1)
	log.Printf("✓ App Methods Exposed: %d\n", 21)
	log.Println("✓ Asset Handler: Embedded FS")
	log.Println("✓ Platform: macOS optimized")

	// Initialize the app with proper context after Wails app creation
	log.Println("🔄 Initializing application context and services...")
	ctx := context.Background()
	app.OnStartup(ctx)
	log.Println("✅ Application initialization complete")

	// Create a window with detailed logging
	log.Println("🪟 Creating application window...")
	window := wailsApp.NewWebviewWindow()

	window.
		SetTitle("Gollama - Ollama Model Manager").
		SetSize(1200, 800).
		SetMinSize(800, 600).
		SetURL("/").
		Show()

	log.Println("✅ Application window created and displayed")
	log.Println("🌐 Window URL: /")
	log.Println("📐 Window Size: 1200x800 (min: 800x600)")

	// Final startup confirmation
	log.Println("=== 🎯 WAILS v3 STARTUP SUMMARY ===")
	log.Println("✅ App struct instantiated")
	log.Println("✅ Wails v3 service registered")
	log.Println("✅ Application configured")
	log.Println("✅ Service methods exposed to JavaScript")
	log.Println("✅ Window created and shown")
	log.Println("🚀 Starting Wails application runtime...")

	// Run the application
	err := wailsApp.Run()
	if err != nil {
		log.Fatalf("❌ CRITICAL ERROR: Wails application failed to run: %v", err)
	}

	log.Println("👋 Gollama application shutdown complete")
}
