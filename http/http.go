package http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HTTPError represents an error during HTTP operations
type HTTPError struct {
	Code       int
	Message    string
	RetryAfter time.Duration
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.Code, e.Message)
}

// Response represents an HTTP response wrapper with useful metadata
type Response struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
	URL        string
	Duration   time.Duration
	Retries    int
}

// Request represents an outbound HTTP request
type Request struct {
	Method     string
	URL        string
	Body       []byte
	Timeout    time.Duration
	RetryCount int
	Cancel     <-chan struct{}
}

// DefaultRetryAfter is the default backoff for rate-limited requests
var DefaultRetryAfter = 30 * time.Second

// HTTPClient handles all HTTP operations with built-in retry logic
type HTTPClient struct {
	client *http.Client
}

// NewHTTPClient creates a new HTTP client with sensible defaults
func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("stopped after 10 redirects")
				}
				return nil
			},
		},
	}
}

// Do performs an HTTP request with automatic retry on transient failures
func (c *HTTPClient) Do(req Request) (*Response, error) {
	start := time.Now()
	var lastErr error

	for attempt := 0; attempt <= req.RetryCount; attempt++ {
		resp, err := c.doRequest(req, start)
		if resp != nil || err == nil {
			return resp, err
		}

		lastErr = err

		// Check if we should retry (5xx errors, timeout, connection refused)
		httpErr, isHTTPErr := err.(*HTTPError)
		if !isHTTPErr || httpErr.Code < 500 {
			break
		}

		if attempt < req.RetryCount {
			retryAfter := httpErr.RetryAfter
			if retryAfter == 0 {
				// Exponential backoff: 1s, 2s, 4s
				retryAfter = time.Duration(1<<uint(attempt)) * time.Second
				if retryAfter > 30*time.Second {
					retryAfter = 30 * time.Second
				}
			}

			select {
			case <-req.Cancel:
				return nil, fmt.Errorf("request cancelled before retry %d", attempt+1)
			case <-time.After(retryAfter):
			}
		}
	}

	return nil, lastErr
}

func (c *HTTPClient) doRequest(req Request, startTime time.Time) (*Response, error) {
	bodyReader := bytes.NewReader(req.Body)
	httpReq, err := http.NewRequestWithContext(c.withCancel(req), req.Method, req.URL, bodyReader)
	if err != nil {
		return nil, &HTTPError{Code: 0, Message: fmt.Sprintf("failed to create request: %v", err)}
	}

	httpReq.Header.Set("User-Agent", "YOLO/1.0 (+https://github.com/scottysgriepentrog/YOLO)")
	httpReq.Header.Set("Accept", "application/json, text/plain, */*")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, c.mapHTTPError(err)
	}

	duration := time.Since(startTime)
	bodyBytes, err := io.ReadAll(resp.Body)
	resp.Body.Close()

	if err != nil {
		return nil, &HTTPError{Code: resp.StatusCode, Message: fmt.Sprintf("failed to read body: %v", err)}
	}

	rateLimited := resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable

	var retryAfter time.Duration
	if rateLimited {
		if rt := resp.Header.Get("Retry-After"); rt != "" {
			retryAfter, _ = time.ParseDuration(rt + "s")
		} else if retryAfter == 0 {
			retryAfter = DefaultRetryAfter
		}
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Body:       bodyBytes,
		Headers:    resp.Header.Clone(),
		URL:        resp.Request.URL.String(),
		Duration:   duration,
		Retries:    0,
	}, nil
}

func (c *HTTPClient) withCancel(req Request) context.Context {
	if req.Cancel == nil {
		return context.Background()
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		select {
		case <-req.Cancel:
			cancel()
		case <-time.After(req.Timeout):
			cancel()
		}
	}()

	return ctx
}

func (c *HTTPClient) mapHTTPError(err error) error {
	if err == context.DeadlineExceeded {
		return &HTTPError{Code: 408, Message: "request timeout"}
	}
	if strings.Contains(err.Error(), "connection refused") {
		return &HTTPError{Code: 503, Message: "connection refused"}
	}
	return err
}

// Get performs a GET request with automatic retry
func (c *HTTPClient) Get(url string, timeout time.Duration, maxRetries int) (*Response, error) {
	req := Request{
		Method:     http.MethodGet,
		URL:        url,
		Timeout:    timeout,
		RetryCount: maxRetries,
	}
	return c.Do(req)
}

// Post performs a POST request with JSON body and automatic retry
func (c *HTTPClient) Post(url string, data []byte, timeout time.Duration, maxRetries int) (*Response, error) {
	req := Request{
		Method:     http.MethodPost,
		URL:        url,
		Body:       data,
		Timeout:    timeout,
		RetryCount: maxRetries,
	}
	return c.Do(req)
}

// GetContent fetches web content and returns it as a string with error handling
func (c *HTTPClient) GetContent(url string, timeout time.Duration) (string, error) {
	resp, err := c.Get(url, timeout, 3)
	if err != nil {
		return "", fmt.Errorf("fetch failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		return "", &HTTPError{Code: resp.StatusCode, Message: string(resp.Body)}
	}

	content := strings.TrimSpace(string(resp.Body))
	if len(content) > 10000 {
		content = content[:9997] + "..."
	}

	return content, nil
}
