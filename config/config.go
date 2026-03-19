// Package config provides global configuration management for YOLO
package config

import (
	"sync"
)

var (
	configMu    sync.RWMutex
	workingDir  string
	currentModel string
)

func init() {
	// Initialize with defaults
	SetWorkingDir("/Users/sgriepentrog/src/yolo")
	SetCurrentModel("qwen3.5:27b-q4_K_M")
}

// WorkingDir returns the current working directory
func WorkingDir() string {
	configMu.RLock()
	defer configMu.RUnlock()
	return workingDir
}

// SetWorkingDir sets the working directory
func SetWorkingDir(dir string) {
	configMu.Lock()
	defer configMu.Unlock()
	workingDir = dir
}

// CurrentModel returns the current model name
func CurrentModel() string {
	configMu.RLock()
	defer configMu.RUnlock()
	return currentModel
}

// SetCurrentModel sets the current model
func SetCurrentModel(model string) {
	configMu.Lock()
	defer configMu.Unlock()
	currentModel = model
}
