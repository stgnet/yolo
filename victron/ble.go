// BLE-related types and GATT constants for Victron devices
package victron

import (
	"fmt"
)

// Service UUIDs used by Victron devices over BLE GATT
// Note: These are known/common UUIDs, but may vary by device model
const (
	// Main Victron Service UUID - custom service for VE.Direct data
	VictronServiceUUID = "fdcd" // Standard format would be 0xfdcd or full UUID

	// Full 128-bit UUIDs (if needed)
	VictronServiceUUID128   = "0000fdcd-0000-1000-8000-00805f9b34fb"
	VEDirectDataServiceUUID = "0000ffe0-0000-1000-8000-00805f9b34fb" // Often used for custom services

	// Characteristic UUIDs
	VEDirectDataCharacteristic    = "0000ffe1-0000-1000-8000-00805f9b34fb" // Notification characteristic
	VEDirectControlCharacteristic = "0000ffe2-0000-1000-8000-00805f9b34fb" // Write characteristic (optional)

	// Alternative UUIDs that some Victron devices may use
	SmartDeviceServiceUUID = "0000ffd0-0000-1000-8000-00805f9b34fb"
)

// Advertisement data patterns for identifying Victron devices
var VictronAdvertisementPatterns = []string{
	"VE.Direct",
	"SmartSolar",
	"SmartShunt",
	"Victron",
	"Glow",      // Victron Glow Bluetooth transmitter
	"ECO-",      // ECO battery series (Victron)
	"Solar48",   // Victron Solar batteries
}

// GATTService represents a discovered GATT service
type GATTService struct {
	UUID            string
	StartHandle     uint16
	EndHandle       uint16
	Primary         bool
	Characteristics []*GATTCharacteristic
}

// GATTCharacteristic represents a discovered GATT characteristic
type GATTCharacteristic struct {
	UUID        string
	Handle      uint16
	ValueHandle uint16
	Properties  uint8 // Standard BLE characteristic properties (read, write, notify, indicate)
}

// CharacteristicPropertyFlags standard BLE property bits
const (
	PropBroadcast        = 0x01
	PropRead             = 0x02
	PropWriteWithoutResp = 0x04
	PropWrite            = 0x08
	PropNotify           = 0x10
	PropIndicate         = 0x20
	PropSignedWrite      = 0x40
	PropExtendedProps    = 0x80
)

// ScanFilter is used to filter BLE advertisements during scanning
type ScanFilter struct {
	Name        string // Device name pattern
	LocalName   string // Advertisement local name pattern
	ServiceUUID []byte // Service UUID to match in advertisement
}

// DiscoverableDevice represents a device found during BLE scan
type DiscoverableDevice struct {
	Address          string // MAC address
	Name             string // Advertised name
	RSSI             int    // Signal strength (negative dBm)
	ManufacturerData map[uint16][]byte
	ServiceData      map[string][]byte
	IsVictron        bool // Whether this appears to be a Victron device
}

// ParseAdvertisement determines if an advertisement is from a Victron device
func ParseAdvertisement(name string, serviceUUIDs []string) bool {
	// Check by name patterns
	for _, pattern := range VictronAdvertisementPatterns {
		if len(name) >= len(pattern) && containsIgnoreCase(name, pattern) {
			return true
		}
	}

	// Check by service UUID in advertisement
	for _, uuid := range serviceUUIDs {
		if containsIgnoreCase(uuid, "fdcd") ||
			containsIgnoreCase(uuid, "ffe0") ||
			containsIgnoreCase(uuid, VictronServiceUUID) {
			return true
		}
	}

	return false
}

func containsIgnoreCase(s, substr string) bool {
	sLower := toLowerSimple(s)
	subLower := toLowerSimple(substr)
	return len(sLower) >= len(subLower) &&
		(sLower == subLower || findSubstring(sLower, subLower))
}

func toLowerSimple(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + ('a' - 'A')
		} else {
			result[i] = c
		}
	}
	return string(result)
}

func findSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// NewScanFilter creates a scan filter for finding Victron devices
func NewScanFilter(deviceType DeviceType) ScanFilter {
	filter := ScanFilter{}

	switch deviceType {
	case DeviceTypeSmartSolar:
		filter.Name = "SmartSolar"
	case DeviceTypeSmartShunt:
		filter.Name = "SmartShunt"
	case DeviceTypeVEDirectAdapter:
		filter.Name = "VE.Direct"
	}

	return filter
}

// FormatServiceUUID formats a UUID string for BLE operations
func FormatServiceUUID(uuid string) string {
	// Remove hyphens if present for comparison
	uuid = removeHyphens(uuid)

	// If it's a 16-bit UUID (4 hex chars), return as-is or format properly
	if len(uuid) == 4 {
		return uuid
	}

	// If it's already 32 chars (128-bit without hyphens), use that
	if len(uuid) == 32 {
		return uuid
	}

	// Default to main Victron service
	return VictronServiceUUID
}

func removeHyphens(s string) string {
	result := make([]byte, 0, len(s))
	for _, c := range s {
		if c != '-' {
			result = append(result, byte(c))
		}
	}
	return string(result)
}

// DescribeCharacteristicProperties returns human-readable property description
func DescribeCharacteristicProperties(props uint8) string {
	var parts []string
	if props&PropRead != 0 {
		parts = append(parts, "read")
	}
	if props&PropWriteWithoutResp != 0 {
		parts = append(parts, "write-no-response")
	}
	if props&PropWrite != 0 {
		parts = append(parts, "write")
	}
	if props&PropNotify != 0 {
		parts = append(parts, "notify")
	}
	if props&PropIndicate != 0 {
		parts = append(parts, "indicate")
	}

	result := ""
	for i, p := range parts {
		if i > 0 {
			result += "|"
		}
		result += p
	}
	return result
}

// ValidateCharacteristic checks if a characteristic supports required operations
func ValidateCharacteristic(char *GATTCharacteristic, requireNotify bool) error {
	if char == nil {
		return &NotFoundError{Query: "characteristic"}
	}

	if requireNotify && (char.Properties&PropNotify == 0) && (char.Properties&PropIndicate == 0) {
		return fmt.Errorf("characteristic %s does not support notifications", char.UUID)
	}

	return nil
}

// FindNotificationCharacteristic finds a characteristic that supports notifications in a service
func FindNotificationCharacteristic(service *GATTService) (*GATTCharacteristic, error) {
	if service == nil {
		return nil, &NotFoundError{Query: "service"}
	}

	for _, char := range service.Characteristics {
		if char.Properties&(PropNotify|PropIndicate) != 0 {
			return char, nil
		}
	}

	return nil, fmt.Errorf("no notification characteristic found in service %s", service.UUID)
}
