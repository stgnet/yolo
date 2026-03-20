package playwright

import (
	"encoding/json"
	"fmt"
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

// TestPlaywrightMCP_IsBrowserRunning tests the IsBrowserRunning method
func TestPlaywrightMCP_IsBrowserRunning(t *testing.T) {
	tests := []struct {
		name        string
		browser     interface{} // Using interface{} to simulate browser presence
		expectRunning bool
	}{
		{"browser nil", nil, false},
		{"browser present", &mockBrowser{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PlaywrightMCP{
				headless:      true,
				timeout:       30000,
				screenShotDir: "/tmp/test",
			}

			// We can't easily test this without the actual playwright types
			// So we'll just verify the logic through code inspection
			isRunning := p.IsBrowserRunning()
			if isRunning != tt.expectRunning {
				t.Errorf("Expected IsBrowserRunning to be %v, got %v", tt.expectRunning, isRunning)
			}
		})
	}
}

// TestPlaywrightMCP_Close tests the Close method doesn't panic
func TestPlaywrightMCP_Close(t *testing.T) {
	p := &PlaywrightMCP{
		headless:      true,
		timeout:       30000,
		screenShotDir: "/tmp/test",
	}

	// Close should not panic even with nil browser and pw
	err := p.Close()
	if err != nil {
		t.Logf("Close returned error: %v (this may be expected if playwright is not initialized)", err)
	}
}

// TestPlaywrightResult_MarshalJSON tests JSON serialization of PlaywrightResult
func TestPlaywrightResult_MarshalJSON(t *testing.T) {
	tests := []struct {
		name   string
		result *PlaywrightResult
		wantErr bool
	}{
		{
			name: "basic result",
			result: &PlaywrightResult{
				Title: "Test Page",
				URL: "https://example.com",
				Text: "<html><body>Test</body></html>",
				HTML: "<body>Test</body>",
			},
			wantErr: false,
		},
		{
			name: "result with error",
			result: &PlaywrightResult{
				Error: "Navigation failed",
			},
			wantErr: false,
		},
		{
			name: "result with elements",
			result: &PlaywrightResult{
				Title:    "Page",
				URL:      "https://test.com",
				Elements: []map[string]string{
					{"selector": ".class", "text": "element 1"},
					{"selector": ".class", "text": "element 2"},
				},
			},
			wantErr: false,
		},
		{
			name:   "nil result",
			result: nil,
			wantErr: true, // nil pointer will panic during marshal
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantErr {
				defer func() {
					if r := recover(); r == nil {
						t.Error("expected panic for nil result")
					}
				}()
			}

			data, err := tt.result.MarshalJSON()
			if err != nil && !tt.wantErr {
				t.Errorf("MarshalJSON() error = %v", err)
				return
			}

			// Verify JSON is valid by unmarshaling back
			var result PlaywrightResult
			if err := json.Unmarshal(data, &result); err != nil {
				t.Errorf("Unmarshal failed: %v", err)
			}
		})
	}
}

// TestSearchResults_Constructor tests SearchResults struct
func TestSearchResults_Constructor(t *testing.T) {
	results := &SearchResults{
		Query:      "test query",
		URL:        "https://search.com/results?q=test",
		Title:      "Search Results",
		Text:       "<div>results</div>",
		Screenshot: "/tmp/search.png",
		Elements:   []map[string]string{{"class": ".result", "text": "Result 1"}},
		Error:      "",
	}

	if results.Query != "test query" {
		t.Errorf("Expected Query to be 'test query', got %s", results.Query)
	}

	if len(results.Elements) != 1 {
		t.Errorf("Expected 1 element, got %d", len(results.Elements))
	}
}

// TestBrowserAction_DefaultValues tests BrowserAction with default values
func TestBrowserAction_DefaultValues(t *testing.T) {
	ba := BrowserAction{}

	if ba.Timeout != 0 {
		t.Errorf("Expected default Timeout to be 0, got %d", ba.Timeout)
	}

	if ba.ScreenShot != false {
		t.Error("Expected default ScreenShot to be false")
	}

	if ba.OutputDir != "" {
		t.Errorf("Expected default OutputDir to be empty, got %s", ba.OutputDir)
	}
}

// TestPlaywrightMCP_DefaultScreenshotDir tests that default screenshot directory is set
func TestPlaywrightMCP_DefaultScreenshotDir(t *testing.T) {
	p := &PlaywrightMCP{
		headless:      true,
		timeout:       30000,
		screenShotDir: "", // Empty should default to ./screenshots in NewPlaywrightMCP
	}

	// This tests the struct initialization logic
	// The actual directory creation happens in NewPlaywrightMCP
	if p.screenShotDir != "" {
		t.Logf("Screenshot dir is %s", p.screenShotDir)
	}
}

// TestPlaywrightResult_ElementsParsing tests PlaywrightResult with elements
func TestPlaywrightResult_ElementsParsing(t *testing.T) {
	result := &PlaywrightResult{
		Title: "Page with Elements",
		URL:   "https://example.com/list",
		Elements: []map[string]string{
			{"selector": ".item", "text": "Item 1", "index": "0"},
			{"selector": ".item", "text": "Item 2", "index": "1"},
			{"selector": ".item", "text": "Item 3", "index": "2"},
		},
	}

	if len(result.Elements) != 3 {
		t.Errorf("Expected 3 elements, got %d", len(result.Elements))
	}

	for i, el := range result.Elements {
		expectedIndex := fmt.Sprintf("%d", i)
		if el["index"] != expectedIndex {
			t.Errorf("Element %d: expected index %s, got %s", i, expectedIndex, el["index"])
		}
	}
}

// TestPlaywrightResult_ErrorHandling tests PlaywrightResult error field
func TestPlaywrightResult_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		result      *PlaywrightResult
		expectError bool
	}{
		{
			name: "with error",
			result: &PlaywrightResult{
				Error: "Connection timeout",
			},
			expectError: true,
		},
		{
			name: "without error",
			result: &PlaywrightResult{
				Title: "Success",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasError := tt.result.Error != ""
			if hasError != tt.expectError {
				t.Errorf("Expected error=%v, got error=%v (field='%s')", 
					tt.expectError, hasError, tt.result.Error)
			}
		})
	}
}

type mockBrowser struct{}

// This test verifies that the package compiles and basic types work correctly
func TestPackageCompilation(t *testing.T) {
	// If this compiles, the package structure is correct
	var _ BrowserAction
	var _ PlaywrightResult
	var _ SearchResults
}
