package victron

import (
	"testing"
)

// TestParseVEDirectMessageComplete provides comprehensive coverage for ParseVEDirectMessage
func TestParseVEDirectMessageComplete(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValid bool
		wantErr   bool
		checkType string // Expected DeviceType ("data" or "address")
	}{
		{
			name:      "empty message returns error",
			input:     "",
			wantValid: false,
			wantErr:   true,
		},
		{
			name:      "single character message handled gracefully",
			input:     "/",
			wantValid: true, // Short message returns valid frame without error
			wantErr:   false,
		},
		{
			name:      "data frame starts with /",
			input:     "/V(12.345)",
			wantValid: false, // Will be invalid due to missing checksum
			wantErr:   true, // Missing checksum returns error
			checkType: "data",
		},
		{
			name:      "address frame starts with @",
			input:     "@/Dev(SmartSolar)",
			wantValid: true,
			wantErr:   false,
			checkType: "address",
		},
		{
			name:      "invalid message type byte",
			input:     "X(V=12.345)",
			wantValid: false,
			wantErr:   true,
		},
		{
			name:      "unknown character type",
			input:     "#(test)",
			wantValid: false,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame, err := ParseVEDirectMessage(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseVEDirectMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if frame == nil && !tt.wantErr {
				t.Error("ParseVEDirectMessage() returned nil frame when no error expected")
				return
			}

			if frame != nil && frame.Valid != tt.wantValid {
				t.Errorf("frame.Valid = %v, want %v", frame.Valid, tt.wantValid)
			}

			if frame != nil && tt.checkType != "" && frame.DeviceType != tt.checkType {
				t.Errorf("frame.DeviceType = %q, want %q", frame.DeviceType, tt.checkType)
			}
		})
	}
}

// TestParseDataFrameComplete provides comprehensive coverage for parseDataFrame
func TestParseDataFrameComplete(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantKey   string
		wantValue float64
		wantValid bool
		wantErr   bool
	}{
		{
			name:      "message too short",
			input:     "/V(1",
			wantKey:   "",
			wantValue: 0,
			wantValid: false,
			wantErr:   true,
		},
		{
			name:      "simple value without key",
			input:     "/V(12.345)",
			wantKey:   "V",
			wantValue: 12.345,
			wantValid: false, // Missing checksum
			wantErr:   true, // Returns error for missing checksum
		},
		{
			name:      "key=value format",
			input:     "/V(V=24.7)",
			wantKey:   "V",
			wantValue: 24.7,
			wantValid: false,
			wantErr:   true, // Returns error for missing checksum
		},
		{
			name:      "NAN string value",
			input:     "/A(NAN)",
			wantKey:   "A",
			wantValue: 0, // NAN converts to 0
			wantValid: false,
			wantErr:   true, // Returns error for missing checksum
		},
		{
			name:      "dash value for not available",
			input:     "/P(-)",
			wantKey:   "P",
			wantValue: 0, // Dash converts to 0
			wantValid: false,
			wantErr:   true, // Returns error for missing checksum
		},
		{
			name:      "negative value",
			input:     "/A(A=-5.25)",
			wantKey:   "A",
			wantValue: -5.25,
			wantValid: false,
			wantErr:   true, // Returns error for missing checksum
		},
		{
			name:      "missing opening paren",
			input:     "/V12.345)",
			wantKey:   "",
			wantValue: 0,
			wantValid: false,
			wantErr:   true,
		},
		{
			name:      "missing closing paren",
			input:     "/V(12.345",
			wantKey:   "",
			wantValue: 0,
			wantValid: false,
			wantErr:   true,
		},
		{
			name:      "parens in wrong order",
			input:     "/V)12.345(",
			wantKey:   "",
			wantValue: 0,
			wantValid: false,
			wantErr:   true,
		},
		{
			name:      "missing checksum byte",
			input:     "/V(12.345)",
			wantKey:   "V",
			wantValue: 12.345,
			wantValid: false,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := &VEDirectFrame{}
			result, err := parseDataFrame(tt.input, frame)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseDataFrame() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if result.DataKey != tt.wantKey {
				t.Errorf("result.DataKey = %q, want %q", result.DataKey, tt.wantKey)
			}

			if result.Value != tt.wantValue {
				t.Errorf("result.Value = %v, want %v", result.Value, tt.wantValue)
			}

			if result.Valid != tt.wantValid {
				t.Errorf("result.Valid = %v, want %v", result.Valid, tt.wantValid)
			}
		})
	}
}

// TestValidateVEDirectMessageCoverage provides full coverage for ValidateVEDirectMessage
func TestValidateVEDirectMessageCoverage(t *testing.T) {
	// Create a message with valid checksum
	createValidMessage := func(msg string) string {
		checksum := calculateChecksum([]byte(msg))
		return msg + string(checksum)
	}

	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{
			name:  "valid message with correct checksum",
			input: createValidMessage("/V(V=12.345)"),
			valid: true,
		},
		{
			name:  "invalid checksum byte",
			input: "/V(V=12.345)" + string(byte(0xFF)), // Wrong checksum
			valid: false,
		},
		{
			name:  "missing closing parenthesis",
			input: "/V(V=12.345",
			valid: false,
		},
		{
			name:  "closing paren at end (no checksum)",
			input: "/V(V=12.345)",
			valid: false,
		},
		{
			name:  "empty string",
			input: "",
			valid: false,
		},
		{
			name:  "no parentheses",
			input: "/V12.345X",
			valid: false,
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

// TestParseVEDirectStreamCoverage provides full coverage for ParseVEDirectStream
func TestParseVEDirectStreamCoverage(t *testing.T) {
	createValidMessage := func(msg string) string {
		checksum := calculateChecksum([]byte(msg))
		return msg + string(checksum)
	}

	tests := []struct {
		name      string
		input     string
		wantCount int
		wantErr   bool
	}{
		{
			name:      "empty stream",
			input:     "",
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:      "whitespace only",
			input:     "   \n\n  ",
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:      "single valid message with correct checksum",
			input:     createValidMessage("/V(V=12.345)") + "\n",
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:      "multiple valid messages",
			input:     createValidMessage("/V(V=12.345)") + "\n" + createValidMessage("/A(A=5.67)") + "\n",
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "mixed valid and invalid messages - invalid filtered out",
			input:     createValidMessage("/V(V=12.345)") + "\n" + "/INVALID\n" + createValidMessage("/A(A=5.67)") + "\n",
			wantCount: 2, // Only valid messages counted, invalid ones filtered
			wantErr:   false,
		},
		{
			name:      "carriage return handling",
			input:     createValidMessage("/V(V=12.345)") + "\r\n",
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:      "no trailing newline",
			input:     createValidMessage("/V(V=12.345)"),
			wantCount: 1,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frames, err := ParseVEDirectStream(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseVEDirectStream() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(frames) != tt.wantCount {
				t.Errorf("len(frames) = %d, want %d. Frames: %+v", len(frames), tt.wantCount, frames)
			}
		})
	}
}

// TestFormatValueCoverage provides full coverage for FormatValue
func TestFormatValueCoverage(t *testing.T) {
	tests := []struct {
		name  string
		frame *VEDirectFrame
		check func(string) bool
	}{
		{
			name:  "nil frame returns empty",
			frame: nil,
			check: func(result string) bool {
				return result == ""
			},
		},
		{
			name: "empty DataKey returns empty",
			frame: &VEDirectFrame{
				DataKey: "",
			},
			check: func(result string) bool {
				return result == ""
			},
		},
		{
			name: "known voltage key formats with unit",
			frame: &VEDirectFrame{
				DataKey:  KeySystemVoltage,
				RawValue: "12.345",
				Value:    12.345,
			},
			check: func(result string) bool {
				return len(result) > 0 && containsString(result, "V") // Contains unit
			},
		},
		{
			name: "unknown key uses raw format",
			frame: &VEDirectFrame{
				DataKey:  "UNKNOWN_KEY",
				RawValue: "test_value",
			},
			check: func(result string) bool {
				return len(result) > 0 && containsString(result, "UNKNOWN_KEY")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatValue(tt.frame)
			if !tt.check(result) {
				t.Errorf("FormatValue() = %q, did not pass validation check", result)
			}
		})
	}
}

// TestConvertToValueCoverage provides full coverage for ConvertToValue
func TestConvertToValueCoverage(t *testing.T) {
	tests := []struct {
		name  string
		frame *VEDirectFrame
	}{
		{
			name: "empty frame",
			frame: &VEDirectFrame{},
		},
		{
			name: "frame with all fields populated",
			frame: &VEDirectFrame{
				DataKey:  KeyBatteryVoltage,
				RawValue: "13.5",
				Value:    13.5,
				Valid:    true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.frame.ConvertToValue()

			if result.Key != tt.frame.DataKey {
				t.Errorf("result.Key = %q, want %q", result.Key, tt.frame.DataKey)
			}

			if result.RawValue != tt.frame.RawValue {
				t.Errorf("result.RawValue = %q, want %q", result.RawValue, tt.frame.RawValue)
			}

			if result.FloatValue != tt.frame.Value {
				t.Errorf("result.FloatValue = %v, want %v", result.FloatValue, tt.frame.Value)
			}

			if result.Timestamp.IsZero() {
				t.Error("result.Timestamp should not be zero")
			}
		})
	}
}

// Helper function for string containment check
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestParseDataFrameSpecialValues tests special value handling in parseDataFrame (covers lines 106-115)
func TestParseDataFrameSpecialValues(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValue float64
		wantErr   bool
	}{
		{
			name:      "NAN value converts to 0",
			input:     "/V(NAN)",
			wantValue: 0,
			wantErr:   true, // Will error due to missing checksum
		},
		{
			name:      "nan lowercase converts to 0",
			input:     "/V(nan)",
			wantValue: 0,
			wantErr:   true,
		},
		{
			name:      "dash value converts to 0",
			input:     "/A(-)",
			wantValue: 0,
			wantErr:   true,
		},
		{
			name:      "key=NAN format",
			input:     "/V(V=NAN)",
			wantValue: 0,
			wantErr:   true,
		},
		{
			name:      "key=- format",
			input:     "/A(A=-)",
			wantValue: 0,
			wantErr:   true,
		},
		{
			name:      "invalid non-numeric value defaults to 0",
			input:     "/P(abc)",
			wantValue: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := &VEDirectFrame{}
			result, err := parseDataFrame(tt.input, frame)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseDataFrame() error = %v, wantErr %v", err, tt.wantErr)
			}

			if result.Value != tt.wantValue {
				t.Errorf("result.Value = %v, want %v", result.Value, tt.wantValue)
			}
		})
	}
}

// TestParseDataFrameChecksumValidation tests checksum validation in parseDataFrame (covers lines 123-130)
func TestParseDataFrameChecksumValidation(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValid bool
		wantErr   bool
	}{
		{
			name: "valid checksum passes",
			input: func() string {
				content := "/V(12.345)"  // Message including the closing paren
				checksum := calculateChecksum([]byte(content))
				return content + string(rune(checksum))
			}(),
			wantValid: true,
			wantErr:   false,
		},
		{
			name:      "invalid checksum fails",
			input:     "/V(12.345)" + string(rune(0xFF)),  // Message with ) then wrong checksum byte
			wantValid: false,
			wantErr:   false, // Returns frame with Valid=false, no error
		},
		{
			name:      "missing checksum byte returns error",
			input:     "/V(12.345)",  // Message with ) but NO checksum byte after it
			wantValid: false,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := &VEDirectFrame{}
			result, err := parseDataFrame(tt.input, frame)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseDataFrame() error = %v, wantErr %v", err, tt.wantErr)
			}

			if result.Valid != tt.wantValid {
				t.Errorf("result.Valid = %v, want %v", result.Valid, tt.wantValid)
			}
		})
	}
}

// TestParseAddressFrameEdgeCases tests edge cases in parseAddressFrame (covers lines 144-158)
func TestParseAddressFrameEdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantKey    string
		wantValue  string
		wantResult bool // Whether result should be non-nil
	}{
		{
			name:       "missing opening paren returns frame",
			input:      "@/Dev123",
			wantKey:    "",
			wantValue:  "",
			wantResult: true,
		},
		{
			name:       "missing closing paren returns frame",
			input:      "@/Dev(123",
			wantKey:    "",
			wantValue:  "",
			wantResult: true,
		},
		{
			name:       "empty value in parentheses",
			input:      "@/Dev()",
			wantKey:    "/Dev",
			wantValue:  "",
			wantResult: true,
		},
		{
			name:       "valid address frame with value",
			input:      "@/Dev(SmartSolar MPPT)",
			wantKey:    "/Dev",
			wantValue:  "SmartSolar MPPT",
			wantResult: true,
		},
		{
			name:       "address frame with key=value format",
			input:      "@/Sn(ID=123456789)",
			wantKey:    "ID", // When there's a = in the value, parts[0] becomes the DataKey
			wantValue:  "123456789",
			wantResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := &VEDirectFrame{}
			result, _ := parseAddressFrame(tt.input, frame)

			if tt.wantResult && result == nil {
				t.Error("Expected non-nil result")
			}

			if !tt.wantResult && result != nil {
				t.Error("Expected nil result")
			}

			if result != nil {
				if result.DataKey != tt.wantKey {
					t.Errorf("result.DataKey = %q, want %q", result.DataKey, tt.wantKey)
				}
				if result.RawValue != tt.wantValue {
					t.Errorf("result.RawValue = %q, want %q", result.RawValue, tt.wantValue)
				}
			}
		})
	}
}

// TestCalculateChecksumEdgeCases tests edge cases for checksum calculation
func TestCalculateChecksumEdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		input  []byte
		expect byte
	}{
		{
			name:   "empty data returns 0",
			input:  []byte{},
			expect: 0,
		},
		{
			name:   "single byte",
			input:  []byte{0x41},
			expect: 0x41,
		},
		{
			name:   "two identical bytes XOR to 0",
			input:  []byte{0x41, 0x41},
			expect: 0x00,
		},
		{
			name:   "multiple bytes",
			input:  []byte{0x41, 0x42, 0x43},
			expect: 0x41 ^ 0x42 ^ 0x43,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateChecksum(tt.input)
			if result != tt.expect {
				t.Errorf("calculateChecksum() = 0x%02X, want 0x%02X", result, tt.expect)
			}
		})
	}
}

// TestValidateVEDirectMessageEdgeCases tests edge cases for message validation
func TestValidateVEDirectMessageEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{
			name:  "message too short",
			input: "/V(",
			valid: false,
		},
		{
			name:  "closing paren at end (no checksum byte)",
			input: "/V(12.345)",
			valid: false,
		},
		{
			name: "valid message with correct checksum",
			input: func() string {
				content := "/V(V=12.345)"  // Message including the closing paren
				checksum := calculateChecksum([]byte(content))
				return content + string(rune(checksum))
			}(),
			valid: true,
		},
		{
			name:  "invalid checksum",
			input: "/V(V=12.345)" + string(rune(0x00)),
			valid: false,
		},
		{
			name:  "no closing paren",
			input: "/V12.345",
			valid: false,
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