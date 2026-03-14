package main

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestAddTodoItem(t *testing.T) {
	tmpDir := t.TempDir()
	originalTodoList := todoList
	tmpTodoList := NewTodoList(tmpDir + "/.todo.json")
	todoList = tmpTodoList
	defer func() { todoList = originalTodoList }()

	result := addTodoItem("Test todo")

	if !strings.Contains(result, "✅ Added TODO:") {
		t.Errorf("Expected '✅ Added TODO:' in result, got: %s", result)
	}
	if !strings.Contains(result, "Test todo") {
		t.Errorf("Expected 'Test todo' in result, got: %s", result)
	}
}

func TestCompleteTodoItem(t *testing.T) {
	tmpDir := t.TempDir()
	originalTodoList := todoList
	tmpTodoList := NewTodoList(tmpDir + "/.todo.json")
	todoList = tmpTodoList
	defer func() { todoList = originalTodoList }()

	addTodoItem("Test todo to complete")
	result := completeTodoItem("Test todo to complete")

	if !strings.Contains(result, "✅ Marked as completed:") {
		t.Errorf("Expected '✅ Marked as completed:' in result, got: %s", result)
	}
}

func TestCompleteTodoNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	originalTodoList := todoList
	tmpTodoList := NewTodoList(tmpDir + "/.todo.json")
	todoList = tmpTodoList
	defer func() { todoList = originalTodoList }()

	result := completeTodoItem("Non-existent todo")

	if !strings.Contains(result, "❌ TODO not found") {
		t.Errorf("Expected '❌ TODO not found' in result, got: %s", result)
	}
}

func TestAddTodoEmptyTitle(t *testing.T) {
	tmpDir := t.TempDir()
	originalTodoList := todoList
	tmpTodoList := NewTodoList(tmpDir + "/.todo.json")
	todoList = tmpTodoList
	defer func() { todoList = originalTodoList }()

	result := addTodoItem("")

	if !strings.Contains(result, "Error:") || !strings.Contains(result, "empty") {
		t.Errorf("Expected error about empty title, got: %s", result)
	}
}

func TestListTodos(t *testing.T) {
	tmpDir := t.TempDir()
	originalTodoList := todoList
	tmpTodoList := NewTodoList(tmpDir + "/.todo.json")
	todoList = tmpTodoList
	defer func() { todoList = originalTodoList }()

	addTodoItem("First todo")
	addTodoItem("Second todo")
	result := listTodos()

	if !strings.Contains(result, "📝 TODO LIST") {
		t.Errorf("Expected '📝 TODO LIST' in result, got: %s", result)
	}
	if !strings.Contains(result, "First todo") {
		t.Errorf("Expected 'First todo' in result, got: %s", result)
	}
}

func TestTodoFilePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	originalTodoList := todoList
	tmpTodoList := NewTodoList(tmpDir + "/.todo.json")
	todoList = tmpTodoList
	defer func() { todoList = originalTodoList }()

	addTodoItem("Persistent todo")

	file, err := os.ReadFile(tmpDir + "/.todo.json")
	if err != nil {
		t.Fatalf("Expected todo file to exist: %v", err)
	}

	content := string(file)
	if !strings.Contains(content, "Persistent todo") {
		t.Errorf("Expected 'Persistent todo' in file content, got: %s", content)
	}
	if !strings.Contains(content, "title") {
		t.Errorf("Expected 'title' field in file content, got: %s", content)
	}
}

func TestTodoListGetAll(t *testing.T) {
	tmpDir := t.TempDir()
	tmpTodoList := NewTodoList(tmpDir + "/.todo.json")

	item1 := tmpTodoList.Add("First todo")
	item2 := tmpTodoList.Add("Second todo")

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

	tmpTodoList.Add("Pending todo")
	doneTodo := tmpTodoList.Add("Completed todo")
	tmpTodoList.todos[1].Done = true
	tmpTodoList.todos[1].CompletedAt = tmpTodoList.todos[1].CreatedAt

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

	if tmpTodoList.Count() != 0 {
		t.Errorf("Expected initial count to be 0, got %d", tmpTodoList.Count())
	}

	tmpTodoList.Add("First todo")
	tmpTodoList.Add("Second todo")

	if tmpTodoList.Count() != 2 {
		t.Errorf("Expected count to be 2, got %d", tmpTodoList.Count())
	}

	tmpTodoList.Complete("First todo")

	if tmpTodoList.Count() != 1 {
		t.Errorf("Expected count to be 1 after completing one, got %d", tmpTodoList.Count())
	}
}

func TestTodoListSave(t *testing.T) {
	tmpDir := t.TempDir()
	tmpTodoList := NewTodoList(tmpDir + "/.todo.json")

	tmpTodoList.Add("Test save")
	err := tmpTodoList.Save()
	if err != nil {
		t.Errorf("Save() returned error: %v", err)
	}

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

	initialContent := `[
  {"title": "Pre-existing todo", "created_at": "2024-01-01T00:00:00Z"}
]`
	err := os.WriteFile(tmpDir+"/.todo.json", []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create initial todo file: %v", err)
	}

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

	addTodoItem("Test completion")
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

	addTodoItem("Todo 1")
	addTodoItem("Todo 2")

	result := listTodos()

	if !strings.Contains(result, "📝 TODO LIST") {
		t.Errorf("Expected header in result, got: %s", result)
	}
	if !strings.Contains(result, "Todo 1") || !strings.Contains(result, "Todo 2") {
		t.Errorf("Expected todos in result, got: %s", result)
	}
}

func TestTodoListMarshalJSON(t *testing.T) {
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

// Test Case-Insensitive Matching
func TestCaseInsensitiveSearch(t *testing.T) {
	tmpDir := t.TempDir()
	tmpTodoList := NewTodoList(tmpDir + "/.todo.json")

	tmpTodoList.Add("Test Todo")

	// Should find with different case
	found, _ := tmpTodoList.FindTodo("TEST TODO")
	if !found {
		t.Error("Expected case-insensitive search to find 'TEST TODO'")
	}

	// Complete should also be case-insensitive
	if !tmpTodoList.Complete("test todo") {
		t.Error("Expected Complete to work case-insensitively")
	}
}

// Test Title Validation
func TestValidateTitleEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	tmpTodoList := NewTodoList(tmpDir + "/.todo.json")

	err := tmpTodoList.validateTitle("")
	if err == nil {
		t.Error("Expected validation error for empty title")
	}

	_, ok := err.(*TodoValidationError)
	if !ok {
		t.Errorf("Expected TodoValidationError, got %T", err)
	}
}

func TestValidateTitleWhitespaceOnly(t *testing.T) {
	tmpDir := t.TempDir()
	tmpTodoList := NewTodoList(tmpDir + "/.todo.json")

	err := tmpTodoList.validateTitle("   ")
	if err == nil {
		t.Error("Expected validation error for whitespace-only title")
	}
}

func TestValidateTitleTooLong(t *testing.T) {
	tmpDir := t.TempDir()
	tmpTodoList := NewTodoList(tmpDir + "/.todo.json")

	longTitle := strings.Repeat("a", 300)
	err := tmpTodoList.validateTitle(longTitle)
	if err == nil {
		t.Error("Expected validation error for title exceeding max length")
	}

	if ve, ok := err.(*TodoValidationError); ok {
		if _, hasField := ve.Errors["title"]; !hasField {
			t.Errorf("Expected 'title' error in TodoValidationError, got errors: %v", ve.Errors)
		}
	}
}

func TestValidateTitleDuplicate(t *testing.T) {
	tmpDir := t.TempDir()
	tmpTodoList := NewTodoList(tmpDir + "/.todo.json")

	tmpTodoList.Add("Duplicate test")

	err := tmpTodoList.validateTitle("Duplicate test")
	if err == nil {
		t.Error("Expected validation error for duplicate title")
	}

	err = tmpTodoList.validateTitle("DUPLICATE TEST") // case insensitive check
	if err == nil {
		t.Error("Expected validation error for duplicate title (case insensitive)")
	}

	if ve, ok := err.(*TodoValidationError); ok {
		if _, hasField := ve.Errors["title"]; !hasField {
			t.Errorf("Expected 'title' error in TodoValidationError, got errors: %v", ve.Errors)
		}
	}
}

func TestCreateTodoItemValidated(t *testing.T) {
	tmpDir := t.TempDir()
	tmpTodoList := NewTodoList(tmpDir + "/.todo.json")

	// Valid creation
	todo, err := tmpTodoList.CreateTodoItem("Valid title")
	if err != nil {
		t.Fatalf("Unexpected error creating valid todo: %v", err)
	}
	if todo.Title != "Valid title" {
		t.Errorf("Expected title 'Valid title', got '%s'", todo.Title)
	}

	// Invalid - empty
	_, err = tmpTodoList.CreateTodoItem("")
	if err == nil {
		t.Error("Expected error for empty title")
	}

	// Duplicate after first creation
	_, err = tmpTodoList.CreateTodoItem("Valid title")
	if err == nil {
		t.Error("Expected error for duplicate title")
	}
}

// Batch Operations Tests

func TestBatchCreateSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	tmpTodoList := NewTodoList(tmpDir + "/.todo.json")

	titles := []string{"Todo 1", "Todo 2", "Todo 3"}
	result := tmpTodoList.BatchCreate(titles)

	if result.SuccessCount != 3 {
		t.Errorf("Expected 3 successes, got %d", result.SuccessCount)
	}
	if result.FailureCount != 0 {
		t.Errorf("Expected 0 failures, got %d", result.FailureCount)
	}

	if len(result.SuccessItems) != 3 {
		t.Errorf("Expected 3 success items, got %d", len(result.SuccessItems))
	}
}

func TestBatchCreateWithFailures(t *testing.T) {
	tmpDir := t.TempDir()
	tmpTodoList := NewTodoList(tmpDir + "/.todo.json")

	titles := []string{"Valid Todo", "", "Another Valid"} // Empty in middle
	result := tmpTodoList.BatchCreate(titles)

	if result.SuccessCount != 2 {
		t.Errorf("Expected 2 successes, got %d", result.SuccessCount)
	}
	if result.FailureCount != 1 {
		t.Errorf("Expected 1 failure, got %d", result.FailureCount)
	}

	if len(result.Failures) != 1 {
		t.Errorf("Expected 1 failure entry, got %d", len(result.Failures))
	}
}

func TestBatchCreateWithDuplicates(t *testing.T) {
	tmpDir := t.TempDir()
	tmpTodoList := NewTodoList(tmpDir + "/.todo.json")

	titles := []string{"First", "First"} // Duplicate
	result := tmpTodoList.BatchCreate(titles)

	if result.SuccessCount != 1 {
		t.Errorf("Expected 1 success (first occurrence), got %d", result.SuccessCount)
	}
	if result.FailureCount != 1 {
		t.Errorf("Expected 1 failure for duplicate, got %d", result.FailureCount)
	}

	// Verify only one was created
	pending := tmpTodoList.GetPending()
	if len(pending) != 1 {
		t.Errorf("Expected 1 pending todo, got %d", len(pending))
	}
}

func TestBatchDeleteSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	tmpTodoList := NewTodoList(tmpDir + "/.todo.json")

	tmpTodoList.Add("Todo to delete 1")
	tmpTodoList.Add("Todo to delete 2")
	tmpTodoList.Add("Todo to keep")

	result, err := tmpTodoList.BatchDelete([]string{"Todo to delete 1", "Todo to delete 2"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result.SuccessCount != 2 {
		t.Errorf("Expected 2 successes, got %d", result.SuccessCount)
	}
	if result.FailureCount != 0 {
		t.Errorf("Expected 0 failures, got %d", result.FailureCount)
	}

	pending := tmpTodoList.GetPending()
	if len(pending) != 1 {
		t.Errorf("Expected 1 pending todo remaining, got %d", len(pending))
	}
	if pending[0].Title != "Todo to keep" {
		t.Errorf("Expected 'Todo to keep' to remain, got '%s'", pending[0].Title)
	}
}

func TestBatchDeleteWithFailures(t *testing.T) {
	tmpDir := t.TempDir()
	tmpTodoList := NewTodoList(tmpDir + "/.todo.json")

	tmpTodoList.Add("Existing Todo")

	result, err := tmpTodoList.BatchDelete([]string{"Existing Todo", "Non-existent"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result.SuccessCount != 1 {
		t.Errorf("Expected 1 success, got %d", result.SuccessCount)
	}
	if result.FailureCount != 1 {
		t.Errorf("Expected 1 failure, got %d", result.FailureCount)
	}

	if len(result.Failures) != 1 {
		t.Errorf("Expected 1 failure entry, got %d", len(result.Failures))
	}
}

func TestBatchCompleteSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	tmpTodoList := NewTodoList(tmpDir + "/.todo.json")

	tmpTodoList.Add("Todo to complete 1")
	tmpTodoList.Add("Todo to complete 2")

	result, err := tmpTodoList.BatchComplete([]string{"Todo to complete 1", "Todo to complete 2"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result.SuccessCount != 2 {
		t.Errorf("Expected 2 successes, got %d", result.SuccessCount)
	}
	if result.FailureCount != 0 {
		t.Errorf("Expected 0 failures, got %d", result.FailureCount)
	}

	// All should be completed now
	completed := tmpTodoList.GetCompleted()
	pending := tmpTodoList.GetPending()

	if len(completed) != 2 {
		t.Errorf("Expected 2 completed todos, got %d", len(completed))
	}
	if len(pending) != 0 {
		t.Errorf("Expected 0 pending todos, got %d", len(pending))
	}
}

func TestBatchCompleteWithFailures(t *testing.T) {
	tmpDir := t.TempDir()
	tmpTodoList := NewTodoList(tmpDir + "/.todo.json")

	tmpTodoList.Add("Existing Todo")

	result, err := tmpTodoList.BatchComplete([]string{"Existing Todo", "Non-existent", "EXISTING TODO"}) // Duplicate attempt
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result.SuccessCount != 1 {
		t.Errorf("Expected 1 success, got %d", result.SuccessCount)
	}
	if result.FailureCount != 2 {
		t.Errorf("Expected 2 failures (non-existent and already completed), got %d", result.FailureCount)
	}
}

// Case-Insensitive Batch Operations
func TestCaseInsensitiveBatchOperations(t *testing.T) {
	tmpDir := t.TempDir()
	tmpTodoList := NewTodoList(tmpDir + "/.todo.json")

	tmpTodoList.Add("MixedCase Todo")

	// Test case-insensitive delete
	result, err := tmpTodoList.BatchDelete([]string{"MIXEDCASE TODO", "mixedcase todo"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result.SuccessCount != 1 {
		t.Errorf("Expected 1 success (case-insensitive), got %d", result.SuccessCount)
	}

	// Test case-insensitive complete
	tmpTodoList.Add("Another MixedCase")
	result, err = tmpTodoList.BatchComplete([]string{"ANOTHER MIXEDCASE"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result.SuccessCount != 1 {
		t.Errorf("Expected 1 success (case-insensitive), got %d", result.SuccessCount)
	}
}

// Error Type Tests

func TestTodoValidationErrorFormatting(t *testing.T) {
	err := &TodoValidationError{
		Field: "title",
		Title: "Test Title",
		Errors: map[string]string{
			"title": "cannot be empty",
		},
	}

	errorMsg := err.Error()
	if !strings.Contains(errorMsg, "validation failed") {
		t.Errorf("Expected 'validation failed' in error message, got: %s", errorMsg)
	}
	if !strings.Contains(errorMsg, "title") {
		t.Errorf("Expected 'title' in error message, got: %s", errorMsg)
	}
}

func TestTodoValidationErrorUnwrap(t *testing.T) {
	baseErr := &TodoValidationError{Field: "title"}
	wrappedErr := fmt.Errorf("wrapped: %w", baseErr)

	if !errors.As(wrappedErr, new(*TodoValidationError)) {
		t.Error("Expected to unwrap TodoValidationError")
	}
}

func TestTodoNotFoundErrorFormatting(t *testing.T) {
	err := &TodoNotFoundError{
		Title: "Missing Todo",
		Existing: map[string]bool{
			"Existing 1": true,
			"Existing 2": true,
		},
	}

	errorMsg := err.Error()
	if !strings.Contains(errorMsg, "Missing Todo") {
		t.Errorf("Expected missing todo title in error, got: %s", errorMsg)
	}
}
