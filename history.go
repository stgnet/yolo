package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ─── History Manager ──────────────────────────────────────────────────

type HistoryMessage struct {
	Role    string         `json:"role"`
	Content string         `json:"content"`
	TS      string         `json:"ts"`
	Meta    map[string]any `json:"meta,omitempty"`
}

// ── Backward Compatibility ────────────────

// MessageHistory is an alias for HistoryManager (for test compatibility)
type MessageHistory struct {
	SessionID        string
	CurrentAssistant *MessageHistoryItem
	CurrentUser      *MessageHistoryItem
	Messages         []HistoryMessage
}

type MessageHistoryItem struct {
	Type    string
	Value   string
	Message string // Added for test compatibility
}

// NewMessageHistory creates a new history with initial system message
func NewMessageHistory(sessionID string) *MessageHistory {
	return &MessageHistory{
		SessionID: sessionID,
		Messages: []HistoryMessage{
			{Role: "system", Content: "You are a helpful assistant.", TS: time.Now().Format(time.RFC3339)},
		},
	}
}

func (h *MessageHistory) AddUserMessage(content string) {
	h.Messages = append(h.Messages, HistoryMessage{
		Role:    "user",
		Content: content,
		TS:      time.Now().Format(time.RFC3339),
	})
}

func (h *MessageHistory) AddAssistantMessage(content string) {
	h.Messages = append(h.Messages, HistoryMessage{
		Role:    "assistant",
		Content: content,
		TS:      time.Now().Format(time.RFC3339),
	})
}

func (h *MessageHistory) StartToolCall(name string, args map[string]any) {
	argsJSON, _ := json.Marshal(args)
	newMsg := fmt.Sprintf("%s(%s)", name, string(argsJSON))

	if h.CurrentAssistant != nil && h.CurrentAssistant.Message != "" {
		h.CurrentAssistant.Value = name
		h.CurrentAssistant.Message += " → " + newMsg
	} else {
		h.CurrentAssistant = &MessageHistoryItem{Type: "tool_call", Value: name, Message: newMsg}
	}
}

func (h *MessageHistory) EndToolCall(result string) {
	h.Messages = append(h.Messages, HistoryMessage{
		Role:    "tool",
		Content: result,
		TS:      time.Now().Format(time.RFC3339),
	})
}

func (h *MessageHistory) Save() string {
	// Create temp file for this session's history
	filename := filepath.Join(os.TempDir(), "yolo_history_"+h.SessionID+"_"+strconv.FormatInt(time.Now().UnixNano(), 10)+".json")

	data, err := json.MarshalIndent(h.Messages, "", "  ")
	if err != nil {
		return ""
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return ""
	}

	return filename
}

// LoadMessageHistory loads a saved history by session ID
func LoadMessageHistory(sessionID string, clearOnFailure bool) (*MessageHistory, error) {
	// Find and load the most recent save file for this session
	pattern := filepath.Join(os.TempDir(), "yolo_history_"+sessionID+"*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		if clearOnFailure {
			return &MessageHistory{SessionID: sessionID}, nil
		}
		return nil, os.ErrNotExist
	}

	// Sort to get the most recent file
	sort.Strings(matches)
	filename := matches[len(matches)-1]

	data, err := os.ReadFile(filename)
	if err != nil {
		if clearOnFailure {
			return &MessageHistory{SessionID: sessionID}, nil
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var messages []HistoryMessage
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	h := &MessageHistory{SessionID: sessionID}
	h.Messages = messages
	return h, nil
}

// ── Color constants for tests ────────

type Color int

const (
	ColorNone Color = iota
	ColorBold
	ColorRed
	ColorGreen
	ColorYellow
	ColorBlue
	ColorMagenta
	ColorCyan
	ColorGray
	BGRed
	BGGreen
)

// escapeMarkdown escapes markdown characters for display
func escapeMarkdown(s string) string {
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

// ────────────────────────────────────────

type EvolutionEntry struct {
	TS     string `json:"ts"`
	Action string `json:"action"`
	Detail string `json:"detail"`
}

type HistoryConfig struct {
	Model   string `json:"model"`
	Created string `json:"created"`
}

type HistoryData struct {
	Version      int              `json:"version"`
	Config       HistoryConfig    `json:"config"`
	Messages     []HistoryMessage `json:"messages"`
	EvolutionLog []EvolutionEntry `json:"evolution_log"`
}

type HistoryManager struct {
	yoloDir     string
	historyFile string
	Data        HistoryData
	mu          sync.Mutex
}

func NewHistoryManager(yoloDir string) *HistoryManager {
	h := &HistoryManager{
		yoloDir:     yoloDir,
		historyFile: filepath.Join(yoloDir, "history.json"),
	}
	h.Data = h.empty()
	return h
}

func (h *HistoryManager) empty() HistoryData {
	return HistoryData{
		Version: 1,
		Config: HistoryConfig{
			Model:   "",
			Created: time.Now().Format(time.RFC3339),
		},
		Messages:     []HistoryMessage{},
		EvolutionLog: []EvolutionEntry{},
	}
}

func (h *HistoryManager) Load() bool {
	data, err := os.ReadFile(h.historyFile)
	if err != nil {
		return false
	}
	if err := json.Unmarshal(data, &h.Data); err != nil {
		cprint(Yellow, "Warning: corrupt history, starting fresh")
		h.Data = h.empty()
		return false
	}
	return true
}

func (h *HistoryManager) Save() {
	h.mu.Lock()
	defer h.mu.Unlock()

	os.MkdirAll(h.yoloDir, 0o755)
	data, err := json.MarshalIndent(h.Data, "", "  ")
	if err != nil {
		return
	}
	tmp := h.historyFile + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return
	}
	os.Rename(tmp, h.historyFile)
}

func (h *HistoryManager) AddMessage(role, content string, meta map[string]any) {
	h.mu.Lock()
	msg := HistoryMessage{
		Role:    role,
		Content: content,
		TS:      time.Now().Format(time.RFC3339),
		Meta:    meta,
	}
	h.Data.Messages = append(h.Data.Messages, msg)
	h.mu.Unlock()
	h.Save()
}

func (h *HistoryManager) AddEvolution(action, description string) {
	h.mu.Lock()
	h.Data.EvolutionLog = append(h.Data.EvolutionLog, EvolutionEntry{
		TS:     time.Now().Format(time.RFC3339),
		Action: action,
		Detail: description,
	})
	h.mu.Unlock()
	h.Save()
}

func (h *HistoryManager) GetContextMessages(maxMsgs int) []ChatMessage {
	msgs := h.Data.Messages
	start := 0
	if len(msgs) > maxMsgs {
		start = len(msgs) - maxMsgs
	}
	recent := msgs[start:]

	var out []ChatMessage
	for _, m := range recent {
		switch m.Role {
		case "user", "assistant":
			out = append(out, ChatMessage{Role: m.Role, Content: m.Content})
		case "tool":
			out = append(out, ChatMessage{Role: "user", Content: "[Tool result]\n" + m.Content})
		case "system":
			out = append(out, ChatMessage{Role: "user", Content: "[SYSTEM] " + m.Content})
		}
	}
	return out
}

func (h *HistoryManager) GetModel() string {
	return h.Data.Config.Model
}

func (h *HistoryManager) SetModel(model string) {
	h.Data.Config.Model = model
	h.Save()
}

// GetLastN returns the last n history messages
func (h *HistoryManager) GetLastN(n int) []HistoryMessage {
	h.mu.Lock()
	defer h.mu.Unlock()
	msgs := h.Data.Messages
	if len(msgs) <= n {
		return msgs
	}
	return msgs[len(msgs)-n:]
}
