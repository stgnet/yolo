// Todo List Management
// Allows YOLO to maintain a persistent todo list with pending and completed items

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
)

const todoFile = ".todo.json"

type TodoItem struct {
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	Done      bool      `json:"done,omitempty"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
}

type TodoList struct {
	mu       sync.RWMutex
	todos    []TodoItem
	filePath string
}

var todoList *TodoList

func init() {
	// Get working directory and set up todo file path
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

	data, err := os.ReadFile(t.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist yet, that's fine
		}
		return err
	}

	var loadedTodos []TodoItem
	err = json.Unmarshal(data, &loadedTodos)
	if err != nil {
		// If we can't parse the file, start fresh
		t.todos = []TodoItem{}
		return nil
	}

	t.todos = loadedTodos
	return nil
}

func (t *TodoList) Save() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	data, err := json.MarshalIndent(t.todos, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(t.filePath, data, 0644)
}

// Add adds a new todo item
func (t *TodoList) Add(title string) TodoItem {
	t.mu.Lock()
	defer t.mu.Unlock()

	todo := TodoItem{
		Title:     title,
		CreatedAt: time.Now(),
		Done:      false,
	}

	t.todos = append(t.todos, todo)
	t.Save()

	return todo
}

// Complete marks a todo item as completed by title
func (t *TodoList) Complete(title string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	for i := range t.todos {
		if strings.EqualFold(t.todos[i].Title, title) && !t.todos[i].Done {
			t.todos[i].Done = true
			t.todos[i].CompletedAt = time.Now()
			t.Save()
			return true
		}
	}

	return false
}

// GetPending returns all pending (not completed) todos, sorted by creation date
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

// GetCompleted returns all completed todos, sorted by completion date (newest first)
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

// GetAll returns all todos (pending first, then completed)
func (t *TodoList) GetAll() []TodoItem {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make([]TodoItem, len(t.todos))
	copy(result, t.todos)
	return result
}

// Count returns the number of pending todos
func (t *TodoList) Count() int {
	pending := t.GetPending()
	return len(pending)
}

// Render renders the todo list as a formatted string for reports
func (t *TodoList) Render() string {
	var sb strings.Builder

	sb.WriteString("📝 TODO LIST\n")
	sb.WriteString(strings.Repeat("─", 50) + "\n\n")

	pending := t.GetPending()
	completed := t.GetCompleted()

	if len(pending) == 0 && len(completed) == 0 {
		sb.WriteString("No todos yet.\n")
		return sb.String()
	}

	// Show pending items
	if len(pending) > 0 {
		sb.WriteString(fmt.Sprintf("⏳ PENDING (%d)\n", len(pending)))
		for _, todo := range pending {
			createdAtStr := todo.CreatedAt.Format("2006-01-02")
			sb.WriteString(fmt.Sprintf("  ☐ %s [%s]\n", todo.Title, createdAtStr))
		}
		sb.WriteString("\n")
	}

	// Show completed items (limited to last 10)
	if len(completed) > 0 {
		limit := 10
		if len(completed) < limit {
			limit = len(completed)
		}
		
		sb.WriteString(fmt.Sprintf("✅ COMPLETED (last %d of %d)\n", limit, len(completed)))
		for i := 0; i < limit; i++ {
			completedAtStr := completed[i].CompletedAt.Format("2006-01-02")
			sb.WriteString(fmt.Sprintf("  ✓ %s (done: %s)\n", completed[i].Title, completedAtStr))
		}
		if len(completed) > limit {
			sb.WriteString(fmt.Sprintf("  ... and %d more completed items\n", len(completed)-limit))
		}
	}

	return sb.String()
}

// ──── Tool Functions ─────

func addTodoItem(title string) string {
	if strings.TrimSpace(title) == "" {
		return "Error: TODO title cannot be empty"
	}

	todo := todoList.Add(title)
	return fmt.Sprintf("✅ Added TODO: %s\n   Created: %s", todo.Title, todo.CreatedAt.Format("2006-01-02 15:04:05"))
}

func completeTodoItem(title string) string {
	if strings.TrimSpace(title) == "" {
		return "Error: TODO title cannot be empty"
	}

	if todoList.Complete(title) {
		return fmt.Sprintf("✅ Marked as completed: %s", title)
	}

	return fmt.Sprintf("❌ TODO not found or already completed: %s", title)
}

func listTodos() string {
	return todoList.Render()
}

// ──── Tool Executions ───────

// addTodo executes the add_todo tool
func (te *ToolExecutor) addTodo(args map[string]any) string {
	title, ok := args["title"].(string)
	if !ok || strings.TrimSpace(title) == "" {
		return "Error: title parameter is required and cannot be empty"
	}
	return addTodoItem(title)
}

// completeTodo executes the complete_todo tool
func (te *ToolExecutor) completeTodo(args map[string]any) string {
	title, ok := args["title"].(string)
	if !ok || strings.TrimSpace(title) == "" {
		return "Error: title parameter is required and cannot be empty"
	}
	return completeTodoItem(title)
}

// listTodosTool executes the list_todos tool
func (te *ToolExecutor) listTodosTool(args map[string]any) string {
	return listTodos()
}
