package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPiperBackend_Name(t *testing.T) {
	b := &piperBackend{path: "/usr/bin/piper"}
	assert.Equal(t, "piper", b.Name())
}

func TestPiperBackend_DefaultVoice(t *testing.T) {
	b := &piperBackend{path: "/usr/bin/piper"}
	assert.Equal(t, "en_US-lessac-medium", b.DefaultVoice())
}

func TestPiperBackend_ListVoices_NoModels(t *testing.T) {
	b := &piperBackend{path: "/usr/bin/piper"}
	// ListVoices with non-existent dirs should return empty, not error
	voices := b.ListVoices("")
	assert.Empty(t, voices)
}

func TestDetectAllBackends(t *testing.T) {
	// Just verify it doesn't panic — may return nil or empty on CI
	_ = detectAllBackends()
}

func TestTTSManager_AvailableBackends(t *testing.T) {
	mgr := NewTTSManager()
	backends := mgr.AvailableBackends()
	// Should return a list (may be empty in CI)
	assert.NotNil(t, backends)
}

func TestTTSManager_SetBackend_Invalid(t *testing.T) {
	mgr := NewTTSManager()
	err := mgr.SetBackend("nonexistent_backend_xyz")
	assert.Error(t, err)
}

func TestSayBackend_Name(t *testing.T) {
	b := &sayBackend{path: "/usr/bin/say"}
	assert.Equal(t, "say", b.Name())
	assert.Equal(t, "Samantha", b.DefaultVoice())
}

func TestEspeakBackend_Name(t *testing.T) {
	b := &espeakBackend{path: "/usr/bin/espeak-ng", ng: true}
	assert.Equal(t, "espeak-ng", b.Name())
	assert.Equal(t, "en", b.DefaultVoice())

	b2 := &espeakBackend{path: "/usr/bin/espeak", ng: false}
	assert.Equal(t, "espeak", b2.Name())
}

func TestCleanTextForTTS_CodeBlocks(t *testing.T) {
	input := "Here is some code:\n```go\nfunc main() {}\n```\nDone."
	result := cleanTextForTTS(input)
	assert.NotContains(t, result, "func main")
	assert.Contains(t, result, "Done")
}

func TestCleanTextForTTS_FileDots(t *testing.T) {
	result := cleanTextForTTS("Edit main.go")
	assert.Contains(t, result, "main dot go")
}

func TestCleanTextForTTS_Markdown(t *testing.T) {
	result := cleanTextForTTS("## Header\n**bold** and *italic*")
	assert.Contains(t, result, "bold")
	assert.Contains(t, result, "italic")
	assert.NotContains(t, result, "##")
	assert.NotContains(t, result, "**")
}
