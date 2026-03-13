// Package utils provides utility functions for common operations with enhanced error handling.
package utils

import (
	"os"
	"path/filepath"
	"time"

	errors "yolo/errors"
)

// ReadFile reads a file and returns its contents as bytes, or a custom FileNotFoundError if missing.
func ReadFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.NewFileNotFoundError("read", path, nil)
		}
		if os.IsPermission(err) {
			return nil, errors.Wrap(err, errors.FileType, map[string]any{"op": "read", "path": path})
		}
		return nil, errors.WithContext(err, "read", path)
	}
	return data, nil
}

// ReadFileString reads a file and returns its contents as a string.
func ReadFileString(path string) (string, error) {
	data, err := ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WriteFile writes data to a file with the specified permissions.
func WriteFile(path string, data []byte, perm os.FileMode) error {
	// Ensure parent directory exists
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return errors.Wrap(err, errors.FileType, map[string]any{"op": "mkdir", "path": dir})
		}
	}

	err := os.WriteFile(path, data, perm)
	if err != nil {
		if os.IsPermission(err) {
			return errors.Wrap(err, errors.FileType, map[string]any{"op": "write", "path": path})
		}
		return errors.WithContext(err, "write", path)
	}
	return nil
}

// WriteFileString writes a string to a file with the specified permissions.
func WriteFileString(path, content string, perm os.FileMode) error {
	return WriteFile(path, []byte(content), perm)
}

// FileExists checks if a file (not directory) exists at the given path.
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// IsDirectory checks if the path exists and is a directory.
func IsDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// EnsureDir creates a directory and all parent directories if they don't exist.
func EnsureDir(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return errors.Wrap(err, errors.FileType, map[string]any{"op": "mkdir", "path": path})
	}
	return nil
}

// DeleteFile removes a file. Returns an error if path is a directory or doesn't exist.
func DeleteFile(path string) error {
	// First check if it exists and what type it is
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.NewFileNotFoundError("delete", path, nil)
		}
		return errors.WithContext(err, "stat", path)
	}

	// Don't allow deleting directories with DeleteFile
	if info.IsDir() {
		return errors.WithContext(os.ErrPermission, "cannot delete directory", path)
	}

	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return errors.NewFileNotFoundError("delete", path, nil)
		}
		return errors.WithContext(err, "delete", path)
	}
	return nil
}

// CopyFile copies a file from src to dst.
func CopyFile(src, dst string) error {
	// Read source file
	data, err := ReadFile(src)
	if err != nil {
		return errors.Wrap(err, errors.FileType, map[string]any{"op": "copy", "path": src})
	}

	// Write to destination
	if err := WriteFile(dst, data, 0644); err != nil {
		return errors.Wrap(err, errors.FileType, map[string]any{"op": "copy", "path": dst})
	}

	return nil
}

// MoveFile moves a file from src to dst by copying and deleting the original.
func MoveFile(src, dst string) error {
	if err := CopyFile(src, dst); err != nil {
		return err
	}
	if err := DeleteFile(src); err != nil {
		// Clean up: try to delete the copied file if original deletion fails
		DeleteFile(dst) // ignore error
		return err
	}
	return nil
}

// GetFileSize returns the size of a file in bytes.
func GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, errors.NewFileNotFoundError("stat", path, nil)
		}
		return 0, errors.WithContext(err, "stat", path)
	}
	return info.Size(), nil
}

// GetFileModTime returns the modification time of a file.
func GetFileModTime(path string) (time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return time.Time{}, errors.NewFileNotFoundError("stat", path, nil)
		}
		return time.Time{}, errors.WithContext(err, "stat", path)
	}
	return info.ModTime(), nil
}

// ReadDir reads a directory and returns its entries.
func ReadDir(path string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.NewFileNotFoundError("readdir", path, nil)
		}
		return nil, errors.WithContext(err, "readdir", path)
	}
	return entries, nil
}

// ListFiles returns a list of files (not directories) in the given path.
func ListFiles(path string) ([]string, error) {
	entries, err := ReadDir(path)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}
	return files, nil
}
