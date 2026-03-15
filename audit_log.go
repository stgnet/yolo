package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// AuditEntry represents a single audit log entry
type AuditEntry struct {
	Timestamp string `json:"timestamp"`
	Action    string `json:"action"`
	Path      string `json:"path,omitempty"`
	Details   string `json:"details,omitempty"`
	Operator  string `json:"operator"` // In autonomous mode: "Yolo"
	Success   bool   `json:"success"`
}

const auditLogPath = ".audit_log.json"

// GetAuditLog loads existing audit entries from file
func GetAuditLog() ([]AuditEntry, error) {
	data, err := os.ReadFile(auditLogPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []AuditEntry{}, nil
		}
		return nil, err
	}

	var entries []AuditEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// SaveAuditLog persists audit entries to file
func SaveAuditLog(entries []AuditEntry) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal audit log: %w", err)
	}
	return os.WriteFile(auditLogPath, data, 0644)
}

// LogDestructiveAction adds a new audit entry for a destructive operation
func LogDestructiveAction(action string, path string, details string, success bool) {
	entry := AuditEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Action:    action,
		Path:      path,
		Details:   details,
		Operator:  "Yolo",
		Success:   success,
	}

	entries, err := GetAuditLog()
	if err != nil {
		fmt.Printf("ERROR: Failed to load audit log: %v\n", err)
		return
	}

	entries = append(entries, entry)

	if err := SaveAuditLog(entries); err != nil {
		fmt.Printf("ERROR: Failed to save audit log: %v\n", err)
		return
	}

	// Log to stdout as well for immediate visibility
	logMessage := fmt.Sprintf("[AUDIT] Action: %s | Path: %s | Details: %s | Success: %t",
		action, path, details, success)
	fmt.Println(logMessage)
}

// GetRecentAuditEntries returns the last N audit entries (default 10)
func GetRecentAuditEntries(limit int) []AuditEntry {
	entries, err := GetAuditLog()
	if err != nil || len(entries) == 0 {
		return nil
	}

	start := 0
	if len(entries) > limit {
		start = len(entries) - limit
	}
	return entries[start:]
}

// PrintAuditSummary prints a summary of recent audit activity
func PrintAuditSummary(limit int) {
	entries := GetRecentAuditEntries(limit)
	if len(entries) == 0 {
		fmt.Println("No audit entries found")
		return
	}

	fmt.Printf("\n=== Recent Audit Log (Last %d Entries) ===\n", limit)
	for _, entry := range entries {
		status := "✓"
		if !entry.Success {
			status = "✗"
		}
		fmt.Printf("[%s] %s | %s | %s\n", status, entry.Timestamp, entry.Action, entry.Path)
		if entry.Details != "" {
			fmt.Printf("    Details: %s\n", entry.Details)
		}
	}
	fmt.Println("===========================================")
}
