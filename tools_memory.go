package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// ─── Memory Manager ──────────────────────────────────────────────────
//
// MemoryManager implements a tiered memory system:
//   - MEMORY.md: curated durable facts (~100 lines), always loaded into system prompt
//   - memory/YYYY-MM-DD.md: daily context logs with raw observations
//   - knowledge.md: legacy knowledge base (still loaded if present)
//
// The agent actively maintains memory via tools. Daily logs are
// automatically loaded (today + yesterday) and can be promoted
// into MEMORY.md via the memory_promote tool.

const (
	MaxMemoryLines = 100 // soft cap for MEMORY.md
	DailyLogDays   = 2  // how many days of daily logs to auto-load (today + yesterday)
)

// MemoryManager owns the .yolo/MEMORY.md and .yolo/memory/ directory.
type MemoryManager struct {
	yoloDir   string // path to .yolo/
	memoryDir string // path to .yolo/memory/
	mu        sync.RWMutex
}

// NewMemoryManager creates a manager rooted in the given .yolo directory.
func NewMemoryManager(yoloDir string) *MemoryManager {
	return &MemoryManager{
		yoloDir:   yoloDir,
		memoryDir: filepath.Join(yoloDir, "memory"),
	}
}

// memoryPath returns the path to MEMORY.md.
func (m *MemoryManager) memoryPath() string {
	return filepath.Join(m.yoloDir, "MEMORY.md")
}

// dailyLogPath returns the path to a daily log file for the given date.
func (m *MemoryManager) dailyLogPath(date time.Time) string {
	return filepath.Join(m.memoryDir, date.Format("2006-01-02")+".md")
}

// ─── Read Operations ─────────────────────────────────────────────────

// ReadMemory returns the contents of MEMORY.md, or empty string if not found.
func (m *MemoryManager) ReadMemory() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	data, err := os.ReadFile(m.memoryPath())
	if err != nil {
		return ""
	}
	return string(data)
}

// ReadDailyLog returns the contents of a specific daily log.
func (m *MemoryManager) ReadDailyLog(date time.Time) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	data, err := os.ReadFile(m.dailyLogPath(date))
	if err != nil {
		return ""
	}
	return string(data)
}

// GetRecentDailyLogs returns the content of today's and yesterday's logs.
func (m *MemoryManager) GetRecentDailyLogs() string {
	now := time.Now()
	var sections []string
	for i := DailyLogDays - 1; i >= 0; i-- {
		date := now.AddDate(0, 0, -i)
		content := m.ReadDailyLog(date)
		if content != "" {
			header := fmt.Sprintf("### %s", date.Format("2006-01-02"))
			if i == 0 {
				header += " (today)"
			} else if i == 1 {
				header += " (yesterday)"
			}
			sections = append(sections, header+"\n"+content)
		}
	}
	if len(sections) == 0 {
		return ""
	}
	return strings.Join(sections, "\n\n")
}

// ListDailyLogs returns a sorted list of all daily log dates.
func (m *MemoryManager) ListDailyLogs() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	entries, err := os.ReadDir(m.memoryDir)
	if err != nil {
		return nil
	}
	var dates []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".md") && len(name) == 13 && isValidDateFilename(name) {
			dates = append(dates, strings.TrimSuffix(name, ".md"))
		}
	}
	sort.Strings(dates)
	return dates
}

// isValidDateFilename checks if a filename like "2006-01-02.md" has a valid date.
func isValidDateFilename(name string) bool {
	dateStr := strings.TrimSuffix(name, ".md")
	_, err := time.Parse("2006-01-02", dateStr)
	return err == nil
}

// GetSystemPromptContext returns the memory sections to inject into the system prompt.
// This includes MEMORY.md and recent daily logs.
func (m *MemoryManager) GetSystemPromptContext() string {
	var sections []string

	// Always load MEMORY.md
	memory := m.ReadMemory()
	if memory != "" {
		sections = append(sections, "## Agent Memory\nDurable facts and preferences you have learned. Update via memory_write tool.\n\n"+memory)
	}

	// Load recent daily logs
	dailyLogs := m.GetRecentDailyLogs()
	if dailyLogs != "" {
		sections = append(sections, "## Recent Context\nDaily observations from recent sessions:\n\n"+dailyLogs)
	}

	if len(sections) == 0 {
		return ""
	}
	return "\n" + strings.Join(sections, "\n\n")
}

// ─── Write Operations ────────────────────────────────────────────────

// WriteMemory replaces the contents of MEMORY.md. Enforces the line cap.
func (m *MemoryManager) WriteMemory(content string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	lines := strings.Split(content, "\n")
	if len(lines) > MaxMemoryLines {
		return fmt.Errorf("MEMORY.md exceeds %d line cap (%d lines). Trim less important entries first", MaxMemoryLines, len(lines))
	}

	if err := os.MkdirAll(m.yoloDir, 0o755); err != nil {
		return fmt.Errorf("create yolo dir: %w", err)
	}
	tmp := m.memoryPath() + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write memory: %w", err)
	}
	return os.Rename(tmp, m.memoryPath())
}

// AppendDailyLog appends an entry to today's daily log.
func (m *MemoryManager) AppendDailyLog(entry string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := os.MkdirAll(m.memoryDir, 0o755); err != nil {
		return fmt.Errorf("create memory dir: %w", err)
	}

	logPath := m.dailyLogPath(time.Now())

	// Read existing content
	existing, _ := os.ReadFile(logPath)

	// Add timestamp prefix and append
	ts := time.Now().Format("15:04")
	line := fmt.Sprintf("- [%s] %s\n", ts, entry)

	var content string
	if len(existing) > 0 {
		content = string(existing) + line
	} else {
		content = line
	}

	tmp := logPath + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write daily log: %w", err)
	}
	return os.Rename(tmp, logPath)
}

// GetPromotionCandidates reads daily logs from the last N days (excluding today)
// and returns them formatted for the LLM to distill into MEMORY.md.
func (m *MemoryManager) GetPromotionCandidates(days int) string {
	now := time.Now()
	var sections []string

	for i := 1; i <= days; i++ {
		date := now.AddDate(0, 0, -i)
		content := m.ReadDailyLog(date)
		if content != "" {
			sections = append(sections, fmt.Sprintf("### %s\n%s", date.Format("2006-01-02"), content))
		}
	}

	if len(sections) == 0 {
		return ""
	}
	return strings.Join(sections, "\n\n")
}

// MemoryLineCount returns the current line count of MEMORY.md.
func (m *MemoryManager) MemoryLineCount() int {
	content := m.ReadMemory()
	if content == "" {
		return 0
	}
	return len(strings.Split(content, "\n"))
}

// ─── Tool Implementations ────────────────────────────────────────────

// memoryRead reads and returns the contents of MEMORY.md.
func (e *ToolExecutor) memoryRead(args map[string]any) string {
	mm := e.agent.memory
	if mm == nil {
		return errorMessage("memory manager not initialized")
	}
	content := mm.ReadMemory()
	if content == "" {
		return "MEMORY.md is empty or does not exist. Use memory_write to create it with durable facts."
	}
	lines := mm.MemoryLineCount()
	return fmt.Sprintf("MEMORY.md (%d/%d lines):\n\n%s", lines, MaxMemoryLines, content)
}

// memoryWrite replaces the contents of MEMORY.md with curated durable facts.
func (e *ToolExecutor) memoryWrite(args map[string]any) string {
	mm := e.agent.memory
	if mm == nil {
		return errorMessage("memory manager not initialized")
	}
	content := getStringArg(args, "content", "")
	if content == "" {
		return errorMessage("content is required")
	}
	if err := mm.WriteMemory(content); err != nil {
		return errorMessage("failed to write memory: %v", err)
	}
	lines := mm.MemoryLineCount()
	return fmt.Sprintf("MEMORY.md updated (%d/%d lines)", lines, MaxMemoryLines)
}

// memoryLog appends an observation to today's daily context log.
func (e *ToolExecutor) memoryLog(args map[string]any) string {
	mm := e.agent.memory
	if mm == nil {
		return errorMessage("memory manager not initialized")
	}
	entry := getStringArg(args, "entry", "")
	if entry == "" {
		return errorMessage("entry is required")
	}
	if err := mm.AppendDailyLog(entry); err != nil {
		return errorMessage("failed to log: %v", err)
	}
	return fmt.Sprintf("Logged to daily context (%s)", time.Now().Format("2006-01-02"))
}

// memoryPromote returns daily log entries for distillation into MEMORY.md.
func (e *ToolExecutor) memoryPromote(args map[string]any) string {
	mm := e.agent.memory
	if mm == nil {
		return errorMessage("memory manager not initialized")
	}
	days := getIntArg(args, "days", 7)
	if days < 1 {
		days = 1
	}
	if days > 30 {
		days = 30
	}

	candidates := mm.GetPromotionCandidates(days)
	if candidates == "" {
		return "No daily logs found for promotion."
	}

	currentMemory := mm.ReadMemory()
	lines := mm.MemoryLineCount()

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Current MEMORY.md (%d/%d lines):\n", lines, MaxMemoryLines))
	if currentMemory != "" {
		result.WriteString(currentMemory)
	} else {
		result.WriteString("(empty)")
	}
	result.WriteString(fmt.Sprintf("\n\n---\nDaily logs from last %d days to review:\n\n", days))
	result.WriteString(candidates)
	result.WriteString("\n\n---\nInstructions: Review the daily logs above. Distill important, durable facts into MEMORY.md using memory_write. ")
	result.WriteString(fmt.Sprintf("Keep it under %d lines. Preserve existing important entries. Remove stale or one-time observations.", MaxMemoryLines))

	return result.String()
}

// memorySearch searches across all memory files for a pattern.
func (e *ToolExecutor) memorySearch(args map[string]any) string {
	mm := e.agent.memory
	if mm == nil {
		return errorMessage("memory manager not initialized")
	}
	query := strings.ToLower(getStringArg(args, "query", ""))
	if query == "" {
		return errorMessage("query is required")
	}

	var results []string

	// Search MEMORY.md
	memory := mm.ReadMemory()
	if memory != "" {
		for _, line := range strings.Split(memory, "\n") {
			if strings.Contains(strings.ToLower(line), query) {
				results = append(results, fmt.Sprintf("[MEMORY.md] %s", line))
			}
		}
	}

	// Search daily logs
	dates := mm.ListDailyLogs()
	for _, dateStr := range dates {
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}
		content := mm.ReadDailyLog(date)
		for _, line := range strings.Split(content, "\n") {
			if strings.Contains(strings.ToLower(line), query) {
				results = append(results, fmt.Sprintf("[%s] %s", dateStr, line))
			}
		}
	}

	if len(results) == 0 {
		return fmt.Sprintf("No matches found for '%s' in memory.", query)
	}
	return fmt.Sprintf("Found %d matches for '%s':\n%s", len(results), query, strings.Join(results, "\n"))
}
