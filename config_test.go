package main

import (
	"os"
	"regexp"
	"testing"
)

// TestYoloConfig tests the configuration constants and variables.
func TestYoloConfig(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected interface{}
	}{
		{"YoloDir", YoloDir, ".yolo"},
		{"MaxContextMessages", MaxContextMessages, 40},
		{"MaxToolOutput", MaxToolOutput, 0}, // 0 means unlimited
		{"CommandTimeout", CommandTimeout, 30},
		{"ToolTimeout", ToolTimeout, 60},
		{"MaxSubagentRounds", MaxSubagentRounds, 20},
		{"DefaultNumCtx", DefaultNumCtx, 8192},
		{"DefaultInputDelay", DefaultInputDelay, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("%s = %v, expected %v", tt.name, tt.value, tt.expected)
			}
		})
	}
}

// TestGetEnvDefaultWithVariousValues tests the getEnvDefault helper function with various scenarios.
func TestGetEnvDefaultWithVariousValues(t *testing.T) {
	// Clean up any existing test env vars after tests
	defer os.Unsetenv("YOLO_CONFIG_TEST_VAR")
	defer os.Unsetenv("YOLO_CONFIG_EMPTY_VAR")

	tests := []struct {
		name      string
		key       string
		fallback  string
		expected  string
		setupFunc func()
	}{
		{
			name:      "unset variable returns fallback",
			key:       "YOLO_CONFIG_NONEXISTENT_VAR_" + t.Name(),
			fallback:  "default_value",
			expected:  "default_value",
			setupFunc: func() {},
		},
		{
			name:     "set variable returns its value",
			key:      "YOLO_CONFIG_TEST_VAR",
			fallback: "default_value",
			expected: "test_value",
			setupFunc: func() {
				os.Setenv("YOLO_CONFIG_TEST_VAR", "test_value")
			},
		},
		{
			name:     "empty variable returns fallback",
			key:      "YOLO_CONFIG_EMPTY_VAR",
			fallback: "default_value",
			expected: "default_value",
			setupFunc: func() {
				os.Setenv("YOLO_CONFIG_EMPTY_VAR", "")
			},
		},
		{
			name:     "variable with special characters",
			key:      "YOLO_CONFIG_TEST_SPECIAL",
			fallback: "default",
			expected: "http://localhost:11434/api/v1",
			setupFunc: func() {
				os.Setenv("YOLO_CONFIG_TEST_SPECIAL", "http://localhost:11434/api/v1")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFunc()
			result := getEnvDefault(tt.key, tt.fallback)
			if result != tt.expected {
				t.Errorf("getEnvDefault(%q, %q) = %q, expected %q", tt.key, tt.fallback, result, tt.expected)
			}
		})
	}
}

// TestHistoryFile tests the HistoryFile path.
func TestHistoryFile(t *testing.T) {
	expected := ".yolo/history.json"
	if HistoryFile != expected {
		t.Errorf("HistoryFile = %q, expected %q", HistoryFile, expected)
	}

	// Verify it's a valid relative path
	if !filepathIsRelative(HistoryFile) {
		t.Error("HistoryFile should be a relative path")
	}
}

// TestSubagentDir tests the SubagentDir path.
func TestSubagentDir(t *testing.T) {
	expected := ".yolo/subagents"
	if SubagentDir != expected {
		t.Errorf("SubagentDir = %q, expected %q", SubagentDir, expected)
	}

	// Verify it's a valid relative path
	if !filepathIsRelative(SubagentDir) {
		t.Error("SubagentDir should be a relative path")
	}
}

// TestFileNameRegex tests the fileNameRegex for parsing sub-agent result files.
func TestFileNameRegex(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		shouldMatch bool
		expectedID  string
	}{
		{"simple agent ID", "agent_1.json", true, "1"},
		{"numeric ID", "agent_123.json", true, "123"},
		{"test agent ID", "agent_test_456.json", true, "test_456"},
		{"complex ID with hyphen", "agent_bug-fix-789.json", true, "bug-fix-789"},
		{"ID with underscore prefix", "agent_test_sub_agent_100.json", true, "test_sub_agent_100"},
		{"wrong extension", "agent_1.txt", false, ""},
		{"missing agent prefix", "run_1.json", false, ""},
		{"no json suffix", "agent_1", false, ""},
		{"empty filename", "", false, ""},
		{"just json", "test.json", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := fileNameRegex.FindStringSubmatch(tt.filename)
			matched := len(matches) > 1

			if matched != tt.shouldMatch {
				t.Errorf("fileNameRegex.Match(%q) = %v, expected %v", tt.filename, matched, tt.shouldMatch)
			}

			if tt.shouldMatch && matched {
				capturedID := matches[1]
				if capturedID != tt.expectedID {
					t.Errorf("fileNameRegex captured ID = %q, expected %q", capturedID, tt.expectedID)
				}
			}
		})
	}
}

// TestOllamaURL tests the OllamaURL configuration.
func TestOllamaURL(t *testing.T) {
	// Default value should be set
	if OllamaURL == "" {
		t.Error("OllamaURL should have a default value")
	}

	// Should start with http:// or https://
	if !regexp.MustCompile(`^https?://`).MatchString(OllamaURL) {
		t.Errorf("OllamaURL = %q, expected to start with http:// or https://", OllamaURL)
	}
}

// TestANSIColors tests that ANSI color constants are properly defined.
func TestANSIColors(t *testing.T) {
	colors := map[string]string{
		"Reset":   Reset,
		"Bold":    Bold,
		"Dim":     Dim,
		"Red":     Red,
		"Green":   Green,
		"Yellow":  Yellow,
		"Blue":    Blue,
		"Magenta": Magenta,
		"Cyan":    Cyan,
		"Gray":    Gray,
	}

	for name, color := range colors {
		if color == "" {
			t.Errorf("ANSI color %s is empty", name)
		}
		// All ANSI codes start with \033 (ESC)
		if len(color) < 2 || color[:1] != "\033" {
			t.Errorf("ANSI color %s = %q, expected to start with \\033", name, color)
		}
	}
}

// TestTimeoutValues tests that timeout values are reasonable.
func TestTimeoutValues(t *testing.T) {
	tests := []struct {
		name     string
		timeout  int
		minValue int
		maxValue int
	}{
		{"CommandTimeout", CommandTimeout, 5, 120},
		{"ToolTimeout", ToolTimeout, 10, 300},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.timeout < tt.minValue {
				t.Errorf("%s = %ds is too low (min: %ds)", tt.name, tt.timeout, tt.minValue)
			}
			if tt.timeout > tt.maxValue {
				t.Errorf("%s = %ds is too high (max: %ds)", tt.name, tt.timeout, tt.maxValue)
			}
		})
	}
}

// TestConfigConstants tests that important configuration constants are set.
func TestConfigConstants(t *testing.T) {
	tests := []struct {
		name         string
		value        int
		minValid     int
		maxValid     int
		cannotBeZero bool
	}{
		{"MaxContextMessages", MaxContextMessages, 10, 200, true},
		{"MaxSubagentRounds", MaxSubagentRounds, 5, 100, true},
		{"DefaultNumCtx", DefaultNumCtx, 4096, 32768, true},
		{"DefaultInputDelay", DefaultInputDelay, 1, 30, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cannotBeZero && tt.value == 0 {
				t.Errorf("%s should not be zero", tt.name)
			}
			if tt.value < tt.minValid || tt.value > tt.maxValid {
				t.Logf("Warning: %s = %d is outside typical range [%d, %d]", tt.name, tt.value, tt.minValid, tt.maxValid)
			}
		})
	}
}

// TestSourceCodeLocation tests the _SourceCodeLocation constant.
func TestSourceCodeLocation(t *testing.T) {
	if _SourceCodeLocation != "." {
		t.Errorf("_SourceCodeLocation = %q, expected \".\"", _SourceCodeLocation)
	}
}

// TestUseRestartTool tests the _UseRestartTool constant.
func TestUseRestartTool(t *testing.T) {
	if !_UseRestartTool {
		t.Error("_UseRestartTool should be true for proper operation")
	}
}

// Helper function to check if a path is relative
func filepathIsRelative(path string) bool {
	return !isPathAbsolute(path)
}

// Helper function to check if a path is absolute
func isPathAbsolute(path string) bool {
	if len(path) == 0 {
		return false
	}

	// Unix-style absolute paths
	if path[0] == '/' {
		return true
	}

	// Windows-style absolute paths (e.g., C:\ or C:/)
	if len(path) >= 2 && path[1] == ':' {
		return true
	}

	return false
}
