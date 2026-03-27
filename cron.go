package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ─── Cron / Scheduler ────────────────────────────────────────────────
//
// CronManager provides a simple cron-style scheduler for YOLO. Schedules
// are persisted in .yolo/cron.json and survive restarts. A background
// goroutine checks for due schedules every 30 seconds.
//
// When a schedule fires, a callback is invoked (typically to inject a
// system message into the agent's history so it acts on the task).
//
// Cron expressions use 5 fields: minute hour day-of-month month day-of-week
// with standard cron syntax including * and */N step values.

// ─── Types ───────────────────────────────────────────────────────────

// CronSchedule is a single scheduled task.
type CronSchedule struct {
	ID        string `json:"id"`                 // unique identifier
	Name      string `json:"name"`               // human-readable name
	CronExpr  string `json:"cron_expr"`          // cron expression (5 fields: min hour dom month dow)
	Prompt    string `json:"prompt"`             // task prompt injected when schedule fires
	Enabled   bool   `json:"enabled"`            // whether the schedule is active
	CreatedAt string `json:"created_at"`         // RFC 3339
	LastRun   string `json:"last_run,omitempty"` // RFC 3339 of last execution
	NextRun   string `json:"next_run,omitempty"` // RFC 3339 of next planned execution
	RunCount  int    `json:"run_count"`          // total times fired
}

// CronData is the top-level structure for .yolo/cron.json.
type CronData struct {
	Version   int            `json:"version"`
	Schedules []CronSchedule `json:"schedules"`
}

// CronManager owns the schedule list and the background ticker.
type CronManager struct {
	yoloDir  string
	dataFile string
	Data     CronData
	onFire   func(schedule CronSchedule) // callback when a schedule fires
	stopCh   chan struct{}
	mu       sync.Mutex
}

// NewCronManager creates a manager that stores schedules in .yolo/cron.json.
func NewCronManager(yoloDir string) *CronManager {
	return &CronManager{
		yoloDir:  yoloDir,
		dataFile: filepath.Join(yoloDir, "cron.json"),
		Data:     CronData{Version: 1},
		stopCh:   make(chan struct{}),
	}
}

// ─── Persistence ─────────────────────────────────────────────────────

// Load reads cron.json from disk.
func (cm *CronManager) Load() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	data, err := os.ReadFile(cm.dataFile)
	if err != nil {
		return
	}
	json.Unmarshal(data, &cm.Data)

	// Recompute next run times for enabled schedules
	now := time.Now()
	for i := range cm.Data.Schedules {
		if cm.Data.Schedules[i].Enabled {
			if next, err := nextCronTime(cm.Data.Schedules[i].CronExpr, now); err == nil {
				cm.Data.Schedules[i].NextRun = next.Format(time.RFC3339)
			}
		}
	}
}

// Save writes the schedule data to disk atomically.
func (cm *CronManager) Save() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return cm.saveLocked()
}

func (cm *CronManager) saveLocked() error {
	if err := os.MkdirAll(cm.yoloDir, 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}
	data, err := json.MarshalIndent(cm.Data, "", "  ")
	if err != nil {
		return err
	}
	tmp := cm.dataFile + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, cm.dataFile)
}

// ─── Schedule Management ─────────────────────────────────────────────

// AddSchedule creates a new schedule. Returns the assigned ID.
func (cm *CronManager) AddSchedule(name, cronExpr, prompt string) (string, error) {
	// Validate cron expression
	if _, err := nextCronTime(cronExpr, time.Now()); err != nil {
		return "", fmt.Errorf("invalid cron expression %q: %w", cronExpr, err)
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	id := fmt.Sprintf("sched_%d", time.Now().UnixNano())

	nextRun := ""
	if next, err := nextCronTime(cronExpr, time.Now()); err == nil {
		nextRun = next.Format(time.RFC3339)
	}

	sched := CronSchedule{
		ID:        id,
		Name:      name,
		CronExpr:  cronExpr,
		Prompt:    prompt,
		Enabled:   true,
		CreatedAt: time.Now().Format(time.RFC3339),
		NextRun:   nextRun,
	}
	cm.Data.Schedules = append(cm.Data.Schedules, sched)
	cm.saveLocked()
	return id, nil
}

// RemoveSchedule removes a schedule by ID or name. Returns true if found.
func (cm *CronManager) RemoveSchedule(idOrName string) bool {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for i, s := range cm.Data.Schedules {
		if s.ID == idOrName || s.Name == idOrName {
			cm.Data.Schedules = append(cm.Data.Schedules[:i], cm.Data.Schedules[i+1:]...)
			cm.saveLocked()
			return true
		}
	}
	return false
}

// SetEnabled enables or disables a schedule by ID or name.
func (cm *CronManager) SetEnabled(idOrName string, enabled bool) bool {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for i, s := range cm.Data.Schedules {
		if s.ID == idOrName || s.Name == idOrName {
			cm.Data.Schedules[i].Enabled = enabled
			if enabled {
				if next, err := nextCronTime(s.CronExpr, time.Now()); err == nil {
					cm.Data.Schedules[i].NextRun = next.Format(time.RFC3339)
				}
			} else {
				cm.Data.Schedules[i].NextRun = ""
			}
			cm.saveLocked()
			return true
		}
	}
	return false
}

// ListSchedules returns all schedules sorted by next run time.
func (cm *CronManager) ListSchedules() []CronSchedule {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	result := make([]CronSchedule, len(cm.Data.Schedules))
	copy(result, cm.Data.Schedules)

	sort.Slice(result, func(i, j int) bool {
		if result[i].NextRun == "" {
			return false
		}
		if result[j].NextRun == "" {
			return true
		}
		return result[i].NextRun < result[j].NextRun
	})
	return result
}

// ─── Background Ticker ──────────────────────────────────────────────

// Start begins the background goroutine that checks schedules every 30 seconds.
func (cm *CronManager) Start(onFire func(CronSchedule)) {
	cm.onFire = onFire
	go cm.tickLoop()
}

// Stop halts the background ticker.
func (cm *CronManager) Stop() {
	select {
	case <-cm.stopCh:
		// Already stopped
	default:
		close(cm.stopCh)
	}
}

func (cm *CronManager) tickLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-cm.stopCh:
			return
		case <-ticker.C:
			cm.checkSchedules()
		}
	}
}

// checkSchedules evaluates all enabled schedules and fires any that are due.
func (cm *CronManager) checkSchedules() {
	cm.mu.Lock()
	now := time.Now()
	var toFire []CronSchedule

	for i := range cm.Data.Schedules {
		s := &cm.Data.Schedules[i]
		if !s.Enabled || s.NextRun == "" {
			continue
		}

		nextRun, err := time.Parse(time.RFC3339, s.NextRun)
		if err != nil {
			continue
		}

		if now.After(nextRun) || now.Equal(nextRun) {
			// Mark as fired
			s.LastRun = now.Format(time.RFC3339)
			s.RunCount++

			// Compute next run time
			if next, err := nextCronTime(s.CronExpr, now); err == nil {
				s.NextRun = next.Format(time.RFC3339)
			}

			toFire = append(toFire, *s)
		}
	}

	if len(toFire) > 0 {
		cm.saveLocked()
	}
	cm.mu.Unlock()

	// Fire callbacks outside the lock
	for _, s := range toFire {
		if cm.onFire != nil {
			cm.onFire(s)
		}
	}
}

// HasDueSchedules returns true if any schedule is due to fire.
func (cm *CronManager) HasDueSchedules() bool {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	now := time.Now()
	for _, s := range cm.Data.Schedules {
		if !s.Enabled || s.NextRun == "" {
			continue
		}
		nextRun, err := time.Parse(time.RFC3339, s.NextRun)
		if err != nil {
			continue
		}
		if now.After(nextRun) || now.Equal(nextRun) {
			return true
		}
	}
	return false
}

// FormatStatus returns a human-readable overview of all schedules.
func (cm *CronManager) FormatStatus() string {
	schedules := cm.ListSchedules()
	if len(schedules) == 0 {
		return "No schedules configured"
	}

	var sb strings.Builder
	for _, s := range schedules {
		status := "enabled"
		if !s.Enabled {
			status = "disabled"
		}
		sb.WriteString(fmt.Sprintf("  %s (%s) [%s]\n", s.Name, s.CronExpr, status))
		sb.WriteString(fmt.Sprintf("    ID: %s\n", s.ID))
		if s.NextRun != "" {
			if t, err := time.Parse(time.RFC3339, s.NextRun); err == nil {
				sb.WriteString(fmt.Sprintf("    Next: %s\n", t.Format("2006-01-02 15:04")))
			}
		}
		if s.LastRun != "" {
			if t, err := time.Parse(time.RFC3339, s.LastRun); err == nil {
				sb.WriteString(fmt.Sprintf("    Last: %s (runs: %d)\n", t.Format("2006-01-02 15:04"), s.RunCount))
			}
		}
		prompt := s.Prompt
		if len(prompt) > 80 {
			prompt = prompt[:77] + "..."
		}
		sb.WriteString(fmt.Sprintf("    Task: %s\n", prompt))
	}
	return sb.String()
}

// ─── Tool Implementations ────────────────────────────────────────────

// scheduleAdd creates a new scheduled task.
func (e *ToolExecutor) scheduleAdd(args map[string]any) string {
	cm := e.agent.cron
	if cm == nil {
		return errorMessage("scheduler not initialized")
	}

	name := getStringArg(args, "name", "")
	cronExpr := getStringArg(args, "cron", "")
	prompt := getStringArg(args, "prompt", "")

	if name == "" {
		return errorMessage("name is required")
	}
	if cronExpr == "" {
		return errorMessage("cron expression is required (format: 'minute hour day month weekday')")
	}
	if prompt == "" {
		return errorMessage("prompt is required")
	}

	id, err := cm.AddSchedule(name, cronExpr, prompt)
	if err != nil {
		return errorMessage("failed to add schedule: %v", err)
	}

	// Find the schedule to show next run
	schedules := cm.ListSchedules()
	for _, s := range schedules {
		if s.ID == id {
			nextInfo := ""
			if s.NextRun != "" {
				if t, err := time.Parse(time.RFC3339, s.NextRun); err == nil {
					nextInfo = fmt.Sprintf(" (next run: %s)", t.Format("2006-01-02 15:04"))
				}
			}
			return fmt.Sprintf("Schedule '%s' created [%s]%s", name, cronExpr, nextInfo)
		}
	}
	return fmt.Sprintf("Schedule '%s' created with ID %s", name, id)
}

// scheduleRemove removes a scheduled task.
func (e *ToolExecutor) scheduleRemove(args map[string]any) string {
	cm := e.agent.cron
	if cm == nil {
		return errorMessage("scheduler not initialized")
	}

	idOrName := getStringArg(args, "id", "")
	if idOrName == "" {
		return errorMessage("id or name is required")
	}

	if cm.RemoveSchedule(idOrName) {
		return fmt.Sprintf("Schedule '%s' removed", idOrName)
	}
	return errorMessage("schedule '%s' not found", idOrName)
}

// scheduleList lists all scheduled tasks.
func (e *ToolExecutor) scheduleList(args map[string]any) string {
	cm := e.agent.cron
	if cm == nil {
		return errorMessage("scheduler not initialized")
	}
	return cm.FormatStatus()
}

// scheduleToggle enables or disables a scheduled task.
func (e *ToolExecutor) scheduleToggle(args map[string]any) string {
	cm := e.agent.cron
	if cm == nil {
		return errorMessage("scheduler not initialized")
	}

	idOrName := getStringArg(args, "id", "")
	if idOrName == "" {
		return errorMessage("id or name is required")
	}
	enabled := getBoolArg(args, "enabled", true)

	if cm.SetEnabled(idOrName, enabled) {
		state := "enabled"
		if !enabled {
			state = "disabled"
		}
		return fmt.Sprintf("Schedule '%s' %s", idOrName, state)
	}
	return errorMessage("schedule '%s' not found", idOrName)
}

// ─── Cron Expression Parser ─────────────────────────────────────────
//
// Supports 5-field cron: minute hour day-of-month month day-of-week
// Field values: * (any), N (exact), */N (step)
// Day-of-week: 0=Sun, 1=Mon, ..., 6=Sat

// cronField represents a parsed cron field.
type cronField struct {
	any    bool  // true for *
	step   int   // step value for */N (0 means not a step)
	values []int // explicit values
}

// parseCronField parses a single cron field (e.g., "*", "5", "*/10", "1,3,5").
func parseCronField(field string, min, max int) (cronField, error) {
	field = strings.TrimSpace(field)

	if field == "*" {
		return cronField{any: true}, nil
	}

	if strings.HasPrefix(field, "*/") {
		stepStr := strings.TrimPrefix(field, "*/")
		step, err := strconv.Atoi(stepStr)
		if err != nil || step <= 0 {
			return cronField{}, fmt.Errorf("invalid step %q", field)
		}
		return cronField{step: step}, nil
	}

	// Handle comma-separated values
	if strings.Contains(field, ",") {
		parts := strings.Split(field, ",")
		var values []int
		for _, p := range parts {
			v, err := strconv.Atoi(strings.TrimSpace(p))
			if err != nil {
				return cronField{}, fmt.Errorf("invalid value %q", p)
			}
			if v < min || v > max {
				return cronField{}, fmt.Errorf("value %d out of range [%d, %d]", v, min, max)
			}
			values = append(values, v)
		}
		return cronField{values: values}, nil
	}

	// Single value
	v, err := strconv.Atoi(field)
	if err != nil {
		return cronField{}, fmt.Errorf("invalid field %q", field)
	}
	if v < min || v > max {
		return cronField{}, fmt.Errorf("value %d out of range [%d, %d]", v, min, max)
	}
	return cronField{values: []int{v}}, nil
}

// matches returns true if the given value matches this cron field.
func (f cronField) matches(value int) bool {
	if f.any {
		return true
	}
	if f.step > 0 {
		return value%f.step == 0
	}
	for _, v := range f.values {
		if v == value {
			return true
		}
	}
	return false
}

// parseCronExpr parses a 5-field cron expression.
func parseCronExpr(expr string) (minute, hour, dom, month, dow cronField, err error) {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return minute, hour, dom, month, dow, fmt.Errorf("expected 5 fields (minute hour day month weekday), got %d", len(fields))
	}

	minute, err = parseCronField(fields[0], 0, 59)
	if err != nil {
		return minute, hour, dom, month, dow, fmt.Errorf("minute: %w", err)
	}
	hour, err = parseCronField(fields[1], 0, 23)
	if err != nil {
		return minute, hour, dom, month, dow, fmt.Errorf("hour: %w", err)
	}
	dom, err = parseCronField(fields[2], 1, 31)
	if err != nil {
		return minute, hour, dom, month, dow, fmt.Errorf("day: %w", err)
	}
	month, err = parseCronField(fields[3], 1, 12)
	if err != nil {
		return minute, hour, dom, month, dow, fmt.Errorf("month: %w", err)
	}
	dow, err = parseCronField(fields[4], 0, 6)
	if err != nil {
		return minute, hour, dom, month, dow, fmt.Errorf("weekday: %w", err)
	}

	return
}

// nextCronTime calculates the next time after `after` that matches the cron expression.
// It searches up to 366 days ahead to handle yearly crons.
func nextCronTime(expr string, after time.Time) (time.Time, error) {
	minute, hour, dom, month, dow, err := parseCronExpr(expr)
	if err != nil {
		return time.Time{}, err
	}

	// Start from the next minute
	t := after.Truncate(time.Minute).Add(time.Minute)

	// Search up to 366 days ahead
	limit := t.Add(366 * 24 * time.Hour)
	for t.Before(limit) {
		if month.matches(int(t.Month())) &&
			dom.matches(t.Day()) &&
			dow.matches(int(t.Weekday())) &&
			hour.matches(t.Hour()) &&
			minute.matches(t.Minute()) {
			return t, nil
		}
		t = t.Add(time.Minute)
	}

	return time.Time{}, fmt.Errorf("no matching time found within 366 days")
}
