//go:build darwin

// macOS-specific BLE backend implementation using Python bleak library
package victron

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/scottstg/yolo/victron/ble"
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

				devices = append(devices, BLEDevice{
					Address: macAddr,
					Name:    name,
					RSSI:    -80, // Default RSSI
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
func parseJSONOutput(jsonStr string) []BLEDevice {
	var devices []BLEDevice

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

			devices = append(devices, BLEDevice{
				Address: address,
				Name:    name,
				RSSI:    -80,
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
func (m *MacOSBackend) Connect(address string) (ble.Connection, error) {
	if !m.initialized {
		return nil, fmt.Errorf("backend not initialized")
	}

	conn := &MacOSConnection{
		address: address,
		closed:  false,
	}

	// Test connection by discovering services
	services, err := discoverServicesWithPython(address)
	if err != nil {
		return nil, fmt.Errorf("failed to discover services: %w", err)
	}

	conn.services = services
	return conn, nil
}

// MacOSConnection represents a BLE connection on macOS
type MacOSConnection struct {
	address  string
	uuid     string
	services []ble.Service
	closed   bool
	mu       sync.Mutex
}

// Address returns the device address
func (mc *MacOSConnection) Address() string {
	return mc.address
}

// UUID returns the device UUID if available
func (mc *MacOSConnection) UUID() string {
	return mc.uuid
}

// IsConnected checks if connection is active
func (mc *MacOSConnection) IsConnected() bool {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	return !mc.closed
}

// Close disconnects from the device
func (mc *MacOSConnection) Close() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.closed = true
	return nil
}

// ReadValue reads a characteristic value
func (mc *MacOSConnection) ReadValue(char ble.Characteristic) ([]byte, error) {
	mc.mu.Lock()
	isClosed := mc.closed
	mc.mu.Unlock()

	if isClosed {
		return nil, fmt.Errorf("connection closed")
	}

	data, err := readCharacteristicWithPython(mc.address, char.UUID)
	if err != nil {
		return nil, fmt.Errorf("failed to read characteristic: %w", err)
	}

	return data, nil
}

// WriteValue writes to a characteristic
func (mc *MacOSConnection) WriteValue(char ble.Characteristic, data []byte) error {
	return fmt.Errorf("not implemented")
}

// SubscribeToNotifications subscribes to notifications from a characteristic
func (mc *MacOSConnection) SubscribeToNotifications(char ble.Characteristic) (<-chan []byte, error) {
	return nil, fmt.Errorf("not implemented")
}

// UnsubscribeFromNotifications stops receiving notifications
func (mc *MacOSConnection) UnsubscribeFromNotifications(char ble.Characteristic) error {
	return fmt.Errorf("not implemented")
}

// DiscoverServices discovers GATT services on a connected device
func (m *MacOSBackend) DiscoverServices(conn ble.Connection) ([]ble.Service, error) {
	if mc, ok := conn.(*MacOSConnection); ok {
		return mc.services, nil
	}
	return nil, fmt.Errorf("invalid connection type")
}

// DiscoverCharacteristics discovers characteristics in a service  
func (m *MacOSBackend) DiscoverCharacteristics(service ble.Service) ([]ble.Characteristic, error) {
	// Characteristics would need to be discovered separately via Python helper
	// For now return empty - could add discoverCharacteristicsWithPython()
	return nil, fmt.Errorf("not implemented")
}

// discoverServicesWithPython calls the Python connector script to discover GATT services
func discoverServicesWithPython(address string) ([]ble.Service, error) {
	cmd := exec.Command("python3", "scripts/victron_connect.py", address, "connect")
	var output strings.Builder
	cmd.Stdout = &output
	cmd.Stderr = &output

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("python script failed: %w (output: %s)", err, output.String())
	}

	// Parse JSON response
	var result struct {
		Success  bool     `json:"success"`
		Message  string   `json:"message"`
		Error    string   `json:"error"`
		Services []struct {
			UUID          string      `json:"uuid"`
			Characteristics []struct {
				UUID       string   `json:"uuid"`
				Properties []string `json:"properties"`
			} `json:"characteristics"`
		} `json:"services"`
	}

	if err := json.Unmarshal([]byte(output.String()), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("%s", result.Error)
	}

	services := make([]ble.Service, 0, len(result.Services))
	for _, svc := range result.Services {
		services = append(services, ble.Service{
			UUID:    svc.UUID,
			Primary: true,
		})
	}

	return services, nil
}

// readCharacteristicWithPython calls the Python connector script to read a characteristic value
func readCharacteristicWithPython(address, charUUID string) ([]byte, error) {
	cmd := exec.Command("python3", "scripts/victron_connect.py", address, "read_char", charUUID)
	var output strings.Builder
	cmd.Stdout = &output
	cmd.Stderr = &output

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("python script failed: %w (output: %s)", err, output.String())
	}

	// Parse JSON response
	var result struct {
		Success     bool   `json:"success"`
		Message     string `json:"message"`
		Error       string `json:"error"`
		ValueHex    string `json:"value_hex"`
		ValueBytes  []int  `json:"value_bytes"`
	}

	if err := json.Unmarshal([]byte(output.String()), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("%s", result.Error)
	}

	// Return the raw bytes
	if len(result.ValueBytes) > 0 {
		data := make([]byte, len(result.ValueBytes))
		for i, b := range result.ValueBytes {
			data[i] = byte(b)
		}
		return data, nil
	}

	// Fallback: parse hex string if bytes array is empty
	if result.ValueHex != "" {
		data, err := hexStringToBytes(result.ValueHex)
		if err != nil {
			return nil, fmt.Errorf("failed to parse hex value: %w", err)
		}
		return data, nil
	}

	return nil, fmt.Errorf("no value returned")
}

// ReadBatteryVoltage reads battery voltage from Glow device using Python helper
func ReadBatteryVoltage(address string) (map[string]float64, error) {
	cmd := exec.Command("python3", "scripts/victron_connect.py", address, "read_battery")
	var output strings.Builder
	cmd.Stdout = &output
	cmd.Stderr = &output

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("python script failed: %w (output: %s)", err, output.String())
	}

	// Parse JSON response
	var result struct {
		Error   string            `json:"error"`
		Results []BatteryReading  `json:"results"`
	}

	if err := json.Unmarshal([]byte(output.String()), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	if result.Error != "" {
		return nil, fmt.Errorf("%s", result.Error)
	}

	voltages := make(map[string]float64)
	for _, r := range result.Results {
		if r.VoltageV != nil {
			key := fmt.Sprintf("%s_%s", r.ServiceUUID[:8], r.CharUUID[:8])
			voltages[key] = *r.VoltageV
		}
	}

	return voltages, nil
}

// BatteryReading represents a battery reading from a characteristic
type BatteryReading struct {
	ServiceUUID string  `json:"service_uuid"`
	CharUUID    string  `json:"char_uuid"`
	ValueHex    string  `json:"value_hex"`
	ValueBytes  []int   `json:"value_bytes"`
	VoltageV    *float64 `json:"voltage_v"`
}

// hexStringToBytes converts a hex string to byte slice
func hexStringToBytes(hexStr string) ([]byte, error) {
	hexStr = strings.ReplaceAll(hexStr, " ", "")
	hexStr = strings.ReplaceAll(hexStr, ":", "")
	
	if len(hexStr)%2 != 0 {
		return nil, fmt.Errorf("invalid hex string length: %d", len(hexStr))
	}

	bytes := make([]byte, len(hexStr)/2)
	for i := 0; i < len(bytes); i++ {
		hexChar1 := hexStr[i*2]
		hexChar2 := hexStr[i*2+1]

		val1 := hexValue(hexChar1)
		val2 := hexValue(hexChar2)

		if val1 == 255 || val2 == 255 {
			return nil, fmt.Errorf("invalid hex character at position %d", i*2)
		}

		bytes[i] = (val1 << 4) | val2
	}

	return bytes, nil
}

func hexValue(c byte) byte {
	if c >= '0' && c <= '9' {
		return c - '0'
	}
	if c >= 'a' && c <= 'f' {
		return c - 'a' + 10
	}
	if c >= 'A' && c <= 'F' {
		return c - 'A' + 10
	}
	return 255
}
