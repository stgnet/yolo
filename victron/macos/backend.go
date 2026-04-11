//go:build darwin

// macOS-specific BLE backend implementation using Python bleak library
package macos

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/scottstg/yolo/victron/ble"
)

// Backend implements BLEBackend for macOS
type Backend struct {
	initialized bool
	scanning    bool
}

// New creates a new macOS BLE backend
func New() (*Backend, error) {
	return &Backend{}, nil
}

// Initialize initializes the BLE adapter
func (b *Backend) Initialize() error {
	b.initialized = true
	return nil
}

// Close shuts down the BLE connection
func (b *Backend) Close() error {
	if b.scanning {
		b.scanning = false
	}
	b.initialized = false
	return nil
}

// ScanForDevices scans for nearby BLE devices using Python bleak library
func (b *Backend) ScanForDevices(ctx context.Context, duration time.Duration) ([]ble.Device, error) {
	if !b.initialized {
		return nil, fmt.Errorf("backend not initialized")
	}

	b.scanning = true
	defer func() { b.scanning = false }()

	// Use Python's bleak library which works well on macOS with CoreBluetooth
	devices, err := scanWithPython(ctx, duration)
	if err != nil || len(devices) == 0 {
		return devices, fmt.Errorf("no bluetooth devices found: %w", err)
	}

	return devices, nil
}

// scanWithPython uses Python's bleak library to scan for BLE devices
func scanWithPython(ctx context.Context, duration time.Duration) ([]ble.Device, error) {
	durationSeconds := int(duration.Seconds())
	if durationSeconds < 1 {
		durationSeconds = 10
	}
	if durationSeconds > 60 {
		durationSeconds = 60
	}

	// Run the Python scanner script
	cmd := exec.Command("python3", "scripts/victron_scan.py", fmt.Sprintf("%d", durationSeconds))
	var output strings.Builder
	cmd.Stdout = &output
	cmd.Stderr = &output

	err := cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start scanner: %w", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-done:
		result := output.String()
		return parseScanOutput(result)
	case <-time.After(time.Duration(durationSeconds+5) * time.Second):
		cmd.Process.Kill()
		<-done
		result := output.String()
		if result != "" {
			devices, err := parseScanOutput(result)
			if err == nil {
				return devices, nil // Return partial results
			}
		}
		return nil, fmt.Errorf("command timed out after %d seconds", durationSeconds+5)
	}
}

// parseScanOutput parses JSON output from Python BLE scanner
func parseScanOutput(output string) ([]ble.Device, error) {
	// First try to find inline Python code results (old format)
	lines := strings.Split(output, "\n")
	var devices []ble.Device

	// Check if any line contains JSON (new format)
	hasJSON := false
	for _, line := range lines {
		if strings.Contains(line, `"devices"`) || strings.HasPrefix(strings.TrimSpace(line), "{") {
			hasJSON = true
			break
		}
	}

	if hasJSON {
		// Parse JSON output from script - combine all lines into one string
		devices = parseJSONOutput(output)
	} else {
		// Try old pipe-delimited format for backwards compatibility
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "Error:") || strings.HasPrefix(line, "Traceback") || strings.HasPrefix(line, "Scanning for") {
				continue
			}

			parts := strings.SplitN(line, "|", 2)
			if len(parts) >= 1 {
				macAddr := parts[0]
				name := ""
				if len(parts) > 1 {
					name = parts[1]
				}

				// Skip if doesn't look like a MAC address
				if !isValidMAC(macAddr) {
					continue
				}

				devices = append(devices, ble.Device{
					Address: macAddr,
					Name:    name,
				})
			}
		}
	}

	if len(devices) == 0 {
		return nil, fmt.Errorf("no valid devices found")
	}

	return devices, nil
}

// parseJSONOutput parses JSON output from the Python scanner script
func parseJSONOutput(jsonStr string) []ble.Device {
	var devices []ble.Device

	// Find all device objects in the JSON array
	// Look for "address" and extract both address and name
	
	for {
		addrKey := strings.Index(jsonStr, `"address":`)
		if addrKey == -1 {
			break
		}

		// Find the colon after "address":
		colonPos := addrKey + len(`"address":`)

		// Skip whitespace
		for colonPos < len(jsonStr) && (jsonStr[colonPos] == ' ' || jsonStr[colonPos] == '\t') {
			colonPos++
		}

		// Should be opening quote for address value
		if colonPos >= len(jsonStr) || jsonStr[colonPos] != '"' {
			break
		}
		start := colonPos + 1

		// Find closing quote
		end := start
		for end < len(jsonStr) && jsonStr[end] != '"' {
			end++
		}

		if end >= len(jsonStr) {
			break
		}

		address := jsonStr[start:end]

		// Validate address format
		if isValidMAC(address) {
			// Try to extract name from this device object
			// The name field should be near the address field in JSON
			name := ""
			
			// Search for "name": within next 200 chars after address
			searchStart := addrKey
			searchEnd := searchStart + 200
			if searchEnd > len(jsonStr) {
				searchEnd = len(jsonStr)
			}
			searchArea := jsonStr[searchStart:searchEnd]
			
			nameKey := strings.Index(searchArea, `"name":`)
			if nameKey != -1 {
				// Found name field, extract value
				nameColonPos := nameKey + len(`"name":`)
				for nameColonPos < len(searchArea) && (searchArea[nameColonPos] == ' ' || searchArea[nameColonPos] == '\t') {
					nameColonPos++
				}
				
				if nameColonPos < len(searchArea) && searchArea[nameColonPos] == '"' {
					nameStart := nameColonPos + 1
					nameEnd := nameStart
					for nameEnd < len(searchArea) && searchArea[nameEnd] != '"' {
						nameEnd++
					}
					if nameEnd < len(searchArea) {
						name = searchArea[nameStart:nameEnd]
					}
				}
			}

			devices = append(devices, ble.Device{
				Address: address,
				Name:    name,
			})
		}

		// Move past this match to find next
		jsonStr = jsonStr[end+1:]
	}

	return devices
}

// isValidMAC checks if a string looks like a valid MAC address or BLE UUID
// Accepts: XX:XX:XX:XX:XX:XX (traditional) or XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX (UUID/macOS)
func isValidMAC(mac string) bool {
	// Check for traditional MAC format: XX:XX:XX:XX:XX:XX (17 chars with colons)
	if len(mac) == 17 {
		for i, c := range mac {
			if i%3 == 2 {
				if c != ':' {
					return false
				}
				continue
			}
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
		return true
	}

	// Check for UUID format used by macOS CoreBluetooth: XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX (36 chars)
	// Format: 8-4-4-4-12 hex digits with dashes
	if len(mac) == 36 {
		dashPositions := []int{8, 13, 18, 23}
		dashIdx := 0
		
		for i, c := range mac {
			// Check for dash at expected positions
			if dashIdx < len(dashPositions) && i == dashPositions[dashIdx] {
				if c != '-' {
					return false
				}
				dashIdx++
				continue
			}
			// Must be hex digit otherwise
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
		return dashIdx == len(dashPositions) // All dashes found in right positions
	}

	return false
}

// Connect establishes a connection to a device
// Connect establishes a connection to a device
func (b *Backend) Connect(address string) (ble.Connection, error) {
	return nil, fmt.Errorf("connect not yet implemented for macOS")
}

// DiscoverServices discovers GATT services on a connected device
func (b *Backend) DiscoverServices(conn ble.Connection) ([]ble.Service, error) {
	return nil, fmt.Errorf("not implemented")
}

// DiscoverCharacteristics discovers characteristics in a service
func (b *Backend) DiscoverCharacteristics(service ble.Service) ([]ble.Characteristic, error) {
	return nil, fmt.Errorf("not implemented")
}