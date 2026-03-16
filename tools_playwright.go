// Playwright MCP Tool - Browser automation for YOLO
// Provides automated browser interactions including navigation, DOM inspection, form filling, etc.

package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// playwrightMCPExecutor implements Playwright MCP operations for browser automation
type playwrightMCPExecutor struct {
	baseDir string
}

// playwrightActionResult represents the result of a browser operation
type playwrichtActionResult struct {
	Status     string `json:"status"`
	URL        string `json:"url,omitempty"`
	Title      string `json:"title,omitempty"`
	Content    string `json:"content,omitempty"`
	Error      string `json:"error,omitempty"`
	Screenshot string `json:"screenshot,omitempty"`
}

// newPlaywrightMCPExecutor creates a new executor for Playwright MCP operations
func newPlaywrightMCPExecutor(baseDir string) *playwrightMCPExecutor {
	return &playwrightMCPExecutor{baseDir: baseDir}
}

// navigate navigates to a URL and returns the page state
func (p *playwrightMCPExecutor) navigate(url string, waitUntil string) string {
	// Use Playwright's CLI via a Node.js wrapper or direct Puppeteer/Playwright execution
	// For now, we'll use run_command to execute Playwright scripts

	script := fmt.Sprintf(`
const playwright = require('@playwright/test');

(async () => {
  const browser = await playwright.chromium.launch();
  const context = await browser.newContext();
  const page = await context.newPage();
  
  // Navigate to the URL
  await page.navigate('%s', { waitUntil: '%s' });
  
  // Get page info
  const title = page.titleText;
  const url = page.url();
  
  // Optionally take screenshot
  const screenshotPath = '/tmp/screenshot.png';
  try {
    await page.screenshot({ path: screenshotPath });
  } catch (e) {
    console.error('Screenshot failed:', e);
  }
  
  console.log(JSON.stringify({
    title,
    url,
    screenshot: screenshotPath
  }));
  
  await browser.close();
})();
`, escapeJS(url), waitUntil)

	// Execute the script using node
	cmd := exec.Command("node", "-e", script)
	cmd.Dir = p.baseDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error navigating to %s: %v\nOutput: %s", url, err, string(output))
	}

	var result playwrichtActionResult
	if err := json.Unmarshal(output, &result); err == nil {
		if result.Error != "" {
			return fmt.Sprintf("Browser error: %s", result.Error)
		}
		return fmt.Sprintf("Navigated to %s\nTitle: %s\nURL: %s", url, result.Title, result.URL)
	}

	return string(output)
}

// clickElement performs a click action on an element
func (p *playwrightMCPExecutor) clickElement(selector string, timeout int) string {
	script := fmt.Sprintf(`
const playwright = require('@playwright/test');

(async () => {
  const browser = await playwright.chromium.launch();
  const context = await browser.newContext();
  const page = await context.newPage();
  
  await page.goto('%s', { waitUntil: 'domcontentloaded' });
  
  // Wait for element and click
  try {
    await page.waitForSelector('%s', { timeout: %d });
    await page.click('%s');
    
    console.log(JSON.stringify({
      success: true,
      message: 'Click successful on ' + '%s'
    }));
  } catch (e) {
    console.log(JSON.stringify({
      success: false,
      error: e.message
    }));
  }
  
  await browser.close();
})();
`, escapeJS("about:blank"), selector, timeout, selector, selector)

	cmd := exec.Command("node", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error clicking %s: %v\nOutput: %s", selector, err, string(output))
	}

	return string(output)
}

// fillInput fills an input field with text
func (p *playwrightMCPExecutor) fillInput(selector, value string) string {
	script := fmt.Sprintf(`
const playwright = require('@playwright/test');

(async () => {
  const browser = await playwright.chromium.launch();
  const context = await browser.newContext();
  const page = await context.newPage();
  
  await page.goto('%s', { waitUntil: 'domcontentloaded' });
  
  // Fill input field
  try {
    await page.waitForSelector('%s');
    await page.fill('%s', '%s');
    
    console.log(JSON.stringify({
      success: true,
      message: 'Filled input %s with text of length %d'
    }, ['%s', %d]));
  } catch (e) {
    console.log(JSON.stringify({
      success: false,
      error: e.message
    }));
  }
  
  await browser.close();
})();
`, escapeJS("about:blank"), selector, selector, escapeJS(value), selector, len(value), selector, len(value))

	cmd := exec.Command("node", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error filling %s: %v\nOutput: %s", selector, err, string(output))
	}

	return string(output)
}

// getHTML retrieves the current page HTML/content
func (p *playwrightMCPExecutor) getHTML(selector string) string {
	script := fmt.Sprintf(`
const playwright = require('@playwright/test');

(async () => {
  const browser = await playwright.chromium.launch();
  const context = await browser.newContext();
  const page = await context.newPage();
  
  await page.goto('%s', { waitUntil: 'networkidle' });
  
  try {
    let content;
    if ('%s') {
      const element = await page.$('%s');
      if (element) {
        content = await element.evaluate(el => el.innerHTML);
      } else {
        content = 'Element not found';
      }
    } else {
      content = await page.content();
    }
    
    console.log(JSON.stringify({
      content,
      url: page.url()
    }));
  } catch (e) {
    console.log(JSON.stringify({
      error: e.message
    }));
  }
  
  await browser.close();
})();
`, escapeJS("about:blank"), selector, selector)

	cmd := exec.Command("node", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error getting HTML: %v\nOutput: %s", err, string(output))
	}

	return string(output)
}

// screenshot captures a screenshot of the page
func (p *playwrightMCPExecutor) screenshot(outputPath string) string {
	absPath := outputPath
	if !filepath.IsAbs(outputPath) {
		absPath = filepath.Join(p.baseDir, outputPath)
	}

	script := fmt.Sprintf(`
const playwright = require('@playwright/test');

(async () => {
  const browser = await playwright.chromium.launch();
  const context = await browser.newContext();
  const page = await context.newPage();
  
  await page.goto('%s', { waitUntil: 'networkidle' });
  
  try {
    await page.screenshot({ path: '%s' });
    console.log(JSON.stringify({
      success: true,
      screenshot: '%s'
    }));
  } catch (e) {
    console.log(JSON.stringify({
      success: false,
      error: e.message
    }));
  }
  
  await browser.close();
})();
`, escapeJS("about:blank"), absPath, absPath)

	cmd := exec.Command("node", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error taking screenshot: %v\nOutput: %s", err, string(output))
	}

	return string(output)
}

// Helper function to escape JavaScript strings
func escapeJS(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}

// Execute a Playwright MCP command
func (t *ToolExecutor) playwrightMCP(args map[string]any) string {
	action := getStringArg(args, "action", "")

	if action == "" {
		return errorMessage("action is required. Available actions: navigate, click, fill, getHTML, screenshot")
	}

	executor := newPlaywrightMCPExecutor(t.baseDir)

	switch action {
	case "navigate":
		url := getStringArg(args, "url", "")
		waitUntil := getStringArg(args, "waitUntil", "domcontentloaded")
		return executor.navigate(url, waitUntil)

	case "click":
		selector := getStringArg(args, "selector", "")
		timeout := getIntArg(args, "timeout", 5000)
		if selector == "" {
			return errorMessage("selector is required for click action")
		}
		return executor.clickElement(selector, timeout)

	case "fill":
		selector := getStringArg(args, "selector", "")
		value := getStringArg(args, "value", "")
		if selector == "" || value == "" {
			return errorMessage("selector and value are required for fill action")
		}
		return executor.fillInput(selector, value)

	case "getHTML":
		selector := getStringArg(args, "selector", "")
		return executor.getHTML(selector)

	case "screenshot":
		path := getStringArg(args, "path", "/tmp/screenshot.png")
		return executor.screenshot(path)

	default:
		return errorMessage("unknown action '%s'. Available actions: navigate, click, fill, getHTML, screenshot", action)
	}
}
