package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ─── History Manager ──────────────────────────────────────────────────
//
// HistoryManager persists the conversation and evolution log to a JSON
// file inside the .yolo directory.  It is safe for concurrent use.

// HistoryMessage is a single timestamped message in the conversation.
type HistoryMessage struct {
	Role    string         `json:"role"` // "user", "assistant", "tool", or "system"
	Content string         `json:"content"`
	TS      string         `json:"ts"`             // RFC 3339 timestamp
	Meta    map[string]any `json:"meta,omitempty"` // optional key-value metadata
}

// EvolutionEntry records a significant agent event (e.g. model switch).
type EvolutionEntry struct {
	TS     string `json:"ts"`
	Action string `json:"action"` // short action tag, e.g. "model_switch"
	Detail string `json:"detail"` // human-readable description
}

// HistoryConfig stores session-level configuration persisted alongside
// messages.
type HistoryConfig struct {
	Model   string `json:"model"`   // currently selected Ollama model
	Created string `json:"created"` // session creation timestamp
}

// HistoryData is the top-level JSON structure written to history.json.
type HistoryData struct {
	Version      int              `json:"version"`
	Config       HistoryConfig    `json:"config"`
	Messages     []HistoryMessage `json:"messages"`
	EvolutionLog []EvolutionEntry `json:"evolution_log"`
}

// HistoryManager owns the in-memory HistoryData and handles reading and
// writing it to disk.  All mutating methods are goroutine-safe.
type HistoryManager struct {
	yoloDir     string
	historyFile string
	Data        HistoryData
	mu          sync.Mutex
}

// NewHistoryManager creates a manager that stores its file in yoloDir.
// The data starts empty; call Load to read an existing file.
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

// Load reads history.json from disk. Returns true on success, false if the
// file is missing or corrupt (in which case Data is reset to empty).
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

// Save writes the current Data to history.json atomically (write-to-tmp then
// rename). It creates the yoloDir if it does not exist.
func (h *HistoryManager) Save() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if err := os.MkdirAll(h.yoloDir, 0o755); err != nil {
		return fmt.Errorf("create history dir: %w", err)
	}
	data, err := json.MarshalIndent(h.Data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal history: %w", err)
	}
	tmp := h.historyFile + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write history: %w", err)
	}
	if err := os.Rename(tmp, h.historyFile); err != nil {
		return fmt.Errorf("rename history: %w", err)
	}
	return nil
}

// AddMessage appends a new message and persists to disk.
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

// AddEvolution appends an evolution event and persists to disk.
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

// GetContextMessages returns the last maxMsgs messages converted to
// ChatMessage format suitable for sending to the LLM. Tool and system
// messages are re-mapped to the "user" role with appropriate prefixes.
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

// GetModel returns the currently configured model name.
func (h *HistoryManager) GetModel() string {
	return h.Data.Config.Model
}

// SetModel updates the configured model and persists to disk.
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
