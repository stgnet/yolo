// Package ollamaclient provides communication with the Ollama API for model listing,
// context-length detection, and streaming chat completions.
package ollamaclient

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ─── Ollama Tool Definitions ────────────────────────────────────────

// ToolParam defines a parameter for a tool function.
type ToolParam struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// ToolSchema defines the schema for tool parameters.
type ToolSchema struct {
	Type       string               `json:"type"`
	Properties map[string]ToolParam `json:"properties"`
	Required   []string             `json:"required,omitempty"`
}

// ToolFunction defines a function that can be called by the model.
type ToolFunction struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Parameters  ToolSchema `json:"parameters"`
}

// ToolDef is the complete tool definition for Ollama.
type ToolDef struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// toolDef creates a new ToolDef with the given name, description, and parameters.
func ToolDefWith(name, desc string, props map[string]ToolParam, required []string) ToolDef {
	return ToolDef{
		Type: "function",
		Function: ToolFunction{
			Name:        name,
			Description: desc,
			Parameters: ToolSchema{
				Type:       "object",
				Properties: props,
				Required:   required,
			},
		},
	}
}

// ─── Ollama Client ──────────────────────────────────────────────────

// OllamaClient communicates with the Ollama REST API for model listing,
// context-length detection, and streaming chat completions.
type OllamaClient struct {
	baseURL  string
	client   *http.Client
	ctxCache map[string]int // cached context lengths per model
	cacheMu  sync.RWMutex   // protects ctxCache from concurrent access
}

// NewOllamaClient creates a client pointing at the given Ollama API base URL.
func NewOllamaClient(baseURL string) *OllamaClient {
	return &OllamaClient{
		baseURL:  strings.TrimRight(baseURL, "/"),
		client:   &http.Client{Timeout: 300 * time.Second},
		ctxCache: make(map[string]int),
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

	var data struct {
		Models []struct{ Name string } `json:"models"`
	}
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
// Returns 0 if the info can't be retrieved.
func (c *OllamaClient) GetModelContextLength(model string) int {
	payload, _ := json.Marshal(map[string]string{"name": model})
	resp, err := c.client.Post(c.baseURL+"/api/show", "application/json", strings.NewReader(string(payload)))
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

	for k, v := range data.ModelInfo {
		if strings.HasSuffix(k, ".context_length") {
			switch n := v.(type) {
			case float64:
				return int(n)
			}
		}
	}
	return 0
}

// ─── Chat Types ────────────────────────────────────────────────────

// ChatMessage is a single message in an Ollama chat conversation.
type ChatMessage struct {
	Role      string     `json:"role"` // "system", "user", "assistant", or "tool"
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ToolCall is a tool invocation returned by the model.
type ToolCall struct {
	ID       string       `json:"id,omitempty"`
	Function ToolCallFunc `json:"function"`
}

// ToolCallFunc carries the name and raw JSON arguments of a tool call.
type ToolCallFunc struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ChatRequest is the JSON body sent to POST /api/chat.
type ChatRequest struct {
	Model    string         `json:"model"`
	Messages []ChatMessage  `json:"messages"`
	Stream   bool           `json:"stream"`
	Tools    []ToolDef      `json:"tools,omitempty"`
	Options  map[string]any `json:"options,omitempty"`
}

// ParsedToolCall is a tool name + arguments ready for ToolExecutor.Execute.
type ParsedToolCall struct {
	Name string
	Args map[string]any
}

// ChatResult is the aggregated result of a complete streaming chat call.
type ChatResult struct {
	DisplayText string           // text shown to the user (may include thinking)
	ContentText string           // raw content from the model
	ToolCalls   []ParsedToolCall // tool calls extracted from the response
}

// StreamMessage is a single message in the streaming /api/chat response.
type StreamMessage struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// StreamResponse is a single JSON object in the streaming /api/chat response.
type StreamResponse struct {
	Message StreamMessage `json:"message"`
	ToolCall bool          `json:"tool_called,omitempty"`
	Done      bool          `json:"done,omitempty"`
	Model     string        `json:"model,omitempty"`
}

// ChatOptions holds options for chat requests.
type ChatOptions struct {
	Temperature float64
	TopP        float64
	NumCtx      int
}

// StreamOutputFn is a callback function for streaming output to a specific destination.
type StreamOutputFn func(string)

// Chat sends a chat request to Ollama and returns the complete result.
// If streamOutputFn is provided, it's called with each line of output as it arrives.
func (c *OllamaClient) Chat(ctx context.Context, model string, messages []ChatMessage, tools []ToolDef, streamOutput StreamOutputFn) (*ChatResult, error) {
	// Build the request
	req := ChatRequest{
		Model:    model,
		Messages: messages,
		Stream:   true,
		Tools:    tools,
	}

	if len(tools) > 0 {
		req.Tools = tools
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal chat request: %w", err)
	}

	resp, err := c.client.Post(c.baseURL+"/api/chat", "application/json", strings.NewReader(string(payload)))
	if err != nil {
		return nil, fmt.Errorf("chat request failed: %w", streamOutput)
	}
	defer resp.Body.Close()

	result := &ChatResult{}
	var fullTextBuilder strings.Builder
	var displayTextBuilder strings.Builder
	var lastDisplay string

	reader := bufio.NewReader(resp.Body)
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err.Error() == "EOF" || len(fullTextBuilder.String()) > 0 {
				break
			}
			return nil, fmt.Errorf("read response: %w", err)
		}

		var streamResp StreamResponse
		if err := json.Unmarshal(line, &streamResp); err != nil {
			continue
		}

		if streamResp.Done {
			break
		}

		// Append content to builders
		fullTextBuilder.WriteString(streamResp.Message.Content)

		if streamOutput != nil {
			streamOutput(streamResp.Message.Content)
		} else if lastDisplay == "" || lastDisplay != streamResp.Message.Content {
			displayTextBuilder.WriteString(streamResp.Message.Content)
			lastDisplay = streamResp.Message.Content
		}

		// Handle tool calls in streaming responses
		for _, tc := range streamResp.Message.ToolCalls {
			argsMap := make(map[string]any)
			if len(tc.Function.Arguments) > 0 {
				if err := json.Unmarshal(tc.Function.Arguments, &argsMap); err != nil {
					continue
				}
			}

			result.ToolCalls = append(result.ToolCalls, ParsedToolCall{
				Name:   tc.Function.Name,
				Args:   argsMap,
			})
		}
	}

	result.ContentText = fullTextBuilder.String()
	result.DisplayText = displayTextBuilder.String()
	return result, nil
}
