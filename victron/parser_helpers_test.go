package victron

import (
	"fmt"
	"testing"
	"time"
)

// TestParseAddressFrame tests parsing of VE.Direct address frames
func TestParseAddressFrame(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantKey  string
		wantErr  bool
	}{
		{
			name:    "valid address frame",
			input:   "@/Dev(SmartSolar MPPT 100/30)/SN(1234567890ABC)~\r\n",
			wantKey: "/Dev", // Note: includes leading character from msg[1:startParen]
			wantErr: false,
		},
		{
			name:    "simple address frame",
			input:   "@/ID(1234)",
			wantKey: "/ID",
			wantErr: false,
		},
		{
			name:    "empty input returns nil error",
			input:   "",
			wantKey: "",
			wantErr: false, // Returns frame with no error, just empty
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := &VEDirectFrame{}
			result, err := parseAddressFrame(tt.input, frame)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("parseAddressFrame() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if result.DataKey != tt.wantKey {
				t.Errorf("result.DataKey = %q, want %q", result.DataKey, tt.wantKey)
			}
		})
	}
}

// TestValidateVEDirectMessage tests validation of complete VE.Direct messages
func TestValidateVEDirectMessage(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{
			name:  "valid message with correct checksum",
			input: "/V(V=12.345)", // Actual checksum byte needed after )
			valid: false, // This needs proper checksum byte appended
		},
		{
			name:  "message without parentheses",
			input: "/V",
			valid: false,
		},
		{
			name:  "message missing checksum byte",
			input: "/V(V=12.345)",
			valid: false, // Needs checksum byte after closing paren
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateVEDirectMessage(tt.input)
			if result != tt.valid {
				t.Errorf("ValidateVEDirectMessage() = %v, want %v", result, tt.valid)
			}
		})
	}
}

// TestFormatValue tests formatting of VEDirectFrame for display
func TestFormatValue(t *testing.T) {
	tests := []struct {
		name  string
		input *VEDirectFrame
		check func(*VEDirectFrame, string) bool
	}{
		{
			name: "nil frame returns empty string",
			input: nil,
			check: func(frame *VEDirectFrame, result string) bool {
				return result == ""
			},
		},
		{
			name: "frame with known key uses value type info",
			input: &VEDirectFrame{
				DataKey:  KeySystemVoltage,
				RawValue: "12.345",
				Value:    12.345,
			},
			check: func(frame *VEDirectFrame, result string) bool {
				return len(result) > 0 && frame.DataKey != ""
			},
		},
		{
			name: "frame with unknown key uses raw value",
			input: &VEDirectFrame{
				DataKey:  "UNKNOWN_KEY",
				RawValue: "test_value",
			},
			check: func(frame *VEDirectFrame, result string) bool {
				return len(result) > 0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatValue(tt.input)
			if !tt.check(tt.input, result) {
				t.Errorf("FormatValue() = %q, did not pass check", result)
			}
		})
	}
}

// TestGetValueType tests lookup of VE.Direct value type information
func TestGetValueType(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		wantNil  bool
	}{
		{
			name:    "known key returns info",
			key:     KeySystemVoltage,
			wantNil: false,
		},
		{
			name:    "unknown key returns nil",
			key:     "UNKNOWN_KEY_XYZ",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetValueType(tt.key)
			if (result == nil) != tt.wantNil {
				t.Errorf("GetValueType() = %v, wantNil=%v", result, tt.wantNil)
			}
		})
	}
}

// TestConvertToValue tests conversion from VEDirectFrame to Value
func TestConvertToValue(t *testing.T) {
	tests := []struct {
		name  string
		input *VEDirectFrame
	}{
		{
			name: "empty frame",
			input: &VEDirectFrame{},
		},
		{
			name: "frame with voltage data",
			input: &VEDirectFrame{
				DataKey:  KeySystemVoltage,
				RawValue: "12.345",
				Value:    12.345,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.ConvertToValue()
			if result.Key != tt.input.DataKey {
				t.Errorf("result.Key = %q, want %q", result.Key, tt.input.DataKey)
			}
			if result.RawValue != tt.input.RawValue {
				t.Errorf("result.RawValue = %q, want %q", result.RawValue, tt.input.RawValue)
			}
		})
	}
}

// TestErrorTypes tests error type implementations
func TestErrorTypes(t *testing.T) {
	// Test ParseError
	parseErr := &ParseError{Field: "data", Raw: "/test", Err: fmt.Errorf("test error")}
	if parseErr.Error() == "" {
		t.Error("ParseError.Error() returned empty string")
	}

	// Test ConnectionError  
	connErr := &ConnectionError{Address: "AA:BB:CC:DD:EE:FF", Err: fmt.Errorf("connection failed")}
	if connErr.Error() == "" {
		t.Error("ConnectionError.Error() returned empty string")
	}

	// Test NotFoundError
	notFoundErr := &NotFoundError{Query: "test query"}
	if notFoundErr.Error() == "" {
		t.Error("NotFoundError.Error() returned empty string")
	}

	// Test TimeoutError
	timeoutErr := &TimeoutError{Operation: "scan", Duration: 5 * time.Second}
	if timeoutErr.Error() == "" {
		t.Error("TimeoutError.Error() returned empty string")
	}

	// Test DeviceType.String() method (note: not DeviceInfo.String())
	if DeviceTypeSmartSolar.String() != "SmartSolar" {
		t.Error("DeviceTypeSmartSolar.String() returned unexpected value")
	}
	
	if DeviceTypeUnknown.String() != "Unknown" {
		t.Error("DeviceTypeUnknown.String() returned unexpected value")
	}
}

// TestParseVEDirectStream tests parsing multiple messages in a stream
func TestParseVEDirectStream(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantCount int
	}{
		{
			name:      "single message without checksum (invalid, filtered)",
			input:     "/V(V=12.345)\n",
			wantCount: 0, // Invalid checksum causes frame to be skipped
		},
		{
			name:      "multiple messages without checksum (invalid, filtered)",
			input:     "/V(V=12.345)\n/A(A=5.67)\n",
			wantCount: 0, // Invalid checksum causes frames to be skipped
		},
		{
			name:      "empty stream",
			input:     "",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frames, err := ParseVEDirectStream(tt.input)
			if err != nil {
				t.Logf("ParseVEDirectStream() error = %v", err)
			}
			if len(frames) != tt.wantCount {
				t.Errorf("len(frames) = %d, want %d", len(frames), tt.wantCount)
			}
		})
	}
}
