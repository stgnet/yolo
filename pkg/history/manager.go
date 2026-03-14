// Package history provides persistent storage for conversation history and logs.
package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// HistoryMessage represents a single message in the conversation.
type HistoryMessage struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	Timestamp string     `json:"timestamp"`
}

// ToolCall represents a tool call in the history.
type ToolCall struct {
	ID        string                 `json:"id"`
	Function  map[string]interface{} `json:"function"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// HistoryLog represents an entry in the evolution log.
type HistoryLogEntry struct {
	Time     string  `json:"time"`
	Type     string  `json:"type"`
	Message  string  `json:"message"`
	Duration float64 `json:"duration,omitempty"`
}

// HistoryData is the complete history structure.
type HistoryData struct {
	Messages    []HistoryMessage  `json:"messages"`
	Evolution   []HistoryLogEntry `json:"evolution"`
	LastUpdated string            `json:"last_updated"`
	Model       string            `json:"model,omitempty"`
}

// HistoryManager manages persistent conversation history.
type HistoryManager struct {
	mu       sync.RWMutex
	data     HistoryData
	filePath string
}

// NewHistoryManager creates a new history manager.
func NewHistoryManager(baseDir, filePath string) *HistoryManager {
	h := &HistoryManager{
		data: HistoryData{
			Messages:  make([]HistoryMessage, 0),
			Evolution: make([]HistoryLogEntry, 0),
		},
		filePath: filePath,
	}

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Printf("Warning: could not create history directory %s: %v\n", dir, err)
	}

	// Load existing history if present
	h.Load()
	return h
}

// Load reads the history from disk.
func (h *HistoryManager) Load() {
	h.mu.Lock()
	defer h.mu.Unlock()

	data, err := os.ReadFile(h.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return // New history, nothing to load
		}
		fmt.Printf("Warning: could not read history file %s: %v\n", h.filePath, err)
		return
	}

	if err := json.Unmarshal(data, &h.data); err != nil {
		fmt.Printf("Warning: could not parse history file %s: %v\n", h.filePath, err)
		h.data = HistoryData{
			Messages:  make([]HistoryMessage, 0),
			Evolution: make([]HistoryLogEntry, 0),
		}
		return
	}

	h.data.LastUpdated = time.Now().Format(time.RFC3339)
}

// Save writes the history to disk.
func (h *HistoryManager) Save() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	data, err := json.MarshalIndent(h.data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal history: %w", err)
	}

	if err := os.WriteFile(h.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write history file: %w", err)
	}

	h.data.LastUpdated = time.Now().Format(time.RFC3339)
	return nil
}

// AddMessage appends a new message to the conversation.
func (h *HistoryManager) AddMessage(role string, content string, toolCalls []ToolCall) {
	h.mu.Lock()
	defer h.mu.Unlock()

	msg := HistoryMessage{
		Role:      role,
		Content:   content,
		ToolCalls: toolCalls,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	h.data.Messages = append(h.data.Messages, msg)
	h.data.LastUpdated = time.Now().Format(time.RFC3339)
}

// AddLogEntry adds an entry to the evolution log.
func (h *HistoryManager) AddLogEntry(logEntry HistoryLogEntry) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.data.Evolution = append(h.data.Evolution, logEntry)
	h.data.LastUpdated = time.Now().Format(time.RFC3339)
}

// GetMessages returns all messages.
func (h *HistoryManager) GetMessages() []HistoryMessage {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Return a copy to prevent external modification
	messages := make([]HistoryMessage, len(h.data.Messages))
	copy(messages, h.data.Messages)
	return messages
}

// GetRecentMessages returns the last n messages.
func (h *HistoryManager) GetRecentMessages(n int) []HistoryMessage {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if n <= 0 || n >= len(h.data.Messages) {
		return h.GetMessages()
	}

	start := len(h.data.Messages) - n
	messages := make([]HistoryMessage, n)
	copy(messages, h.data.Messages[start:])
	return messages
}

// GetModel returns the current model from history.
func (h *HistoryManager) GetModel() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.data.Model
}

// SetModel sets the model name in history.
func (h *HistoryManager) SetModel(model string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.data.Model = model
	h.data.LastUpdated = time.Now().Format(time.RFC3339)
}

// Reset clears all messages and logs.
func (h *HistoryManager) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.data.Messages = make([]HistoryMessage, 0)
	h.data.Evolution = make([]HistoryLogEntry, 0)
	h.data.LastUpdated = time.Now().Format(time.RFC3339)
}

// Truncate removes the oldest messages to keep only the most recent n.
func (h *HistoryManager) Truncate(maxMessages int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.data.Messages) <= maxMessages {
		return
	}
	h.data.Messages = h.data.Messages[len(h.data.Messages)-maxMessages:]
	h.data.LastUpdated = time.Now().Format(time.RFC3339)
}

// Stats returns basic statistics about the history.
func (h *HistoryManager) Stats() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return map[string]interface{}{
		"total_messages": len(h.data.Messages),
		"total_logs":     len(h.data.Evolution),
		"last_updated":   h.data.LastUpdated,
		"model":          h.data.Model,
	}
}
