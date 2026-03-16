package types

import "time"

// Status represents a status response for HTTP endpoints
type Status struct {
	Request       string    `json:"request"`
	Response      string    `json:"response"`
	Code          int       `json:"code"`
	Uptime        string    `json:"uptime"`
	RequestsTotal int64     `json:"requests_total"`
	ServerTime    time.Time `json:"server_time"`
	Version       string    `json:"version"`
}

// AgentStatus represents the status of the YOLO agent
type AgentStatus struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	Message string `json:"message"`
}

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
	Timestamp      time.Time   `json:"timestamp"`
	CompletedTasks []string    `json:"completed_tasks"`
	PendingTasks   []string    `json:"pending_tasks"`
	IssuesFound    []string    `json:"issues_found,omitempty"`
	Suggestions    []string    `json:"suggestions,omitempty"`
	NextSteps      []string    `json:"next_steps"`
	EmailSummary   interface{} `json:"email_summary,omitempty"`
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

// ModelProvider represents different AI model providers
type ModelProvider string

const (
	ModelProviderOllama    ModelProvider = "ollama"
	ModelProviderOpenAI    ModelProvider = "openai"
	ModelProviderAnthropic ModelProvider = "anthropic"
)
