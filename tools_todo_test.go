package main

import (
	stderrors "errors"
	"os"
	"strings"
	"testing"
)

// ──── Helpers ────────────────────────────────────────────────────────

// newTestTodoList creates a TodoList in a temp directory, swaps it into
// the global todoList, and returns a cleanup function.
func newTestTodoList(t *testing.T) (*TodoList, func()) {
	t.Helper()
	orig := todoList
	tl := NewTodoList(t.TempDir() + "/.todo.json")
	todoList = tl
	return tl, func() { todoList = orig }
}

// ──── Basic CRUD ─────────────────────────────────────────────────────

func TestAddTodoItem(t *testing.T) {
	_, cleanup := newTestTodoList(t)
	defer cleanup()

	result := addTodoItem("Test todo")
	if !strings.Contains(result, "Added TODO:") || !strings.Contains(result, "Test todo") {
		t.Errorf("unexpected result: %s", result)
	}
}

func TestAddTodoEmptyTitle(t *testing.T) {
	_, cleanup := newTestTodoList(t)
	defer cleanup()

	result := addTodoItem("")
	if !strings.Contains(result, "Error:") || !strings.Contains(result, "empty") {
		t.Errorf("expected error about empty title, got: %s", result)
	}
}

func TestCompleteTodoItem(t *testing.T) {
	_, cleanup := newTestTodoList(t)
	defer cleanup()

	addTodoItem("Test todo to complete")
	result := completeTodoItem("Test todo to complete")
	if !strings.Contains(result, "Marked as completed:") {
		t.Errorf("expected success, got: %s", result)
	}
}

func TestCompleteTodoNotFound(t *testing.T) {
	_, cleanup := newTestTodoList(t)
	defer cleanup()

	result := completeTodoItem("Non-existent todo")
	if !strings.Contains(result, "Error:") || !strings.Contains(result, "not found") {
		t.Errorf("expected not found error, got: %s", result)
	}
}

func TestDeleteTodoItem(t *testing.T) {
	_, cleanup := newTestTodoList(t)
	defer cleanup()

	addTodoItem("Delete me")
	result := deleteTodoItem("Delete me")
	if !strings.Contains(result, "Deleted TODO:") {
		t.Errorf("expected deletion success, got: %s", result)
	}

	// Verify it's gone
	list := listTodos()
	if strings.Contains(list, "Delete me") {
		t.Error("deleted todo still appears in list")
	}
}

func TestDeleteTodoNotFound(t *testing.T) {
	_, cleanup := newTestTodoList(t)
	defer cleanup()

	result := deleteTodoItem("Ghost")
	if !strings.Contains(result, "Error:") || !strings.Contains(result, "not found") {
		t.Errorf("expected not found error, got: %s", result)
	}
}

func TestListTodos(t *testing.T) {
	_, cleanup := newTestTodoList(t)
	defer cleanup()

	addTodoItem("First todo")
	addTodoItem("Second todo")
	result := listTodos()

	if !strings.Contains(result, "TODO LIST") {
		t.Errorf("expected header, got: %s", result)
	}
	if !strings.Contains(result, "First todo") || !strings.Contains(result, "Second todo") {
		t.Errorf("expected both todos, got: %s", result)
	}
}

func TestListTodosEmpty(t *testing.T) {
	_, cleanup := newTestTodoList(t)
	defer cleanup()

	result := listTodos()
	if !strings.Contains(result, "No todos yet") {
		t.Errorf("expected empty message, got: %s", result)
	}
}

// ──── Persistence ────────────────────────────────────────────────────

func TestTodoFilePersistence(t *testing.T) {
	tl, cleanup := newTestTodoList(t)
	defer cleanup()

	addTodoItem("Persistent todo")

	data, err := os.ReadFile(tl.filePath)
	if err != nil {
		t.Fatalf("expected todo file to exist: %v", err)
	}
	if !strings.Contains(string(data), "Persistent todo") {
		t.Error("todo not found in persisted file")
	}
}

func TestTodoListLoad(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/.todo.json"

	err := os.WriteFile(path, []byte(`[{"title":"Pre-existing","created_at":"2024-01-01T00:00:00Z"}]`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	tl := NewTodoList(path)
	if err := tl.Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	pending := tl.GetPending()
	if len(pending) != 1 || pending[0].Title != "Pre-existing" {
		t.Errorf("unexpected pending: %v", pending)
	}
}

func TestTodoListLoadCorrupt(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/.todo.json"
	os.WriteFile(path, []byte(`{not json`), 0644)

	tl := NewTodoList(path)
	if err := tl.Load(); err != nil {
		t.Fatalf("Load() should not error on corrupt file: %v", err)
	}
	if len(tl.GetAll()) != 0 {
		t.Error("expected empty list after corrupt file load")
	}
}

func TestTodoListLoadMissing(t *testing.T) {
	tl := NewTodoList(t.TempDir() + "/nonexistent.json")
	if err := tl.Load(); err != nil {
		t.Fatalf("Load() should not error on missing file: %v", err)
	}
}

func TestTodoListSave(t *testing.T) {
	dir := t.TempDir()
	tl := NewTodoList(dir + "/.todo.json")

	tl.Add("Test save")
	if err := tl.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	data, err := os.ReadFile(dir + "/.todo.json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Test save") {
		t.Error("saved file missing expected content")
	}
}

// ──── Query Operations ───────────────────────────────────────────────

func TestTodoListGetAll(t *testing.T) {
	tl := NewTodoList(t.TempDir() + "/.todo.json")
	tl.Add("First")
	tl.Add("Second")

	all := tl.GetAll()
	if len(all) != 2 {
		t.Errorf("expected 2, got %d", len(all))
	}
}

func TestTodoListGetCompleted(t *testing.T) {
	tl := NewTodoList(t.TempDir() + "/.todo.json")
	tl.Add("Pending")
	tl.Add("Done")
	tl.Complete("Done")

	completed := tl.GetCompleted()
	if len(completed) != 1 || completed[0].Title != "Done" {
		t.Errorf("unexpected completed: %v", completed)
	}
}

func TestTodoListCount(t *testing.T) {
	tl := NewTodoList(t.TempDir() + "/.todo.json")

	if tl.Count() != 0 {
		t.Errorf("expected 0, got %d", tl.Count())
	}

	tl.Add("One")
	tl.Add("Two")
	if tl.Count() != 2 {
		t.Errorf("expected 2, got %d", tl.Count())
	}

	tl.Complete("One")
	if tl.Count() != 1 {
		t.Errorf("expected 1 after completion, got %d", tl.Count())
	}
}

// ──── FindTodo ───────────────────────────────────────────────────────

func TestFindTodo(t *testing.T) {
	tl := NewTodoList(t.TempDir() + "/.todo.json")
	tl.Add("Find Me")

	found, item := tl.FindTodo("Find Me")
	if !found || item.Title != "Find Me" {
		t.Error("expected to find todo")
	}

	found, _ = tl.FindTodo("Not Here")
	if found {
		t.Error("should not find non-existent todo")
	}
}

func TestFindTodoCaseInsensitive(t *testing.T) {
	tl := NewTodoList(t.TempDir() + "/.todo.json")
	tl.Add("Test Todo")

	found, _ := tl.FindTodo("TEST TODO")
	if !found {
		t.Error("expected case-insensitive find to work")
	}

	found, _ = tl.FindTodo("test todo")
	if !found {
		t.Error("expected case-insensitive find to work (lowercase)")
	}
}

// ──── Delete ─────────────────────────────────────────────────────────

func TestDelete(t *testing.T) {
	tl := NewTodoList(t.TempDir() + "/.todo.json")
	tl.Add("To Delete")
	tl.Add("To Keep")

	if !tl.Delete("To Delete") {
		t.Error("Delete should return true")
	}
	if tl.Count() != 1 {
		t.Errorf("expected 1 remaining, got %d", tl.Count())
	}

	found, _ := tl.FindTodo("To Delete")
	if found {
		t.Error("deleted todo should not be findable")
	}
}

func TestDeleteCaseInsensitive(t *testing.T) {
	tl := NewTodoList(t.TempDir() + "/.todo.json")
	tl.Add("MixedCase Todo")

	if !tl.Delete("MIXEDCASE TODO") {
		t.Error("case-insensitive delete should work")
	}
}

func TestDeleteNotFound(t *testing.T) {
	tl := NewTodoList(t.TempDir() + "/.todo.json")
	if tl.Delete("Ghost") {
		t.Error("Delete of non-existent should return false")
	}
}

// ──── Validation ─────────────────────────────────────────────────────

func TestValidateTitleEmpty(t *testing.T) {
	tl := NewTodoList(t.TempDir() + "/.todo.json")
	err := tl.validateTitle("")
	if err == nil {
		t.Fatal("expected error for empty title")
	}
	var ve *TodoValidationError
	if !stderrors.As(err, &ve) {
		t.Fatalf("expected TodoValidationError, got %T", err)
	}
}

func TestValidateTitleWhitespaceOnly(t *testing.T) {
	tl := NewTodoList(t.TempDir() + "/.todo.json")
	if err := tl.validateTitle("   "); err == nil {
		t.Error("expected error for whitespace-only title")
	}
}

func TestValidateTitleTooLong(t *testing.T) {
	tl := NewTodoList(t.TempDir() + "/.todo.json")
	long := strings.Repeat("a", maxTitleLength+1)
	err := tl.validateTitle(long)
	if err == nil {
		t.Fatal("expected error for long title")
	}
	var ve *TodoValidationError
	if !stderrors.As(err, &ve) {
		t.Fatalf("expected TodoValidationError, got %T", err)
	}
	if _, ok := ve.Errors["title"]; !ok {
		t.Errorf("expected 'title' error, got: %v", ve.Errors)
	}
}

func TestValidateTitleDuplicate(t *testing.T) {
	tl := NewTodoList(t.TempDir() + "/.todo.json")
	tl.Add("Duplicate test")

	err := tl.validateTitle("Duplicate test")
	if err == nil {
		t.Error("expected error for duplicate title")
	}

	// Case-insensitive duplicate
	err = tl.validateTitle("DUPLICATE TEST")
	if err == nil {
		t.Error("expected error for case-insensitive duplicate")
	}
}

func TestValidateTitleAllowsCompletedDuplicate(t *testing.T) {
	tl := NewTodoList(t.TempDir() + "/.todo.json")
	tl.Add("Done task")
	tl.Complete("Done task")

	// Should allow re-adding a completed todo
	if err := tl.validateTitle("Done task"); err != nil {
		t.Errorf("should allow re-adding completed todo, got: %v", err)
	}
}

// ──── CreateTodoItem (validated Add) ─────────────────────────────────

func TestCreateTodoItem(t *testing.T) {
	tl := NewTodoList(t.TempDir() + "/.todo.json")

	item, err := tl.CreateTodoItem("Valid title")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Title != "Valid title" {
		t.Errorf("expected 'Valid title', got %q", item.Title)
	}

	// Empty should fail
	_, err = tl.CreateTodoItem("")
	if err == nil {
		t.Error("expected error for empty title")
	}

	// Duplicate should fail
	_, err = tl.CreateTodoItem("Valid title")
	if err == nil {
		t.Error("expected error for duplicate title")
	}
}

// ──── Batch Operations ───────────────────────────────────────────────

func TestBatchCreateSuccess(t *testing.T) {
	tl := NewTodoList(t.TempDir() + "/.todo.json")
	result := tl.BatchCreate([]string{"A", "B", "C"})

	if result.SuccessCount != 3 {
		t.Errorf("expected 3 successes, got %d", result.SuccessCount)
	}
	if result.FailureCount != 0 {
		t.Errorf("expected 0 failures, got %d", result.FailureCount)
	}
	if len(result.SuccessItems) != 3 {
		t.Errorf("expected 3 success items, got %d", len(result.SuccessItems))
	}
}

func TestBatchCreateWithFailures(t *testing.T) {
	tl := NewTodoList(t.TempDir() + "/.todo.json")
	result := tl.BatchCreate([]string{"Valid", "", "Also Valid"})

	if result.SuccessCount != 2 {
		t.Errorf("expected 2 successes, got %d", result.SuccessCount)
	}
	if result.FailureCount != 1 {
		t.Errorf("expected 1 failure, got %d", result.FailureCount)
	}
}

func TestBatchCreateWithDuplicates(t *testing.T) {
	tl := NewTodoList(t.TempDir() + "/.todo.json")
	result := tl.BatchCreate([]string{"First", "First"})

	if result.SuccessCount != 1 {
		t.Errorf("expected 1 success, got %d", result.SuccessCount)
	}
	if result.FailureCount != 1 {
		t.Errorf("expected 1 failure for duplicate, got %d", result.FailureCount)
	}
	if tl.Count() != 1 {
		t.Errorf("expected 1 pending, got %d", tl.Count())
	}
}

func TestBatchDeleteSuccess(t *testing.T) {
	tl := NewTodoList(t.TempDir() + "/.todo.json")
	tl.Add("Del 1")
	tl.Add("Del 2")
	tl.Add("Keep")

	result, err := tl.BatchDelete([]string{"Del 1", "Del 2"})
	if err != nil {
		t.Fatal(err)
	}
	if result.SuccessCount != 2 || result.FailureCount != 0 {
		t.Errorf("expected 2/0, got %d/%d", result.SuccessCount, result.FailureCount)
	}

	pending := tl.GetPending()
	if len(pending) != 1 || pending[0].Title != "Keep" {
		t.Errorf("unexpected pending: %v", pending)
	}
}

func TestBatchDeleteWithFailures(t *testing.T) {
	tl := NewTodoList(t.TempDir() + "/.todo.json")
	tl.Add("Exists")

	result, _ := tl.BatchDelete([]string{"Exists", "Ghost"})
	if result.SuccessCount != 1 || result.FailureCount != 1 {
		t.Errorf("expected 1/1, got %d/%d", result.SuccessCount, result.FailureCount)
	}
}

func TestBatchCompleteSuccess(t *testing.T) {
	tl := NewTodoList(t.TempDir() + "/.todo.json")
	tl.Add("C1")
	tl.Add("C2")

	result, err := tl.BatchComplete([]string{"C1", "C2"})
	if err != nil {
		t.Fatal(err)
	}
	if result.SuccessCount != 2 || result.FailureCount != 0 {
		t.Errorf("expected 2/0, got %d/%d", result.SuccessCount, result.FailureCount)
	}
	if len(tl.GetCompleted()) != 2 || len(tl.GetPending()) != 0 {
		t.Error("unexpected state after batch complete")
	}
}

func TestBatchCompleteWithFailures(t *testing.T) {
	tl := NewTodoList(t.TempDir() + "/.todo.json")
	tl.Add("Exists")

	// "EXISTS" will succeed (completing "Exists"), then fail on re-complete
	result, _ := tl.BatchComplete([]string{"Exists", "Ghost", "EXISTS"})
	if result.SuccessCount != 1 {
		t.Errorf("expected 1 success, got %d", result.SuccessCount)
	}
	if result.FailureCount != 2 {
		t.Errorf("expected 2 failures, got %d", result.FailureCount)
	}
}

func TestCaseInsensitiveBatchOperations(t *testing.T) {
	tl := NewTodoList(t.TempDir() + "/.todo.json")
	tl.Add("MixedCase Todo")

	result, _ := tl.BatchDelete([]string{"MIXEDCASE TODO", "mixedcase todo"})
	if result.SuccessCount != 1 {
		t.Errorf("expected 1 success (case-insensitive delete), got %d", result.SuccessCount)
	}

	tl.Add("Another MixedCase")
	result, _ = tl.BatchComplete([]string{"ANOTHER MIXEDCASE"})
	if result.SuccessCount != 1 {
		t.Errorf("expected 1 success (case-insensitive complete), got %d", result.SuccessCount)
	}
}

// ──── MarshalJSON ────────────────────────────────────────────────────

func TestTodoItemMarshalJSON(t *testing.T) {
	pending := TodoItem{Title: "Pending"}
	data, err := pending.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), `"completed_at"`) {
		// completed_at should be zero/omitted for pending items
		s := string(data)
		if !strings.Contains(s, `"0001-01-01`) {
			t.Errorf("unexpected completed_at in pending item: %s", s)
		}
	}
}

// ──── Error Types ────────────────────────────────────────────────────

func TestTodoValidationErrorFormatting(t *testing.T) {
	err := &TodoValidationError{
		Field: "title",
		Title: "Test",
		Errors: map[string]string{
			"title": "cannot be empty",
		},
	}
	msg := err.Error()
	if !strings.Contains(msg, "validation failed") || !strings.Contains(msg, "title") {
		t.Errorf("unexpected error message: %s", msg)
	}
}

func TestTodoValidationErrorUnwrap(t *testing.T) {
	baseErr := &TodoValidationError{Field: "title"}
	wrapped := stderrors.Join(stderrors.New("outer"), baseErr)
	var ve *TodoValidationError
	if !stderrors.As(wrapped, &ve) {
		t.Error("expected to unwrap TodoValidationError")
	}
}

func TestTodoNotFoundErrorFormatting(t *testing.T) {
	err := &TodoNotFoundError{
		Title:    "Missing",
		Existing: map[string]bool{"A": true, "B": true},
	}
	msg := err.Error()
	if !strings.Contains(msg, "Missing") {
		t.Errorf("expected title in error, got: %s", msg)
	}
}

// ──── RenderPendingContext ───────────────────────────────────────────

func TestRenderPendingContextEmpty(t *testing.T) {
	tl := NewTodoList(t.TempDir() + "/.todo.json")
	if ctx := tl.RenderPendingContext(); ctx != "" {
		t.Errorf("expected empty string for no pending, got: %q", ctx)
	}
}

func TestRenderPendingContextWithItems(t *testing.T) {
	tl := NewTodoList(t.TempDir() + "/.todo.json")
	tl.Add("Fix bug")
	tl.Add("Add feature")

	ctx := tl.RenderPendingContext()
	if !strings.Contains(ctx, "Pending Todos (2)") {
		t.Errorf("expected header with count, got: %q", ctx)
	}
	if !strings.Contains(ctx, "- Fix bug") || !strings.Contains(ctx, "- Add feature") {
		t.Errorf("expected todo items in context, got: %q", ctx)
	}
}

func TestRenderPendingContextExcludesCompleted(t *testing.T) {
	tl := NewTodoList(t.TempDir() + "/.todo.json")
	tl.Add("Pending")
	tl.Add("Done")
	tl.Complete("Done")

	ctx := tl.RenderPendingContext()
	if !strings.Contains(ctx, "Pending Todos (1)") {
		t.Errorf("expected count 1, got: %q", ctx)
	}
	if strings.Contains(ctx, "Done") {
		t.Error("completed todo should not appear in pending context")
	}
}

// ──── Tool Executor Methods ──────────────────────────────────────────

func TestToolExecutorAddTodo(t *testing.T) {
	_, cleanup := newTestTodoList(t)
	defer cleanup()

	te := &ToolExecutor{}
	result := te.addTodo(map[string]any{"title": "From executor"})
	if !strings.Contains(result, "Added TODO:") {
		t.Errorf("unexpected: %s", result)
	}

	// Missing title
	result = te.addTodo(map[string]any{})
	if !strings.Contains(result, "Error:") {
		t.Errorf("expected error for missing title: %s", result)
	}
}

func TestToolExecutorCompleteTodo(t *testing.T) {
	_, cleanup := newTestTodoList(t)
	defer cleanup()

	addTodoItem("Complete me")

	te := &ToolExecutor{}
	result := te.completeTodo(map[string]any{"title": "Complete me"})
	if !strings.Contains(result, "Marked as completed:") {
		t.Errorf("unexpected: %s", result)
	}
}

func TestToolExecutorDeleteTodo(t *testing.T) {
	_, cleanup := newTestTodoList(t)
	defer cleanup()

	addTodoItem("Delete me")

	te := &ToolExecutor{}
	result := te.deleteTodo(map[string]any{"title": "Delete me"})
	if !strings.Contains(result, "Deleted TODO:") {
		t.Errorf("unexpected: %s", result)
	}
}

func TestToolExecutorListTodos(t *testing.T) {
	_, cleanup := newTestTodoList(t)
	defer cleanup()

	addTodoItem("Item 1")
	te := &ToolExecutor{}
	result := te.listTodosTool(map[string]any{})
	if !strings.Contains(result, "TODO LIST") || !strings.Contains(result, "Item 1") {
		t.Errorf("unexpected: %s", result)
	}
}
