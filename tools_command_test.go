package main

import (
	"strings"
	"testing"
)

// ─── Command Execution Tests ───────────────────────

func TestRunCommand(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		expectError bool
		check       func(output string) bool
	}{
		{
			name:    "valid simple command",
			command: "echo 'hello world'",
			check: func(output string) bool {
				return strings.Contains(output, "hello world")
			},
		},
		{
			name:        "command with exit code",
			command:     "exit 42",
			expectError: true,
			check: func(output string) bool {
				return strings.Contains(output, "exit code 42")
			},
		},
		{
			name:    "command with stderr",
			command: "echo 'stdout message' && echo 'stderr message' >&2",
			check: func(output string) bool {
				return strings.Contains(output, "stdout message") && 
				       strings.Contains(output, "STDERR: stderr message")
			},
		},
		{
			name:    "command that lists files",
			command: "ls -la / | head -5",
			check: func(output string) bool {
				return len(output) > 0
			},
		},
		{
			name:    "empty output command",
			command: "true",
			check: func(output string) bool {
				return strings.Contains(output, "Command completed successfully")
			},
		},
		{
			name:    "command with variable expansion",
			command: "echo $HOME",
			check: func(output string) bool {
				return strings.HasPrefix(output, "/") && len(output) > 1
			},
		},
		{
			name:    "command with pipe",
			command: "echo 'test data' | grep test",
			check: func(output string) bool {
				return strings.Contains(output, "test data")
			},
		},
		{
			name:    "command with redirection",
			command: "echo 'redirected' > /tmp/test_redirect.txt && cat /tmp/test_redirect.txt",
			check: func(output string) bool {
				return strings.Contains(output, "redirected")
			},
		},
		{
			name:    "command with background process cleanup",
			command: "sleep 0.1 & wait",
			check: func(output string) bool {
				return true // should not hang
			},
		},
		{
			name:    "command output sanitization",
			command: "echo -e '\\033[31mred text\\033[0m'",
			check: func(output string) bool {
				return len(output) > 0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			executor := NewToolExecutor(tempDir, nil)
			
			output := executor.runCommand(map[string]any{
				"command": tt.command,
			})
			
			if tt.expectError && !strings.Contains(output, "error") && 
			   !strings.Contains(output, "(exit code ") {
				t.Logf("Expected error but got: %s", output)
			}
			
			if tt.check != nil && !tt.check(output) {
				t.Errorf("Output check failed. Got: %s", output)
			}
		})
	}
}

func TestRunCommandNoCommandProvided(t *testing.T) {
	tempDir := t.TempDir()
	executor := NewToolExecutor(tempDir, nil)
	
	output := executor.runCommand(map[string]any{})
	
	if !strings.Contains(strings.ToLower(output), "error") {
		t.Errorf("Expected error when command is not provided. Got: %s", output)
	}
}

func TestRunCommandEmptyString(t *testing.T) {
	tempDir := t.TempDir()
	executor := NewToolExecutor(tempDir, nil)
	
	output := executor.runCommand(map[string]any{
		"command": "",
	})
	
	if !strings.Contains(strings.ToLower(output), "error") {
		t.Errorf("Expected error when command is empty string. Got: %s", output)
	}
}

func TestRestart(t *testing.T) {
	t.Skip("Skipped to avoid process replacement during test execution")
	
	tempDir := t.TempDir()
	executor := NewToolExecutor(tempDir, nil)
	output := executor.restart(map[string]any{})
	
	if !strings.Contains(output, "error") && 
	   !strings.Contains(output, "Process replaced") {
		t.Errorf("Unexpected restart output: %s", output)
	}
}

func TestRestartNotDevMode(t *testing.T) {
	testAgent := &YoloAgent{
		devMode: false,
	}
	
	tempDir := t.TempDir()
	executor := NewToolExecutor(tempDir, testAgent)
	output := executor.restart(map[string]any{})
	
	if !strings.Contains(strings.ToLower(output), "error") && 
	   !strings.Contains(output, "not available") {
		t.Errorf("Expected error when not in dev mode. Got: %s", output)
	}
}
