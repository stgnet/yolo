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
			WebQuery:        "autonomous AI agent best practices 2025 implementation patterns tool use memory state management error handling",
			RedditSubreddit: "LocalLLaMA",
			RedditSearch:    "autonomous agent implementation tool use patterns state management",
			Keywords:        []string{"autonomous", "planning", "memory", "tools", "multi-agent", "implementation", "pattern", "state"},
		},
		{
			Category:        "LLM Tool Integration",
			WebQuery:        "LLM function calling implementation patterns error handling context limits timeout retry 2025",
			RedditSubreddit: "LocalLLaMA",
			RedditSearch:    "function calling implementation best practices error handling timeout",
			Keywords:        []string{"function calling", "tool use", "context management", "error handling", "implementation", "retry"},
		},
		{
			Category:        "Developer Experience",
			WebQuery:        "AI coding assistant developer productivity automation workflow features implementation 2025",
			RedditSubreddit: "golang",
			RedditSearch:    "Go AI tools productivity automation best practices workflow",
			Keywords:        []string{"developer experience", "productivity", "automation", "workflow", "implementation"},
		},
		{
			Category:        "Testing & Evaluation",
			WebQuery:        "AI agent testing evaluation frameworks benchmarking performance metrics regression 2025",
			RedditSubreddit: "MachineLearning",
			RedditSearch:    "testing AI agents evaluation benchmarks implementation regression",
			Keywords:        []string{"testing", "evaluation", "benchmarking", "metrics", "regression"},
		},
		{
			Category:        "Go Performance",
			WebQuery:        "Go concurrent programming patterns performance optimization race conditions deadlock 2025",
			RedditSubreddit: "golang",
			RedditSearch:    "Go concurrency patterns performance best practices race condition deadlock",
			Keywords:        []string{"performance", "concurrency", "optimization", "race condition", "goroutine"},
		},
		{
			Category:        "Security & Reliability",
			WebQuery:        "AI agent security sandboxing file system access safe path validation injection prevention 2025",
			RedditSubreddit: "security",
			RedditSearch:    "AI security sandboxing file access best practices injection",
			Keywords:        []string{"security", "sandboxing", "file access", "validation", "safe"},
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

	// Remove duplicates based on title similarity
	session.Improvements = lm.removeDuplicateImprovements(session.Improvements)

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

	// Filter out generic encyclopedia/intro content
	genericPatterns := []string{
		" is a ", " refers to", "in other words", "etymology", "see also",
		"wikipedia", "encyclopedia", "introduction to", "overview of",
		"according to", "source:", "url:", "https://", "http://",
	}

	// Parse the structured JSON output from web_search
	// Look for specific sections and extract meaningful content

	// Extract instant answers (usually high-quality summaries)
	if strings.Contains(result, "Instant Answer") {
		start := strings.Index(result, "Instant Answer:")
		if start != -1 {
			start += len("Instant Answer:")
			end := strings.Index(result[start:], "\n\n")
			if end == -1 {
				end = len(result) - start
			}
			content := strings.TrimSpace(result[start : start+end])
			// Validate: must be substantial, relevant, and contain actionable insights
			if len(content) > 100 && !containsGenericPattern(content, genericPatterns) &&
				lm.isRelevant(content, area.Keywords) && containsActionableContent(content) {
				imp := lm.createImprovement(area, content, "web", "", "instant_answer")
				if imp != nil {
					improvements = append(improvements, *imp)
				}
			}
		}
	}

	// Extract related topics (often contain specific recommendations)
	if strings.Contains(result, "Related Topics:") {
		start := strings.Index(result, "Related Topics:")
		if start != -1 {
			end := strings.Index(result[start:], "\n\n")
			if end == -1 {
				end = len(result) - start
			}
			topicSection := result[start : start+end]
			// Split into individual topics and process each
			topics := strings.Split(topicSection, "\n")
			for _, topic := range topics {
				topic = strings.TrimSpace(topic)
				// Remove "Related Topics:" header and bullet points
				topic = strings.TrimPrefix(topic, "Related Topics:")
				topic = strings.TrimPrefix(topic, "• ")
				topic = strings.TrimSpace(topic)

				if len(topic) > 50 && !containsGenericPattern(topic, genericPatterns) &&
					lm.isRelevant(topic, area.Keywords) {
					description := fmt.Sprintf("%s. This is a related topic worth exploring for implementation.", topic)
					imp := lm.createImprovement(area, description, "web", "", "related_topic")
					if imp != nil {
						improvements = append(improvements, *imp)
					}
				}
			}
		}
	}

	// Extract abstract snippets (usually contain actionable information)
	lines := strings.Split(result, "\n")
	inAbstracts := false
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "Abstract:") || strings.Contains(line, "abstract:") {
			inAbstracts = true
			continue
		}

		if inAbstracts && strings.HasPrefix(line, "  •") {
			// This is an abstract snippet - often contains specific recommendations
			snippet := strings.TrimPrefix(line, "  • ")
			snippet = strings.TrimSpace(snippet)

			// Filter: must be substantial, relevant, actionable, and not generic
			if len(snippet) > 80 && !containsGenericPattern(snippet, genericPatterns) &&
				lm.isRelevant(snippet, area.Keywords) && containsActionableContent(snippet) {
				imp := lm.createImprovement(area, truncateText(snippet, 500), "web", "", "abstract")
				if imp != nil {
					improvements = append(improvements, *imp)
				}
			}
		}

		// Reset flag when we hit a new section
		if line != "" && !strings.HasPrefix(line, "  ") &&
			(strings.Contains(line, ":") || strings.Contains(line, "##")) {
			inAbstracts = false
		}
	}

	return improvements
}

// extractImprovementsFromReddit parses Reddit results into improvements
func (lm *LearningManager) extractImprovementsFromReddit(area ResearchArea, result string) []Improvement {
	var improvements []Improvement

	// Filter out low-quality or generic content
	genericPatterns := []string{
		"edit:", "thanks for sharing", "upvote if you agree",
		"just wanted to say", "thought you might like",
		"wikipedia", "according to",
	}

	// Reddit results are JSON-formatted - extract meaningful content
	// Look for post titles and body content that contain actionable insights

	lines := strings.Split(result, "\n")
	var currentTitle string
	var currentBody strings.Builder
	inPost := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Detect start of a new post title
		if strings.HasPrefix(line, "• ") && len(line) > 20 && len(line) < 500 {
			// Save previous post if it has substantial content
			if inPost && currentBody.Len() > 150 {
				bodyContent := currentBody.String()
				fullContent := fmt.Sprintf("%s: %s", currentTitle, bodyContent)

				if !containsGenericPattern(fullContent, genericPatterns) &&
					lm.isRelevant(fullContent, area.Keywords) && containsActionableContent(fullContent) {
					imp := lm.createImprovement(area, truncateText(fullContent, 500), "reddit",
						fmt.Sprintf("https://reddit.com/r/%s", area.RedditSubreddit), "reddit_post")
					if imp != nil {
						improvements = append(improvements, *imp)
					}
				}
			}

			// Start new post
			currentTitle = strings.TrimPrefix(line, "• ")
			currentBody.Reset()
			inPost = true
		} else if inPost && !strings.HasPrefix(line, "•") {
			// Add to current post body (if it's not a new title)
			if len(line) > 10 {
				currentBody.WriteString(" " + line)
			}
		}
	}

	// Don't forget the last post
	if inPost && currentBody.Len() > 150 {
		bodyContent := currentBody.String()
		fullContent := fmt.Sprintf("%s: %s", currentTitle, bodyContent)

		if !containsGenericPattern(fullContent, genericPatterns) &&
			lm.isRelevant(fullContent, area.Keywords) && containsActionableContent(fullContent) {
			imp := lm.createImprovement(area, truncateText(fullContent, 500), "reddit",
				fmt.Sprintf("https://reddit.com/r/%s", area.RedditSubreddit), "reddit_post")
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
		title := sentences[0]
		if !strings.HasSuffix(title, ".") {
			title += "."
		}
		return title
	}

	if len(content) > 100 {
		return content[:97] + "..*"
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
	suffix := "...*"
	return text[:maxLen-len(suffix)] + suffix
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

// containsGenericPattern checks if text contains generic/encyclopedic patterns
func containsGenericPattern(text string, patterns []string) bool {
	textLower := strings.ToLower(text)
	for _, pattern := range patterns {
		if strings.Contains(textLower, pattern) {
			return true
		}
	}
	return false
}

// containsActionableContent checks if text has actionable improvement content
func containsActionableContent(text string) bool {
	actionableKeywords := []string{
		"should", "recommend", "best practice", "improve", "optimize",
		"implement", "use case", "pattern", "solution", "approach",
		"tip", "trick", "hack", "feature", "enhancement",
		"consider", "important", "critical", "essential", "necessary",
		"method", "technique", "strategy", "framework", "architecture",
		"handle", "manage", "process", "validate", "verify",
		"performance", "scalability", "reliability", "efficiency",
	}
	textLower := strings.ToLower(text)
	for _, kw := range actionableKeywords {
		if strings.Contains(textLower, kw) {
			return true
		}
	}
	return false
}

// removeDuplicateImprovements removes duplicate or near-duplicate improvements
func (lm *LearningManager) removeDuplicateImprovements(improvements []Improvement) []Improvement {
	if len(improvements) <= 1 {
		return improvements
	}

	var unique []Improvement
	for i, imp := range improvements {
		isDuplicate := false
		for j := 0; j < i; j++ {
			// Simple duplicate detection: if titles are very similar (>70% overlap)
			if lm.similarity(improvements[j].Title, imp.Title) > 0.7 {
				isDuplicate = true
				break
			}
		}
		if !isDuplicate {
			unique = append(unique, imp)
		}
	}

	return unique
}

// similarity calculates string similarity (simple Jaccard-like metric)
func (lm *LearningManager) similarity(s1, s2 string) float64 {
	words1 := strings.Fields(strings.ToLower(s1))
	words2 := strings.Fields(strings.ToLower(s2))

	if len(words1) == 0 || len(words2) == 0 {
		return 0
	}

	// Count common words
	common := 0
	for _, w1 := range words1 {
		for _, w2 := range words2 {
			if w1 == w2 {
				common++
				break
			}
		}
	}

	total := len(words1) + len(words2)
	return float64(common*2) / float64(total)
}
