// Todo List Tool Functions
// Wrappers for LLM tools that delegate to the todo package.

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"yolo/tools/todo"
)

// addTodoItem adds a new todo item using the todo package.
func addTodoItem(title string) string {
	if strings.TrimSpace(title) == "" {
		return "Error: TODO title cannot be empty"
	}

	todoList := todo.GetGlobalTodoList()
	item, err := todoList.Add(title)
	if err != nil {
		return "Error: " + err.Error()
	}

	return "Added TODO: " + item.Title + "\n   Created: " + item.CreatedAt.Format("2006-01-02 15:04:05")
}

// completeTodoItem marks a todo as completed.
func completeTodoItem(title string) string {
	if strings.TrimSpace(title) == "" {
		return "Error: TODO title cannot be empty"
	}

	todoList := todo.GetGlobalTodoList()
	found, err := todoList.Complete(title)
	if err != nil {
		return "Error: " + err.Error()
	}

	if found {
		return "Marked as completed: " + title
	}

	return "Error: TODO not found or already completed: " + title
}

// deleteTodoItem removes a todo item.
func deleteTodoItem(title string) string {
	if strings.TrimSpace(title) == "" {
		return "Error: TODO title cannot be empty"
	}

	todoList := todo.GetGlobalTodoList()
	found, err := todoList.Delete(title)
	if err != nil {
		return "Error: " + err.Error()
	}

	if found {
		return "Deleted TODO: " + title
	}

	return "Error: TODO not found: " + title
}

// listTodos returns a formatted string of all todos.
func listTodos() string {
	todoList := todo.GetGlobalTodoList()
	return todoList.FormatAllTodos()
}

// getPendingTodos returns a formatted string of pending todos only.
func getPendingTodos() string {
	todoList := todo.GetGlobalTodoList()
	return todoList.FormatPendingTodos()
}

// ──── Tool Executor Methods ────────────────────────────────────

// addTodo handles the "add_todo" tool invocation.
func (te *ToolExecutor) addTodo(args map[string]any) string {
	title, ok := args["title"].(string)
	if !ok || strings.TrimSpace(title) == "" {
		return "Error: title parameter is required and cannot be empty"
	}
	return addTodoItem(title)
}

// completeTodo handles the "complete_todo" tool invocation.
func (te *ToolExecutor) completeTodo(args map[string]any) string {
	title, ok := args["title"].(string)
	if !ok || strings.TrimSpace(title) == "" {
		return "Error: title parameter is required and cannot be empty"
	}
	return completeTodoItem(title)
}

// deleteTodo handles the "delete_todo" tool invocation.
func (te *ToolExecutor) deleteTodo(args map[string]any) string {
	title, ok := args["title"].(string)
	if !ok || strings.TrimSpace(title) == "" {
		return "Error: title parameter is required and cannot be empty"
	}
	return deleteTodoItem(title)
}

// listTodosTool handles the "list_todos" tool invocation.
func (te *ToolExecutor) listTodosTool(args map[string]any) string {
	return listTodos()
}

// checkOllamaStatus checks if Ollama is running and reads recent log entries.
func (te *ToolExecutor) checkOllamaStatus(args map[string]any) string {
	linesToRead := 50
	if lines, ok := args["lines"].(int); ok && lines > 0 {
		linesToRead = lines
	}

	var result strings.Builder

	// Check if ollama is running
	cmd := exec.Command("pgrep", "-f", "ollama serve")
	ollamaRunning := cmd.Run() == nil

	result.WriteString("=== Ollama Status ===\n\n")
	result.WriteString(fmt.Sprintf("Ollama Running: %v\n\n", ollamaRunning))

	if !ollamaRunning {
		return result.String() + "Ollama is not running. YOLO cannot function without Ollama.\n"
	}

	// Try multiple log file locations
	logPaths := []string{
		"./logs/ollama.log",
		"/tmp/ollama.log",
		"./logs/ollama.log.err",
	}

	var logFile string
	var data []byte
	var err error

	for _, path := range logPaths {
		data, err = os.ReadFile(path)
		if err == nil {
			logFile = path
			break
		}
	}

	if err != nil {
		result.WriteString("Note: No Ollama log file found.\n")
		result.WriteString("To enable logging:\n")
		result.WriteString("  1. Set YOLO_OLLAMA_LOG=1 before starting YOLO\n")
		result.WriteString("  2. Or restart ollama manually with: nohup ollama serve > logs/ollama.log 2>&1 &\n")
		return result.String()
	}

	lines := strings.Split(string(data), "\n")
	startIdx := len(lines) - linesToRead
	if startIdx < 0 {
		startIdx = 0
	}

	logLines := lines[startIdx:]

	// Look for error patterns
	var errors []string
	var warnings []string
	for i, line := range logLines {
		if strings.Contains(line, "ERROR") || strings.Contains(line, "error") || strings.Contains(line, "failed") || strings.Contains(line, "panic") {
			errors = append(errors, fmt.Sprintf("Line %d: %s", startIdx+i+1, line))
		} else if strings.Contains(line, "WARN") || strings.Contains(line, "warning") {
			warnings = append(warnings, fmt.Sprintf("Line %d: %s", startIdx+i+1, line))
		}
	}

	result.WriteString(fmt.Sprintf("Log File: %s\n", logFile))
	result.WriteString(fmt.Sprintf("Lines Read: %d (last %d of %d total)\n\n", len(logLines), linesToRead, len(lines)))

	if len(errors) > 0 {
		result.WriteString(fmt.Sprintf("=== ERRORS FOUND (%d) ===\n\n", len(errors)))
		for _, e := range errors[:10] { // Limit to first 10 errors
			result.WriteString(e + "\n")
		}
		if len(errors) > 10 {
			result.WriteString(fmt.Sprintf("\n... and %d more errors\n", len(errors)-10))
		}
		result.WriteString("\n")
	}

	if len(warnings) > 0 {
		result.WriteString(fmt.Sprintf("=== WARNINGS FOUND (%d) ===\n\n", len(warnings)))
		for _, w := range warnings[:10] { // Limit to first 10 warnings
			result.WriteString(w + "\n")
		}
		if len(warnings) > 10 {
			result.WriteString(fmt.Sprintf("\n... and %d more warnings\n", len(warnings)-10))
		}
		result.WriteString("\n")
	}

	result.WriteString("=== RECENT LOG LINES ===\n\n")
	for _, line := range logLines {
		if line != "" {
			result.WriteString(line + "\n")
		}
	}

	return result.String()
}
