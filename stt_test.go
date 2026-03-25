package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSTTManager_NewDefault(t *testing.T) {
	mgr := NewSTTManager()
	// Should not panic, backends may or may not be available
	assert.NotNil(t, mgr)
	assert.Equal(t, 5*time.Second, mgr.GetDuration())
	assert.Equal(t, "", mgr.GetWakeWord())
	assert.False(t, mgr.IsListening())
}

func TestSTTManager_SetDuration(t *testing.T) {
	mgr := NewSTTManager()

	mgr.SetDuration(10 * time.Second)
	assert.Equal(t, 10*time.Second, mgr.GetDuration())

	// Clamp to minimum
	mgr.SetDuration(100 * time.Millisecond)
	assert.Equal(t, 1*time.Second, mgr.GetDuration())

	// Clamp to maximum
	mgr.SetDuration(60 * time.Second)
	assert.Equal(t, 30*time.Second, mgr.GetDuration())
}

func TestSTTManager_WakeWord(t *testing.T) {
	mgr := NewSTTManager()

	mgr.SetWakeWord("YOLO")
	assert.Equal(t, "yolo", mgr.GetWakeWord())

	mgr.SetWakeWord("  Hey Computer  ")
	assert.Equal(t, "hey computer", mgr.GetWakeWord())

	mgr.SetWakeWord("")
	assert.Equal(t, "", mgr.GetWakeWord())
}

func TestSTTManager_BackendName(t *testing.T) {
	mgr := &STTManager{}
	assert.Equal(t, "none", mgr.BackendName())

	mgr.backend = &whisperBackend{command: "/usr/bin/whisper", model: "base"}
	assert.Equal(t, "whisper", mgr.BackendName())
}

func TestSTTManager_RecorderName(t *testing.T) {
	mgr := &STTManager{}
	assert.Equal(t, "none", mgr.RecorderName())

	mgr.recorder = &audioRecorder{command: "arecord"}
	assert.Equal(t, "arecord", mgr.RecorderName())
}

func TestSTTManager_Available(t *testing.T) {
	mgr := &STTManager{}
	assert.False(t, mgr.Available())

	mgr.backend = &whisperBackend{command: "whisper"}
	assert.False(t, mgr.Available()) // still need recorder

	mgr.recorder = &audioRecorder{command: "rec"}
	assert.True(t, mgr.Available())
}

func TestSTTManager_GetStatus(t *testing.T) {
	mgr := NewSTTManager()
	status := mgr.GetStatus()
	assert.Contains(t, status, "STT backend:")
	assert.Contains(t, status, "Recorder:")
	assert.Contains(t, status, "Available:")
	assert.Contains(t, status, "Duration:")
	assert.Contains(t, status, "Listening: false")
}

func TestSTTManager_GetStatus_WithWakeWord(t *testing.T) {
	mgr := NewSTTManager()
	mgr.SetWakeWord("hey yolo")
	status := mgr.GetStatus()
	assert.Contains(t, status, "Wake word:")
	assert.Contains(t, status, "hey yolo")
}

func TestSTTManager_ContinuousListening_StartStop(t *testing.T) {
	mgr := &STTManager{
		stopCh: make(chan struct{}),
		// No backend/recorder — listenLoop will error and retry,
		// but we stop it quickly
	}

	assert.False(t, mgr.IsListening())

	// StartContinuousListening without backend does nothing harmful
	// but sets listening=true
	mgr.backend = &whisperBackend{command: "nonexistent"}
	mgr.recorder = &audioRecorder{command: "nonexistent"}
	mgr.StartContinuousListening(func(text string) {})
	assert.True(t, mgr.IsListening())

	// Double start should be a no-op
	mgr.StartContinuousListening(func(text string) {})
	assert.True(t, mgr.IsListening())

	mgr.StopContinuousListening()
	assert.False(t, mgr.IsListening())

	// Double stop should not panic
	mgr.StopContinuousListening()
}

func TestWhisperBackend_Name(t *testing.T) {
	b := &whisperBackend{command: "/usr/bin/whisper", model: "base"}
	assert.Equal(t, "whisper", b.Name())
}
