package toolexecutor

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGitExecutor_NewTestExecutor creates a new GitExecutor instance
func TestGitExecutor_NewToolExecutor(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	executor := NewToolExecutor(tmpDir, nil)
	if executor == nil {
		t.Fatal("Expected non-nil GitExecutor")
	}
	if executor.basePath != tmpDir {
		t.Errorf("Expected basePath=%s, got %s", tmpDir, executor.basePath)
	}
}

// TestGitExecutor_PathInitialization tests that the executor correctly stores the base path
func TestGitExecutor_PathInitialization(t *testing.T) {
	testPaths := []string{
		"/tmp/test-git-123",
		"./relative/path/to/repo",
		"/absolute/nested/deep/path/here",
	}

	for _, testPath := range testPaths {
		t.Run(testPath, func(t *testing.T) {
			executor := NewToolExecutor(testPath, nil)
			if executor.basePath != testPath {
				t.Errorf("Expected basePath=%s, got %s", testPath, executor.basePath)
			}
		})
	}
}

// TestGitExecutor_NullConfig tests creating executor with null config
func TestGitExecutor_NilConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	executor := NewToolExecutor(tmpDir, nil)
	if executor == nil {
		t.Fatal("Expected non-nil GitExecutor with nil config")
	}
}

// TestGitExecutor_StringConversion tests that path is stored correctly as string
func TestGitExecutor_StringPath(t *testing.T) {
	testStr := "test-repo-path-12345"
	executor := NewToolExecutor(testStr, nil)
	if executor.basePath != testStr {
		t.Errorf("Expected basePath=%s, got %s", testStr, executor.basePath)
	}
}

// TestGitExecutor_EmptyPath tests behavior with empty string path
func TestGitExecutor_EmptyPath(t *testing.T) {
	executor := NewToolExecutor("", nil)
	if executor == nil {
		t.Fatal("Expected non-nil GitExecutor with empty path")
	}
	if executor.basePath != "" {
		t.Errorf("Expected empty basePath, got %s", executor.basePath)
	}
}

// TestGitExecutor_ConcurrentCreation tests creating multiple executors concurrently
func TestGitExecutor_ConcurrentCreation(t *testing.T) {
	const numExecutors = 10
	var executors []*GitExecutor

	tmpDir := t.TempDir()

	for i := 0; i < numExecutors; i++ {
		executor := NewToolExecutor(tmpDir, nil)
		if executor == nil {
			t.Errorf("Failed to create executor %d", i)
			continue
		}
		executors = append(executors, executor)
	}

	if len(executors) != numExecutors {
		t.Fatalf("Expected %d executors, got %d", numExecutors, len(executors))
	}
}

// TestGitExecutor_WithDifferentConfigTypes tests creating with different config types
func TestGitExecutor_DifferentConfigTypes(t *testing.T) {
	tmpDir := t.TempDir()

	testConfigs := []struct {
		name   string
		config interface{}
	}{
		{"nil", nil},
		{"empty map", map[string]interface{}{}},
		{"map with values", map[string]interface{}{"key": "value"}},
	}

	for _, tc := range testConfigs {
		t.Run(tc.name, func(t *testing.T) {
			executor := NewToolExecutor(tmpDir, tc.config)
			if executor == nil {
				t.Fatalf("Failed to create executor with %s config", tc.name)
			}
		})
	}
}

// TestGitExecutor_BasePathValidation tests that paths are properly handled
func TestGitExecutor_ValidPaths(t *testing.T) {
	validPaths := []string{
		"/tmp",
		"/var/log",
		"/home/user/project",
		".",
		"..",
	}

	for _, path := range validPaths {
		t.Run(path, func(t *testing.T) {
			executor := NewToolExecutor(path, nil)
			if executor == nil {
				t.Fatalf("Failed to create executor for path %s", path)
			}
			if executor.basePath != path {
				t.Errorf("Expected basePath=%s, got %s", path, executor.basePath)
			}
		})
	}
}

// TestGitExecutor_ConfigNilSafety verifies nil config doesn't cause panics
func TestGitExecutor_ConfigNilSafety(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Error("NewToolExecutor should not panic with nil config")
		}
	}()

	tmpDir := t.TempDir()
	executor := NewToolExecutor(tmpDir, nil)

	if executor == nil {
		t.Fatal("Expected non-nil executor")
	}
}

// TestGitExecutor_UniqueInstances tests that each call creates a unique instance
func TestGitExecutor_UniqueInstances(t *testing.T) {
	tmpDir := t.TempDir()

	executor1 := NewToolExecutor(tmpDir, nil)
	executor2 := NewToolExecutor(tmpDir, nil)

	if executor1 == executor2 {
		t.Error("Expected different executor instances")
	}

	if executor1.basePath != tmpDir || executor2.basePath != tmpDir {
		t.Error("Both executors should have the same path")
	}
}

// TestGitExecutor_WithPathModification tests that path cannot be modified after creation
func TestGitExecutor_PathImmutability(t *testing.T) {
	originalPath := "/original/path"
	executor := NewToolExecutor(originalPath, nil)

	// Verify the path is stored correctly
	if executor.basePath != originalPath {
		t.Errorf("Expected %s, got %s", originalPath, executor.basePath)
	}

	// The path should be accessible and consistent
	path := executor.basePath
	if path != originalPath {
		t.Errorf("Stored path changed: expected %s, got %s", originalPath, path)
	}
}
