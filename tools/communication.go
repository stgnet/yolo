// Package tools provides email, GOG (Google), and learning tool capabilities
package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// SendEmailTool implements email sending functionality
type SendEmailTool struct{}

func (t *SendEmailTool) Name() string { return "send_email" }
func (t *SendEmailTool) Description() string { return "Send an email via sendmail from yolo@b-haven.org" }
func (t *SendEmailTool) Type() ToolType { return ToolTypeCommunication }

func (t *SendEmailTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	start := time.Now()
	
	subject, ok := args["subject"].(string)
	if !ok || subject == "" {
		return &ToolResult{Success: false, Error: "subject is required", Duration: time.Since(start)}, nil
	}
	
	body, ok := args["body"].(string)
	if !ok || body == "" {
		return &ToolResult{Success: false, Error: "body is required", Duration: time.Since(start)}, nil
	}
	
	to, _ := args["to"].(string)
	if to == "" {
		to = "scott@stg.net"
	}
	
	err := sendEmail(to, subject, body)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error(), Duration: time.Since(start)}, nil
	}
	
	return &ToolResult{
		Success:  true,
		Output:   fmt.Sprintf("Email sent to %s with subject: %s", to, subject),
		Duration: time.Since(start),
	}, nil
}

// SendReportTool implements progress report sending
type SendReportTool struct{}

func (t *SendReportTool) Name() string { return "send_report" }
func (t *SendReportTool) Description() string { return "Send a progress report email to scott@stg.net" }
func (t *SendReportTool) Type() ToolType { return ToolTypeCommunication }

func (t *SendReportTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	start := time.Now()
	
	body, ok := args["body"].(string)
	if !ok || body == "" {
		return &ToolResult{Success: false, Error: "body is required", Duration: time.Since(start)}, nil
	}
	
	subject, _ := args["subject"].(string)
	if subject == "" {
		subject = "YOLO Progress Report"
	}
	
	err := sendReport(subject, body)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error(), Duration: time.Since(start)}, nil
	}
	
	return &ToolResult{
		Success:  true,
		Output:   "Progress report sent to scott@stg.net",
		Duration: time.Since(start),
	}, nil
}

// CheckInboxTool implements email inbox checking
type CheckInboxTool struct{}

func (t *CheckInboxTool) Name() string { return "check_inbox" }
func (t *CheckInboxTool) Description() string { return "Read emails from Maildir inbox at /var/mail/b-haven.org/yolo/new/" }
func (t *CheckInboxTool) Type() ToolType { return ToolTypeCommunication }

func (t *CheckInboxTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	start := time.Now()
	
	markRead, _ := args["mark_read"].(bool)
	
	emails, err := checkInbox(markRead)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error(), Duration: time.Since(start)}, nil
	}
	
	output := fmt.Sprintf("Found %d email(s):\n\n", len(emails))
	for i, email := range emails {
		output += fmt.Sprintf("--- Email %d ---\n%s\n\n", i+1, email)
	}
	
	return &ToolResult{
		Success:  true,
		Output:   output,
		Metadata: map[string]interface{}{"count": len(emails)},
		Duration: time.Since(start),
	}, nil
}

// ProcessInboxTool implements complete email handling workflow
type ProcessInboxTool struct{}

func (t *ProcessInboxTool) Name() string { return "process_inbox_with_response" }
func (t *ProcessInboxTool) Description() string { return "Process all inbound emails: read, auto-respond via LLM, delete original" }
func (t *ProcessInboxTool) Type() ToolType { return ToolTypeCommunication }

func (t *ProcessInboxTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	start := time.Now()
	
	processed, skipped, err := processInboxWithResponse()
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error(), Duration: time.Since(start)}, nil
	}
	
	output := fmt.Sprintf("Processed %d email(s), skipped %d\n", processed, skipped)
	
	return &ToolResult{
		Success:  true,
		Output:   output,
		Metadata: map[string]interface{}{"processed": processed, "skipped": skipped},
		Duration: time.Since(start),
	}, nil
}

// GOGTool implements Google API interactions (Gmail, Calendar, Drive, etc.)
type GOGTool struct{}

func (t *GOGTool) Name() string { return "gog" }
func (t *GOGTool) Description() string { return "Google CLI tool for Gmail, Calendar, Drive, Docs, Sheets, Slides, Contacts, Tasks, People, Chat, Classroom" }
func (t *GOGTool) Type() ToolType { return ToolTypeCommunication }

func (t *GOGTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	start := time.Now()
	
	command, ok := args["command"].(string)
	if !ok || command == "" {
		return &ToolResult{Success: false, Error: "command is required", Duration: time.Since(start)}, nil
	}
	
	output, err := executeGOG(command)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error(), Duration: time.Since(start)}, nil
	}
	
	return &ToolResult{
		Success:  true,
		Output:   output,
		Duration: time.Since(start),
	}, nil
}

// LearnTool implements autonomous research and self-improvement
type LearnTool struct{}

func (t *LearnTool) Name() string { return "learn" }
func (t *LearnTool) Description() string { return "Autonomously research and discover self-improvement opportunities from the internet" }
func (t *LearnTool) Type() ToolType { return ToolTypeAI }

func (t *LearnTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	start := time.Now()
	
	improvements, err := runLearning()
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error(), Duration: time.Since(start)}, nil
	}
	
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Discovered %d improvements:\n\n", len(improvements)))
	for i, improvement := range improvements {
		sb.WriteString(fmt.Sprintf("%d. %s\nPriority: %s\nCategory: %s\nSource: %s\n---\n\n",
			i+1, improvement.Title, improvement.Priority, improvement.Category, improvement.Source))
	}
	
	return &ToolResult{
		Success:  true,
		Output:   sb.String(),
		Metadata: map[string]interface{}{"count": len(improvements)},
		Duration: time.Since(start),
	}, nil
}

// ImplementTool automatically implements improvements discovered by the learning system
type ImplementTool struct{}

func (t *ImplementTool) Name() string { return "implement" }
func (t *ImplementTool) Description() string { return "Automatically implement improvements discovered by the learning system" }
func (t *ImplementTool) Type() ToolType { return ToolTypeAI }

func (t *ImplementTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	start := time.Now()
	
	count, _ := args["count"].(int)
	if count <= 0 {
		count = 2
	}
	
	result, err := implementImprovements(count)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error(), Duration: time.Since(start)}, nil
	}
	
	return &ToolResult{
		Success:  true,
		Output:   result,
		Duration: time.Since(start),
	}, nil
}

// ListModelsTool shows available Ollama models
type ListModelsTool struct{}

func (t *ListModelsTool) Name() string { return "list_models" }
func (t *ListModelsTool) Description() string { return "List available Ollama models" }
func (t *ListModelsTool) Type() ToolType { return ToolTypeAI }

func (t *ListModelsTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	start := time.Now()
	
	models, err := listOllamaModels()
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error(), Duration: time.Since(start)}, nil
	}
	
	output := fmt.Sprintf("Available models:\n")
	for _, model := range models {
		output += fmt.Sprintf("- %s\n", model)
	}
	
	return &ToolResult{
		Success:  true,
		Output:   output,
		Metadata: map[string]interface{}{"count": len(models)},
		Duration: time.Since(start),
	}, nil
}

// SwitchModelTool switches to a different Ollama model
type SwitchModelTool struct{}

func (t *SwitchModelTool) Name() string { return "switch_model" }
func (t *SwitchModelTool) Description() string { return "Switch to a different Ollama model" }
func (t *SwitchModelTool) Type() ToolType { return ToolTypeAI }

func (t *SwitchModelTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	start := time.Now()
	
	model, ok := args["model"].(string)
	if !ok || model == "" {
		return &ToolResult{Success: false, Error: "model is required", Duration: time.Since(start)}, nil
	}
	
	err := switchToModel(model)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error(), Duration: time.Since(start)}, nil
	}
	
	return &ToolResult{
		Success:  true,
		Output:   fmt.Sprintf("Switched to model: %s", model),
		Duration: time.Since(start),
	}, nil
}

// Todo management tools

type AddTodoTool struct{}

func (t *AddTodoTool) Name() string { return "add_todo" }
func (t *AddTodoTool) Description() string { return "Add a new item to the todo list" }
func (t *AddTodoTool) Type() ToolType { return ToolTypeSystem }

func (t *AddTodoTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	start := time.Now()
	
	title, ok := args["title"].(string)
	if !ok || title == "" {
		return &ToolResult{Success: false, Error: "title is required", Duration: time.Since(start)}, nil
	}
	
	err := addTodoItem(title)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error(), Duration: time.Since(start)}, nil
	}
	
	return &ToolResult{
		Success:  true,
		Output:   fmt.Sprintf("Added todo: %s", title),
		Duration: time.Since(start),
	}, nil
}

type CompleteTodoTool struct{}

func (t *CompleteTodoTool) Name() string { return "complete_todo" }
func (t *CompleteTodoTool) Description() string { return "Mark a todo item as completed by title" }
func (t *CompleteTodoTool) Type() ToolType { return ToolTypeSystem }

func (t *CompleteTodoTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	start := time.Now()
	
	title, ok := args["title"].(string)
	if !ok || title == "" {
		return &ToolResult{Success: false, Error: "title is required", Duration: time.Since(start)}, nil
	}
	
	err := completeTodoItem(title)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error(), Duration: time.Since(start)}, nil
	}
	
	return &ToolResult{
		Success:  true,
		Output:   fmt.Sprintf("Completed todo: %s", title),
		Duration: time.Since(start),
	}, nil
}

type DeleteTodoTool struct{}

func (t *DeleteTodoTool) Name() string { return "delete_todo" }
func (t *DeleteTodoTool) Description() string { return "Delete a todo item by title (removes it entirely)" }
func (t *DeleteTodoTool) Type() ToolType { return ToolTypeSystem }

func (t *DeleteTodoTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	start := time.Now()
	
	title, ok := args["title"].(string)
	if !ok || title == "" {
		return &ToolResult{Success: false, Error: "title is required", Duration: time.Since(start)}, nil
	}
	
	err := deleteTodoItem(title)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error(), Duration: time.Since(start)}, nil
	}
	
	return &ToolResult{
		Success:  true,
		Output:   fmt.Sprintf("Deleted todo: %s", title),
		Duration: time.Since(start),
	}, nil
}

type ListTodosTool struct{}

func (t *ListTodosTool) Name() string { return "list_todos" }
func (t *ListTodosTool) Description() string { return "List all todos (pending and completed) from .todo.json file" }
func (t *ListTodosTool) Type() ToolType { return ToolTypeSystem }

func (t *ListTodosTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	start := time.Now()
	
	todos, err := listAllTodos()
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error(), Duration: time.Since(start)}, nil
	}
	
	output := formatTodos(todos)
	
	return &ToolResult{
		Success:  true,
		Output:   output,
		Metadata: map[string]interface{}{"count": len(todos)},
		Duration: time.Since(start),
	}, nil
}

// CheckOllamaStatusTool checks Ollama server status and reads debug logs
type CheckOllamaStatusTool struct{}

func (t *CheckOllamaStatusTool) Name() string { return "check_ollama_status" }
func (t *CheckOllamaStatusTool) Description() string { return "Check Ollama server status and read debug logs. Returns whether Ollama is running, recent log lines, and any errors found." }
func (t *CheckOllamaStatusTool) Type() ToolType { return ToolTypeSystem }

func (t *CheckOllamaStatusTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	start := time.Now()
	
	lines, _ := args["lines"].(int)
	if lines <= 0 {
		lines = 50
	}
	
	running, logLines, errorsFound, err := checkOllamaStatus(lines)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error(), Duration: time.Since(start)}, nil
	}
	
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Ollama Status:\n"))
	sb.WriteString(fmt.Sprintf("- Running: %v\n", running))
	sb.WriteString(fmt.Sprintf("- Errors in logs: %d\n", len(errorsFound)))
	if len(errorsFound) > 0 {
		sb.WriteString("\nRecent errors:\n")
		for i, err := range errorsFound[:min(len(errorsFound), 5)] {
			sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, err))
		}
	}
	if logLines != "" {
		sb.WriteString("\nRecent log lines:\n")
		sb.WriteString(logLines)
	}
	
	return &ToolResult{
		Success:  true,
		Output:   sb.String(),
		Metadata: map[string]interface{}{"running": running, "error_count": len(errorsFound)},
		Duration: time.Since(start),
	}, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// checkOllamaStatus checks if Ollama is running and reads recent log lines
func checkOllamaStatus(lines int) (bool, string, []string, error) {
	// Check if ollama is running
	cmd := exec.Command("pgrep", "-f", "ollama serve")
	running := cmd.Run() == nil
	
	var logLines string
	var errorsFound []string
	
	// Try to read the log file
	logFile := "./logs/ollama.log"
	if data, err := os.ReadFile(logFile); err == nil {
		content := string(data)
		allLines := strings.Split(content, "\n")
		
		// Get last N lines
		startIdx := len(allLines) - lines
		if startIdx < 0 {
			startIdx = 0
		}
		recentLines := allLines[startIdx:]
		logLines = strings.Join(recentLines, "\n")
		
		// Find error lines
		for _, line := range recentLines {
			lowerLine := strings.ToLower(line)
			if strings.Contains(lowerLine, "error") || 
			   strings.Contains(lowerLine, "fail") || 
			   strings.Contains(lowerLine, "panic") ||
			   strings.Contains(lowerLine, "exception") {
				errorsFound = append(errorsFound, line)
			}
		}
	} else {
		logLines = "Log file not found at " + logFile
	}
	
	return running, logLines, errorsFound, nil
}
