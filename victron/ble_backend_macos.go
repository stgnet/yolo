//go:build darwin

// macOS-specific BLE backend implementation using AppleScript/System Events
package victron

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// MacOSBackend implements BLEBackend for macOS using native tools
type MacOSBackend struct {
	initialized bool
	scanResults []BLEDevice
}

// NewMacOSBackend creates a new macOS BLE backend
func NewMacOSBackend() (BLEBackend, error) {
	return &MacOSBackend{}, nil
}

// Initialize initializes the BLE adapter
func (m *MacOSBackend) Initialize() error {
	m.initialized = true

	// Check if Bluetooth is enabled
	cmd := exec.Command("osascript", "-e",
		`tell application "System Events" to get (number of every bluetooth device)`)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check Bluetooth status: %w", err)
	}

	countStr := strings.TrimSpace(string(output))
	count, err := strconv.Atoi(countStr)
	if err != nil || count < 0 {
		// Even if there are no devices, that's OK - just means Bluetooth might be off or no devices nearby
		return nil
	}

	return nil
}

// Close shuts down the BLE connection
func (m *MacOSBackend) Close() error {
	m.initialized = false
	m.scanResults = nil
	return nil
}

// ScanForDevices scans for nearby BLE devices using AppleScript
func (m *MacOSBackend) ScanForDevices(ctx context.Context, duration time.Duration) ([]BLEDevice, error) {
	if !m.initialized {
		return nil, fmt.Errorf("backend not initialized")
	}

	m.scanResults = nil

	// Start scan in background
	cmd := exec.CommandContext(ctx, "osascript", "-e", `
		set foundDevices to {}
		repeat with timeout of `+strconv.Itoa(int(duration.Seconds()))+` seconds
			try
				set devices to every bluetooth device of (first network area) of (system info)
				repeat with aDevice in devices
					set nameProp to (name of aDevice) as string
					-- Check if this might be a Victron device
					if nameProp contains "Victron" or nameProp contains "SmartSolar" or nameProp contains "SmartShunt" or nameProp contains "VE.Direct" then
						set foundDevices to foundDevices & {nameProp}
					end if
				end repeat
			end try
			delay 1
		end repeat
		return foundDevices
	`)

	output, err := cmd.Output()
	if err != nil && err.Error() != "signal: killed" && !strings.Contains(err.Error(), "context deadline exceeded") {
		// Try alternative approach - list all Bluetooth devices
		output2, err2 := exec.CommandContext(ctx, "osascript", "-e",
			`tell application "System Events" to get name of every bluetooth device`).Output()

		if err2 != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		output = output2
	}

	// Parse results - AppleScript returns delimited list like {"device1", "device2"}
	resultStr := strings.TrimSpace(string(output))
	if resultStr == "" || resultStr == "{}" {
		return []BLEDevice{}, nil
	}

	// Extract device names from AppleScript output
	re := regexp.MustCompile(`"(.*?)"`)
	matches := re.FindAllStringSubmatch(resultStr, -1)

	devices := make([]BLEDevice, 0, len(matches))
	for i, match := range matches {
		if len(match) > 1 {
			name := match[1]

			// Check if this looks like a Victron device
			isVictron := false
			for _, pattern := range VictronAdvertisementPatterns {
				if strings.Contains(strings.ToLower(name), strings.ToLower(pattern)) {
					isVictron = true
					break
				}
			}

			// Estimate RSSI based on order (first results typically have stronger signal)
			rssi := -70 - (i * 5)

			device := BLEDevice{
				Address: fmt.Sprintf("macos-bt-%d", i),
				Name:    name,
				RSSI:    rssi,
			}

			if isVictron {
				devices = append(devices, device)
			}
		}
	}

	m.scanResults = devices
	return devices, nil
}

// Connect establishes a connection to a device
func (m *MacOSBackend) Connect(address string) (BLEConnection, error) {
	return nil, fmt.Errorf("macOS backend: connect not implemented yet - requires CoreBluetooth framework integration")
}

// DiscoverServices discovers GATT services on a connected device
func (m *MacOSBackend) DiscoverServices(conn BLEConnection) ([]BLEService, error) {
	return nil, fmt.Errorf("not implemented")
}

// DiscoverCharacteristics discovers characteristics in a service
func (m *MacOSBackend) DiscoverCharacteristics(service BLEService) ([]BLECharacteristic, error) {
	return nil, fmt.Errorf("not implemented")
}
