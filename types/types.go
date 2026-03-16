package yolo

import (
	"encoding/json"
	"fmt"
	"time"
)

// PackageInfo holds package metadata from go.mod
type PackageInfo struct {
	Path       string `json:"path"`
	Name       string `json:"name"`
	GoMod      string `json:"go_mod"`
	GoVersion  string `json:"go_version"`
	Main       bool   `json:"main"`
	ImportPath string `json:"import_path"`
}

// ToolOutput represents the result of a tool call
type ToolOutput struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Status      string        `json:"status"` // "success", "error"
	Content     interface{}   `json:"content,omitempty"`
	Error       error         `json:"error,omitempty"`
	Timestamp   time.Time     `json:"timestamp"`
	Duration    time.Duration `json:"duration,omitempty"`
	ToolVersion string        `json:"tool_version,omitempty"`
}

// CommandResult represents the output of a shell command execution
type CommandResult struct {
	Command       string        `json:"command"`
	Output        string        `json:"output"`
	Error         error         `json:"error,omitempty"`
	ExitCode      int           `json:"exit_code"`
	ExecutionTime time.Duration `json:"execution_time"`
}

// SubagentConfig holds configuration for a spawned sub-agent
type SubagentConfig struct {
	Name           string            `json:"name,omitempty"`
	Description    string            `json:"description"`
	Prompt         string            `json:"prompt"`
	TimeoutSeconds int               `json:"timeout_seconds,omitempty"`
	MaxRetries     int               `json:"max_retries,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// ProcessState captures the current processing state of YOLO
type ProcessState struct {
	Timestamp        time.Time              `json:"timestamp"`
	Status           string                 `json:"status"` // "idle", "processing", "completed"
	CurrentTask      string                 `json:"current_task"`
	ProgressPercent  float64                `json:"progress_percent"`
	SubagentsCount   int                    `json:"subagents_count"`
	LastActionTime   time.Time              `json:"last_action_time"`
	ContextVariables map[string]interface{} `json:"context_variables,omitempty"`
}

// ModelProvider represents different AI model providers
type ModelProvider string

const (
	ModelProviderOllama    ModelProvider = "ollama"
	ModelProviderOpenAI    ModelProvider = "openai"
	ModelProviderAnthropic ModelProvider = "anthropic"
)

// Email represents an email message for processing or sending
type Email struct {
	From     string            `json:"from"`
	To       string            `json:"to"`
	Subject  string            `json:"subject"`
	Body     string            `json:"body"`
	HTMLBody string            `json:"html_body,omitempty"`
	InboxDir string            `json:"inbox_dir,omitempty"`
	Received time.Time         `json:"received"`
	Read     bool              `json:"read"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// ProgressReport represents a summary of YOLO's progress for external reporting
type ProgressReport struct {
	Timestamp      time.Time              `json:"timestamp"`
	CompletedTasks []string               `json:"completed_tasks"`
	PendingTasks   []string               `json:"pending_tasks"`
	IssuesFound    []string               `json:"issues_found,omitempty"`
	Suggestions    []string               `json:"suggestions,omitempty"`
	NextSteps      []string               `json:"next_steps"`
	EmailSummary   map[string]interface{} `json:"email_summary,omitempty"`
}

// TodoItem represents a task or todo in YOLO's todo list
type TodoItem struct {
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Status      string    `json:"status"` // "pending", "completed", "deleted"
	CreatedAt   time.Time `json:"created_at"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
	Priority    int       `json:"priority,omitempty"` // 1-5, higher is more important
	Tags        []string  `json:"tags,omitempty"`
}

// TodoList manages the collection of todo items
type TodoList struct {
	Items     []*TodoItem `json:"items"`
	Version   int         `json:"version"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// UnmarshalJSON implements custom unmarshaling for PackageInfo
func (p *PackageInfo) UnmarshalJSON(data []byte) error {
	type Alias PackageInfo
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(p),
	}
	return json.Unmarshal(data, aux)
}

// MarshalJSON implements custom marshaling for ToolOutput to handle interface{} fields
func (t *ToolOutput) MarshalJSON() ([]byte, error) {
	type Alias ToolOutput
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(t),
	}
	return json.Marshal(aux)
}

// String returns a string representation of ProcessState for logging
func (p *ProcessState) String() string {
	return fmt.Sprintf("ProcessState{Status=%s, CurrentTask=%s, Progress=%.1f%%}",
		p.Status, p.CurrentTask, p.ProgressPercent)
}

// AddTodo creates a new todo item with the given title
func (t *TodoList) AddTodo(title string, priority int, tags []string) {
	item := &TodoItem{
		Title:     title,
		Status:    "pending",
		CreatedAt: time.Now(),
		Priority:  priority,
		Tags:      tags,
	}
	t.Items = append(t.Items, item)
	t.Version++
	t.UpdatedAt = time.Now()
}

// CompleteTodo marks a todo as completed by title
func (t *TodoList) CompleteTodo(title string) bool {
	for _, item := range t.Items {
		if item.Title == title && item.Status == "pending" {
			item.Status = "completed"
			item.CompletedAt = time.Now()
			t.Version++
			t.UpdatedAt = time.Now()
			return true
		}
	}
	return false
}

// DeleteTodo removes a todo item entirely
func (t *TodoList) DeleteTodo(title string) bool {
	newItems := make([]*TodoItem, 0, len(t.Items))
	for _, item := range t.Items {
		if item.Title != title {
			newItems = append(newItems, item)
		}
	}
	if len(newItems) < len(t.Items) {
		t.Items = newItems
		t.Version++
		t.UpdatedAt = time.Now()
		return true
	}
	return false
}

// GetPendingTodos returns all pending todo items sorted by priority (descending)
func (t *TodoList) GetPendingTodos() []*TodoItem {
	var pending []*TodoItem
	for _, item := range t.Items {
		if item.Status == "pending" {
			pending = append(pending, item)
		}
	}
	// Sort by priority (highest first)
	for i := 0; i < len(pending); i++ {
		for j := i + 1; j < len(pending); j++ {
			if pending[j].Priority > pending[i].Priority {
				pending[i], pending[j] = pending[j], pending[i]
			}
		}
	}
	return pending
}
