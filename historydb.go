// HistoryDB persists conversation history and evolution events to a SQLite
// database with full-text search (FTS5). Every message is stored permanently
// and indexed for keyword retrieval, giving the agent unlimited memory depth.
//
// The in-memory Data field holds a recent window (MaxHistoryMessages) for
// backward compatibility with code that reads Data.Messages directly. The
// database itself is never pruned.

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// HistoryDB owns the SQLite database and an in-memory cache of recent
// messages. All mutating methods are goroutine-safe.
type HistoryDB struct {
	db      *sql.DB
	yoloDir string
	dbPath  string
	Data    HistoryData // in-memory window of recent messages
	mu      sync.Mutex
}

// NewHistoryDB creates a history database manager. Call Load() to open
// the database and populate the in-memory cache.
func NewHistoryDB(yoloDir string) *HistoryDB {
	return &HistoryDB{
		yoloDir: yoloDir,
		dbPath:  filepath.Join(yoloDir, "history.db"),
		Data: HistoryData{
			Version:      1,
			Config:       HistoryConfig{Created: time.Now().Format(time.RFC3339)},
			Messages:     []HistoryMessage{},
			EvolutionLog: []EvolutionEntry{},
		},
	}
}

// open creates the database file (and parent directory) if needed,
// then runs schema migrations.
func (h *HistoryDB) open() error {
	if h.db != nil {
		return nil
	}
	os.MkdirAll(h.yoloDir, 0o755)
	db, err := sql.Open("sqlite3", h.dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return fmt.Errorf("open history db: %w", err)
	}
	h.db = db
	return h.migrate()
}

// migrate creates tables and indexes if they do not exist.
func (h *HistoryDB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS messages (
		id        INTEGER PRIMARY KEY AUTOINCREMENT,
		role      TEXT    NOT NULL,
		content   TEXT    NOT NULL,
		ts        TEXT    NOT NULL,
		metadata  TEXT
	);
	CREATE TABLE IF NOT EXISTS evolution (
		id     INTEGER PRIMARY KEY AUTOINCREMENT,
		ts     TEXT NOT NULL,
		action TEXT NOT NULL,
		detail TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS config (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);
	`
	if _, err := h.db.Exec(schema); err != nil {
		return fmt.Errorf("create tables: %w", err)
	}

	// Create FTS5 virtual table if it doesn't exist.
	// content-sync triggers keep the index up to date automatically.
	fts := `
	CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(
		content,
		content=messages,
		content_rowid=id
	);
	`
	if _, err := h.db.Exec(fts); err != nil {
		return fmt.Errorf("create fts: %w", err)
	}

	triggers := `
	CREATE TRIGGER IF NOT EXISTS messages_ai AFTER INSERT ON messages BEGIN
		INSERT INTO messages_fts(rowid, content) VALUES (new.id, new.content);
	END;
	CREATE TRIGGER IF NOT EXISTS messages_ad AFTER DELETE ON messages BEGIN
		INSERT INTO messages_fts(messages_fts, rowid, content) VALUES('delete', old.id, old.content);
	END;
	CREATE TRIGGER IF NOT EXISTS messages_au AFTER UPDATE ON messages BEGIN
		INSERT INTO messages_fts(messages_fts, rowid, content) VALUES('delete', old.id, old.content);
		INSERT INTO messages_fts(rowid, content) VALUES (new.id, new.content);
	END;
	`
	if _, err := h.db.Exec(triggers); err != nil {
		// Triggers may already exist; non-fatal.
	}

	return nil
}

// Load opens the database, migrates legacy history.json if present, and
// populates the in-memory cache. Returns true if messages were loaded.
func (h *HistoryDB) Load() bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	if err := h.open(); err != nil {
		cprint(Red, fmt.Sprintf("  History DB error: %v", err))
		return false
	}

	// One-time migration from legacy history.json
	jsonPath := filepath.Join(h.yoloDir, "history.json")
	if _, err := os.Stat(jsonPath); err == nil {
		h.importJSON(jsonPath)
	}

	h.loadRecent(MaxHistoryMessages)
	h.loadConfig()
	h.loadEvolution(MaxEvolutionEntries)

	return len(h.Data.Messages) > 0
}

// ── Internal loaders ──

func (h *HistoryDB) loadRecent(n int) {
	rows, err := h.db.Query(
		`SELECT role, content, ts, metadata FROM messages ORDER BY id DESC LIMIT ?`, n)
	if err != nil {
		return
	}
	defer rows.Close()

	var msgs []HistoryMessage
	for rows.Next() {
		var m HistoryMessage
		var metaJSON sql.NullString
		if err := rows.Scan(&m.Role, &m.Content, &m.TS, &metaJSON); err != nil {
			continue
		}
		if metaJSON.Valid && metaJSON.String != "" {
			json.Unmarshal([]byte(metaJSON.String), &m.Meta)
		}
		msgs = append(msgs, m)
	}
	// Reverse: query returns DESC but we want chronological order
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	h.Data.Messages = msgs
}

func (h *HistoryDB) loadConfig() {
	var model, created string
	h.db.QueryRow(`SELECT value FROM config WHERE key = 'model'`).Scan(&model)
	h.db.QueryRow(`SELECT value FROM config WHERE key = 'created'`).Scan(&created)
	h.Data.Config.Model = model
	if created != "" {
		h.Data.Config.Created = created
	}
}

func (h *HistoryDB) loadEvolution(n int) {
	rows, err := h.db.Query(
		`SELECT ts, action, detail FROM evolution ORDER BY id DESC LIMIT ?`, n)
	if err != nil {
		return
	}
	defer rows.Close()

	var entries []EvolutionEntry
	for rows.Next() {
		var e EvolutionEntry
		if err := rows.Scan(&e.TS, &e.Action, &e.Detail); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}
	h.Data.EvolutionLog = entries
}

// importJSON reads a legacy history.json, inserts its data into the
// database, and renames the file to history.json.migrated.
func (h *HistoryDB) importJSON(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var hd HistoryData
	if err := json.Unmarshal(data, &hd); err != nil {
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		return
	}
	stmt, _ := tx.Prepare(`INSERT INTO messages (role, content, ts, metadata) VALUES (?, ?, ?, ?)`)
	for _, m := range hd.Messages {
		metaJSON := ""
		if m.Meta != nil {
			b, _ := json.Marshal(m.Meta)
			metaJSON = string(b)
		}
		stmt.Exec(m.Role, m.Content, m.TS, metaJSON)
	}
	stmt.Close()

	estmt, _ := tx.Prepare(`INSERT INTO evolution (ts, action, detail) VALUES (?, ?, ?)`)
	for _, e := range hd.EvolutionLog {
		estmt.Exec(e.TS, e.Action, e.Detail)
	}
	estmt.Close()

	if hd.Config.Model != "" {
		tx.Exec(`INSERT OR REPLACE INTO config (key, value) VALUES ('model', ?)`, hd.Config.Model)
	}
	if hd.Config.Created != "" {
		tx.Exec(`INSERT OR REPLACE INTO config (key, value) VALUES ('created', ?)`, hd.Config.Created)
	}

	tx.Commit()
	os.Rename(path, path+".migrated")
	cprint(Green, "  Migrated history.json → history.db")
}

// ── Write methods ──

// Save is a no-op — writes happen immediately in AddMessage/AddEvolution.
// Kept for interface compatibility with code that calls Save() explicitly.
func (h *HistoryDB) Save() error {
	return nil
}

// AddMessage inserts a message into the database (permanently) and
// appends it to the in-memory cache (pruned to MaxHistoryMessages).
func (h *HistoryDB) AddMessage(role, content string, meta map[string]any) {
	h.mu.Lock()
	defer h.mu.Unlock()

	ts := time.Now().Format(time.RFC3339)
	metaJSON := ""
	if meta != nil {
		b, _ := json.Marshal(meta)
		metaJSON = string(b)
	}

	if h.db != nil {
		h.db.Exec(`INSERT INTO messages (role, content, ts, metadata) VALUES (?, ?, ?, ?)`,
			role, content, ts, metaJSON)
	}

	msg := HistoryMessage{Role: role, Content: content, TS: ts, Meta: meta}
	h.Data.Messages = append(h.Data.Messages, msg)
	if len(h.Data.Messages) > MaxHistoryMessages {
		h.Data.Messages = h.Data.Messages[len(h.Data.Messages)-MaxHistoryMessages:]
	}
}

// AddEvolution records an evolution event.
func (h *HistoryDB) AddEvolution(action, description string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	ts := time.Now().Format(time.RFC3339)
	if h.db != nil {
		h.db.Exec(`INSERT INTO evolution (ts, action, detail) VALUES (?, ?, ?)`,
			ts, action, description)
	}

	h.Data.EvolutionLog = append(h.Data.EvolutionLog, EvolutionEntry{
		TS: ts, Action: action, Detail: description,
	})
	if len(h.Data.EvolutionLog) > MaxEvolutionEntries {
		h.Data.EvolutionLog = h.Data.EvolutionLog[len(h.Data.EvolutionLog)-MaxEvolutionEntries:]
	}
}

// ── Read methods ──

// GetContextMessages returns the last maxMsgs messages formatted for the LLM.
func (h *HistoryDB) GetContextMessages(maxMsgs int) []ChatMessage {
	msgs := h.Data.Messages
	start := 0
	if len(msgs) > maxMsgs {
		start = len(msgs) - maxMsgs
	}

	var out []ChatMessage
	for _, m := range msgs[start:] {
		switch m.Role {
		case "user", "assistant":
			out = append(out, ChatMessage{Role: m.Role, Content: m.Content})
		case "tool":
			out = append(out, ChatMessage{Role: "user",
				Content: "[Tool output — this is a previous tool execution result, not user input]\n" + m.Content})
		case "system":
			out = append(out, ChatMessage{Role: "system", Content: m.Content})
		}
	}
	return out
}

// GetLastN returns the last n messages from the in-memory cache.
func (h *HistoryDB) GetLastN(n int) []HistoryMessage {
	h.mu.Lock()
	defer h.mu.Unlock()
	msgs := h.Data.Messages
	if len(msgs) <= n {
		return msgs
	}
	return msgs[len(msgs)-n:]
}

// GetModel returns the configured model name.
func (h *HistoryDB) GetModel() string {
	return h.Data.Config.Model
}

// SetModel updates the model and persists to the database.
func (h *HistoryDB) SetModel(model string) {
	h.Data.Config.Model = model
	if h.db != nil {
		h.db.Exec(`INSERT OR REPLACE INTO config (key, value) VALUES ('model', ?)`, model)
	}
}

// ── Search ──

// SearchResult is a single full-text search hit with context.
type SearchResult struct {
	Message   HistoryMessage `json:"message"`
	Snippet   string         `json:"snippet,omitempty"` // highlighted excerpt
	Relevance float64        `json:"relevance"`
}

// Search performs a full-text search across all messages ever recorded.
// The query uses SQLite FTS5 syntax (words are ANDed by default, use OR
// for alternatives, "quotes" for exact phrases, prefix* for prefixes).
// Returns up to limit results ordered by relevance.
func (h *HistoryDB) Search(query string, limit int) []SearchResult {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.db == nil {
		return nil
	}
	if limit <= 0 {
		limit = 20
	}

	// Escape the query to prevent FTS5 syntax errors from user input.
	// Wrap each token in double quotes so special characters are treated as literals.
	safeQuery := ftsEscapeQuery(query)

	rows, err := h.db.Query(`
		SELECT m.role, m.content, m.ts, m.metadata,
		       snippet(messages_fts, 0, '>>>', '<<<', '...', 40) AS snip,
		       rank
		FROM messages_fts fts
		JOIN messages m ON m.id = fts.rowid
		WHERE messages_fts MATCH ?
		ORDER BY rank
		LIMIT ?
	`, safeQuery, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		var metaJSON sql.NullString
		var rank float64
		if err := rows.Scan(&r.Message.Role, &r.Message.Content, &r.Message.TS,
			&metaJSON, &r.Snippet, &rank); err != nil {
			continue
		}
		if metaJSON.Valid && metaJSON.String != "" {
			json.Unmarshal([]byte(metaJSON.String), &r.Message.Meta)
		}
		r.Relevance = -rank // FTS5 rank is negative; negate for display
		results = append(results, r)
	}
	return results
}

// MessageCount returns the total number of messages in the database.
func (h *HistoryDB) MessageCount() int {
	if h.db == nil {
		return len(h.Data.Messages)
	}
	var count int
	h.db.QueryRow(`SELECT COUNT(*) FROM messages`).Scan(&count)
	return count
}

// Close closes the database connection.
func (h *HistoryDB) Close() {
	if h.db != nil {
		h.db.Close()
		h.db = nil
	}
}

// ftsEscapeQuery sanitizes a search query for FTS5. Words are joined
// with OR so that any matching term returns results (better for recall).
// Use explicit AND between terms to require all. Quoted phrases and
// recognized operators (AND, OR, NOT) are preserved as-is.
func ftsEscapeQuery(query string) string {
	tokens := strings.Fields(query)
	if len(tokens) == 0 {
		return query
	}
	if len(tokens) == 1 {
		t := tokens[0]
		if strings.ContainsAny(t, "^*:-+(){}[]") {
			t = strings.ReplaceAll(t, "\"", "\"\"")
			return "\"" + t + "\""
		}
		return t
	}
	var escaped []string
	for _, t := range tokens {
		upper := strings.ToUpper(t)
		if upper == "OR" || upper == "AND" || upper == "NOT" {
			escaped = append(escaped, upper)
			continue
		}
		// Already-quoted phrases pass through
		if strings.HasPrefix(t, "\"") && strings.HasSuffix(t, "\"") {
			escaped = append(escaped, t)
			continue
		}
		// Quote tokens with special FTS5 characters
		if strings.ContainsAny(t, "^*:-+(){}[]") {
			t = strings.ReplaceAll(t, "\"", "\"\"")
			escaped = append(escaped, "\""+t+"\"")
		} else {
			escaped = append(escaped, t)
		}
	}
	// Join with OR so any matching term returns results.
	// If the user already used explicit operators, they'll be preserved.
	result := escaped[0]
	for i := 1; i < len(escaped); i++ {
		prev := strings.ToUpper(escaped[i-1])
		curr := strings.ToUpper(escaped[i])
		if prev == "OR" || prev == "AND" || prev == "NOT" ||
			curr == "OR" || curr == "AND" || curr == "NOT" {
			result += " " + escaped[i]
		} else {
			result += " OR " + escaped[i]
		}
	}
	return result
}
