package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ─── Subagent Management Tools ───────────────────────────────────────

func (t *ToolExecutor) spawnSubagent(args map[string]any) string {
	prompt := getStringArg(args, "prompt", "")
	if prompt == "" {
		return errorMessage("'prompt' parameter is required")
	}

	name := getStringArg(args, "name", "")

	if t.agent != nil {
		task := prompt
		if name != "" {
			task = fmt.Sprintf("[%s] %s", name, prompt)
		}
		return t.agent.spawnSubagent(task, "")
	}

	return errorMessage("no agent context")
}

func (t *ToolExecutor) listSubagents(args map[string]any) string {
	files, err := filepath.Glob(filepath.Join(t.agent.config.GetSubagentDir(), "agent_*.json"))
	if err != nil {
		return errorMessage("could not read subagent directory: %v", err)
	}

	if len(files) == 0 {
		return "No subagents found"
	}

	var results []string
	for _, file := range files {
		filename := filepath.Base(file)
		idMatch := fileNameRegex.FindStringSubmatch(filename)
		if len(idMatch) < 2 {
			continue
		}
		agentID := idMatch[1]

		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		var result map[string]any
		if err := json.Unmarshal(data, &result); err != nil {
			continue
		}

		status := getStringArg(result, "status", "")
		task := truncateString(getStringArg(result, "task", ""), 40)
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		modTime := info.ModTime().Format("15:04:05")

		results = append(results, fmt.Sprintf("#%s [%s] %s (updated: %s)", agentID, status, task, modTime))
	}

	return "Active subagents:\n" + strings.Join(results, "\n")
}

func (t *ToolExecutor) readSubagentResult(args map[string]any) string {
	agentID := getIntArg(args, "id", 0)
	if agentID == 0 {
		return errorMessage("required parameter 'id' is missing")
	}

	resultFile := filepath.Join(t.agent.config.GetSubagentDir(), fmt.Sprintf("agent_%d.json", agentID))
	data, err := os.ReadFile(resultFile)
	if err != nil {
		return errorMessage("could not read subagent result: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return errorMessage("error parsing result: %v", err)
	}

	status := getStringArg(result, "status", "")
	output := fmt.Sprintf("Sub-agent #%d Result:\n", agentID)
	output += fmt.Sprintf("  Task: %s\n", getStringArg(result, "task", ""))
	output += fmt.Sprintf("  Model: %s\n", getStringArg(result, "model", ""))
	output += fmt.Sprintf("  Status: %s\n", status)
	if status == "in-progress" {
		output += "  Result: (still running, check back later)\n"
	} else {
		output += fmt.Sprintf("  Result: %s\n", getStringArg(result, "result", ""))
	}

	return output
}

func (t *ToolExecutor) summarizeSubagents(args map[string]any) string {
	files, err := filepath.Glob(filepath.Join(t.agent.config.GetSubagentDir(), "agent_*.json"))
	if err != nil {
		return errorMessage("could not read subagent directory: %v", err)
	}

	if len(files) == 0 {
		return "Subagent Summary (0 total):\n  Completed: 0\n  Errors: 0\n\nRecent subagents:\n(no subagents running)"
	}

	completed := 0
	errors := 0
	var summaries []string

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		var result map[string]any
		if err := json.Unmarshal(data, &result); err != nil {
			continue
		}

		id := getIntArg(result, "id", 0)
		status := getStringArg(result, "status", "")
		task := getStringArg(result, "task", "")

		if status == "complete" {
			completed++
		} else if status == "error" {
			errors++
		}

		summaries = append(summaries, fmt.Sprintf("  #%d [%s]: %s", id, status, truncateString(task, 50)))
	}

	output := fmt.Sprintf("Subagent Summary (%d total):\n", len(files))
	output += fmt.Sprintf("  Completed: %d\n", completed)
	output += fmt.Sprintf("  Errors: %d\n", errors)
	output += "\nRecent subagents:\n" + strings.Join(summaries, "\n")

	return output
}
