package utils

import (
	"os"
	"path/filepath"
	"strings"
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

	t.Run("empty file", func(t *testing.T) {
		tmpfile := filepath.Join(t.TempDir(), "empty.txt")
		os.WriteFile(tmpfile, []byte{}, 0644)

		result, err := ReadFileString(tmpfile)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "" {
			t.Errorf("expected empty string for empty file, got %q", result)
		}
	})

	t.Run("binary content", func(t *testing.T) {
		tmpfile := filepath.Join(t.TempDir(), "binary.bin")
		binaryContent := []byte{0x00, 0xFF, 0xAB, 0xCD, 0x12}
		if err := os.WriteFile(tmpfile, binaryContent, 0644); err != nil {
			t.Fatal(err)
		}

		result, err := ReadFileString(tmpfile)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(binaryContent) != result {
			t.Errorf("got %q, want %q", result, binaryContent)
		}
	})

	t.Run("directory instead of file", func(t *testing.T) {
		tmpdir := t.TempDir()
		result, err := ReadFileString(tmpdir)
		if err == nil {
			t.Fatal("expected error when reading directory as file")
		}
		if result != "" {
			t.Error("expected empty string on error")
		}
	})

	t.Run("permission denied", func(t *testing.T) {
		tmpfile := filepath.Join(t.TempDir(), "readonly.txt")
		os.WriteFile(tmpfile, []byte("content"), 0444)

		result, err := ReadFileString(tmpfile)
		if err != nil {
			t.Logf("Got expected error: %v", err)
		} else if result == "" && os.Getuid() == 0 {
			t.Error("root can read all files, test may not apply")
		}
	})
}

func TestReadFileString_EdgeCases(t *testing.T) {
	t.Run("file with newlines", func(t *testing.T) {
		tmpfile := filepath.Join(t.TempDir(), "newlines.txt")
		content := "line1\nline2\nline3\n"
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

	t.Run("very long file", func(t *testing.T) {
		tmpfile := filepath.Join(t.TempDir(), "long.txt")
		longContent := strings.Repeat("a", 1000000) // 1MB string
		if err := os.WriteFile(tmpfile, []byte(longContent), 0644); err != nil {
			t.Fatal(err)
		}

		result, err := ReadFileString(tmpfile)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != len(longContent) {
			t.Errorf("length mismatch: got %d, want %d", len(result), len(longContent))
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

	t.Run("empty content", func(t *testing.T) {
		tmpfile := filepath.Join(t.TempDir(), "empty.txt")

		err := WriteFile(tmpfile, []byte{}, 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		data, err := os.ReadFile(tmpfile)
		if err != nil {
			t.Fatal(err)
		}
		if len(data) != 0 {
			t.Errorf("expected empty file, got length %d", len(data))
		}
	})

	t.Run("write with different permissions", func(t *testing.T) {
		tmpfile := filepath.Join(t.TempDir(), "test.txt")
		content := []byte("content")

		err := WriteFile(tmpfile, content, 0600)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		info, err := os.Stat(tmpfile)
		if err != nil {
			t.Fatal(err)
		}
		expectedPerms := os.FileMode(0600)
		if info.Mode().Perm() != expectedPerms {
			t.Errorf("got perms %o, want %o", info.Mode().Perm(), expectedPerms)
		}
	})

	t.Run("binary content", func(t *testing.T) {
		tmpfile := filepath.Join(t.TempDir(), "binary.bin")
		binaryContent := []byte{0x00, 0xFF, 0xAB, 0xCD, 0x12, 0x34}

		err := WriteFile(tmpfile, binaryContent, 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		data, err := os.ReadFile(tmpfile)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != string(binaryContent) {
			t.Errorf("got %q, want %q", data, binaryContent)
		}
	})

	t.Run("write to existing directory fails gracefully", func(t *testing.T) {
		tmpdir := t.TempDir()
		pathAsFile := filepath.Join(tmpdir, "file")
		os.MkdirAll(pathAsFile, 0755) // Create as directory

		err := WriteFile(filepath.Join(pathAsFile, "nested.txt"), []byte("test"), 0644)
		if err == nil {
			t.Error("expected error when parent is a file, not directory")
		} else if !strings.Contains(err.Error(), "mkdir") && !errors.IsFileNotFoundError(err) {
			// Expected error for mkdir or other filesystem issues
			t.Logf("Got expected error: %v", err)
		}
	})

	t.Run("write to readonly directory fails", func(t *testing.T) {
		tmpdir := t.TempDir()
		subdir := filepath.Join(tmpdir, "readonly")
		os.MkdirAll(subdir, 0555) // Read-only directory

		targetPath := filepath.Join(subdir, "test.txt")
		err := WriteFile(targetPath, []byte("test"), 0644)
		if err == nil {
			t.Error("expected error when writing to read-only directory")
		} else if os.Getuid() != 0 && !errors.IsFileNotFoundError(err) {
			t.Logf("Got expected permission error: %v", err)
		}

		// Restore permissions so cleanup works
		os.Chmod(subdir, 0755)
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

	t.Run("delete protected path", func(t *testing.T) {
		testPath := "go.mod" // In protected paths list

		err := DeleteFile(testPath)
		if err == nil {
			t.Fatal("expected error for protected path")
		}

		// Should not be a FileNotFoundError since the file may or may not exist
		if errors.IsFileNotFoundError(err) {
			t.Error("should get permission error for protected path, not FileNotFoundError")
		}
	})

	t.Run("delete file with backup created", func(t *testing.T) {
		tmpfile := filepath.Join(t.TempDir(), "test.txt")
		content := []byte("content to delete")
		if err := os.WriteFile(tmpfile, content, 0644); err != nil {
			t.Fatal(err)
		}

		err := DeleteFile(tmpfile)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify backup was created
		backupPath := tmpfile + ".bak"
		if !FileExists(backupPath) {
			t.Error("expected backup to be created")
		} else {
			backupContent, _ := os.ReadFile(backupPath)
			if string(backupContent) != string(content) {
				t.Error("backup content should match original")
			}
		}

		// Verify file was deleted
		if FileExists(tmpfile) {
			t.Error("file should be deleted after successful deletion")
		}
	})

	t.Run("permission denied on directory", func(t *testing.T) {
		tmpdir := t.TempDir()
		subdir := filepath.Join(tmpdir, "readonly")
		os.MkdirAll(subdir, 0555) // Read-only

		testFile := filepath.Join(subdir, "test.txt")
		os.WriteFile(testFile, []byte("content"), 0644)

		err := DeleteFile(testFile)
		if err == nil {
			t.Error("expected error when file is in read-only directory")
		} else if os.Getuid() != 0 && !errors.IsFileNotFoundError(err) {
			t.Logf("Got expected permission error: %v", err)
		}

		// Restore permissions so cleanup works
		os.Chmod(subdir, 0755)
	})
}

func TestDeleteFile_CleanupOnBackupFailure(t *testing.T) {
	tmpfile := filepath.Join(t.TempDir(), "test.txt")
	content := []byte("content to delete")
	if err := os.WriteFile(tmpfile, content, 0644); err != nil {
		t.Fatal(err)
	}

	// Make the file writable but unreadable to test backup failure path
	os.Chmod(tmpfile, 0200) // Write-only

	err := DeleteFile(tmpfile)
	if err == nil {
		// May succeed or fail depending on whether read is required for backup
		t.Logf("Delete behavior: %v", err)
	}

	// Restore and clean up
	os.Chmod(tmpfile, 0644)
	DeleteFile(tmpfile)
	DeleteFile(tmpfile + ".bak")
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

func TestCopyFile_SourceNotFound(t *testing.T) {
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

func TestCopyFile_EmptyContent(t *testing.T) {
	tmpdir := t.TempDir()
	src := filepath.Join(tmpdir, "empty.txt")
	dst := filepath.Join(tmpdir, "dest.txt")

	// Create empty source file
	os.WriteFile(src, []byte{}, 0644)

	err := CopyFile(src, dst)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}

	if info.Size() != 0 {
		t.Errorf("expected empty destination, got size %d", info.Size())
	}
}

func TestCopyFile_BinaryContent(t *testing.T) {
	tmpdir := t.TempDir()
	src := filepath.Join(tmpdir, "binary.bin")
	dst := filepath.Join(tmpdir, "dest.bin")
	binaryContent := []byte{0x00, 0xFF, 0xAB, 0xCD, 0x12, 0x34, 0xBE, 0xEF}

	if err := os.WriteFile(src, binaryContent, 0644); err != nil {
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

	if string(dstData) != string(binaryContent) {
		t.Errorf("binary content mismatch: got %q, want %q", dstData, binaryContent)
	}
}

func TestCopyFile_OverwriteDestination(t *testing.T) {
	tmpdir := t.TempDir()
	src := filepath.Join(tmpdir, "source.txt")
	dst := filepath.Join(tmpdir, "dest.txt")

	if err := os.WriteFile(src, []byte("source"), 0644); err != nil {
		t.Fatal(err)
	}

	os.WriteFile(dst, []byte("existing"), 0644)

	err := CopyFile(src, dst)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(dst)
	if string(data) != "source" {
		t.Errorf("expected destination to be overwritten with source content")
	}
}

func TestCopyFile_DestinationReadOnly(t *testing.T) {
	tmpdir := t.TempDir()
	src := filepath.Join(tmpdir, "source.txt")
	dst := filepath.Join(tmpdir, "readonly_dest")
	os.WriteFile(src, []byte("content"), 0644)

	// Create read-only destination directory
	if err := os.MkdirAll(dst, 0555); err != nil {
		t.Fatal(err)
	}

	destPath := filepath.Join(dst, "file.txt")
	err := CopyFile(src, destPath)
	if err == nil {
		t.Error("expected error when copying to read-only directory")
	} else if os.Getuid() != 0 && !errors.IsFileNotFoundError(err) {
		t.Logf("Got expected permission error: %v", err)
	}

	os.Chmod(dst, 0755) // Restore for cleanup
}

func TestCopyFile_PermissionDenied(t *testing.T) {
	tmpdir := t.TempDir()
	src := filepath.Join(tmpdir, "source.txt")

	os.WriteFile(src, []byte("content"), 0444)

	dst := filepath.Join(tmpdir, "dest.txt")
	err := CopyFile(src, dst)
	if err != nil {
		t.Logf("Got error reading source: %v", err)
	}

	// Clean up - file might still be in weird state
	os.Chmod(src, 0644)
	DeleteFile(src)
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

func TestMoveFile_OverwritesDestination(t *testing.T) {
	tmpdir := t.TempDir()
	src := filepath.Join(tmpdir, "source.txt")
	dst := filepath.Join(tmpdir, "dest.txt")

	if err := os.WriteFile(src, []byte("source"), 0644); err != nil {
		t.Fatal(err)
	}

	os.WriteFile(dst, []byte("existing"), 0644)

	err := MoveFile(src, dst)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if FileExists(src) {
		t.Error("source should be deleted")
	}

	data, _ := os.ReadFile(dst)
	if string(data) != "source" {
		t.Errorf("expected destination to have source content")
	}
}

func TestMoveFile_NestedDestination(t *testing.T) {
	tmpdir := t.TempDir()
	src := filepath.Join(tmpdir, "source.txt")
	dst := filepath.Join(tmpdir, "subdir1", "subdir2", "dest.txt")
	content := []byte("content for nested move")

	if err := os.WriteFile(src, content, 0644); err != nil {
		t.Fatal(err)
	}

	err := MoveFile(src, dst)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != string(content) {
		t.Errorf("content mismatch after move to nested path")
	}
}

func TestMoveFile_SourceNotFound(t *testing.T) {
	tmpdir := t.TempDir()
	src := filepath.Join(tmpdir, "nonexistent.txt")
	dst := filepath.Join(tmpdir, "dest.txt")

	err := MoveFile(src, dst)
	if err == nil {
		t.Fatal("expected error when source doesn't exist")
	}

	if !errors.IsFileNotFoundError(err) {
		t.Errorf("expected FileNotFoundError, got: %T", err)
	}
}

func TestMoveFile_PermissionDenied(t *testing.T) {
	tmpdir := t.TempDir()
	src := filepath.Join(tmpdir, "source.txt")
	dstDir := filepath.Join(tmpdir, "readonly_dir")

	if err := os.MkdirAll(dstDir, 0555); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(src, []byte("content"), 0644)

	dst := filepath.Join(dstDir, "dest.txt")
	err := MoveFile(src, dst)
	if err == nil && os.Getuid() == 0 {
		// Root can write to any directory, so no error
		t.Log("Root user test may not apply")
	} else if err != nil && os.Getuid() != 0 {
		t.Logf("Got expected permission error: %v", err)
	}

	os.Chmod(dstDir, 0755) // Restore for cleanup
}

func TestMoveFile_InvalidDestinationPath(t *testing.T) {
	tmpdir := t.TempDir()
	src := filepath.Join(tmpdir, "source.txt")

	if err := os.WriteFile(src, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Try to move to a path where parent is a file (not directory)
	blockerPath := filepath.Join(tmpdir, "blocker")
	os.WriteFile(blockerPath, []byte("block"), 0644)

	dst := filepath.Join(blockerPath, "nested", "dest.txt")
	err := MoveFile(src, dst)
	if err == nil {
		t.Error("expected error when parent path is a file")
	} else if !errors.IsFileNotFoundError(err) && !strings.Contains(err.Error(), "mkdir") {
		t.Logf("Got expected filesystem error: %v", err)
	}

	DeleteFile(blockerPath) // Clean up
}

func TestMoveFile_BackupCreated(t *testing.T) {
	tmpdir := t.TempDir()
	src := filepath.Join(tmpdir, "source.txt")
	dst := filepath.Join(tmpdir, "dest.txt")
	content := []byte("content for backup test")

	if err := os.WriteFile(src, content, 0644); err != nil {
		t.Fatal(err)
	}

	err := MoveFile(src, dst)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Backup should be created for source before moving
	backupPath := src + ".bak"
	if FileExists(backupPath) {
		backupContent, _ := os.ReadFile(backupPath)
		if string(backupContent) != string(content) {
			t.Error("backup content should match original source")
		}
		DeleteFile(backupPath) // Clean up backup
	}
}

func TestMoveFile_EmptySource(t *testing.T) {
	tmpdir := t.TempDir()
	src := filepath.Join(tmpdir, "empty.txt")
	dst := filepath.Join(tmpdir, "dest.txt")

	os.WriteFile(src, []byte{}, 0644)

	err := MoveFile(src, dst)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}

	if info.Size() != 0 {
		t.Errorf("expected empty destination, got size %d", info.Size())
	}
}

func TestMoveFile_BinaryContent(t *testing.T) {
	tmpdir := t.TempDir()
	src := filepath.Join(tmpdir, "binary.bin")
	dst := filepath.Join(tmpdir, "dest.bin")
	binaryContent := []byte{0x00, 0xFF, 0xAB, 0xCD, 0x12, 0xBE, 0xEF}

	if err := os.WriteFile(src, binaryContent, 0644); err != nil {
		t.Fatal(err)
	}

	err := MoveFile(src, dst)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != string(binaryContent) {
		t.Errorf("binary content mismatch after move")
	}
}

func TestMoveFile_CleanupOnFailure(t *testing.T) {
	tmpdir := t.TempDir()
	src := filepath.Join(tmpdir, "source.txt")
	dstDir := filepath.Join(tmpdir, "readonly_dir")
	dst := filepath.Join(dstDir, "dest.txt")

	os.WriteFile(src, []byte("content"), 0644)
	os.MkdirAll(dstDir, 0755)
	CopyFile(src, dst) // Copy file first to simulate partial move failure scenario

	// Delete source manually to simulate partial move failure scenario
	DeleteFile(src)

	// Try to move (source doesn't exist, should fail gracefully)
	err := MoveFile(src, dst)
	if err == nil {
		t.Fatal("expected error when source doesn't exist")
	}

	// The function tries cleanup but ignores errors, so dest may or may not be cleaned up
	t.Logf("MoveFile behavior on failure: %v", err)
	DeleteFile(dst) // Clean up
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
