package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// package-level mutex for thread-safe session ID generation
var sessionIDMutex sync.Mutex

// ToolSession manages state across multiple tool calls for complex workflows
type ToolSession struct {
	ID           string            `json:"id"`
	CreatedAt    time.Time         `json:"created_at"`
	LastActivity time.Time         `json:"last_activity"`
	State        map[string]any    `json:"state"`
	Results      []ToolResult      `json:"results"`
	Metadata     map[string]string `json:"metadata"`
	Active       bool              `json:"active"`
	maxIdleTime  time.Duration     // Maximum idle time before session expires
	mu           sync.RWMutex
}

// ToolResult stores the result of a single tool invocation within a session
type ToolResult struct {
	Tool      string    `json:"tool"`
	CallIndex int       `json:"call_index"`
	Input     any       `json:"input"`
	Output    any       `json:"output"`
	Error     string    `json:"error,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// SessionManager handles creation, retrieval, and expiration of tool sessions
type SessionManager struct {
	sessions map[string]*ToolSession
	mu       sync.RWMutex
	ttl      time.Duration // Session TTL in seconds
}

// NewSessionManager creates a new session manager with default TTL
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*ToolSession),
		ttl:      24 * time.Hour, // Sessions expire after 24 hours of inactivity
	}
}

// CreateSession creates a new tool session with an optional ID
func (sm *SessionManager) CreateSession(sessionID string) *ToolSession {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sessionID == "" {
		sessionID = generateSessionID()
	}

	now := time.Now()
	session := &ToolSession{
		ID:           sessionID,
		CreatedAt:    now,
		LastActivity: now,
		State:        make(map[string]any),
		Results:      make([]ToolResult, 0),
		Metadata:     make(map[string]string),
		Active:       true,
		maxIdleTime:  sm.ttl,
	}

	sm.sessions[sessionID] = session
	return session
}

// GetSession retrieves an existing session by ID, returning nil if not found or expired.
// Returns nil without updating LastActivity if another concurrent call will expire it first.
// This ensures proper lock ordering: sm.mu is held for the entire duration of validation
// and expiration check, preventing race conditions between map access and session expiry.
func (sm *SessionManager) GetSession(sessionID string) *ToolSession {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists || !session.Active {
		return nil
	}

	// Check expiration atomically with map access while holding sm.mu to prevent race conditions
	// This ensures the session hasn't been marked inactive or deleted by another GetSession call
	if time.Since(session.LastActivity) > session.maxIdleTime {
		session.Active = false
		delete(sm.sessions, sessionID)
		return nil
	}

	// Update last activity timestamp while holding sm.mu to prevent concurrent access issues
	// Only update if we haven't been superseded by another expiry check
	if time.Since(session.LastActivity) <= session.maxIdleTime {
		session.LastActivity = time.Now()
	}

	return session
}

// SaveResult stores a tool result in the session
func (s *ToolSession) SaveResult(tool string, input any, output any, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := ToolResult{
		Tool:      tool,
		CallIndex: len(s.Results),
		Input:     input,
		Output:    output,
		Timestamp: time.Now(),
	}

	if err != nil {
		result.Error = err.Error()
	}

	s.Results = append(s.Results, result)
	s.LastActivity = time.Now()
}

// SetState sets a state value in the session
func (s *ToolSession) SetState(key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.State[key] = value
	s.LastActivity = time.Now()
}

// GetState retrieves a state value from the session
func (s *ToolSession) GetState(key string) (any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, exists := s.State[key]
	return value, exists
}

// SetMetadata sets metadata for the session
func (s *ToolSession) SetMetadata(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Metadata[key] = value
}

// GetMetadata retrieves metadata from the session
func (s *ToolSession) GetMetadata(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, exists := s.Metadata[key]
	return value, exists
}

// GetAllResults returns all results from the session
func (s *ToolSession) GetAllResults() []ToolResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	results := make([]ToolResult, len(s.Results))
	copy(results, s.Results)
	return results
}

// GetLastResult returns the most recent result from a specific tool
func (s *ToolSession) GetLastResult(tool string) *ToolResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := len(s.Results) - 1; i >= 0; i-- {
		if s.Results[i].Tool == tool && s.Results[i].Error == "" {
			result := s.Results[i]
			return &result
		}
	}

	return nil
}

// Serialize converts the session to JSON
func (s *ToolSession) Serialize() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return json.Marshal(s)
}

// Deserialize creates a session from JSON
func DeserializeSession(data []byte) (*ToolSession, error) {
	var session ToolSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to deserialize session: %w", err)
	}
	return &session, nil
}

// CleanupExpired removes all expired sessions from the manager
//
// Lock Ordering Pattern: This function only acquires sm.mu (manager mutex).
// It collects expired session IDs while holding sm.mu, then releases the lock
// before performing individual deletion operations. This prevents potential
// deadlocks that could occur if we held both sm.mu and session.mu simultaneously,
// which would be inconsistent with other methods like GetSession() that only
// acquire sm.mu during map iteration.
//
// The two-phase approach (collect-then-delete) eliminates the TOCTOU race condition
// where a session could be accessed after being removed from the map but while
// still holding references.
func (sm *SessionManager) CleanupExpired() int {
	sm.mu.Lock()

	// Phase 1: Collect expired session IDs while holding manager mutex only.
	// This minimizes lock contention and prevents deadlock scenarios by not
	// holding both sm.mu and session.mu simultaneously.
	expiredIds := make([]string, 0)
	now := time.Now()

	for id, session := range sm.sessions {
		session.mu.Lock()
		isExpired := !session.Active || now.Sub(session.LastActivity) > session.maxIdleTime
		if isExpired {
			expiredIds = append(expiredIds, id)
		}
		session.mu.Unlock()
	}

	// Phase 2: Delete collected sessions after releasing manager mutex.
	// By collecting IDs first and releasing sm.mu before deletions, we:
	// - Prevent deadlocks from inconsistent lock ordering
	// - Avoid TOCTOU races where a session might be accessed while deleted
	// - Allow concurrent access to the session map for other operations
	sm.mu.Unlock()

	count := 0
	for _, id := range expiredIds {
		sm.mu.Lock()
		if session, exists := sm.sessions[id]; exists && session.Active == false {
			delete(sm.sessions, id)
			count++
		}
		sm.mu.Unlock()
	}

	return count
}

// ListActiveSessions returns IDs of all active sessions
func (sm *SessionManager) ListActiveSessions() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	active := make([]string, 0, len(sm.sessions))
	now := time.Now()

	for id, session := range sm.sessions {
		session.mu.RLock()
		if session.Active && now.Sub(session.LastActivity) <= session.maxIdleTime {
			active = append(active, id)
		}
		session.mu.RUnlock()
	}

	return active
}

// SessionCount returns the number of active sessions
func (sm *SessionManager) SessionCount() int {
	return len(sm.ListActiveSessions())
}

// generateSessionID creates a unique session identifier using cryptographically secure random bytes.
// This function is thread-safe due to the package-level sessionIDMutex protecting concurrent access.
func generateSessionID() string {
	sessionIDMutex.Lock()
	defer sessionIDMutex.Unlock()

	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Sprintf("failed to generate random bytes: %v", err))
	}
	return fmt.Sprintf("session_%d_%s", time.Now().UnixNano(), hex.EncodeToString(b[:]))
}

// generateRandomBytes returns cryptographically secure random bytes of the specified length
func generateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return b, nil
}

// randInt generates a cryptographically secure random integer
func randInt() int {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Sprintf("failed to generate random bytes: %v", err))
	}
	return int(b[0]) | int(b[1])<<8 | int(b[2])<<16 | int(b[3])<<24
}

// SessionContext adds session support to context
type SessionContext struct {
	context.Context
	Session *ToolSession
}

// WithSession creates a new context with session information
func WithSession(ctx context.Context, session *ToolSession) SessionContext {
	return SessionContext{
		Context: ctx,
		Session: session,
	}
}
