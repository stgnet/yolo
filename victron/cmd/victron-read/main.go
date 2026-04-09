// Package main provides a command-line tool to read values from Victron Energy devices via BLE.
// This is a simple example showing how to scan for and connect to Victron devices.
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/scottstg/yolo/victron"
)

func main() {
	if len(os.Args) < 2 {
		log.Printf("Usage: %s <device-mac>\n", os.Args[0])
		log.Printf("Example: %s AA:BB:CC:DD:EE:FF\n", os.Args[0])
		log.Printf("\nOr run without arguments to scan for nearby Victron devices:\n")
		log.Printf("  %s\n", os.Args[0])
		os.Exit(1)
	}

	// Initialize BLE backend first (required before scanning/connecting)
	if err := victron.InitializeBackend(); err != nil {
		log.Fatalf("Failed to initialize BLE backend: %v", err)
	}

	fmt.Println("=========================================")
	fmt.Println("  Victron Energy BLE Device Reader")
	fmt.Println("==========================================")

	// Create client
	client := victron.NewClient()

	if len(os.Args) == 2 {
		// Direct connect mode: user specified a MAC address
		mac := os.Args[1]
		connectToDevice(client, mac)
	} else {
		// Scan mode: discover nearby Victron devices
		fmt.Println("\nScanning for nearby Victron devices...")
		
		results, err := client.Scan(10 * time.Second)
		if err != nil {
			log.Fatalf("Scan failed: %v", err)
		}

		victronDevices := filterVictronDevices(results)
		if len(victronDevices) == 0 {
			fmt.Println("\nNo Victron devices found.")
			os.Exit(0)
		}

		fmt.Printf("\nFound %d Victron device(s):\n", len(victronDevices))
		for i, device := range victronDevices {
			fmt.Printf("  %d. %s (%s, RSSI: %d)\n", i+1, device.Address, device.Name, device.RSSI)
		}

		if len(victronDevices) == 1 {
			// Auto-connect to single device
			connectToDevice(client, victronDevices[0].Address)
		} else {
			fmt.Printf("\nConnect to which device? (1-%d): ", len(victronDevices))
			var choice int
			if _, err := fmt.Scan(&choice); err != nil || choice < 1 || choice > len(victronDevices) {
				log.Fatal("Invalid selection")
			}
			connectToDevice(client, victronDevices[choice-1].Address)
		}
	}

	fmt.Println("\nSession complete. Disconnecting...")
	time.Sleep(2 * time.Second)
}

// filterVictronDevices returns only devices identified as Victron products
func filterVictronDevices(devices []victron.DiscoverableDevice) []victron.DiscoverableDevice {
	var filtered []victron.DiscoverableDevice
	for _, d := range devices {
		if d.IsVictron {
			filtered = append(filtered, d)
		}
	}
	return filtered
}

// connectToDevice connects to a specific device and displays its values
func connectToDevice(client *victron.Client, mac string) {
	fmt.Printf("\nConnecting to device: %s\n\n", mac)

	device, err := client.Connect(mac)
	if err != nil {
		log.Fatalf("Failed to create connection: %v", err)
	}
	defer device.Disconnect()

	if err := device.Connect(); err != nil {
		log.Fatalf("Failed to establish BLE connection: %v", err)
	}

	if err := device.WaitForConnection(10 * time.Second); err != nil {
		log.Fatalf("Connection timeout: %v", err)
	}

	fmt.Println("✓ Connected successfully!")
	fmt.Println("")

	// Display all current values
	values := device.GetAllValues()
	displayValues(values)

	// Monitor real-time updates for 10 seconds
	fmt.Println("\nMonitoring live updates (10 seconds)...")
	fmt.Println("--------------------------------------------------")

	done := make(chan struct{})
	time.AfterFunc(10*time.Second, func() {
		close(done)
	})

	updateCount := 0
	for {
		select {
		case <-done:
			fmt.Printf("\n\nMonitoring complete! Total updates received: %d\n", updateCount)
			return
		case values := <-device.SubscribeBatched():
			updateCount++
			displayKeyMetrics(values)
		}
	}
}

// displayValues prints all available sensor values with units
func displayValues(values map[string]victron.Value) {
	fmt.Println("Current Sensor Values:")
	fmt.Println("--------------------------------------------------")

	for _, key := range []string{
		victron.KeySystemVoltage, victron.KeySystemCurrent, victron.KeySystemPower,
		victron.KeyPVInputVoltage1, victron.KeyPVInputCurrent1, victron.KeyPVInputPower1,
		victron.KeyBatteryVoltage, victron.KeyBatteryCurrent, victron.KeyStateOfCharge,
	} {
		if val, exists := values[key]; exists {
			displayValue := val.RawValue
			if displayValue == "" {
				displayValue = fmt.Sprintf("%.2f", val.FloatValue)
			}
			fmt.Printf("  %-30s: %s%s\n", key, displayValue, getUnit(key))
		}
	}
	fmt.Println("")
}

// displayKeyMetrics prints just the important metrics inline
func displayKeyMetrics(values map[string]victron.Value) {
	voltage := getValue(values, victron.KeySystemVoltage)
	current := getValue(values, victron.KeySystemCurrent)
	power := getValue(values, victron.KeySystemPower)

	vStr := voltage.RawValue
	if vStr == "" {
		vStr = fmt.Sprintf("%.2f", voltage.FloatValue)
	}
	cStr := current.RawValue
	if cStr == "" {
		cStr = fmt.Sprintf("%.2f", current.FloatValue)
	}
	pStr := power.RawValue
	if pStr == "" {
		pStr = fmt.Sprintf("%.2f", power.FloatValue)
	}

	fmt.Printf("\r  %s: %s%s | %s: %s%s | %s: %s%s",
		victron.KeySystemVoltage, vStr, getUnit(victron.KeySystemVoltage),
		victron.KeySystemCurrent, cStr, getUnit(victron.KeySystemCurrent),
		victron.KeySystemPower, pStr, getUnit(victron.KeySystemPower))
}

// getValue retrieves a value by key or returns an empty value
func getValue(values map[string]victron.Value, key string) victron.Value {
	if val, exists := values[key]; exists {
		return val
	}
	return victron.Value{}
}

// getUnit returns the appropriate unit string for a given key
func getUnit(key string) string {
	units := map[string]string{
		victron.KeySystemVoltage:   " V",
		victron.KeySystemCurrent:   " A",
		victron.KeySystemPower:     " W",
		victron.KeyPVInputVoltage1: " V",
		victron.KeyPVInputCurrent1: " A",
		victron.KeyPVInputPower1:   " W",
		victron.KeyBatteryVoltage:  " V",
		victron.KeyBatteryCurrent:  " A",
		victron.KeyStateOfCharge:   "%",
	}
	return units[key]
}
