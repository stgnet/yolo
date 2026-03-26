package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContextManager_NewEmpty(t *testing.T) {
	dir := t.TempDir()
	cm := NewContextManager(dir)

	assert.Equal(t, "", cm.GetSummary())
	assert.Equal(t, 0, cm.Data.Compactions)
}

func TestContextManager_SetAndGetSummary(t *testing.T) {
	dir := t.TempDir()
	cm := NewContextManager(dir)

	cm.SetSummary("The user worked on refactoring the auth module.", 10)

	assert.Equal(t, "The user worked on refactoring the auth module.", cm.GetSummary())
	assert.Equal(t, 10, cm.Data.MessageCount)
	assert.Equal(t, 1, cm.Data.Compactions)
	assert.NotEmpty(t, cm.Data.LastCompacted)
}

func TestContextManager_SummaryTruncation(t *testing.T) {
	dir := t.TempDir()
	cm := NewContextManager(dir)

	// Create a summary that exceeds the max length
	longSummary := make([]byte, SummaryMaxLength+500)
	for i := range longSummary {
		longSummary[i] = 'a'
	}
	cm.SetSummary(string(longSummary), 5)

	assert.LessOrEqual(t, len(cm.GetSummary()), SummaryMaxLength+20) // +20 for truncation marker
	assert.Contains(t, cm.GetSummary(), "...(truncated)")
}

func TestContextManager_Persistence(t *testing.T) {
	dir := t.TempDir()
	cm := NewContextManager(dir)
	cm.SetSummary("persistent summary", 5)

	// Create a new manager that should load the saved data
	cm2 := NewContextManager(dir)
	assert.Equal(t, "persistent summary", cm2.GetSummary())
	assert.Equal(t, 5, cm2.Data.MessageCount)
	assert.Equal(t, 1, cm2.Data.Compactions)
}

func TestContextManager_ClearSummary(t *testing.T) {
	dir := t.TempDir()
	cm := NewContextManager(dir)
	cm.SetSummary("something", 3)
	assert.NotEmpty(t, cm.GetSummary())

	cm.ClearSummary()
	assert.Equal(t, "", cm.GetSummary())
	assert.Equal(t, 0, cm.Data.Compactions)
}

func TestContextManager_NeedsCompaction(t *testing.T) {
	dir := t.TempDir()
	cm := NewContextManager(dir)

	// With limit 40, threshold at 75% = 30
	assert.False(t, cm.NeedsCompaction(20, 40))
	assert.False(t, cm.NeedsCompaction(29, 40))
	assert.True(t, cm.NeedsCompaction(30, 40))
	assert.True(t, cm.NeedsCompaction(40, 40))
	assert.True(t, cm.NeedsCompaction(50, 40))
}

func TestContextManager_GetStatus(t *testing.T) {
	dir := t.TempDir()
	cm := NewContextManager(dir)

	// Empty status
	status := cm.GetStatus()
	assert.Contains(t, status, "No conversation summary")

	// With summary
	cm.SetSummary("test summary", 15)
	status = cm.GetStatus()
	assert.Contains(t, status, "Compactions: 1")
	assert.Contains(t, status, "Messages summarized: 15")
}

func TestContextManager_LoadCorruptFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "context_summary.json"), []byte("not json"), 0o644))

	cm := NewContextManager(dir)
	// Should not crash, summary should be empty
	assert.Equal(t, "", cm.GetSummary())
}

func TestDynamicContextLimit(t *testing.T) {
	// Zero/negative returns default
	assert.Equal(t, MaxContextMessages, DynamicContextLimit(0))
	assert.Equal(t, MaxContextMessages, DynamicContextLimit(-1))

	// Small model context
	limit := DynamicContextLimit(4096)
	assert.GreaterOrEqual(t, limit, MinContextMessages)

	// Large model context
	limit = DynamicContextLimit(128000)
	assert.LessOrEqual(t, limit, MaxDynamicContextMessages)

	// Medium model context (8192 tokens)
	// 8192 * 0.5 / 150 ≈ 27
	limit = DynamicContextLimit(8192)
	assert.GreaterOrEqual(t, limit, 20)
	assert.LessOrEqual(t, limit, 40)
}

func TestDynamicContextLimit_Scaling(t *testing.T) {
	// Bigger context window should give more messages
	small := DynamicContextLimit(4096)
	medium := DynamicContextLimit(16384)
	large := DynamicContextLimit(65536)

	assert.LessOrEqual(t, small, medium)
	assert.LessOrEqual(t, medium, large)
}

func TestExtractSummary(t *testing.T) {
	messages := []HistoryMessage{
		{Role: "user", Content: "Fix the login bug"},
		{Role: "assistant", Content: "I found the issue in auth.go line 42"},
	}

	// Without existing summary
	summary := extractSummary("", messages)
	assert.Contains(t, summary, "Fix the login bug")
	assert.Contains(t, summary, "auth.go line 42")

	// With existing summary
	summary = extractSummary("Earlier, the user set up the project.", messages)
	assert.Contains(t, summary, "Earlier, the user set up the project.")
	assert.Contains(t, summary, "Fix the login bug")
}

func TestExtractSummary_SkipsSystemMessages(t *testing.T) {
	messages := []HistoryMessage{
		{Role: "system", Content: "You are an AI assistant"},
		{Role: "user", Content: "Hello"},
	}

	summary := extractSummary("", messages)
	assert.NotContains(t, summary, "AI assistant")
	assert.Contains(t, summary, "Hello")
}

func TestFirstLine(t *testing.T) {
	assert.Equal(t, "hello", firstLine("hello\nworld", 100))
	assert.Equal(t, "hel...", firstLine("hello", 3))
	assert.Equal(t, "short", firstLine("short", 100))
}

func TestCompactHistory_TooFewMessages(t *testing.T) {
	dir := t.TempDir()
	hm := NewHistoryDB(dir)
	cm := NewContextManager(dir)

	// Add fewer messages than keepRecent
	for i := 0; i < 5; i++ {
		hm.AddMessage("user", "msg", nil)
	}

	compacted, err := CompactHistory(hm, cm, CompactionKeepRecent)
	assert.NoError(t, err)
	assert.Equal(t, 0, compacted)
}

func TestCompactHistory_CompactsOldMessages(t *testing.T) {
	dir := t.TempDir()
	hm := NewHistoryDB(dir)
	cm := NewContextManager(dir)

	// Add more messages than keepRecent
	for i := 0; i < 25; i++ {
		hm.AddMessage("user", "test message", nil)
	}

	compacted, err := CompactHistory(hm, cm, 15)
	assert.NoError(t, err)
	assert.Equal(t, 10, compacted)
	assert.Equal(t, 15, len(hm.Data.Messages))
	assert.NotEmpty(t, cm.GetSummary())
}
