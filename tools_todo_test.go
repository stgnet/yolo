package main

import (
	"strings"
	"testing"
)

func TestNewTodoList(t *testing.T) {
	tmpDir := t.TempDir()
	todoFile := tmpDir + "/test_todos.json"
	
	list := NewTodoList(todoFile)
	if list == nil {
		t.Fatal("NewTodoList returned nil")
	}
	if len(list.todos) != 0 {
		t.Errorf("Expected empty todos slice, got %d items", len(list.todos))
	}
}

func TestTodoList_Add(t *testing.T) {
	tmpDir := t.TempDir()
	todoFile := tmpDir + "/test_todos.json"
	list := NewTodoList(todoFile)
	
	// Test adding a valid todo
	todo, err := list.Add("Test task")
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if todo == nil {
		t.Fatal("Add returned nil todo")
	}
	if todo.Title != "Test task" {
		t.Errorf("Expected title 'Test task', got '%s'", todo.Title)
	}
	if todo.Completed {
		t.Error("Expected Completed to be false for new todo")
	}
	
	// Verify todos slice has the item
	if len(list.todos) != 1 {
		t.Errorf("Expected 1 todo, got %d", len(list.todos))
	}
}

func TestTodoList_Add_EmptyTitle(t *testing.T) {
	tmpDir := t.TempDir()
	todoFile := tmpDir + "/test_todos.json"
	list := NewTodoList(todoFile)
	
	// Test adding with empty title should fail
	_, err := list.Add("")
	if err == nil {
		t.Error("Expected error for empty title")
	}
	
	// Test adding with whitespace only title should fail
	_, err = list.Add("   ")
	if err == nil {
		t.Error("Expected error for whitespace-only title")
	}
}

func TestTodoList_Complete(t *testing.T) {
	tmpDir := t.TempDir()
	todoFile := tmpDir + "/test_todos.json"
	list := NewTodoList(todoFile)
	
	todo, _ := list.Add("Test task to complete")
	
	// Complete the todo - returns (bool, error)
	found, err := list.Complete(todo.Title)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if !found {
		t.Error("Expected found=true for valid completion")
	}
	
	// Verify todo is marked as completed
	if len(list.todos) != 1 {
		t.Fatalf("Expected 1 todo, got %d", len(list.todos))
	}
	if !list.todos[0].Completed {
		t.Error("Expected todo to be marked as completed")
	}
}

func TestTodoList_Complete_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	todoFile := tmpDir + "/test_todos.json"
	list := NewTodoList(todoFile)
	
	found, err := list.Complete("Non-existent task")
	if found {
		t.Error("Expected found=false for non-existent todo")
	}
	if err != nil {
		t.Errorf("Complete should not return error for non-existent todo, got: %v", err)
	}
}

func TestTodoList_Complete_AlreadyCompleted(t *testing.T) {
	tmpDir := t.TempDir()
	todoFile := tmpDir + "/test_todos.json"
	list := NewTodoList(todoFile)
	
	todo, _ := list.Add("Task to complete twice")
	// Complete once
	list.Complete(todo.Title)
	
	// Try to complete again - should return error
	found, err := list.Complete(todo.Title)
	if found {
		t.Error("Expected found=false for already completed todo")
	}
	if err == nil || !strings.Contains(err.Error(), "already completed") {
		t.Errorf("Expected 'already completed' error, got: %v", err)
	}
}

func TestTodoList_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	todoFile := tmpDir + "/test_todos.json"
	list := NewTodoList(todoFile)
	
	// Add a todo first
	todo, _ := list.Add("Task to delete")
	if len(list.todos) != 1 {
		t.Fatalf("Expected 1 todo before delete, got %d", len(list.todos))
	}
	
	// Delete the todo - returns (bool, error)
	found, err := list.Delete(todo.Title)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if !found {
		t.Error("Expected found=true for valid deletion")
	}
	
	// Verify todo was removed
	if len(list.todos) != 0 {
		t.Errorf("Expected 0 todos after delete, got %d", len(list.todos))
	}
}

func TestTodoList_Delete_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	todoFile := tmpDir + "/test_todos.json"
	list := NewTodoList(todoFile)
	
	found, err := list.Delete("Non-existent task")
	if found {
		t.Error("Expected found=false for non-existent todo")
	}
	if err != nil {
		t.Errorf("Delete should not return error for non-existent todo, got: %v", err)
	}
}

func TestTodoList_FormatAllTodos(t *testing.T) {
	tmpDir := t.TempDir()
	todoFile := tmpDir + "/test_todos.json"
	list := NewTodoList(todoFile)
	
	// Test empty list
	result := list.FormatAllTodos()
	if result != "No todos found." {
		t.Errorf("Expected 'No todos found.', got '%s'", result)
	}
	
	// Add some todos
	list.Add("Task 1")
	list.Add("Task 2")
	list.Complete("Task 2")
	
	result = list.FormatAllTodos()
	if !strings.Contains(result, "=== TODO LIST ===") {
		t.Error("Result should contain header")
	}
	if !strings.Contains(result, "Task 1") || !strings.Contains(result, "Task 2") {
		t.Error("Result should contain all tasks")
	}
	if !strings.Contains(result, "Total:") {
		t.Error("Result should contain total count")
	}
}

func TestTodoList_FormatPendingTodos(t *testing.T) {
	tmpDir := t.TempDir()
	todoFile := tmpDir + "/test_todos.json"
	list := NewTodoList(todoFile)
	
	// Test empty list
	result := list.FormatPendingTodos()
	if result != "No pending todos." {
		t.Errorf("Expected 'No pending todos.', got '%s'", result)
	}
	
	// Add some todos
	list.Add("Task 1")
	list.Add("Task 2")
	list.Complete("Task 2")
	
	result = list.FormatPendingTodos()
	if !strings.Contains(result, "=== PENDING TODOS ===") {
		t.Error("Result should contain header")
	}
	if strings.Contains(result, "Task 2") {
		t.Error("Result should not contain completed tasks")
	}
	if !strings.Contains(result, "Task 1") {
		t.Error("Result should contain Task 1")
	}
}

func TestTodoList_GetStats(t *testing.T) {
	tmpDir := t.TempDir()
	todoFile := tmpDir + "/test_todos.json"
	list := NewTodoList(todoFile)
	
	// Test empty list
	total, pending, completed := list.GetStats()
	if total != 0 || pending != 0 || completed != 0 {
		t.Errorf("Expected all zeros for empty list, got (%d, %d, %d)", total, pending, completed)
	}
	
	// Add some todos
	list.Add("Task 1")
	list.Add("Task 2")
	list.Complete("Task 2")
	
	total, pending, completed = list.GetStats()
	if total != 2 || pending != 1 || completed != 1 {
		t.Errorf("Expected (2, 1, 1), got (%d, %d, %d)", total, pending, completed)
	}
}

func TestTodoList_Persist(t *testing.T) {
	tmpDir := t.TempDir()
	todoFile := tmpDir + "/test_todos.json"
	
	// Create list and add todos
	list1 := NewTodoList(todoFile)
	list1.Add("Task to persist")
	list1.Add("Another task")
	
	// Simulate loading fresh (like a new process)
	list2 := NewTodoList(todoFile)
	
	if len(list2.todos) != 2 {
		t.Errorf("Expected 2 persisted todos, got %d", len(list2.todos))
	}
	if list2.todos[0].Title != "Task to persist" || list2.todos[1].Title != "Another task" {
		t.Errorf("Expected correct titles, got '%s' and '%s'", list2.todos[0].Title, list2.todos[1].Title)
	}
}

func TestTodoList_GetAllTodos(t *testing.T) {
	tmpDir := t.TempDir()
	todoFile := tmpDir + "/test_todos.json"
	list := NewTodoList(todoFile)
	
	list.Add("Task 1")
	list.Add("Task 2")
	
	all := list.GetAllTodos()
	if len(all) != 2 {
		t.Errorf("Expected 2 todos, got %d", len(all))
	}
	
	// Modify the returned slice should not affect internal state
	all[0].Title = "Modified"
	internal := list.GetAllTodos()
	if internal[0].Title == "Modified" {
		t.Error("GetAllTodos should return a copy, not reference to internal data")
	}
}

func TestTodoList_SortByCreation(t *testing.T) {
	tmpDir := t.TempDir()
	todoFile := tmpDir + "/test_todos.json"
	list := NewTodoList(todoFile)
	
	// Manually set todos out of order (simulating a corrupted file or manual insertion)
	list.todos = []Todo{
		{Title: "Third", CreatedAt: list.nowPlusMinutes(10)},
		{Title: "First", CreatedAt: list.nowPlusMinutes(0)},
		{Title: "Second", CreatedAt: list.nowPlusMinutes(5)},
	}
	
	list.SortByCreation()
	
	if list.todos[0].Title != "First" || list.todos[1].Title != "Second" || list.todos[2].Title != "Third" {
		t.Errorf("Todos should be sorted by creation time")
	}
}

func (tl *TodoList) nowPlusMinutes(m int64) interface{} {
	return nil // Not used, just for helper reference
}
