package playwright

import (
	"testing"
)

// TestPlaywright_SkipExternalTests skips tests that require external services
func TestPlaywright_SkipExternalTests(t *testing.T) {
	t.Skip("Skipping Playwright browser tests - requires browser driver installation and external network access")
}

// TestNewPlaywrightMCP_Constructor tests the constructor doesn't panic
func TestNewPlaywrightMCP_Constructor(t *testing.T) {
	// This test verifies the struct can be created (but won't initialize playwright runtime)
	t.Run("Struct initialization", func(t *testing.T) {
		p := &PlaywrightMCP{
			headless:      true,
			timeout:       30000, // milliseconds
			screenShotDir: "/tmp/test-screenshots",
		}

		if p == nil {
			t.Fatal("Expected non-nil PlaywrightMCP")
		}

		if !p.headless {
			t.Error("Expected headless to be true")
		}

		if p.timeout != 30000 {
			t.Errorf("Expected timeout to be 30000, got %d", p.timeout)
		}

		if p.screenShotDir != "/tmp/test-screenshots" {
			t.Errorf("Expected screenshot dir to be /tmp/test-screenshots, got %s", p.screenShotDir)
		}
	})
}

// TestBrowserAction_Constructor tests BrowserAction struct
func TestBrowserAction_Constructor(t *testing.T) {
	ba := BrowserAction{
		Timeout:    30000, // milliseconds
		ScreenShot: true,
		OutputDir:  "/tmp/output",
	}

	if ba.Timeout != 30000 {
		t.Errorf("Expected timeout to be 30000, got %d", ba.Timeout)
	}

	if !ba.ScreenShot {
		t.Error("Expected ScreenShot to be true")
	}

	if ba.OutputDir != "/tmp/output" {
		t.Errorf("Expected OutputDir to be /tmp/output, got %s", ba.OutputDir)
	}
}

// TestPlaywrightResult_Constructor tests PlaywrightResult struct
func TestPlaywrightResult_Constructor(t *testing.T) {
	result := &PlaywrightResult{
		Title:      "Test Page",
		URL:        "https://example.com",
		Text:       "<html>...</html>",
		HTML:       "<body>...</body>",
		Screenshot: "/tmp/screenshot.png",
		Error:      "",
		Caption:    "Test caption",
	}

	if result.Title != "Test Page" {
		t.Errorf("Expected Title to be 'Test Page', got %s", result.Title)
	}

	if result.URL != "https://example.com" {
		t.Errorf("Expected URL to be 'https://example.com', got %s", result.URL)
	}

	if result.Error != "" {
		t.Errorf("Expected Error to be empty, got %s", result.Error)
	}
}

// TestPlaywrightResult_WithError tests PlaywrightResult with error
func TestPlaywrightResult_WithError(t *testing.T) {
	result := &PlaywrightResult{
		Error: "Failed to navigate",
	}

	if result.Error != "Failed to navigate" {
		t.Errorf("Expected Error to be 'Failed to navigate', got %s", result.Error)
	}
}

// TestBoolPtr tests the boolPtr helper function
func TestBoolPtr(t *testing.T) {
	tests := []struct {
		name string
		val  bool
		want *bool
	}{
		{"true value", true, func() *bool { b := true; return &b }()},
		{"false value", false, func() *bool { b := false; return &b }()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := boolPtr(tt.val)
			if got == nil {
				t.Fatal("Expected non-nil pointer")
			}
			if *got != tt.val {
				t.Errorf("Expected %v, got %v", tt.val, *got)
			}
		})
	}
}
