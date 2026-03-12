package main

import (
	"strings"
	"testing"
)

// TestEmailResponseShowsActionsTaken verifies that actions taken are shown in responses
func TestEmailResponseShowsActionsTaken(t *testing.T) {
	agent := &YoloAgent{config: NewYoloConfig(".")}
	tex := NewToolExecutor(".", agent)

	email := EmailMessage{
		From:    "scott@stg.net",
		Subject: "How is it going?",
		Content: "What's the current test coverage? Are all tests passing?",
	}

	response := tex.composeResponseToEmail(email)

	t.Logf("Response:\n%s\n", response)

	// Should show actions taken
	if !strings.Contains(response, "ACTIONS TAKEN:") {
		t.Error("Expected response to include ACTIONS TAKEN section showing what was done")
	}

	// Should show at least one action
	hasAction := strings.Contains(response, "Generating system status") || 
		         strings.Contains(response, "Gathering current status") ||
		         strings.Contains(response, "Checking test coverage")
	if !hasAction {
		t.Error("Expected response to show at least one action taken")
	}

	// Should provide actual answers with data
	hasData := strings.Contains(response, "Test Coverage:") ||
	          strings.Contains(response, "tests") && strings.Contains(response, "passing") ||
	          strings.Contains(response, "status")
	if !hasData {
		t.Error("Expected response to include actual system data/status information")
	}

	// Should be personalized for Scott
	if !strings.Contains(response, "Hi Scott") {
		t.Error("Expected personalized greeting for Scott")
	}
}
