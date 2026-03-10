package main

import (
	"errors"

	"testing"
	"time"
)

var ErrFileNotFound = errors.New("file not found")

func TestNewSessionManager(t *testing.T) {
	sm := NewSessionManager()
	if sm == nil {
		t.Fatal("expected non-nil SessionManager")
	}
	if sm.sessions == nil {
		t.Error("expected sessions map to be initialized")
	}
	if sm.ttl != 24*time.Hour {
		t.Errorf("expected default TTL of 24h, got %v", sm.ttl)
	}
}

func TestCreateSession(t *testing.T) {
	sm := NewSessionManager()
	session := sm.CreateSession("")

	if session == nil {
		t.Fatal("expected non-nil session")
	}

	if session.ID == "" {
		t.Error("expected auto-generated session ID")
	}

	if !session.Active {
		t.Error("expected session to be active")
	}

	if session.State == nil {
		t.Error("expected State map to be initialized")
	}

	if len(session.Results) != 0 {
		t.Error("expected empty Results slice")
	}

	if _, exists := sm.sessions[session.ID]; !exists {
		t.Error("expected session to be stored in manager")
	}
}

func TestCreateSessionWithID(t *testing.T) {
	sm := NewSessionManager()
	customID := "my-custom-session-123"
	session := sm.CreateSession(customID)

	if session.ID != customID {
		t.Errorf("expected session ID %q, got %q", customID, session.ID)
	}
}

func TestGetSession(t *testing.T) {
	sm := NewSessionManager()
	session := sm.CreateSession("test-session")

	retrieved := sm.GetSession("test-session")
	if retrieved == nil {
		t.Fatal("expected to retrieve existing session")
	}

	if retrieved.ID != session.ID {
		t.Error("expected same session ID")
	}
}

func TestGetNonExistentSession(t *testing.T) {
	sm := NewSessionManager()
	session := sm.GetSession("non-existent-id")

	if session != nil {
		t.Error("expected nil for non-existent session")
	}
}

func TestSessionStateManagement(t *testing.T) {
	sm := NewSessionManager()
	session := sm.CreateSession("")

	// Test SetState and GetState
	session.SetState("key1", "value1")
	session.SetState("key2", 42)
	session.SetState("key3", []string{"a", "b", "c"})

	if val, ok := session.GetState("key1"); !ok || val != "value1" {
		t.Errorf("expected value1, got %v", val)
	}

	if val, ok := session.GetState("key2"); !ok || val != 42 {
		t.Errorf("expected 42, got %v", val)
	}

	if _, ok := session.GetState("nonexistent"); ok {
		t.Error("expected false for nonexistent key")
	}
}

func TestSessionMetadata(t *testing.T) {
	sm := NewSessionManager()
	session := sm.CreateSession("")

	session.SetMetadata("author", "YOLO Agent")
	session.SetMetadata("version", "1.0.0")

	if val, ok := session.GetMetadata("author"); !ok || val != "YOLO Agent" {
		t.Errorf("expected 'YOLO Agent', got %v", val)
	}

	if _, ok := session.GetMetadata("nonexistent"); ok {
		t.Error("expected false for nonexistent metadata key")
	}
}

func TestSaveResult(t *testing.T) {
	sm := NewSessionManager()
	session := sm.CreateSession("")

	// Save a successful result
	session.SaveResult("web_search", map[string]any{"query": "test"}, "Search results", nil)

	// Save a failed result
	session.SaveResult("read_file", map[string]any{"path": "test.txt"}, nil, ErrFileNotFound)

	results := session.GetAllResults()
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	if results[0].Tool != "web_search" {
		t.Errorf("expected tool 'web_search', got %q", results[0].Tool)
	}

	if results[0].CallIndex != 0 {
		t.Errorf("expected call index 0, got %d", results[0].CallIndex)
	}

	if results[1].Error == "" {
		t.Error("expected error message for failed tool")
	}
}

func TestGetLastResult(t *testing.T) {
	sm := NewSessionManager()
	session := sm.CreateSession("")

	session.SaveResult("web_search", map[string]any{"query": "first"}, nil, ErrFileNotFound)
	session.SaveResult("web_search", map[string]any{"query": "second"}, "Results for second", nil)
	session.SaveResult("read_file", map[string]any{"path": "test.txt"}, "File content", nil)

	// Get last successful web_search result
	result := session.GetLastResult("web_search")
	if result == nil {
		t.Fatal("expected last successful web_search result")
	}

	if result.CallIndex != 1 {
		t.Errorf("expected call index 1, got %d", result.CallIndex)
	}

	// Get non-existent tool result
	result = session.GetLastResult("nonexistent_tool")
	if result != nil {
		t.Error("expected nil for nonexistent tool")
	}
}

func TestSessionSerialization(t *testing.T) {
	sm := NewSessionManager()
	session := sm.CreateSession("")

	session.SetState("test_key", "test_value")
	session.SetMetadata("meta_key", "meta_value")
	session.SaveResult("test_tool", "input123", "output456", nil)

	data, err := session.Serialize()
	if err != nil {
		t.Fatalf("expected no error during serialization: %v", err)
	}

	deserialized, err := DeserializeSession(data)
	if err != nil {
		t.Fatalf("expected no error during deserialization: %v", err)
	}

	if deserialized.ID != session.ID {
		t.Error("session ID mismatch after deserialization")
	}

	if val, ok := deserialized.GetState("test_key"); !ok || val != "test_value" {
		t.Error("state not preserved after deserialization")
	}

	if len(deserialized.GetAllResults()) != 1 {
		t.Error("results not preserved after deserialization")
	}
}

func TestSessionExpiration(t *testing.T) {
	sm := NewSessionManager()
	sm.ttl = 100 * time.Millisecond // Short TTL for testing
	session := sm.CreateSession("")

	if session == nil || session.ID == "" {
		t.Fatal("failed to create session")
	}

	// Session should exist immediately
	if sm.GetSession(session.ID) == nil {
		t.Error("expected session to exist")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Session should be expired now
	if sm.GetSession(session.ID) != nil {
		t.Error("expected session to be expired")
	}
}

func TestCleanupExpired(t *testing.T) {
	sm := NewSessionManager()
	sm.ttl = 50 * time.Millisecond

	// Create sessions with different ages
	_ = sm.CreateSession("")
	time.Sleep(10 * time.Millisecond)
	_ = sm.CreateSession("")
	time.Sleep(10 * time.Millisecond)
	_ = sm.CreateSession("")

	time.Sleep(80 * time.Millisecond)

	// All sessions should be expired now
	cleaned := sm.CleanupExpired()
	if cleaned != 3 {
		t.Errorf("expected to clean up 3 sessions, got %d", cleaned)
	}

	if sm.SessionCount() != 0 {
		t.Errorf("expected 0 active sessions after cleanup, got %d", sm.SessionCount())
	}
}

func TestListActiveSessions(t *testing.T) {
	sm := NewSessionManager()
	session1 := sm.CreateSession("session-1")
	_ = sm.CreateSession("session-2")

	active := sm.ListActiveSessions()
	if len(active) != 2 {
		t.Errorf("expected 2 active sessions, got %d", len(active))
	}

	// Manually expire one session
	sm.mu.Lock()
	session1.Active = false
	sm.mu.Unlock()

	active = sm.ListActiveSessions()
	if len(active) != 1 {
		t.Errorf("expected 1 active session after marking one as inactive, got %d", len(active))
	}
}

func TestSessionCount(t *testing.T) {
	sm := NewSessionManager()

	if sm.SessionCount() != 0 {
		t.Error("expected 0 sessions initially")
	}

	sm.CreateSession("session-1")
	if sm.SessionCount() != 1 {
		t.Error("expected 1 session after creation")
	}

	sm.CreateSession("session-2")
	if sm.SessionCount() != 2 {
		t.Error("expected 2 sessions after second creation")
	}
}

func TestSessionLastActivityUpdate(t *testing.T) {
	sm := NewSessionManager()
	session := sm.CreateSession("")

	initialActivity := session.LastActivity
	time.Sleep(10 * time.Millisecond)

	// Trigger activity update by getting the session
	sm.GetSession(session.ID)

	if session.LastActivity.Before(initialActivity) || session.LastActivity.Equal(initialActivity) {
		t.Error("expected LastActivity to be updated after GetSession")
	}
}
