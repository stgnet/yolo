// Package tools provides additional file operation tools
package tools

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
	
	"yolo/config"
)

// CopyFileTool implements file copying functionality
type CopyFileTool struct{}

func (t *CopyFileTool) Name() string { return "copy_file" }
func (t *CopyFileTool) Description() string { return "Copy a file from source to destination. Creates destination directory if needed." }
func (t *CopyFileTool) Type() ToolType { return ToolTypeFile }

func (t *CopyFileTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	start := time.Now()
	
	source, ok := args["source"].(string)
	if !ok || source == "" {
		return &ToolResult{Success: false, Error: "source is required", Duration: time.Since(start)}, nil
	}
	
	dest, ok := args["dest"].(string)
	if !ok || dest == "" {
		return &ToolResult{Success: false, Error: "dest is required", Duration: time.Since(start)}, nil
	}
	
	err := copyFile(source, dest)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error(), Duration: time.Since(start)}, nil
	}
	
	return &ToolResult{
		Success:  true,
		Output:   fmt.Sprintf("Copied %s to %s", source, dest),
		Duration: time.Since(start),
	}, nil
}

// MoveFileTool implements file moving functionality
type MoveFileTool struct{}

func (t *MoveFileTool) Name() string { return "move_file" }
func (t *MoveFileTool) Description() string { return "Move a file from source to destination. Creates destination directory if needed." }
func (t *MoveFileTool) Type() ToolType { return ToolTypeFile }

func (t *MoveFileTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	start := time.Now()
	
	source, ok := args["source"].(string)
	if !ok || source == "" {
		return &ToolResult{Success: false, Error: "source is required", Duration: time.Since(start)}, nil
	}
	
	dest, ok := args["dest"].(string)
	if !ok || dest == "" {
		return &ToolResult{Success: false, Error: "dest is required", Duration: time.Since(start)}, nil
	}
	
	err := moveFile(source, dest)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error(), Duration: time.Since(start)}, nil
	}
	
	return &ToolResult{
		Success:  true,
		Output:   fmt.Sprintf("Moved %s to %s", source, dest),
		Duration: time.Since(start),
	}, nil
}

// Helper functions for copy/move operations

func copyFile(source, dest string) error {
	sourcePath := filepath.Join(config.GetYoloDir(), source)
	destPath := filepath.Join(config.GetYoloDir(), dest)
	
	// Create destination directory if needed
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}
	
	// Read source file
	srcFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()
	
	// Create destination file
	dstFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()
	
	// Copy content
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		os.Remove(destPath) // Clean up on error
		return fmt.Errorf("failed to copy file content: %w", err)
	}
	
	return nil
}

func moveFile(source, dest string) error {
	sourcePath := filepath.Join(config.GetYoloDir(), source)
	destPath := filepath.Join(config.GetYoloDir(), dest)
	
	// Copy first
	if err := copyFile(source, dest); err != nil {
		return err
	}
	
	// Then remove original
	if err := os.Remove(sourcePath); err != nil {
		// Try to clean up the copy if removal fails
		os.Remove(destPath)
		return fmt.Errorf("failed to remove source file: %w", err)
	}
	
	return nil
}
