package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"sync"
)

// ─── Text-to-Speech (TTS) Support ──────┬──────────

// TTSManager handles text-to-speech output for YOLO
type TTSManager struct {
	enabled bool
	mu      sync.Mutex
	voice   string
	queue   chan string // Queue to ensure sequential speech
	closed  bool
}

// NewTTSManager creates a new TTS manager with default settings
func NewTTSManager() *TTSManager {
	tts := &TTSManager{
		enabled: true, // Enabled by default so YOLO speaks responses
		voice:   "Samantha", // Better quality voice than Fred
		queue:   make(chan string, 100), // Buffer for up to 100 messages
	}
	go tts.processQueue()
	return tts
}

// SetVoice changes the TTS voice (e.g., "Samantha", "Fred", "Zira", "Alice")
func (t *TTSManager) SetVoice(voice string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.voice = voice
	fmt.Printf("  ✓ TTS voice changed to: %s\n", voice)
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

// Speak queues text to be spoken sequentially using the system's TTS engine
func (t *TTSManager) Speak(text string) {
	t.mu.Lock()
	enabled := t.enabled
	closed := t.closed
	t.mu.Unlock()

	if !enabled || text == "" || closed {
		return
	}

	// Clean the text for TTS
	cleanText := t.cleanTextForTTS(text)
	if cleanText == "" {
		return
	}

	// Limit to first 500 characters to avoid very long speech
	if len(cleanText) > 500 {
		cleanText = cleanText[:500] + "..."
	}

	// Add to queue (non-blocking with capacity limit)
	select {
	case t.queue <- cleanText:
		// Queued successfully
	default:
		// Queue full, drop message to prevent blocking
		fmt.Println("  ⚠ TTS queue full, message dropped")
	}
}

// processQueue handles the TTS queue sequentially
func (t *TTSManager) processQueue() {
	for text := range t.queue {
		t.mu.Lock()
		enabled := t.enabled
		voice := t.voice
		t.mu.Unlock()

		if enabled && text != "" {
			// Run say command synchronously to ensure sequential speech
			cmd := exec.Command("say", "-v", voice, text)
			_ = cmd.Run() // Wait for completion before next message
		}
	}
}

// SpeakSync outputs text immediately (for emergency/interactive use)
func (t *TTSManager) SpeakSync(text string) {
	t.mu.Lock()
	enabled := t.enabled
	voice := t.voice
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

	cmd := exec.Command("say", "-v", voice, cleanText)
	_ = cmd.Run() // Wait for completion (blocking)
}

// Stop shuts down the TTS queue processor
func (t *TTSManager) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.closed {
		close(t.queue)
		t.closed = true
	}
}

// cleanTextForTTS removes ANSI codes, excessive whitespace, and other problematic characters
func (t *TTSManager) cleanTextForTTS(text string) string {
	// Remove ANSI escape sequences (color codes)
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	text = ansiRegex.ReplaceAllString(text, "")

	// Replace multiple newlines with single newline
	text = strings.Join(strings.Fields(text), " ")

	// Remove excessive punctuation that might cause issues
	text = strings.Map(func(r rune) rune {
		switch r {
		case '.', '!', '?', ',':
			return r // Keep basic sentence punctuation
		case ';', ':', '-', '_', '/', '\\', '@', '#', '$', '%', '^', '&', '*', '(', ')', '[', ']', '{', '}', '|', '~', '`':
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
		return fmt.Sprintf("TTS: ON (%s)", t.voice)
	}
	return "TTS: OFF"
}
