package playwright_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
)

func TestPlaywrightBasic(t *testing.T) {
	// Start a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		switch r.URL.Path {
		case "/":
			w.Write([]byte("<html><body><h1>Test Page</h1><button id='btn'>Click Me</button></body></html>"))
		case "/form":
			w.Write([]byte("<html><body><form><input name='username'><button type='submit'>Submit</button></form></body></html>"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	// Create Playwright instance
	mcp, err := NewPlaywrightMCP(true, 30*time.Second, t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create playwright MCP: %v", err)
	}
	defer mcp.Close()

	tests := []struct {
		name string
		fn   func(*testing.T, *PlaywrightMCP) error
	}{
		{"NavigateTo", testNavigateTo},
		{"ClickElement", testClickElement},
		{"GetElements", testGetElements},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fn(t, mcp); err != nil {
				t.Errorf("Test failed: %v", err)
			}
		})
	}
}

func testNavigateTo(t *testing.T, mcp *PlaywrightMCP) error {
	ctx := context.Background()
	action := BrowserAction{
		Timeout:    5 * time.Second,
		ScreenShot: true,
		OutputDir:  t.TempDir(),
	}

	result, err := mcp.NavigateTo(ctx, "http://example.com", action)
	if err != nil {
		return err
	}

	if result.URL == "" {
		return fmt.Errorf("expected URL in result")
	}

	if result.Title != "" && len(result.Text) == 0 {
		return fmt.Errorf("expected text content")
	}

	return nil
}

func testClickElement(t *testing.T, mcp *PlaywrightMCP) error {
	ctx := context.Background()
	action := BrowserAction{
		Timeout:    5 * time.Second,
		ScreenShot: false,
		OutputDir:  t.TempDir(),
	}

	result, err := mcp.NavigateTo(ctx, "http://example.com", action)
	if err != nil {
		return err
	}

	if result.Title == "" {
		return fmt.Errorf("expected page title")
	}

	clickResult, err := mcp.ClickElement(ctx, "#btn", action)
	if err != nil {
		return fmt.Errorf("click element failed: %v", err)
	}

	if clickResult.Title == "" {
		return fmt.Errorf("expected title after click")
	}

	return nil
}

func testGetElements(t *testing.T, mcp *PlaywrightMCP) error {
	ctx := context.Background()
	action := BrowserAction{
		Timeout:    5 * time.Second,
		ScreenShot: false,
		OutputDir:  t.TempDir(),
	}

	result, err := mcp.NavigateTo(ctx, "http://example.com", action)
	if err != nil {
		return err
	}

	// Test getting all h1 elements
	elements, err := mcp.GetElements(ctx, "http://example.com", "h1", action)
	if err != nil {
		return fmt.Errorf("get elements failed: %v", err)
	}

	if len(elements) == 0 {
		return fmt.Errorf("expected at least one element")
	}

	return nil
}

func TestScreenshotDirCreation(t *testing.T) {
	tempDir := t.TempDir()
	screenshotPath := filepath.Join(tempDir, "test_screenshots")

	mcp, err := NewPlaywrightMCP(true, 30*time.Second, screenshotPath)
	if err != nil {
		t.Fatalf("Failed to create playwright MCP: %v", err)
	}
	defer mcp.Close()

	// Verify directory was created
	_, err = os.Stat(screenshotPath)
	if err != nil {
		t.Errorf("Screenshot directory was not created: %v", err)
	}
}

func TestTimeoutHandling(t *testing.T) {
	mcp, err := NewPlaywrightMCP(true, 500*time.Millisecond, t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create playwright MCP: %v", err)
	}
	defer mcp.Close()

	ctx := context.Background()
	action := BrowserAction{
		Timeout:    1 * time.Second,
		ScreenShot: false,
		OutputDir:  t.TempDir(),
	}

	// Navigate to a slow-loading page
	result, err := mcp.NavigateTo(ctx, "https://httpbin.org/delay/5", action)

	if err != nil {
		// Timeout expected
		return
	}

	if result.Error == "" {
		t.Errorf("Expected error for slow navigation")
	}
}

func TestFormSubmission(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/form":
			if r.Method == "POST" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("<html><body>Form submitted successfully</body></html>"))
			} else {
				w.Header().Set("Content-Type", "text/html")
				w.Write([]byte("<html><body><form method='post'><input name='username' placeholder='Enter username'><button type='submit'>Submit</button></form></body></html>"))
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	mcp, err := NewPlaywrightMCP(true, 30*time.Second, t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create playwright MCP: %v", err)
	}
	defer mcp.Close()

	ctx := context.Background()
	action := BrowserAction{
		Timeout:    5 * time.Second,
		ScreenShot: false,
		OutputDir:  t.TempDir(),
	}

	fields := map[string]string{
		"input[name='username']": "testuser",
	}

	result, err := mcp.FillForm(ctx, server.URL+"/form", fields, action)
	if err != nil {
		t.Fatalf("Form submission failed: %v", err)
	}

	if result.Title == "" {
		t.Errorf("Expected title after form submission")
	}
}

func TestBrowserLaunch(t *testing.T) {
	mcp, err := NewPlaywrightMCP(true, 30*time.Second, t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create playwright MCP: %v", err)
	}
	defer mcp.Close()

	err = mcp.LaunchBrowser()
	if err != nil {
		t.Errorf("Browser launch failed: %v", err)
	}

	// Verify browser is running
	if mcp.browser == nil {
		t.Error("Browser should be initialized after launch")
	}
}

func TestCloseCleanup(t *testing.T) {
	mcp, err := NewPlaywrightMCP(true, 30*time.Second, t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create playwright MCP: %v", err)
	}

	err = mcp.LaunchBrowser()
	if err != nil {
		t.Errorf("Browser launch failed: %v", err)
	}

	err = mcp.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	if mcp.browser != nil {
		t.Error("Browser should be closed after Close()")
	}
}

func TestMultiplePages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html><body>Page: " + r.URL.Path[1:] + "</body></html>"))
	}))
	defer server.Close()

	mcp, err := NewPlaywrightMCP(true, 30*time.Second, t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create playwright MCP: %v", err)
	}
	defer mcp.Close()

	ctx := context.Background()
	action := BrowserAction{
		Timeout:    5 * time.Second,
		ScreenShot: false,
		OutputDir:  t.TempDir(),
	}

	// Navigate to multiple pages
	urls := []string{"/page1", "/page2", "/page3"}
	for _, url := range urls {
		result, err := mcp.NavigateTo(ctx, server.URL+url, action)
		if err != nil {
			t.Errorf("Navigation failed for %s: %v", url, err)
		}

		expectedTitle := "Page: " + url[1:]
		if result.Title != expectedTitle {
			t.Errorf("Expected title '%s' but got '%s'", expectedTitle, result.Title)
		}
	}
}

func TestElementNotExists(t *testing.T) {
	mcp, err := NewPlaywrightMCP(true, 30*time.Second, t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create playwright MCP: %v", err)
	}
	defer mcp.Close()

	ctx := context.Background()
	action := BrowserAction{
		Timeout:    5 * time.Second,
		ScreenShot: false,
		OutputDir:  t.TempDir(),
	}

	result, err := mcp.NavigateTo(ctx, "http://example.com", action)
	if err != nil {
		t.Fatalf("Navigation failed: %v", err)
	}

	// Try to get non-existent elements
	elements, err := mcp.GetElements(ctx, "http://example.com", "nonexistent", action)
	if err != nil {
		t.Errorf("GetElements should handle missing elements gracefully: %v", err)
	}

	if len(elements) == 0 {
		t.Error("Expected at least one element for non-existent selector")
	}
}
