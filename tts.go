package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// ─── Text-to-Speech (TTS) Support ──────┬──────────

// Pre-compiled regexes for text cleaning.
var (
	ansiRegex      = regexp.MustCompile(`\x1b\[[0-9;]*m`)
	fencedCodeRe   = regexp.MustCompile("(?s)```[^\n]*\n.*?```")
	inlineCodeRe   = regexp.MustCompile("`[^`]+`")
	urlRe          = regexp.MustCompile(`https?://\S+`)
	markdownLinkRe = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)
	headerRe       = regexp.MustCompile(`(?m)^#{1,6}\s+`)
	boldRe         = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	italicRe       = regexp.MustCompile(`\*([^*]+)\*`)
	listBulletRe   = regexp.MustCompile(`(?m)^\s*[-*+]\s+`)
	numberedListRe = regexp.MustCompile(`(?m)^\s*\d+\.\s+`)
	multiSpaceRe   = regexp.MustCompile(`\s{2,}`)
)

// ─── TTS Backend Interface ─────────────────────────

// ttsBackend abstracts platform-specific TTS command invocation.
type ttsBackend interface {
	// Speak runs the TTS command synchronously for the given text and voice.
	Speak(text, voice string) error
	// ListVoices returns available voices, optionally filtered by locale prefix.
	ListVoices(localeFilter string) []TTSVoice
	// DefaultVoice returns a sensible default voice name for this backend.
	DefaultVoice() string
	// Name returns the backend name (e.g. "say", "espeak-ng").
	Name() string
}

// TTSVoice represents an available system voice.
type TTSVoice struct {
	Name   string
	Locale string
}

// detectBackend finds the best available TTS backend on this system.
func detectBackend() ttsBackend {
	if path, err := exec.LookPath("say"); err == nil {
		return &sayBackend{path: path}
	}
	if path, err := exec.LookPath("espeak-ng"); err == nil {
		return &espeakBackend{path: path, ng: true}
	}
	if path, err := exec.LookPath("espeak"); err == nil {
		return &espeakBackend{path: path, ng: false}
	}
	return nil
}

// ─── macOS "say" backend ────────────────────────────

type sayBackend struct {
	path string
}

func (b *sayBackend) Name() string { return "say" }

func (b *sayBackend) DefaultVoice() string { return "Samantha" }

func (b *sayBackend) Speak(text, voice string) error {
	cmd := exec.Command(b.path, "-v", voice, text)
	return cmd.Run()
}

func (b *sayBackend) ListVoices(localeFilter string) []TTSVoice {
	out, err := exec.Command(b.path, "-v", "?").Output()
	if err != nil {
		return nil
	}
	var voices []TTSVoice
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Format: "Name              locale   # sample text"
		hashIdx := strings.Index(line, "#")
		if hashIdx < 0 {
			continue
		}
		prefix := strings.TrimSpace(line[:hashIdx])
		fields := strings.Fields(prefix)
		if len(fields) < 2 {
			continue
		}
		locale := fields[len(fields)-1]
		name := strings.TrimSpace(prefix[:strings.LastIndex(prefix, locale)])
		if localeFilter != "" && !strings.HasPrefix(locale, localeFilter) {
			continue
		}
		voices = append(voices, TTSVoice{Name: name, Locale: locale})
	}
	sort.Slice(voices, func(i, j int) bool {
		return voices[i].Name < voices[j].Name
	})
	return voices
}

// ─── espeak / espeak-ng backend ─────────────────────

type espeakBackend struct {
	path string
	ng   bool // true for espeak-ng, false for espeak
}

func (b *espeakBackend) Name() string {
	if b.ng {
		return "espeak-ng"
	}
	return "espeak"
}

func (b *espeakBackend) DefaultVoice() string { return "en" }

func (b *espeakBackend) Speak(text, voice string) error {
	cmd := exec.Command(b.path, "-v", voice, text)
	return cmd.Run()
}

func (b *espeakBackend) ListVoices(localeFilter string) []TTSVoice {
	out, err := exec.Command(b.path, "--voices").Output()
	if err != nil {
		return nil
	}
	var voices []TTSVoice
	for i, line := range strings.Split(string(out), "\n") {
		if i == 0 { // skip header line
			continue
		}
		fields := strings.Fields(line)
		// Format: "Pty  Language  Age/Gender  VoiceName  File  (Other)"
		if len(fields) < 4 {
			continue
		}
		locale := fields[1]
		name := fields[3]
		if localeFilter != "" && !strings.HasPrefix(locale, localeFilter) {
			continue
		}
		voices = append(voices, TTSVoice{Name: name, Locale: locale})
	}
	sort.Slice(voices, func(i, j int) bool {
		return voices[i].Name < voices[j].Name
	})
	return voices
}

// ─── TTSManager ─────────────────────────────────────

// TTSManager handles text-to-speech output for YOLO
type TTSManager struct {
	enabled bool
	mu      sync.Mutex
	voice   string
	backend ttsBackend
	queue   chan string // Queue to ensure sequential speech
	closed  bool
}

// NewTTSManager creates a new TTS manager, auto-detecting the system backend.
func NewTTSManager() *TTSManager {
	backend := detectBackend()
	voice := ""
	if backend != nil {
		voice = backend.DefaultVoice()
	}
	tts := &TTSManager{
		enabled: backend != nil,
		voice:   voice,
		backend: backend,
		queue:   make(chan string, 100),
	}
	go tts.processQueue()
	return tts
}

// Backend returns the backend name, or "none" if no TTS is available.
func (t *TTSManager) Backend() string {
	if t.backend == nil {
		return "none"
	}
	return t.backend.Name()
}

// SetEnabled enables or disables TTS output
func (t *TTSManager) SetEnabled(enabled bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.enabled = enabled
}

// IsEnabled returns whether TTS is currently enabled
func (t *TTSManager) IsEnabled() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.enabled
}

// GetVoice returns the current voice name.
func (t *TTSManager) GetVoice() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.voice
}

// SetVoice changes the TTS voice. Returns an error if the voice is not available.
func (t *TTSManager) SetVoice(name string) error {
	if t.backend == nil {
		return fmt.Errorf("no TTS backend available")
	}
	voices := t.backend.ListVoices("")
	for _, v := range voices {
		if strings.EqualFold(v.Name, name) {
			t.mu.Lock()
			t.voice = v.Name
			t.mu.Unlock()
			return nil
		}
	}
	return fmt.Errorf("voice %q not found — use /tts voices to list available voices", name)
}

// ListVoices returns available voices, optionally filtered by locale prefix.
func (t *TTSManager) ListVoices(localeFilter string) []TTSVoice {
	if t.backend == nil {
		return nil
	}
	return t.backend.ListVoices(localeFilter)
}

// Speak queues text to be spoken sequentially using the system's TTS engine
func (t *TTSManager) Speak(text string) {
	t.mu.Lock()
	enabled := t.enabled
	closed := t.closed
	hasBackend := t.backend != nil
	t.mu.Unlock()

	if !enabled || !hasBackend || text == "" || closed {
		return
	}

	cleanText := cleanTextForTTS(text)
	if cleanText == "" {
		return
	}

	// Add to queue (non-blocking with capacity limit)
	select {
	case t.queue <- cleanText:
	default:
		// Queue full, drop message to prevent blocking
	}
}

// processQueue handles the TTS queue sequentially
func (t *TTSManager) processQueue() {
	for text := range t.queue {
		t.mu.Lock()
		enabled := t.enabled
		voice := t.voice
		backend := t.backend
		t.mu.Unlock()

		if enabled && backend != nil && text != "" {
			_ = backend.Speak(text, voice)
		}
	}
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

// ─── Text Cleaning ──────────────────────────────────

// cleanTextForTTS transforms LLM markdown output into natural spoken text.
func cleanTextForTTS(text string) string {
	// Remove ANSI escape sequences
	text = ansiRegex.ReplaceAllString(text, "")

	// Remove fenced code blocks entirely — code is not useful spoken aloud
	text = fencedCodeRe.ReplaceAllString(text, "")

	// Remove inline code backticks (keep the text inside for short identifiers)
	text = inlineCodeRe.ReplaceAllStringFunc(text, func(s string) string {
		return strings.Trim(s, "`")
	})

	// Remove bare URLs
	text = urlRe.ReplaceAllString(text, "")

	// Convert markdown links [text](url) to just the link text
	text = markdownLinkRe.ReplaceAllString(text, "$1")

	// Strip markdown header markers
	text = headerRe.ReplaceAllString(text, "")

	// Strip bold/italic markers, keep the text
	text = boldRe.ReplaceAllString(text, "$1")
	text = italicRe.ReplaceAllString(text, "$1")

	// Strip list bullet prefixes
	text = listBulletRe.ReplaceAllString(text, "")
	text = numberedListRe.ReplaceAllString(text, "")

	// Replace characters that cause TTS artifacts with spaces.
	// Keep colons and semicolons — they create natural pauses.
	text = strings.Map(func(r rune) rune {
		switch r {
		case '_', '\\', '@', '#', '$', '%', '^', '&', '*', '(', ')', '[', ']', '{', '}', '|', '~', '`':
			return ' '
		default:
			return r
		}
	}, text)

	// Collapse whitespace and trim
	text = multiSpaceRe.ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}
