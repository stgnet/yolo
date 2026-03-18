package playwright

import (
	"testing"
	"time"
)

// TestNewPlaywrightMCPSkip skips the test that requires external browser driver
func TestNewPlaywrightMCPSkip(t *testing.T) {
	t.Skip("Skipping PlaywrightMCP constructor test - requires browser drivers to be installed")
}

// TestBoolPtr tests the helper function for bool pointers
func TestBoolPtr(t *testing.T) {
	tests := []struct {
		name     string
		input    bool
		expected *bool
	}{
		{"true value", true, func() *bool { v := true; return &v }()},
		{"false value", false, func() *bool { v := false; return &v }()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := boolPtr(tt.input)
			if result == nil {
				t.Fatal("Expected non-nil pointer")
			}
			if *result != tt.input {
				t.Errorf("Expected %v, got %v", tt.input, *result)
			}
		})
	}
}

// TestPlaywrightResultError tests error result creation
func TestPlaywrightResultError(t *testing.T) {
	err := "test error"
	result := &PlaywrightResult{
		Error: err,
	}

	if result.Error != err {
		t.Errorf("Expected error %q, got %q", err, result.Error)
	}
	if result.Title != "" {
		t.Error("Expected empty title")
	}
	if result.URL != "" {
		t.Error("Expected empty URL")
	}
}

// TestPlaywrightResultSuccess tests successful result creation
func TestPlaywrightResultSuccess(t *testing.T) {
	result := &PlaywrightResult{
		Title:  "Test Page",
		URL:    "https://example.com",
		Text:   "Sample text",
		Caption: "A test caption",
	}

	if result.Title != "Test Page" {
		t.Errorf("Expected title 'Test Page', got %q", result.Title)
	}
	if result.URL != "https://example.com" {
		t.Errorf("Expected URL 'https://example.com', got %q", result.URL)
	}
}

// TestBrowserActionDefaults tests BrowserAction with default values
func TestBrowserActionDefaults(t *testing.T) {
	action := BrowserAction{}

	if action.Timeout != 0 {
		t.Errorf("Expected default timeout to be 0, got %v", action.Timeout)
	}
	if action.ScreenShot != false {
		t.Error("Expected default ScreenShot to be false")
	}
	if action.OutputDir != "" {
		t.Error("Expected default OutputDir to be empty")
	}
}

// TestBrowserActionWithOptions tests BrowserAction with custom options
func TestBrowserActionWithOptions(t *testing.T) {
	timeout := 10 * time.Second
	outputDir := "/tmp/test"
	action := BrowserAction{
		Timeout:   timeout,
		ScreenShot: true,
		OutputDir: outputDir,
	}

	if action.Timeout != timeout {
		t.Errorf("Expected timeout %v, got %v", timeout, action.Timeout)
	}
	if !action.ScreenShot {
		t.Error("Expected ScreenShot to be true")
	}
	if action.OutputDir != outputDir {
		t.Errorf("Expected OutputDir %q, got %q", outputDir, action.OutputDir)
	}
}

// TestPlaywrightMCPFields tests PlaywrightMCP struct field initialization
func TestPlaywrightMCPFields(t *testing.T) {
	// We can't fully initialize the struct without a real playwright instance,
	// but we can verify the fields exist and have expected types
	mcp := &PlaywrightMCP{
		headless:      true,
		timeout:       30 * time.Second,
		screenShotDir: "/tmp/screenshots",
	}

	if !mcp.headless {
		t.Error("Expected headless to be true")
	}
	if mcp.timeout != 30*time.Second {
		t.Errorf("Expected timeout %v, got %v", 30*time.Second, mcp.timeout)
	}
	if mcp.screenShotDir != "/tmp/screenshots" {
		t.Errorf("Expected screenShotDir %q, got %q", "/tmp/screenshots", mcp.screenShotDir)
	}
}

// TestPlaywrightMCPZeroValues tests PlaywrightMCP with zero values
func TestPlaywrightMCPZeroValues(t *testing.T) {
	mcp := &PlaywrightMCP{}

	if mcp.headless != false {
		t.Error("Expected default headless to be false")
	}
	if mcp.timeout != 0 {
		t.Errorf("Expected default timeout to be 0, got %v", mcp.timeout)
	}
	if mcp.screenShotDir != "" {
		t.Errorf("Expected default screenShotDir to be empty, got %q", mcp.screenShotDir)
	}
}

// TestPlaywrightResultElements tests the elements field in result
func TestPlaywrightResultElements(t *testing.T) {
	elements := []map[string]string{
		{"tag": "div", "class": "container"},
		{"tag": "span", "id": "title"},
	}

	result := &PlaywrightResult{
		Elements: elements,
	}

	if len(result.Elements) != 2 {
		t.Errorf("Expected 2 elements, got %d", len(result.Elements))
	}
	if result.Elements[0]["tag"] != "div" {
		t.Errorf("Expected first element tag 'div', got %q", result.Elements[0]["tag"])
	}
}

// TestPlaywrightResultWithScreenshot tests result with screenshot path
func TestPlaywrightResultWithScreenshot(t *testing.T) {
	screenshotPath := "/tmp/test.png"
	result := &PlaywrightResult{
		Screenshot: screenshotPath,
		Title:      "Test Page",
	}

	if result.Screenshot != screenshotPath {
		t.Errorf("Expected screenshot path %q, got %q", screenshotPath, result.Screenshot)
	}
}
