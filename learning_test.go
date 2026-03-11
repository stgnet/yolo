package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func TestNewLearningManager(t *testing.T) {
	executor := &ToolExecutor{}
	lm := NewLearningManager(".", executor)

	if lm == nil {
		t.Fatal("Expected non-nil LearningManager")
	}
	if lm.executor != executor {
		t.Error("Expected executor to be set")
	}
	if !strings.Contains(lm.historyPath, learningHistoryFile) {
		t.Errorf("Expected historyPath to contain %s, got %s", learningHistoryFile, lm.historyPath)
	}
}

func TestLearningManagerLoadHistory(t *testing.T) {
	executor := &ToolExecutor{}

	// Create a unique temp file to avoid conflicts with existing files
	tmpFile := ".yolo_learning_test_" + t.Name() + ".json"
	defer os.Remove(tmpFile)

	// Test with non-existent file
	lm := NewLearningManager(".", executor)
	lm.historyPath = tmpFile
	err := lm.LoadHistory()
	if err != nil {
		t.Errorf("Expected no error for non-existent file, got %v", err)
	}
	if len(lm.sessions) != 0 {
		t.Errorf("Expected empty sessions, got %d", len(lm.sessions))
	}

	// Test with valid JSON
	testData := `[]`
	os.WriteFile(tmpFile, []byte(testData), 0644)

	lm.sessions = nil // Reset for next test
	err = lm.LoadHistory()
	if err != nil {
		t.Errorf("Expected no error for valid empty JSON, got %v", err)
	}

	// Test with session data
	session := LearningSession{
		Timestamp:    time.Now(),
		Duration:     10,
		Improvements: []Improvement{},
	}
	testData2, _ := json.Marshal(session)
	os.WriteFile(tmpFile, []byte("["+string(testData2)+"]"), 0644)

	lm.sessions = nil
	err = lm.LoadHistory()
	if err != nil {
		t.Errorf("Expected no error for valid session JSON, got %v", err)
	}
	if len(lm.sessions) != 1 {
		t.Errorf("Expected 1 session, got %d", len(lm.sessions))
	}

	// Test with invalid JSON
	os.WriteFile(tmpFile, []byte("{invalid"), 0644)
	err = lm.LoadHistory()
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestLearningManagerSaveHistory(t *testing.T) {
	executor := &ToolExecutor{}
	lm := NewLearningManager(".", executor)
	tmpFile := ".yolo_learning_test.json"
	lm.historyPath = tmpFile
	defer os.Remove(tmpFile)

	session := LearningSession{
		Timestamp: time.Now(),
		Duration:  5,
		Improvements: []Improvement{
			{
				ID:          "IMP-1",
				Category:    "Test",
				Priority:    5,
				Title:       "Test Improvement",
				Description: "Test description",
				Source:      "web",
				Status:      "discovered",
			},
		},
	}
	lm.sessions = append(lm.sessions, session)

	err := lm.SaveHistory()
	if err != nil {
		t.Errorf("Expected no error saving history, got %v", err)
	}

	// Verify file was created
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "Test Improvement") {
		t.Error("Expected saved file to contain improvement title")
	}
}

func TestResearchAndLearn(t *testing.T) {
	executor := &ToolExecutor{}
	lm := NewLearningManager(".", executor)

	// This is an integration test that makes actual web/reddit calls
	// Skip if we don't want to make network requests
	t.Skip("Skipping integration test that makes network requests")

	session, err := lm.ResearchAndLearn()
	if err != nil {
		t.Fatalf("ResearchAndLearn failed: %v", err)
	}

	if session == nil {
		t.Fatal("Expected non-nil session")
	}

	if len(session.Improvements) == 0 {
		t.Log("Warning: No improvements discovered (may be expected if searches returned no relevant results)")
	}
}

func TestExtractImprovementsFromWeb(t *testing.T) {
	executor := &ToolExecutor{}
	lm := NewLearningManager(".", executor)

	area := ResearchArea{
		Category: "AI Agent Architecture",
		Keywords: []string{"autonomous", "planning"},
	}

	// Test with instant answer
	result := `{
	"instant_answer": "Autonomous agents use planning and memory systems"
}`

	improvements := lm.extractImprovementsFromWeb(area, result)
	if len(improvements) == 0 {
		t.Log("No improvements extracted from mock web result (may need better mock data)")
	}
}

func TestExtractImprovementsFromReddit(t *testing.T) {
	executor := &ToolExecutor{}
	lm := NewLearningManager(".", executor)

	area := ResearchArea{
		Category:        "AI Agent Architecture",
		RedditSubreddit: "MachineLearning",
		Keywords:        []string{"autonomous"},
	}

	result := `{"posts": [{"title": "Autonomous agent design patterns"}]}`
	improvements := lm.extractImprovementsFromReddit(area, result)
	if len(improvements) == 0 {
		t.Log("No improvements extracted from mock Reddit result (may need better mock data)")
	}
}

func TestCreateImprovement(t *testing.T) {
	executor := &ToolExecutor{}
	lm := NewLearningManager(".", executor)

	area := ResearchArea{
		Category: "Test Category",
		Keywords: []string{"test"},
	}

	content := "This is a test improvement description that should create a valid improvement."
	imp := lm.createImprovement(area, content, "web", "http://example.com", "search_result")

	if imp == nil {
		t.Fatal("Expected non-nil improvement")
	}

	if imp.Category != area.Category {
		t.Errorf("Expected category %s, got %s", area.Category, imp.Category)
	}

	if imp.Title == "" {
		t.Error("Expected non-empty title")
	}

	if imp.Status != "discovered" {
		t.Errorf("Expected status 'discovered', got %s", imp.Status)
	}

	if len(imp.Keywords) == 0 {
		t.Error("Expected keywords to be extracted")
	}
}

func TestCreateImprovement_ShortContent(t *testing.T) {
	executor := &ToolExecutor{}
	lm := NewLearningManager(".", executor)

	area := ResearchArea{Category: "Test"}
	content := "Too short"
	imp := lm.createImprovement(area, content, "web", "", "")

	if imp != nil {
		t.Error("Expected nil improvement for short content")
	}
}

func TestGenerateTitle(t *testing.T) {
	executor := &ToolExecutor{}
	lm := NewLearningManager(".", executor)

	// Test single sentence
	content1 := "This is a test."
	title1 := lm.generateTitle(content1, "Test")
	if title1 != "This is a test." {
		t.Errorf("Expected 'This is a test.', got %q", title1)
	}

	// Test multiple sentences
	content2 := "First sentence. Second sentence."
	title2 := lm.generateTitle(content2, "Test")
	if title2 != "First sentence." {
		t.Errorf("Expected 'First sentence.', got %q", title2)
	}

	// Test long content (should truncate)
	longContent := strings.Repeat("Word ", 50) + "end"
	title3 := lm.generateTitle(longContent, "Test")
	if len(title3) > 105 {
		t.Errorf("Expected truncated title <= 105 chars, got %d", len(title3))
	}
}

func TestCalculatePriority(t *testing.T) {
	executor := &ToolExecutor{}
	lm := NewLearningManager(".", executor)

	// Instant answer gets highest priority
	priority1 := lm.calculatePriority("some content", "instant_answer")
	if priority1 != 5 {
		t.Errorf("Expected priority 5 for instant_answer, got %d", priority1)
	}

	// Best practice gets high priority
	priority2 := lm.calculatePriority("best practice example", "reddit")
	if priority2 != 4 {
		t.Errorf("Expected priority 4 for best practice, got %d", priority2)
	}

	// Performance/security gets high priority
	priority3 := lm.calculatePriority("performance optimization", "web")
	if priority3 != 4 {
		t.Errorf("Expected priority 4 for performance, got %d", priority3)
	}

	// Default is medium
	priority4 := lm.calculatePriority("regular content", "web")
	if priority4 != 3 {
		t.Errorf("Expected priority 3 (default), got %d", priority4)
	}
}

func TestIsRelevant(t *testing.T) {
	executor := &ToolExecutor{}
	lm := NewLearningManager(".", executor)

	keywords := []string{"autonomous", "planning"}
	content := "This is about autonomous agents"

	if !lm.isRelevant(content, keywords) {
		t.Error("Expected content to be relevant")
	}

	content2 := "Unrelated content"
	if lm.isRelevant(content2, keywords) {
		t.Error("Expected content to be not relevant")
	}
}

func TestAnalyzeTrends(t *testing.T) {
	executor := &ToolExecutor{}
	lm := NewLearningManager(".", executor)

	improvements := []Improvement{
		{Keywords: []string{"autonomous", "planning"}},
		{Keywords: []string{"autonomous", "memory"}},
		{Keywords: []string{"planning", "tools"}},
	}

	trends := lm.analyzeTrends(improvements)

	if len(trends) == 0 {
		t.Error("Expected to find trends")
	}

	// Check that "autonomous" and "planning" appear (they each appear twice)
	foundAutonomous := false
	foundPlanning := false
	for _, trend := range trends {
		if strings.Contains(trend, "autonomous") {
			foundAutonomous = true
		}
		if strings.Contains(trend, "planning") {
			foundPlanning = true
		}
	}

	if !foundAutonomous {
		t.Error("Expected to find 'autonomous' trend")
	}
	if !foundPlanning {
		t.Error("Expected to find 'planning' trend")
	}
}

func TestGetPendingImprovements(t *testing.T) {
	executor := &ToolExecutor{}
	lm := NewLearningManager(".", executor)

	lm.sessions = []LearningSession{
		{
			Improvements: []Improvement{
				{ID: "1", Status: "discovered"},
				{ID: "2", Status: "planned"},
				{ID: "3", Status: "implemented"},
				{ID: "4", Status: "rejected"},
			},
		},
	}

	pending := lm.GetPendingImprovements(10)

	if len(pending) != 2 {
		t.Errorf("Expected 2 pending improvements, got %d", len(pending))
	}

	statusMap := make(map[string]bool)
	for _, p := range pending {
		statusMap[p.Status] = true
	}

	if statusMap["implemented"] || statusMap["rejected"] {
		t.Error("Pending should not include implemented or rejected improvements")
	}
}

func TestGetPendingImprovements_Limit(t *testing.T) {
	executor := &ToolExecutor{}
	lm := NewLearningManager(".", executor)

	lm.sessions = []LearningSession{
		{
			Improvements: []Improvement{
				{ID: "1", Status: "discovered"},
				{ID: "2", Status: "discovered"},
				{ID: "3", Status: "discovered"},
			},
		},
	}

	pending := lm.GetPendingImprovements(2)

	if len(pending) != 2 {
		t.Errorf("Expected limit of 2, got %d", len(pending))
	}
}

func TestTruncateText(t *testing.T) {
	// Short text
	short := "Hello"
	result1 := truncateText(short, 50)
	if result1 != short {
		t.Errorf("Expected unchanged short text, got %q", result1)
	}

	// Long text
	long := strings.Repeat("x", 100)
	result2 := truncateText(long, 50)
	if len(result2) != 50 {
		t.Errorf("Expected truncated text to be 50 chars, got %d", len(result2))
	}
	if !strings.HasSuffix(result2, "..*") {
		t.Error("Expected truncated text to end with '..*'")
	}
}

func TestGenerateImprovementID(t *testing.T) {
	id1 := generateImprovementID("Test Title")
	id2 := generateImprovementID("Test Title")
	id3 := generateImprovementID("Different Title")

	if id1 != id2 {
		t.Error("Expected same title to produce same ID")
	}

	if id1 == id3 {
		t.Log("Warning: Different titles produced same ID (hash collision, but acceptable)")
	}

	if !strings.HasPrefix(id1, "IMP-") {
		t.Errorf("Expected ID to start with 'IMP-', got %q", id1)
	}
}

func TestExtractKeywords(t *testing.T) {
	text := "This is about autonomous planning and tools"
	suggested := []string{"autonomous", "planning", "memory"}

	keywords := extractKeywords(text, suggested)

	expectedCount := 2 // "autonomous" and "planning" should match
	if len(keywords) != expectedCount {
		t.Errorf("Expected %d keywords, got %d: %v", expectedCount, len(keywords), keywords)
	}
}

func TestFilterImprovementsByPriority(t *testing.T) {
	improvements := []Improvement{
		{ID: "1", Priority: 5},
		{ID: "2", Priority: 3},
		{ID: "3", Priority: 4},
		{ID: "4", Priority: 2},
	}

	filtered := filterImprovementsByPriority(improvements, 4)

	if len(filtered) != 2 {
		t.Errorf("Expected 2 improvements with priority >= 4, got %d", len(filtered))
	}

	for _, imp := range filtered {
		if imp.Priority < 4 {
			t.Errorf("Expected all filtered improvements to have priority >= 4, got %d", imp.Priority)
		}
	}
}

func TestResearchArea(t *testing.T) {
	executor := &ToolExecutor{}
	lm := NewLearningManager(".", executor)

	area := ResearchArea{
		Category:        "Test",
		WebQuery:        "test query",
		RedditSubreddit: "test",
		RedditSearch:    "test search",
		Keywords:        []string{"test"},
	}

	session := &LearningSession{}
	improvements, err := lm.researchArea(area, session)

	if err != nil {
		t.Logf("researchArea returned error (may be expected): %v", err)
	}

	t.Logf("Discovered %d improvements for test area", len(improvements))
}

func TestContainsGenericPattern(t *testing.T) {
	patterns := []string{" is a ", "wikipedia", "overview of"}

	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{"generic intro", "Grok is a AI assistant", true},
		{"wikipedia reference", "See Wikipedia for more", true},
		{"overview text", "This is an overview of the topic", true},
		{"actionable content", "You should implement this pattern to improve performance", false},
		{"specific implementation", "The recommended approach is to use goroutines", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsGenericPattern(tt.text, patterns)
			if result != tt.expected {
				t.Errorf("containsGenericPattern(%q) = %v, expected %v", tt.text, result, tt.expected)
			}
		})
	}
}

func TestContainsActionableContent(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{"recommendation", "We recommend using this pattern", true},
		{"best practice", "The best practice is to handle errors", true},
		{"implementation tip", "A useful implementation tip is to cache results", true},
		{"generic statement", "This is a statement about the topic", false},
		{"definition text", "This term refers to a concept", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsActionableContent(tt.text)
			if result != tt.expected {
				t.Errorf("containsActionableContent(%q) = %v, expected %v", tt.text, result, tt.expected)
			}
		})
	}
}
