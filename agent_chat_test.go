package main

import (
	"testing"
)

// TestAgentAutonomousModeMessage verifies the autonomous mode system message is added correctly
func TestAgentAutonomousModeMessage(t *testing.T) {
	a := NewYoloAgent()

	// Simulate what happens in chatWithAgent when autonomous=true
	autonomousMsg := "No new user input. You are in autonomous mode. Continue making progress on your own — do NOT ask the user for input or confirmation. Pick the most impactful next task and execute it using tools. Focus on: code quality, bug fixes, tests, self-improvement, or new features. Act decisively. Do the work, then move to the next thing."

	// Add the message like chatWithAgent does (this is added as a "user" message)
	a.history.AddMessage("user", autonomousMsg, nil)

	// Get messages and verify
	msgs := a.history.GetContextMessages(MaxContextMessages)

	if len(msgs) == 0 {
		t.Error("Expected at least one message in history")
	}

	lastMsg := msgs[len(msgs)-1]
	if lastMsg.Role != "user" {
		t.Errorf("Expected last message role to be 'user', got '%s'", lastMsg.Role)
	}

	if len(lastMsg.Content) < 100 {
		t.Error("Expected autonomous mode message to have substantial content")
	}
}

// TestAgentUserMessageAdded verifies user messages are properly recorded
func TestAgentUserMessageAdded(t *testing.T) {
	a := NewYoloAgent()

	testMsg := "Test user input message"
	a.history.AddMessage("user", testMsg, nil)

	msgs := a.history.GetContextMessages(MaxContextMessages)

	if len(msgs) == 0 {
		t.Error("Expected at least one message in history")
	}

	lastMsg := msgs[len(msgs)-1]
	if lastMsg.Role != "user" {
		t.Errorf("Expected last message role to be 'user', got '%s'", lastMsg.Role)
	}

	if lastMsg.Content != testMsg {
		t.Errorf("Expected message content '%s', got '%s'", testMsg, lastMsg.Content)
	}
}

// TestAgentSystemPromptGeneration verifies system prompt is generated correctly
func TestAgentSystemPromptGeneration(t *testing.T) {
	a := NewYoloAgent()

	prompt := a.getSystemPrompt()

	if len(prompt) == 0 {
		t.Error("Expected non-empty system prompt")
	}

	if len(prompt) < 100 {
		t.Error("Expected substantial system prompt content")
	}

	// Verify key components are present
	expectedKeywords := []string{"YOLO", "autonomous", "tools"}
	for _, keyword := range expectedKeywords {
		found := false
		// Simple substring check
		promptLower := toLowerOnly(prompt)
		keywordLower := toLowerOnly(keyword)
		if len(promptLower) >= len(keywordLower) {
			maxIdx := len(promptLower) - len(keywordLower)
			for i := 0; i <= maxIdx; i++ {
				if promptLower[i:i+len(keywordLower)] == keywordLower {
					found = true
					break
				}
			}
		}
		if !found {
			t.Logf("Warning: System prompt may be missing keyword: %s", keyword)
		}
	}
}

// toLowerOnly converts string to lowercase without imports
func toLowerOnly(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c = c + ('a' - 'A')
		}
		result[i] = c
	}
	return string(result)
}

func promptLen(s string) int {
	return len(s)
}
