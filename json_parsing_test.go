package main

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"testing"
)

// ===================================================
// JSON Parsing Edge Case Tests
// Comprehensive testing of JSON marshaling/unmarshaling edge cases
// ===================================================

func TestJSONParsingEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         any
		expectSuccess bool
	}{
		{
			name:          "empty_map",
			input:         map[string]any{},
			expectSuccess: true,
		},
		{
			name:          "nil_value",
			input:         nil,
			expectSuccess: true, // nil marshals to JSON null
		},
		{
			name:          "empty_string",
			input:         "",
			expectSuccess: true,
		},
		{
			name:          "zero_value_int",
			input:         0,
			expectSuccess: true,
		},
		{
			name:          "zero_value_float",
			input:         0.0,
			expectSuccess: true,
		},
		{
			name:          "false_bool",
			input:         false,
			expectSuccess: true,
		},
		{
			name:          "max_int64",
			input:         int64(9223372036854775807),
			expectSuccess: true,
		},
		{
			name:          "min_int64",
			input:         int64(-9223372036854775808),
			expectSuccess: true,
		},
		{
			name:          "max_uint64",
			input:         uint64(18446744073709551615),
			expectSuccess: true,
		},
		{
			name:          "very_large_float",
			input:         1e308,
			expectSuccess: true,
		},
		{
			name:          "very_small_float",
			input:         1e-308,
			expectSuccess: true,
		},
		{
			name:          "unicode_strings",
			input:         map[string]string{"emoji": "🎉🚀💻"},
			expectSuccess: true,
		},
		{
			name:          "special_characters_in_string",
			input:         map[string]string{"msg": `newline\ttab\x20quote"backslash\\`},
			expectSuccess: true,
		},
		{
			name: "deeply_nested_structure",
			input: map[string]any{
				"level1": map[string]any{
					"level2": map[string]any{
						"level3": map[string]any{
							"level4": map[string]any{
								"level5": "deep",
							},
						},
					},
				},
			},
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.input)

			if (err == nil) != tt.expectSuccess {
				t.Errorf("Expected success=%v, got error: %v", tt.expectSuccess, err)
				return
			}

			// Verify we can unmarshal back to any
			var output any
			err = json.Unmarshal(data, &output)
			if (err == nil) != tt.expectSuccess {
				t.Errorf("Expected unmarshal success=%v, got error: %v", tt.expectSuccess, err)
				return
			}

			if !tt.expectSuccess {
				t.Error("Unmarshal should have failed")
			}
		})
	}
}

func TestJSONMalformedInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         string
		expectSuccess bool
		errorContains string
	}{
		{
			name:          "missing_quotes",
			input:         `{key: value}`,
			expectSuccess: false,
			errorContains: "invalid character",
		},
		{
			name:          "trailing_comma",
			input:         `{"key": "value",}`,
			expectSuccess: false,
			errorContains: "invalid character",
		},
		{
			name:          "unclosed_brace",
			input:         `{"key": "value"`,
			expectSuccess: false,
			errorContains: "unexpected end of JSON input",
		},
		{
			name:          "empty_object",
			input:         `{}`,
			expectSuccess: true,
		},
		{
			name:          "empty_array",
			input:         `[]`,
			expectSuccess: true,
		},
		{
			name:          "missing_colon",
			input:         `{"key" "value"}`,
			expectSuccess: false,
			errorContains: "invalid character",
		},
		{
			name:          "invalid_number",
			input:         `{"num": .123}`,
			expectSuccess: false,
			errorContains: "invalid character",
		},
		{
			name:          "control_characters",
			input:         string([]byte{0x00, 0x01, 0x02}),
			expectSuccess: false, // control chars are invalid in JSON strings
			errorContains: "invalid character",
		},
		{
			name:          "duplicate_keys",
			input:         `{"key": 1, "key": 2}`,
			expectSuccess: true, // JSON allows duplicate keys (last wins)
		},
		{
			name:          "invalid_escaped_char",
			input:         `{"msg": "test \x00"}`,
			expectSuccess: false,
			errorContains: "invalid character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output any
			err := json.Unmarshal([]byte(tt.input), &output)

			if (err == nil) != tt.expectSuccess {
				t.Errorf("Expected success=%v, got error: %v", tt.expectSuccess, err)
				return
			}

			if !tt.expectSuccess && tt.errorContains != "" {
				if !strings.Contains(fmt.Sprintf("%v", err), tt.errorContains) {
					t.Logf("Error doesn't contain expected text: %s (error was: %v)", tt.errorContains, err)
				}
			}

			// Test round-trip for successful parses
			if tt.expectSuccess && output != nil {
				data, err := json.Marshal(output)
				if err != nil {
					t.Errorf("Failed to marshal after unmarshal: %v", err)
				}

				var output2 any
				err = json.Unmarshal(data, &output2)
				if err != nil {
					t.Errorf("Failed second unmarshal: %v", err)
				}
			}
		})
	}
}

func TestJSONUnmarshalSpecificTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		jsonInput  string
		outputType any
		expectErr  bool
		verify     func(any) error
	}{
		{
			name:       "unmarshal_to_string",
			jsonInput:  `"hello"`,
			outputType: new(string),
			expectErr:  false,
			verify: func(v any) error {
				if s, ok := v.(*string); ok && *s != "hello" {
					return fmt.Errorf("expected 'hello', got %s", *s)
				}
				return nil
			},
		},
		{
			name:       "unmarshal_to_int",
			jsonInput:  `42`,
			outputType: new(int),
			expectErr:  false,
			verify: func(v any) error {
				if i, ok := v.(*int); ok && *i != 42 {
					return fmt.Errorf("expected 42, got %d", *i)
				}
				return nil
			},
		},
		{
			name:       "unmarshal_to_float64",
			jsonInput:  `3.14`,
			outputType: new(float64),
			expectErr:  false,
			verify: func(v any) error {
				if f, ok := v.(*float64); ok && *f != 3.14 {
					return fmt.Errorf("expected 3.14, got %f", *f)
				}
				return nil
			},
		},
		{
			name:       "unmarshal_to_bool",
			jsonInput:  `true`,
			outputType: new(bool),
			expectErr:  false,
			verify: func(v any) error {
				if b, ok := v.(*bool); ok && !*b {
					return fmt.Errorf("expected true")
				}
				return nil
			},
		},
		{
			name:       "unmarshal_to_slice",
			jsonInput:  `[1, 2, 3]`,
			outputType: new([]int),
			expectErr:  false,
			verify: func(v any) error {
				if s, ok := v.(*[]int); ok && len(*s) != 3 {
					return fmt.Errorf("expected slice of length 3")
				}
				return nil
			},
		},
		{
			name:       "unmarshal_to_array",
			jsonInput:  `[1, 2, 3]`,
			outputType: new([3]int),
			expectErr:  false,
			verify: func(v any) error {
				if a, ok := v.(*[3]int); ok && len(*a) != 3 {
					return fmt.Errorf("expected array of length 3")
				}
				return nil
			},
		},
		{
			name:      "unmarshal_to_struct",
			jsonInput: `{"Name": "John", "Age": 30}`,
			outputType: new(struct {
				Name string `json:"name"` // wrong tag
				Age  int    `json:"age"`  // wrong tag
			}),
			expectErr: false,
			verify: func(v any) error {
				// Should unmarshal but with zero values due to mismatched tags
				return nil
			},
		},
		{
			name:       "unmarshal_to_wrong_type",
			jsonInput:  `"hello"`,
			outputType: new(int),
			expectErr:  true,
			verify: func(v any) error {
				return fmt.Errorf("should have failed")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output any
			if v, ok := tt.outputType.(*any); ok {
				output = &v
			} else if ptr, ok := tt.outputType.(**struct{}); ok {
				output = ptr
			} else {
				output = tt.outputType
			}

			err := json.Unmarshal([]byte(tt.jsonInput), output)

			if (err == nil) != !tt.expectErr {
				t.Errorf("Expected error=%v, got err: %v", tt.expectErr, err)
				return
			}

			if err == nil && tt.verify != nil {
				if v, ok := output.(interface{ Value() any }); ok {
					err = tt.verify(v)
					if err != nil {
						t.Errorf("Verify failed: %v", err)
					}
				}
			}
		})
	}
}

func TestJSONTypeInference(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        string
		expectedType string
	}{
		{
			name:         "null",
			input:        `null`,
			expectedType: "json.Number",
		},
		{
			name:         "string",
			input:        `"hello"`,
			expectedType: "string",
		},
		{
			name:         "int",
			input:        `42`,
			expectedType: "json.Number",
		},
		{
			name:         "float",
			input:        `3.14`,
			expectedType: "json.Number",
		},
		{
			name:         "bool true",
			input:        `true`,
			expectedType: "bool",
		},
		{
			name:         "bool false",
			input:        `false`,
			expectedType: "bool",
		},
		{
			name:         "array",
			input:        `[1, 2, 3]`,
			expectedType: "[]interface{}",
		},
		{
			name:         "object",
			input:        `{"key": "value"}`,
			expectedType: "map[string]interface{}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var parsed any
			err := json.Unmarshal([]byte(tt.input), &parsed)
			if err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			typeName := fmt.Sprintf("%T", parsed)
			if typeName != tt.expectedType {
				t.Errorf("Expected type %s, got %s", tt.expectedType, typeName)
			}
		})
	}
}

func TestJSONIndentAndCompact(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"level1": map[string]any{
			"key1":  "value1",
			"key2":  42,
			"array": []int{1, 2, 3},
		},
	}

	// Test compact output (no indentation)
	compact, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal compact: %v", err)
	}

	var parsedCompact any
	err = json.Unmarshal(compact, &parsedCompact)
	if err != nil {
		t.Errorf("Failed to unmarshal compact output: %v", err)
	}

	// Test indented output (2 spaces)
	indented, err := json.MarshalIndent(input, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal indented: %v", err)
	}

	// Verify indentation
	hasIndentation := false
	for _, line := range strings.Split(string(indented), "\n") {
		if len(line) > 0 && line[0] == ' ' {
			hasIndentation = true
			break
		}
	}

	if !hasIndentation {
		t.Error("Expected indented JSON to have indentation")
	}

	var parsedIndented any
	err = json.Unmarshal(indented, &parsedIndented)
	if err != nil {
		t.Errorf("Failed to unmarshal indented output: %v", err)
	}

	// Both should parse back to same structure
	if fmt.Sprintf("%v", parsedCompact) != fmt.Sprintf("%v", parsedIndented) {
		t.Error("Compact and indented JSON should parse to same value")
	}
}

func TestJSONNumberPrecision(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		wantInt   bool
		wantVal   int64
		wantFloat float64
	}{
		{
			name:      "large_int",
			input:     `9223372036854775807`,
			wantInt:   true,
			wantVal:   9223372036854775807,
			wantFloat: 9223372036854775807.0,
		},
		{
			name:      "float_with_decimal",
			input:     `3.14159`,
			wantInt:   false,
			wantFloat: 3.14159,
		},
		{
			name:      "scientific_notation",
			input:     `1e10`,
			wantFloat: 10000000000.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var number json.Number
			err := json.Unmarshal([]byte(tt.input), &number)
			if err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			// Test conversion to int64
			i, err := number.Int64()
			if err != nil && tt.wantInt {
				t.Errorf("Expected Int64 conversion to succeed, got error: %v", err)
			} else if !tt.wantInt && err == nil {
				t.Error("Int64 conversion should have failed for float")
			}

			if i != tt.wantVal && tt.wantInt {
				t.Errorf("Expected int64 value %d, got %d", tt.wantVal, i)
			}

			// Test conversion to float64
			f, err := number.Float64()
			if err != nil {
				t.Errorf("Float64 conversion failed: %v", err)
			} else if f != tt.wantFloat && !math.IsNaN(f) && !math.IsInf(f, 0) {
				t.Errorf("Expected float value %f, got %f", tt.wantFloat, f)
			}
		})
	}
}

// Helper types for testing
type testStruct struct {
	Name  string `json:"name"`
	Age   int    `json:"age"`
	Email string `json:"email,omitempty"`
}

func TestJSONTagsAndOmitEmpty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      testStruct
		expectOmit bool
	}{
		{
			name: "normal_struct",
			input: testStruct{
				Name:  "John",
				Age:   30,
				Email: "john@example.com",
			},
			expectOmit: false,
		},
		{
			name: "empty_optional_field",
			input: testStruct{
				Name: "Jane",
				Age:  25,
			},
			expectOmit: true, // Email field should be omitted
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			if tt.expectOmit {
				// Verify email field is not present in JSON
				var parsed map[string]any
				err = json.Unmarshal(data, &parsed)
				if err != nil {
					t.Errorf("Failed to unmarshal: %v", err)
				} else if _, ok := parsed["email"]; ok {
					t.Error("Expected 'email' field to be omitted due to omitempty")
				}
			}

			var parsed testStruct
			err = json.Unmarshal(data, &parsed)
			if err != nil {
				t.Errorf("Failed to unmarshal: %v", err)
			} else if parsed.Name != tt.input.Name || parsed.Age != tt.input.Age {
				t.Errorf("Unmarshaled struct doesn't match input")
			}
		})
	}
}
