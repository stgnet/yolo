package playwright

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/playwright-community/playwright-go"
)

// PlaywrightMCP wraps Playwright for browser automation
type PlaywrightMCP struct {
	pw            *playwright.Playwright
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
	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to start playwright: %w", err)
	}

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
	headlessPtr := boolPtr(p.headless)
	opts := playwright.BrowserTypeLaunchOptions{
		Headless: headlessPtr,
	}
	var err error
	p.browser, err = browserType.Launch(opts)
	return err
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

	page.SetDefaultTimeout(float64(action.Timeout.Milliseconds()))

	resp, err := page.Goto(url)
	if err != nil {
		return nil, fmt.Errorf("failed to navigate to %s: %w", url, err)
	}

	result := &PlaywrightResult{
		URL:        resp.URL(),
		Title:      "",
		Text:       "",
		HTML:       "",
		Screenshot: "",
		Error:      "",
	}

	time.Sleep(1 * time.Second)

	title, err := page.Title()
	if err == nil {
		result.Title = title
	}
	text, _ := page.Content()
	result.Text = text
	html, _ := page.InnerHTML("body")
	result.HTML = html

	if action.ScreenShot {
		screenshotPath := filepath.Join(p.screenShotDir, fmt.Sprintf("screenshot_%d.png", time.Now().UnixNano()))
		page.Screenshot(playwright.PageScreenshotOptions{
			Path: playwright.String(screenshotPath),
		})
		result.Screenshot = screenshotPath
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

	page.SetDefaultTimeout(float64(action.Timeout.Milliseconds()))

	resp, _ := page.Goto("about:blank")

	page.WaitForSelector(selector)
	page.Click(selector)

	result := &PlaywrightResult{
		URL:   resp.URL(),
		Title: "",
		Text:  "",
	}

	title, _ := page.Title()
	result.Title = title
	text, _ := page.Content()
	result.Text = text

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

	page.SetDefaultTimeout(float64(action.Timeout.Milliseconds()))

	resp, _ := page.Goto(url)

	for selector, value := range fields {
		page.Fill(selector, value)
	}

	submitButton := "button[type='submit'], input[type='submit']"
	page.Click(submitButton)

	time.Sleep(2 * time.Second)

	result := &PlaywrightResult{
		URL:   resp.URL(),
		Title: "",
		Text:  "",
	}

	title, _ := page.Title()
	result.Title = title
	text, _ := page.Content()
	result.Text = text

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

	page.SetDefaultTimeout(float64(action.Timeout.Milliseconds()) * 1000)

	_, _ = page.Goto(url)

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

		element["index"] = fmt.Sprintf("%d", i)

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

// MarshalJSON custom marshaling for PlaywrightResult to handle JSON properly
func (p *PlaywrightResult) MarshalJSON() ([]byte, error) {
	type Alias PlaywrightResult
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(p),
	})
}

// Helper function to create bool pointer
func boolPtr(b bool) *bool {
	return &b
}
