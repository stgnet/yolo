// Tests for yoloconfig.go - YOLO configuration management
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestNewYoloConfig tests the constructor
func TestNewYoloConfig(t *testing.T) {
	config := NewYoloConfig("/test/dir")
	
	if config.yoloDir != "/test/dir" {
		t.Errorf("Expected yoloDir to be '/test/dir', got '%s'", config.yoloDir)
	}
	
	expectedPath := filepath.Join("/test/dir", "config.json")
	if config.configFile != expectedPath {
		t.Errorf("Expected configFile to be '%s', got '%s'", expectedPath, config.configFile)
	}
	
	if config.Data.Version != 1 {
		t.Errorf("Expected default version to be 1, got %d", config.Data.Version)
	}
	
	if config.Data.Model != "" {
		t.Errorf("Expected default model to be empty string, got '%s'", config.Data.Model)
	}
}

// TestYoloConfigLoad_Success tests successful config loading
func TestYoloConfigLoad_Success(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")
	
	// Write a valid config file
	data := YoloConfigData{
		Version:      2,
		Model:        "llama3.2",
		TerminalMode: true,
	}
	jsonData, _ := json.MarshalIndent(data, "", "  ")
	os.WriteFile(configFile, jsonData, 0o644)
	
	config := &YoloConfig{
		yoloDir:    tmpDir,
		configFile: configFile,
		Data:       YoloConfigData{Version: 1}, // Start with different values
	}
	
	success := config.Load()
	if !success {
		t.Error("Expected Load() to return true for valid config")
	}
	
	if config.Data.Version != 2 {
		t.Errorf("Expected version to be 2, got %d", config.Data.Version)
	}
	
	if config.Data.Model != "llama3.2" {
		t.Errorf("Expected model to be 'llama3.2', got '%s'", config.Data.Model)
	}
	
	if !config.Data.TerminalMode {
		t.Error("Expected terminal_mode to be true")
	}
}

// TestYoloConfigLoad_FileNotFound tests loading when file doesn't exist
func TestYoloConfigLoad_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "nonexistent.json")
	
	config := &YoloConfig{
		yoloDir:    tmpDir,
		configFile: configFile,
		Data:       YoloConfigData{Version: 1},
	}
	
	success := config.Load()
	if success {
		t.Error("Expected Load() to return false when file doesn't exist")
	}
	
	// Verify config data was preserved (not reset to defaults)
	if config.Data.Version != 1 {
		t.Errorf("Expected version to remain 1, got %d", config.Data.Version)
	}
}

// TestYoloConfigLoad_InvalidJSON tests loading with invalid JSON
func TestYoloConfigLoad_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")
	
	// Write invalid JSON
	os.WriteFile(configFile, []byte("{invalid json}"), 0o644)
	
	config := &YoloConfig{
		yoloDir:    tmpDir,
		configFile: configFile,
		Data: YoloConfigData{
			Version:      5,
			Model:        "old-model",
			TerminalMode: false,
		},
	}
	
	success := config.Load()
	if success {
		t.Error("Expected Load() to return false for invalid JSON")
	}
	
	// Verify config was reset to defaults on error
	if config.Data.Version != 1 {
		t.Errorf("Expected version to be reset to 1, got %d", config.Data.Version)
	}
	
	if config.Data.Model != "" {
		t.Errorf("Expected model to be empty after invalid JSON, got '%s'", config.Data.Model)
	}
}

// TestYoloConfigLoad_ConcurrentSafety tests concurrent access safety
func TestYoloConfigLoad_ConcurrentSafety(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")
	
	data := YoloConfigData{Version: 1, Model: "test"}
	jsonData, _ := json.MarshalIndent(data, "", "  ")
	os.WriteFile(configFile, jsonData, 0o644)
	
	config := &YoloConfig{
		yoloDir:    tmpDir,
		configFile: configFile,
		Data:       YoloConfigData{Version: 1},
	}
	
	// Run multiple goroutines trying to load simultaneously
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			config.Load()
		}()
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestYoloConfigSave_Success tests successful config save
func TestYoloConfigSave_Success(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")
	
	config := &YoloConfig{
		yoloDir:    tmpDir,
		configFile: configFile,
		Data: YoloConfigData{
			Version:      2,
			Model:        "gpt-4",
			TerminalMode: true,
		},
	}
	
	err := config.Save()
	if err != nil {
		t.Fatalf("Expected Save() to succeed, got error: %v", err)
	}
	
	// Verify file was created
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Error("Expected config file to be created")
	}
	
	// Read and verify contents
	data, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("Failed to read saved config: %v", err)
	}
	
	var loadedData YoloConfigData
	if err := json.Unmarshal(data, &loadedData); err != nil {
		t.Fatalf("Failed to parse saved JSON: %v", err)
	}
	
	if loadedData.Version != 2 {
		t.Errorf("Expected version to be 2 in file, got %d", loadedData.Version)
	}
	
	if loadedData.Model != "gpt-4" {
		t.Errorf("Expected model to be 'gpt-4' in file, got '%s'", loadedData.Model)
	}
	
	if !loadedData.TerminalMode {
		t.Error("Expected terminal_mode to be true in file")
	}
}

// TestYoloConfigSave_AtomicWrite tests atomic file writing
func TestYoloConfigSave_AtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")
	
	config := &YoloConfig{
		yoloDir:    tmpDir,
		configFile: configFile,
		Data:       YoloConfigData{Version: 1},
	}
	
	err := config.Save()
	if err != nil {
		t.Fatalf("Expected Save() to succeed, got error: %v", err)
	}
	
	// Verify no temp file left behind
	tempFile := configFile + ".tmp"
	if _, err := os.Stat(tempFile); !os.IsNotExist(err) {
		t.Error("Expected no .tmp file after successful save")
	}
}

// TestYoloConfigSave_UnreadableDir tests saving to unreadable directory
func TestYoloConfigSave_UnreadableDir(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")
	
	config := &YoloConfig{
		yoloDir:    tmpDir,
		configFile: configFile,
		Data:       YoloConfigData{Version: 1},
	}
	
	// Remove read permissions from directory
	os.Chmod(tmpDir, 0o000)
	defer os.Chmod(tmpDir, 0o755) // Restore for cleanup
	
	err := config.Save()
	if err == nil {
		t.Error("Expected Save() to fail when directory is unreadable")
	}
}

// TestYoloConfigSave_ConcurrentSafety tests concurrent save safety
func TestYoloConfigSave_ConcurrentSafety(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")
	
	config := &YoloConfig{
		yoloDir:    tmpDir,
		configFile: configFile,
		Data:       YoloConfigData{Version: 1},
	}
	
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(i int) {
			defer func() { done <- true }()
			config.Data.Version = i + 2
			config.Save()
		}(i)
	}
	
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestYoloConfigGetModel tests GetModel function
func TestYoloConfigGetModel(t *testing.T) {
	config := &YoloConfig{
		Data: YoloConfigData{Model: "test-model"},
	}
	
	model := config.GetModel()
	if model != "test-model" {
		t.Errorf("Expected 'test-model', got '%s'", model)
	}
}

// TestYoloConfigSetModel tests SetModel function
func TestYoloConfigSetModel(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")
	
	data := YoloConfigData{Version: 1}
	jsonData, _ := json.MarshalIndent(data, "", "  ")
	os.WriteFile(configFile, jsonData, 0o644)
	
	config := &YoloConfig{
		yoloDir:    tmpDir,
		configFile: configFile,
		Data:       YoloConfigData{Model: "old-model"},
	}
	
	config.SetModel("new-model")
	
	// Verify model was updated
	if config.Data.Model != "new-model" {
		t.Errorf("Expected model to be 'new-model', got '%s'", config.Data.Model)
	}
	
	// Verify save happened
	data2 := YoloConfigData{}
	jsonFile, _ := os.ReadFile(configFile)
	json.Unmarshal(jsonFile, &data2)
	
	if data2.Model != "new-model" {
		t.Errorf("Expected saved model to be 'new-model', got '%s'", data2.Model)
	}
}

// TestYoloConfigGetTerminalMode tests GetTerminalMode function
func TestYoloConfigGetTerminalMode(t *testing.T) {
	tests := []struct {
		name     string
		terminal bool
		expected bool
	}{
		{"enabled", true, true},
		{"disabled", false, false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &YoloConfig{
				Data: YoloConfigData{TerminalMode: tt.terminal},
			}
			
			result := config.GetTerminalMode()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestYoloConfigSetTerminalMode tests SetTerminalMode function
func TestYoloConfigSetTerminalMode(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")
	
	data := YoloConfigData{Version: 1, TerminalMode: false}
	jsonData, _ := json.MarshalIndent(data, "", "  ")
	os.WriteFile(configFile, jsonData, 0o644)
	
	config := &YoloConfig{
		yoloDir:    tmpDir,
		configFile: configFile,
		Data:       YoloConfigData{TerminalMode: false},
	}
	
	config.SetTerminalMode(true)
	
	if !config.Data.TerminalMode {
		t.Error("Expected TerminalMode to be true after SetTerminalMode")
	}
	
	// Verify save happened
	data2 := YoloConfigData{}
	jsonFile, _ := os.ReadFile(configFile)
	json.Unmarshal(jsonFile, &data2)
	
	if !data2.TerminalMode {
		t.Error("Expected saved TerminalMode to be true")
	}
}