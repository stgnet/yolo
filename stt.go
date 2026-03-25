package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ─── Speech-to-Text (STT) Support ───────────────────────────────────
//
// STTManager provides voice input via local speech-to-text backends:
//   - whisper (OpenAI Whisper CLI via whisper.cpp or python whisper)
//   - whisper-cpp (standalone whisper.cpp executable)
//
// Voice input is recorded via system microphone (arecord/sox/ffmpeg),
// transcribed locally, and injected as user input.

// ─── STT Backend Interface ──────────────────────────────────────────

type sttBackend interface {
	// Transcribe converts an audio file to text.
	Transcribe(ctx context.Context, audioPath string) (string, error)
	// Name returns the backend name.
	Name() string
}

// ─── Audio Recorder ─────────────────────────────────────────────────

// audioRecorder abstracts microphone recording.
type audioRecorder struct {
	command string // the recording command found on PATH
}

// detectRecorder finds an available audio recording tool.
func detectRecorder() *audioRecorder {
	// Prefer sox (rec), then arecord, then ffmpeg
	for _, cmd := range []string{"rec", "arecord", "ffmpeg"} {
		if _, err := exec.LookPath(cmd); err == nil {
			return &audioRecorder{command: cmd}
		}
	}
	return nil
}

// Record captures audio from the microphone for the given duration.
// Returns the path to the recorded WAV file.
func (r *audioRecorder) Record(ctx context.Context, duration time.Duration, outPath string) error {
	secs := fmt.Sprintf("%d", int(duration.Seconds()))

	var cmd *exec.Cmd
	switch r.command {
	case "rec":
		// sox's rec command
		cmd = exec.CommandContext(ctx, "rec", "-q", "-r", "16000", "-c", "1", "-b", "16", outPath, "trim", "0", secs)
	case "arecord":
		cmd = exec.CommandContext(ctx, "arecord", "-q", "-f", "S16_LE", "-r", "16000", "-c", "1", "-d", secs, outPath)
	case "ffmpeg":
		cmd = exec.CommandContext(ctx, "ffmpeg", "-y", "-f", "pulse", "-i", "default",
			"-ar", "16000", "-ac", "1", "-t", secs, outPath)
		cmd.Stderr = nil // suppress ffmpeg output
	}

	if cmd == nil {
		return fmt.Errorf("no recording command configured")
	}

	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

// ─── Whisper Backend ────────────────────────────────────────────────

type whisperBackend struct {
	command string // path to whisper command
	model   string // whisper model name (tiny, base, small, medium, large)
}

// detectWhisperBackend finds a whisper executable on PATH.
func detectWhisperBackend() *whisperBackend {
	// Try whisper-cpp first (faster, C++ implementation)
	if path, err := exec.LookPath("whisper-cpp"); err == nil {
		return &whisperBackend{command: path, model: "base"}
	}
	// Try main (whisper.cpp compiled binary often named "main")
	if path, err := exec.LookPath("whisper"); err == nil {
		return &whisperBackend{command: path, model: "base"}
	}
	return nil
}

func (b *whisperBackend) Name() string { return "whisper" }

func (b *whisperBackend) Transcribe(ctx context.Context, audioPath string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, b.command, "--model", b.model, "--file", audioPath, "--output-txt")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("whisper error: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}

	// whisper-cpp outputs to audioPath.txt
	txtPath := audioPath + ".txt"
	if data, err := os.ReadFile(txtPath); err == nil {
		os.Remove(txtPath) // cleanup
		return strings.TrimSpace(string(data)), nil
	}

	// Fallback: check stdout
	text := strings.TrimSpace(stdout.String())
	if text != "" {
		return text, nil
	}

	return "", fmt.Errorf("whisper produced no output")
}

// ─── STT Manager ────────────────────────────────────────────────────

// STTManager manages speech-to-text input.
type STTManager struct {
	backend    sttBackend
	recorder   *audioRecorder
	wakeWord   string        // optional wake word to listen for
	listening  bool          // true when in continuous listening mode
	duration   time.Duration // recording duration per utterance
	mu         sync.Mutex
	stopCh     chan struct{}
	onInput    func(string)  // callback when speech is transcribed
}

// NewSTTManager creates an STT manager, auto-detecting available backends.
func NewSTTManager() *STTManager {
	return &STTManager{
		backend:  detectWhisperBackend(),
		recorder: detectRecorder(),
		duration: 5 * time.Second,
		stopCh:   make(chan struct{}),
	}
}

// Available returns true if both a recorder and STT backend are found.
func (s *STTManager) Available() bool {
	return s.backend != nil && s.recorder != nil
}

// BackendName returns the STT backend name, or "none".
func (s *STTManager) BackendName() string {
	if s.backend == nil {
		return "none"
	}
	return s.backend.Name()
}

// RecorderName returns the audio recorder command, or "none".
func (s *STTManager) RecorderName() string {
	if s.recorder == nil {
		return "none"
	}
	return s.recorder.command
}

// SetDuration sets the recording duration per utterance.
func (s *STTManager) SetDuration(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if d < 1*time.Second {
		d = 1 * time.Second
	}
	if d > 30*time.Second {
		d = 30 * time.Second
	}
	s.duration = d
}

// GetDuration returns the current recording duration.
func (s *STTManager) GetDuration() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.duration
}

// SetWakeWord sets the wake word for continuous listening mode.
func (s *STTManager) SetWakeWord(word string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.wakeWord = strings.ToLower(strings.TrimSpace(word))
}

// GetWakeWord returns the current wake word.
func (s *STTManager) GetWakeWord() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.wakeWord
}

// Listen records a single utterance and returns the transcribed text.
func (s *STTManager) Listen(ctx context.Context) (string, error) {
	if !s.Available() {
		return "", fmt.Errorf("STT not available (backend: %s, recorder: %s)", s.BackendName(), s.RecorderName())
	}

	// Record to temp file
	tmpDir := os.TempDir()
	audioPath := filepath.Join(tmpDir, fmt.Sprintf("yolo_stt_%d.wav", time.Now().UnixNano()))
	defer os.Remove(audioPath)

	s.mu.Lock()
	dur := s.duration
	s.mu.Unlock()

	if err := s.recorder.Record(ctx, dur, audioPath); err != nil {
		return "", fmt.Errorf("recording failed: %w", err)
	}

	// Check if file was created and has content
	info, err := os.Stat(audioPath)
	if err != nil || info.Size() < 100 {
		return "", fmt.Errorf("recording too short or empty")
	}

	text, err := s.backend.Transcribe(ctx, audioPath)
	if err != nil {
		return "", err
	}

	return text, nil
}

// StartContinuousListening begins a background loop that records and
// transcribes repeatedly, calling onInput with each transcription.
// If a wake word is set, only transcriptions containing the wake word
// are forwarded (with the wake word stripped).
func (s *STTManager) StartContinuousListening(onInput func(string)) {
	s.mu.Lock()
	if s.listening {
		s.mu.Unlock()
		return
	}
	s.listening = true
	s.onInput = onInput
	s.stopCh = make(chan struct{})
	s.mu.Unlock()

	go s.listenLoop()
}

// StopContinuousListening stops the background listening loop.
func (s *STTManager) StopContinuousListening() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listening {
		close(s.stopCh)
		s.listening = false
	}
}

// IsListening returns whether continuous listening is active.
func (s *STTManager) IsListening() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.listening
}

func (s *STTManager) listenLoop() {
	for {
		select {
		case <-s.stopCh:
			return
		default:
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		text, err := s.Listen(ctx)
		cancel()

		if err != nil {
			// Brief pause before retrying on error
			time.Sleep(1 * time.Second)
			continue
		}

		if text == "" {
			continue
		}

		s.mu.Lock()
		wakeWord := s.wakeWord
		onInput := s.onInput
		s.mu.Unlock()

		// If wake word is set, check for it
		if wakeWord != "" {
			lower := strings.ToLower(text)
			idx := strings.Index(lower, wakeWord)
			if idx == -1 {
				continue // wake word not found, ignore
			}
			// Strip wake word and any leading/trailing space
			text = strings.TrimSpace(text[idx+len(wakeWord):])
			if text == "" {
				continue
			}
		}

		if onInput != nil {
			onInput(text)
		}
	}
}

// GetStatus returns a human-readable status summary.
func (s *STTManager) GetStatus() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  STT backend: %s\n", s.BackendName()))
	sb.WriteString(fmt.Sprintf("  Recorder: %s\n", s.RecorderName()))
	sb.WriteString(fmt.Sprintf("  Available: %v\n", s.Available()))
	sb.WriteString(fmt.Sprintf("  Duration: %v\n", s.GetDuration()))
	if ww := s.GetWakeWord(); ww != "" {
		sb.WriteString(fmt.Sprintf("  Wake word: %q\n", ww))
	}
	sb.WriteString(fmt.Sprintf("  Listening: %v\n", s.IsListening()))
	return sb.String()
}
