package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ─── Ollama Tool Definitions ─────────────────────────────────────────

type ToolParam struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type ToolSchema struct {
	Type       string               `json:"type"`
	Properties map[string]ToolParam `json:"properties"`
	Required   []string             `json:"required,omitempty"`
}

type ToolFunction struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Parameters  ToolSchema `json:"parameters"`
}

type ToolDef struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

func toolDef(name, desc string, props map[string]ToolParam, required []string) ToolDef {
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

// ─── Ollama Client ────────────────────────────────────────────────────

type OllamaClient struct {
	baseURL string
	client  *http.Client
}

func NewOllamaClient(baseURL string) *OllamaClient {
	return &OllamaClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 300 * time.Second},
	}
}

type OllamaModel struct {
	Name string `json:"name"`
}

type OllamaTagsResponse struct {
	Models []OllamaModel `json:"models"`
}

func (c *OllamaClient) ListModels() []string {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(c.baseURL + "/api/tags")
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

// Chat message types
type ChatMessage struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

type ToolCall struct {
	ID       string       `json:"id,omitempty"`
	Function ToolCallFunc `json:"function"`
}

type ToolCallFunc struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type ChatRequest struct {
	Model    string         `json:"model"`
	Messages []ChatMessage  `json:"messages"`
	Stream   bool           `json:"stream"`
	Tools    []ToolDef      `json:"tools,omitempty"`
	Options  map[string]any `json:"options,omitempty"`
}

type StreamResponse struct {
	Message StreamMessage `json:"message"`
	Done    bool          `json:"done"`
}

type StreamMessage struct {
	Thinking  string     `json:"thinking,omitempty"`
	Content   string     `json:"content"`
	ToolCalls []StreamTC `json:"tool_calls,omitempty"`
}

type StreamTC struct {
	Function StreamTCFunc `json:"function"`
}

type StreamTCFunc struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// ChatResult holds the result of a chat call
type ChatResult struct {
	DisplayText string
	ContentText string
	ToolCalls   []ParsedToolCall
}

type ParsedToolCall struct {
	Name string
	Args map[string]any
}

func (c *OllamaClient) Chat(ctx context.Context, model string, messages []ChatMessage, tools []ToolDef) (*ChatResult, error) {
	payload := ChatRequest{
		Model:    model,
		Messages: messages,
		Stream:   true,
		Options:  map[string]any{"num_ctx": 8192},
	}
	if len(tools) > 0 {
		payload.Tools = tools
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/chat", strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("request error: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request error: %w", err)
	}
	defer resp.Body.Close()

	spinner := NewSpinner("yolo> ", Blue)
	spinner.Start()

	// outPrint writes to the output region (inline, no input redraw per token)
	outPrint := func(s string) {
		if globalUI != nil {
			globalUI.OutputPrintInline(s)
		} else {
			rawWrite(s)
		}
	}

	var thinkingParts, contentParts []string
	var toolCalls []ParsedToolCall
	inThinking := false
	gotFirstOutput := false

	scanner := bufio.NewScanner(resp.Body)
	// Increase scanner buffer for large responses
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var obj StreamResponse
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			continue
		}

		msg := obj.Message
		thinking := msg.Thinking
		content := msg.Content
		tcList := msg.ToolCalls

		// On first real output, stop the spinner
		if !gotFirstOutput && (thinking != "" || content != "" || len(tcList) > 0) {
			gotFirstOutput = true
			spinner.Stop()
			outPrint(fmt.Sprintf("%s%syolo>%s ", Blue, Bold, Reset))
		}

		// Handle thinking tokens
		if thinking != "" {
			if !inThinking {
				outPrint(fmt.Sprintf("%s[thinking] ", Gray))
				inThinking = true
			}
			outPrint(thinking)
			thinkingParts = append(thinkingParts, thinking)
		}

		// Handle content tokens
		if content != "" {
			if inThinking {
				outPrint(fmt.Sprintf("%s\n", Reset))
				inThinking = false
			}
			outPrint(content)
			contentParts = append(contentParts, content)
		}

		// Collect native tool calls
		for _, tc := range tcList {
			if tc.Function.Name != "" {
				toolCalls = append(toolCalls, ParsedToolCall{
					Name: tc.Function.Name,
					Args: tc.Function.Arguments,
				})
			}
		}

		if obj.Done {
			break
		}
	}

	// Clean up spinner if model returned nothing
	if !gotFirstOutput {
		spinner.Stop()
		outPrint(fmt.Sprintf("%s%syolo>%s ", Blue, Bold, Reset))
	}

	if inThinking {
		outPrint(Reset)
	}
	outPrint("\n")
	// Redraw input line after streaming output is done
	if globalUI != nil {
		globalUI.OutputFinishLine()
	}

	contentText := strings.Join(contentParts, "")
	thinkingText := strings.Join(thinkingParts, "")
	displayText := contentText
	if displayText == "" {
		displayText = thinkingText
	}

	return &ChatResult{
		DisplayText: displayText,
		ContentText: contentText,
		ToolCalls:   toolCalls,
	}, nil
}
