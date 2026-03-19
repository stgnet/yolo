package todo

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

const defaultTodoFile = ".todo.json"

// Todo represents a single todo item.
type Todo struct {
	Title     string    `json:"title"`
	Completed bool      `json:"completed"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TodoList manages a collection of todos with thread-safe operations.
type TodoList struct {
	mu     sync.RWMutex
	todos  []Todo
	filePath string
}

// Global todo list singleton
var globalTodoList *TodoList
var globalOnce sync.Once

// GetGlobalTodoList returns a singleton TodoList instance.
func GetGlobalTodoList() *TodoList {
	globalOnce.Do(func() {
		globalTodoList = NewTodoList(defaultTodoFile)
	})
	return globalTodoList
}

// NewTodoList creates a new TodoList with the given file path for persistence.
func NewTodoList(filePath string) *TodoList {
	tl := &TodoList{
		filePath: filePath,
		todos:    make([]Todo, 0),
	}
	tl.loadFromFile()
	return tl
}

// loadFromFile loads todos from the JSON file if it exists.
func (tl *TodoList) loadFromFile() {
	data, err := os.ReadFile(tl.filePath)
	if err != nil {
		// File doesn't exist or can't be read - start with empty list
		return
	}

	var loadedTodos []Todo
	if err := json.Unmarshal(data, &loadedTodos); err != nil {
		// Invalid JSON - start with empty list
		return
	}

	tl.todos = loadedTodos
}

// saveToFile persists todos to the JSON file.
func (tl *TodoList) saveToFile() error {
	data, err := json.MarshalIndent(tl.todos, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal todos: %w", err)
	}

	if err := os.WriteFile(tl.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write todos file: %w", err)
	}

	return nil
}

// Add creates a new todo item.
func (tl *TodoList) Add(title string) (*Todo, error) {
	if strings.TrimSpace(title) == "" {
		return nil, fmt.Errorf("title cannot be empty")
	}

	now := time.Now()
	todo := &Todo{
		Title:     title,
		Completed: false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	tl.mu.Lock()
	defer tl.mu.Unlock()

	tl.todos = append(tl.todos, *todo)
	if err := tl.saveToFile(); err != nil {
		// Rollback on save failure
		tl.todos = tl.todos[:len(tl.todos)-1]
		return nil, fmt.Errorf("failed to save: %w", err)
	}

	return todo, nil
}

// Complete marks a todo as completed by title (case-insensitive partial match).
func (tl *TodoList) Complete(title string) (bool, error) {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	for i := range tl.todos {
		if strings.Contains(strings.ToLower(tl.todos[i].Title), strings.ToLower(title)) {
			if tl.todos[i].Completed {
				return false, fmt.Errorf("already completed")
			}
			tl.todos[i].Completed = true
			tl.todos[i].UpdatedAt = time.Now()

			if err := tl.saveToFile(); err != nil {
				return false, fmt.Errorf("failed to save: %w", err)
			}
			return true, nil
		}
	}

	return false, nil // Not found is not an error, just no match
}

// Delete removes a todo by title (case-insensitive partial match).
func (tl *TodoList) Delete(title string) (bool, error) {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	for i := range tl.todos {
		if strings.Contains(strings.ToLower(tl.todos[i].Title), strings.ToLower(title)) {
			tl.todos = append(tl.todos[:i], tl.todos[i+1:]...)

			if err := tl.saveToFile(); err != nil {
				return false, fmt.Errorf("failed to save: %w", err)
			}
			return true, nil
		}
	}

	return false, nil // Not found is not an error
}

// FormatAllTodos returns a formatted string of all todos.
func (tl *TodoList) FormatAllTodos() string {
	tl.mu.RLock()
	defer tl.mu.RUnlock()

	if len(tl.todos) == 0 {
		return "No todos found."
	}

	var sb strings.Builder
	sb.WriteString("=== TODO LIST ===\n\n")

	pendingCount := 0
	completedCount := 0

	for i, todo := range tl.todos {
		status := "⬜"
		if todo.Completed {
			status = "✓"
			completedCount++
		} else {
			pendingCount++
		}

		timestamp := todo.CreatedAt.Format("Jan 2, 2006 3:04PM")
		sb.WriteString(fmt.Sprintf("%s %d. %s\n    Created: %s\n\n", status, i+1, todo.Title, timestamp))
	}

	sb.WriteString(fmt.Sprintf("Total: %d pending, %d completed\n", pendingCount, completedCount))
	return sb.String()
}

// FormatPendingTodos returns a formatted string of pending (incomplete) todos only.
func (tl *TodoList) FormatPendingTodos() string {
	tl.mu.RLock()
	defer tl.mu.RUnlock()

	var pending []Todo
	for _, todo := range tl.todos {
		if !todo.Completed {
			pending = append(pending, todo)
		}
	}

	if len(pending) == 0 {
		return "No pending todos."
	}

	var sb strings.Builder
	sb.WriteString("=== PENDING TODOS ===\n\n")

	for i, todo := range pending {
		timestamp := todo.CreatedAt.Format("Jan 2, 2006 3:04PM")
		sb.WriteString(fmt.Sprintf("%d. %s\n    Created: %s\n\n", i+1, todo.Title, timestamp))
	}

	sb.WriteString(fmt.Sprintf("Total pending: %d\n", len(pending)))
	return sb.String()
}

// GetStats returns summary statistics about the todo list.
func (tl *TodoList) GetStats() (total, pending, completed int) {
	tl.mu.RLock()
	defer tl.mu.RUnlock()

	for _, todo := range tl.todos {
		total++
		if todo.Completed {
			completed++
		} else {
			pending++
		}
	}
	return total, pending, completed
}

// SortByCreation sorts todos by creation time (oldest first).
func (tl *TodoList) SortByCreation() {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	sort.Slice(tl.todos, func(i, j int) bool {
		return tl.todos[i].CreatedAt.Before(tl.todos[j].CreatedAt)
	})
}

// GetAllTodos returns a copy of all todos (thread-safe).
func (tl *TodoList) GetAllTodos() []Todo {
	tl.mu.RLock()
	defer tl.mu.RUnlock()
	
	result := make([]Todo, len(tl.todos))
	copy(result, tl.todos)
	return result
}
