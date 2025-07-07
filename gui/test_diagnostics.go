package main

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"
)

// TestDiagnosticMethods tests all the new diagnostic methods
func TestDiagnosticMethods(t *testing.T) {
	// Create app instance
	app := NewApp()
	if app == nil {
		t.Fatal("Failed to create App instance")
	}

	// Initialize app
	ctx := context.Background()
	app.OnStartup(ctx)

	// Test GetDiagnosticInfo
	t.Run("GetDiagnosticInfo", func(t *testing.T) {
		info, err := app.GetDiagnosticInfo()
		if err != nil {
			t.Errorf("GetDiagnosticInfo failed: %v", err)
			return
		}

		// Check required fields
		if info["timestamp"] == nil {
			t.Error("Missing timestamp in diagnostic info")
		}
		if info["initialized"] == nil {
			t.Error("Missing initialized status in diagnostic info")
		}
		if info["exposed_methods"] == nil {
			t.Error("Missing exposed_methods in diagnostic info")
		}

		fmt.Printf("✅ GetDiagnosticInfo returned %d fields\n", len(info))
	})

	// Test RunDiagnosticTests
	t.Run("RunDiagnosticTests", func(t *testing.T) {
		results, err := app.RunDiagnosticTests()
		if err != nil {
			t.Errorf("RunDiagnosticTests failed: %v", err)
			return
		}

		// Check required fields
		if results["timestamp"] == nil {
			t.Error("Missing timestamp in diagnostic tests")
		}
		if results["tests"] == nil {
			t.Error("Missing tests in diagnostic tests")
		}
		if results["summary"] == nil {
			t.Error("Missing summary in diagnostic tests")
		}

		summary := results["summary"].(map[string]interface{})
		total := summary["total"].(int)
		passed := summary["passed"].(int)

		fmt.Printf("✅ RunDiagnosticTests: %d/%d tests passed\n", passed, total)
	})

	// Test VerifyServiceBinding
	t.Run("VerifyServiceBinding", func(t *testing.T) {
		verification, err := app.VerifyServiceBinding()
		if err != nil {
			t.Errorf("VerifyServiceBinding failed: %v", err)
			return
		}

		// Check required fields
		if verification["binding_status"] == nil {
			t.Error("Missing binding_status in verification")
		}
		if verification["service_status"] == nil {
			t.Error("Missing service_status in verification")
		}

		bindingStatus := verification["binding_status"].(string)
		serviceStatus := verification["service_status"].(string)

		fmt.Printf("✅ VerifyServiceBinding: binding=%s, service=%s\n", bindingStatus, serviceStatus)
	})

	// Test GetServiceStatus (existing method)
	t.Run("GetServiceStatus", func(t *testing.T) {
		status, err := app.GetServiceStatus()
		if err != nil {
			t.Errorf("GetServiceStatus failed: %v", err)
			return
		}

		if status.Status == "" {
			t.Error("Missing status in service status")
		}

		fmt.Printf("✅ GetServiceStatus: %s\n", status.Status)
	})

	// Test TestServiceBinding (existing method)
	t.Run("TestServiceBinding", func(t *testing.T) {
		result := app.TestServiceBinding()
		if result == "" {
			t.Error("TestServiceBinding returned empty result")
		}

		fmt.Printf("✅ TestServiceBinding: %s\n", result)
	})
}

// Manual test function that can be run independently
func ManualDiagnosticTest() {
	fmt.Println("=== 🔍 MANUAL DIAGNOSTIC TEST ===")

	// Create and initialize app
	app := NewApp()
	ctx := context.Background()
	app.OnStartup(ctx)

	// Wait a moment for initialization
	time.Sleep(1 * time.Second)

	// Test all diagnostic methods
	fmt.Println("\n1. Testing GetDiagnosticInfo...")
	if info, err := app.GetDiagnosticInfo(); err != nil {
		fmt.Printf("❌ GetDiagnosticInfo failed: %v\n", err)
	} else {
		fmt.Printf("✅ GetDiagnosticInfo: %d fields returned\n", len(info))
	}

	fmt.Println("\n2. Testing RunDiagnosticTests...")
	if results, err := app.RunDiagnosticTests(); err != nil {
		fmt.Printf("❌ RunDiagnosticTests failed: %v\n", err)
	} else {
		summary := results["summary"].(map[string]interface{})
		fmt.Printf("✅ RunDiagnosticTests: %d/%d tests passed\n",
			summary["passed"], summary["total"])
	}

	fmt.Println("\n3. Testing VerifyServiceBinding...")
	if verification, err := app.VerifyServiceBinding(); err != nil {
		fmt.Printf("❌ VerifyServiceBinding failed: %v\n", err)
	} else {
		fmt.Printf("✅ VerifyServiceBinding: binding=%s, service=%s\n",
			verification["binding_status"], verification["service_status"])
	}

	fmt.Println("\n4. Testing GetServiceStatus...")
	if status, err := app.GetServiceStatus(); err != nil {
		fmt.Printf("❌ GetServiceStatus failed: %v\n", err)
	} else {
		fmt.Printf("✅ GetServiceStatus: %s\n", status.Status)
	}

	fmt.Println("\n5. Testing TestServiceBinding...")
	result := app.TestServiceBinding()
	fmt.Printf("✅ TestServiceBinding: %s\n", result)

	fmt.Println("\n=== ✅ MANUAL DIAGNOSTIC TEST COMPLETE ===")
}

// Run manual test if this file is executed directly
func init() {
	// This will run when the package is imported
	log.Println("Diagnostic test functions loaded")
}
