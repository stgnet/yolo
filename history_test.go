package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHistoryManagerSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	hm := NewHistoryManager(dir)

	hm.AddMessage("user", "hello", nil)
	hm.AddMessage("assistant", "hi there", nil)

	if err := hm.Save(); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(filepath.Join(dir, "history.json")); err != nil {
		t.Fatalf("history.json not created: %v", err)
	}

	// Load into a new manager
	hm2 := NewHistoryManager(dir)
	if !hm2.Load() {
		t.Fatal("Load() returned false")
	}

	if len(hm2.Data.Messages) != 2 {
		t.Fatalf("Expected 2 messages after load, got %d", len(hm2.Data.Messages))
	}
	if hm2.Data.Messages[0].Role != "user" || hm2.Data.Messages[0].Content != "hello" {
		t.Errorf("Message 0 mismatch: %+v", hm2.Data.Messages[0])
	}
	if hm2.Data.Messages[1].Role != "assistant" || hm2.Data.Messages[1].Content != "hi there" {
		t.Errorf("Message 1 mismatch: %+v", hm2.Data.Messages[1])
	}
}

func TestHistoryManagerLoadMissing(t *testing.T) {
	dir := t.TempDir()
	hm := NewHistoryManager(dir)
	if hm.Load() {
		t.Error("Load() should return false for missing file")
	}
}

func TestHistoryManagerLoadCorrupt(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "history.json"), []byte("not json{{{"), 0o644)

	hm := NewHistoryManager(dir)
	if hm.Load() {
		t.Error("Load() should return false for corrupt file")
	}
	// Should reset to empty
	if len(hm.Data.Messages) != 0 {
		t.Errorf("Expected empty messages after corrupt load, got %d", len(hm.Data.Messages))
	}
}

func TestHistoryManagerAddEvolution(t *testing.T) {
	dir := t.TempDir()
	hm := NewHistoryManager(dir)

	hm.AddEvolution("test_action", "test description")

	if len(hm.Data.EvolutionLog) != 1 {
		t.Fatalf("Expected 1 evolution entry, got %d", len(hm.Data.EvolutionLog))
	}
	entry := hm.Data.EvolutionLog[0]
	if entry.Action != "test_action" || entry.Detail != "test description" {
		t.Errorf("Evolution entry mismatch: %+v", entry)
	}
	if entry.TS == "" {
		t.Error("Evolution entry should have a timestamp")
	}
}

func TestHistoryManagerGetContextMessages(t *testing.T) {
	dir := t.TempDir()
	hm := NewHistoryManager(dir)

	hm.AddMessage("user", "msg1", nil)
	hm.AddMessage("assistant", "msg2", nil)
	hm.AddMessage("tool", "tool result", nil)
	hm.AddMessage("system", "sys msg", nil)
	hm.AddMessage("user", "msg5", nil)

	// Get all
	msgs := hm.GetContextMessages(10)
	if len(msgs) != 5 {
		t.Fatalf("Expected 5 context messages, got %d", len(msgs))
	}
	// user/assistant pass through
	if msgs[0].Role != "user" || msgs[0].Content != "msg1" {
		t.Errorf("msg[0] mismatch: %+v", msgs[0])
	}
	if msgs[1].Role != "assistant" || msgs[1].Content != "msg2" {
		t.Errorf("msg[1] mismatch: %+v", msgs[1])
	}
	// tool -> user with prefix
	if msgs[2].Role != "user" || msgs[2].Content != "[Tool result]\ntool result" {
		t.Errorf("tool msg mismatch: %+v", msgs[2])
	}
	// system -> user with prefix
	if msgs[3].Role != "user" || msgs[3].Content != "[SYSTEM] sys msg" {
		t.Errorf("system msg mismatch: %+v", msgs[3])
	}

	// Test truncation to max
	msgs = hm.GetContextMessages(2)
	if len(msgs) != 2 {
		t.Fatalf("Expected 2 context messages with max=2, got %d", len(msgs))
	}
	// Should be the last 2 messages
	if msgs[0].Content != "[SYSTEM] sys msg" {
		t.Errorf("Expected system msg as first of last 2, got: %+v", msgs[0])
	}
	if msgs[1].Content != "msg5" {
		t.Errorf("Expected msg5 as second of last 2, got: %+v", msgs[1])
	}
}

func TestHistoryManagerGetSetModel(t *testing.T) {
	dir := t.TempDir()
	hm := NewHistoryManager(dir)

	if hm.GetModel() != "" {
		t.Errorf("Expected empty model initially, got %q", hm.GetModel())
	}

	hm.SetModel("llama3")
	if hm.GetModel() != "llama3" {
		t.Errorf("Expected model 'llama3', got %q", hm.GetModel())
	}
}

func TestHistoryManagerGetLastN(t *testing.T) {
	dir := t.TempDir()
	hm := NewHistoryManager(dir)

	hm.AddMessage("user", "msg1", nil)
	hm.AddMessage("user", "msg2", nil)
	hm.AddMessage("user", "msg3", nil)

	// Get last 2
	msgs := hm.GetLastN(2)
	if len(msgs) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Content != "msg2" || msgs[1].Content != "msg3" {
		t.Errorf("GetLastN returned wrong messages: %+v", msgs)
	}

	// Get more than available
	msgs = hm.GetLastN(10)
	if len(msgs) != 3 {
		t.Fatalf("Expected 3 messages when requesting 10, got %d", len(msgs))
	}

	// Get 0
	msgs = hm.GetLastN(0)
	if len(msgs) != 0 {
		t.Fatalf("Expected 0 messages for GetLastN(0), got %d", len(msgs))
	}
}

func TestHistoryManagerAddMessageMeta(t *testing.T) {
	dir := t.TempDir()
	hm := NewHistoryManager(dir)

	meta := map[string]any{"key": "value", "count": 42.0}
	hm.AddMessage("user", "with meta", meta)

	if len(hm.Data.Messages) != 1 {
		t.Fatal("Expected 1 message")
	}
	msg := hm.Data.Messages[0]
	if msg.Meta["key"] != "value" {
		t.Errorf("Meta key mismatch: %v", msg.Meta)
	}

	// Verify meta persists through save/load
	hm.Save()
	hm2 := NewHistoryManager(dir)
	hm2.Load()
	if hm2.Data.Messages[0].Meta["key"] != "value" {
		t.Error("Meta not preserved through save/load")
	}
}

func TestHistoryManagerSaveCreatesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nonexistent", "subdir")
	hm := NewHistoryManager(dir)
	hm.AddMessage("user", "test", nil)

	if err := hm.Save(); err != nil {
		t.Fatalf("Save() should create directories: %v", err)
	}

	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("Directory not created: %v", err)
	}
}

func TestHistoryManagerEmpty(t *testing.T) {
	dir := t.TempDir()
	hm := NewHistoryManager(dir)

	if hm.Data.Version != 1 {
		t.Errorf("Expected version 1, got %d", hm.Data.Version)
	}
	if hm.Data.Config.Created == "" {
		t.Error("Expected non-empty Created timestamp")
	}
	if hm.Data.Messages == nil {
		t.Error("Messages should be initialized, not nil")
	}
	if hm.Data.EvolutionLog == nil {
		t.Error("EvolutionLog should be initialized, not nil")
	}
}
