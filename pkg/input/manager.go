// Package input provides async terminal input handling for the YOLO agent.
package input

import (
	"bufio"
	"os"
	"sync"
	"time"

	"golang.org/x/term"
)

// InputManager handles asynchronous user input from the terminal.
// It reads lines and queues them as input events with timestamps.
type InputManager struct {
	mu          sync.Mutex
	reader      *bufio.Reader
	stopChan    chan struct{}
	eventsChan  chan InputEvent
	delay       time.Duration
	lastInput   time.Time
	inputBuffer []byte // Buffer for accumulating input during delay
	enabled     bool
}

// InputEvent represents a user input event.
type InputEvent struct {
	Timestamp time.Time
	Text      string
}

// InputConfig holds configuration for InputManager.
type InputConfig struct {
	Delay      time.Duration
	BufferSize int // Size of input events channel buffer
}

// DefaultInputConfig returns default configuration.
func DefaultInputConfig() InputConfig {
	return InputConfig{
		Delay:      10 * time.Second,
		BufferSize: 100,
	}
}

// NewInputManager creates a new input manager.
func NewInputManager(cfg InputConfig) *InputManager {
	im := &InputManager{
		reader:      bufio.NewReader(os.Stdin),
		stopChan:    make(chan struct{}),
		eventsChan:  make(chan InputEvent, cfg.BufferSize),
		delay:       cfg.Delay,
		inputBuffer: make([]byte, 0),
	}

	// Start background input processor
	go im.processInput()
	return im
}

// Enable enables the input manager.
func (im *InputManager) Enable() {
	im.mu.Lock()
	defer im.mu.Unlock()
	im.enabled = true
}

// Disable disables the input manager.
func (im *InputManager) Disable() {
	im.mu.Lock()
	defer im.mu.Unlock()
	im.enabled = false
}

// processInput reads from stdin and queues input events.
func (im *InputManager) processInput() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-im.stopChan:
			return
		case <-ticker.C:
			im.checkInput()
		}
	}
}

// checkInput checks if there's new input and processes it.
func (im *InputManager) checkInput() {
	im.mu.Lock()
	if !im.enabled {
		im.mu.Unlock()
		return
	}

	now := time.Now()

	// If we have buffered input and delay has passed, send it
	if len(im.inputBuffer) > 0 {
		if now.Sub(im.lastInput) >= im.delay {
			text := string(im.inputBuffer)
			im.inputBuffer = im.inputBuffer[:0]

			select {
			case im.eventsChan <- InputEvent{
				Timestamp: now,
				Text:      text,
			}:
			default:
				// Channel full, drop this event (buffer overflow protection)
			}
			im.mu.Unlock()
			return
		}
	}

	// If input ready and no buffering in progress, start buffering
	if term.IsTerminal(int(os.Stdin.Fd())) {
		im.lastInput = now
	}
	im.mu.Unlock()
}

// Read reads the next available input event.
func (im *InputManager) Read() (InputEvent, bool) {
	select {
	case event := <-im.eventsChan:
		return event, true
	default:
		return InputEvent{}, false
	}
}

// TryReadNonBlocking attempts to read without blocking.
// Returns false if no input is available.
func (im *InputManager) TryReadNonBlocking() (string, bool) {
	select {
	case event := <-im.eventsChan:
		return event.Text, true
	default:
		return "", false
	}
}

// GetEvents returns all pending events (up to the buffer size).
func (im *InputManager) GetEvents() []InputEvent {
	events := make([]InputEvent, 0, 100)

	for {
		select {
		case event := <-im.eventsChan:
			events = append(events, event)
		default:
			return events
		}
	}
}

// Close stops the input manager.
func (im *InputManager) Close() {
	close(im.stopChan)
}

// IsEnabled returns whether the input manager is enabled.
func (im *InputManager) IsEnabled() bool {
	im.mu.Lock()
	defer im.mu.Unlock()
	return im.enabled
}

// SetDelay updates the input delay.
func (im *InputManager) SetDelay(d time.Duration) {
	im.mu.Lock()
	defer im.mu.Unlock()
	im.delay = d
}
