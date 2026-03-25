package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Cron Expression Parser Tests ────────────────────────────────────

func TestParseCronField_Star(t *testing.T) {
	f, err := parseCronField("*", 0, 59)
	require.NoError(t, err)
	assert.True(t, f.any)
	assert.True(t, f.matches(0))
	assert.True(t, f.matches(30))
	assert.True(t, f.matches(59))
}

func TestParseCronField_Step(t *testing.T) {
	f, err := parseCronField("*/10", 0, 59)
	require.NoError(t, err)
	assert.Equal(t, 10, f.step)
	assert.True(t, f.matches(0))
	assert.True(t, f.matches(10))
	assert.True(t, f.matches(20))
	assert.False(t, f.matches(5))
	assert.False(t, f.matches(15))
}

func TestParseCronField_SingleValue(t *testing.T) {
	f, err := parseCronField("5", 0, 59)
	require.NoError(t, err)
	assert.True(t, f.matches(5))
	assert.False(t, f.matches(0))
	assert.False(t, f.matches(10))
}

func TestParseCronField_CommaValues(t *testing.T) {
	f, err := parseCronField("1,3,5", 0, 6)
	require.NoError(t, err)
	assert.True(t, f.matches(1))
	assert.True(t, f.matches(3))
	assert.True(t, f.matches(5))
	assert.False(t, f.matches(2))
	assert.False(t, f.matches(4))
}

func TestParseCronField_OutOfRange(t *testing.T) {
	_, err := parseCronField("60", 0, 59)
	assert.Error(t, err)

	_, err = parseCronField("-1", 0, 59)
	assert.Error(t, err)
}

func TestParseCronField_Invalid(t *testing.T) {
	_, err := parseCronField("abc", 0, 59)
	assert.Error(t, err)

	_, err = parseCronField("*/0", 0, 59)
	assert.Error(t, err)

	_, err = parseCronField("*/-1", 0, 59)
	assert.Error(t, err)
}

func TestParseCronExpr(t *testing.T) {
	// Every day at 9 AM
	min, hr, dom, mon, dow, err := parseCronExpr("0 9 * * *")
	require.NoError(t, err)
	assert.True(t, min.matches(0))
	assert.False(t, min.matches(30))
	assert.True(t, hr.matches(9))
	assert.False(t, hr.matches(10))
	assert.True(t, dom.any)
	assert.True(t, mon.any)
	assert.True(t, dow.any)
}

func TestParseCronExpr_WrongFieldCount(t *testing.T) {
	_, _, _, _, _, err := parseCronExpr("0 9 *")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected 5 fields")

	_, _, _, _, _, err = parseCronExpr("0 9 * * * *")
	assert.Error(t, err)
}

func TestNextCronTime_EveryMinute(t *testing.T) {
	now := time.Date(2025, 6, 15, 10, 30, 0, 0, time.Local)
	next, err := nextCronTime("* * * * *", now)
	require.NoError(t, err)
	assert.Equal(t, now.Add(time.Minute), next)
}

func TestNextCronTime_DailyAt9AM(t *testing.T) {
	// If it's 10 AM, next should be 9 AM tomorrow
	now := time.Date(2025, 6, 15, 10, 0, 0, 0, time.Local)
	next, err := nextCronTime("0 9 * * *", now)
	require.NoError(t, err)
	assert.Equal(t, 9, next.Hour())
	assert.Equal(t, 0, next.Minute())
	assert.Equal(t, 16, next.Day())
}

func TestNextCronTime_Every10Min(t *testing.T) {
	now := time.Date(2025, 6, 15, 10, 5, 0, 0, time.Local)
	next, err := nextCronTime("*/10 * * * *", now)
	require.NoError(t, err)
	assert.Equal(t, 10, next.Minute())
	assert.Equal(t, 10, next.Hour())
}

func TestNextCronTime_SpecificDOW(t *testing.T) {
	// Monday = 1
	// Start on a Wednesday (June 18, 2025)
	now := time.Date(2025, 6, 18, 10, 0, 0, 0, time.Local)
	next, err := nextCronTime("0 9 * * 1", now)
	require.NoError(t, err)
	assert.Equal(t, time.Monday, next.Weekday())
	assert.Equal(t, 9, next.Hour())
}

func TestNextCronTime_InvalidExpr(t *testing.T) {
	_, err := nextCronTime("bad expr", time.Now())
	assert.Error(t, err)
}

// ─── CronManager Tests ──────────────────────────────────────────────

func TestCronManager_AddAndList(t *testing.T) {
	dir := t.TempDir()
	cm := NewCronManager(dir)

	id, err := cm.AddSchedule("test-job", "*/10 * * * *", "check build status")
	require.NoError(t, err)
	assert.NotEmpty(t, id)

	schedules := cm.ListSchedules()
	assert.Len(t, schedules, 1)
	assert.Equal(t, "test-job", schedules[0].Name)
	assert.Equal(t, "*/10 * * * *", schedules[0].CronExpr)
	assert.Equal(t, "check build status", schedules[0].Prompt)
	assert.True(t, schedules[0].Enabled)
	assert.NotEmpty(t, schedules[0].NextRun)
}

func TestCronManager_AddInvalidCron(t *testing.T) {
	dir := t.TempDir()
	cm := NewCronManager(dir)

	_, err := cm.AddSchedule("bad", "not valid", "prompt")
	assert.Error(t, err)
}

func TestCronManager_Remove(t *testing.T) {
	dir := t.TempDir()
	cm := NewCronManager(dir)

	cm.AddSchedule("job1", "0 9 * * *", "morning check")
	cm.AddSchedule("job2", "0 17 * * *", "evening check")

	assert.Len(t, cm.ListSchedules(), 2)

	// Remove by name
	assert.True(t, cm.RemoveSchedule("job1"))
	assert.Len(t, cm.ListSchedules(), 1)

	// Remove non-existent
	assert.False(t, cm.RemoveSchedule("nonexistent"))

	// Remove remaining
	assert.True(t, cm.RemoveSchedule("job2"))
	assert.Empty(t, cm.ListSchedules())
}

func TestCronManager_Toggle(t *testing.T) {
	dir := t.TempDir()
	cm := NewCronManager(dir)

	cm.AddSchedule("toggleable", "0 12 * * *", "noon task")

	// Disable
	assert.True(t, cm.SetEnabled("toggleable", false))
	schedules := cm.ListSchedules()
	assert.False(t, schedules[0].Enabled)
	assert.Empty(t, schedules[0].NextRun)

	// Enable
	assert.True(t, cm.SetEnabled("toggleable", true))
	schedules = cm.ListSchedules()
	assert.True(t, schedules[0].Enabled)
	assert.NotEmpty(t, schedules[0].NextRun)

	// Non-existent
	assert.False(t, cm.SetEnabled("nope", true))
}

func TestCronManager_Persistence(t *testing.T) {
	dir := t.TempDir()
	cm := NewCronManager(dir)

	cm.AddSchedule("persist-job", "30 8 * * *", "morning standup")

	// Create new manager, should load from disk
	cm2 := NewCronManager(dir)
	cm2.Load()

	schedules := cm2.ListSchedules()
	assert.Len(t, schedules, 1)
	assert.Equal(t, "persist-job", schedules[0].Name)
}

func TestCronManager_CheckSchedules(t *testing.T) {
	dir := t.TempDir()
	cm := NewCronManager(dir)

	var fired []string
	cm.onFire = func(s CronSchedule) {
		fired = append(fired, s.Name)
	}

	// Create a schedule with next_run in the past so it fires immediately
	cm.mu.Lock()
	cm.Data.Schedules = append(cm.Data.Schedules, CronSchedule{
		ID:       "test",
		Name:     "past-due",
		CronExpr: "* * * * *",
		Prompt:   "do something",
		Enabled:  true,
		NextRun:  time.Now().Add(-1 * time.Minute).Format(time.RFC3339),
	})
	cm.mu.Unlock()

	cm.checkSchedules()

	assert.Len(t, fired, 1)
	assert.Equal(t, "past-due", fired[0])

	// Verify it updated last_run and run_count
	cm.mu.Lock()
	s := cm.Data.Schedules[0]
	cm.mu.Unlock()
	assert.NotEmpty(t, s.LastRun)
	assert.Equal(t, 1, s.RunCount)
}

func TestCronManager_HasDueSchedules(t *testing.T) {
	dir := t.TempDir()
	cm := NewCronManager(dir)

	assert.False(t, cm.HasDueSchedules())

	// Add a past-due schedule
	cm.mu.Lock()
	cm.Data.Schedules = append(cm.Data.Schedules, CronSchedule{
		ID:       "test",
		Name:     "due",
		CronExpr: "* * * * *",
		Prompt:   "task",
		Enabled:  true,
		NextRun:  time.Now().Add(-1 * time.Minute).Format(time.RFC3339),
	})
	cm.mu.Unlock()

	assert.True(t, cm.HasDueSchedules())
}

func TestCronManager_FormatStatus(t *testing.T) {
	dir := t.TempDir()
	cm := NewCronManager(dir)

	assert.Equal(t, "No schedules configured", cm.FormatStatus())

	cm.AddSchedule("daily-check", "0 9 * * *", "run tests")
	status := cm.FormatStatus()
	assert.Contains(t, status, "daily-check")
	assert.Contains(t, status, "0 9 * * *")
	assert.Contains(t, status, "run tests")
}

func TestCronManager_LoadCorruptFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cron.json"), []byte("bad"), 0o644))

	cm := NewCronManager(dir)
	cm.Load()
	assert.Empty(t, cm.ListSchedules())
}

func TestCronManager_StopIdempotent(t *testing.T) {
	dir := t.TempDir()
	cm := NewCronManager(dir)
	cm.Start(nil)

	// Double stop should not panic
	cm.Stop()
	cm.Stop()
}

// ─── Tool Implementation Tests ───────────────────────────────────────

func TestScheduleTools(t *testing.T) {
	dir := t.TempDir()
	cm := NewCronManager(dir)
	agent := &YoloAgent{cron: cm}
	te := &ToolExecutor{agent: agent}

	// Add
	result := te.scheduleAdd(map[string]any{
		"name":   "test-sched",
		"cron":   "0 */6 * * *",
		"prompt": "check status every 6 hours",
	})
	assert.Contains(t, result, "created")

	// List
	result = te.scheduleList(nil)
	assert.Contains(t, result, "test-sched")

	// Toggle disable
	result = te.scheduleToggle(map[string]any{"id": "test-sched", "enabled": false})
	assert.Contains(t, result, "disabled")

	// Toggle enable
	result = te.scheduleToggle(map[string]any{"id": "test-sched", "enabled": true})
	assert.Contains(t, result, "enabled")

	// Remove
	result = te.scheduleRemove(map[string]any{"id": "test-sched"})
	assert.Contains(t, result, "removed")
}

func TestScheduleTools_Errors(t *testing.T) {
	agent := &YoloAgent{cron: nil}
	te := &ToolExecutor{agent: agent}

	assert.Contains(t, te.scheduleAdd(map[string]any{"name": "x", "cron": "* * * * *", "prompt": "y"}), "Error:")
	assert.Contains(t, te.scheduleRemove(map[string]any{"id": "x"}), "Error:")
	assert.Contains(t, te.scheduleList(nil), "Error:")
	assert.Contains(t, te.scheduleToggle(map[string]any{"id": "x", "enabled": true}), "Error:")
}

func TestScheduleTools_Validation(t *testing.T) {
	dir := t.TempDir()
	cm := NewCronManager(dir)
	agent := &YoloAgent{cron: cm}
	te := &ToolExecutor{agent: agent}

	// Missing fields
	assert.Contains(t, te.scheduleAdd(map[string]any{}), "Error:")
	assert.Contains(t, te.scheduleAdd(map[string]any{"name": "x"}), "Error:")
	assert.Contains(t, te.scheduleAdd(map[string]any{"name": "x", "cron": "bad"}), "Error:")

	// Remove non-existent
	assert.Contains(t, te.scheduleRemove(map[string]any{"id": "none"}), "Error:")

	// Toggle non-existent
	assert.Contains(t, te.scheduleToggle(map[string]any{"id": "none", "enabled": true}), "Error:")
}

func TestCronManager_LoadRecomputesNextRun(t *testing.T) {
	dir := t.TempDir()
	cm := NewCronManager(dir)
	cm.AddSchedule("recompute-test", "0 9 * * *", "task")

	// Manually set next_run to a past time
	cm.mu.Lock()
	cm.Data.Schedules[0].NextRun = time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
	cm.saveLocked()
	cm.mu.Unlock()

	// Load should recompute next_run to future
	cm2 := NewCronManager(dir)
	cm2.Load()
	schedules := cm2.ListSchedules()
	require.Len(t, schedules, 1)

	nextRun, err := time.Parse(time.RFC3339, schedules[0].NextRun)
	require.NoError(t, err)
	assert.True(t, nextRun.After(time.Now()))
}

func TestSaveLoadRoundtrip(t *testing.T) {
	dir := t.TempDir()
	cm := NewCronManager(dir)

	cm.AddSchedule("job1", "0 9 * * *", "morning")
	cm.AddSchedule("job2", "30 17 * * 1,3,5", "MWF afternoon")

	// Read raw JSON
	data, err := os.ReadFile(filepath.Join(dir, "cron.json"))
	require.NoError(t, err)

	var cronData CronData
	require.NoError(t, json.Unmarshal(data, &cronData))
	assert.Equal(t, 1, cronData.Version)
	assert.Len(t, cronData.Schedules, 2)
}
