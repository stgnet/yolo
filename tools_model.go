package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ─── Model and Status Tools ─────────────────────────────────────────

func (t *ToolExecutor) listModels() string {
	if t.agent != nil {
		models := t.agent.ollama.ListModels()
		if len(models) == 0 {
			return "No models found"
		}
		return strings.Join(models, "\n")
	}
	return errorMessage("no agent context")
}

func (t *ToolExecutor) switchModel(args map[string]any) string {
	model := getStringArg(args, "model", "")
	if t.agent != nil {
		return t.agent.switchModel(model)
	}
	return errorMessage("no agent context")
}

// getBaseDir returns the YOLO project root directory
func getBaseDir() string {
	dirs := []string{".", "..", "../../.."}

	for _, dir := range dirs {
		goMod := filepath.Join(dir, "go.mod")
		if info, err := os.Stat(goMod); err == nil && !info.IsDir() {
			absPath, _ := filepath.Abs(dir)
			return absPath
		}
	}

	absPath, _ := filepath.Abs(".")
	return absPath
}

// checkOllamaStatus checks Ollama server status and reads debug logs
func (t *ToolExecutor) checkOllamaStatus(args map[string]any) string {
	lines := getIntArg(args, "lines", 50)

	var result strings.Builder

	cmd := exec.Command("pgrep", "-f", "ollama serve")
	ollamaRunning := cmd.Run() == nil

	result.WriteString("=== Ollama Status ===\n\n")
	if ollamaRunning {
		result.WriteString("✓ Ollama server is running\n\n")
	} else {
		result.WriteString("✗ Ollama server is NOT running\n\n")
		result.WriteString("Hint: Run 'ollama serve' in the background to start it.\n\n")
	}

	logFile := "./logs/ollama.log"
	errLogFile := "./logs/ollama.err.log"

	_ = getBaseDir()

	hasLogs := false

	if info, err := os.Stat(logFile); err == nil && !info.IsDir() {
		hasLogs = true
		result.WriteString(fmt.Sprintf("=== Log File: %s ===\n", logFile))

		content, err := os.ReadFile(logFile)
		if err != nil {
			result.WriteString(fmt.Sprintf("Error reading log file: %v\n", err))
		} else {
			linesArray := strings.Split(strings.TrimSpace(string(content)), "\n")
			start := 0
			if len(linesArray) > lines {
				start = len(linesArray) - lines
			}

			for i := start; i < len(linesArray); i++ {
				if linesArray[i] != "" {
					result.WriteString(linesArray[i] + "\n")
				}
			}
		}
		result.WriteString("\n")
	}

	if info, err := os.Stat(errLogFile); err == nil && !info.IsDir() {
		hasLogs = true
		result.WriteString(fmt.Sprintf("=== Error Log File: %s ===\n", errLogFile))

		content, err := os.ReadFile(errLogFile)
		if err != nil {
			result.WriteString(fmt.Sprintf("Error reading error log file: %v\n", err))
		} else {
			linesArray := strings.Split(strings.TrimSpace(string(content)), "\n")
			start := 0
			if len(linesArray) > lines {
				start = len(linesArray) - lines
			}

			for i := start; i < len(linesArray); i++ {
				if linesArray[i] != "" {
					result.WriteString(linesArray[i] + "\n")
				}
			}
		}
		result.WriteString("\n")
	}

	if !hasLogs {
		result.WriteString("No log files found at ./logs/\n")
		result.WriteString("Enable logging by setting OLLAMA_DEBUG=1 or YOLO_OLLAMA_LOG=1\n\n")

		if ollamaRunning {
			client := NewOllamaClient("http://localhost:11434", "")
			models := client.ListModels()
			if len(models) > 0 {
				result.WriteString("API Status: Ollama API is reachable.\n")
				result.WriteString("Available models:\n")
				for _, m := range models {
					result.WriteString(fmt.Sprintf("  - %s\n", m))
				}
			} else if len(models) == 0 {
				result.WriteString("API Status: Ollama API responded but no models found.\n")
			}
		}
	}

	return result.String()
}
