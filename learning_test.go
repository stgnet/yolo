package main

import (
	"strings"
	"testing"
)

func TestContainsGenericPattern(t *testing.T) {
	tests := []struct {
		text     string
		patterns []string
		want     bool
	}{
		{"This is a test", []string{" is a "}, true},
		{"No pattern here", []string{"xyz"}, false},
		{"This refers to something", []string{" refers to"}, true},
	}

	for _, tt := range tests {
		got := containsGenericPattern(tt.text, tt.patterns)
		if got != tt.want {
			t.Errorf("containsGenericPattern(%q, %v) = %v, want %v", tt.text, tt.patterns, got, tt.want)
		}
	}
}

func TestContainsActionableContent(t *testing.T) {
	tests := []struct {
		text string
		want bool
	}{
		{"Use dependency injection for better testability and improve code maintainability", true}, // contains "improve"
		{"Implement error handling with custom types following best practice guidelines", true},    // contains "implement" and "best practice"
		{"This is just informational text without any actionable content", false},                  // no keywords
	}

	for _, tt := range tests {
		got := containsActionableContent(tt.text)
		if got != tt.want {
			t.Errorf("containsActionableContent(%q) = %v, want %v", tt.text, got, tt.want)
		}
	}
}

func TestExtractKeywords(t *testing.T) {
	tests := []struct {
		text          string
		suggestedKeys []string
		wantMinCount  int
	}{
		{
			text:          "Go routines and channels for concurrency in parallel processing",
			suggestedKeys: []string{"go", "routines", "channels", "concurrency"},
			wantMinCount:  3,
		},
		{
			text:          "Simple test case",
			suggestedKeys: []string{"simple", "test"},
			wantMinCount:  2,
		},
	}

	for _, tt := range tests {
		got := extractKeywords(tt.text, tt.suggestedKeys)
		if len(got) < tt.wantMinCount {
			t.Errorf("extractKeywords() = %d keywords (%v), want at least %d", len(got), got, tt.wantMinCount)
		}
	}
}

func TestTruncateText(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		maxLen  int
		wantMax int
		wantEnd string
	}{
		{
			name:    "text shorter than max",
			text:    "Short text",
			maxLen:  100,
			wantMax: 100,
			wantEnd: "",
		},
		{
			name:    "text longer than max",
			text:    strings.Repeat("A", 200),
			maxLen:  50,
			wantMax: 54, // 50 + "...*" (4 chars)
			wantEnd: "...*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateText(tt.text, tt.maxLen)
			if len(result) > tt.wantMax {
				t.Errorf("truncateText() length = %d, want max %d", len(result), tt.wantMax)
			}
			if tt.wantEnd != "" && !strings.HasSuffix(result, tt.wantEnd) {
				t.Errorf("truncateText() doesn't end with %q, got: %q", tt.wantEnd, result)
			}
		})
	}
}

func TestGenerateImprovementID(t *testing.T) {
	// Note: This tests the improvement ID generation which doesn't need a LearningModel
	id := generateImprovementID("test improvement for code quality")

	if id == "" {
		t.Error("generateImprovementID() returned empty string")
	}

	// ID should be IMP- followed by 5 digits, so length is between 8 and 10
	if len(id) < 8 || len(id) > 10 {
		t.Errorf("generateImprovementID() = %q (len=%d), want between 8 and 10", id, len(id))
	}

	// Should start with "IMP-"
	if !strings.HasPrefix(id, "IMP-") {
		t.Errorf("generateImprovementID() doesn't start with 'IMP-', got: %q", id)
	}
}

func TestLearningModel_ExtractCompleteSentences(t *testing.T) {
	lm := &LearningManager{} // Create a minimal instance for testing

	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{
			name:     "multiple long sentences",
			text:     `Go routines are lightweight threads managed by the Go runtime that enable concurrent execution with minimal overhead and memory usage for high-performance applications. Channel communication provides type-safe data passing between goroutines which ensures thread-safe access to shared resources without explicit locking mechanisms. The context package enables cancellation and timeouts for operations that need graceful termination signals during application shutdown or timeout scenarios.`,
			expected: 3,
		},
		{
			name:     "short fragments filtered out",
			text:     "Use proper error handling in production systems to prevent crashes and data loss from unhandled exceptions during runtime execution and ensure system stability. in production mode.",
			expected: 1,
		},
		{
			name:     "html entities decoded properly",
			text:     `Go's &#039;context&#039; package enables cancellation and timeouts for operations that need to be gracefully terminated when deadlines are exceeded or parent contexts cancel in distributed systems. The &quot;canceled&quot; context stops all child operations gracefully without leaking goroutines or resources during application shutdown procedures.`,
			expected: 2,
		},
		{
			name:     "empty text returns nothing",
			text:     "",
			expected: 0,
		},
		{
			name:     "mixed content with fragments",
			text:     `Software engineering best practices include writing tests first before implementing production code to ensure quality standards and maintain long-term code health. in the development cycle. Automated testing ensures code quality and prevents regressions from breaking changes during refactoring operations while maintaining backward compatibility requirements. of features added later.`,
			expected: 2,
		},
		{
			name:     "lowercase fragments filtered out",
			text:     `Important development principles should be followed consistently across all projects and teams to ensure code quality and maintainability standards. this is a lowercase fragment that should be skipped entirely from the results because it starts with a lowercase letter. Use comprehensive error handling strategies in production systems to ensure reliability and maintainability of critical services.`,
			expected: 2, // Only the two sentences starting with capitals (both are 50+ chars)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := lm.extractCompleteSentences(tt.text)
			if len(result) != tt.expected {
				t.Errorf("extractCompleteSentences() = %d sentences, want %d", len(result), tt.expected)
				for i, s := range result {
					t.Logf("sentence %d: %q (len=%d)", i, s, len(s))
				}
			}
		})
	}
}

func TestLearningModel_GenerateTitle(t *testing.T) {
	lm := &LearningManager{} // Create a minimal instance for testing

	tests := []struct {
		name       string
		content    string
		wantLenMin int
		wantLenMax int
	}{
		{
			name:       "long content truncated properly",
			content:    strings.Repeat("This is a sentence that describes the content. ", 20),
			wantLenMin: 10,
			wantLenMax: 110,
		},
		{
			name:       "short single sentence",
			content:    "Testing improvements.",
			wantLenMin: 10,
			wantLenMax: 120,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title := lm.generateTitle(tt.content, "web")
			if len(title) < tt.wantLenMin || len(title) > tt.wantLenMax {
				t.Errorf("generateTitle() length = %d (title=%q), want between %d and %d",
					len(title), title, tt.wantLenMin, tt.wantLenMax)
			}
		})
	}
}
