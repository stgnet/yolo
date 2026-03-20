package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"sync"
)

// ─── Text-to-Speech (TTS) Support ──────┬───────────────

// TTSManager handles text-to-speech output for YOLO
type TTSManager struct {
	enabled bool
	mu      sync.Mutex
}

// NewTTSManager creates a new TTS manager with default settings
func NewTTSManager() *TTSManager {
	return &TTSManager{
		enabled: false, // Disabled by default
	}
}

// SetEnabled enables or disables TTS output
func (t *TTSManager) SetEnabled(enabled bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.enabled = enabled
	if enabled {
		fmt.Println("  ✓ TTS enabled - YOLO will now speak responses")
	} else {
		fmt.Println("  ✗ TTS disabled - YOLO will no longer speak responses")
	}
}

// IsEnabled returns whether TTS is currently enabled
func (t *TTSManager) IsEnabled() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.enabled
}

// Speak outputs text using the system's TTS engine (macOS `say` command)
func (t *TTSManager) Speak(text string) {
	t.mu.Lock()
	enabled := t.enabled
	t.mu.Unlock()

	if !enabled || text == "" {
		return
	}

	// Clean the text for TTS - remove ANSI color codes, trim whitespace, limit length
	cleanText := t.cleanTextForTTS(text)
	if cleanText == "" {
		return
	}

	// Limit to first 500 characters to avoid very long speech
	if len(cleanText) > 500 {
		cleanText = cleanText[:500] + "..."
	}

	// Run the say command asynchronously (non-blocking)
	go func() {
		cmd := exec.Command("say", "-v", "Fred", cleanText)
		_ = cmd.Start() // Start but don't wait (fire and forget)
	}()
}

// SpeakSync outputs text synchronously (blocking)
func (t *TTSManager) SpeakSync(text string) {
	t.mu.Lock()
	enabled := t.enabled
	t.mu.Unlock()

	if !enabled || text == "" {
		return
	}

	cleanText := t.cleanTextForTTS(text)
	if cleanText == "" {
		return
	}

	if len(cleanText) > 500 {
		cleanText = cleanText[:500] + "..."
	}

	cmd := exec.Command("say", "-v", "Fred", cleanText)
	_ = cmd.Run() // Wait for completion (blocking)
}

// cleanTextForTTS removes ANSI codes, excessive whitespace, and other problematic characters
func (t *TTSManager) cleanTextForTTS(text string) string {
	// Remove ANSI escape sequences (color codes)
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	text = ansiRegex.ReplaceAllString(text, "")

	// Replace multiple newlines with single newline
	text = strings.Join(strings.Fields(text), " ")

	// Remove excessive punctuation
	text = strings.Map(func(r rune) rune {
		switch r {
		case '.', '!', '?':
			return r
		case ',', ';', ':', '-', '_', '/', '\\', '@', '#', '$', '%', '^', '&', '*', '(', ')', '[', ']', '{', '}', '|', '~', '`':
			return ' ' // Replace with space
		default:
			return r
		}
	}, text)

	// Trim and return
	return strings.TrimSpace(text)
}

// getStatus returns current TTS status for display
func (t *TTSManager) getStatus() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.enabled {
		return "TTS: ON"
	}
	return "TTS: OFF"
}
