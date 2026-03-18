// Package todo provides a persistent todo list management system.
// Todos are stored in a JSON file and can be queried, added, completed, or deleted.
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

	"yolo/errors"
	"yolo/utils"
)

const (
	todoFile       = ".todo.json"
	maxTitleLength = 256
)

// ──── Error Types ────────

// TodoValidationError is returned when a todo title fails validation.
type TodoValidationError struct {
	Field  string
	Title  string
	Errors map[string]string
}

func (e *TodoValidationError) Error() string {
	var parts []string
	for field, msg := range e.Errors {
		parts = append(parts, fmt.Sprintf("%s: %s", field, msg))
	}
	return fmt.Sprintf("validation failed: %s", strings.Join(parts, "; "))
}

// TodoNotFoundError is returned when a todo cannot be found by title.
type TodoNotFoundError struct {
	Title    string
	Existing map[string]bool
}

func (e *TodoNotFoundError) Error() string {
	return fmt.Sprintf("todo not found: %q", e.Title)
}

// ──── Data Types ────────

// TodoItem represents a single todo item.
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

// BatchResult holds the outcome of a batch operation.
type BatchResult struct {
	SuccessCount int
	FailureCount int
	SuccessItems []TodoItem
	Failures     []BatchFailure
}

// BatchFailure represents a failed batch operation item.
type BatchFailure struct {
	Title string
	Error string
}

// ──── TodoList ────────

// TodoList manages a persistent collection of todo items.
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

// Load reads todos from disk.
func (t *TodoList) Load() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	data, err := utils.ReadFile(t.filePath)
	if err != nil {
		if errors.IsFileNotFoundError(err) {
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
	return utils.WriteFile(t.filePath, data, 0644)
}

// Save writes todos to disk.
func (t *TodoList) Save() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.saveLocked()
}

// ──── Validation ────────

// validateTitle checks that a title is non-empty, within length limits,
// and not a duplicate of an existing pending todo.
func (t *TodoList) validateTitle(title string) error {
	errs := map[string]string{}

	trimmed := strings.TrimSpace(title)
	if trimmed == "" {
		errs["title"] = "cannot be empty"
		return &TodoValidationError{Field: "title", Title: title, Errors: errs}
	}
	if len(trimmed) > maxTitleLength {
		errs["title"] = fmt.Sprintf("exceeds maximum length of %d characters", maxTitleLength)
		return &TodoValidationError{Field: "title", Title: title, Errors: errs}
	}

	// Check for duplicates among pending todos (caller must hold at least RLock)
	for _, todo := range t.todos {
		if !todo.Done && strings.EqualFold(todo.Title, trimmed) {
			errs["title"] = "duplicate of existing pending todo"
			return &TodoValidationError{Field: "title", Title: title, Errors: errs}
		}
	}

	return nil
}

// ──── Core Operations ────────

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
// Returns (found, item) where found indicates if a match was found.
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

// ──── Batch Operations ────────

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
func (t *TodoList) BatchDelete(titles []string) (BatchResult, error) {
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
	return result, nil
}

// BatchComplete marks multiple todos as completed by title.
func (t *TodoList) BatchComplete(titles []string) (BatchResult, error) {
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
	return result, nil
}

// ──── Query Operations ────────

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
	return len(t.GetPending())
}

// ──── Rendering ────────

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

// RenderPendingContext returns a short summary of pending todos suitable
// for injection into the system prompt so the agent is aware of outstanding work.
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

// ──── Convenience Functions ────────

var defaultTodoList *TodoList
var defaultOnce sync.Once

func getDefaultTodoList() *TodoList {
	defaultOnce.Do(func() {
		wd, _ := os.Getwd()
		defaultTodoList = NewTodoList(filepath.Join(wd, todoFile))
		defaultTodoList.Load()
	})
	return defaultTodoList
}

// AddItem adds a new todo item (without validation).
func AddItem(title string) TodoItem {
	return getDefaultTodoList().Add(title)
}

// CreateItem creates a todo after validating the title.
func CreateItem(title string) (TodoItem, error) {
	return getDefaultTodoList().CreateTodoItem(title)
}

// CompleteItem marks a todo as completed by title.
func CompleteItem(title string) bool {
	return getDefaultTodoList().Complete(title)
}

// DeleteItem removes a todo by title.
func DeleteItem(title string) bool {
	return getDefaultTodoList().Delete(title)
}

// ListItems returns the rendered todo list.
func ListItems() string {
	return getDefaultTodoList().Render()
}

// GetPendingContext returns pending todos for context injection.
func GetPendingContext() string {
	return getDefaultTodoList().RenderPendingContext()
}

// CountPending returns the number of pending todos.
func CountPending() int {
	return getDefaultTodoList().Count()
}
