// +build ignore

// Test BLE scanning on macOS with proper initialization
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/go-ble/ble"
	_ "github.com/go-ble/ble/darwin" // macOS backend
)

func main() {
	fmt.Println("Testing BLE scan on macOS...")

	// Initialize the macOS backend first
	err := ble.Init(nil)
	if err != nil {
		fmt.Printf("ERROR: Failed to initialize BLE: %v\n", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	fmt.Println("[INFO] Starting scan...")
	
	err = ble.Scan(ctx, false, func(adv ble.Advertisement) {
		name := adv.LocalName()
		if name == "" {
			name = "(no name)"
		}
		fmt.Printf("Found: %s - %s\n", 
			adv.Addr().String(),
			name,
		)
		
		// Check for Victron devices
		lowerName := fmt.Sprintf("%s", name)
		if contains(lowerName, "glow") || contains(lowerName, "victron") || contains(lowerName, "ve.direct") {
			fmt.Printf("  ^^^ Possible Victron device!\n")
		}
	}, nil)

	if err != nil {
		fmt.Printf("ERROR: Scan failed: %v\n", err)
		return
	}

	fmt.Println("[INFO] Scan complete")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
