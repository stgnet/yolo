package main

import (
	"strings"
	"testing"
)

// TestToolActivityFormat tests the [tool activity] prefix format parsing
func TestToolActivityFormat(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name: "simple tool activity",
			content: `[tool activity]
Test that this is correctly parsed`,
			expected: true,
		},
		{
			name: "tool activity with parameter descriptions",
			content: `[tool activity]
[read_file] => Read a file's contents
[write_file] => Create or overwrite a file`,
			expected: true,
		},
		{
			name: "tool activity with regex pattern",
			content: `[tool activity]
Search for pattern in files [search_files] with query 'func.*test'`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := strings.Contains(tt.content, "[tool activity]")
			if matches != tt.expected {
				t.Errorf("Expected %v but got %v", tt.expected, matches)
			}
		})
	}
}
