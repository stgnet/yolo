package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
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

// OllamaClient communicates with the Ollama REST API for model listing,
// context-length detection, and streaming chat completions.
type OllamaClient struct {
	baseURL  string
	client   *http.Client
	ctxCache map[string]int // cached context lengths per model
}

// NewOllamaClient creates a client pointing at the given Ollama API base URL.
func NewOllamaClient(baseURL string) *OllamaClient {
	return &OllamaClient{
		baseURL:  strings.TrimRight(baseURL, "/"),
		client:   &http.Client{Timeout: 300 * time.Second},
		ctxCache: make(map[string]int),
	}
}

type OllamaModel struct {
	Name string `json:"name"`
}

type OllamaTagsResponse struct {
	Models []OllamaModel `json:"models"`
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

	// Look for context_length in model_info (key varies by architecture,
	// e.g. "llama.context_length", "qwen2.context_length", etc.)
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

// ─── Chat types ──────────────────────────────────────────────────────

// ChatMessage is a single message in an Ollama chat conversation.
type ChatMessage struct {
	Role       string     `json:"role"` // "system", "user", "assistant", or "tool"
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
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

// StreamResponse is a single JSON object in the streaming /api/chat response.
type StreamResponse struct {
	Message StreamMessage `json:"message"`
	Done    bool          `json:"done"`
}

// StreamMessage holds the incremental content from one streamed chunk.
type StreamMessage struct {
	Thinking  string     `json:"thinking,omitempty"`
	Content   string     `json:"content"`
	ToolCalls []StreamTC `json:"tool_calls,omitempty"`
}

// StreamTC is a tool call within a streamed response chunk.
type StreamTC struct {
	Function StreamTCFunc `json:"function"`
}

// StreamTCFunc carries the parsed name and arguments of a streamed tool call.
type StreamTCFunc struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// ChatResult is the aggregated result of a complete streaming chat call.
type ChatResult struct {
	DisplayText string           // text shown to the user (may include thinking)
	ContentText string           // raw content from the model
	ToolCalls   []ParsedToolCall // tool calls extracted from the response
}

// ParsedToolCall is a tool name + arguments ready for ToolExecutor.Execute.
type ParsedToolCall struct {
	Name string
	Args map[string]any
}

// deduplicateToolCalls removes duplicate tool calls that have identical name
// and arguments.  This prevents the same write_file / edit_file / etc. from
// being executed twice when a model (or Ollama backend) sends the same tool
// call in more than one streaming chunk.
func deduplicateToolCalls(calls []ParsedToolCall) []ParsedToolCall {
	if len(calls) <= 1 {
		return calls
	}
	seen := make(map[string]bool, len(calls))
	out := make([]ParsedToolCall, 0, len(calls))
	for _, tc := range calls {
		argsJSON, _ := json.Marshal(tc.Args)
		key := tc.Name + "|" + string(argsJSON)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, tc)
	}
	return out
}

// Chat sends a streaming chat request to Ollama and returns the accumulated
// result. Display text is printed to the terminal as it arrives. The ctx
// parameter allows the caller to cancel the request (e.g. on Ctrl-C).
// If outFn is non-nil, it receives output text instead of the default globalUI.
func (c *OllamaClient) Chat(ctx context.Context, model string, messages []ChatMessage, tools []ToolDef, outFn func(string)) (*ChatResult, error) {
	numCtx := DefaultNumCtx
	if NumCtxOverride != "" {
		if n, err := strconv.Atoi(NumCtxOverride); err == nil && n > 0 {
			numCtx = n
		}
	} else if cached, ok := c.ctxCache[model]; ok {
		numCtx = cached
	} else if detected := c.GetModelContextLength(model); detected > 0 {
		c.ctxCache[model] = detected
		numCtx = detected
	}

	payload := ChatRequest{
		Model:    model,
		Messages: messages,
		Stream:   true,
		Options:  map[string]any{"num_ctx": numCtx},
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

	// outPrint writes to the output region (inline, no input redraw per token)
	outPrint := func(s string) {
		if outFn != nil {
			outFn(s)
		} else if globalUI != nil {
			globalUI.OutputPrintInline(s)
		} else if bufferUI != nil {
			bufferUI.Write(s)
		} else {
			rawWrite(s)
		}
	}

	var thinkingParts, contentParts []string
	var toolCalls []ParsedToolCall
	inThinking := false
	inToolActivity := false
	// pendingBuf accumulates text that might contain a partial
	// "[tool activity]" or "[/tool activity]" marker split across tokens.
	var pendingBuf string

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

		// Strip carriage returns from LLM output to prevent line overwrites.
		// \r\n becomes \n (rawWrite will re-add \r\n for raw mode);
		// standalone \r is simply removed.
		// Tab expansion is handled by OutputPrintInline (for the UI path)
		// so the cursor tracker and terminal agree on column positions.
		thinking = strings.ReplaceAll(thinking, "\r", "")
		content = strings.ReplaceAll(content, "\r", "")

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
			contentParts = append(contentParts, content)

			// Buffer content to detect tool activity markers that may span
			// token boundaries. Recognizes both [tool activity] blocks and
			// <tool_call>/<function=...> XML-style tool calls.
			pendingBuf += content
			for pendingBuf != "" {
				if !inToolActivity {
					// Check for any of the tool activity open markers
					type markerMatch struct {
						idx    int
						marker string
					}
					var best *markerMatch
					for _, openMarker := range []string{"[tool activity]", "<tool_call>", "<function="} {
						idx := strings.Index(pendingBuf, openMarker)
						if idx >= 0 && (best == nil || idx < best.idx) {
							best = &markerMatch{idx: idx, marker: openMarker}
						}
					}
					if best != nil {
						// Flush text before marker in default color
						if best.idx > 0 {
							outPrint(pendingBuf[:best.idx])
						}
						// Print marker in yellow and stay in yellow
						outPrint(Yellow + best.marker)
						pendingBuf = pendingBuf[best.idx+len(best.marker):]
						inToolActivity = true
						continue
					}
					// Check if the tail of pendingBuf could be a partial marker
					partial := false
					for _, openMarker := range []string{"[tool activity]", "<tool_call>", "<function="} {
						for i := 1; i < len(openMarker) && i <= len(pendingBuf); i++ {
							if strings.HasSuffix(pendingBuf, openMarker[:i]) {
								// Flush everything except the potential partial match
								safe := pendingBuf[:len(pendingBuf)-i]
								if safe != "" {
									outPrint(safe)
								}
								pendingBuf = pendingBuf[len(pendingBuf)-i:]
								partial = true
								break
							}
						}
						if partial {
							break
						}
					}
					if !partial {
						outPrint(pendingBuf)
						pendingBuf = ""
					}
					break // wait for more tokens
				} else {
					// Check for any of the tool activity close markers
					type markerMatch struct {
						idx    int
						marker string
					}
					var best *markerMatch
					for _, closeMarker := range []string{"[/tool activity]", "</tool_call>", "</function>"} {
						idx := strings.Index(pendingBuf, closeMarker)
						if idx >= 0 && (best == nil || idx < best.idx) {
							best = &markerMatch{idx: idx, marker: closeMarker}
						}
					}
					if best != nil {
						// Print text before closing marker in yellow
						if best.idx > 0 {
							outPrint(pendingBuf[:best.idx])
						}
						// Print closing marker in yellow, then reset
						outPrint(best.marker + Reset)
						pendingBuf = pendingBuf[best.idx+len(best.marker):]
						inToolActivity = false
						continue
					}
					// Check for partial closing marker at end
					partial := false
					for _, closeMarker := range []string{"[/tool activity]", "</tool_call>", "</function>"} {
						for i := 1; i < len(closeMarker) && i <= len(pendingBuf); i++ {
							if strings.HasSuffix(pendingBuf, closeMarker[:i]) {
								safe := pendingBuf[:len(pendingBuf)-i]
								if safe != "" {
									outPrint(safe)
								}
								pendingBuf = pendingBuf[len(pendingBuf)-i:]
								partial = true
								break
							}
						}
						if partial {
							break
						}
					}
					if !partial {
						outPrint(pendingBuf)
						pendingBuf = ""
					}
					break // wait for more tokens
				}
			}
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

	if inThinking {
		outPrint(Reset)
	}
	// Flush any remaining pending buffer from tool activity detection
	if pendingBuf != "" {
		outPrint(pendingBuf)
		pendingBuf = ""
	}
	if inToolActivity {
		outPrint(Reset)
	}
	outPrint("\n")
	// Redraw input line after streaming output is done (only for main agent output)
	if outFn == nil && globalUI != nil {
		globalUI.OutputFinishLine()
	}

	contentText := strings.Join(contentParts, "")
	thinkingText := strings.Join(thinkingParts, "")
	displayText := contentText
	if displayText == "" {
		displayText = thinkingText
	}

	// Deduplicate tool calls that may arrive in multiple streaming chunks.
	// Some models/backends send the same tool call in both an intermediate
	// chunk and the done chunk, causing the same operation to execute twice.
	toolCalls = deduplicateToolCalls(toolCalls)

	return &ChatResult{
		DisplayText: displayText,
		ContentText: contentText,
		ToolCalls:   toolCalls,
	}, nil
}
