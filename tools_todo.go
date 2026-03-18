// Todo List Tool Functions
// Wrappers for LLM tools that delegate to the todo package.

package main

import (
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

// ──── Tool Executor Methods ──────────────────────────────────────────

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
