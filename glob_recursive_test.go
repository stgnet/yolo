package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGlobRecursive(t *testing.T) {
	// Create temp directory structure for testing
	tmpDir := t.TempDir()

	// Create test directory structure:
	// tmpDir/
	//   file1.txt
	//   subdir1/
	//     file2.txt
	//     nested/
	//       file3.txt
	//   subdir2/
	//     file4.go

	subdir1 := filepath.Join(tmpDir, "subdir1")
	nested := filepath.Join(subdir1, "nested")
	subdir2 := filepath.Join(tmpDir, "subdir2")

	mustCreateDir(t, subdir1)
	mustCreateDir(t, nested)
	mustCreateDir(t, subdir2)

	// Create test files
	testFiles := map[string]string{
		filepath.Join(tmpDir, "file1.txt"):  "content1",
		filepath.Join(subdir1, "file2.txt"): "content2",
		filepath.Join(nested, "file3.txt"):  "content3",
		filepath.Join(subdir2, "file4.go"):  "package main",
	}

	for path, content := range testFiles {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	tests := []struct {
		name    string
		pattern string
		wantLen int
		hasFile string // filename that should be in results (basename only)
	}{
		{
			name:    "matches txt files with **/*.txt",
			pattern: "**/*.txt",
			wantLen: 3,
			hasFile: "file2.txt",
		},
		{
			name:    "matches go files with **/*.go",
			pattern: "**/*.go",
			wantLen: 1,
			hasFile: "file4.go",
		},
		{
			name:    "matches specific file type",
			pattern: "*.txt",
			wantLen: 1, // Only matches top level file1.txt since no **
			hasFile: "file1.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := globFiles(tmpDir, tt.pattern)
			if err != nil {
				t.Fatalf("globFiles() error = %v", err)
			}

			if len(got) != tt.wantLen {
				t.Errorf("globFiles() got %d files, want %d. Got: %v", len(got), tt.wantLen, got)
			}

			if tt.hasFile != "" {
				found := false
				for _, f := range got {
					if filepath.Base(f) == tt.hasFile {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("globFiles() expected to find %s, got: %v", tt.hasFile, got)
				}
			}
		})
	}

	// Test with non-existent directory
	t.Run("non-existent pattern", func(t *testing.T) {
		got, err := globFiles("/non/existent/path", "**/*.go")
		if err != nil {
			t.Logf("Expected error for non-existent path: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("Expected empty result for non-existent path, got: %v", got)
		}
	})
}

func mustCreateDir(t *testing.T, path string) {
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatalf("Failed to create dir %s: %v", path, err)
	}
}
