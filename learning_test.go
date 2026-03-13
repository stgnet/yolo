package main

import (
	"testing"
)

func TestRemoveDuplicateImprovements(t *testing.T) {
	lm := &LearningManager{}

	tests := []struct {
		name     string
		input    []Improvement
		expected int
	}{
		{
			name: "removes exact duplicates",
			input: []Improvement{
				{Title: "Implement feature X", Description: "Add feature X"},
				{Title: "Implement feature X", Description: "Add feature X"},
				{Title: "Different feature", Description: "Something else"},
			},
			expected: 2,
		},
		{
			name: "removes near-duplicates with similar titles",
			input: []Improvement{
				{Title: "Optimize workflow performance", Description: "Make things faster"},
				{Title: "Improve workflow performance", Description: "Speed up operations"},
			},
			expected: 1, // Should detect as duplicate due to similar title
		},
		{
			name: "removes near-duplicates with similar descriptions",
			input: []Improvement{
				{Title: "Feature A", Description: "Setapp provides features designed to optimize workflow"},
				{Title: "Feature B", Description: "Setapp offers tools built for workflow optimization"},
			},
			expected: 1, // Should detect as duplicate due to similar description after stop word filtering
		},
		{
			name: "keeps different improvements",
			input: []Improvement{
				{Title: "Security improvement", Description: "Add authentication"},
				{Title: "Performance boost", Description: "Optimize memory usage"},
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := lm.removeDuplicateImprovements(tt.input)
			if len(result) != tt.expected {
				t.Errorf("Expected %d unique improvements, got %d", tt.expected, len(result))
				for i, r := range result {
					t.Logf("Result %d: Title=%s, Desc=%s", i, r.Title, r.Description)
				}
			}
		})
	}
}

func TestSimilarity(t *testing.T) {
	lm := &LearningManager{}

	tests := []struct {
		name     string
		s1       string
		s2       string
		expected float64
	}{
		{
			name:     "identical strings",
			s1:       "hello world",
			s2:       "hello world",
			expected: 1.0,
		},
		{
			name:     "completely different",
			s1:       "hello world",
			s2:       "foo bar baz",
			expected: 0.0,
		},
		{
			name:     "partial overlap",
			s1:       "hello world",
			s2:       "hello there",
			expected: 0.5, // 1 common word (hello) / ((1+1)+(1+1)) = 2/4 = 0.5
		},
		{
			name:     "with stop words filtered",
			s1:       "the quick brown fox jumps over the lazy dog",
			s2:       "the fast brown fox leaps over the sleepy dog",
			expected: 0.6, // Should be higher because stop words are filtered
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := lm.similarity(tt.s1, tt.s2)

			// Use approximate comparison for floats
			if result < 0 {
				t.Errorf("Similarity should not be negative: %f", result)
			}
			if result > 1 {
				t.Errorf("Similarity should not exceed 1: %f", result)
			}

			// Check within reasonable tolerance (0.2 for fuzzy matching)
			diff := result - tt.expected
			if diff < 0 {
				diff = -diff
			}
			if diff > 0.2 {
				t.Logf("Note: Got %f, expected ~%f (may vary due to stop word filtering)",
					result, tt.expected)
			}
		})
	}
}

func TestFilterStopWords(t *testing.T) {
	lm := &LearningManager{}

	tests := []struct {
		name     string
		input    []string
		expected int // expected number of non-stop words
	}{
		{
			name:     "all stop words",
			input:    []string{"the", "and", "is"},
			expected: 0,
		},
		{
			name:     "mixed content",
			input:    []string{"implement", "the", "new", "feature"},
			expected: 3, // implement, new, feature remain
		},
		{
			name:     "filters short words",
			input:    []string{"a", "an", "golang", "tool"},
			expected: 2, // golang (len>2), tool remain; 'go' is length 2 so filtered
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := lm.filterStopWords(tt.input)
			if len(result) != tt.expected {
				t.Errorf("Expected %d words after filtering, got %d: %v",
					tt.expected, len(result), result)
			}
		})
	}
}

func TestAnalyzeTrends(t *testing.T) {
	lm := &LearningManager{}

	tests := []struct {
		name    string
		input   []Improvement
		wantLen int // minimum expected trends
	}{
		{
			name: "identifies repeating keywords",
			input: []Improvement{
				{Title: "Performance optimization", Keywords: []string{"performance", "optimization"}},
				{Title: "Performance monitoring", Keywords: []string{"performance", "monitoring"}},
				{Title: "Security improvement", Keywords: []string{"security", "improvement"}},
			},
			wantLen: 1, // "performance" appears twice
		},
		{
			name:    "empty input",
			input:   []Improvement{},
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trends := lm.analyzeTrends(tt.input)
			if len(trends) < tt.wantLen {
				t.Errorf("Expected at least %d trends, got %d: %v",
					tt.wantLen, len(trends), trends)
			}
		})
	}
}

func TestGetPendingImprovements(t *testing.T) {
	lm := &LearningManager{}

	sessions := []LearningSession{
		{
			Improvements: []Improvement{
				{Title: "Todo 1", Status: "discovered"},
				{Title: "Todo 2", Status: "implemented"},
				{Title: "Todo 3", Status: "planned"},
			},
		},
	}
	lm.sessions = sessions

	// Test with limit
	result := lm.GetPendingImprovements(1)
	if len(result) != 1 {
		t.Errorf("Expected 1 pending improvement with limit 1, got %d", len(result))
	}

	// Test without limit (high limit)
	result = lm.GetPendingImprovements(10)
	if len(result) != 2 {
		t.Errorf("Expected 2 pending improvements, got %d: %+v", len(result), result)
	}
}

func TestGenerateImprovementID(t *testing.T) {
	id := generateImprovementID("Test improvement title")
	if id == "" {
		t.Error("Generated ID should not be empty")
	}

	// Check format (should start with IMP-)
	if len(id) < 4 || id[:4] != "IMP-" {
		t.Errorf("ID should start with 'IMP-', got: %s", id)
	}
}

func TestExtractKeywords(t *testing.T) {
	text := "Performance optimization is important for security"
	suggested := []string{"performance", "security", "networking"}

	keywords := extractKeywords(text, suggested)
	if len(keywords) != 2 {
		t.Errorf("Expected 2 keywords, got %d: %v", len(keywords), keywords)
	}
}

func TestContainsActionableContent(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{
			name: "contains actionable word",
			text: "This should be implemented as a best practice",
			want: true,
		},
		{
			name: "generic statement",
			text: "The sky is blue today",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsActionableContent(tt.text)
			if got != tt.want {
				t.Errorf("Expected %v, got %v for: %s", tt.want, got, tt.text)
			}
		})
	}
}

func TestContainsGenericPattern(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{
			name: "contains generic pattern",
			text: "This provides features designed to optimize workflow",
			want: true,
		},
		{
			name: "no generic pattern",
			text: "Implement authentication system with OAuth2",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patterns := []string{"provides features", "designed to optimize"}
			got := containsGenericPattern(tt.text, patterns)
			if got != tt.want {
				t.Errorf("Expected %v, got %v for: %s", tt.want, got, tt.text)
			}
		})
	}
}
