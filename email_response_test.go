package main

import (
	"strings"
	"testing"
)

// TestComposeResponseToEmail_Feedback tests response to feedback about not answering
func TestComposeResponseToEmail_Feedback(t *testing.T) {
	t.Skip("Skipping: composeResponseToEmail runs shell commands (go test, sendmail) that block in CI")
	agent := &YoloAgent{config: NewYoloConfig(".")}
	tex := NewToolExecutor(".", agent)

	email := EmailMessage{
		From:    "scott@stg.net",
		Subject: "Email responses need improvement",
		Content: "You are still not answering my questions. That is a problem.",
	}

	response := tex.composeResponseToEmail(email)

	// Should acknowledge the feedback specifically
	if !strings.Contains(response, "not answering") && !strings.Contains(response, "generic responses") {
		t.Errorf("Expected response to acknowledge the feedback about not answering")
	}

	// Should show actions were taken
	if !strings.Contains(response, "ACTIONS TAKEN:") {
		t.Errorf("Expected ACTIONS TAKEN section in response")
	}

	// Should include actual system checks
	if !strings.Contains(response, "test coverage") && !strings.Contains(response, "coverage") {
		t.Errorf("Expected response to include actual test coverage information")
	}
}

// TestComposeResponseToEmail_StatusQuestion tests response to status questions
func TestComposeResponseToEmail_StatusQuestion(t *testing.T) {
	t.Skip("Skipping: composeResponseToEmail runs shell commands (go test, sendmail) that block in CI")
	agent := &YoloAgent{config: NewYoloConfig(".")}
	tex := NewToolExecutor(".", agent)

	email := EmailMessage{
		From:    "scott@stg.net",
		Subject: "How is it going?",
		Content: "Hey, how is progress going? Update me on what you're working on.",
	}

	response := tex.composeResponseToEmail(email)

	// Should gather and report status
	if !strings.Contains(response, "status") && !strings.Contains(response, "Status") {
		t.Errorf("Expected response to include status information")
	}

	// Should show actions were taken
	if !strings.Contains(response, "ACTIONS TAKEN:") {
		t.Errorf("Expected ACTIONS TAKEN section in response")
	}
}

// TestComposeResponseToEmail_CapabilitiesQuestion tests response to capability questions
func TestComposeResponseToEmail_CapabilitiesQuestion(t *testing.T) {
	t.Skip("Skipping: composeResponseToEmail runs shell commands (go test, sendmail) that block in CI")
	agent := &YoloAgent{config: NewYoloConfig(".")}
	tex := NewToolExecutor(".", agent)

	email := EmailMessage{
		From:    "scott@stg.net",
		Subject: "What can you do?",
		Content: "What are you capable of? Can you actually read and modify code?",
	}

	response := tex.composeResponseToEmail(email)

	// Should list capabilities
	if !strings.Contains(response, "Read and modify") {
		t.Errorf("Expected response to list capability: Read and modify code")
	}

	if !strings.Contains(response, "tests") {
		t.Errorf("Expected response to mention running tests")
	}

	if !strings.Contains(response, "web") || !strings.Contains(response, "search") {
		t.Errorf("Expected response to mention web search capability")
	}
}

// TestComposeResponseToEmail_Request tests response to actionable requests
func TestComposeResponseToEmail_Request(t *testing.T) {
	t.Skip("Skipping: composeResponseToEmail runs shell commands (go test, sendmail) that block in CI")
	agent := &YoloAgent{config: NewYoloConfig(".")}
	tex := NewToolExecutor(".", agent)

	email := EmailMessage{
		From:    "scott@stg.net",
		Subject: "Help me with something",
		Content: "Please help me improve the test coverage. I need you to add more tests.",
	}

	response := tex.composeResponseToEmail(email)

	// Should acknowledge the request
	if !strings.Contains(response, "request") {
		t.Errorf("Expected response to acknowledge the request")
	}

	// Should show actions were taken
	if !strings.Contains(response, "ACTIONS TAKEN:") {
		t.Errorf("Expected ACTIONS TAKEN section in response")
	}
}

// TestComposeResponseToEmail_FactualQuestion tests response to factual questions requiring web search
func TestComposeResponseToEmail_FactualQuestion(t *testing.T) {
	t.Skip("Skipping: composeResponseToEmail runs shell commands (go test, sendmail) that block in CI")
	agent := &YoloAgent{config: NewYoloConfig(".")}
	tex := NewToolExecutor(".", agent)

	email := EmailMessage{
		From:    "scott@stg.net",
		Subject: "How does Go concurrency work?",
		Content: "Can you explain how goroutines work in Go? What is the difference between channels and mutexes?",
	}

	response := tex.composeResponseToEmail(email)

	// Should attempt to search for information
	if !strings.Contains(response, "ACTIONS TAKEN:") {
		t.Errorf("Expected ACTIONS TAKEN section showing web search was attempted")
	}

	// Should either provide info or acknowledge search limitation
	hasAnswer := strings.Contains(response, "goroutine") || strings.Contains(response, "channel") ||
		strings.Contains(response, "mutex") || strings.Contains(response, "searched for")
	if !hasAnswer {
		t.Errorf("Expected response to include information from web search or acknowledgment of search attempt")
	}
}

// TestComposeResponseToEmail_Generic tests response to generic emails without questions
func TestComposeResponseToEmail_Generic(t *testing.T) {
	t.Skip("Skipping: composeResponseToEmail runs shell commands (go test, sendmail) that block in CI")
	agent := &YoloAgent{config: NewYoloConfig(".")}
	tex := NewToolExecutor(".", agent)

	email := EmailMessage{
		From:    "scott@stg.net",
		Subject: "Hello",
		Content: "Just wanted to say hello and check in.",
	}

	response := tex.composeResponseToEmail(email)

	// Should provide a polite acknowledgment
	if !strings.Contains(response, "Thank you") && !strings.Contains(response, "Best regards") {
		t.Errorf("Expected response to include polite acknowledgment")
	}

	// Generic emails may not trigger ACTIONS TAKEN if no specific request detected
	// This is acceptable behavior for purely social messages
}
