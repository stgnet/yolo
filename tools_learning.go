package main

import (
	"fmt"
	"strings"
	"time"
)

func (t *ToolExecutor) learn(args map[string]any) string {
	// Initialize learning manager
	lm := NewLearningManager(".", t)

	// Load existing history
	if err := lm.LoadHistory(); err != nil {
		return fmt.Sprintf("Error loading learning history: %v", err)
	}

	// Check if we have recent learning sessions (within last 1 hour for testing)
	now := time.Now()
	for _, session := range lm.sessions {
		if now.Sub(session.Timestamp).Hours() < 1 {
			return fmt.Sprintf("Learning already performed recently at %s. Found %d improvements.\n\n",
				session.Timestamp.Format("Jan 2, 2006 15:04"), len(session.Improvements))
		}
	}

	// Perform research and learning
	fmt.Println("\n🔍 Starting autonomous research for self-improvement...")
	session, err := lm.ResearchAndLearn()
	if err != nil {
		return fmt.Sprintf("Error during research: %v", err)
	}

	// Save the session
	if err := lm.SaveSession(session); err != nil {
		return fmt.Sprintf("Research completed but error saving: %v", err)
	}

	// Format results
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n📚 Learning Session Completed\n"))
	sb.WriteString(fmt.Sprintf("⏱️  Duration: %d seconds\n", session.Duration))
	sb.WriteString(fmt.Sprintf("📊 Improvements Discovered: %d\n\n", len(session.Improvements)))

	// Show top improvements by priority
	highPriority := filterImprovementsByPriority(session.Improvements, 4)
	if len(highPriority) > 0 {
		sb.WriteString("🔥 High Priority Improvements (Priority 4-5):\n")
		for i, imp := range highPriority {
			if i >= 3 {
				break // Show top 3
			}
			sb.WriteString(fmt.Sprintf("  %d. [%s] %s\n", i+1, imp.Category, imp.Title))
			sb.WriteString(fmt.Sprintf("     Source: %s | Status: %s\n", imp.Source, imp.Status))
		}
		sb.WriteString("\n")
	}

	// Show trends
	if len(session.Trends) > 0 {
		sb.WriteString("📈 Identified Trends:\n")
		for _, trend := range session.Trends {
			sb.WriteString(fmt.Sprintf("  • %s\n", trend))
		}
		sb.WriteString("\n")
	}

	// Summary by category
	categoryCounts := make(map[string]int)
	for _, imp := range session.Improvements {
		categoryCounts[imp.Category]++
	}

	sb.WriteString("📋 Improvements by Category:\n")
	for category, count := range categoryCounts {
		sb.WriteString(fmt.Sprintf("  • %s: %d\n", category, count))
	}

	sb.WriteString("\n💡 All improvements saved to .yolo_learning.json\n")

	// Show pending improvements from previous sessions
	pending := lm.GetPendingImprovements(5)
	if len(pending) > 0 {
		sb.WriteString(fmt.Sprintf("\n📝 %d pending improvements from previous sessions\n", len(pending)))

		// Offer to implement top improvements
		implCount := 2 // Limit implementation count
		fmt.Printf("💡 Found %d pending improvements. Implementing top %d...\n", len(pending), implCount)
		if err := lm.ImplementTopImprovements(implCount); err != nil {
			return fmt.Sprintf("Learning completed but implementation failed: %v", err)
		}
	}

	return sb.String()
}

// filterImprovementsByPriority returns improvements with priority >= minPriority
func filterImprovementsByPriority(improvements []Improvement, minPriority int) []Improvement {
	var filtered []Improvement
	for _, imp := range improvements {
		if imp.Priority >= minPriority {
			filtered = append(filtered, imp)
		}
	}
	return filtered
}

// implementTop improvements tool - executes high-priority improvements
func (t *ToolExecutor) implement(args map[string]any) string {
	// Initialize learning manager
	lm := NewLearningManager(".", t)

	// Load existing history
	if err := lm.LoadHistory(); err != nil {
		return fmt.Sprintf("Error loading learning history: %v", err)
	}

	// Get implementation count (default 2 if not specified)
	implCount := 2
	if count, ok := args["count"].(int); ok {
		implCount = count
	}

	fmt.Printf("\n🔧 Implementing top %d improvements...\n", implCount)

	err := lm.ImplementTopImprovements(implCount)
	if err != nil {
		return fmt.Sprintf("Error during implementation: %v", err)
	}

	pending := lm.GetPendingImprovements(5)
	var sb strings.Builder
	sb.WriteString("\n✅ Implementation Complete\n")
	sb.WriteString(fmt.Sprintf("   Pending improvements remaining: %d\n", len(pending)))

	return sb.String()
}
