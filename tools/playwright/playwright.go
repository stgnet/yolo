package playwright

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/playwright-community/playwright-go"
)

// PlaywrightMCP wraps Playwright for browser automation
type PlaywrightMCP struct {
	pw            *playwright.API
	browser       playwright.Browser
	headless      bool
	timeout       time.Duration
	screenShotDir string
}

// BrowserAction represents a browser action to perform
type BrowserAction struct {
	Timeout    time.Duration
	ScreenShot bool
	OutputDir  string
}

// PlaywrightResult represents the result of browser automation
type PlaywrightResult struct {
	Title      string              `json:"title,omitempty"`
	URL        string              `json:"url,omitempty"`
	Text       string              `json:"text,omitempty"`
	HTML       string              `json:"html,omitempty"`
	Screenshot string              `json:"screenshot,omitempty"`
	Elements   []map[string]string `json:"elements,omitempty"`
	Error      string              `json:"error,omitempty"`
	Caption    string              `json:"caption,omitempty"`
}

// NewPlaywrightMCP creates a new PlaywrightMCP instance
func NewPlaywrightMCP(headless bool, timeout time.Duration, screenshotDir string) (*PlaywrightMCP, error) {
	// Initialize Playwright
	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to start playwright: %w", err)
	}

	// Set screenshot directory
	if screenshotDir == "" {
		screenshotDir = "./screenshots"
	}
	if err := os.MkdirAll(screenshotDir, 0755); err != nil {
		pw.Stop()
		return nil, fmt.Errorf("failed to create screenshot directory: %w", err)
	}

	return &PlaywrightMCP{
		pw:            pw,
		headless:      headless,
		timeout:       timeout,
		screenShotDir: screenshotDir,
	}, nil
}

// LaunchBrowser launches a new browser instance
func (p *PlaywrightMCP) LaunchBrowser() error {
	if p.browser != nil {
		p.browser.Close()
	}

	browserType := p.pw.Chromium
	if p.headless {
		p.browser, _ = browserType.Launch(
			playwright.BrowserTypeLaunchOptions{
				Headless: playwright.Bool(p.headless),
			},
		)
	} else {
		p.browser, _ = browserType.Launch()
	}

	return nil
}

// NavigateTo navigates to a URL
func (p *PlaywrightMCP) NavigateTo(ctx context.Context, url string, action BrowserAction) (*PlaywrightResult, error) {
	if p.browser == nil {
		if err := p.LaunchBrowser(); err != nil {
			return nil, err
		}
	}

	page, err := p.browser.NewPage()
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}
	defer page.Close()

	// Set timeout
	page.SetDefaultTimeout(playwright.Float(float64(action.Timeout.Milliseconds())))

	// Navigate to URL
	ctxFunc, cancel := context.WithTimeout(ctx, action.Timeout)
	defer cancel()

	resp, err := page.GotoContext(ctxFunc, url, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateCommit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to navigate to %s: %w", url, err)
	}

	result := &PlaywrightResult{
		URL:        resp.URL(),
		Title:      page.Title(),
		Text:       "",
		HTML:       "",
		Screenshot: "",
		Error:      "",
	}

	// Wait for network to be idle
	time.Sleep(1 * time.Second)

	// Get content
	result.Text = page.Content()
	result.HTML, _ = page.InnerHTML("body")

	// Take screenshot if requested
	if action.ScreenShot {
		screenshotPath := filepath.Join(p.screenShotDir, fmt.Sprintf("screenshot_%d.png", time.Now().UnixNano()))
		err = page.Screenshot(
			playwright.PageScreenshotOptions{
				Path: playwright.String(screenshotPath),
			},
		)
		if err == nil {
			result.Screenshot = screenshotPath
		}
	}

	return result, nil
}

// ClickElement clicks an element by selector
func (p *PlaywrightMCP) ClickElement(ctx context.Context, selector string, action BrowserAction) (*PlaywrightResult, error) {
	if p.browser == nil {
		if err := p.LaunchBrowser(); err != nil {
			return nil, err
		}
	}

	page, err := p.browser.NewPage()
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}
	defer page.Close()

	page.SetDefaultTimeout(playwright.Float(float64(action.Timeout.Milliseconds())))

	ctxFunc, cancel := context.WithTimeout(ctx, action.Timeout)
	defer cancel()

	resp, err := page.GotoContext(ctxFunc, "about:blank", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateCommit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to navigate: %w", err)
	}

	// Wait for element and click
	err = page.WaitForSelector(selector, playwright.PageWaitForSelectorOptions{})
	if err != nil {
		return nil, fmt.Errorf("element not found: %s", selector)
	}

	err = page.Click(selector, playwright.PageClickOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to click element: %w", err)
	}

	result := &PlaywrightResult{
		URL:   resp.URL(),
		Title: page.Title(),
		Text:  page.Content(),
	}

	return result, nil
}

// FillForm fills form fields and submits
func (p *PlaywrightMCP) FillForm(ctx context.Context, url string, fields map[string]string, action BrowserAction) (*PlaywrightResult, error) {
	if p.browser == nil {
		if err := p.LaunchBrowser(); err != nil {
			return nil, err
		}
	}

	page, err := p.browser.NewPage()
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}
	defer page.Close()

	page.SetDefaultTimeout(playwright.Float(float64(action.Timeout.Milliseconds())))

	ctxFunc, cancel := context.WithTimeout(ctx, action.Timeout)
	defer cancel()

	_, err = page.GotoContext(ctxFunc, url, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateCommit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to navigate: %w", err)
	}

	// Fill form fields
	for selector, value := range fields {
		err = page.Fill(selector, value, playwright.PageFillOptions{})
		if err != nil {
			log.Printf("Warning: failed to fill field %s: %v", selector, err)
		}
	}

	// Submit form (try the first submit button found)
	submitButton := "button[type='submit'], input[type='submit']"
	err = page.Click(submitButton, playwright.PageClickOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to submit form: %w", err)
	}

	time.Sleep(2 * time.Second)

	result := &PlaywrightResult{
		URL:   page.URL(),
		Title: page.Title(),
		Text:  page.Content(),
	}

	return result, nil
}

// GetElements extracts elements matching selector
func (p *PlaywrightMCP) GetElements(ctx context.Context, url, selector string, action BrowserAction) ([]map[string]string, error) {
	if p.browser == nil {
		if err := p.LaunchBrowser(); err != nil {
			return nil, err
		}
	}

	page, err := p.browser.NewPage()
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}
	defer page.Close()

	page.SetDefaultTimeout(playwright.Float(float64(action.Timeout.Milliseconds()) * 1000))

	ctxFunc, cancel := context.WithTimeout(ctx, action.Timeout)
	defer cancel()

	_, err = page.GotoContext(ctxFunc, url, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateCommit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to navigate: %w", err)
	}

	elements, err := p.QueryElements(page, selector)
	if err != nil {
		return nil, fmt.Errorf("failed to query elements: %w", err)
	}

	resultElements := make([]map[string]string, 0)
	for i, el := range elements {
		element := make(map[string]string)
		element["selector"] = selector

		text, _ := el.InnerText()
		element["text"] = text

		html, _ := el.InnerHTML()
		element["html"] = html

		resultElements = append(resultElements, element)
	}

	if len(resultElements) == 0 {
		return []map[string]string{
			{"selector": selector, "text": "No elements found"},
		}, nil
	}

	return resultElements, nil
}

// Close closes the browser and playwright instance
func (p *PlaywrightMCP) Close() error {
	if p.browser != nil {
		p.browser.Close()
	}
	p.pw.Stop()
	return nil
}

// Helper function to iterate over all matching elements
func pageQueryAll(selector string, callback func(int, playwright.ElementHandle) error) error {
	return nil // Placeholder - implementation would require Playwright API access
}

// QueryElements queries the page for elements matching a selector
func (p *PlaywrightMCP) QueryElements(page playwright.Page, selector string) ([]playwright.ElementHandle, error) {
	elements, err := page.QuerySelectorAll(selector)
	if err != nil {
		return nil, err
	}

	result := make([]playwright.ElementHandle, 0)
	for _, el := range elements {
		result = append(result, el)
	}

	return result, nil
}

// SearchResults represents the output structure for search actions
type SearchResults struct {
	Query      string              `json:"query"`
	URL        string              `json:"url"`
	Title      string              `json:"title"`
	Text       string              `json:"text"`
	Screenshot string              `json:"screenshot,omitempty"`
	Elements   []map[string]string `json:"elements,omitempty"`
	Error      string              `json:"error,omitempty"`
}
