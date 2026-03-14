package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// ─── Main Agent ───────────────────────────────────────────────────────

// YoloAgent is the central orchestrator. It reads user input, sends messages
// to the LLM via OllamaClient, dispatches tool calls through ToolExecutor,
// and persists conversation state with HistoryManager. When no user input is
// pending it immediately enters autonomous thinking.
// handoffResult holds the outcome of a background tool execution that was
// forked from the main agent when a user message arrived mid-tool-loop.
type handoffResult struct {
	ID      int              // monotonic handoff ID
	Results []toolExecResult // tool name + output for each executed tool
	Done    chan struct{}    // closed when the background executor finishes
}

// toolExecResult is a single tool call's name, arguments, and output string.
type toolExecResult struct {
	Name   string
	Args   map[string]any
	Result string
}

type YoloAgent struct {
	baseDir         string             // working directory; all file operations are relative to this
	scriptPath      string             // path to the running binary (used for self-restart)
	ollama          *OllamaClient      // LLM communication
	history         *HistoryManager    // persistent conversation and evolution log
	config          *YoloConfig        // persistent configuration (model, etc.)
	tools           *ToolExecutor      // tool dispatcher
	inputMgr        *InputManager      // async terminal input
	running         bool               // false signals the main loop to exit
	busy            bool               // true while the agent is processing a chat round
	subagentCounter int                // monotonic ID for spawned sub-agents
	handoffCounter  int                // monotonic ID for background handoffs
	pendingHandoffs []*handoffResult   // in-flight background tool executions
	mu              sync.Mutex         // protects busy, cancelChat, subagentCounter, handoffCounter, pendingHandoffs
	cancelChat      context.CancelFunc // cancels the in-flight Chat HTTP request
}

// NewYoloAgent creates an agent rooted in the current working directory
// and connects to the Ollama instance at OllamaURL.
func NewYoloAgent() *YoloAgent {
	baseDir, _ := os.Getwd()
	execPath, _ := os.Executable()

	// Clear stale subagent results from any prior run so that
	// listSubagents/readSubagentResult don't return leftover data and the
	// monotonic ID counter (starting at 0) doesn't collide with old files.
	if files, err := filepath.Glob(filepath.Join(SubagentDir, "agent_*.json")); err == nil {
		for _, f := range files {
			os.Remove(f)
		}
	}

	a := &YoloAgent{
		baseDir:    baseDir,
		scriptPath: execPath,
		ollama:     NewOllamaClient(OllamaURL),
		history:    NewHistoryManager(YoloDir),
		config:     NewYoloConfig(YoloDir),
		running:    true,
	}
	a.tools = NewToolExecutor(baseDir, a)
	return a
}

// getSystemPrompt loads SYSTEM_PROMPT.md and interpolates runtime values
// (working directory, model name, timestamp, etc.).
func (a *YoloAgent) getSystemPrompt() string {
	// Load the system prompt template from file
	systemPromptPath := filepath.Join(a.baseDir, "SYSTEM_PROMPT.md")
	templateContent, err := os.ReadFile(systemPromptPath)
	if err != nil {
		cprint(Red, fmt.Sprintf("  Error: Could not read SYSTEM_PROMPT.md: %v\n", err))
		cprint(Red, "  SYSTEM_PROMPT.md is required. Please ensure it exists in the working directory.\n")
		os.Exit(1)
	}

	// Load knowledge base if it exists
	var kbSection string
	kbPath := filepath.Join(a.baseDir, ".yolo", "knowledge.md")
	if content, err := os.ReadFile(kbPath); err == nil {
		kbSection = "\n## Knowledge Base\n" + string(content)
	}

	// Replace template variables in the system prompt
	prompt := string(templateContent)
	prompt = strings.ReplaceAll(prompt, "{baseDir}", a.baseDir)
	prompt = strings.ReplaceAll(prompt, "{scriptPath}", a.scriptPath)
	if a.config != nil {
		prompt = strings.ReplaceAll(prompt, "{model}", a.config.GetModel())
	} else {
		prompt = strings.ReplaceAll(prompt, "{model}", "unknown")
	}
	prompt = strings.ReplaceAll(prompt, "{timestamp}", time.Now().Format(time.RFC3339))
	prompt = strings.ReplaceAll(prompt, "{knowledgeBase}", kbSection)

	// Inject pending todos so the agent is aware of outstanding work
	todoContext := todoList.RenderPendingContext()
	prompt += todoContext

	return prompt
}

// restart rebuilds the binary from source and replaces the running process
// via syscall.Exec. It does not return on success.
func (a *YoloAgent) restart() {
	a.tools.restart(make(map[string]any))
}

// ── Setup ──

// setupFirstRun runs on first launch (no history file). It connects to Ollama,
// lets the user pick a model, and records the choice.
func (a *YoloAgent) setupFirstRun() {
	cprint(Cyan+Bold, "\n  YOLO - Your Own Living Operator")
	cprint(Gray, "  A self-evolving AI agent for software development")
	cprint(Gray, fmt.Sprintf("  Working directory: %s", a.baseDir))
	fmt.Println()

	cprint(Yellow, "  Connecting to Ollama...")
	models := a.ollama.ListModels()
	if len(models) == 0 {
		cprint(Red, "  Error: Cannot reach Ollama or no models installed.")
		cprint(Red, "  Make sure Ollama is running: ollama serve")
		os.Exit(1)
	}

	cprint(Green, fmt.Sprintf("  Found %d model(s):", len(models)))
	for i, m := range models {
		fmt.Printf("    %s%2d%s. %s\n", Bold, i+1, Reset, m)
	}
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("  %sSelect model (1-%d): %s", Green, len(models), Reset)
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)
		var idx int
		if _, err := fmt.Sscanf(choice, "%d", &idx); err != nil || idx < 1 || idx > len(models) {
			fmt.Println("  Invalid selection, try again.")
			continue
		}
		a.config.SetModel(models[idx-1])
		if err := a.history.Save(); err != nil {
			cprint(Red, fmt.Sprintf("  Warning: could not save history: %v\n", err))
		}
		cprint(Green, fmt.Sprintf("\n  Model: %s%s%s", Bold, models[idx-1], Reset))
		break
	}
	a.showHelpHint()
}

// resumeSession loads history for session resumption (silent, no output)
// displaySessionResumption shows the resuming message with formatting
func (a *YoloAgent) displaySessionResumption() {
	cprint(Cyan+Bold, "\n  YOLO - Your Own Living Operator")
	cprint(Green, fmt.Sprintf("  Resuming — model: %s%s%s", Bold, a.config.GetModel(), Reset))
	n := len(a.history.Data.Messages)
	cprint(Gray, fmt.Sprintf("  History: %d messages loaded", n))

	// Find and display the last assistant message
	var lastAssistantMsg *HistoryMessage
	for i := len(a.history.Data.Messages) - 1; i >= 0; i-- {
		if a.history.Data.Messages[i].Role == "assistant" && a.history.Data.Messages[i].Content != "" {
			lastAssistantMsg = &a.history.Data.Messages[i]
			break
		}
	}

	if lastAssistantMsg != nil {
		cprint(Yellow+Bold, "\n  🔄 RESUMING FROM LAST ACTIVITY:")
		cprint(Gray, fmt.Sprintf("    Role: %s", lastAssistantMsg.Role))

		// Show full content if it's a tool result or short message, otherwise truncate with indicator
		content := lastAssistantMsg.Content
		if len(content) > 200 {
			content = content[:200] + "..."
		}
		cprint(Gray, fmt.Sprintf("    Content: %s", content))

		// Check if it was a tool call from metadata
		if lastAssistantMsg.Meta != nil && lastAssistantMsg.Meta["tool_name"] != nil {
			toolName := fmt.Sprintf("%v", lastAssistantMsg.Meta["tool_name"])
			cprint(Yellow, fmt.Sprintf("    Tool: %s%s%s", Bold, toolName, Reset))
		}
		fmt.Println()
	} else {
		cprint(Yellow, "  ⚠️ No recent activity found in history")
		fmt.Println()
	}

	// Also show last few messages for context
	lastMsgs := a.history.GetLastN(3)
	if len(lastMsgs) > 0 {
		cprint(Gray, "  Recent context:")
		for _, m := range lastMsgs {
			prefix := ""
			switch m.Role {
			case "user":
				prefix = "You"
			case "assistant":
				prefix = "Agent"
			case "tool":
				prefix = "Tool"
			default:
				prefix = m.Role
			}
			content := truncateString(m.Content, 50)
			cprint(Gray, fmt.Sprintf("    [%s] %s", prefix, content))
		}
		fmt.Println()
	}
	a.showHelpHint()
}

func (a *YoloAgent) showHelpHint() {
	cprint(Gray, "  Type a message, or /help for commands.\n")
}

// enableTerminalMode switches from buffer mode to the classic split-screen terminal UI.
func (a *YoloAgent) enableTerminalMode() {
	bufferUI = nil
	globalUI = NewTerminalUI()
	globalUI.Setup()
	a.config.SetTerminalMode(true)
	cprint(Cyan, "  Terminal mode enabled (split-screen UI)")
}

// disableTerminalMode switches from the split-screen terminal UI to buffer mode.
func (a *YoloAgent) disableTerminalMode() {
	if globalUI != nil {
		globalUI.Teardown()
		globalUI = nil
	}
	bufferUI = NewBufferUI()
	a.config.SetTerminalMode(false)
	cprint(Cyan, "  Terminal mode disabled (buffer mode)")
}

// ── Chat loop ──

// chatWithAgent sends userMessage (or an autonomous prompt when userMessage is
// empty and autonomous is true) to the LLM and iterates: each response may
// contain tool calls which are executed and fed back until the model produces
// a final text-only reply.
func (a *YoloAgent) chatWithAgent(userMessage string, autonomous bool) {
	// Clear the user's input line so agent output appears cleanly
	if a.inputMgr != nil {
		a.inputMgr.ClearLine()
	}

	// Ingest any completed background handoff results so the agent has
	// full context of work that was forked earlier.
	a.ingestHandoffResults()

	if userMessage != "" {
		a.history.AddMessage("user", userMessage, nil)
	}

	if autonomous {
		a.history.AddMessage("system",
			"No new user input. You are in autonomous mode. "+
				"Continue making progress on your own — do NOT ask the user "+
				"for input or confirmation. Pick the most impactful next task "+
				"and execute it using tools. Focus on: code quality, bug fixes, "+
				"tests, self-improvement, or new features. "+
				"Act decisively. Do the work, then move to the next thing.", nil)
	}

	// Base context from persistent history
	baseMsgs := []ChatMessage{
		{Role: "system", Content: a.getSystemPrompt()},
	}
	baseMsgs = append(baseMsgs, a.history.GetContextMessages(MaxContextMessages)...)

	// In-memory messages for the current tool-calling chain
	var roundMsgs []ChatMessage
	type toolLogEntry struct {
		name   string
		args   map[string]any
		result string
	}
	var toolLog []toolLogEntry
	var finalText string

	roundNum := 0
	for {
		allMsgs := make([]ChatMessage, 0, len(baseMsgs)+len(roundMsgs))
		allMsgs = append(allMsgs, baseMsgs...)
		allMsgs = append(allMsgs, roundMsgs...)

		ctx, cancel := context.WithCancel(context.Background())
		a.mu.Lock()
		a.cancelChat = cancel
		a.mu.Unlock()

		result, err := a.ollama.Chat(ctx, a.config.GetModel(), allMsgs, ollamaTools, nil)
		cancel()
		a.mu.Lock()
		a.cancelChat = nil
		a.mu.Unlock()

		if err != nil {
			if ctx.Err() != nil {
				cprint(Yellow, "\n  Interrupted.")
				return
			}
			cprint(Red, fmt.Sprintf("\nError: %v", err))
			return
		}

		toolCalls := result.ToolCalls
		textParsed := false

		// Also check for text-based tool calls as fallback
		if len(toolCalls) == 0 {
			toolCalls = a.parseTextToolCalls(result.DisplayText)
			if len(toolCalls) > 0 {
				textParsed = true
			}
		}

		// Deduplicate: streaming or text-parsing may yield duplicate calls
		// that would cause the agent to write the same file twice.
		toolCalls = deduplicateToolCalls(toolCalls)

		if len(toolCalls) == 0 {
			// Detect hallucinated tool activity: the model emitted
			// [tool activity] markers but none matched a valid tool
			// call format.  Feed an error back so it can self-correct
			// instead of silently treating the output as final text.
			if strings.Contains(result.DisplayText, "[tool activity]") {
				cprint(Yellow, "\n  [agent produced unrecognized tool call format — sending correction]\n")
				roundMsgs = append(roundMsgs, ChatMessage{
					Role:    "assistant",
					Content: result.ContentText,
				})
				roundMsgs = append(roundMsgs, ChatMessage{
					Role: "user",
					Content: "Error: Your tool calls were not recognized. You used '[tool activity]' markers " +
						"followed by natural language descriptions instead of actual tool call syntax. " +
						"To call tools, use the proper format, for example:\n" +
						"  [tool activity] read_file(path=\"tools.go\", offset=100, limit=100)\n" +
						"  [tool activity] search_files(pattern=\"check_inbox\", path=\".\")\n" +
						"Do NOT write descriptions like '[tool activity] Reading lines 100-200'. " +
						"Use the actual tool function name with parameters. " +
						"Available tools: " + strings.Join(validTools, ", "),
				})
				continue
			}
			finalText = result.DisplayText
			break
		}

		// Build proper assistant message with tool_calls
		var nativeTCs []ToolCall
		for i, tc := range toolCalls {
			argsJSON, _ := json.Marshal(tc.Args)
			nativeTCs = append(nativeTCs, ToolCall{
				ID: fmt.Sprintf("call_%d_%d", roundNum, i),
				Function: ToolCallFunc{
					Name:      tc.Name,
					Arguments: json.RawMessage(argsJSON),
				},
			})
		}

		// When tool calls were parsed from text (not native), strip the
		// tool call markup from the assistant content so the model only
		// sees clean content alongside the native tool_calls and results.
		// Leaving both the text-based syntax and native tool_calls confuses
		// the model into thinking its tools didn't execute.
		assistantContent := result.ContentText
		if textParsed {
			assistantContent = stripTextToolCalls(assistantContent)
		}

		roundMsgs = append(roundMsgs, ChatMessage{
			Role:      "assistant",
			Content:   assistantContent,
			ToolCalls: nativeTCs,
		})

		// Execute each tool and add tool-role result.
		// Track whether the user typed something mid-tool-loop.
		// We don't interrupt execution — tools keep running normally —
		// but we consume the message so the agent sees it on
		// its next LLM round.
		userInterjected := false
		for i, call := range toolCalls {
			name := call.Name
			args := call.Args
			if args == nil {
				args = map[string]any{}
			}

			shortArgs, _ := json.Marshal(args)
			shortStr := string(shortArgs)
			if len(shortStr) > 80 {
				shortStr = shortStr[:80] + "..."
			}
			cprint(Yellow, fmt.Sprintf("  [%s] %s", name, shortStr))

			resultStr := executeWithTimeout(a.tools, name, args)

			preview := resultStr
			if len(preview) > 200 {
				preview = preview[:200] + "..."
			}
			preview = strings.ReplaceAll(preview, "\r", "")
			preview = strings.ReplaceAll(preview, "\n", " ")
			cprint(Gray, fmt.Sprintf("  => %s", preview))

			cleanResult := filterToolActivityMarkers(resultStr)
			roundMsgs = append(roundMsgs, ChatMessage{
				Role:       "tool",
				Content:    cleanResult,
				ToolCallID: fmt.Sprintf("call_%d_%d", roundNum, i),
			})
			toolLog = append(toolLog, toolLogEntry{name: name, args: args, result: cleanResult})
		}

		// After the tool batch completes, check if the user typed something
		// while tools were running.  Consume the message and inject it as
		// a real user message so the agent responds on the next LLM round.
		if len(a.inputMgr.Lines) > 0 {
			select {
			case qLine := <-a.inputMgr.Lines:
				qText := strings.TrimSpace(qLine.Text)
				qLower := strings.ToLower(qText)
				if qText != "" {
					if qLower == "exit" || qLower == "quit" {
						a.running = false
						return
					} else if strings.HasPrefix(qText, "/") {
						a.handleCommand(qText)
						if !a.running {
							return
						}
					} else {
						cprint(Cyan, "  [interjection] Delivering user message to agent")
						a.echoUserInput(qText)
						a.history.AddMessage("user", qText, nil)
						roundMsgs = append(roundMsgs, ChatMessage{
							Role:    "user",
							Content: qText,
						})
						userInterjected = true
						_ = userInterjected // reserved for future use
					}
				}
			default:
				// Channel drained between len check and receive; no-op
			}
		}

		roundNum++
	}

	// Save to persistent history (only final assistant text, not internal tracking)
	if finalText != "" {
		// Response was already streamed to the terminal by Chat(), just save to history
		a.history.AddMessage("assistant", finalText, nil)
	}
}

// parseTextToolCalls extracts tool calls from plain-text LLM output when the
// model does not use native tool_calls. It tries seven formats in order and
// returns on the first that produces results:
//
//  1. <tool_call>{"name":"...", "args":{...}}</tool_call>      (JSON)
//  2. <tool_call><function=name>...</function></tool_call>     (XML)
//  3. [tool_name] {"key":"value"}                              (Bracket)
//  4. <tool_name>{"key":"value"}</tool_name>                   (Tag)
//  5. [tool activity]\n[tool_name] => params\n[/tool activity] (Activity block)
//  6. [tool activity] tool_name(key="value")                   (Inline activity)
//  7. [tool_name](key="value", ...)                            (Markdown link)
func (a *YoloAgent) parseTextToolCalls(text string) []ParsedToolCall {
	var calls []ParsedToolCall

	// Format 1: <tool_call>{"name": ..., "args": ...}</tool_call>
	re1 := regexp.MustCompile(`(?s)<tool_call>\s*(\{.*?\})\s*</tool_call>`)
	for _, match := range re1.FindAllStringSubmatch(text, -1) {
		var obj map[string]any
		if err := json.Unmarshal([]byte(match[1]), &obj); err == nil {
			if name, ok := obj["name"].(string); ok {
				args, _ := obj["args"].(map[string]any)
				if args == nil {
					args = map[string]any{}
				}
				calls = append(calls, ParsedToolCall{Name: name, Args: args})
			}
		}
	}

	// Format 2: <tool_call><function=name><parameter=key>value</parameter>...</function></tool_call>
	if len(calls) == 0 {
		re2 := regexp.MustCompile(`(?s)<tool_call>\s*<function=(\w+)>(.*?)</function>\s*</tool_call>`)
		reParam := regexp.MustCompile(`(?s)<parameter=(\w+)>\s*(.*?)\s*</parameter>`)
		for _, match := range re2.FindAllStringSubmatch(text, -1) {
			name := match[1]
			body := match[2]
			args := map[string]any{}
			for _, pm := range reParam.FindAllStringSubmatch(body, -1) {
				args[pm[1]] = convertParamValue(pm[2])
			}
			calls = append(calls, ParsedToolCall{Name: name, Args: args})
		}
	}

	// Format 2b: bare <function=name><parameter=key>value</parameter>...</function> (no <tool_call> wrapper)
	if len(calls) == 0 {
		re2b := regexp.MustCompile(`(?s)<function=(\w+)>(.*?)</function>`)
		reParam2b := regexp.MustCompile(`(?s)<parameter=(\w+)>\s*(.*?)\s*</parameter>`)
		for _, match := range re2b.FindAllStringSubmatch(text, -1) {
			name := match[1]
			body := match[2]
			args := map[string]any{}
			for _, pm := range reParam2b.FindAllStringSubmatch(body, -1) {
				args[pm[1]] = convertParamValue(pm[2])
			}
			calls = append(calls, ParsedToolCall{Name: name, Args: args})
		}
	}

	// Format 2c: <function=name><parameter=key>value without proper closing tags
	// Some models emit <tool_call><function=name><parameter=key>\nvalue\n but never
	// close with </parameter>, </function>, or </tool_call>.
	if len(calls) == 0 {
		re2c := regexp.MustCompile(`(?s)<function=(\w+)>(.*?)(?:</function>|\z)`)
		reParamHeader := regexp.MustCompile(`<parameter=(\w+)>`)
		for _, match := range re2c.FindAllStringSubmatch(text, -1) {
			name := match[1]
			body := match[2]
			args := map[string]any{}
			// Find all parameter start positions
			paramMatches := reParamHeader.FindAllStringSubmatchIndex(body, -1)
			for i, pm := range paramMatches {
				paramName := body[pm[2]:pm[3]]
				valueStart := pm[1] // right after <parameter=name>
				var valueEnd int
				if i+1 < len(paramMatches) {
					valueEnd = paramMatches[i+1][0] // start of next <parameter=
				} else {
					valueEnd = len(body)
				}
				val := body[valueStart:valueEnd]
				// Strip optional </parameter> closing tag from value
				val = strings.TrimSuffix(strings.TrimSpace(val), "</parameter>")
				val = strings.TrimSpace(val)
				if val != "" {
					args[paramName] = convertParamValue(val)
				}
			}
			if len(args) > 0 {
				calls = append(calls, ParsedToolCall{Name: name, Args: args})
			}
		}
	}

	// Format 3: [tool_name] {"key": "value", ...} or [tool_name] => result
	if len(calls) == 0 {
		re3 := regexp.MustCompile(`(?m)\[(\w+)\]\s*(?:=>[^{]*)?\s*(\{.*?\})`)
		validToolSet := map[string]bool{}
		for _, t := range validTools {
			validToolSet[t] = true
		}
		for _, match := range re3.FindAllStringSubmatch(text, -1) {
			name := match[1]
			if validToolSet[name] {
				var args map[string]any
				if err := json.Unmarshal([]byte(match[2]), &args); err == nil {
					calls = append(calls, ParsedToolCall{Name: name, Args: args})
				}
			}
		}
	}

	// Format 4: <tool_name>{"key": "value"}</tool_name> or <tool_name><key>value</key></tool_name>
	if len(calls) == 0 {
		for _, toolName := range validTools {
			re4 := regexp.MustCompile(fmt.Sprintf(`(?s)<%s>(.*?)</%s>`, regexp.QuoteMeta(toolName), regexp.QuoteMeta(toolName)))
			for _, match := range re4.FindAllStringSubmatch(text, -1) {
				body := strings.TrimSpace(match[1])
				var args map[string]any
				if err := json.Unmarshal([]byte(body), &args); err != nil {
					args = map[string]any{}
					// Parse XML-style <key>value</key> params
					reParam := regexp.MustCompile(`<(\w+)>(.*?)</\w+>`)
					for _, pm := range reParam.FindAllStringSubmatch(body, -1) {
						args[pm[1]] = pm[2]
					}
				}
				if len(args) > 0 {
					calls = append(calls, ParsedToolCall{Name: toolName, Args: args})
				}
			}
		}
	}

	// Format 5: [tool activity] blocks with tool calls on following lines
	if len(calls) == 0 {
		reFormat5 := regexp.MustCompile(`(?s)\[tool activity\]\s*\n(.*)`)
		for _, match := range reFormat5.FindAllStringSubmatch(text, -1) {
			if len(match) >= 2 {
				activityBlock := strings.TrimSpace(match[1])
				lines := strings.Split(activityBlock, "\n")
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if line == "" {
						continue
					}
					// Try [tool] or [tool()] format, optionally with parameters after =>
					reBracketTool := regexp.MustCompile(`^\[([^\]]+)\]\s*(?:=>\s*(.*))?$`)
					if match5 := reBracketTool.FindStringSubmatch(line); len(match5) >= 2 {
						toolName := strings.TrimSpace(match5[1])
						// Strip parentheses and arguments if present: spawn_subagent() -> spawn_subagent
						if idx := strings.Index(toolName, "("); idx >= 0 {
							toolName = toolName[:idx]
						}
						validToolSet := map[string]bool{}
						for _, t := range validTools {
							validToolSet[t] = true
						}
						if validToolSet[toolName] {
							args := map[string]any{}
							if len(match5) >= 3 && strings.TrimSpace(match5[2]) != "" {
								args = parseParamString(match5[2])
							}
							calls = append(calls, ParsedToolCall{Name: toolName, Args: args})
						}
					}
				}
			}
		}
	}

	// Format 6: [tool activity] tool_name(key="value", ...) on the SAME line
	// Some models emit tool calls inline with the [tool activity] marker
	// instead of on a separate line, e.g.:
	//   [tool activity] run_command(command="ls -la")
	//   [tool activity] read_file(path="main.go", limit=100)
	if len(calls) == 0 {
		reFormat6 := regexp.MustCompile(`\[tool activity\]\s+(\w+)\(([^)]*)\)`)
		validToolSet := map[string]bool{}
		for _, t := range validTools {
			validToolSet[t] = true
		}
		for _, match := range reFormat6.FindAllStringSubmatch(text, -1) {
			toolName := match[1]
			if validToolSet[toolName] {
				args := parseFuncCallArgs(match[2])
				calls = append(calls, ParsedToolCall{Name: toolName, Args: args})
			}
		}
	}

	// Format 7: [tool_name](key="value", ...) — markdown-link-style tool calls
	// Some models emit tool calls that look like markdown links, e.g.:
	//   [run_command](command="git status --porcelain")
	//   [read_file](path="main.go")
	if len(calls) == 0 {
		reFormat7 := regexp.MustCompile(`\[(\w+)\]\(([^)]*)\)`)
		validToolSet := map[string]bool{}
		for _, t := range validTools {
			validToolSet[t] = true
		}
		for _, match := range reFormat7.FindAllStringSubmatch(text, -1) {
			toolName := match[1]
			if validToolSet[toolName] {
				args := parseFuncCallArgs(match[2])
				calls = append(calls, ParsedToolCall{Name: toolName, Args: args})
			}
		}
	}

	return calls
}

// convertParamValue attempts to convert a string parameter value to its
// appropriate Go type (int64, float64, bool, or string). This ensures that
// Format 2/2b XML-style tool calls produce the same typed args as other formats.
func convertParamValue(val string) any {
	if num, err := strconv.ParseInt(val, 10, 64); err == nil {
		return num
	}
	if floatVal, err := strconv.ParseFloat(val, 64); err == nil {
		return floatVal
	}
	if boolVal, err := strconv.ParseBool(val); err == nil {
		return boolVal
	}
	return val
}

// parseParamString converts "key=value, key2=value2" into a JSON-serializable map
func parseParamString(paramStr string) map[string]any {
	result := make(map[string]any)
	pairs := strings.Split(paramStr, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		eqIdx := strings.Index(pair, "=")
		if eqIdx < 0 {
			continue
		}
		key := strings.TrimSpace(pair[:eqIdx])
		value := strings.TrimSpace(pair[eqIdx+1:])
		if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			value = value[1 : len(value)-1]
		} else if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
			value = value[1 : len(value)-1]
		}
		if num, err := strconv.ParseInt(value, 10, 64); err == nil {
			result[key] = num
		} else if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			result[key] = floatVal
		} else if boolVal, err := strconv.ParseBool(value); err == nil {
			result[key] = boolVal
		} else {
			result[key] = value
		}
	}
	return result
}

// parseFuncCallArgs parses function-call style arguments like:
//
//	command="cd /src && ls -la", limit=100
//
// It respects quoted values (which may contain commas) and converts
// unquoted values to appropriate Go types.
func parseFuncCallArgs(s string) map[string]any {
	result := make(map[string]any)
	s = strings.TrimSpace(s)
	if s == "" {
		return result
	}

	for s != "" {
		s = strings.TrimSpace(s)
		// Find key
		eqIdx := strings.Index(s, "=")
		if eqIdx < 0 {
			break
		}
		key := strings.TrimSpace(s[:eqIdx])
		s = s[eqIdx+1:]

		var value string
		if len(s) > 0 && (s[0] == '"' || s[0] == '\'') {
			// Quoted value — find matching close quote
			quote := s[0]
			end := -1
			for i := 1; i < len(s); i++ {
				if s[i] == '\\' && i+1 < len(s) {
					i++ // skip escaped character
					continue
				}
				if s[i] == quote {
					end = i
					break
				}
			}
			if end >= 0 {
				value = s[1:end]
				s = s[end+1:]
			} else {
				// No closing quote — take the rest
				value = s[1:]
				s = ""
			}
		} else {
			// Unquoted value — take until comma or end
			commaIdx := strings.Index(s, ",")
			if commaIdx >= 0 {
				value = strings.TrimSpace(s[:commaIdx])
				s = s[commaIdx:]
			} else {
				value = strings.TrimSpace(s)
				s = ""
			}
		}

		// Skip comma separator
		s = strings.TrimSpace(s)
		if len(s) > 0 && s[0] == ',' {
			s = s[1:]
		}

		result[key] = convertParamValue(value)
	}
	return result
}

// stripTextToolCalls removes text-based tool call syntax from assistant
// content so that the model does not see both its own textual tool calls
// and the native tool_calls representation. This prevents the model from
// getting confused and thinking its tools didn't execute.
func stripTextToolCalls(text string) string {
	// Remove [tool activity]...[/tool activity] blocks (may span multiple lines)
	reActivity := regexp.MustCompile(`(?s)\[tool activity\].*?\[/tool activity\]`)
	text = reActivity.ReplaceAllString(text, "")

	// Remove inline [tool activity] tool_name(...) calls (Format 6, no closing tag)
	reInlineActivity := regexp.MustCompile(`\[tool activity\]\s+\w+\([^)]*\)`)
	text = reInlineActivity.ReplaceAllString(text, "")

	// Remove <tool_call>...</tool_call> blocks
	reToolCall := regexp.MustCompile(`(?s)<tool_call>.*?</tool_call>`)
	text = reToolCall.ReplaceAllString(text, "")

	// Remove bare <function=name>...</function> blocks (without <tool_call> wrapper)
	reBareFunc := regexp.MustCompile(`(?s)<function=\w+>.*?</function>`)
	text = reBareFunc.ReplaceAllString(text, "")

	// Remove [tool_name](args) markdown-link-style tool calls (Format 7)
	reMarkdownLink := regexp.MustCompile(`\[\w+\]\([^)]*\)`)
	text = reMarkdownLink.ReplaceAllString(text, "")

	// Remove unclosed <tool_call> or <function=name> blocks (no closing tags)
	reUnclosedToolCall := regexp.MustCompile(`(?s)<tool_call>\s*<function=\w+>.*?\z`)
	text = reUnclosedToolCall.ReplaceAllString(text, "")
	reUnclosedFunc := regexp.MustCompile(`(?s)<function=\w+>\s*<parameter=.*?\z`)
	text = reUnclosedFunc.ReplaceAllString(text, "")

	// Collapse multiple blank lines into one
	reBlank := regexp.MustCompile(`\n{3,}`)
	text = reBlank.ReplaceAllString(text, "\n\n")

	return strings.TrimSpace(text)
}

// ── Model switching ──

// switchModel validates that model is available in Ollama, updates the
// config, and logs an evolution event. Returns an error string if the
// model is not found.
func (a *YoloAgent) switchModel(model string) string {
	models := a.ollama.ListModels()
	found := false
	for _, m := range models {
		if m == model {
			found = true
			break
		}
	}
	if !found {
		return fmt.Sprintf("Model '%s' not found. Available: %s", model, strings.Join(models, ", "))
	}
	old := a.config.GetModel()
	a.config.SetModel(model)
	a.history.AddEvolution("model_switch", fmt.Sprintf("%s -> %s", old, model))
	cprint(Cyan, fmt.Sprintf("  Switched model: %s -> %s", old, model))
	return fmt.Sprintf("Switched from %s to %s", old, model)
}

// ── Sub-agents ──

// spawnSubagent launches a background goroutine that sends task to the LLM
// with tool access and writes the result to .yolo/subagents/agent_{id}.json.
// The sub-agent runs a tool-calling loop (up to MaxSubagentRounds) using the
// safe subset of tools defined by SubagentTools().
func (a *YoloAgent) spawnSubagent(task, model string) string {
	a.mu.Lock()
	a.subagentCounter++
	aid := a.subagentCounter
	a.mu.Unlock()

	useModel := model
	if useModel == "" {
		useModel = a.config.GetModel()
	}
	os.MkdirAll(SubagentDir, 0o755)
	resultFile := filepath.Join(SubagentDir, fmt.Sprintf("agent_%d.json", aid))

	// Write an initial "in-progress" result file so the parent agent can
	// see that this subagent exists and is still working.
	initialData, _ := json.MarshalIndent(map[string]any{
		"id":     aid,
		"task":   task,
		"model":  useModel,
		"status": "in-progress",
		"result": "",
		"ts":     time.Now().Format(time.RFC3339),
	}, "", "  ")
	os.WriteFile(resultFile, initialData, 0o644)

	go func() {
		// Create a dedicated window for this subagent
		if globalUI != nil {
			globalUI.AddSubagentWindow(aid, fmt.Sprintf("subagent #%d", aid))
		}
		cprint(Magenta, fmt.Sprintf("  [sub-agent #%d] started (%s)", aid, useModel))

		prefix := fmt.Sprintf("  [sub-agent #%d]", aid)
		saTools := SubagentTools()

		msgs := []ChatMessage{
			{
				Role: "system",
				Content: fmt.Sprintf("You are a sub-agent with tool access. Use the provided tools to complete this task concisely:\n\n%s\n\nWorking directory: %s",
					task, a.baseDir),
			},
		}

		// Output function that writes to the subagent's window
		var outFn func(string)
		if globalUI != nil {
			outFn = func(text string) {
				globalUI.WriteToSubagentWindow(aid, text)
			}
		}

		status := "complete"
		finalText := ""

		// Tool-calling loop
		var roundMsgs []ChatMessage
		for round := 0; round < MaxSubagentRounds; round++ {
			allMsgs := make([]ChatMessage, 0, len(msgs)+len(roundMsgs))
			allMsgs = append(allMsgs, msgs...)
			allMsgs = append(allMsgs, roundMsgs...)

			chatResult, err := a.ollama.Chat(context.Background(), useModel, allMsgs, saTools, outFn)
			if err != nil {
				finalText = err.Error()
				status = "error"
				break
			}

			toolCalls := chatResult.ToolCalls
			if len(toolCalls) == 0 {
				toolCalls = a.parseTextToolCalls(chatResult.DisplayText)
			}
			toolCalls = deduplicateToolCalls(toolCalls)

			if len(toolCalls) == 0 {
				finalText = chatResult.DisplayText
				break
			}

			// Build assistant message with tool_calls
			var nativeTCs []ToolCall
			for i, tc := range toolCalls {
				argsJSON, _ := json.Marshal(tc.Args)
				nativeTCs = append(nativeTCs, ToolCall{
					ID: fmt.Sprintf("sa%d_call_%d_%d", aid, round, i),
					Function: ToolCallFunc{
						Name:      tc.Name,
						Arguments: json.RawMessage(argsJSON),
					},
				})
			}
			roundMsgs = append(roundMsgs, ChatMessage{
				Role:      "assistant",
				Content:   chatResult.ContentText,
				ToolCalls: nativeTCs,
			})

			// Execute each tool
			for i, call := range toolCalls {
				args := call.Args
				if args == nil {
					args = map[string]any{}
				}

				shortArgs, _ := json.Marshal(args)
				shortStr := string(shortArgs)
				if len(shortStr) > 80 {
					shortStr = shortStr[:80] + "..."
				}
				cprint(Yellow, fmt.Sprintf("%s [%s] %s", prefix, call.Name, shortStr))

				resultStr := executeWithTimeout(a.tools, call.Name, args)

				preview := resultStr
				if len(preview) > 200 {
					preview = preview[:200] + "..."
				}
				preview = strings.ReplaceAll(preview, "\r", "")
				preview = strings.ReplaceAll(preview, "\n", " ")
				cprint(Gray, fmt.Sprintf("%s => %s", prefix, preview))

				cleanResult := filterToolActivityMarkers(resultStr)
				roundMsgs = append(roundMsgs, ChatMessage{
					Role:       "tool",
					Content:    cleanResult,
					ToolCallID: fmt.Sprintf("sa%d_call_%d_%d", aid, round, i),
				})
			}
		}

		if finalText == "" {
			finalText = "(no output)"
		}

		data, _ := json.MarshalIndent(map[string]any{
			"id":     aid,
			"task":   task,
			"model":  useModel,
			"status": status,
			"result": finalText,
			"ts":     time.Now().Format(time.RFC3339),
		}, "", "  ")
		os.WriteFile(resultFile, data, 0o644)
		cprint(Magenta, fmt.Sprintf("\n  [sub-agent #%d] %s. See %s", aid, status, resultFile))

		// Mark window as complete (starts 300s removal timer)
		if globalUI != nil {
			globalUI.MarkSubagentComplete(aid)
		}
	}()

	return fmt.Sprintf("Sub-agent #%d spawned (%s). Results -> %s", aid, useModel, resultFile)
}

// handoffRemainingTools forks the remaining unexecuted tool calls to a
// background goroutine so the main agent can immediately process user input.
// The results are collected in a handoffResult struct that the main agent
// ingests after its next conversation turn.
func (a *YoloAgent) handoffRemainingTools(remaining []ParsedToolCall) *handoffResult {
	a.mu.Lock()
	a.handoffCounter++
	hid := a.handoffCounter
	a.mu.Unlock()

	hr := &handoffResult{
		ID:   hid,
		Done: make(chan struct{}),
	}

	a.mu.Lock()
	a.pendingHandoffs = append(a.pendingHandoffs, hr)
	a.mu.Unlock()

	go func() {
		defer close(hr.Done)
		cprint(Magenta, fmt.Sprintf("  [handoff #%d] executing %d remaining tool(s) in background", hid, len(remaining)))

		var results []toolExecResult
		for _, call := range remaining {
			args := call.Args
			if args == nil {
				args = map[string]any{}
			}

			shortArgs, _ := json.Marshal(args)
			shortStr := string(shortArgs)
			if len(shortStr) > 80 {
				shortStr = shortStr[:80] + "..."
			}
			cprint(Yellow, fmt.Sprintf("  [handoff #%d] [%s] %s", hid, call.Name, shortStr))

			resultStr := executeWithTimeout(a.tools, call.Name, args)

			preview := resultStr
			if len(preview) > 200 {
				preview = preview[:200] + "..."
			}
			preview = strings.ReplaceAll(preview, "\r", "")
			preview = strings.ReplaceAll(preview, "\n", " ")
			cprint(Gray, fmt.Sprintf("  [handoff #%d] => %s", hid, preview))

			results = append(results, toolExecResult{
				Name:   call.Name,
				Args:   args,
				Result: filterToolActivityMarkers(resultStr),
			})
		}

		a.mu.Lock()
		hr.Results = results
		cprint(Magenta, fmt.Sprintf("  [handoff #%d] complete (%d tools executed)", hid, len(results)))
		a.mu.Unlock()
	}()

	return hr
}

// ingestHandoffResults collects completed background handoff results and
// injects them into the conversation history so the main agent has full
// context of what happened. Returns the number of handoffs ingested.
func (a *YoloAgent) ingestHandoffResults() int {
	a.mu.Lock()
	pending := a.pendingHandoffs
	a.mu.Unlock()

	if len(pending) == 0 {
		return 0
	}

	ingested := 0
	var remaining []*handoffResult
	for _, hr := range pending {
		select {
		case <-hr.Done:
			// Handoff complete — build a summary for the agent's context
			var summary strings.Builder
			summary.WriteString(fmt.Sprintf("[Background handoff #%d completed]\n", hr.ID))
			summary.WriteString("The following tools were executed in the background while you were responding to the user:\n\n")
			for _, r := range hr.Results {
				argsJSON, _ := json.Marshal(r.Args)
				summary.WriteString(fmt.Sprintf("Tool: %s\nArgs: %s\nResult: %s\n\n", r.Name, string(argsJSON), r.Result))
			}
			a.history.AddMessage("system", summary.String(), nil)
			cprint(Cyan, fmt.Sprintf("  [handoff #%d] results injected into context", hr.ID))
			ingested++
		default:
			// Still running — keep it in the pending list
			remaining = append(remaining, hr)
		}
	}

	a.mu.Lock()
	a.pendingHandoffs = remaining
	a.mu.Unlock()

	return ingested
}

// ── Slash commands ──

// handleCommand processes interactive slash commands (/help, /model, /clear, etc.).
func (a *YoloAgent) handleCommand(cmd string) {
	parts := strings.SplitN(cmd, " ", 2)
	command := strings.ToLower(parts[0])
	arg := ""
	if len(parts) > 1 {
		arg = parts[1]
	}

	switch command {
	case "/help", "/h":
		cprint(Cyan, "Commands:")
		cprint(Reset, "  /help            Show this help")
		cprint(Reset, "  /terminal [on|off] Toggle terminal mode (split-screen UI)")
		cprint(Reset, "  /model           Current model")
		cprint(Reset, "  /models          List available models")
		cprint(Reset, "  /switch <name>   Switch model")
		cprint(Reset, "  /history         Message count")
		cprint(Reset, "  /clear           Clear conversation history")
		cprint(Reset, "  /status          Agent status")
		cprint(Reset, "  /cache           Show/clear search cache stats")
		cprint(Reset, "  /learn           Run autonomous research for self-improvement")
		cprint(Reset, "  /restart         Restart YOLO")
		cprint(Reset, "  /exit, /quit     Exit YOLO")

	case "/terminal":
		currentMode := a.config.GetTerminalMode()
		switch strings.ToLower(strings.TrimSpace(arg)) {
		case "on":
			if currentMode {
				cprint(Cyan, "  Terminal mode is already enabled")
			} else {
				a.enableTerminalMode()
			}
		case "off":
			if !currentMode {
				cprint(Cyan, "  Terminal mode is already disabled (buffer mode)")
			} else {
				a.disableTerminalMode()
			}
		case "":
			// Toggle
			if currentMode {
				a.disableTerminalMode()
			} else {
				a.enableTerminalMode()
			}
		default:
			cprint(Red, "  Usage: /terminal [on|off]")
		}

	case "/model":
		cprint(Cyan, fmt.Sprintf("  Model: %s", a.config.GetModel()))

	case "/models":
		models := a.ollama.ListModels()
		cprint(Cyan, "  Available models:")
		for _, m := range models {
			marker := ""
			if m == a.config.GetModel() {
				marker = fmt.Sprintf(" %s<- current%s", Green, Reset)
			}
			cprint(Reset, fmt.Sprintf("    %s%s", m, marker))
		}

	case "/switch":
		if arg != "" {
			a.switchModel(arg)
		} else {
			cprint(Red, "  Usage: /switch <model-name>")
		}

	case "/history":
		n := len(a.history.Data.Messages)
		e := len(a.history.Data.EvolutionLog)
		cprint(Cyan, fmt.Sprintf("  Messages: %d  |  Evolution events: %d", n, e))

	case "/clear":
		a.history.Data.Messages = []HistoryMessage{}
		if err := a.history.Save(); err != nil {
			cprint(Red, fmt.Sprintf("  Warning: could not save history: %v\n", err))
		}
		cprint(Cyan, "  History cleared (config preserved)")

	case "/cache":
		a.showCacheStatus(arg)

	case "/learn":
		cprint(Yellow, "  Starting autonomous learning research...")
		go func() {
			time.Sleep(500 * time.Millisecond)
			result := a.tools.learn(map[string]any{})
			cprint(Cyan, result)
		}()
		return // Let the goroutine handle the learn tool

	case "/status":
		cprint(Cyan, "Status:")
		cprint(Reset, fmt.Sprintf("  Model:       %s", a.config.GetModel()))
		cprint(Reset, fmt.Sprintf("  Messages:    %d", len(a.history.Data.Messages)))
		cprint(Reset, fmt.Sprintf("  Evolutions:  %d", len(a.history.Data.EvolutionLog)))
		cprint(Reset, fmt.Sprintf("  Working dir: %s", a.baseDir))
		cprint(Reset, fmt.Sprintf("  Script:      %s", a.scriptPath))

	case "/restart":
		cprint(Yellow, "  Restarting YOLO...")
		// a.running = false
		go func() {
			time.Sleep(1 * time.Second)
			a.restart()
		}()
		return // Let the goroutine handle restart

	case "/exit", "/quit":
		a.running = false
		return // Exit the function to stop processing further input

	default:
		cprint(Red, fmt.Sprintf("  Unknown command: %s  (try /help)", command))
	}
}

// showCacheStatus displays web search cache statistics or clears it
func (a *YoloAgent) showCacheStatus(arg string) {
	if strings.ToLower(strings.TrimSpace(arg)) == "clear" {
		searchCache.Clear()
		cprint(Green, "  Search cache cleared")
		return
	}

	// Count cache entries and expired entries
	count := 0
	expired := 0
	now := time.Now()
	searchCache.Range(func(key, value any) bool {
		count++
		if entry, ok := value.(*searchCacheEntry); ok {
			if now.Sub(entry.Ts) >= searchCacheTTL {
				expired++
			}
		}
		return true
	})

	cprint(Cyan, "Search Cache:")
	cprint(Reset, fmt.Sprintf("  Total entries: %d", count))
	cprint(Reset, fmt.Sprintf("  Expired entries: %d", expired))
	cprint(Reset, fmt.Sprintf("  Valid entries: %d", count-expired))
	cprint(Reset, fmt.Sprintf("  TTL: %v", searchCacheTTL))
	if arg != "clear" {
		cprint(Reset, "  Usage: /cache clear (to clear cache)")
	}
}

// ── Main loop ──

func (a *YoloAgent) showPrompt() {
	if bufferUI != nil && globalUI == nil {
		// Buffer mode: prompt appears when user starts typing.
		return
	}
	// Terminal mode: the divider label "──you──" serves as the indicator.
	// Just trigger a redraw of the input area.
	a.inputMgr.ShowPrompt("")
}

// echoUserInput prints the user's (possibly multiline) message to the
// output area with a "you>" prefix on the first line. In buffer mode the
// text is already visible in the scrollback, so nothing is printed.
func (a *YoloAgent) echoUserInput(text string) {
	if bufferUI != nil && globalUI == nil {
		return // text already on screen in buffer mode
	}
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if i == 0 {
			cprint(Green, fmt.Sprintf("  you> %s", line))
		} else {
			cprint(Green, fmt.Sprintf("        %s", line))
		}
	}
}

// Run is the top-level entry point. It loads (or creates) session history,
// initialises the terminal UI and input manager, registers signal handlers,
// and enters the main event loop. It blocks until the user exits.
func (a *YoloAgent) Run() {
	a.config.Load()
	hasHistory := a.history.Load()

	hasModel := a.config.GetModel() != ""
	if hasModel {
		// Config has a model; display happens later via displaySessionResumption()
	} else {
		a.setupFirstRun()
	}

	// Set up UI based on terminal mode config.
	// Default (terminal_mode=false) uses buffer mode for scrollable history.
	if a.config.GetTerminalMode() {
		globalUI = NewTerminalUI()
		globalUI.Setup()
	} else {
		bufferUI = NewBufferUI()
	}
	defer func() {
		if globalUI != nil {
			globalUI.Teardown()
			globalUI = nil
		}
		bufferUI = nil
	}()

	// Display session resumption message AFTER terminal is set up
	if hasModel && hasHistory {
		a.displaySessionResumption()
	}

	// Start async input manager
	a.inputMgr = NewInputManager(a)
	a.inputMgr.Start()
	defer a.inputMgr.Stop()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGWINCH)
	go func() {
		for sig := range sigCh {
			switch sig {
			case syscall.SIGWINCH:
				if globalUI != nil {
					globalUI.RefreshSize()
				}
			case syscall.SIGINT:
				a.mu.Lock()
				cancel := a.cancelChat
				a.mu.Unlock()
				if cancel != nil {
					cancel()
				} else {
					a.running = false
					cprint(Cyan, "\n  Interrupted — saving session...")
				}
			}
		}
	}()

	a.showPrompt()

	for a.running {
		select {
		case line := <-a.inputMgr.Lines:
			if !line.OK {
				a.running = false
				break
			}

			stripped := strings.TrimSpace(line.Text)
			lower := strings.ToLower(stripped)

			if lower == "exit" || lower == "quit" {
				a.running = false
			} else if strings.HasPrefix(stripped, "/") {
				a.handleCommand(stripped)
				a.showPrompt()
			} else if stripped != "" {
				a.mu.Lock()
				a.busy = true
				a.mu.Unlock()

				// Echo user's multiline input
				a.echoUserInput(stripped)

				a.chatWithAgent(stripped, false)

				a.mu.Lock()
				a.busy = false
				a.mu.Unlock()

				a.showPrompt()
			}

		case <-time.After(100 * time.Millisecond):
			// No user input — immediately enter autonomous thinking
			// (the agent is always thinking unless spoken to)
			a.inputMgr.mu.Lock()
			bufEmpty := len(a.inputMgr.buf) == 0
			a.inputMgr.mu.Unlock()
			if bufEmpty {
				a.inputMgr.ClearLine()

				a.mu.Lock()
				a.busy = true
				a.mu.Unlock()

				a.chatWithAgent("", true)

				a.mu.Lock()
				a.busy = false
				a.mu.Unlock()

				a.showPrompt()
			}
		}
	}

	if err := a.history.Save(); err != nil {
		cprint(Red, fmt.Sprintf("  Warning: could not save history: %v\n", err))
	}
	fmt.Print("\r\n")
	cprint(Cyan, "  Session saved. Goodbye!\n")
}
