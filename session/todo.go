// Package session provides persistent state management including conversation history and configuration.
package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	todoFile       = ".todo.json"
	maxTitleLength = 256
)

// TodoItem represents a single todo item.
type TodoItem struct {
	Title       string    `json:"title"`
	CreatedAt   time.Time `json:"created_at"`
	Done        bool      `json:"done,omitempty"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
}

// MarshalJSON custom marshaling to handle zero time values.
func (t TodoItem) MarshalJSON() ([]byte, error) {
	type Alias TodoItem
	aux := struct {
		*Alias
	}{
		Alias: (*Alias)(&t),
	}

	if !t.Done || t.CompletedAt.IsZero() {
		aux.CompletedAt = time.Time{}
	}

	return json.Marshal(aux)
}

// BatchFailure represents a failed batch operation.
type BatchFailure struct {
	Title string
	Error string
}

// BatchResult holds the outcome of a batch operation.
type BatchResult struct {
	SuccessCount int
	FailureCount int
	SuccessItems []TodoItem
	Failures     []BatchFailure
}

// TodoList manages a persistent todo list.
type TodoList struct {
	mu       sync.RWMutex
	todos    []TodoItem
	filePath string
}

// NewTodoList creates a new TodoList with the given file path.
func NewTodoList(filePath string) *TodoList {
	return &TodoList{
		todos:    []TodoItem{},
		filePath: filePath,
	}
}

// Load loads todos from disk. Returns nil on success, error if failed.
func (t *TodoList) Load() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	data, err := os.ReadFile(t.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var loadedTodos []TodoItem
	if err := json.Unmarshal(data, &loadedTodos); err != nil {
		t.todos = []TodoItem{}
		return nil
	}

	t.todos = loadedTodos
	return nil
}

// saveLocked writes todos to disk. Caller must hold t.mu.
func (t *TodoList) saveLocked() error {
	data, err := json.MarshalIndent(t.todos, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(t.filePath, data, 0o644)
}

// Save persists todos to disk.
func (t *TodoList) Save() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.saveLocked()
}

// validateTitle checks that a title is non-empty, within length limits,
// and not a duplicate of an existing pending todo.
func (t *TodoList) validateTitle(title string) error {
	errs := map[string]string{}

	trimmed := strings.TrimSpace(title)
	if trimmed == "" {
		errs["title"] = "cannot be empty"
		return fmt.Errorf("validation failed: %s", errs["title"])
	}
	if len(trimmed) > maxTitleLength {
		errs["title"] = fmt.Sprintf("exceeds maximum length of %d characters", maxTitleLength)
		return fmt.Errorf("validation failed: %s", errs["title"])
	}

	for _, todo := range t.todos {
		if !todo.Done && strings.EqualFold(todo.Title, trimmed) {
			errs["title"] = "duplicate of existing pending todo"
			return fmt.Errorf("validation failed: %s", errs["title"])
		}
	}

	return nil
}

// Add adds a new todo item without validation.
func (t *TodoList) Add(title string) TodoItem {
	t.mu.Lock()
	defer t.mu.Unlock()

	todo := TodoItem{
		Title:     title,
		CreatedAt: time.Now(),
	}
	t.todos = append(t.todos, todo)
	t.saveLocked()
	return todo
}

// CreateTodoItem adds a todo after validating the title.
func (t *TodoList) CreateTodoItem(title string) (TodoItem, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if err := t.validateTitle(title); err != nil {
		return TodoItem{}, err
	}

	todo := TodoItem{
		Title:     strings.TrimSpace(title),
		CreatedAt: time.Now(),
	}
	t.todos = append(t.todos, todo)
	t.saveLocked()
	return todo, nil
}

// Complete marks a todo item as completed by title (case-insensitive).
func (t *TodoList) Complete(title string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	for i := range t.todos {
		if strings.EqualFold(t.todos[i].Title, title) && !t.todos[i].Done {
			t.todos[i].Done = true
			t.todos[i].CompletedAt = time.Now()
			t.saveLocked()
			return true
		}
	}
	return false
}

// Delete removes a todo item by title (case-insensitive). Returns true if found.
func (t *TodoList) Delete(title string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	for i := range t.todos {
		if strings.EqualFold(t.todos[i].Title, title) {
			t.todos = append(t.todos[:i], t.todos[i+1:]...)
			t.saveLocked()
			return true
		}
	}
	return false
}

// FindTodo searches for a todo by title (case-insensitive).
func (t *TodoList) FindTodo(title string) (bool, TodoItem) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	for _, todo := range t.todos {
		if strings.EqualFold(todo.Title, title) {
			return true, todo
		}
	}
	return false, TodoItem{}
}

// BatchCreate adds multiple todos, skipping invalid/duplicate titles.
func (t *TodoList) BatchCreate(titles []string) BatchResult {
	result := BatchResult{}

	for _, title := range titles {
		item, err := t.CreateTodoItem(title)
		if err != nil {
			result.FailureCount++
			result.Failures = append(result.Failures, BatchFailure{
				Title: title,
				Error: err.Error(),
			})
		} else {
			result.SuccessCount++
			result.SuccessItems = append(result.SuccessItems, item)
		}
	}
	return result
}

// BatchDelete removes multiple todos by title.
func (t *TodoList) BatchDelete(titles []string) BatchResult {
	result := BatchResult{}

	for _, title := range titles {
		if t.Delete(title) {
			result.SuccessCount++
			result.SuccessItems = append(result.SuccessItems, TodoItem{Title: title})
		} else {
			result.FailureCount++
			result.Failures = append(result.Failures, BatchFailure{
				Title: title,
				Error: "not found",
			})
		}
	}
	return result
}

// BatchComplete marks multiple todos as completed by title.
func (t *TodoList) BatchComplete(titles []string) BatchResult {
	result := BatchResult{}

	for _, title := range titles {
		if t.Complete(title) {
			result.SuccessCount++
			result.SuccessItems = append(result.SuccessItems, TodoItem{Title: title})
		} else {
			result.FailureCount++
			result.Failures = append(result.Failures, BatchFailure{
				Title: title,
				Error: "not found or already completed",
			})
		}
	}
	return result
}

// GetPending returns all pending todos, sorted by creation date.
func (t *TodoList) GetPending() []TodoItem {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var pending []TodoItem
	for _, todo := range t.todos {
		if !todo.Done {
			pending = append(pending, todo)
		}
	}
	sort.Slice(pending, func(i, j int) bool {
		return pending[i].CreatedAt.Before(pending[j].CreatedAt)
	})
	return pending
}

// GetCompleted returns all completed todos, sorted newest first.
func (t *TodoList) GetCompleted() []TodoItem {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var completed []TodoItem
	for _, todo := range t.todos {
		if todo.Done {
			completed = append(completed, todo)
		}
	}
	sort.Slice(completed, func(i, j int) bool {
		return completed[i].CompletedAt.After(completed[j].CompletedAt)
	})
	return completed
}

// GetAll returns all todos.
func (t *TodoList) GetAll() []TodoItem {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make([]TodoItem, len(t.todos))
	copy(result, t.todos)
	return result
}

// Count returns the number of pending todos.
func (t *TodoList) Count() int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var count int
	for _, todo := range t.todos {
		if !todo.Done {
			count++
		}
	}
	return count
}

// Render renders the todo list as a formatted string for reports/display.
func (t *TodoList) Render() string {
	var sb strings.Builder

	sb.WriteString("TODO LIST\n")
	sb.WriteString(strings.Repeat("-", 50) + "\n\n")

	pending := t.GetPending()
	completed := t.GetCompleted()

	if len(pending) == 0 && len(completed) == 0 {
		sb.WriteString("No todos yet.\n")
		return sb.String()
	}

	if len(pending) > 0 {
		sb.WriteString(fmt.Sprintf("PENDING (%d)\n", len(pending)))
		for _, todo := range pending {
			sb.WriteString(fmt.Sprintf("  [ ] %s [%s]\n", todo.Title, todo.CreatedAt.Format("2006-01-02")))
		}
		sb.WriteString("\n")
	}

	if len(completed) > 0 {
		limit := 10
		if len(completed) < limit {
			limit = len(completed)
		}
		sb.WriteString(fmt.Sprintf("COMPLETED (last %d of %d)\n", limit, len(completed)))
		for i := 0; i < limit; i++ {
			sb.WriteString(fmt.Sprintf("  [x] %s (done: %s)\n", completed[i].Title, completed[i].CompletedAt.Format("2006-01-02")))
		}
		if len(completed) > limit {
			sb.WriteString(fmt.Sprintf("  ... and %d more completed items\n", len(completed)-limit))
		}
	}

	return sb.String()
}

// RenderPendingContext returns a short summary of pending todos.
func (t *TodoList) RenderPendingContext() string {
	pending := t.GetPending()
	if len(pending) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n## Pending Todos (%d)\n", len(pending)))
	for _, todo := range pending {
		sb.WriteString(fmt.Sprintf("- %s\n", todo.Title))
	}
	return sb.String()
}

// ─── String conversion utilities ────────────────────────────────────────

// ConvertParamValue attempts to convert a string parameter value to its appropriate Go type.
func ConvertParamValue(val string) any {
	if num, err := strconv.ParseInt(val, 10, 64); err == nil {
		return num
	}
	if floatVal, err := strconv.ParseFloat(val, 64); err == nil {
		return floatVal
	}
	if boolVal, err := strconv.ParseBool(val); err == nil {
		return boolVal
	}
	return val
}

// ParseParamString converts "key=value, key2=value2" into a map.
func ParseParamString(paramStr string) map[string]any {
	result := make(map[string]any)
	pairs := strings.Split(paramStr, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		eqIdx := strings.Index(pair, "=")
		if eqIdx < 0 {
			continue
		}
		key := strings.TrimSpace(pair[:eqIdx])
		value := strings.TrimSpace(pair[eqIdx+1:])
		if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			value = value[1 : len(value)-1]
		} else if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
			value = value[1 : len(value)-1]
		}
		if num, err := strconv.ParseInt(value, 10, 64); err == nil {
			result[key] = num
		} else if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			result[key] = floatVal
		} else if boolVal, err := strconv.ParseBool(value); err == nil {
			result[key] = boolVal
		} else {
			result[key] = value
		}
	}
	return result
}
