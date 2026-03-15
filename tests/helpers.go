// Test helpers for YOLO unit tests
package main

import "strings"

// containsString checks if result contains a substring
func containsString(result, substr string) bool {
	return len(result) > 0 && len(substr) > 0 && 
		strings.Contains(result, substr)
}
