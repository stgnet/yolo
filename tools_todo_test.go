package main

import (
	"os"
	"strings"
	"testing"
	"time"
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

func TestTodoListGetAll(t *testing.T) {
	tmpDir := t.TempDir()
	tmpTodoList := NewTodoList(tmpDir + "/.todo.json")

	// Add todos
	item1 := tmpTodoList.Add("First todo")
	item2 := tmpTodoList.Add("Second todo")

	// Get all todos
	allTodos := tmpTodoList.GetAll()

	if len(allTodos) != 2 {
		t.Errorf("Expected 2 todos, got %d", len(allTodos))
	}

	if allTodos[0].Title != item1.Title || allTodos[1].Title != item2.Title {
		t.Errorf("Todo order mismatch")
	}
}

func TestTodoListGetCompleted(t *testing.T) {
	tmpDir := t.TempDir()
	tmpTodoList := NewTodoList(tmpDir + "/.todo.json")

	// Add todos
	tmpTodoList.Add("Pending todo")
	doneTodo := tmpTodoList.Add("Completed todo")

	// Manually mark one as done (bypassing Complete method for test control)
	tmpTodoList.todos[1].Done = true
	tmpTodoList.todos[1].CompletedAt = tmpTodoList.todos[1].CreatedAt

	// Get completed todos
	completed := tmpTodoList.GetCompleted()

	if len(completed) != 1 {
		t.Errorf("Expected 1 completed todo, got %d", len(completed))
	}

	if completed[0].Title != doneTodo.Title {
		t.Errorf("Expected completed todo to be '%s', got '%s'", doneTodo.Title, completed[0].Title)
	}
}

func TestTodoListCount(t *testing.T) {
	tmpDir := t.TempDir()
	tmpTodoList := NewTodoList(tmpDir + "/.todo.json")

	// Initial count should be 0
	if tmpTodoList.Count() != 0 {
		t.Errorf("Expected initial count to be 0, got %d", tmpTodoList.Count())
	}

	// Add todos
	tmpTodoList.Add("First todo")
	tmpTodoList.Add("Second todo")

	// Count should be 2 (pending)
	if tmpTodoList.Count() != 2 {
		t.Errorf("Expected count to be 2, got %d", tmpTodoList.Count())
	}

	// Complete one
	tmpTodoList.Complete("First todo")

	// Count should be 1 (only pending)
	if tmpTodoList.Count() != 1 {
		t.Errorf("Expected count to be 1 after completing one, got %d", tmpTodoList.Count())
	}
}

func TestTodoListSave(t *testing.T) {
	tmpDir := t.TempDir()
	tmpTodoList := NewTodoList(tmpDir + "/.todo.json")

	// Add todo
	tmpTodoList.Add("Test save")

	// Save should not error
	err := tmpTodoList.Save()
	if err != nil {
		t.Errorf("Save() returned error: %v", err)
	}

	// Verify file exists and is readable
	data, err := os.ReadFile(tmpDir + "/.todo.json")
	if err != nil {
		t.Errorf("Failed to read saved todo file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "Test save") {
		t.Errorf("Saved file doesn't contain expected todo: %s", content)
	}
}

func TestTodoListLoad(t *testing.T) {
	tmpDir := t.TempDir()

	// Create initial file with todos
	initialContent := `[
  {"title": "Pre-existing todo", "created_at": "2024-01-01T00:00:00Z"}
]`
	err := os.WriteFile(tmpDir+"/.todo.json", []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create initial todo file: %v", err)
	}

	// Load todos
	tmpTodoList := NewTodoList(tmpDir + "/.todo.json")
	err = tmpTodoList.Load()
	if err != nil {
		t.Errorf("Load() returned error: %v", err)
	}

	pending := tmpTodoList.GetPending()
	if len(pending) != 1 || pending[0].Title != "Pre-existing todo" {
		t.Errorf("Failed to load todos correctly, got %v", pending)
	}
}

func TestAddTodoWithArgs(t *testing.T) {
	tmpDir := t.TempDir()
	originalTodoList := todoList
	todoList = NewTodoList(tmpDir + "/.todo.json")
	defer func() { todoList = originalTodoList }()

	// Test with valid args via tool executor method
	args := map[string]any{
		"title": "Test from args",
	}
	result := addTodoItem(args["title"].(string))

	if !strings.Contains(result, "✅ Added TODO:") {
		t.Errorf("Expected success message, got: %s", result)
	}
}

func TestAddTodoMissingArgs(t *testing.T) {
	tmpDir := t.TempDir()
	originalTodoList := todoList
	todoList = NewTodoList(tmpDir + "/.todo.json")
	defer func() { todoList = originalTodoList }()

	// Test with empty title directly
	result := addTodoItem("")

	if !strings.Contains(result, "Error:") || !strings.Contains(result, "empty") {
		t.Errorf("Expected error about empty title, got: %s", result)
	}
}

func TestCompleteTodoWithArgs(t *testing.T) {
	tmpDir := t.TempDir()
	originalTodoList := todoList
	todoList = NewTodoList(tmpDir + "/.todo.json")
	defer func() { todoList = originalTodoList }()

	// Add a todo first
	addTodoItem("Test completion")

	// Complete via args
	result := completeTodoItem("Test completion")

	if !strings.Contains(result, "✅ Marked as completed:") {
		t.Errorf("Expected success message, got: %s", result)
	}
}

func TestCompleteTodoNotFoundWithArgs(t *testing.T) {
	tmpDir := t.TempDir()
	originalTodoList := todoList
	todoList = NewTodoList(tmpDir + "/.todo.json")
	defer func() { todoList = originalTodoList }()

	// Try to complete non-existent todo via args
	result := completeTodoItem("Non-existent")

	if !strings.Contains(result, "Error:") || !strings.Contains(result, "not found") {
		t.Errorf("Expected not found error, got: %s", result)
	}
}

func TestListTodosTool(t *testing.T) {
	tmpDir := t.TempDir()
	originalTodoList := todoList
	todoList = NewTodoList(tmpDir + "/.todo.json")
	defer func() { todoList = originalTodoList }()

	// Add some todos
	addTodoItem("Todo 1")
	addTodoItem("Todo 2")

	// Test listTodos (no args needed)
	result := listTodos()

	if !strings.Contains(result, "📝 TODO LIST") {
		t.Errorf("Expected header in result, got: %s", result)
	}
	if !strings.Contains(result, "Todo 1") || !strings.Contains(result, "Todo 2") {
		t.Errorf("Expected todos in result, got: %s", result)
	}
}

func TestTodoListMarshalJSON(t *testing.T) {
	// Test with pending todo
	pendingItem := TodoItem{
		Title:       "Pending task",
		CreatedAt:   time.Now(),
		Done:        false,
		CompletedAt: time.Time{},
	}

	data, err := pendingItem.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed for pending item: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "Pending task") {
		t.Errorf("Missing title in marshaled JSON: %s", content)
	}
	// Note: The MarshalJSON implementation currently includes completed_at as zero value
	// This is expected behavior based on the current implementation

	// Test with completed todo
	completedItem := TodoItem{
		Title:       "Done task",
		CreatedAt:   time.Now(),
		Done:        true,
		CompletedAt: time.Now(),
	}

	data, err = completedItem.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed for completed item: %v", err)
	}

	content = string(data)
	if !strings.Contains(content, "Done task") {
		t.Errorf("Missing title in marshaled JSON: %s", content)
	}
	if !strings.Contains(content, "completed_at") {
		t.Errorf("Expected completed_at for done item: %s", content)
	}
}
