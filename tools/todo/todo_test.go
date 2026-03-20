package todo

import (
	"os"
	"path/filepath"
	"sync"
	"strings"
	"testing"
	"time"
)

// setupTestEnv creates a temporary directory and todo file for testing
func setupTestEnv(t *testing.T) (*TodoList, func()) {
	t.Helper()
	
	tmpDir := t.TempDir()
	todoFile := filepath.Join(tmpDir, "test_todos.json")
	
	tl := NewTodoList(todoFile)
	
	cleanup := func() {
		// Reset global singleton for next test
		globalOnce = sync.Once{}
		globalTodoList = nil
	}
	
	return tl, cleanup
}

func TestNewTodoList(t *testing.T) {
	tl, _ := setupTestEnv(t)
	
	if tl == nil {
		t.Fatal("Expected non-nil TodoList")
	}
	
	if tl.todos == nil {
		t.Error("Expected todos slice to be initialized")
	}
}

func TestAddTodo(t *testing.T) {
	tl, _ := setupTestEnv(t)
	
	// Add a todo
	added, err := tl.Add("Test todo item")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	
	if added == nil {
		t.Fatal("Expected non-nil todo")
	}
	
	if added.Title != "Test todo item" {
		t.Errorf("Expected title 'Test todo item', got '%s'", added.Title)
	}
	
	if added.Completed {
		t.Error("Expected todo to be incomplete by default")
	}
	
	if added.CreatedAt.IsZero() {
		t.Error("Expected non-zero CreatedAt")
	}
	
	if len(tl.GetAllTodos()) != 1 {
		t.Errorf("Expected 1 todo, got %d", len(tl.GetAllTodos()))
	}
}

func TestAddTodoEmptyTitle(t *testing.T) {
	tl, _ := setupTestEnv(t)
	
	_, err := tl.Add("")
	if err == nil {
		t.Error("Expected error for empty title")
	}
	
	_, err = tl.Add("   ")
	if err == nil {
		t.Error("Expected error for whitespace-only title")
	}
}

func TestAddTodoPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	todoFile := filepath.Join(tmpDir, "test_todos.json")
	
	tl1 := NewTodoList(todoFile)
	tl1.Add("Persistent todo")
	
	// Create new instance to verify persistence
	tl2 := NewTodoList(todoFile)
	todos := tl2.GetAllTodos()
	
	if len(todos) != 1 {
		t.Fatalf("Expected 1 persisted todo, got %d", len(todos))
	}
	
	if todos[0].Title != "Persistent todo" {
		t.Errorf("Expected 'Persistent todo', got '%s'", todos[0].Title)
	}
}

func TestCompleteTodo(t *testing.T) {
	tl, _ := setupTestEnv(t)
	
	// Add a todo first
	tl.Add("Task to complete")
	
	// Complete it
	matched, err := tl.Complete("Task to complete")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	
	if !matched {
		t.Error("Expected todo to be matched and completed")
	}
	
	// Verify completion
	todos := tl.GetAllTodos()
	if len(todos) != 1 || !todos[0].Completed {
		t.Error("Expected todo to be marked as completed")
	}
}

func TestCompleteTodoCaseInsensitive(t *testing.T) {
	tl, _ := setupTestEnv(t)
	
	tl.Add("My Important Task")
	
	// Try with different case
	matched, _ := tl.Complete("my important task")
	if !matched {
		t.Error("Expected case-insensitive match")
	}
}

func TestCompleteTodoPartialMatch(t *testing.T) {
	tl, _ := setupTestEnv(t)
	
	tl.Add("Fix the bug in the parser")
	
	// Try with partial match (actual substring)
	matched, _ := tl.Complete("fix the bug")
	if !matched {
		t.Error("Expected partial match to work")
	}
}

func TestCompleteAlreadyCompleted(t *testing.T) {
	tl, _ := setupTestEnv(t)
	
	tl.Add("Already done")
	tl.Complete("Already done")
	
	// Try to complete again
	_, err := tl.Complete("Already done")
	if err == nil || !strings.Contains(err.Error(), "already completed") {
		t.Errorf("Expected 'already completed' error, got: %v", err)
	}
}

func TestCompleteNonExistent(t *testing.T) {
	tl, _ := setupTestEnv(t)
	
	matched, err := tl.Complete("Non-existent task")
	if matched || err != nil {
		t.Errorf("Expected no match and no error for non-existent todo")
	}
}

func TestDeleteTodo(t *testing.T) {
	tl, _ := setupTestEnv(t)
	
	tl.Add("To be deleted")
	tl.Add("To stay")
	
	deleted, err := tl.Delete("to be deleted")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	
	if !deleted {
		t.Error("Expected todo to be deleted")
	}
	
	todos := tl.GetAllTodos()
	if len(todos) != 1 || todos[0].Title != "To stay" {
		t.Errorf("Expected 1 remaining todo 'To stay', got: %v", todos)
	}
}

func TestDeleteNonExistent(t *testing.T) {
	tl, _ := setupTestEnv(t)
	
	tl.Add("Keep this")
	
	deleted, err := tl.Delete("Non-existent")
	if deleted || err != nil {
		t.Errorf("Expected no deletion and no error for non-existent todo")
	}
	
	todos := tl.GetAllTodos()
	if len(todos) != 1 {
		t.Error("Expected todo to remain")
	}
}

func TestFormatAllTodos(t *testing.T) {
	tl, _ := setupTestEnv(t)
	
	tl.Add("Pending task")
	tl.Add("Completed task")
	tl.Complete("completed task")
	
	output := tl.FormatAllTodos()
	
	if !strings.Contains(output, "=== TODO LIST ===") {
		t.Error("Expected header in output")
	}
	
	if !strings.Contains(output, "Pending task") || !strings.Contains(output, "Completed task") {
		t.Error("Expected todos in output")
	}
	
	if !strings.Contains(output, "1 pending, 1 completed") {
		t.Error("Expected correct summary counts")
	}
}

func TestFormatAllTodosEmpty(t *testing.T) {
	tl, _ := setupTestEnv(t)
	
	output := tl.FormatAllTodos()
	
	if output != "No todos found." {
		t.Errorf("Expected 'No todos found.', got: %q", output)
	}
}

func TestFormatPendingTodos(t *testing.T) {
	tl, _ := setupTestEnv(t)
	
	tl.Add("Pending 1")
	tl.Add("Pending 2")
	tl.Add("Done")
	tl.Complete("done")
	
	output := tl.FormatPendingTodos()
	
	if !strings.Contains(output, "=== PENDING TODOS ===") {
		t.Error("Expected header")
	}
	
	if !strings.Contains(output, "Pending 1") || !strings.Contains(output, "Pending 2") {
		t.Error("Expected pending todos in output")
	}
	
	if strings.Contains(output, "Done") {
		t.Error("Completed todo should not appear")
	}
	
	if !strings.Contains(output, "Total pending: 2") {
		t.Error("Expected correct count")
	}
}

func TestGetStats(t *testing.T) {
	tl, _ := setupTestEnv(t)
	
	tl.Add("Task 1")
	tl.Add("Task 2")
	tl.Add("Task 3")
	tl.Complete("task 1")
	tl.Complete("task 3")
	
	total, pending, completed := tl.GetStats()
	
	if total != 3 || pending != 1 || completed != 2 {
		t.Errorf("Expected total=3, pending=1, completed=2, got total=%d, pending=%d, completed=%d", total, pending, completed)
	}
}

func TestGetStatsEmpty(t *testing.T) {
	tl, _ := setupTestEnv(t)
	
	total, pending, completed := tl.GetStats()
	
	if total != 0 || pending != 0 || completed != 0 {
		t.Error("Expected all zeros for empty list")
	}
}

func TestSortByCreation(t *testing.T) {
	tl, _ := setupTestEnv(t)
	
	// Add todos with artificial time delays
	tl.Add("First")
	time.Sleep(10 * time.Millisecond)
	tl.Add("Second")
	time.Sleep(10 * time.Millisecond)
	tl.Add("Third")
	
	// Shuffle by modifying order directly (bypassing lock for test)
	tl.todos[0], tl.todos[2] = tl.todos[2], tl.todos[0]
	
	tl.SortByCreation()
	
	if tl.todos[0].Title != "First" || tl.todos[1].Title != "Second" || tl.todos[2].Title != "Third" {
		t.Errorf("Expected sorted order, got: %v", tl.GetAllTodos())
	}
}

func TestGetAllTodosReturnsCopy(t *testing.T) {
	tl, _ := setupTestEnv(t)
	
	tl.Add("Original")
	
	todos1 := tl.GetAllTodos()
	todos1[0].Title = "Modified"
	
	todos2 := tl.GetAllTodos()
	if todos2[0].Title != "Original" {
		t.Error("Expected GetAllTodos to return a copy, not reference")
	}
}

func TestAddMultipleTodos(t *testing.T) {
	tl, _ := setupTestEnv(t)
	
	for i := 1; i <= 10; i++ {
		_, err := tl.Add("Todo item ")
		if err != nil {
			t.Fatalf("Failed to add todo %d: %v", i, err)
		}
	}
	
	total, _, _ := tl.GetStats()
	if total != 10 {
		t.Errorf("Expected 10 todos, got %d", total)
	}
}

func TestConcurrency(t *testing.T) {
	tl, _ := setupTestEnv(t)
	
	done := make(chan bool)
	
	// Add todos concurrently
	for i := 0; i < 10; i++ {
		go func(id int) {
			tl.Add("Concurrent task ")
			done <- true
		}(i)
	}
	
	// Wait for all to complete
	for i := 0; i < 10; i++ {
		<-done
	}
	
	total, _, _ := tl.GetStats()
	if total != 10 {
		t.Errorf("Expected 10 concurrent todos, got %d", total)
	}
}

func TestTodoFileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	todoFile := filepath.Join(tmpDir, "nonexistent.json")
	
	// Should not panic when file doesn't exist
	tl := NewTodoList(todoFile)
	if tl == nil || len(tl.GetAllTodos()) != 0 {
		t.Error("Expected empty todo list for non-existent file")
	}
}

func TestInvalidJSONFile(t *testing.T) {
	tmpDir := t.TempDir()
	todoFile := filepath.Join(tmpDir, "invalid.json")
	
	// Write invalid JSON
	os.WriteFile(todoFile, []byte("not valid json {{{"), 0644)
	
	// Should handle gracefully
	tl := NewTodoList(todoFile)
	if tl == nil || len(tl.GetAllTodos()) != 0 {
		t.Error("Expected empty todo list for invalid JSON")
	}
}

func TestGetGlobalTodoList(t *testing.T) {
	// Note: We can't properly test GetGlobalTodoList in isolation because it uses a const for file path
	// and we can't modify that without code changes. This test verifies basic functionality exists.
	
	tl := GetGlobalTodoList()
	if tl == nil {
		t.Fatal("Expected non-nil global TodoList")
	}
	
	// Add something to verify it works
	_, err := tl.Add("Global test todo")
	if err != nil {
		t.Fatalf("Failed to add todo: %v", err)
	}
	
	total, _, _ := tl.GetStats()
	if total < 1 {
		t.Error("Expected at least 1 todo in global list")
	}
}
