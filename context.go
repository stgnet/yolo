package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ─── Context Window Manager ──────────────────────────────────────────
//
// ContextManager handles intelligent context window management:
//   - Maintains a running conversation summary that persists across sessions
//   - Automatically compacts when message count approaches the context limit
//   - Adjusts context message limits based on detected model context window
//   - Provides /compact command for manual compaction
//
// The summary is stored in .yolo/context_summary.json and is prepended
// to the context window as a system message, giving the LLM awareness
// of older conversation even after messages are pruned.

const (
	// CompactionThreshold is the fraction of MaxContextMessages at which
	// auto-compaction triggers. At 0.75, compaction fires when we reach
	// 75% of the context message limit.
	CompactionThreshold = 0.75

	// CompactionKeepRecent is how many recent messages to preserve after
	// compaction. Older messages are summarized and removed.
	CompactionKeepRecent = 15

	// SummaryMaxLength caps the summary length in characters to prevent
	// it from consuming too much of the context window.
	SummaryMaxLength = 3000

	// TokensPerMessage is a rough estimate of tokens per message for
	// dynamic context sizing. Used to convert token budgets to message counts.
	TokensPerMessage = 150

	// MinContextMessages is the floor for dynamic context sizing.
	MinContextMessages = 20

	// MaxDynamicContextMessages is the ceiling for dynamic context sizing.
	MaxDynamicContextMessages = 100

	// ContextBudgetFraction is the fraction of the model's context window
	// reserved for conversation history (remainder is for response).
	ContextBudgetFraction = 0.7
)

// ContextSummaryData is persisted to .yolo/context_summary.json.
type ContextSummaryData struct {
	Summary       string `json:"summary"`        // running conversation summary
	MessageCount  int    `json:"message_count"`  // messages summarized so far
	LastCompacted string `json:"last_compacted"` // RFC 3339 timestamp
	Compactions   int    `json:"compactions"`    // total compaction count
}

// ContextManager owns the conversation summary and compaction logic.
type ContextManager struct {
	yoloDir  string
	dataFile string
	Data     ContextSummaryData
	mu       sync.Mutex
}

// NewContextManager creates a manager that stores state in .yolo/.
func NewContextManager(yoloDir string) *ContextManager {
	cm := &ContextManager{
		yoloDir:  yoloDir,
		dataFile: filepath.Join(yoloDir, "context_summary.json"),
	}
	cm.Load()
	return cm
}

// Load reads the persisted summary from disk.
func (cm *ContextManager) Load() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	data, err := os.ReadFile(cm.dataFile)
	if err != nil {
		return
	}
	json.Unmarshal(data, &cm.Data)
}

// Save persists the summary to disk atomically.
func (cm *ContextManager) Save() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return cm.saveLocked()
}

func (cm *ContextManager) saveLocked() error {
	if err := os.MkdirAll(cm.yoloDir, 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}
	data, err := json.MarshalIndent(cm.Data, "", "  ")
	if err != nil {
		return err
	}
	tmp := cm.dataFile + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, cm.dataFile)
}

// GetSummary returns the current conversation summary, or empty string.
func (cm *ContextManager) GetSummary() string {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return cm.Data.Summary
}

// SetSummary updates the summary and persists to disk.
func (cm *ContextManager) SetSummary(summary string, messagesCompacted int) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	// Truncate if too long
	if len(summary) > SummaryMaxLength {
		summary = summary[:SummaryMaxLength] + "\n...(truncated)"
	}
	cm.Data.Summary = summary
	cm.Data.MessageCount += messagesCompacted
	cm.Data.LastCompacted = time.Now().Format(time.RFC3339)
	cm.Data.Compactions++
	cm.saveLocked()
}

// NeedsCompaction returns true if the message count has reached the threshold.
func (cm *ContextManager) NeedsCompaction(messageCount, contextLimit int) bool {
	threshold := int(float64(contextLimit) * CompactionThreshold)
	return messageCount >= threshold
}

// GetStatus returns a human-readable status string.
func (cm *ContextManager) GetStatus() string {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.Data.Summary == "" {
		return "No conversation summary (no compactions performed)"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  Compactions: %d\n", cm.Data.Compactions))
	sb.WriteString(fmt.Sprintf("  Messages summarized: %d\n", cm.Data.MessageCount))
	if cm.Data.LastCompacted != "" {
		sb.WriteString(fmt.Sprintf("  Last compacted: %s\n", cm.Data.LastCompacted))
	}
	sb.WriteString(fmt.Sprintf("  Summary length: %d chars\n", len(cm.Data.Summary)))
	return sb.String()
}

// ClearSummary resets the summary.
func (cm *ContextManager) ClearSummary() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.Data = ContextSummaryData{}
	cm.saveLocked()
}

// ─── Dynamic Context Sizing ─────────────────────────────────────────

// DynamicContextLimit calculates the optimal MaxContextMessages based
// on the model's context window size. Returns the default if the
// model context can't be determined.
func DynamicContextLimit(modelContextTokens int) int {
	if modelContextTokens <= 0 {
		return MaxContextMessages
	}

	// Reserve a fraction of the context window for conversation messages
	tokenBudget := int(float64(modelContextTokens) * ContextBudgetFraction)
	msgLimit := tokenBudget / TokensPerMessage

	if msgLimit < MinContextMessages {
		return MinContextMessages
	}
	if msgLimit > MaxDynamicContextMessages {
		return MaxDynamicContextMessages
	}
	return msgLimit
}

// ─── Compaction Logic ────────────────────────────────────────────────

// extractSummary builds a concise programmatic summary from messages
// without calling the LLM. It extracts user questions, tool operations,
// and key outcomes into a structured summary.
func extractSummary(existingSummary string, messages []HistoryMessage) string {
	var sb strings.Builder

	// Carry forward existing summary (condensed)
	if existingSummary != "" {
		// Keep only the most recent portion if existing summary is long
		existing := existingSummary
		if len(existing) > SummaryMaxLength/2 {
			existing = existing[len(existing)-SummaryMaxLength/2:]
			// Find first newline to avoid cutting mid-sentence
			if idx := strings.Index(existing, "\n"); idx > 0 {
				existing = existing[idx+1:]
			}
		}
		sb.WriteString(existing)
		sb.WriteString("\n")
	}

	for _, m := range messages {
		content := strings.TrimSpace(m.Content)
		if content == "" {
			continue
		}
		switch m.Role {
		case "user":
			// Capture user requests (first line, truncated)
			line := firstLine(content, 120)
			sb.WriteString(fmt.Sprintf("User: %s\n", line))
		case "assistant":
			// Capture assistant decisions/responses (first line, truncated)
			line := firstLine(content, 120)
			sb.WriteString(fmt.Sprintf("Agent: %s\n", line))
		case "tool":
			// Capture tool results briefly
			line := firstLine(content, 80)
			sb.WriteString(fmt.Sprintf("Tool result: %s\n", line))
		case "system":
			// Skip system messages in summary (they're ephemeral)
		}
	}

	result := sb.String()
	if len(result) > SummaryMaxLength {
		result = result[:SummaryMaxLength]
		// Trim to last complete line
		if idx := strings.LastIndex(result, "\n"); idx > 0 {
			result = result[:idx]
		}
	}
	return result
}

// firstLine returns the first line of s, truncated to maxLen characters.
func firstLine(s string, maxLen int) string {
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		s = s[:idx]
	}
	if len(s) > maxLen {
		s = s[:maxLen] + "..."
	}
	return s
}

// CompactHistory summarizes older messages and removes them from history.
// It uses fast programmatic extraction (no LLM call) to build the summary,
// then keeps only recent messages.
// Returns the number of messages compacted and any error.
func CompactHistory(history *HistoryDB, ctxMgr *ContextManager, keepRecent int) (int, error) {

	history.mu.Lock()
	msgCount := len(history.Data.Messages)
	history.mu.Unlock()

	if msgCount <= keepRecent {
		return 0, nil
	}

	// Split messages: older ones to summarize, recent ones to keep
	history.mu.Lock()
	toSummarize := make([]HistoryMessage, msgCount-keepRecent)
	copy(toSummarize, history.Data.Messages[:msgCount-keepRecent])
	history.mu.Unlock()

	existingSummary := ctxMgr.GetSummary()
	summary := extractSummary(existingSummary, toSummarize)

	if summary == "" {
		return 0, nil
	}

	// Update the summary
	ctxMgr.SetSummary(summary, len(toSummarize))

	// Remove summarized messages from history, keeping recent ones
	history.mu.Lock()
	currentCount := len(history.Data.Messages)
	removeCount := currentCount - keepRecent
	if removeCount > 0 {
		history.Data.Messages = history.Data.Messages[removeCount:]
	}
	history.mu.Unlock()
	history.Save()

	return len(toSummarize), nil
}
