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

type HistoryMessage struct {
	Role    string         `json:"role"`
	Content string         `json:"content"`
	TS      string         `json:"ts"`
	Meta    map[string]any `json:"meta,omitempty"`
}

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
