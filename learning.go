// Learning system for autonomous self-improvement through internet research
// This module uses web_search and reddit tools to discover improvement opportunities

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

const learningHistoryFile = ".yolo_learning.json"

// LearningSession represents a complete learning session
type LearningSession struct {
	Timestamp    time.Time     `json:"timestamp"`
	Duration     int           `json:"duration_seconds"`
	Improvements []Improvement `json:"improvements"`
	Trends       []string      `json:"trends"`
	Implemented  []string      `json:"implemented,omitempty"` // IDs of implemented improvements
}

// Improvement represents a discovered opportunity for self-improvement
type Improvement struct {
	ID                  string    `json:"id"`
	Category            string    `json:"category"`
	Priority            int       `json:"priority"` // 1-5, 5 being highest
	Title               string    `json:"title"`
	Description         string    `json:"description"`
	Source              string    `json:"source"` // "web" or "reddit"
	URL                 string    `json:"url,omitempty"`
	Keywords            []string  `json:"keywords,omitempty"`
	ImplementationNotes string    `json:"implementation_notes,omitempty"`
	Status              string    `json:"status"` // "discovered", "planned", "in_progress", "implemented", "rejected"
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at,omitempty"`
}

// LearningManager handles autonomous learning and improvement tracking
type LearningManager struct {
	historyPath string
	sessions    []LearningSession
	executor    *ToolExecutor // Reference to tool executor for web/reddit calls
}

// NewLearningManager creates a new learning manager
func NewLearningManager(baseDir string, executor *ToolExecutor) *LearningManager {
	return &LearningManager{
		historyPath: baseDir + "/" + learningHistoryFile,
		executor:    executor,
	}
}

// LoadHistory loads the learning history from disk
func (lm *LearningManager) LoadHistory() error {
	data, err := os.ReadFile(lm.historyPath)
	if err != nil {
		if os.IsNotExist(err) {
			lm.sessions = []LearningSession{}
			return nil
		}
		return err
	}

	err = json.Unmarshal(data, &lm.sessions)
	if err != nil {
		return fmt.Errorf("failed to parse learning history: %v", err)
	}
	return nil
}

// SaveHistory saves the learning history to disk
func (lm *LearningManager) SaveHistory() error {
	data, err := json.MarshalIndent(lm.sessions, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(lm.historyPath, data, 0644)
}

// ResearchAndLearn performs autonomous research to discover improvements
func (lm *LearningManager) ResearchAndLearn() (*LearningSession, error) {
	startTime := time.Now()

	session := &LearningSession{
		Timestamp:    startTime,
		Improvements: []Improvement{},
		Trends:       []string{},
	}

	// Define research areas based on YOLO's capabilities
	researchAreas := []ResearchArea{
		{
			Category:        "AI Agent Architecture",
			WebQuery:        "AI agent architecture patterns 2025 best practices",
			RedditSubreddit: "ArtificialIntelligence",
			RedditSearch:    "AI agent design patterns",
			Keywords:        []string{"autonomous", "planning", "memory", "tools", "multi-agent"},
		},
		{
			Category:        "LLM Tool Integration",
			WebQuery:        "LLM function calling tool use best practices 2025",
			RedditSubreddit: "LocalLLaMA",
			RedditSearch:    "tool use function calling patterns",
			Keywords:        []string{"function calling", "tool use", "context management"},
		},
		{
			Category:        "Developer Experience",
			WebQuery:        "AI coding assistant features 2025 developer productivity",
			RedditSubreddit: "programming",
			RedditSearch:    "AI coding tools best features",
			Keywords:        []string{"developer experience", "productivity", "automation"},
		},
		{
			Category:        "Testing & Evaluation",
			WebQuery:        "AI agent testing evaluation frameworks 2025",
			RedditSubreddit: "MachineLearning",
			RedditSearch:    "testing AI agents evaluation",
			Keywords:        []string{"testing", "evaluation", "benchmarking"},
		},
		{
			Category:        "Go Performance",
			WebQuery:        "Go performance optimization patterns 2025 concurrency",
			RedditSubreddit: "golang",
			RedditSearch:    "Go performance best practices",
			Keywords:        []string{"performance", "concurrency", "optimization"},
		},
	}

	// Research each area
	for _, area := range researchAreas {
		improvements, err := lm.researchArea(area, session)
		if err != nil {
			fmt.Printf("Warning: Error researching %s: %v\n", area.Category, err)
			continue
		}
		session.Improvements = append(session.Improvements, improvements...)
	}

	// Analyze trends across all findings
	session.Trends = lm.analyzeTrends(session.Improvements)

	// Calculate duration
	session.Duration = int(time.Since(startTime).Seconds())

	return session, nil
}

// ResearchArea defines an area to research
type ResearchArea struct {
	Category        string   `json:"category"`
	WebQuery        string   `json:"web_query"`
	RedditSubreddit string   `json:"reddit_subreddit"`
	RedditSearch    string   `json:"reddit_search"`
	Keywords        []string `json:"keywords"`
}

// researchArea performs research on a specific area and returns improvements
func (lm *LearningManager) researchArea(area ResearchArea, session *LearningSession) ([]Improvement, error) {
	var improvements []Improvement

	// Search the web for insights using tool executor
	webResult := lm.executor.webSearch(map[string]any{
		"query": area.WebQuery,
		"count": 5,
	})

	// Parse web search results and extract improvements
	improvements = append(improvements, lm.extractImprovementsFromWeb(area, webResult)...)

	// Search Reddit for community insights
	redditResult := lm.executor.reddit(map[string]any{
		"action":    "subreddit",
		"subreddit": area.RedditSubreddit,
		"limit":     10,
	})

	// Parse Reddit results and extract improvements
	improvements = append(improvements, lm.extractImprovementsFromReddit(area, redditResult)...)

	return improvements, nil
}

// extractImprovementsFromWeb parses web search results into improvements
func (lm *LearningManager) extractImprovementsFromWeb(area ResearchArea, result string) []Improvement {
	var improvements []Improvement

	// Simple text-based extraction from the JSON-like result
	// Look for key sections in the output

	if strings.Contains(result, "Instant Answer") {
		// Extract instant answer content
		start := strings.Index(result, "Instant Answer:")
		if start != -1 {
			start += len("Instant Answer:")
			end := strings.Index(result[start:], "\n\n")
			if end == -1 {
				end = len(result) - start
			}
			content := strings.TrimSpace(result[start : start+end])
			if len(content) > 50 && lm.isRelevant(content, area.Keywords) {
				imp := lm.createImprovement(area, content, "web", "", "instant_answer")
				if imp != nil {
					improvements = append(improvements, *imp)
				}
			}
		}
	}

	// Extract related topics and results similarly
	lines := strings.Split(result, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 100 && lm.isRelevant(line, area.Keywords) {
			imp := lm.createImprovement(area, truncateText(line, 500), "web", "", "search_result")
			if imp != nil {
				improvements = append(improvements, *imp)
			}
		}
	}

	return improvements
}

// extractImprovementsFromReddit parses Reddit results into improvements
func (lm *LearningManager) extractImprovementsFromReddit(area ResearchArea, result string) []Improvement {
	var improvements []Improvement

	// Parse the Reddit JSON response structure
	lines := strings.Split(result, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 100 && lm.isRelevant(line, area.Keywords) {
			url := fmt.Sprintf("https://reddit.com/r/%s", area.RedditSubreddit)
			imp := lm.createImprovement(area, truncateText(line, 500), "reddit", url, "reddit_post")
			if imp != nil {
				improvements = append(improvements, *imp)
			}
		}
	}

	return improvements
}

// createImprovement creates a new Improvement struct from content
func (lm *LearningManager) createImprovement(area ResearchArea, content string, source, url, sourceType string) *Improvement {
	if len(content) < 30 {
		return nil
	}

	title := lm.generateTitle(content, area.Category)

	return &Improvement{
		ID:          generateImprovementID(title),
		Category:    area.Category,
		Priority:    lm.calculatePriority(content, sourceType),
		Title:       title,
		Description: content,
		Source:      source,
		URL:         url,
		Keywords:    extractKeywords(content, area.Keywords),
		Status:      "discovered",
		CreatedAt:   time.Now(),
	}
}

// generateTitle creates a concise title from content
func (lm *LearningManager) generateTitle(content string, category string) string {
	// Take first sentence or first 100 chars
	sentences := strings.Split(content, ". ")
	if len(sentences) > 0 && len(sentences[0]) < 150 {
		return sentences[0] + "."
	}

	if len(content) > 100 {
		return content[:97] + "...*"
	}
	return content
}

// calculatePriority determines the priority based on source and content
func (lm *LearningManager) calculatePriority(content string, sourceType string) int {
	priority := 3 // default medium

	// Higher priority for instant answers (often high-quality summaries)
	if sourceType == "instant_answer" {
		priority = 5
	} else if sourceType == "reddit" && strings.Contains(strings.ToLower(content), "best practice") {
		priority = 4
	} else if strings.Contains(strings.ToLower(content), "performance") ||
		strings.Contains(strings.ToLower(content), "security") {
		priority = 4
	}

	return priority
}

// isRelevant checks if content is relevant to the keywords
func (lm *LearningManager) isRelevant(content string, keywords []string) bool {
	contentLower := strings.ToLower(content)
	for _, keyword := range keywords {
		if strings.Contains(contentLower, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

// analyzeTrends identifies common trends across improvements
func (lm *LearningManager) analyzeTrends(improvements []Improvement) []string {
	trendMap := make(map[string]int)

	for _, imp := range improvements {
		for _, keyword := range imp.Keywords {
			trendMap[keyword]++
		}
	}

	var trends []string
	for keyword, count := range trendMap {
		if count >= 2 { // Appears in multiple improvements
			trends = append(trends, fmt.Sprintf("%s (mentioned %d times)", keyword, count))
		}
	}

	return trends
}

// SaveSession saves a completed learning session
func (lm *LearningManager) SaveSession(session *LearningSession) error {
	lm.sessions = append(lm.sessions, *session)
	return lm.SaveHistory()
}

// GetPendingImprovements returns improvements that haven't been implemented yet
func (lm *LearningManager) GetPendingImprovements(limit int) []Improvement {
	var pending []Improvement

	for _, session := range lm.sessions {
		for _, imp := range session.Improvements {
			if imp.Status != "implemented" && imp.Status != "rejected" {
				pending = append(pending, imp)
				if len(pending) >= limit {
					return pending
				}
			}
		}
	}

	return pending
}

// Helper functions

func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen-3] + "...*"
}

func generateImprovementID(title string) string {
	// Simple hash-based ID generation
	hash := 0
	for _, c := range title {
		hash = hash*31 + int(c)
	}
	if hash < 0 {
		hash = -hash
	}
	return fmt.Sprintf("IMP-%d", hash%100000)
}

func extractKeywords(text string, suggested []string) []string {
	var keywords []string
	textLower := strings.ToLower(text)

	for _, kw := range suggested {
		if strings.Contains(textLower, strings.ToLower(kw)) {
			keywords = append(keywords, kw)
		}
	}

	return keywords
}
