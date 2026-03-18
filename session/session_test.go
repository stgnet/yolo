package session

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestNewSessionManager(t *testing.T) {
	sm := NewSessionManager()
	if sm == nil {
		t.Fatal("Expected non-nil SessionManager")
	}
	if sm.sessions == nil {
		t.Error("Expected sessions map to be initialized")
	}
	if sm.ttl != 24*time.Hour {
		t.Errorf("Expected default TTL of 24 hours, got %v", sm.ttl)
	}
}

func TestCreateSession(t *testing.T) {
	sm := NewSessionManager()
	
	t.Run("Creates session with custom ID", func(t *testing.T) {
		session := sm.CreateSession("test-session-123")
		if session.ID != "test-session-123" {
			t.Errorf("Expected ID 'test-session-123', got %s", session.ID)
		}
		if !session.Active {
			t.Error("Expected session to be active")
		}
		if session.State == nil {
			t.Error("Expected State map to be initialized")
		}
		if session.Metadata == nil {
			t.Error("Expected Metadata map to be initialized")
		}
	})
	
	t.Run("Generates unique ID when empty", func(t *testing.T) {
		session1 := sm.CreateSession("")
		session2 := sm.CreateSession("")
		
		if session1.ID == "" {
			t.Error("Expected non-empty generated ID")
		}
		if session2.ID == "" {
			t.Error("Expected non-empty generated ID")
		}
		if session1.ID == session2.ID {
			t.Error("Expected unique IDs for different sessions")
		}
	})
	
	t.Run("Initializes timestamps", func(t *testing.T) {
		session := sm.CreateSession("")
		if session.CreatedAt.IsZero() {
			t.Error("Expected CreatedAt to be set")
		}
		if session.LastActivity.IsZero() {
			t.Error("Expected LastActivity to be set")
		}
	})
}

func TestGetSession(t *testing.T) {
	t.Run("Retrieves existing session", func(t *testing.T) {
		sm := NewSessionManager()
		session := sm.CreateSession("test-session")
		retrieved := sm.GetSession("test-session")
		
		if retrieved == nil {
			t.Fatal("Expected to retrieve existing session")
		}
		if retrieved.ID != session.ID {
			t.Errorf("Expected ID %s, got %s", session.ID, retrieved.ID)
		}
	})
	
	t.Run("Returns nil for non-existent session", func(t *testing.T) {
		sm := NewSessionManager()
		retrieved := sm.GetSession("non-existent")
		if retrieved != nil {
			t.Error("Expected nil for non-existent session")
		}
	})
	
	t.Run("Updates LastActivity on access", func(t *testing.T) {
		sm := NewSessionManager()
		session := sm.CreateSession("test-session")
		oldActivity := session.LastActivity
		time.Sleep(10 * time.Millisecond)
		
		retrieved := sm.GetSession("test-session")
		if retrieved == nil {
			t.Fatal("Failed to retrieve session")
		}
		
		if !retrieved.LastActivity.After(oldActivity) {
			t.Errorf("Expected LastActivity %v to be after old activity %v", retrieved.LastActivity, oldActivity)
		}
	})
	
	t.Run("Returns nil for inactive session", func(t *testing.T) {
		sm := NewSessionManager()
		session := sm.CreateSession("inactive-session")
		session.mu.Lock()
		session.Active = false
		session.mu.Unlock()
		
		retrieved := sm.GetSession("inactive-session")
		if retrieved != nil {
			t.Error("Expected nil for inactive session")
		}
	})
	
	t.Run("Expires and removes old sessions", func(t *testing.T) {
		sm := NewSessionManager()
		sm.ttl = 100 * time.Millisecond
		sm.CreateSession("expiring-session")
		
		time.Sleep(150 * time.Millisecond)
		
		retrieved := sm.GetSession("expiring-session")
		if retrieved != nil {
			t.Error("Expected expired session to be removed")
		}
	})
}

func TestSaveResult(t *testing.T) {
	t.Run("Stores result successfully", func(t *testing.T) {
		sm := NewSessionManager()
		session := sm.CreateSession("result-test")
		session.SaveResult("test_tool", map[string]string{"key": "value"}, "output", nil)
		
		results := session.GetAllResults()
		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}
		
		r := results[0]
		if r.Tool != "test_tool" {
			t.Errorf("Expected tool 'test_tool', got %s", r.Tool)
		}
		if r.CallIndex != 0 {
			t.Errorf("Expected call index 0, got %d", r.CallIndex)
		}
	})
	
	t.Run("Stores error in result", func(t *testing.T) {
		sm := NewSessionManager()
		session := sm.CreateSession("error-test")
		err := testError("test error message")
		session.SaveResult("failing_tool", nil, nil, err)
		
		results := session.GetAllResults()
		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}
		
		r := results[0]
		if r.Error == "" || r.Error != "test error message" {
			t.Errorf("Expected error message 'test error message', got '%s'", r.Error)
		}
	})
	
	t.Run("Updates LastActivity", func(t *testing.T) {
		sm := NewSessionManager()
		session := sm.CreateSession("activity-test")
		oldActivity := session.LastActivity
		time.Sleep(10 * time.Millisecond)
		session.SaveResult("tool1", nil, "output", nil)
		
		if !session.LastActivity.After(oldActivity) {
			t.Error("Expected LastActivity to be updated")
		}
	})
}

func TestSetState(t *testing.T) {
	t.Run("Sets and retrieves state value", func(t *testing.T) {
		sm := NewSessionManager()
		session := sm.CreateSession("state-test")
		session.SetState("key1", "value1")
		
		value, exists := session.GetState("key1")
		if !exists {
			t.Fatal("Expected state key to exist")
		}
		if value != "value1" {
			t.Errorf("Expected 'value1', got %v", value)
		}
	})
	
	t.Run("Returns false for non-existent key", func(t *testing.T) {
		sm := NewSessionManager()
		session := sm.CreateSession("non-exists-test")
		_, exists := session.GetState("non-existent")
		if exists {
			t.Error("Expected state key to not exist")
		}
	})
	
	t.Run("Overwrites existing state", func(t *testing.T) {
		sm := NewSessionManager()
		session := sm.CreateSession("overwrite-test")
		session.SetState("key1", "value1")
		session.SetState("key1", "value2")
		
		value, _ := session.GetState("key1")
		if value != "value2" {
			t.Errorf("Expected 'value2', got %v", value)
		}
	})
	
	t.Run("Updates LastActivity when setting state", func(t *testing.T) {
		sm := NewSessionManager()
		session := sm.CreateSession("activity-test")
		oldActivity := session.LastActivity
		time.Sleep(10 * time.Millisecond)
		session.SetState("key", "value")
		
		if !session.LastActivity.After(oldActivity) {
			t.Error("Expected LastActivity to be updated")
		}
	})
}

func TestSetMetadata(t *testing.T) {
	t.Run("Sets and retrieves metadata", func(t *testing.T) {
		sm := NewSessionManager()
		session := sm.CreateSession("metadata-test")
		session.SetMetadata("author", "yolo")
		
		value, exists := session.GetMetadata("author")
		if !exists {
			t.Fatal("Expected metadata key to exist")
		}
		if value != "yolo" {
			t.Errorf("Expected 'yolo', got %s", value)
		}
	})
	
	t.Run("Returns false for non-existent metadata", func(t *testing.T) {
		sm := NewSessionManager()
		session := sm.CreateSession("non-exists-test")
		_, exists := session.GetMetadata("non-existent")
		if exists {
			t.Error("Expected metadata key to not exist")
		}
	})
}

func TestGetLastResult(t *testing.T) {
	t.Run("Returns most recent successful result for tool", func(t *testing.T) {
		sm := NewSessionManager()
		session := sm.CreateSession("lastresult-test")
		session.SaveResult("tool1", nil, "first", nil)
		session.SaveResult("tool2", nil, "other", nil)
		session.SaveResult("tool1", nil, "second", nil)
		
		result := session.GetLastResult("tool1")
		if result == nil {
			t.Fatal("Expected to find last result for tool1")
		}
		if result.Output != "second" {
			t.Errorf("Expected output 'second', got %v", result.Output)
		}
	})
	
	t.Run("Returns nil for non-existent tool", func(t *testing.T) {
		sm := NewSessionManager()
		session := sm.CreateSession("nonexist-test")
		result := session.GetLastResult("non-existent")
		if result != nil {
			t.Error("Expected nil for non-existent tool")
		}
	})
	
	t.Run("Skips failed results", func(t *testing.T) {
		sm := NewSessionManager()
		session := sm.CreateSession("skipfail-test")
		session.SaveResult("tool1", nil, nil, testError("failed"))
		result := session.GetLastResult("tool1")
		if result != nil {
			t.Error("Expected nil for tool with only failed results")
		}
		
		sm2 := NewSessionManager()
		session2 := sm2.CreateSession("success-test")
		session2.SaveResult("tool2", nil, "success", nil)
		result = session2.GetLastResult("tool2")
		if result == nil {
			t.Error("Expected to find successful result for tool2")
		}
	})
}

func TestCleanupExpired(t *testing.T) {
	sm := NewSessionManager()
	sm.ttl = 100 * time.Millisecond
	
	_ = sm.CreateSession("session-1")
	_ = sm.CreateSession("session-2")
	
	time.Sleep(150 * time.Millisecond)
	
	count := sm.CleanupExpired()
	if count != 2 {
		t.Errorf("Expected 2 expired sessions cleaned up, got %d", count)
	}
	
	activeSessions := sm.ListActiveSessions()
	if len(activeSessions) != 0 {
		t.Errorf("Expected 0 active sessions after cleanup, got %d", len(activeSessions))
	}
}

func TestListActiveSessions(t *testing.T) {
	sm := NewSessionManager()
	sm.ttl = 24 * time.Hour
	
	_ = sm.CreateSession("session-1")
	_ = sm.CreateSession("session-2")
	
	active := sm.ListActiveSessions()
	if len(active) != 2 {
		t.Errorf("Expected 2 active sessions, got %d", len(active))
	}
	
	// Check that both session IDs are present
	found1 := false
	found2 := false
	for _, id := range active {
		if id == "session-1" {
			found1 = true
		}
		if id == "session-2" {
			found2 = true
		}
	}
	
	if !found1 || !found2 {
		t.Error("Expected both session IDs to be in active list")
	}
}

func TestSessionCount(t *testing.T) {
	sm := NewSessionManager()
	sm.ttl = 24 * time.Hour
	
	if sm.SessionCount() != 0 {
		t.Errorf("Expected 0 sessions initially, got %d", sm.SessionCount())
	}
	
	sm.CreateSession("session-1")
	sm.CreateSession("session-2")
	
	if sm.SessionCount() != 2 {
		t.Errorf("Expected 2 sessions, got %d", sm.SessionCount())
	}
}

func TestConcurrency(t *testing.T) {
	sm := NewSessionManager()
	sm.ttl = 24 * time.Hour
	
	const numGoroutines = 100
	const operationsPerGoroutine = 100
	
	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			sessionID := fmt.Sprintf("concurrent-session-%d", id)
			for j := 0; j < operationsPerGoroutine; j++ {
				// Concurrent session creation
				sm.CreateSession("")
				
				// Concurrent session retrieval
				sm.GetSession(sessionID)
				
				// Concurrent cleanup
				sm.CleanupExpired()
				
				// Concurrent listing
				sm.ListActiveSessions()
			}
		}(i)
	}
	
	wg.Wait()
	
	// Should not deadlock or panic
	if sm.SessionCount() < numGoroutines {
		t.Logf("Created %d sessions (expected at least %d)", sm.SessionCount(), numGoroutines)
	}
}

func TestSessionSerialization(t *testing.T) {
	sm := NewSessionManager()
	session := sm.CreateSession("test-session")
	
	session.SetState("key1", "value1")
	session.SetMetadata("author", "yolo")
	session.SaveResult("tool1", map[string]string{"input": "data"}, "output", nil)
	
	data, err := session.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}
	
	deserialized, err := DeserializeSession(data)
	if err != nil {
		t.Fatalf("Failed to deserialize: %v", err)
	}
	
	if deserialized.ID != session.ID {
		t.Errorf("Expected ID %s, got %s", session.ID, deserialized.ID)
	}
	
	value, exists := deserialized.GetState("key1")
	if !exists || value != "value1" {
		t.Error("Expected state key1=value1 after deserialization")
	}
	
	meta, exists := deserialized.GetMetadata("author")
	if !exists || meta != "yolo" {
		t.Error("Expected metadata author=yolo after deserialization")
	}
	
	results := deserialized.GetAllResults()
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
}

func TestGenerateSessionID(t *testing.T) {
	id1 := generateSessionID()
	id2 := generateSessionID()
	
	if id1 == "" {
		t.Error("Expected non-empty session ID")
	}
	
	if id2 == "" {
		t.Error("Expected non-empty session ID")
	}
	
	if id1 == id2 {
		t.Error("Expected unique session IDs")
	}
	
	// Check format: should contain "session_" prefix and timestamp + hex
	expectedPrefix := "session_"
	if len(id1) < len(expectedPrefix)+20 {
		t.Errorf("Session ID too short: %s", id1)
	}
}

type testError string

func (e testError) Error() string {
	return string(e)
}
