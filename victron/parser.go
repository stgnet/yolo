// VE.Direct protocol parser
// Victron VE.Direct uses ASCII-based messages with a specific format
// This parser handles the format regardless of transport (serial or BLE)
package victron

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// VEDirectFrame represents a parsed VE.Direct message frame
type VEDirectFrame struct {
	RawMessage   string // Original raw message
	DeviceType   string // Device identifier (/Dev, etc.)
	DataKey      string // Data field key (V, A, P.W, etc.)
	RawValue     string // Raw value as received
	Value        float64 // Parsed numeric value if applicable
	Checksum     byte   // Checksum byte from message
	Valid        bool   // Whether checksum is valid
}

// VE DIRECT Protocol format:
// Format: [device][field(value)checksum
// Example: /V(V=12.345)A  (voltage = 12.345V)
// 
// Components:
// - First character after [/] indicates device type or message type
//   - / = VE.Direct data frame
//   - @ = Address frame
// - Field identifier (V, A, P.W, etc.)
// - Value in parentheses (value)
// - Single byte checksum

// ParseVEDirectMessage parses a single VE.Direct message string into structured data
func ParseVEDirectMessage(msg string) (*VEDirectFrame, error) {
	if len(msg) == 0 {
		return nil, &ParseError{Field: "message", Raw: "", Err: fmt.Errorf("empty message")}
	}

	frame := &VEDirectFrame{
		RawMessage: msg,
		Valid:      true,
	}

	// VE.Direct messages start with / or @
	if len(msg) < 2 {
		return frame, nil
	}

	// First character indicates message type
	switch msg[0] {
	case '/':
		// Data frame - contains actual telemetry
		frame.DeviceType = "data"
		return parseDataFrame(msg, frame)
	case '@':
		// Address/ID frame - device identification
		frame.DeviceType = "address"
		return parseAddressFrame(msg, frame)
	default:
		// Unknown frame type
		frame.Valid = false
		return frame, fmt.Errorf("unknown message type byte: %c", msg[0])
	}
}

func parseDataFrame(msg string, frame *VEDirectFrame) (*VEDirectFrame, error) {
	// Format: /[field(value)checksum
	// Example: /V(V=12.5)A
	
	if len(msg) < 4 {
		frame.Valid = false
		return frame, fmt.Errorf("message too short")
	}

	// Find the field name and value in parentheses
	startParen := strings.Index(msg, "(")
	endParen := strings.Index(msg, ")")

	if startParen == -1 || endParen == -1 || endParen <= startParen {
		frame.Valid = false
		return frame, fmt.Errorf("invalid message format: missing or malformed parentheses")
	}

	// Extract field key (between first char and opening paren)
	field := msg[1:startParen]
	frame.DataKey = field

	// Extract value (inside parentheses)
	valueStr := msg[startParen+1 : endParen]
	
	// Value might be "key=value" or just "value"
	parts := strings.SplitN(valueStr, "=", 2)
	if len(parts) == 2 {
		frame.DataKey = parts[0]
		frame.RawValue = parts[1]
	} else {
		frame.RawValue = valueStr
	}

	// Parse numeric value if possible
	if frame.RawValue != "" {
		// Handle special values
		switch strings.ToUpper(frame.RawValue) {
		case "NAN", "-":
			// Not a number or not available
			frame.Value = 0
		default:
			val, err := strconv.ParseFloat(frame.RawValue, 64)
			if err == nil {
				frame.Value = val
			}
		}
	}

	// Extract checksum byte (last character before newline if present)
	checksumPos := endParen + 1
	if checksumPos < len(msg) {
		frame.Checksum = msg[checksumPos]
		
		// Validate checksum (XOR of all bytes from start to end paren)
		expectedChecksum := calculateChecksum([]byte(msg[:endParen+1]))
		if frame.Checksum != expectedChecksum {
			frame.Valid = false
		}
	} else {
		frame.Valid = false
		return frame, fmt.Errorf("missing checksum byte")
	}

	return frame, nil
}

func parseAddressFrame(msg string, frame *VEDirectFrame) (*VEDirectFrame, error) {
	// Format: /[field(value)checksum
	// Used for device identification, firmware version, etc.
	
	// Similar parsing to data frame but different interpretation
	startParen := strings.Index(msg, "(")
	endParen := strings.Index(msg, ")")

	if startParen == -1 || endParen == -1 {
		return frame, nil // Less strict for address frames
	}

	field := msg[1:startParen]
	frame.DataKey = field
	
	valueStr := msg[startParen+1 : endParen]
	parts := strings.SplitN(valueStr, "=", 2)
	if len(parts) == 2 {
		frame.DataKey = parts[0]
		frame.RawValue = parts[1]
	} else {
		frame.RawValue = valueStr
	}

	return frame, nil
}

// calculateChecksum calculates VE.Direct checksum (XOR of all bytes)
func calculateChecksum(data []byte) byte {
	if len(data) == 0 {
		return 0
	}
	
	checksum := byte(0)
	for _, b := range data {
		checksum ^= b
	}
	return checksum
}

// ValidateVEDirectMessage validates the checksum of a VE.Direct message
func ValidateVEDirectMessage(msg string) bool {
	endParen := strings.Index(msg, ")")
	if endParen == -1 || endParen >= len(msg)-1 {
		return false
	}
	
	expectedChecksum := calculateChecksum([]byte(msg[:endParen+1]))
	actualChecksum := msg[endParen+1]
	
	return expectedChecksum == actualChecksum
}

// ParseVEDirectStream parses a stream of VE.Direct messages (may contain multiple frames)
func ParseVEDirectStream(data string) ([]*VEDirectFrame, error) {
	// Split by newline or carriage return
	lines := strings.Split(data, "\n")
	var frames []*VEDirectFrame
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		
		// Remove carriage return if present
		line = strings.TrimRight(line, "\r")
		
		frame, err := ParseVEDirectMessage(line)
		if err != nil && frame != nil && !frame.Valid {
			// Log or handle invalid frame
			continue
		}
		
		if frame != nil {
			frames = append(frames, frame)
		}
	}
	
	return frames, nil
}

// VEDirectValueTypes defines known VE.Direct data keys and their properties
var VEDirectValueTypes = map[string]ValueTypeInfo{
	// Voltage values (in volts, sometimes scaled by 1000)
	KeySystemVoltage:       {Name: "System Voltage", Unit: "V", Scale: 1},
	KeyBatteryVoltage:      {Name: "Battery Voltage", Unit: "V", Scale: 1},
	KeyPVInputVoltage1:     {Name: "PV Input Voltage 1", Unit: "V", Scale: 1},
	KeyPVInputVoltage2:     {Name: "PV Input Voltage 2", Unit: "V", Scale: 1},
	
	// Current values (in amps, sometimes scaled by 1000)
	KeySystemCurrent:       {Name: "System Current", Unit: "A", Scale: 1},
	KeyBatteryCurrent:      {Name: "Battery Current", Unit: "A", Scale: 1},
	KeyPVInputCurrent1:     {Name: "PV Input Current 1", Unit: "A", Scale: 1},
	KeyPVInputCurrent2:     {Name: "PV Input Current 2", Unit: "A", Scale: 1},
	
	// Power values (in watts)
	KeySystemPower:         {Name: "System Power", Unit: "W", Scale: 1},
	KeyPVInputPower1:       {Name: "PV Input Power 1", Unit: "W", Scale: 1},
	KeyPVInputPower2:       {Name: "PV Input Power 2", Unit: "W", Scale: 1},
	
	// State of charge (percentage)
	KeyStateOfCharge:       {Name: "State of Charge", Unit: "%", Scale: 1},
	
	// Energy values
	KeyEnergyYieldToday:    {Name: "Energy Yield Today", Unit: "Ah", Scale: 1},
	KeyEnergyWhToday:       {Name: "Energy Yield Today", Unit: "Wh", Scale: 1},
	
	// Temperature (in Celsius)
	KeyTemperature:         {Name: "Device Temperature", Unit: "°C", Scale: 1},
	
	// Algorithm/State values
	KeyAlgorithmState:      {Name: "Charging Algorithm State", Unit: "", Scale: 0},
	KeyStage:               {Name: "Stage", Unit: "", Scale: 0},
	
	// Error codes
	KeyErrorCode:           {Name: "Error Code", Unit: "", Scale: 0},
	
	// Device identification
	KeyDeviceVersion:       {Name: "Device Version", Unit: "", Scale: 0, StringType: true},
	KeyDeviceRevision:      {Name: "Device Revision", Unit: "", Scale: 0, StringType: true},
	KeySerialNumber:        {Name: "Serial Number", Unit: "", Scale: 0, StringType: true},
}

// ValueTypeInfo describes a known VE.Direct value type
type ValueTypeInfo struct {
	Name       string // Human-readable name
	Unit       string // Unit of measurement
	Scale      float64 // Scaling factor (multiply by this)
	StringType bool    // Whether this is a string value, not numeric
}

// GetValueType returns information about a known VE.Direct data key
func GetValueType(key string) *ValueTypeInfo {
	if info, ok := VEDirectValueTypes[key]; ok {
		return &info
	}
	return nil
}

// FormatValue formats a VE.Direct value with its unit and name
func FormatValue(frame *VEDirectFrame) string {
	if frame == nil || frame.DataKey == "" {
		return ""
	}
	
	info := GetValueType(frame.DataKey)
	if info != nil && !info.StringType {
		return fmt.Sprintf("%s: %.3f %s", info.Name, frame.Value*info.Scale, info.Unit)
	}
	
	return fmt.Sprintf("%s: %s", frame.DataKey, frame.RawValue)
}

// ConvertToValue converts a VEDirectFrame to the generic Value type
func (frame *VEDirectFrame) ConvertToValue() Value {
	return Value{
		Key:        frame.DataKey,
		RawValue:   frame.RawValue,
		FloatValue: frame.Value,
		Timestamp:  time.Now(),
	}
}
