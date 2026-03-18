package main

import (
	"sync"
	"testing"
	"time"
)

// TestNewSessionManager tests the creation of a new session manager
func TestNewSessionManager(t *testing.T) {
	sm := NewSessionManager()
	
	if sm == nil {
		t.Fatal("Expected non-nil SessionManager")
	}
	if sm.sessions == nil {
		t.Error("Expected sessions map to be initialized")
	}
	if len(sm.sessions) != 0 {
		t.Errorf("Expected empty sessions map, got %d entries", len(sm.sessions))
	}
	if sm.ttl != 24*time.Hour {
		t.Errorf("Expected TTL of 24 hours, got %v", sm.ttl)
	}
}

// TestCreateSession tests session creation with auto-generated ID
func TestCreateSession(t *testing.T) {
	sm := NewSessionManager()
	session := sm.CreateSession("")
	
	if session == nil {
		t.Fatal("Expected non-nil session")
	}
	if session.ID == "" {
		t.Error("Expected non-empty session ID")
	}
	if !session.Active {
		t.Error("Expected session to be active")
	}
	if len(session.State) != 0 {
		t.Error("Expected empty state map")
	}
	if len(session.Results) != 0 {
		t.Error("Expected empty results slice")
	}
	if len(session.Metadata) != 0 {
		t.Error("Expected empty metadata map")
	}
	
	// Verify session was stored in manager
	sm.mu.RLock()
	stored, exists := sm.sessions[session.ID]
	sm.mu.RUnlock()
	
	if !exists {
		t.Error("Expected session to be stored in manager")
	}
	if stored != session {
		t.Error("Expected stored session to match created session")
	}
}

// TestCreateSessionWithID tests session creation with custom ID
func TestCreateSessionWithID(t *testing.T) {
	sm := NewSessionManager()
	customID := "test-session-123"
	session := sm.CreateSession(customID)
	
	if session.ID != customID {
		t.Errorf("Expected session ID to be %q, got %q", customID, session.ID)
	}
}

// TestGetSession tests retrieving an existing session
func TestGetSession(t *testing.T) {
	sm := NewSessionManager()
	session := sm.CreateSession("test-session")
	
	retrieved := sm.GetSession("test-session")
	if retrieved == nil {
		t.Fatal("Expected non-nil retrieved session")
	}
	if retrieved.ID != session.ID {
		t.Error("Expected retrieved session ID to match created session")
	}
	if retrieved != session {
		t.Error("Expected retrieved session to be the same instance")
	}
}

// TestGetSessionNotFound tests retrieving a non-existent session
func TestGetSessionNotFound(t *testing.T) {
	sm := NewSessionManager()
	retrieved := sm.GetSession("non-existent-session")
	
	if retrieved != nil {
		t.Error("Expected nil for non-existent session")
	}
}

// TestGetSessionInactive tests retrieving an inactive session
func TestGetSessionInactive(t *testing.T) {
	sm := NewSessionManager()
	session := sm.CreateSession("test-session")
	session.Active = false
	
	retrieved := sm.GetSession("test-session")
	if retrieved != nil {
		t.Error("Expected nil for inactive session")
	}
}

// TestGenerateSessionID tests the session ID generation function
func TestGenerateSessionID(t *testing.T) {
	id1 := generateSessionID()
	id2 := generateSessionID()
	
	if id1 == "" {
		t.Error("Expected non-empty session ID")
	}
	if len(id1) < 32 {
		t.Errorf("Expected session ID length of at least 32, got %d", len(id1))
	}
	if id1 == id2 {
		t.Error("Expected different session IDs on successive calls")
	}
	
	// Verify it starts with "session_" prefix
	if len(id1) <= 8 || id1[:8] != "session_" {
		t.Errorf("Expected session ID to start with 'session_', got %q", id1)
	}
}

// Helper function to test if a string is valid hex
func hexStringToBytes(s string) ([]byte, error) {
	return []byte{}, nil // Stub for testing
}

// TestSessionStateManagement tests setting and getting state values
func TestSessionStateManagement(t *testing.T) {
	sm := NewSessionManager()
	session := sm.CreateSession("")
	
	// Set state values
	session.State["key1"] = "value1"
	session.State["key2"] = 42
	session.State["key3"] = []string{"a", "b", "c"}
	
	if session.State["key1"] != "value1" {
		t.Error("Expected key1 to be 'value1'")
	}
	if session.State["key2"] != 42 {
		t.Error("Expected key2 to be 42")
	}
}

// TestSessionMetadataManagement tests setting and getting metadata values
func TestSessionMetadataManagement(t *testing.T) {
	sm := NewSessionManager()
	session := sm.CreateSession("")
	
	// Set metadata values
	session.Metadata["user"] = "testuser"
	session.Metadata["source"] = "api"
	
	if session.Metadata["user"] != "testuser" {
		t.Error("Expected user metadata to be 'testuser'")
	}
	if session.Metadata["source"] != "api" {
		t.Error("Expected source metadata to be 'api'")
	}
}

// TestSessionResultsStorage tests storing tool results in a session
func TestSessionResultsStorage(t *testing.T) {
	sm := NewSessionManager()
	session := sm.CreateSession("")
	
	result := ToolResult{
		Tool:      "web_search",
		CallIndex: 1,
		Input:     map[string]any{"query": "test"},
		Output:    "Search results",
		Timestamp: time.Now(),
	}
	session.Results = append(session.Results, result)
	
	if len(session.Results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(session.Results))
	}
	if session.Results[0].Tool != "web_search" {
		t.Error("Expected tool to be 'web_search'")
	}
}

// TestSessionErrorRecording tests recording errors in tool results
func TestSessionErrorRecording(t *testing.T) {
	sm := NewSessionManager()
	session := sm.CreateSession("")
	
	result := ToolResult{
		Tool:      "file_read",
		CallIndex: 2,
		Input:     map[string]any{"path": "/nonexistent"},
		Error:     "file not found",
		Timestamp: time.Now(),
	}
	session.Results = append(session.Results, result)
	
	if session.Results[0].Error != "file not found" {
		t.Error("Expected error to be recorded")
	}
}

// TestConcurrentSessionAccess tests thread-safe concurrent access to sessions
func TestConcurrentSessionAccess(t *testing.T) {
	sm := NewSessionManager()
	
	var wg sync.WaitGroup
	numGoroutines := 100
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			
			sessionID := "test"
			if index%2 == 0 {
				// Create sessions with unique IDs
				sessionID = "concurrent-session-" + string(rune('A'+index%26))
				sm.CreateSession(sessionID)
			} else {
				// Try to retrieve the session
				sm.GetSession("concurrent-session-A")
			}
		}(i)
	}
	
	wg.Wait()
	
	// Verify no panic occurred and sessions are still accessible
	session := sm.GetSession("concurrent-session-A")
	if session == nil {
		t.Error("Expected session to exist after concurrent access")
	}
}
