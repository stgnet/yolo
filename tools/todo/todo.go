// Package todo provides comprehensive todo list management functionality.
// Allows YOLO to maintain a persistent todo list with pending and completed items.
package todo

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"yolo/utils"
)

const (
	todoFile       = ".todo.json"
	maxTitleLength = 256
)

// TodoItem represents a single todo item with metadata
type TodoItem struct {
	Title       string    `json:"title"`
	CreatedAt   time.Time `json:"created_at"`
	Done        bool      `json:"done,omitempty"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
}

// MarshalJSON custom marshaling to handle zero time values properly
func (t TodoItem) MarshalJSON() ([]byte, error) {
	type Alias TodoItem
	aux := struct {
		*Alias
	}{
		Alias: (*Alias)(&t),
	}

	// Only include CompletedAt if the item is done and has a valid timestamp
	if !t.Done || t.CompletedAt.IsZero() {
		aux.CompletedAt = time.Time{}
	}

	return json.Marshal(aux)
}

// TodoList manages a collection of todo items with thread-safe operations
type TodoList struct {
	mu       sync.RWMutex
	todos    []TodoItem
	filePath string
}

// Global todo list instance - kept for backwards compatibility
var globalTodoList *TodoList

func init() {
	wd, _ := os.Getwd()
	globalTodoList = NewTodoList(filepath.Join(wd, todoFile))
	globalTodoList.Load()
}

// GetGlobalTodoList returns the global todo list instance
func GetGlobalTodoList() *TodoList {
	return globalTodoList
}

// NewTodoList creates a new TodoList with the specified file path
func NewTodoList(filePath string) *TodoList {
	return &TodoList{
		todos:    []TodoItem{},
		filePath: filePath,
	}
}

// Load loads todos from the JSON file
func (t *TodoList) Load() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	data, err := utils.ReadFile(t.filePath)
	if err != nil {
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
	return utils.WriteFile(t.filePath, data, 0644)
}

// Save persists todos to the JSON file
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
		return fmt.Errorf("validation failed: title %s", errs["title"])
	}
	if len(trimmed) > maxTitleLength {
		errs["title"] = fmt.Sprintf("exceeds maximum length of %d characters", maxTitleLength)
		return fmt.Errorf("validation failed: title %s", errs["title"])
	}

	// Check for duplicates among pending todos (caller must hold at least RLock)
	for _, todo := range t.todos {
		if !todo.Done && strings.EqualFold(todo.Title, trimmed) {
			errs["title"] = "duplicate of existing pending todo"
			return fmt.Errorf("validation failed: title %s", errs["title"])
		}
	}

	return nil
}

// Add adds a new todo item after validating the title.
func (t *TodoList) Add(title string) (TodoItem, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	trimmed := strings.TrimSpace(title)
	if err := t.validateTitle(trimmed); err != nil {
		return TodoItem{}, err
	}

	todo := TodoItem{
		Title:     trimmed,
		CreatedAt: time.Now(),
	}
	t.todos = append(t.todos, todo)
	t.saveLocked()
	return todo, nil
}

// Complete marks a todo as completed by exact title match.
func (t *TodoList) Complete(title string) (bool, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for i := range t.todos {
		if !t.todos[i].Done && strings.EqualFold(t.todos[i].Title, title) {
			t.todos[i].Done = true
			t.todos[i].CompletedAt = time.Now()
			err := t.saveLocked()
			return true, err
		}
	}
	return false, nil
}

// Delete removes a todo item by exact title match.
func (t *TodoList) Delete(title string) (bool, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for i := range t.todos {
		if strings.EqualFold(t.todos[i].Title, title) {
			t.todos = append(t.todos[:i], t.todos[i+1:]...)
			err := t.saveLocked()
			return true, err
		}
	}
	return false, nil
}

// List returns all todo items (pending and completed).
func (t *TodoList) List() []TodoItem {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make([]TodoItem, len(t.todos))
	copy(result, t.todos)
	return result
}

// GetPendingTodos returns only pending (incomplete) todos.
func (t *TodoList) GetPendingTodos() []TodoItem {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var pending []TodoItem
	for _, item := range t.todos {
		if !item.Done {
			pending = append(pending, item)
		}
	}
	return pending
}

// GetCompletedTodos returns only completed todos.
func (t *TodoList) GetCompletedTodos() []TodoItem {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var completed []TodoItem
	for _, item := range t.todos {
		if item.Done {
			completed = append(completed, item)
		}
	}
	return completed
}

// CountPending returns the number of pending todos.
func (t *TodoList) CountPending() int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	count := 0
	for _, item := range t.todos {
		if !item.Done {
			count++
		}
	}
	return count
}

// CountCompleted returns the number of completed todos.
func (t *TodoList) CountCompleted() int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	count := 0
	for _, item := range t.todos {
		if item.Done {
			count++
		}
	}
	return count
}

// SearchTodos searches for todos containing the query string in their title.
func (t *TodoList) SearchTodos(query string, includeCompleted bool) []TodoItem {
	t.mu.RLock()
	defer t.mu.RUnlock()

	query = strings.ToLower(strings.TrimSpace(query))
	var results []TodoItem

	for _, item := range t.todos {
		if !includeCompleted && item.Done {
			continue
		}
		if strings.Contains(strings.ToLower(item.Title), query) {
			results = append(results, item)
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].CreatedAt.Before(results[j].CreatedAt)
	})

	return results
}

// ClearCompleted removes all completed todos from the list.
func (t *TodoList) ClearCompleted() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	var pending []TodoItem
	for _, item := range t.todos {
		if !item.Done {
			pending = append(pending, item)
		}
	}

	t.todos = pending
	return t.saveLocked()
}

// BatchAdd adds multiple todo items and returns results for each.
func (t *TodoList) BatchAdd(titles []string) ([]TodoItem, []error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	results := make([]TodoItem, 0, len(titles))
	errors := make([]error, 0, len(titles))

	for _, title := range titles {
		trimmed := strings.TrimSpace(title)
		if err := t.validateTitle(trimmed); err != nil {
			errors = append(errors, fmt.Errorf("adding %q: %w", trimmed, err))
			continue
		}

		todo := TodoItem{
			Title:     trimmed,
			CreatedAt: time.Now(),
		}
		t.todos = append(t.todos, todo)
		results = append(results, todo)
	}

	if len(results) > 0 {
		t.saveLocked()
	}

	return results, errors
}

// FormatPendingTodos returns a formatted string of pending todos for display.
func (t *TodoList) FormatPendingTodos() string {
	todos := t.GetPendingTodos()
	if len(todos) == 0 {
		return "No pending todos."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Pending Todos (%d):\n", len(todos)))
	for i, todo := range todos {
		sb.WriteString(fmt.Sprintf("%2d. %s (created: %s)\n", i+1, todo.Title, todo.CreatedAt.Format("2006-01-02 15:04:05")))
	}

	return sb.String()
}

// FormatAllTodos returns a formatted string of all todos for display.
func (t *TodoList) FormatAllTodos() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.todos) == 0 {
		return "No todos."
	}

	var sb strings.Builder
	pendingCount := 0
	completedCount := 0

	for _, todo := range t.todos {
		if todo.Done {
			completedCount++
		} else {
			pendingCount++
		}
	}

	sb.WriteString(fmt.Sprintf("Todos (%d pending, %d completed):\n", pendingCount, completedCount))
	sb.WriteString(strings.Repeat("-", 50))
	sb.WriteString("\n\n")

	// Pending todos
	sb.WriteString("PENDING:\n")
	for i, todo := range t.todos {
		if !todo.Done {
			sb.WriteString(fmt.Sprintf("%2d. %s (created: %s)\n", i+1, todo.Title, todo.CreatedAt.Format("2006-01-02 15:04:05")))
		}
	}

	if completedCount > 0 {
		sb.WriteString("\nCOMPLETED:\n")
		for i, todo := range t.todos {
			if todo.Done {
				sb.WriteString(fmt.Sprintf("%2d. ~~%s~~ (completed: %s)\n", i+1, todo.Title, todo.CompletedAt.Format("2006-01-02 15:04:05")))
			}
		}
	}

	return sb.String()
}
