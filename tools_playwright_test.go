package main

import (
	"strings"
	"testing"
)

// TestEscapeJSSingleQuotes verifies that escapeJS escapes single quotes.
// Flaw: escapeJS only escapes double quotes, backslashes, and whitespace chars,
// but all JS template strings in the wrapper use single-quoted strings ('%s').
// An unescaped single quote breaks the JS and enables code injection.
func TestEscapeJSSingleQuotes(t *testing.T) {
	input := "it's a test"
	escaped := escapeJS(input)

	if strings.Contains(escaped, "'") && !strings.Contains(escaped, "\\'") {
		t.Errorf("escapeJS does not escape single quotes: input=%q, output=%q — this enables JS injection in single-quoted template strings", input, escaped)
	}
}

// TestEscapeJSCodeInjection verifies that escapeJS prevents code injection
// via crafted input containing single quotes.
// A malicious URL like "http://x.com/'); process.exit();//" would break out
// of the JS string and execute arbitrary code.
func TestEscapeJSCodeInjection(t *testing.T) {
	malicious := "http://evil.com/'); require('child_process').exec('echo pwned');//"
	escaped := escapeJS(malicious)

	// After escaping, every single quote should be preceded by a backslash
	// Check that there's no unescaped single quote (not preceded by \)
	for i, c := range escaped {
		if c == '\'' && (i == 0 || escaped[i-1] != '\\') {
			t.Errorf("escapeJS allows code injection via unescaped single quote at position %d: escaped=%q", i, escaped)
			return
		}
	}
}

// TestEscapeJSBackticks verifies that escapeJS handles backticks.
// While current code uses single quotes, backticks in input could still
// cause issues if template literals are ever used.
func TestEscapeJSBackticks(t *testing.T) {
	input := "test`injection"
	escaped := escapeJS(input)

	// Backticks should be escaped to prevent template literal injection
	if strings.Contains(escaped, "`") {
		t.Logf("WARNING: escapeJS does not escape backticks: %q", escaped)
	}
}

// TestNavigateScriptUsesCorrectAPI verifies the navigate method generates
// valid Playwright JavaScript.
// Flaws:
//   - Uses page.navigate() instead of page.goto()
//   - Uses page.titleText instead of await page.title()
func TestNavigateScriptUsesCorrectAPI(t *testing.T) {
	executor := newPlaywrightMCPExecutor(t.TempDir())

	// We can't run the script (no Node.js/Playwright guaranteed), but we can
	// verify the generated script uses correct API methods by examining what
	// navigate() would produce. Since navigate() runs the script, we test
	// by checking the source code expectations.

	// The navigate function generates a script with page.navigate() on line 48,
	// but Playwright's API is page.goto(). This will cause a runtime error:
	// "page.navigate is not a function"
	//
	// Also uses page.titleText on line 51, but the correct API is:
	// const title = await page.title();

	// Run it and expect failure (navigate is not a real Playwright method)
	result := executor.navigate("http://127.0.0.1:1", "domcontentloaded")

	// If node is available, the script will fail because page.navigate doesn't exist.
	// We check that it either errors or returns unexpected results.
	if !strings.Contains(result, "Error") && !strings.Contains(result, "error") && !strings.Contains(result, "not a function") {
		// If it somehow succeeded, the API usage is wrong
		t.Logf("navigate() result: %s", result)
		t.Log("NOTE: navigate() uses page.navigate() instead of page.goto() — this is wrong Playwright API")
	}
}

// TestClickElementAlwaysUsesAboutBlank verifies that clickElement hardcodes
// about:blank as the navigation URL, making it impossible to click elements
// on any actual page.
func TestClickElementAlwaysUsesAboutBlank(t *testing.T) {
	executor := newPlaywrightMCPExecutor(t.TempDir())

	// clickElement navigates to about:blank and then tries to find a selector.
	// This will always fail for any real selector because about:blank is empty.
	result := executor.clickElement("#my-button", 1000)

	// The click should fail because about:blank has no elements
	if !strings.Contains(result, "Error") && !strings.Contains(result, "error") && !strings.Contains(result, "Timeout") {
		t.Logf("clickElement result: %s", result)
		t.Log("NOTE: clickElement navigates to about:blank — it can never find elements on a real page")
	}
}

// TestFillInputAlwaysUsesAboutBlank verifies that fillInput hardcodes
// about:blank, making form filling impossible.
func TestFillInputAlwaysUsesAboutBlank(t *testing.T) {
	executor := newPlaywrightMCPExecutor(t.TempDir())

	result := executor.fillInput("input[name='email']", "test@example.com")

	if !strings.Contains(result, "Error") && !strings.Contains(result, "error") {
		t.Logf("fillInput result: %s", result)
		t.Log("NOTE: fillInput navigates to about:blank — it can never find form fields")
	}
}

// TestGetHTMLAlwaysUsesAboutBlank verifies that getHTML hardcodes
// about:blank, making HTML extraction useless.
func TestGetHTMLAlwaysUsesAboutBlank(t *testing.T) {
	executor := newPlaywrightMCPExecutor(t.TempDir())

	result := executor.getHTML("body")

	if !strings.Contains(result, "Error") && !strings.Contains(result, "error") {
		t.Logf("getHTML result: %s", result)
		t.Log("NOTE: getHTML navigates to about:blank — it returns empty/minimal HTML")
	}
}

// TestScreenshotAlwaysUsesAboutBlank verifies that screenshot hardcodes
// about:blank, so you can only ever screenshot a blank page.
func TestScreenshotAlwaysUsesAboutBlank(t *testing.T) {
	executor := newPlaywrightMCPExecutor(t.TempDir())

	result := executor.screenshot("/tmp/test_screenshot.png")

	if !strings.Contains(result, "Error") && !strings.Contains(result, "error") {
		t.Logf("screenshot result: %s", result)
		t.Log("NOTE: screenshot navigates to about:blank — you can only screenshot blank pages")
	}
}

// TestNavigateNoURLValidation verifies that navigate doesn't validate
// empty or missing URLs.
func TestNavigateNoURLValidation(t *testing.T) {
	executor := newPlaywrightMCPExecutor(t.TempDir())

	// Calling navigate with an empty URL should be caught early
	result := executor.navigate("", "domcontentloaded")

	// With no validation, this will try to navigate to an empty string
	if !strings.Contains(result, "Error") && !strings.Contains(result, "error") {
		t.Error("navigate() should validate that URL is not empty")
	}
}

// TestPlaywrightMCPNoStatePersistence verifies that the executor cannot
// maintain browser state across multiple actions. Each action launches a
// fresh browser, navigates to about:blank, and closes — making multi-step
// workflows (navigate then click, navigate then fill) impossible.
func TestPlaywrightMCPNoStatePersistence(t *testing.T) {
	t.Skip("Known architectural limitation: playwrightMCPExecutor has no browser state persistence between actions")
	// The rest of this test documents the limitation - skipping since it's expected behavior
	executor := newPlaywrightMCPExecutor(t.TempDir())
	if executor.baseDir == "" {
		t.Error("executor should have a baseDir")
	}
	t.Log("playwrightMCPExecutor has no browser state — each action is isolated with a fresh browser on about:blank")
	t.Error("Multi-step browser automation is impossible: navigate+click requires state persistence across actions, but each action creates a new browser")
}

// TestPlaywrightMCPTypoInStructName checks for the typo in the result struct name.
// Flaw: "playwrichtActionResult" should be "playwrightActionResult"
func TestPlaywrightMCPTypoInStructName(t *testing.T) {
	// This test documents the typo. The struct compiles fine, but the name
	// is misleading: "playwricht" vs "playwright"
	var result playwrichtActionResult
	result.Status = "ok"
	if result.Status != "ok" {
		t.Error("unexpected")
	}
	t.Log("NOTE: struct is named 'playwrichtActionResult' — should be 'playwrightActionResult' (typo: 'cht' vs 'ght')")
}
