// Todo List Management
// Allows YOLO to maintain a persistent todo list with pending and completed items.
// Pending todos are injected into the system prompt so the agent is aware of
// outstanding work across sessions.

package main

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

// ──── Error Types ────────────────────────────────────────────────────

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

// ──── Data Types ─────────────────────────────────────────────────────

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

type BatchFailure struct {
	Title string
	Error string
}

// ──── TodoList ───────────────────────────────────────────────────────

type TodoList struct {
	mu       sync.RWMutex
	todos    []TodoItem
	filePath string
}

var todoList *TodoList

func init() {
	wd, _ := os.Getwd()
	todoList = NewTodoList(filepath.Join(wd, todoFile))
	todoList.Load()
}

func NewTodoList(filePath string) *TodoList {
	return &TodoList{
		todos:    []TodoItem{},
		filePath: filePath,
	}
}

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

func (t *TodoList) Save() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.saveLocked()
}

// ──── Validation ─────────────────────────────────────────────────────

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

// ──── Core Operations ────────────────────────────────────────────────

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

// ──── Batch Operations ───────────────────────────────────────────────

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

// ──── Query Operations ───────────────────────────────────────────────

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

// ──── Rendering ──────────────────────────────────────────────────────

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

// ──── Tool Functions (string-returning wrappers for LLM tools) ───────

func addTodoItem(title string) string {
	if strings.TrimSpace(title) == "" {
		return "Error: TODO title cannot be empty"
	}

	todo := todoList.Add(title)
	return fmt.Sprintf("Added TODO: %s\n   Created: %s", todo.Title, todo.CreatedAt.Format("2006-01-02 15:04:05"))
}

func completeTodoItem(title string) string {
	if strings.TrimSpace(title) == "" {
		return "Error: TODO title cannot be empty"
	}

	if todoList.Complete(title) {
		return fmt.Sprintf("Marked as completed: %s", title)
	}

	return fmt.Sprintf("Error: TODO not found or already completed: %s", title)
}

func deleteTodoItem(title string) string {
	if strings.TrimSpace(title) == "" {
		return "Error: TODO title cannot be empty"
	}

	if todoList.Delete(title) {
		return fmt.Sprintf("Deleted TODO: %s", title)
	}

	return fmt.Sprintf("Error: TODO not found: %s", title)
}

func listTodos() string {
	return todoList.Render()
}

// ──── Tool Executor Methods ──────────────────────────────────────────

func (te *ToolExecutor) addTodo(args map[string]any) string {
	title, ok := args["title"].(string)
	if !ok || strings.TrimSpace(title) == "" {
		return "Error: title parameter is required and cannot be empty"
	}
	return addTodoItem(title)
}

func (te *ToolExecutor) completeTodo(args map[string]any) string {
	title, ok := args["title"].(string)
	if !ok || strings.TrimSpace(title) == "" {
		return "Error: title parameter is required and cannot be empty"
	}
	return completeTodoItem(title)
}

func (te *ToolExecutor) deleteTodo(args map[string]any) string {
	title, ok := args["title"].(string)
	if !ok || strings.TrimSpace(title) == "" {
		return "Error: title parameter is required and cannot be empty"
	}
	return deleteTodoItem(title)
}

func (te *ToolExecutor) listTodosTool(args map[string]any) string {
	return listTodos()
}
