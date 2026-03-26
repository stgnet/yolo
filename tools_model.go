package main

import (
	"fmt"
	"os/exec"
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

// checkOllamaStatus checks whether the Ollama server is running and reachable.
func (t *ToolExecutor) checkOllamaStatus(args map[string]any) string {
	var result strings.Builder

	cmd := exec.Command("pgrep", "-f", "ollama serve")
	ollamaRunning := cmd.Run() == nil

	result.WriteString("=== Ollama Status ===\n\n")
	if ollamaRunning {
		result.WriteString("Ollama server is running\n\n")
	} else {
		result.WriteString("Ollama server is NOT running\n\n")
		result.WriteString("Hint: Run 'ollama serve' in the background to start it.\n\n")
	}

	if ollamaRunning && t.agent != nil {
		models := t.agent.ollama.ListModels()
		if len(models) > 0 {
			result.WriteString("API Status: Ollama API is reachable.\n")
			result.WriteString(fmt.Sprintf("Available models (%d):\n", len(models)))
			for _, m := range models {
				result.WriteString(fmt.Sprintf("  - %s\n", m))
			}
		} else {
			result.WriteString("API Status: Ollama API responded but no models found.\n")
		}
	}

	return result.String()
}
