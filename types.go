package main

// AgentConfig holds the configuration for the YOLO agent
type AgentConfig struct {
	Model      string
	WorkingDir string
	TempDir    string
}

// Message represents a chat message in the conversation
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Response represents an AI response from the model
type Response struct {
	ID      string   `json:"id"`
	Model   string   `json:"model"`
	Created int64    `json:"created"`
	Choices []Choice `json:"choices"`
}

// Choice represents a single choice in a response
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	Logprobs     *string `json:"logprobs"`
	FinishReason string  `json:"finish_reason"`
}

// Task represents a task to be executed by the agent
type Task struct {
	ID          string
	Name        string
	Description string
	Priority    int
	Status      string // pending, running, completed, failed
	CreatedAt   int64
	CompletedAt *int64
	Result      string
}

// Subagent represents a background sub-agent for parallel tasks
type Subagent struct {
	ID          string
	Name        string
	Description string
	Status      string // pending, running, completed, error
	Prompt      string
	CreatedAt   int64
	Progress    int
	Message     string
}

// TODOItem represents an item in the todo list
type TODOItem struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Completed   bool   `json:"completed"`
	CreatedAt   int64  `json:"created_at"`
	CompletedAt *int64 `json:"completed_at,omitempty"`
}

// Email represents an email message
type Email struct {
	ID      string
	From    string
	To      string
	Subject string
	Body    string
	Date    int64
	Read    bool
}

// ProgressReport represents a progress report to send
type ProgressReport struct {
	Date      int64    `json:"date"`
	Tasks     []string `json:"tasks"`
	Status    string   `json:"status"`
	NextSteps []string `json:"next_steps"`
}
