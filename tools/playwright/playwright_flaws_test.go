// These tests expose flaws in the Playwright library implementation.
//
// NOTE: The base code (playwright.go) has 11+ compilation errors due to
// incorrect playwright-go API usage:
//   - playwright.API type doesn't exist (should be *playwright.Playwright)
//   - page.GotoContext doesn't exist (should be page.Goto)
//   - page.Title() returns (string, error) but error is discarded
//   - page.Content() returns (string, error) but error is discarded
//   - page.Screenshot() returns ([]byte, error) but used as single-value
//   - page.SetDefaultTimeout takes float64, not *float64
//   - page.WaitForSelector returns (ElementHandle, error) but error only captured
//
// Until those compilation errors are fixed, these tests cannot run.
// They are written to be ready once the code compiles.

package playwright

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// playwrightInstalled checks if playwright driver is available
func playwrightInstalled() bool {
	_, err := NewPlaywrightMCP(true, 30*time.Second, "/tmp")
	return err == nil || (!strings.Contains(err.Error(), "please install the driver"))
}

// TestLaunchBrowserErrorNotSwallowed verifies that LaunchBrowser propagates
// errors from browserType.Launch() instead of silently discarding them.
// Flaw: lines 76-83 use `_ =` to discard the launch error.
func TestLaunchBrowserErrorNotSwallowed(t *testing.T) {
	mcp, err := NewPlaywrightMCP(true, 30*time.Second, t.TempDir())
	if err != nil && strings.Contains(err.Error(), "please install the driver") {
		t.Skip("playwright driver not installed - skipping integration test")
	}
	if err != nil {
		t.Fatalf("Failed to create PlaywrightMCP: %v", err)
	}
	defer mcp.Close()

	err = mcp.LaunchBrowser()
	if err != nil {
		t.Fatalf("LaunchBrowser failed: %v", err)
	}

	// After a successful launch, browser must not be nil
	if mcp.browser == nil {
		t.Error("LaunchBrowser succeeded but browser is nil — launch error was silently swallowed")
	}
}

// TestCloseNilsBrowser verifies that Close() sets browser to nil so the
// instance can detect it's been shut down.
// Flaw: Close() calls browser.Close() but never sets p.browser = nil.
func TestCloseNilsBrowser(t *testing.T) {
	mcp, err := NewPlaywrightMCP(true, 30*time.Second, t.TempDir())
	if err != nil && strings.Contains(err.Error(), "please install the driver") {
		t.Skip("playwright driver not installed - skipping integration test")
	}
	if err != nil {
		t.Fatalf("Failed to create PlaywrightMCP: %v", err)
	}

	if err := mcp.LaunchBrowser(); err != nil {
		t.Fatalf("LaunchBrowser failed: %v", err)
	}

	if err := mcp.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if mcp.browser != nil {
		t.Error("After Close(), browser should be nil but it is still set — Close does not nil out the field")
	}
}

// TestGetElementsTimeoutNotMultiplied verifies that GetElements uses the
// correct timeout value instead of multiplying milliseconds by 1000.
// Flaw: line 260 does `action.Timeout.Milliseconds() * 1000`, turning 5s into ~83 minutes.
func TestGetElementsTimeoutNotMultiplied(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html><body><h1>Hello</h1></body></html>"))
	}))
	defer server.Close()

	mcp, err := NewPlaywrightMCP(true, 30*time.Second, t.TempDir())
	if err != nil && strings.Contains(err.Error(), "please install the driver") {
		t.Skip("playwright driver not installed - skipping integration test")
	}
	if err != nil {
		t.Fatalf("Failed to create PlaywrightMCP: %v", err)
	}
	defer mcp.Close()

	ctx := context.Background()

	// Use a very short timeout: 500ms
	// With the bug, page timeout = 500ms * 1000 = 500,000ms = 8+ minutes
	shortAction := BrowserAction{
		Timeout:    500 * time.Millisecond,
		ScreenShot: false,
	}

	start := time.Now()
	_, _ = mcp.GetElements(ctx, server.URL, "nonexistent-element-xyz", shortAction)
	elapsed := time.Since(start)

	// Should complete in well under 5 seconds with a 500ms timeout.
	// With the bug (500ms * 1000 = 500s), it would hang far longer.
	if elapsed > 10*time.Second {
		t.Errorf("GetElements took %v — timeout was likely multiplied by 1000 (bug: ms * 1000)", elapsed)
	}
}

// TestNavigateToUsesLocalServer verifies NavigateTo works with a local
// httptest server and returns correct content.
// Exposes that existing tests hit external URLs (example.com, httpbin.org).
func TestNavigateToUsesLocalServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html><head><title>Local Test</title></head><body><h1>Hello World</h1></body></html>"))
	}))
	defer server.Close()

	mcp, err := NewPlaywrightMCP(true, 30*time.Second, t.TempDir())
	if err != nil && strings.Contains(err.Error(), "please install the driver") {
		t.Skip("playwright driver not installed - skipping integration test")
	}
	if err != nil {
		t.Fatalf("Failed to create PlaywrightMCP: %v", err)
	}
	defer mcp.Close()

	ctx := context.Background()
	action := BrowserAction{
		Timeout:    10 * time.Second,
		ScreenShot: false,
	}

	result, err := mcp.NavigateTo(ctx, server.URL, action)
	if err != nil {
		t.Fatalf("NavigateTo failed: %v", err)
	}

	if result.URL == "" {
		t.Error("Expected URL in result")
	}
	if !strings.Contains(result.URL, "127.0.0.1") && !strings.Contains(result.URL, "localhost") {
		t.Errorf("Expected local URL, got: %s", result.URL)
	}
	if !strings.Contains(result.HTML, "Hello World") {
		t.Errorf("Expected body to contain 'Hello World', got: %s", result.HTML)
	}
}

// TestMultiplePagesWithTitleTags verifies page titles are read correctly.
// Flaw: existing TestMultiplePages checks result.Title but the test server
// doesn't emit <title> tags, so assertions can never pass.
func TestMultiplePagesWithTitleTags(t *testing.T) {
	if !playwrightInstalled() {
		t.Skip("playwright driver not installed - skipping integration test")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		name := strings.TrimPrefix(r.URL.Path, "/")
		w.Write([]byte(fmt.Sprintf("<html><head><title>%s</title></head><body>Page: %s</body></html>", name, name)))
	}))
	defer server.Close()

	mcp, err := NewPlaywrightMCP(true, 30*time.Second, t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create PlaywrightMCP: %v", err)
	}
	defer mcp.Close()

	ctx := context.Background()
	action := BrowserAction{
		Timeout:    10 * time.Second,
		ScreenShot: false,
	}

	pages := []string{"page1", "page2", "page3"}
	for _, name := range pages {
		result, err := mcp.NavigateTo(ctx, server.URL+"/"+name, action)
		if err != nil {
			t.Errorf("Navigation to /%s failed: %v", name, err)
			continue
		}

		if result.Title != name {
			t.Errorf("For /%s: expected title '%s', got '%s'", name, name, result.Title)
		}
		if !strings.Contains(result.HTML, "Page: "+name) {
			t.Errorf("For /%s: expected body to contain 'Page: %s', got: %s", name, name, result.HTML)
		}
	}
}

// TestClickElementWorksCorrectly verifies that ClickElement navigates to the correct page
// and clicks the element properly.
func TestClickElementWorksCorrectly(t *testing.T) {
	if !playwrightInstalled() {
		t.Skip("playwright driver not installed - skipping integration test")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body><button id="btn">Click Me</button></body></html>`))
	}))
	defer server.Close()

	mcp, err := NewPlaywrightMCP(true, 30*time.Second, t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create PlaywrightMCP: %v", err)
	}
	defer mcp.Close()

	ctx := context.Background()
	action := BrowserAction{
		Timeout:    3 * time.Second,
		ScreenShot: false,
	}

	result, err := mcp.ClickElement(ctx, server.URL, "#btn", action)
	if err != nil {
		t.Fatalf("ClickElement failed: %v", err)
	}

	if result.URL == "" {
		t.Error("Expected URL in result")
	}

	// Verify the click worked
	if result.Title == "" {
		t.Errorf("Expected title in result after click")
	}
}

// TestNavigateToUsesProperWait verifies that NavigateTo uses Playwright's
// built-in wait mechanisms instead of hardcoding time.Sleep(1s).
func TestNavigateToUsesProperWait(t *testing.T) {
	if !playwrightInstalled() {
		t.Skip("playwright driver not installed - skipping integration test")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html><body><p>Fast content</p></body></html>"))
	}))
	defer server.Close()

	mcp, err := NewPlaywrightMCP(true, 30*time.Second, t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create PlaywrightMCP: %v", err)
	}
	defer mcp.Close()

	ctx := context.Background()
	action := BrowserAction{
		Timeout:    10 * time.Second,
		ScreenShot: false,
	}

	start := time.Now()
	result, err := mcp.NavigateTo(ctx, server.URL, action)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("NavigateTo failed: %v", err)
	}

	// Verify result is valid
	if result == nil || result.URL == "" {
		t.Error("Expected valid result from NavigateTo")
	}

	// Should complete in under 500ms for an instant page (no hardcoded sleep)
	if elapsed >= 1*time.Second {
		t.Errorf("NavigateTo took %v for an instant page — may have unnecessary delays", elapsed)
	}
}

// TestPageQueryAllIsStub verifies that pageQueryAll is a non-functional stub.
// NOTE: Disabled - pageQueryAll function no longer exists in the codebase after cleanup.
/*func TestPageQueryAllIsStub(t *testing.T) {
	callbackCalled := false
	err := pageQueryAll("h1", func(i int, el playwright.ElementHandle) error {
		callbackCalled = true
		return nil
	})

	if err != nil {
		t.Errorf("pageQueryAll returned error: %v", err)
	}

	if !callbackCalled {
		t.Error("pageQueryAll is a stub — callback was never invoked, the function does nothing")
	}
}*/
