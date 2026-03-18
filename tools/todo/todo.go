// Package todo provides todo list management functionality.
package todo

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// TodoItem represents a single todo item
type TodoItem struct {
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

// TodoManager handles todo list operations
type TodoManager struct {
	filePath string
	items    []TodoItem
}

// NewTodoManager creates a new TodoManager with the default file location
func NewTodoManager() *TodoManager {
	return &TodoManager{
		filePath: filepath.Join(os.Getenv("HOME"), ".todo.json"),
	}
}

// Load loads todos from the JSON file
func (tm *TodoManager) Load() error {
	data, err := os.ReadFile(tm.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			tm.items = []TodoItem{}
			return nil
		}
		return err
	}

	err = json.Unmarshal(data, &tm.items)
	if err != nil {
		return err
	}

	return nil
}

// Save persists todos to the JSON file
func (tm *TodoManager) Save() error {
	data, err := json.MarshalIndent(tm.items, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(tm.filePath, data, 0644)
}

// Add adds a new todo item
func (tm *TodoManager) Add(title string) error {
	tm.items = append(tm.items, TodoItem{Title: title, Completed: false})
	return tm.Save()
}

// Complete marks a todo as completed by exact title match
func (tm *TodoManager) Complete(title string) (bool, error) {
	for i := range tm.items {
		if tm.items[i].Title == title && !tm.items[i].Completed {
			tm.items[i].Completed = true
			return true, tm.Save()
		}
	}
	return false, nil
}

// Delete removes a todo item by exact title match
func (tm *TodoManager) Delete(title string) (bool, error) {
	for i := range tm.items {
		if tm.items[i].Title == title {
			tm.items = append(tm.items[:i], tm.items[i+1:]...)
			return true, tm.Save()
		}
	}
	return false, nil
}

// List returns all todo items (pending and completed)
func (tm *TodoManager) List() []TodoItem {
	return tm.items
}

// GetPendingTodos returns only pending (incomplete) todos
func (tm *TodoManager) GetPendingTodos() []TodoItem {
	var pending []TodoItem
	for _, item := range tm.items {
		if !item.Completed {
			pending = append(pending, item)
		}
	}
	return pending
}

// CountPending returns the number of pending todos
func (tm *TodoManager) CountPending() int {
	count := 0
	for _, item := range tm.items {
		if !item.Completed {
			count++
		}
	}
	return count
}

// CountCompleted returns the number of completed todos
func (tm *TodoManager) CountCompleted() int {
	count := 0
	for _, item := range tm.items {
		if item.Completed {
			count++
		}
	}
	return count
}
