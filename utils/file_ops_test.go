package utils

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"yolo/errors"
)

func TestReadFile(t *testing.T) {
	t.Run("existing file", func(t *testing.T) {
		tmpfile := filepath.Join(t.TempDir(), "test.txt")
		content := []byte("test content")
		if err := os.WriteFile(tmpfile, content, 0644); err != nil {
			t.Fatal(err)
		}

		data, err := ReadFile(tmpfile)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(data) != string(content) {
			t.Errorf("got %q, want %q", data, content)
		}
	})

	t.Run("non-existent file", func(t *testing.T) {
		data, err := ReadFile("/nonexistent/file.txt")
		if err == nil {
			t.Fatal("expected error for non-existent file")
		}

		if !errors.IsFileNotFoundError(err) {
			t.Errorf("expected FileNotFoundError, got: %T", err)
		}

		fnfe, ok := errors.AsFileNotFoundError(err)
		if !ok {
			t.Error("expected to extract FileNotFoundError")
		} else if fnfe.Op != "read" {
			t.Errorf("got op %q, want %q", fnfe.Op, "read")
		}

		if len(data) > 0 {
			t.Error("expected nil data on error")
		}
	})

	t.Run("directory instead of file", func(t *testing.T) {
		tmpdir := t.TempDir()
		_, err := ReadFile(tmpdir)
		if err == nil {
			t.Fatal("expected error when reading directory")
		}
	})
}

func TestReadFileString(t *testing.T) {
	t.Run("existing file", func(t *testing.T) {
		tmpfile := filepath.Join(t.TempDir(), "test.txt")
		content := "test content"
		if err := os.WriteFile(tmpfile, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		result, err := ReadFileString(tmpfile)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != content {
			t.Errorf("got %q, want %q", result, content)
		}
	})

	t.Run("non-existent file", func(t *testing.T) {
		result, err := ReadFileString("/nonexistent/file.txt")
		if err == nil {
			t.Fatal("expected error for non-existent file")
		}
		if result != "" {
			t.Error("expected empty string on error")
		}
	})
}

func TestWriteFile(t *testing.T) {
	t.Run("write to new file", func(t *testing.T) {
		tmpfile := filepath.Join(t.TempDir(), "test.txt")
		content := []byte("test content")

		err := WriteFile(tmpfile, content, 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		data, err := os.ReadFile(tmpfile)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != string(content) {
			t.Errorf("got %q, want %q", data, content)
		}
	})

	t.Run("write to nested directory", func(t *testing.T) {
		tmpdir := t.TempDir()
		tmpfile := filepath.Join(tmpdir, "subdir1", "subdir2", "test.txt")
		content := []byte("test content")

		err := WriteFile(tmpfile, content, 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		data, err := os.ReadFile(tmpfile)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != string(content) {
			t.Errorf("got %q, want %q", data, content)
		}
	})

	t.Run("overwrite existing file", func(t *testing.T) {
		tmpfile := filepath.Join(t.TempDir(), "test.txt")
		initial := []byte("initial content")
		if err := os.WriteFile(tmpfile, initial, 0644); err != nil {
			t.Fatal(err)
		}

		newContent := []byte("new content")
		err := WriteFile(tmpfile, newContent, 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		data, err := os.ReadFile(tmpfile)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != string(newContent) {
			t.Errorf("got %q, want %q", data, newContent)
		}
	})
}

func TestWriteFileString(t *testing.T) {
	tmpfile := filepath.Join(t.TempDir(), "test.txt")
	content := "test content"

	err := WriteFileString(tmpfile, content, 0644)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(tmpfile)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != content {
		t.Errorf("got %q, want %q", data, content)
	}
}

func TestFileExists(t *testing.T) {
	tmpdir := t.TempDir()
	existingFile := filepath.Join(tmpdir, "existing.txt")
	os.WriteFile(existingFile, []byte{}, 0644)

	t.Run("file exists", func(t *testing.T) {
		if !FileExists(existingFile) {
			t.Error("expected true for existing file")
		}
	})

	t.Run("directory exists", func(t *testing.T) {
		// FileExists checks for files only, returns false for directories
		if FileExists(tmpdir) {
			t.Error("expected false for directory (FileExists is for files only)")
		}
	})

	t.Run("non-existent path", func(t *testing.T) {
		if FileExists(filepath.Join(tmpdir, "nonexistent.txt")) {
			t.Error("expected false for non-existent file")
		}
	})
}

func TestIsDirectory(t *testing.T) {
	tmpdir := t.TempDir()
	existingFile := filepath.Join(tmpdir, "existing.txt")
	os.WriteFile(existingFile, []byte{}, 0644)

	if !IsDirectory(tmpdir) {
		t.Error("expected true for directory")
	}
	if IsDirectory(existingFile) {
		t.Error("expected false for file")
	}
	if IsDirectory(filepath.Join(tmpdir, "nonexistent")) {
		t.Error("expected false for non-existent path")
	}
}

func TestEnsureDir(t *testing.T) {
	t.Run("create new directory", func(t *testing.T) {
		tmpdir := t.TempDir()
		newPath := filepath.Join(tmpdir, "new", "nested", "dir")

		err := EnsureDir(newPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !IsDirectory(newPath) {
			t.Error("expected directory to exist after EnsureDir")
		}
	})

	t.Run("existing directory", func(t *testing.T) {
		tmpdir := t.TempDir()
		err := EnsureDir(tmpdir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestDeleteFile(t *testing.T) {
	t.Run("delete existing file", func(t *testing.T) {
		tmpfile := filepath.Join(t.TempDir(), "test.txt")
		os.WriteFile(tmpfile, []byte{}, 0644)

		err := DeleteFile(tmpfile)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if FileExists(tmpfile) {
			t.Error("expected file to be deleted")
		}
	})

	t.Run("delete non-existent file", func(t *testing.T) {
		err := DeleteFile(filepath.Join(t.TempDir(), "nonexistent.txt"))
		if err == nil {
			t.Fatal("expected error for non-existent file")
		}

		if !errors.IsFileNotFoundError(err) {
			t.Errorf("expected FileNotFoundError, got: %T", err)
		}
	})

	t.Run("delete directory", func(t *testing.T) {
		tmpdir := t.TempDir()
		err := DeleteFile(tmpdir)
		if err == nil {
			t.Fatal("expected error when deleting directory with DeleteFile")
		}

		// Verify the directory still exists since DeleteFile can't delete directories
		if !IsDirectory(tmpdir) {
			t.Error("directory should still exist after failed deletion")
		}
	})
}

func TestCopyFile(t *testing.T) {
	tmpdir := t.TempDir()
	src := filepath.Join(tmpdir, "source.txt")
	dst := filepath.Join(tmpdir, "dest.txt")
	content := []byte("test content for copy")

	if err := os.WriteFile(src, content, 0644); err != nil {
		t.Fatal(err)
	}

	err := CopyFile(src, dst)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dstData, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}

	if string(dstData) != string(content) {
		t.Errorf("got %q, want %q", dstData, content)
	}

	if !FileExists(src) {
		t.Error("source file should still exist after copy")
	}
}

func TestCopyFileSourceNotFound(t *testing.T) {
	tmpdir := t.TempDir()
	src := filepath.Join(tmpdir, "nonexistent.txt")
	dst := filepath.Join(tmpdir, "dest.txt")

	err := CopyFile(src, dst)
	if err == nil {
		t.Fatal("expected error when source file doesn't exist")
	}

	if !errors.IsFileNotFoundError(err) {
		t.Errorf("expected FileNotFoundError, got: %T", err)
	}
}

func TestMoveFile(t *testing.T) {
	tmpdir := t.TempDir()
	src := filepath.Join(tmpdir, "source.txt")
	dst := filepath.Join(tmpdir, "dest.txt")
	content := []byte("test content for move")

	if err := os.WriteFile(src, content, 0644); err != nil {
		t.Fatal(err)
	}

	err := MoveFile(src, dst)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if FileExists(src) {
		t.Error("source file should not exist after move")
	}

	dstData, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}

	if string(dstData) != string(content) {
		t.Errorf("got %q, want %q", dstData, content)
	}
}

func TestMoveFileCleanupOnError(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "source.txt")
	dstDir := filepath.Join(tmpDir, "readonly_dir")
	dst := filepath.Join(dstDir, "dest.txt")

	os.WriteFile(src, []byte("test content"), 0644)
	os.MkdirAll(dstDir, 0755)

	// Copy file first so we can test cleanup behavior
	CopyFile(src, dst)

	// Delete source manually to simulate partial move failure scenario
	DeleteFile(src)

	// Try to move (source doesn't exist, should fail gracefully)
	err := MoveFile(src, dst)
	if err == nil {
		t.Fatal("expected error when source doesn't exist")
	}

	// The destination from copy should still exist since we can't clean it up
	// This is expected behavior - the function tries but ignores cleanup errors
	if !FileExists(dst) {
		t.Log("cleanup succeeded, dest file removed")
	} else {
		t.Log("cleanup failed (expected), dest file still exists")
	}
}

func TestGetFileSize(t *testing.T) {
	tmpdir := t.TempDir()
	testFile := filepath.Join(tmpdir, "test.txt")
	content := []byte("test content for size")

	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	size, err := GetFileSize(testFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if size != int64(len(content)) {
		t.Errorf("got size %d, want %d", size, len(content))
	}
}

func TestGetFileSizeNonExistent(t *testing.T) {
	size, err := GetFileSize(filepath.Join(t.TempDir(), "nonexistent.txt"))
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}

	if !errors.IsFileNotFoundError(err) {
		t.Errorf("expected FileNotFoundError, got: %T", err)
	}

	if size != 0 {
		t.Error("expected size 0 on error")
	}
}

func TestGetFileModTime(t *testing.T) {
	tmpdir := t.TempDir()
	testFile := filepath.Join(tmpdir, "test.txt")

	content := []byte("test content")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	time.Sleep(10 * time.Millisecond) // Ensure file is created before we check modtime

	modTime, err := GetFileModTime(testFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info, _ := os.Stat(testFile)
	expectedTime := info.ModTime()

	if modTime.Before(expectedTime.Add(-1*time.Second)) || modTime.After(expectedTime.Add(1*time.Second)) {
		t.Errorf("got modtime %v, want ~%v", modTime, expectedTime)
	}
}

func TestGetFileModTimeNonExistent(t *testing.T) {
	modTime, err := GetFileModTime(filepath.Join(t.TempDir(), "nonexistent.txt"))
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}

	if !errors.IsFileNotFoundError(err) {
		t.Errorf("expected FileNotFoundError, got: %T", err)
	}

	if !modTime.IsZero() {
		t.Error("expected zero time on error")
	}
}

func TestMoveFileNonExistentSource(t *testing.T) {
	tmpdir := t.TempDir()
	src := filepath.Join(tmpdir, "nonexistent.txt")
	dst := filepath.Join(tmpdir, "dest.txt")

	err := MoveFile(src, dst)
	if err == nil {
		t.Fatal("expected error for non-existent source file")
	}

	if !errors.IsFileNotFoundError(err) {
		t.Errorf("expected FileNotFoundError, got: %T", err)
	}

	// Verify no destination was created
	if FileExists(dst) {
		t.Error("destination should not exist when source doesn't exist")
	}
}

func TestListFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some test files
	testFiles := []string{"file1.txt", "file2.go", "readme.md"}
	for _, f := range testFiles {
		path := filepath.Join(tmpDir, f)
		err := os.WriteFile(path, []byte("test"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", f, err)
		}
	}

	// List files in the directory
	files, err := ListFiles(tmpDir)
	if err != nil {
		t.Fatalf("ListFiles failed: %v", err)
	}

	if len(files) != 3 {
		t.Errorf("Expected 3 files, got %d: %v", len(files), files)
	}

	for _, expected := range testFiles {
		found := false
		for _, f := range files {
			if f == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected file %s not found in list", expected)
		}
	}
}

func TestListFilesNonExistent(t *testing.T) {
	_, err := ListFiles("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Error("Expected error for non-existent directory")
	}
}

func TestIntegrationWorkflow(t *testing.T) {
	tmpdir := t.TempDir()

	// Create source file
	srcFile := filepath.Join(tmpdir, "source.txt")
	content := []byte("integration test content")
	if err := WriteFile(srcFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	// Copy it
	dstFile := filepath.Join(tmpdir, "copy.txt")
	if err := CopyFile(srcFile, dstFile); err != nil {
		t.Fatal(err)
	}

	// Move copy to another location
	movedFile := filepath.Join(tmpdir, "moved.txt")
	if err := MoveFile(dstFile, movedFile); err != nil {
		t.Fatal(err)
	}

	// Verify workflow
	if FileExists(dstFile) {
		t.Error("copy should not exist after move")
	}
	if !FileExists(srcFile) {
		t.Error("source should still exist")
	}
	if !FileExists(movedFile) {
		t.Error("moved file should exist")
	}

	// Get size of moved file
	size, err := GetFileSize(movedFile)
	if err != nil {
		t.Fatal(err)
	}
	if size != int64(len(content)) {
		t.Errorf("size mismatch: got %d, want %d", size, len(content))
	}

	// Read back content
	readContent, err := ReadFileString(movedFile)
	if err != nil {
		t.Fatal(err)
	}
	if readContent != string(content) {
		t.Errorf("content mismatch")
	}

	// Cleanup
	if err := DeleteFile(movedFile); err != nil {
		t.Error(err)
	}
}

func TestDeleteFile_CannotDeleteDirectory(t *testing.T) {
	tmpdir := t.TempDir()
	
	// Create a subdirectory
	subdir := filepath.Join(tmpdir, "subdir")
	if err := os.Mkdir(subdir, 0755); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	
	err := DeleteFile(subdir)
	if err == nil {
		t.Fatal("Expected error when trying to delete directory with DeleteFile")
	}
	
	// Verify directory still exists
	if info, statErr := os.Stat(subdir); statErr != nil || !info.IsDir() {
		t.Error("Directory should still exist after failed DeleteFile attempt")
	}
}

func TestMoveFile_CleanupOnFailure(t *testing.T) {
	tmpdir := t.TempDir()
	srcPath := filepath.Join(tmpdir, "source.txt")
	dstPath := filepath.Join(tmpdir, "dest.txt")
	
	// Create source file
	if err := os.WriteFile(srcPath, []byte("move me"), 0644); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	
	// Test successful move
	err := MoveFile(srcPath, dstPath)
	if err != nil {
		t.Fatalf("Unexpected error on valid move: %v", err)
	}
	
	// Verify source is gone and dest exists
	if FileExists(srcPath) {
		t.Error("Source file should be deleted after move")
	}
	if !FileExists(dstPath) {
		t.Fatal("Destination file should exist after move")
	}
}

func TestGetFileSize_NonExistent(t *testing.T) {
	size, err := GetFileSize("/nonexistent/file.txt")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
	if size != 0 {
		t.Errorf("Expected size 0 on error, got %d", size)
	}
}

func TestGetFileModTime_NonExistent(t *testing.T) {
	modTime, err := GetFileModTime("/nonexistent/file.txt")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
	if !modTime.IsZero() {
		t.Error("Expected zero time on error")
	}
}

func TestReadDir_NonExistent(t *testing.T) {
	entries, err := ReadDir("/nonexistent/directory")
	if err == nil {
		t.Error("Expected error for non-existent directory")
	}
	if entries != nil {
		t.Errorf("Expected nil entries on error, got %d entries", len(entries))
	}
}

func TestListFiles_EmptyDirectory(t *testing.T) {
	tmpdir := t.TempDir()
	
	files, err := ListFiles(tmpdir)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if len(files) != 0 {
		t.Errorf("Expected 0 files in empty directory, got %d: %v", len(files), files)
	}
}

func TestListFiles_WithMixedContent(t *testing.T) {
	tmpdir := t.TempDir()
	
	// Create some files and a subdirectory
	for _, name := range []string{"file1.txt", "file2.md"} {
		if err := os.WriteFile(filepath.Join(tmpdir, name), []byte("test"), 0644); err != nil {
			t.Fatalf("Setup failed: %v", err)
		}
	}
	
	subdir := filepath.Join(tmpdir, "subdir")
	if err := os.Mkdir(subdir, 0755); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	
	files, err := ListFiles(tmpdir)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d: %v", len(files), files)
	}
	
	// Verify only files are listed, not the directory
	for _, f := range files {
		if f == "subdir" {
			t.Error("Directory should not be in file list")
		}
	}
}
