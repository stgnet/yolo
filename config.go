package main

import (
	"os"
	"path/filepath"
	"regexp"
)

// ─── Configuration ────────────────────────────────────────────────────

const (
	YoloDir            = ".yolo"
	IdleThinkDelay     = 30  // seconds of no input before autonomous thinking
	ThinkLoopDelay     = 120 // seconds between autonomous think cycles
	MaxContextMessages = 40
	MaxToolOutput      = 0  // 0 = no truncation
	ToolNudgeAfter     = 0  // 0 = disabled
	CommandTimeout     = 30   // shell command timeout in seconds
	DefaultNumCtx      = 8192 // default context window size for Ollama models
)

var (
	HistoryFile   = filepath.Join(YoloDir, "history.json")
	OllamaURL     = getEnvDefault("OLLAMA_URL", "http://localhost:11434")
	NumCtxOverride = os.Getenv("YOLO_NUM_CTX") // if set, overrides auto-detected context size
	SubagentDir   = filepath.Join(YoloDir, "subagents")
	fileNameRegex = regexp.MustCompile(`agent_(\d+)\.json`)
)

func getEnvDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ─── ANSI Colors ──────────────────────────────────────────────────────

const (
	Reset   = "\033[0m"
	Bold    = "\033[1m"
	Dim     = "\033[2m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	Gray    = "\033[90m"
)
