// Package ollama provides the client for communicating with Ollama API.
package ollama

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// OllamaClient communicates with the Ollama REST API for model listing,
// context-length detection, and streaming chat completions.
type OllamaClient struct {
	baseURL  string
	client   *http.Client
	ctxCache map[string]int // cached context lengths per model
	cacheMu  sync.RWMutex   // protects ctxCache from concurrent access
	timeout  time.Duration  // HTTP request timeout
}

// ClientConfig holds configuration for OllamaClient.
type ClientConfig struct {
	BaseURL     string
	HTTPTimeout time.Duration
}

// DefaultClientConfig returns a default client configuration.
func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		BaseURL:     "http://localhost:11434",
		HTTPTimeout: 300 * time.Second,
	}
}

// NewOllamaClient creates a client pointing at the given Ollama API base URL.
func NewOllamaClient(baseURL string) *OllamaClient {
	return &OllamaClient{
		baseURL:  strings.TrimRight(baseURL, "/"),
		client:   &http.Client{Timeout: 300 * time.Second},
		ctxCache: make(map[string]int),
		timeout:  300 * time.Second,
	}
}

// NewOllamaClientWithConfig creates a client with custom configuration.
func NewOllamaClientWithConfig(cfg ClientConfig) *OllamaClient {
	return &OllamaClient{
		baseURL:  strings.TrimRight(cfg.BaseURL, "/"),
		client:   &http.Client{Timeout: cfg.HTTPTimeout},
		ctxCache: make(map[string]int),
		timeout:  cfg.HTTPTimeout,
	}
}

// setContextCache sets a context length value in the cache (write lock required).
func (c *OllamaClient) setContextCache(model string, length int) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	c.ctxCache[model] = length
}

// deleteContextCache removes a model from the cache (write lock required).
func (c *OllamaClient) deleteContextCache(model string) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	delete(c.ctxCache, model)
}

// getContextCache retrieves a cached context length (read lock required).
// Returns the value and ok flag.
func (c *OllamaClient) getContextCache(model string) (int, bool) {
	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()
	val, ok := c.ctxCache[model]
	return val, ok
}

// getAndDeleteContextCache atomically reads and deletes a cached context length.
// Returns the value and ok flag. This is useful to avoid race conditions between
// checking for cache presence and reading the value.
func (c *OllamaClient) getAndDeleteContextCache(model string) (int, bool) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	val, ok := c.ctxCache[model]
	if ok {
		delete(c.ctxCache, model)
	}
	return val, ok
}

// ListModels returns the names of all models available in Ollama, or nil on error.
func (c *OllamaClient) ListModels() []string {
	resp, err := c.client.Get(c.baseURL + "/api/tags")
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	var data OllamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil
	}
	models := make([]string, len(data.Models))
	for i, m := range data.Models {
		models[i] = m.Name
	}
	return models
}

// GetModelContextLength queries the Ollama API for a model's context length.
// Returns 0 if the info can't be retrieved or is not available.
func (c *OllamaClient) GetModelContextLength(model string) int {
	// Check cache first
	if cached, ok := c.getContextCache(model); ok {
		return cached
	}

	// Fetch from API
	length := c.fetchContextLength(model)

	// Cache the result
	c.setContextCache(model, length)
	return length
}

// fetchContextLength queries the Ollama API for a model's context length without caching.
func (c *OllamaClient) fetchContextLength(model string) int {
	payload, _ := json.Marshal(map[string]string{"name": model})
	req, err := http.NewRequest("POST", c.baseURL+"/api/show", strings.NewReader(string(payload)))
	if err != nil {
		return 0
	}
	req.Header.Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := c.client.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	var data struct {
		ModelInfo map[string]any `json:"model_info"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0
	}

	// Look for context_length in model_info (key varies by architecture,
	// e.g. "llama.context_length", "qwen2.context_length", etc.)
	for k, v := range data.ModelInfo {
		if strings.HasSuffix(k, ".context_length") {
			switch n := v.(type) {
			case float64:
				return int(n)
			case int64:
				return int(n)
			}
		}
	}
	return 0
}

// ListModelsWithContext returns models with their context lengths.
func (c *OllamaClient) ListModelsWithContext() ([]ModelInfo, error) {
	models := c.ListModels()
	if len(models) == 0 {
		return nil, fmt.Errorf("no models found")
	}

	result := make([]ModelInfo, len(models))
	for i, name := range models {
		result[i] = ModelInfo{
			Name:          name,
			ContextLength: c.GetModelContextLength(name),
		}
	}

	return result, nil
}

// Chat streams messages to Ollama and returns a channel of responses.
func (c *OllamaClient) Chat(ctx context.Context, opts ChatOptions) (<-chan ChatResponse, error) {
	if len(opts.Messages) == 0 {
		return nil, fmt.Errorf("messages is required")
	}

	payload := ChatRequest{
		Model:    opts.Model,
		Messages: opts.Messages,
		Stream:   true,
		Options: map[string]any{
			"num_ctx": opts.CtxLen,
		},
	}

	if len(opts.Tools) > 0 {
		payload.Tools = opts.Tools
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/chat", strings.NewReader(string(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	ch := make(chan ChatResponse, 100)

	go func() {
		defer resp.Body.Close()
		scanner := bufio.NewScanner(resp.Body)

		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
				var line StreamResponse
				if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
					ch <- ChatResponse{
						Error: fmt.Errorf("failed to parse response: %w", err),
					}
					close(ch)
					return
				}

				ch <- ChatResponse{
					Message:   line.Message,
					UsedCtx:   line.UsedContext,
					Done:      line.Done,
					Error:     nil,
					RawStream: bufio.NewReader(resp.Body),
				}

				if line.Done {
					break
				}
			}
		}

		if err := scanner.Err(); err != nil {
			select {
			case ch <- ChatResponse{Error: fmt.Errorf("stream error: %w", err)}:
			default:
			}
		}

		close(ch)
	}()

	return ch, nil
}

// ChatNonStreaming sends a single message to Ollama and waits for the complete response.
func (c *OllamaClient) ChatNonStreaming(ctx context.Context, opts ChatOptions) (*ChatMessage, int, error) {
	if len(opts.Messages) == 0 {
		return nil, 0, fmt.Errorf("messages is required")
	}

	payload := ChatRequest{
		Model:    opts.Model,
		Messages: opts.Messages,
		Stream:   false,
		Options: map[string]any{
			"num_ctx": opts.CtxLen,
		},
	}

	if len(opts.Tools) > 0 {
		payload.Tools = opts.Tools
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/chat", strings.NewReader(string(data)))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, 0, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, 0, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result.Message, result.UsedContext, nil
}
