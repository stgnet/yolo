// Example usage of the Victron BLE library
package main

import (
	"fmt"
	"time"
	
	"github.com/scottstg/yolo/victron"
)

func main() {
	// Create a new client
	client := victron.NewClient()
	
	fmt.Println("Scanning for Victron devices...")
	
	// Scan for nearby Victron devices (scans for 15 seconds)
	results, err := client.Scan(15 * time.Second)
	if err != nil {
		fmt.Printf("Scan error: %v\n", err)
		return
	}
	
	if len(results) == 0 {
		fmt.Println("No Victron devices found!")
		return
	}
	
	fmt.Printf("Found %d device(s):\n", len(results))
	for i, device := range results {
		fmt.Printf("  %d. %s (MAC: %s, RSSI: %ddBm)\n", 
			i+1, device.Name, device.Address, device.RSSI)
	}
	
	// Connect to the first device found
	if len(results) > 0 {
		deviceAddress := results[0].Address
		
		fmt.Printf("\nConnecting to %s...\n", deviceAddress)
		
		victronDevice, err := client.Connect(deviceAddress)
		if err != nil {
			fmt.Printf("Connection error: %v\n", err)
			return
		}
		
		// Establish the connection
		err = victronDevice.Connect()
		if err != nil {
			fmt.Printf("Failed to connect: %v\n", err)
			return
		}
		
		defer victronDevice.Disconnect()
		
		fmt.Println("Connected! Receiving data...")
		
		// Subscribe to real-time updates
		valueChan := victronDevice.Subscribe()
		
		// Process values for 30 seconds
		done := time.After(30 * time.Second)
		
		for {
			select {
			case val := <-valueChan:
				// Print the received value
				info := victron.GetValueType(val.Key)
				if info != nil {
					fmt.Printf("[%s] %s: %.3f %s\n", 
						val.Timestamp.Format("15:04:05"),
						info.Name, val.FloatValue, info.Unit)
				} else {
					fmt.Printf("[%s] %s: %s\n", 
						val.Timestamp.Format("15:04:05"),
						val.Key, val.RawValue)
				}
				
			case <-done:
				fmt.Println("\nDone!")
				return
			}
		}
	}
}
