package utils

import (
	"os"
	"path/filepath"
	"testing"
)

// Additional tests to improve coverage for error paths

func TestEnsureDir_NonExistentParent(t *testing.T) {
	tmpdir := t.TempDir()

	// Create a file where we want to create a directory
	filePath := filepath.Join(tmpdir, "file_as_dir")
	if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Try to create a directory with the same name as existing file
	err := EnsureDir(filePath)
	if err == nil {
		t.Fatal("Expected error when trying to create directory at existing file path")
	}
}

func TestCopyFile_SourceIsDirectory(t *testing.T) {
	tmpdir := t.TempDir()
	srcDir := filepath.Join(tmpdir, "source_dir")
	dstFile := filepath.Join(tmpdir, "dest.txt")

	if err := os.Mkdir(srcDir, 0755); err != nil {
		t.Fatal(err)
	}

	err := CopyFile(srcDir, dstFile)
	if err == nil {
		t.Error("Expected error when trying to copy directory as file")
	}

	// Destination should not be created
	if FileExists(dstFile) {
		t.Error("Destination should not exist on source directory error")
	}
}

func TestWriteFile_ErrorPath(t *testing.T) {
	// Try to write to a path where parent cannot be created
	// This tests the MkdirAll error path in WriteFile
	tmpdir := t.TempDir()

	// Create a file where we want to create a subdirectory
	blockerPath := filepath.Join(tmpdir, "blocker")
	if err := os.WriteFile(blockerPath, []byte("block"), 0644); err != nil {
		t.Fatal(err)
	}

	// Try to write to a path under the blocker file (should fail on MkdirAll)
	targetPath := filepath.Join(blockerPath, "nested", "file.txt")
	err := WriteFile(targetPath, []byte("test"), 0644)
	if err == nil {
		t.Error("Expected error when parent path is a file")
	}
}

func TestGetFileSize_Directory(t *testing.T) {
	tmpdir := t.TempDir()

	// GetFileSize actually works for directories (returns their size from stat)
	// Test that it returns a valid size
	size, err := GetFileSize(tmpdir)
	if err != nil {
		t.Fatalf("GetFileSize should work on directories: %v", err)
	}

	if size < 0 {
		t.Errorf("Expected non-negative size, got %d", size)
	}
}

func TestGetFileModTime_Directory(t *testing.T) {
	tmpdir := t.TempDir()

	// GetFileModTime actually works for directories (returns their modtime from stat)
	// Test that it returns a valid time
	modTime, err := GetFileModTime(tmpdir)
	if err != nil {
		t.Fatalf("GetFileModTime should work on directories: %v", err)
	}

	if modTime.IsZero() {
		t.Error("Expected non-zero time for existing directory")
	}
}

func TestReadDir_FileInsteadOfDir(t *testing.T) {
	tmpdir := t.TempDir()
	testFile := filepath.Join(tmpdir, "file.txt")

	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	entries, err := ReadDir(testFile)
	if err == nil {
		t.Error("Expected error when reading file as directory")
	}

	if entries != nil {
		t.Errorf("Expected nil entries on error, got %d", len(entries))
	}
}

func TestMoveFile_DestinationExists(t *testing.T) {
	tmpdir := t.TempDir()
	src := filepath.Join(tmpdir, "source.txt")
	dst := filepath.Join(tmpdir, "dest.txt")

	// Create both files
	os.WriteFile(src, []byte("source"), 0644)
	os.WriteFile(dst, []byte("dest"), 0644)

	err := MoveFile(src, dst)
	if err != nil {
		t.Fatalf("Move should succeed even when dest exists: %v", err)
	}

	if FileExists(src) {
		t.Error("Source should be deleted")
	}
	if !FileExists(dst) {
		t.Fatal("Destination should exist")
	}

	// Destination should have source content (overwritten)
	content, _ := ReadFileString(dst)
	if content != "source" {
		t.Errorf("Expected 'source' content, got %q", content)
	}
}

func TestListFiles_Subdirectory(t *testing.T) {
	tmpdir := t.TempDir()

	// Create nested structure
	subdir1 := filepath.Join(tmpdir, "sub1")
	if err := os.Mkdir(subdir1, 0755); err != nil {
		t.Fatal(err)
	}

	subdir2 := filepath.Join(tmpdir, "sub2")
	if err := os.Mkdir(subdir2, 0755); err != nil {
		t.Fatal(err)
	}

	// Create files in subdirectories
	if err := os.WriteFile(filepath.Join(subdir1, "file.txt"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	// ListFiles should only list immediate children, not recurse
	files, err := ListFiles(tmpdir)
	if err != nil {
		t.Fatalf("ListFiles failed: %v", err)
	}

	// Should only return the two subdirectories (which are directories, so filtered out)
	if len(files) != 0 {
		t.Errorf("Expected 0 files in directory with only subdirs, got %d: %v", len(files), files)
	}
}
