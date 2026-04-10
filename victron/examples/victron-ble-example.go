// Example demonstrating how to scan for and connect to Victron BLE devices
//
// Usage: go run examples/victron-ble-example.go
//
// This example shows:
// 1. How to scan for nearby Victron devices (including "Glow" LED indicators)
// 2. How to connect to a discovered device
// 3. How to discover services and characteristics
// 4. How to read values from the device
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/scottstg/yolo/victron/ble"
	_ "github.com/scottstg/yolo/victron/macos" // macOS BLE backend
)

func main() {
	fmt.Println("=== Victron BLE Device Scanner Example ===\n")

	// Check if a device address was provided as argument
	var targetAddress string
	if len(os.Args) > 1 {
		targetAddress = os.Args[1]
		fmt.Printf("[INFO] Will connect to specific device: %s\n\n", targetAddress)
	}

	// Create BLE backend
	fmt.Println("[1] Initializing BLE backend...")
	b, err := ble.New()
	if err != nil {
		fmt.Printf("[ERROR] Failed to create BLE backend: %v\n", err)
		os.Exit(1)
	}

	err = b.Initialize()
	if err != nil {
		fmt.Printf("[ERROR] Failed to initialize BLE: %v\n", err)
		os.Exit(1)
	}
	defer b.Close()
	fmt.Println("[OK] BLE backend initialized successfully\n")

	// If no specific device was provided, scan for all devices
	if targetAddress == "" {
		fmt.Println("[2] Scanning for Victron devices...")
		devices, err := scanForDevices(b)
		if err != nil {
			fmt.Printf("[ERROR] Scan failed: %v\n", err)
			os.Exit(1)
		}

		if len(devices) == 0 {
			fmt.Println("\n[WARNING] No devices found!")
			fmt.Println("\nTroubleshooting tips:")
			fmt.Println("  - Ensure Bluetooth is enabled on your Mac")
			fmt.Println("  - Make sure the Victron device is powered on")
			fmt.Println("  - Bring the device closer (within ~10m)")
			fmt.Println("  - Some devices need to be actively transmitting data")
			fmt.Println("\nGlow LED Indicator Devices:")
			fmt.Println("  - Glow devices advertise with names like \"glow\" or \"Glow\"")
			fmt.Println("  - They show battery status via different colors")
			fmt.Println("  - May need to press a button to wake them up from sleep")
			os.Exit(0)
		}

		// Filter for Victron devices
		victronDevices := filterVictronDevices(devices)
		
		fmt.Printf("\n[OK] Found %d device(s):", len(victronDevices))
		for i, dev := range victronDevices {
			fmt.Printf("\n  [%d] %s (%s)", i+1, dev.Name, dev.Address)
			if dev.IsVictron {
				fmt.Print(" [VICTRON]")
			}
		}
		fmt.Println()

		// If only one Victron device found, use it automatically
		if len(victronDevices) == 1 {
			targetAddress = victronDevices[0].Address
			fmt.Printf("\n[INFO] Automatically using: %s\n", targetAddress)
		} else if len(victronDevices) > 1 {
			fmt.Println("\nPlease specify a device address to connect, or run with:")
			fmt.Printf("  go run examples/victron-ble-example.go %s\n", victronDevices[0].Address)
			os.Exit(0)
		} else {
			fmt.Println("\nNo Victron devices found (devices may be non-Victron BLE peripherals)")
			fmt.Println("To connect anyway, specify a device address:")
			fmt.Printf("  go run examples/victron-ble-example.go %s\n", devices[0].Address)
			os.Exit(0)
		}
	}

	// Connect to the target device
	fmt.Printf("\n[3] Connecting to Victron device: %s\n", targetAddress)
	if err := connectAndRead(b, targetAddress); err != nil {
		fmt.Printf("[ERROR] Connection failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n=== Example Complete ===")
}

func scanForDevices(b ble.Backend) ([]ble.Device, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	devices, err := b.ScanForDevices(ctx, 10*time.Second)
	if err != nil && err != context.DeadlineExceeded {
		return nil, err
	}

	// Print scan summary
	fmt.Printf("[INFO] Scan completed - found %d device(s) total\n", len(devices))
	return devices, nil
}

func filterVictronDevices(devices []ble.Device) []ble.Device {
	var victronDevices []ble.Device
	for _, dev := range devices {
		nameLower := strings.ToLower(dev.Name)
		if strings.Contains(nameLower, "glow") ||
			strings.Contains(nameLower, "victron") ||
			dev.IsVictron {
			victronDevices = append(victronDevices, dev)
		}
	}
	return victronDevices
}

func connectAndRead(b ble.Backend, address string) error {
	fmt.Println("[INFO] Connecting...")

	conn, err := b.Connect(address)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()
	fmt.Println("[OK] Connected successfully\n")

	fmt.Println("[INFO] Discovering services...")
	services, err := b.DiscoverServices(conn)
	if err != nil {
		return fmt.Errorf("failed to discover services: %w", err)
	}
	fmt.Printf("[OK] Found %d service(s)\n", len(services))

	// Victron devices use specific UUIDs for their data characteristics
	victronServiceUUID := "0000fff0-0000-1000-8000-00805f9b34fb"
	victronCharUUID := "0000fff1-0000-1000-8000-00805f9b34fb"

	// Try to find the Victron service and read data
	for _, svc := range services {
		fmt.Printf("  Service: %s\n", svc.UUID)
		
		if svc.UUID == victronServiceUUID {
			fmt.Println("    ^ Found Victron service!")
			
			// Note: Reading actual data requires additional protocol parsing
			// This is simplified for the example - real implementation would:
			// 1. Find characteristics within this service
			// 2. Subscribe to notifications or read values
			// 3. Parse VE.Direct protocol frames
			fmt.Println("\n[INFO] To read actual measurements:")
			fmt.Println("  - Use b.Subscribe() to get real-time updates")
			fmt.Println("  - Parse VE.Direct protocol (see victron/pkg/parser.go)")
			fmt.Println("  - Measurements include: voltage, current, battery status")
			break
		}
	}

	return nil
}
