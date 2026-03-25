package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryManager_ReadWriteMemory(t *testing.T) {
	dir := t.TempDir()
	mm := NewMemoryManager(dir)

	// Initially empty
	assert.Equal(t, "", mm.ReadMemory())
	assert.Equal(t, 0, mm.MemoryLineCount())

	// Write some content
	content := "# User Preferences\n- Prefers tabs over spaces\n- Uses Go 1.25"
	require.NoError(t, mm.WriteMemory(content))

	// Read it back
	assert.Equal(t, content, mm.ReadMemory())
	assert.Equal(t, 3, mm.MemoryLineCount())
}

func TestMemoryManager_WriteMemory_LineCap(t *testing.T) {
	dir := t.TempDir()
	mm := NewMemoryManager(dir)

	// Build content exceeding the line cap
	lines := make([]string, MaxMemoryLines+5)
	for i := range lines {
		lines[i] = "line"
	}
	content := strings.Join(lines, "\n")

	err := mm.WriteMemory(content)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds")
}

func TestMemoryManager_AppendDailyLog(t *testing.T) {
	dir := t.TempDir()
	mm := NewMemoryManager(dir)

	require.NoError(t, mm.AppendDailyLog("discovered new API endpoint"))
	require.NoError(t, mm.AppendDailyLog("fixed test flake in auth module"))

	content := mm.ReadDailyLog(time.Now())
	assert.Contains(t, content, "discovered new API endpoint")
	assert.Contains(t, content, "fixed test flake in auth module")
	// Check timestamp format
	assert.Contains(t, content, "[")
	assert.Contains(t, content, "]")
}

func TestMemoryManager_ListDailyLogs(t *testing.T) {
	dir := t.TempDir()
	mm := NewMemoryManager(dir)

	// Create memory dir and a couple of log files
	memDir := filepath.Join(dir, "memory")
	require.NoError(t, os.MkdirAll(memDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(memDir, "2025-01-15.md"), []byte("log1"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(memDir, "2025-01-16.md"), []byte("log2"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(memDir, "not-a-date.md"), []byte("skip"), 0o644))

	dates := mm.ListDailyLogs()
	assert.Equal(t, []string{"2025-01-15", "2025-01-16"}, dates)
}

func TestMemoryManager_GetRecentDailyLogs(t *testing.T) {
	dir := t.TempDir()
	mm := NewMemoryManager(dir)

	// Write today's log
	require.NoError(t, mm.AppendDailyLog("today's observation"))

	logs := mm.GetRecentDailyLogs()
	assert.Contains(t, logs, "today's observation")
	assert.Contains(t, logs, "(today)")
}

func TestMemoryManager_GetSystemPromptContext(t *testing.T) {
	dir := t.TempDir()
	mm := NewMemoryManager(dir)

	// Empty state
	assert.Equal(t, "", mm.GetSystemPromptContext())

	// With MEMORY.md
	require.NoError(t, mm.WriteMemory("- User prefers concise output"))
	ctx := mm.GetSystemPromptContext()
	assert.Contains(t, ctx, "Agent Memory")
	assert.Contains(t, ctx, "User prefers concise output")

	// With daily log too
	require.NoError(t, mm.AppendDailyLog("test entry"))
	ctx = mm.GetSystemPromptContext()
	assert.Contains(t, ctx, "Recent Context")
}

func TestMemoryManager_GetPromotionCandidates(t *testing.T) {
	dir := t.TempDir()
	mm := NewMemoryManager(dir)

	// Create a log for yesterday
	memDir := filepath.Join(dir, "memory")
	require.NoError(t, os.MkdirAll(memDir, 0o755))
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	require.NoError(t, os.WriteFile(filepath.Join(memDir, yesterday+".md"), []byte("- [10:00] found a bug\n"), 0o644))

	candidates := mm.GetPromotionCandidates(7)
	assert.Contains(t, candidates, "found a bug")
	assert.Contains(t, candidates, yesterday)
}

func TestMemoryManager_Search(t *testing.T) {
	dir := t.TempDir()
	mm := NewMemoryManager(dir)

	require.NoError(t, mm.WriteMemory("- User likes Go\n- Project uses PostgreSQL"))
	require.NoError(t, mm.AppendDailyLog("migrated to PostgreSQL 16"))

	// Create a ToolExecutor with a mock agent
	agent := &YoloAgent{memory: mm}
	te := &ToolExecutor{agent: agent}

	result := te.memorySearch(map[string]any{"query": "postgresql"})
	assert.Contains(t, result, "PostgreSQL")
	assert.Contains(t, result, "MEMORY.md")
	assert.Contains(t, result, time.Now().Format("2006-01-02"))
}

func TestMemoryTools_ReadWriteLog(t *testing.T) {
	dir := t.TempDir()
	mm := NewMemoryManager(dir)
	agent := &YoloAgent{memory: mm}
	te := &ToolExecutor{agent: agent}

	// Read empty
	result := te.memoryRead(nil)
	assert.Contains(t, result, "empty")

	// Write
	result = te.memoryWrite(map[string]any{"content": "- test fact"})
	assert.Contains(t, result, "updated")

	// Read back
	result = te.memoryRead(nil)
	assert.Contains(t, result, "test fact")

	// Log
	result = te.memoryLog(map[string]any{"entry": "observation"})
	assert.Contains(t, result, "Logged")

	// Promote
	result = te.memoryPromote(nil)
	// No old logs to promote
	assert.Contains(t, result, "No daily logs found")
}

func TestMemoryTools_ErrorCases(t *testing.T) {
	agent := &YoloAgent{memory: nil}
	te := &ToolExecutor{agent: agent}

	assert.Contains(t, te.memoryRead(nil), "Error:")
	assert.Contains(t, te.memoryWrite(map[string]any{"content": "x"}), "Error:")
	assert.Contains(t, te.memoryLog(map[string]any{"entry": "x"}), "Error:")
	assert.Contains(t, te.memoryPromote(nil), "Error:")
	assert.Contains(t, te.memorySearch(map[string]any{"query": "x"}), "Error:")
}
