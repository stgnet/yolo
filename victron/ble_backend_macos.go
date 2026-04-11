//go:build darwin

// macOS-specific BLE backend implementation using Python bleak library
package victron

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// MacOSBackend implements BLEBackend for macOS
type MacOSBackend struct {
	initialized bool
	scanning    bool
}

// NewMacOSBackend creates a new macOS BLE backend
func NewMacOSBackend() (BLEBackend, error) {
	return &MacOSBackend{}, nil
}

// Initialize initializes the BLE adapter
func (m *MacOSBackend) Initialize() error {
	m.initialized = true
	return nil
}

// Close shuts down the BLE connection
func (m *MacOSBackend) Close() error {
	if m.scanning {
		m.scanning = false
	}
	m.initialized = false
	return nil
}

// ScanForDevices scans for nearby BLE devices using Python bleak library
func (m *MacOSBackend) ScanForDevices(ctx context.Context, duration time.Duration) ([]BLEDevice, error) {
	if !m.initialized {
		return nil, fmt.Errorf("backend not initialized")
	}

	m.scanning = true
	defer func() { m.scanning = false }()

	// Use Python's bleak library which works well on macOS with CoreBluetooth
	devices, err := scanWithPython(ctx, duration)
	if err != nil || len(devices) == 0 {
		return devices, fmt.Errorf("no bluetooth devices found: %w", err)
	}

	return devices, nil
}

// scanWithPython uses Python's bleak library to scan for BLE devices
func scanWithPython(ctx context.Context, duration time.Duration) ([]BLEDevice, error) {
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
func parseScanOutput(output string) ([]BLEDevice, error) {
	// First try to find inline Python code results (old format)
	lines := strings.Split(output, "\n")
	var devices []BLEDevice
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Error:") || strings.HasPrefix(line, "Traceback") || strings.HasPrefix(line, "Scanning for") {
			continue
		}
		
		// Check if it's JSON (new format)
		if strings.HasPrefix(line, "{") && strings.Contains(line, "\"devices\"") {
			// Parse JSON output from script
			devices = parseJSONOutput(line)
			break
		}
		
		// Try old pipe-delimited format for backwards compatibility
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
			
			devices = append(devices, BLEDevice{
				Address: macAddr,
				Name:    name,
				RSSI:    -80, // Default RSSI
			})
		}
	}
	
	if len(devices) == 0 {
		return nil, fmt.Errorf("no valid devices found")
	}
	
	return devices, nil
}

// parseJSONOutput parses JSON output from the Python scanner script
func parseJSONOutput(jsonStr string) []BLEDevice {
	var devices []BLEDevice
	
	// Simple JSON parsing without external dependencies
	// Look for patterns like "address":"XX:XX:XX:XX:XX:XX" and extract them
	
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
		
		// Validate MAC address format
		if isValidMAC(address) {
			devices = append(devices, BLEDevice{
				Address: address,
				Name:    "", // Extracting name requires more complex JSON parsing
				RSSI:    -80,
			})
		}
		
		// Move past this match to find next
		jsonStr = jsonStr[end+1:]
	}
	
	return devices
}

// isValidMAC checks if a string looks like a valid MAC address (XX:XX:XX:XX:XX:XX format)
func isValidMAC(mac string) bool {
	if len(mac) != 17 { // XX:XX:XX:XX:XX:XX = 6*2 + 5 colons
		return false
	}
	
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


// Connect establishes a connection to a device
func (m *MacOSBackend) Connect(address string) (BLEConnection, error) {
	return nil, fmt.Errorf("connect not yet implemented for macOS")
}

// DiscoverServices discovers GATT services on a connected device
func (m *MacOSBackend) DiscoverServices(conn BLEConnection) ([]BLEService, error) {
	return nil, fmt.Errorf("not implemented")
}

// DiscoverCharacteristics discovers characteristics in a service  
func (m *MacOSBackend) DiscoverCharacteristics(service BLEService) ([]BLECharacteristic, error) {
	return nil, fmt.Errorf("not implemented")
}
