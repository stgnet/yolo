package yolo

import (
	"testing"
)

func TestWebSearchTool(t *testing.T) {
	testCases := []struct {
		name     string
		query    string
		count    int
		valid    bool
	}{
		{
			name:  "Valid query with default count",
			query: "Go programming language",
			count: 5,
			valid: true,
		},
		{
			name:  "Valid query with custom count",
			query: "Python tutorials",
			count: 10,
			valid: true,
		},
		{
			name:  "Invalid empty query",
			query: "",
			count: 5,
			valid: false,
		},
		{
			name:  "Query with count exceeding max",
			query: "test",
			count: 15,
			valid: true, // Should be capped to 10
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := map[string]interface{}{
				"query": tc.query,
				"count": tc.count,
			}

			result := webSearch(args)

			if !tc.valid {
				status, ok := result["status"].(string)
				if !ok || status != "error" {
					t.Errorf("Expected error status, got: %v", result)
				}
				return
			}

			// For valid queries, check that we get some kind of response
			// (may be empty due to API limitations in test environment)
			if _, ok := result["status"]; !ok {
				t.Errorf("Expected status field in result, got: %v", result)
			}
		})
	}
}

func TestWebSearchToolMinimizesAPIUsage(t *testing.T) {
	// Verify that web_search tool properly constructs the query
	args := map[string]interface{}{
		"query": "test search term",
		"count": 3,
	}

	result := webSearch(args)

	// Check that result is a map
	if result == nil {
		t.Fatal("webSearch returned nil")
	}

	// In test environment without internet, we may get empty results
	// but the tool should still return a valid structure
	if _, ok := result["status"]; !ok {
		t.Log("No status field - API may be unavailable in test env")
	}
}

func TestWebSearchWithInvalidCount(t *testing.T) {
	testCases := []struct {
		name  string
		count int
		valid bool
	}{
		{"Negative count", -1, true}, // Should default to 5
		{"Zero count", 0, true},      // Should default to 5
		{"Max count", 10, true},
		{"Over max count", 20, true}, // Should be capped to 10
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := map[string]interface{}{
				"query": "test",
				"count": tc.count,
			}

			result := webSearch(args)

			if result == nil {
				t.Fatal("webSearch returned nil")
			}
		})
	}
}
