package main

import (
	"context"
	"fmt"
	"reflect"
)

// VerifyServiceBinding checks that all expected methods are available on the App struct
func VerifyServiceBinding() {
	fmt.Println("=== 🔍 VERIFYING WAILS v3 SERVICE BINDING ===")

	app := NewApp()
	appType := reflect.TypeOf(app)

	expectedMethods := []string{
		"OnStartup",
		"OnShutdown",
		"GetModels",
		"GetModel",
		"GetRunningModels",
		"RunModel",
		"DeleteModel",
		"UnloadModel",
		"CopyModel",
		"PushModel",
		"PullModel",
		"GetModelDetails",
		"EstimateVRAM",
		"GetConfig",
		"UpdateConfig",
		"HealthCheck",
		"TestServiceBinding",
		"GetServiceStatus",
	}

	fmt.Printf("📊 Checking %d expected methods on App struct...\n", len(expectedMethods))

	foundMethods := 0
	for _, methodName := range expectedMethods {
		if method, found := appType.MethodByName(methodName); found {
			fmt.Printf("  ✅ %s - %s\n", methodName, getMethodSignature(method))
			foundMethods++
		} else {
			fmt.Printf("  ❌ %s - NOT FOUND\n", methodName)
		}
	}

	fmt.Printf("\n📈 RESULTS: %d/%d methods found (%.1f%%)\n",
		foundMethods, len(expectedMethods),
		float64(foundMethods)/float64(len(expectedMethods))*100)

	if foundMethods == len(expectedMethods) {
		fmt.Println("✅ ALL METHODS VERIFIED - Service binding should work correctly")
	} else {
		fmt.Println("❌ MISSING METHODS - Service binding may have issues")
	}

	// Test basic initialization
	fmt.Println("\n🧪 Testing basic App initialization...")
	ctx := context.Background()
	app.OnStartup(ctx)

	if app.ctx != nil {
		fmt.Println("✅ Context properly set")
	} else {
		fmt.Println("❌ Context not set")
	}

	// Test service binding method
	fmt.Println("\n🔗 Testing service binding...")
	result := app.TestServiceBinding()
	fmt.Printf("TestServiceBinding result: %s\n", result)

	fmt.Println("=== ✅ SERVICE BINDING VERIFICATION COMPLETE ===")
}

func getMethodSignature(method reflect.Method) string {
	methodType := method.Type

	// Build parameter list
	var params []string
	for i := 1; i < methodType.NumIn(); i++ { // Skip receiver (index 0)
		params = append(params, methodType.In(i).String())
	}

	// Build return type list
	var returns []string
	for i := 0; i < methodType.NumOut(); i++ {
		returns = append(returns, methodType.Out(i).String())
	}

	paramStr := ""
	if len(params) > 0 {
		paramStr = fmt.Sprintf("(%s)", joinStrings(params, ", "))
	}

	returnStr := ""
	if len(returns) > 0 {
		if len(returns) == 1 {
			returnStr = fmt.Sprintf(" -> %s", returns[0])
		} else {
			returnStr = fmt.Sprintf(" -> (%s)", joinStrings(returns, ", "))
		}
	}

	return fmt.Sprintf("%s%s", paramStr, returnStr)
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}

	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
