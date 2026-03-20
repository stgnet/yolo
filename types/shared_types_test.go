package types

import (
	"encoding/json"
	"testing"
	"time"
)

func TestStatusJSONSerialization(t *testing.T) {
	status := Status{
		Request:       "GET /status",
		Response:      "OK",
		Code:          200,
		Uptime:        "1h2m3s",
		RequestsTotal: 1234,
		ServerTime:    time.Now(),
		Version:       "v1.0.0",
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Failed to marshal Status: %v", err)
	}

	var unmarshaled Status
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal Status: %v", err)
	}

	if unmarshaled.Request != status.Request {
		t.Errorf("Expected Request %q, got %q", status.Request, unmarshaled.Request)
	}
	if unmarshaled.Code != status.Code {
		t.Errorf("Expected Code %d, got %d", status.Code, unmarshaled.Code)
	}
}

func TestAgentStatusJSONSerialization(t *testing.T) {
	status := AgentStatus{
		Status:  "running",
		Version: "v1.0.0",
		Message: "All systems operational",
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Failed to marshal AgentStatus: %v", err)
	}

	var unmarshaled AgentStatus
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal AgentStatus: %v", err)
	}

	if unmarshaled.Status != status.Status {
		t.Errorf("Expected Status %q, got %q", status.Status, unmarshaled.Status)
	}
}

func TestToolOutputJSONSerialization(t *testing.T) {
	output := ToolOutput{
		ID:        "tool-123",
		Name:      "file-read",
		Status:    "success",
		Content:   map[string]interface{}{"content": "test data"},
		Timestamp: time.Now(),
		Duration:  100 * time.Millisecond,
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal ToolOutput: %v", err)
	}

	var unmarshaled ToolOutput
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal ToolOutput: %v", err)
	}

	if unmarshaled.ID != output.ID {
		t.Errorf("Expected ID %q, got %q", output.ID, unmarshaled.ID)
	}
	if unmarshaled.Status != output.Status {
		t.Errorf("Expected Status %q, got %q", output.Status, unmarshaled.Status)
	}
}

func TestCommandResultJSONSerialization(t *testing.T) {
	result := CommandResult{
		Command:       "ls -la",
		Output:        "total 100\ndrwxr-xr-x .",
		ExitCode:      0,
		ExecutionTime: 50 * time.Millisecond,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal CommandResult: %v", err)
	}

	var unmarshaled CommandResult
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal CommandResult: %v", err)
	}

	if unmarshaled.Command != result.Command {
		t.Errorf("Expected Command %q, got %q", result.Command, unmarshaled.Command)
	}
	if unmarshaled.ExitCode != result.ExitCode {
		t.Errorf("Expected ExitCode %d, got %d", result.ExitCode, unmarshaled.ExitCode)
	}
}

func TestEmailJSONSerialization(t *testing.T) {
	email := Email{
		From:    "sender@example.com",
		To:      "recipient@example.com",
		Subject: "Test Subject",
		Body:    "Test body content",
		Read:    false,
	}

	data, err := json.Marshal(email)
	if err != nil {
		t.Fatalf("Failed to marshal Email: %v", err)
	}

	var unmarshaled Email
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal Email: %v", err)
	}

	if unmarshaled.From != email.From {
		t.Errorf("Expected From %q, got %q", email.From, unmarshaled.From)
	}
	if unmarshaled.Subject != email.Subject {
		t.Errorf("Expected Subject %q, got %q", email.Subject, unmarshaled.Subject)
	}
}

func TestTodoItemJSONSerialization(t *testing.T) {
	now := time.Now()
	todo := TodoItem{
		Title:       "Test task",
		Description: "A test description",
		Status:      "pending",
		CreatedAt:   now,
		Priority:    3,
		Tags:        []string{"urgent", "testing"},
	}

	data, err := json.Marshal(todo)
	if err != nil {
		t.Fatalf("Failed to marshal TodoItem: %v", err)
	}

	var unmarshaled TodoItem
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal TodoItem: %v", err)
	}

	if unmarshaled.Title != todo.Title {
		t.Errorf("Expected Title %q, got %q", todo.Title, unmarshaled.Title)
	}
	if unmarshaled.Priority != todo.Priority {
		t.Errorf("Expected Priority %d, got %d", todo.Priority, unmarshaled.Priority)
	}
	if len(unmarshaled.Tags) != len(todo.Tags) {
		t.Errorf("Expected %d tags, got %d", len(todo.Tags), len(unmarshaled.Tags))
	}
}

func TestProgressReportJSONSerialization(t *testing.T) {
	report := ProgressReport{
		Timestamp:      time.Now(),
		CompletedTasks: []string{"task1", "task2"},
		PendingTasks:   []string{"task3"},
		IssuesFound:    []string{"issue1"},
		NextSteps:      []string{"step1"},
	}

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("Failed to marshal ProgressReport: %v", err)
	}

	var unmarshaled ProgressReport
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal ProgressReport: %v", err)
	}

	if len(unmarshaled.CompletedTasks) != len(report.CompletedTasks) {
		t.Errorf("Expected %d completed tasks, got %d", len(report.CompletedTasks), len(unmarshaled.CompletedTasks))
	}
	if len(unmarshaled.NextSteps) != len(report.NextSteps) {
		t.Errorf("Expected %d next steps, got %d", len(report.NextSteps), len(unmarshaled.NextSteps))
	}
}

func TestModelProviderConstants(t *testing.T) {
	if ModelProviderOllama != "ollama" {
		t.Errorf("Expected ModelProviderOllama to be 'ollama', got '%s'", ModelProviderOllama)
	}
	if ModelProviderOpenAI != "openai" {
		t.Errorf("Expected ModelProviderOpenAI to be 'openai', got '%s'", ModelProviderOpenAI)
	}
	if ModelProviderAnthropic != "anthropic" {
		t.Errorf("Expected ModelProviderAnthropic to be 'anthropic', got '%s'", ModelProviderAnthropic)
	}
}
