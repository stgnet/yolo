package main

// ─── History Types ──────────────────────────────────────────────────
//
// Data types for conversation history, evolution events, and session
// configuration. The actual storage backend is HistoryDB in historydb.go.

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

// HistoryData is the in-memory representation of recent history.
// Used for backward compatibility with code that reads Data.Messages.
type HistoryData struct {
	Version      int              `json:"version"`
	Config       HistoryConfig    `json:"config"`
	Messages     []HistoryMessage `json:"messages"`
	EvolutionLog []EvolutionEntry `json:"evolution_log"`
}

// MaxHistoryMessages caps how many messages are kept in the in-memory
// Data.Messages cache. The database retains everything permanently.
const MaxHistoryMessages = 200

// MaxEvolutionEntries caps in-memory evolution entries.
const MaxEvolutionEntries = 100
