// Package tools provides helper functions for tool implementations
package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	
	"yolo/config"
)

// Email-related helpers

func sendEmail(to, subject, body string) error {
	if to == "" {
		to = "scott@stg.net"
	}
	
	cmd := exec.Command("sendmail", "-t")
	input := fmt.Sprintf("To: %s\nFrom: yolo@b-haven.org\nSubject: %s\n\n%s\n", to, subject, body)
	
	cmd.Stdin = strings.NewReader(input)
	return cmd.Run()
}

func sendReport(subject, body string) error {
	if subject == "" {
		subject = "YOLO Progress Report"
	}
	
	to := "scott@stg.net"
	return sendEmail(to, subject, body)
}

func checkInbox(markRead bool) ([]string, error) {
	inboxDir := "/var/mail/b-haven.org/yolo/new/"
	
	entries, err := os.ReadDir(inboxDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read inbox: %w", err)
	}
	
	var emails []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		
		emailPath := filepath.Join(inboxDir, entry.Name())
		content, err := os.ReadFile(emailPath)
		if err != nil {
			continue
		}
		
		emails = append(emails, string(content))
		
		if markRead {
			// Move to cur directory
			curDir := "/var/mail/b-haven.org/yolo/cur/"
			os.MkdirAll(curDir, 0755)
			destPath := filepath.Join(curDir, entry.Name())
			
			data, _ := os.ReadFile(emailPath)
			os.WriteFile(destPath, data, 0644)
			os.Remove(emailPath)
		}
	}
	
	return emails, nil
}

func processInboxWithResponse() (int, int, error) {
	processed := 0
	skipped := 0
	
	emails, err := checkInbox(false)
	if err != nil {
		return 0, 0, err
	}
	
	for _, _ = range emails {
		// For each email, generate LLM response and send it
		// This is a simplified version - full implementation in main package
		processed++
	}
	
	return processed, skipped, nil
}

// GOG helper

func executeGOG(command string) (string, error) {
	// Placeholder for Google API commands
	// In production, this would call actual Google APIs
	return fmt.Sprintf("GOG command executed: %s", command), nil
}

// Learning helpers

type Improvement struct {
	Title      string `json:"title"`
	Priority   string `json:"priority"`
	Category   string `json:"category"`
	Source     string `json:"source"`
	Descraption string `json:"description"`
}

func runLearning() ([]Improvement, error) {
	// Placeholder for learning implementation
	return []Improvement{
		{
			Title:      "Improve error handling in HTTP handlers",
			Priority:   "HIGH",
			Category:   "Code Quality",
			Source:     "Web search",
			Descraption: "Add proper error logging and user-friendly error messages",
		},
	}, nil
}

func implementImprovements(count int) (string, error) {
	// Placeholder for implementation logic
	return fmt.Sprintf("Implementation logic for %d improvements", count), nil
}

// Model helpers

func listOllamaModels() ([]string, error) {
	cmd := exec.Command("ollama", "list")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}
	
	var models []string
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines[1:] { // Skip header
		parts := strings.Fields(line)
		if len(parts) > 0 {
			models = append(models, parts[0])
		}
	}
	
	return models, nil
}

func switchToModel(model string) error {
	// Update current model in config
	// This would call config.SetCurrentModel(model)
	return nil
}

// Todo helpers

type Todo struct {
	Title      string    `json:"title"`
	Created    time.Time `json:"created"`
	Completed  bool      `json:"completed"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

func addTodoItem(title string) error {
	todos, err := loadTodos()
	if err != nil {
		return err
	}
	
	newTodo := Todo{
		Title:     title,
		Created:   time.Now(),
		Completed: false,
	}
	
	todos = append(todos, newTodo)
	return saveTodos(todos)
}

func completeTodoItem(title string) error {
	todos, err := loadTodos()
	if err != nil {
		return err
	}
	
	for i := range todos {
		if todos[i].Title == title && !todos[i].Completed {
			now := time.Now()
			todos[i].Completed = true
			todos[i].CompletedAt = &now
			return saveTodos(todos)
		}
	}
	
	return fmt.Errorf("todo not found: %s", title)
}

func deleteTodoItem(title string) error {
	todos, err := loadTodos()
	if err != nil {
		return err
	}
	
	for i := range todos {
		if todos[i].Title == title {
			todos = append(todos[:i], todos[i+1:]...)
			return saveTodos(todos)
		}
	}
	
	return fmt.Errorf("todo not found: %s", title)
}

func listAllTodos() ([]Todo, error) {
	return loadTodos()
}

func formatTodos(todos []Todo) string {
	var sb strings.Builder
	
	pendingCount := 0
	completedCount := 0
	
	for _, todo := range todos {
		if todo.Completed {
			completedCount++
		} else {
			pendingCount++
		}
	}
	
	sb.WriteString(fmt.Sprintf("Total: %d pending, %d completed\n\n", pendingCount, completedCount))
	
	if pendingCount > 0 {
		sb.WriteString("--- PENDING ---\n")
		for _, todo := range todos {
			if !todo.Completed {
				sb.WriteString(fmt.Sprintf("- [ ] %s (created: %s)\n", 
					todo.Title, todo.Created.Format("Jan 2, 2006 3:04PM")))
			}
		}
		sb.WriteString("\n")
	}
	
	if completedCount > 0 {
		sb.WriteString("--- COMPLETED ---\n")
		for _, todo := range todos {
			if todo.Completed {
				completedStr := "N/A"
				if todo.CompletedAt != nil {
					completedStr = todo.CompletedAt.Format("Jan 2, 2006 3:04PM")
				}
				sb.WriteString(fmt.Sprintf("- [x] %s (completed: %s)\n", 
					todo.Title, completedStr))
			}
		}
	}
	
	return sb.String()
}

func loadTodos() ([]Todo, error) {
	todoPath := filepath.Join(config.WorkingDir(), ".todo.json")
	
	data, err := os.ReadFile(todoPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Todo{}, nil
		}
		return nil, fmt.Errorf("failed to read todos: %w", err)
	}
	
	var todos []Todo
	if err := json.Unmarshal(data, &todos); err != nil {
		return nil, fmt.Errorf("failed to parse todos: %w", err)
	}
	
	return todos, nil
}

func saveTodos(todos []Todo) error {
	todoPath := filepath.Join(config.WorkingDir(), ".todo.json")
	
	data, err := json.MarshalIndent(todos, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize todos: %w", err)
	}
	
	if err := os.WriteFile(todoPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write todos: %w", err)
	}
	
	return nil
}

// Helper function for running send_command with buffering
func runSendmailCommand(to, subject, body string) error {
	cmd := exec.Command("sendmail", "-t")
	
	var stdin bytes.Buffer
	fmt.Fprintf(&stdin, "To: %s\n", to)
	fmt.Fprintf(&stdin, "From: yolo@b-haven.org\n")
	fmt.Fprintf(&stdin, "Subject: %s\n", subject)
	fmt.Fprintln(&stdin)
	fmt.Fprintln(&stdin, body)
	
	cmd.Stdin = &stdin
	return cmd.Run()
}
