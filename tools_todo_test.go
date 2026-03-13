package main

import (
	"os"
	"strings"
	"testing"
)

func TestAddTodoItem(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Save original todoList and restore later
	originalTodoList := todoList

	// Create a new TodoList in temp dir
	tmpTodoList := NewTodoList(tmpDir + "/.todo.json")
	todoList = tmpTodoList

	// Add a new todo
	result := addTodoItem("Test todo")

	// Restore original
	todoList = originalTodoList

	// Verify the result contains expected text
	if !strings.Contains(result, "✅ Added TODO:") {
		t.Errorf("Expected '✅ Added TODO:' in result, got: %s", result)
	}
	if !strings.Contains(result, "Test todo") {
		t.Errorf("Expected 'Test todo' in result, got: %s", result)
	}
}

func TestCompleteTodoItem(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Save original todoList and restore later
	originalTodoList := todoList

	// Create a new TodoList in temp dir
	tmpTodoList := NewTodoList(tmpDir + "/.todo.json")
	todoList = tmpTodoList

	// Add a todo first
	addTodoItem("Test todo to complete")

	// Complete the todo
	result := completeTodoItem("Test todo to complete")

	// Restore original
	todoList = originalTodoList

	// Verify the result contains expected text
	if !strings.Contains(result, "✅ Marked as completed:") {
		t.Errorf("Expected '✅ Marked as completed:' in result, got: %s", result)
	}
}

func TestCompleteTodoNotFound(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Save original todoList and restore later
	originalTodoList := todoList

	// Create a new TodoList in temp dir (empty)
	tmpTodoList := NewTodoList(tmpDir + "/.todo.json")
	todoList = tmpTodoList

	// Try to complete non-existent todo
	result := completeTodoItem("Non-existent todo")

	// Restore original
	todoList = originalTodoList

	// Verify the result contains expected text
	if !strings.Contains(result, "❌ TODO not found") {
		t.Errorf("Expected '❌ TODO not found' in result, got: %s", result)
	}
}

func TestAddTodoEmptyTitle(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Save original todoList and restore later
	originalTodoList := todoList

	// Create a new TodoList in temp dir
	tmpTodoList := NewTodoList(tmpDir + "/.todo.json")
	todoList = tmpTodoList

	// Add empty title
	result := addTodoItem("")

	// Restore original
	todoList = originalTodoList

	// Verify the result contains expected text
	if !strings.Contains(result, "Error:") {
		t.Errorf("Expected 'Error:' in result for empty title, got: %s", result)
	}
}

func TestListTodos(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Save original todoList and restore later
	originalTodoList := todoList

	// Create a new TodoList in temp dir
	tmpTodoList := NewTodoList(tmpDir + "/.todo.json")
	todoList = tmpTodoList

	// Add todos
	addTodoItem("First todo")
	addTodoItem("Second todo")

	// List todos should not panic and contain expected text
	result := listTodos()

	// Restore original
	todoList = originalTodoList

	if !strings.Contains(result, "📝 TODO LIST") {
		t.Errorf("Expected '📝 TODO LIST' in result, got: %s", result)
	}
	if !strings.Contains(result, "First todo") {
		t.Errorf("Expected 'First todo' in result, got: %s", result)
	}
}

func TestTodoFilePersistence(t *testing.T) {
	tmpDir := t.TempDir()

	// Save original todoList and restore later
	originalTodoList := todoList

	// Create a new TodoList in temp dir
	tmpTodoList := NewTodoList(tmpDir + "/.todo.json")
	todoList = tmpTodoList

	// Add todos
	addTodoItem("Persistent todo")

	// Verify file exists
	file, err := os.ReadFile(tmpDir + "/.todo.json")
	if err != nil {
		t.Fatalf("Expected todo file to exist: %v", err)
	}

	// Restore original
	todoList = originalTodoList

	// Verify file contains expected data
	content := string(file)
	if !strings.Contains(content, "Persistent todo") {
		t.Errorf("Expected 'Persistent todo' in file content, got: %s", content)
	}
	// Note: The json.Marshal custom logic may omit 'done' when false due to omitempty
	// Just verify the title exists and structure is valid
	if !strings.Contains(content, "title") {
		t.Errorf("Expected 'title' field in file content, got: %s", content)
	}
}
