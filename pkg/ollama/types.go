// Package ollama provides the client for communicating with Ollama API.
package ollama

import (
	"bufio"
	"time"
)

// ToolParam represents a single parameter in a tool definition.
type ToolParam struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// ToolSchema defines the schema for a tool's parameters.
type ToolSchema struct {
	Type       string               `json:"type"`
	Properties map[string]ToolParam `json:"properties"`
	Required   []string             `json:"required,omitempty"`
}

// ToolFunction defines a function/tool that can be called by the LLM.
type ToolFunction struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Parameters  ToolSchema `json:"parameters"`
}

// ToolDef is the complete tool definition sent to Ollama.
type ToolDef struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ChatMessage represents a single message in a chat conversation.
type ChatMessage struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ToolCall represents a tool call from the model.
type ToolCall struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
	ID        string                 `json:"id"`
}

// ChatRequest is the payload for a chat request to Ollama.
type ChatRequest struct {
	Model    string         `json:"model"`
	Messages []ChatMessage  `json:"messages"`
	Stream   bool           `json:"stream"`
	Options  map[string]any `json:"options,omitempty"`
	Tools    []ToolDef      `json:"tools,omitempty"`
}

// StreamResponse is a single line response from a streaming chat.
type StreamResponse struct {
	Model              string      `json:"model"`
	CreatedAt          int64       `json:"created_at"`
	Message            ChatMessage `json:"message"`
	Done               bool        `json:"done"`
	UsedContext        int         `json:"used_context"`
	TotalDuration      int64       `json:"total_duration,omitempty"`
	LoadDuration       int64       `json:"load_duration,omitempty"`
	PromptEvalCount    int         `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64       `json:"prompt_eval_duration,omitempty"`
	EvalCount          int         `json:"eval_count,omitempty"`
	EvalDuration       int64       `json:"eval_duration,omitempty"`
}

// Response is the full response from a non-streaming chat.
type Response struct {
	Model              string      `json:"model"`
	CreatedAt          int64       `json:"created_at"`
	Message            ChatMessage `json:"message"`
	Done               bool        `json:"done"`
	UsedContext        int         `json:"used_context"`
	TotalDuration      int64       `json:"total_duration,omitempty"`
	LoadDuration       int64       `json:"load_duration,omitempty"`
	PromptEvalCount    int         `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64       `json:"prompt_eval_duration,omitempty"`
	EvalCount          int         `json:"eval_count,omitempty"`
	EvalDuration       int64       `json:"eval_duration,omitempty"`
}

// OllamaTagsResponse is the response from /api/tags endpoint.
type OllamaTagsResponse struct {
	Models []struct {
		Name       string `json:"name"`
		Size       int64  `json:"size"`
		Digest     string `json:"digest"`
		ModifiedAt int64  `json:"modified_at"`
		SizeSha256 string `json:"size_sha256,omitempty"`
	} `json:"models"`
}

// ChatOptions holds options for chat requests.
type ChatOptions struct {
	Model        string
	Stream       bool
	SystemPrompt string
	Messages     []ChatMessage
	Tools        []ToolDef
	CtxLen       int
	Timeout      time.Duration
}

// ChatResponse is the complete response from a streaming chat.
type ChatResponse struct {
	Message   ChatMessage
	UsedCtx   int
	Done      bool
	Error     error
	RawStream *bufio.Reader // For reading raw stream if needed
}

// ModelInfo contains information about an Ollama model.
type ModelInfo struct {
	Name          string `json:"name"`
	ContextLength int    `json:"context_length"`
}
