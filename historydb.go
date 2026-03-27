// HistoryDB persists conversation history and evolution events to a SQLite
// database with in-memory full-text search. Every message is stored permanently
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
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

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

	// Note: FTS5 is not required — we use pure Go full-text search instead.
	// This eliminates the dependency on SQLite's FTS5 extension and works with any build.

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

// msgRow represents a message from the database.
type msgRow struct {
	id, role, content, ts string
	meta                  sql.NullString
}

// Search performs a full-text search across all messages ever recorded.
// Uses pure Go tokenization and ranking (no FTS5 dependency).
// Query supports: words, "exact phrases", OR operator, prefix* terms.
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

	// Load all messages from database
	rows, err := h.db.Query(`SELECT id, role, content, ts, metadata FROM messages ORDER BY id`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var messages []msgRow
	for rows.Next() {
		var m msgRow
		if err := rows.Scan(&m.id, &m.role, &m.content, &m.ts, &m.meta); err != nil {
			continue
		}
		messages = append(messages, m)
	}

	if len(messages) == 0 {
		return nil
	}

	// Build inverted index and parse query
	index := newInvertedIndex(messages)
	terms := tokenizeQuery(query)

	// Score all messages
	type scoredMsg struct {
		msg      msgRow
		score    float64
		matchPos []int // positions where terms matched for snippet
	}

	var scored []scoredMsg
	for _, m := range messages {
		score, matchPos := index.scoreDocument(m.content, terms)
		if score > 0 {
			scored = append(scored, scoredMsg{msg: m, score: score, matchPos: matchPos})
		}
	}

	// Sort by score (descending)
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Limit results and build SearchResult slice
	if len(scored) > limit {
		scored = scored[:limit]
	}

	var results []SearchResult
	for _, sm := range scored {
		var meta map[string]any
		if sm.msg.meta.Valid && sm.msg.meta.String != "" {
			json.Unmarshal([]byte(sm.msg.meta.String), &meta)
		}

		// Generate snippet with highlighted matches
		snippet := generateSnippet(sm.msg.content, sm.matchPos, 100)

		results = append(results, SearchResult{
			Message: HistoryMessage{
				Role:    sm.msg.role,
				Content: sm.msg.content,
				TS:      sm.msg.ts,
				Meta:    meta,
			},
			Snippet:   snippet,
			Relevance: sm.score,
		})
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

// ── Pure Go Full-Text Search Implementation (no FTS5 dependency) ──

// SearchTerm represents a parsed search term with optional operators.
type SearchTerm struct {
	Word     string // the word/token to match
	Prefix   bool   // true if this is a prefix search (word*)
	Position int    // position in query for AND/OR logic
}

// tokenizeQuery parses a query string into search terms.
// Supports: words, "exact phrases" (as single tokens), OR operator, prefix* terms.
func tokenizeQuery(query string) []SearchTerm {
	// First, handle quoted phrases - protect them from splitting
	phraseRegex := regexp.MustCompile(`"([^"]+)"`)
	phrases := phraseRegex.FindAllStringSubmatch(query, -1)

	// Replace quoted phrases with placeholders
	placeholders := make(map[string]string)
	for i, match := range phrases {
		if len(match) >= 2 {
			placeholder := fmt.Sprintf("__PHRASE%d__", i)
			placeholders[placeholder] = match[1]
			query = strings.Replace(query, match[0], placeholder, 1)
		}
	}

	// Split on whitespace and operators
	var tokens []string
	for _, part := range regexp.MustCompile(`\s+`).Split(query, -1) {
		if part != "" && !isOperator(part) {
			tokens = append(tokens, part)
		} else if isOperator(part) {
			tokens = append(tokens, strings.ToUpper(part))
		}
	}

	// Build search terms
	var terms []SearchTerm
	for i, token := range tokens {
		if _, isPlaceholder := placeholders[token]; isPlaceholder {
			// Restore phrase - treat as single word for now (simplified)
			for p, original := range placeholders {
				if token == p {
					terms = append(terms, SearchTerm{Word: strings.ToLower(original), Position: i})
					break
				}
			}
		} else if isOperator(token) {
			continue // Operators handled in scoring logic
		} else {
			// Check for prefix search (word*)
			prefix := false
			word := strings.ToLower(token)
			if strings.HasSuffix(word, "*") {
				prefix = true
				word = strings.TrimSuffix(word, "*")
			}
			terms = append(terms, SearchTerm{Word: word, Prefix: prefix, Position: i})
		}
	}

	return terms
}

// isOperator checks if a token is a search operator.
func isOperator(token string) bool {
	t := strings.ToUpper(strings.TrimSpace(token))
	return t == "AND" || t == "OR" || t == "NOT"
}

// invertedIndex maps words to document information.
type invertedIndex struct {
	documents []string               // documents[i] = full text of message i
	postings  map[string]map[int]int // word -> docID -> frequency
	totalDocs int                    // total number of documents
}

// newInvertedIndex builds an in-memory inverted index from all messages.
func newInvertedIndex(messages []msgRow) *invertedIndex {
	index := &invertedIndex{
		documents: make([]string, len(messages)),
		postings:  make(map[string]map[int]int),
		totalDocs: len(messages),
	}

	for i, m := range messages {
		index.documents[i] = strings.ToLower(m.content)

		// Tokenize content
		words := tokenizeText(m.content)
		wordCount := make(map[string]int)
		for _, word := range words {
			if len(word) > 1 { // Skip single characters
				wordCount[word]++
			}
		}

		// Add to index
		for word, count := range wordCount {
			if index.postings[word] == nil {
				index.postings[word] = make(map[int]int)
			}
			index.postings[word][i] = count
		}
	}

	return index
}

// tokenizeText splits text into words (lowercased, no punctuation).
func tokenizeText(text string) []string {
	text = strings.ToLower(text)

	// Split on non-alphanumeric characters
	var words []string
	current := strings.Builder{}
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current.WriteRune(r)
		} else {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
		}
	}
	if current.Len() > 0 {
		words = append(words, current.String())
	}

	return words
}

// scoreDocument computes a relevance score for a document against search terms.
// Returns (score, matchPositions) where matchPositions are indices of matched words.
func (idx *invertedIndex) scoreDocument(content string, terms []SearchTerm) (float64, []int) {
	if len(terms) == 0 {
		return 0, nil
	}

	contentLower := strings.ToLower(content)
	words := tokenizeText(content)

	var totalScore float64
	var matchPositions []int

	for _, term := range terms {
		score := 0.0

		if term.Prefix {
			// Prefix matching: find all words starting with the prefix
			for i, word := range words {
				if strings.HasPrefix(word, term.Word) {
					score += 1.5 // Boost prefix matches
					matchPositions = append(matchPositions, i)
				}
			}
		} else {
			// Exact word matching with position tracking
			for i, word := range words {
				if word == term.Word {
					score += 1.0
					matchPositions = append(matchPositions, i)
				}
			}

			// Also check for phrase matches if term contains spaces
			if strings.Contains(term.Word, " ") {
				if strings.Contains(contentLower, term.Word) {
					score += 2.0 // Boost phrase matches
					// Find all phrase occurrences
					start := 0
					for {
						pos := strings.Index(contentLower[start:], term.Word)
						if pos == -1 {
							break
						}
						matchPositions = append(matchPositions, start+pos)
						start += pos + len(term.Word)
					}
				}
			}
		}

		totalScore += score
	}

	if totalScore > 0 {
		// Apply TF-IDF-like normalization
		// Boost shorter documents and penalize very long ones
		docLengthFactor := float64(len(words))
		if docLengthFactor > 100 {
			docLengthFactor = 100 + (docLengthFactor-100)*0.5 // Diminish returns after 100 words
		}
		totalScore = totalScore / math.Sqrt(docLengthFactor)
	}

	return totalScore, matchPositions
}

// generateSnippet creates a highlighted excerpt around matched positions.
func generateSnippet(content string, matchPos []int, maxLen int) string {
	if len(matchPos) == 0 || len(content) <= maxLen {
		if len(content) > maxLen {
			return content[:maxLen-3] + "..."
		}
		return content
	}

	// Find the best match position to center on
	centerPos := matchPos[0]

	// Try to find a word boundary before centerPos
	start := centerPos - maxLen/2
	if start < 0 {
		start = 0
	} else {
		// Move back to word boundary
		for start > 0 && content[start] != ' ' {
			start--
		}
	}

	// Try to find a word boundary after centerPos
	end := centerPos + maxLen/2
	if end > len(content) {
		end = len(content)
	} else {
		// Move forward to word boundary
		for end < len(content) && content[end] != ' ' {
			end++
		}
	}

	snippet := content[start:end]

	// Add ellipsis if we truncated
	if start > 0 {
		snippet = "... " + snippet
	}
	if end < len(content) {
		snippet = snippet + " ..."
	}

	return snippet
}
