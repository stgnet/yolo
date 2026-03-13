package utils

import (
	"os"
	"path/filepath"
	"testing"
)

// Test safety config through public API that uses it

func TestDeleteFileWithConfig_NoBackup(t *testing.T) {
	tmpdir := t.TempDir()
	testFile := filepath.Join(tmpdir, "test.txt")
	
	// Create test file
	content := []byte("content to delete without backup")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Config with backups disabled
	config := &SafetyConfig{
		CreateBackup: false,
	}

	err := DeleteFileWithConfig(testFile, config)
	if err != nil {
		t.Fatalf("DeleteFileWithConfig failed: %v", err)
	}

	// Verify file is deleted
	if FileExists(testFile) {
		t.Error("File should be deleted")
	}

	// Verify no backup was created
	backupPath := testFile + ".bak"
	if FileExists(backupPath) {
		t.Error("Backup should not exist when CreateBackup is false")
	}
}

func TestDeleteFileWithConfig_WithBackup(t *testing.T) {
	tmpdir := t.TempDir()
	testFile := filepath.Join(tmpdir, "test.txt")
	
	// Create test file
	content := []byte("content to delete with backup")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Config with backups enabled
	config := &SafetyConfig{
		CreateBackup: true,
	}

	err := DeleteFileWithConfig(testFile, config)
	if err != nil {
		t.Fatalf("DeleteFileWithConfig failed: %v", err)
	}

	// Verify file is deleted
	if FileExists(testFile) {
		t.Error("File should be deleted")
	}

	// Verify backup was created
	backupPath := testFile + ".bak"
	if !FileExists(backupPath) {
		t.Error("Backup should exist when CreateBackup is true")
	}

	// Verify backup content
	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("Failed to read backup: %v", err)
	}
	if string(backupContent) != string(content) {
		t.Errorf("Backup content mismatch")
	}
}

func TestMoveFileWithConfig_NoBackup(t *testing.T) {
	tmpdir := t.TempDir()
	src := filepath.Join(tmpdir, "source.txt")
	dst := filepath.Join(tmpdir, "dest.txt")
	
	if err := os.WriteFile(src, []byte("move content"), 0644); err != nil {
		t.Fatal(err)
	}

	config := &SafetyConfig{CreateBackup: false}
	err := MoveFileWithConfig(src, dst, config)
	if err != nil {
		t.Fatalf("MoveFileWithConfig failed: %v", err)
	}

	if FileExists(src) {
		t.Error("Source should be deleted")
	}
	if !FileExists(dst) {
		t.Error("Destination should exist")
	}
	if FileExists(src+".bak") {
		t.Error("Backup should not exist when CreateBackup is false")
	}
}

func TestMoveFileWithConfig_WithBackup(t *testing.T) {
	tmpdir := t.TempDir()
	src := filepath.Join(tmpdir, "source.txt")
	dst := filepath.Join(tmpdir, "dest.txt")
	content := []byte("move content with backup")
	
	if err := os.WriteFile(src, content, 0644); err != nil {
		t.Fatal(err)
	}

	config := &SafetyConfig{CreateBackup: true}
	err := MoveFileWithConfig(src, dst, config)
	if err != nil {
		t.Fatalf("MoveFileWithConfig failed: %v", err)
	}

	if FileExists(src) {
		t.Error("Source should be deleted")
	}
	if !FileExists(dst) {
		t.Error("Destination should exist")
	}
	
	// Backup is created for the source before deletion
	if !FileExists(src+".bak") {
		t.Error("Backup should exist when CreateBackup is true")
	}
	
	backupContent, _ := os.ReadFile(src + ".bak")
	if string(backupContent) != string(content) {
		t.Errorf("Backup content mismatch")
	}
}

func TestSafetyConfigSizeLimit(t *testing.T) {
	tmpdir := t.TempDir()
	testFile := filepath.Join(tmpdir, "large.txt")
	
	// Create a file larger than the limit
	largeContent := make([]byte, 2048) // 2KB file
	if err := os.WriteFile(testFile, largeContent, 0644); err != nil {
		t.Fatal(err)
	}

	// Config with small size limit
	config := &SafetyConfig{
		EnableSizeCheck: true,
		MaxDeleteSize:   1024, // 1KB limit - file is 2KB
	}

	err := DeleteFileWithConfig(testFile, config)
	if err == nil {
		t.Fatal("Expected error when deleting file larger than MaxDeleteSize")
	}

	// Verify file still exists
	if !FileExists(testFile) {
		t.Error("File should not be deleted due to size limit")
	}
}

func TestSafetyConfigSizeLimit_Allowed(t *testing.T) {
	tmpdir := t.TempDir()
	testFile := filepath.Join(tmpdir, "small.txt")
	
	// Create a small file
	smallContent := []byte("small content")
	if err := os.WriteFile(testFile, smallContent, 0644); err != nil {
		t.Fatal(err)
	}

	// Config with generous size limit
	config := &SafetyConfig{
		EnableSizeCheck: true,
		MaxDeleteSize:   1024 * 1024, // 1MB limit
	}

	err := DeleteFileWithConfig(testFile, config)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if FileExists(testFile) {
		t.Error("Small file should be deleted")
	}
}

func TestSafetyConfigDisabledSizeCheck(t *testing.T) {
	tmpdir := t.TempDir()
	testFile := filepath.Join(tmpdir, "any.txt")
	
	largeContent := make([]byte, 2048) // 2KB file
	if err := os.WriteFile(testFile, largeContent, 0644); err != nil {
		t.Fatal(err)
	}

	// Config with size check disabled
	config := &SafetyConfig{
		EnableSizeCheck: false,
		MaxDeleteSize:   1024, // Would block if enabled
	}

	err := DeleteFileWithConfig(testFile, config)
	if err != nil {
		t.Fatalf("Unexpected error when size check disabled: %v", err)
	}

	if FileExists(testFile) {
		t.Error("File should be deleted when size check is disabled")
	}
}

func TestProtectedPathBlocking(t *testing.T) {
	tmpdir := t.TempDir()
	
	// Create a file with a protected path prefix simulation
	protectedSubdir := filepath.Join(tmpdir, "simulated_protected")
	testFile := filepath.Join(protectedSubdir, "file.txt")
	
	if err := os.MkdirAll(protectedSubdir, 0755); err != nil {
		t.Fatal(err)
	}
	
	if err := os.WriteFile(testFile, []byte("protected content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Config with this path protected
	config := &SafetyConfig{
		ProtectedPaths:   []string{protectedSubdir},
		CreateBackup:     false,
		EnableSizeCheck:  false,
	}

	err := DeleteFileWithConfig(testFile, config)
	if err == nil {
		t.Fatal("Expected error when deleting file in protected path")
	}

	// Verify file still exists
	if !FileExists(testFile) {
		t.Error("Protected file should not be deleted")
	}
}
