package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupEditTest(t *testing.T, content string) (*ToolExecutor, string) {
	dir := t.TempDir()
	te := &ToolExecutor{baseDir: dir, agent: &YoloAgent{}}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "test.txt"), []byte(content), 0o644))
	return te, dir
}

func readTestFile(t *testing.T, dir, name string) string {
	data, err := os.ReadFile(filepath.Join(dir, name))
	require.NoError(t, err)
	return string(data)
}

// ─── edit_file tests ─────────────────────────────────────────────────

func TestEditFile_FirstOccurrence(t *testing.T) {
	te, dir := setupEditTest(t, "foo bar foo baz foo")

	result := te.editFile(map[string]any{"path": "test.txt", "old_text": "foo", "new_text": "qux"})
	assert.Contains(t, result, "Edited")
	assert.Contains(t, result, "1 of 3")
	assert.Equal(t, "qux bar foo baz foo", readTestFile(t, dir, "test.txt"))
}

func TestEditFile_ReplaceAll(t *testing.T) {
	te, dir := setupEditTest(t, "foo bar foo baz foo")

	result := te.editFile(map[string]any{"path": "test.txt", "old_text": "foo", "new_text": "qux", "replace_all": true})
	assert.Contains(t, result, "3 replacements")
	assert.Equal(t, "qux bar qux baz qux", readTestFile(t, dir, "test.txt"))
}

func TestEditFile_LastOccurrence(t *testing.T) {
	te, dir := setupEditTest(t, "foo bar foo baz foo")

	result := te.editFile(map[string]any{"path": "test.txt", "old_text": "foo", "new_text": "qux", "occurrence": -1})
	assert.Contains(t, result, "Edited")
	assert.Equal(t, "foo bar foo baz qux", readTestFile(t, dir, "test.txt"))
}

func TestEditFile_NthOccurrence(t *testing.T) {
	te, dir := setupEditTest(t, "foo bar foo baz foo")

	result := te.editFile(map[string]any{"path": "test.txt", "old_text": "foo", "new_text": "qux", "occurrence": 2})
	assert.Contains(t, result, "Edited")
	assert.Equal(t, "foo bar qux baz foo", readTestFile(t, dir, "test.txt"))
}

func TestEditFile_NthOccurrence_OutOfRange(t *testing.T) {
	te, _ := setupEditTest(t, "foo bar foo")

	result := te.editFile(map[string]any{"path": "test.txt", "old_text": "foo", "new_text": "qux", "occurrence": 5})
	assert.Contains(t, result, "Error:")
	assert.Contains(t, result, "only 2 found")
}

func TestEditFile_DryRun(t *testing.T) {
	te, dir := setupEditTest(t, "line1\nfoo\nline3")

	result := te.editFile(map[string]any{"path": "test.txt", "old_text": "foo", "new_text": "bar", "dry_run": true})
	assert.Contains(t, result, "dry-run")
	assert.Contains(t, result, "line ~2")
	// File should not be modified
	assert.Equal(t, "line1\nfoo\nline3", readTestFile(t, dir, "test.txt"))
}

func TestEditFile_DryRunReplaceAll(t *testing.T) {
	te, _ := setupEditTest(t, "aaa bbb aaa")

	result := te.editFile(map[string]any{"path": "test.txt", "old_text": "aaa", "new_text": "ccc", "replace_all": true, "dry_run": true})
	assert.Contains(t, result, "dry-run")
	assert.Contains(t, result, "2 occurrences")
}

func TestEditFile_NotFound(t *testing.T) {
	te, _ := setupEditTest(t, "hello world")

	result := te.editFile(map[string]any{"path": "test.txt", "old_text": "missing", "new_text": "x"})
	assert.Contains(t, result, "Error:")
	assert.Contains(t, result, "not found")
}

// ─── edit_file_lines tests ───────────────────────────────────────────

func TestEditFileLines_Basic(t *testing.T) {
	te, dir := setupEditTest(t, "line1\nline2\nline3\nline4\nline5")

	result := te.editFileLines(map[string]any{
		"path": "test.txt", "start_line": 2, "end_line": 3, "content": "new2\nnew3\nnewExtra",
	})
	assert.Contains(t, result, "replaced lines 2-3")
	assert.Equal(t, "line1\nnew2\nnew3\nnewExtra\nline4\nline5", readTestFile(t, dir, "test.txt"))
}

func TestEditFileLines_DeleteLines(t *testing.T) {
	te, dir := setupEditTest(t, "line1\nline2\nline3\nline4")

	result := te.editFileLines(map[string]any{
		"path": "test.txt", "start_line": 2, "end_line": 3, "content": "",
	})
	assert.Contains(t, result, "0 new lines")
	assert.Equal(t, "line1\nline4", readTestFile(t, dir, "test.txt"))
}

func TestEditFileLines_DryRun(t *testing.T) {
	te, dir := setupEditTest(t, "line1\nline2\nline3")

	result := te.editFileLines(map[string]any{
		"path": "test.txt", "start_line": 2, "end_line": 2, "content": "newline2", "dry_run": true,
	})
	assert.Contains(t, result, "dry-run")
	assert.Contains(t, result, "line2")
	// File unchanged
	assert.Equal(t, "line1\nline2\nline3", readTestFile(t, dir, "test.txt"))
}

func TestEditFileLines_InvalidRange(t *testing.T) {
	te, _ := setupEditTest(t, "line1\nline2")

	result := te.editFileLines(map[string]any{"path": "test.txt", "start_line": 0, "end_line": 1, "content": "x"})
	assert.Contains(t, result, "Error:")

	result = te.editFileLines(map[string]any{"path": "test.txt", "start_line": 3, "end_line": 1, "content": "x"})
	assert.Contains(t, result, "Error:")

	result = te.editFileLines(map[string]any{"path": "test.txt", "start_line": 5, "end_line": 5, "content": "x"})
	assert.Contains(t, result, "Error:")
}

// ─── patch_file tests ────────────────────────────────────────────────

func TestPatchFile_SingleHunk(t *testing.T) {
	content := "line1\nline2\nline3\nline4\nline5"
	te, dir := setupEditTest(t, content)

	diff := `@@ -2,3 +2,3 @@
 line2
-line3
+line3_modified
 line4`

	result := te.patchFile(map[string]any{"path": "test.txt", "diff": diff})
	assert.Contains(t, result, "1 hunk(s) applied")
	assert.Equal(t, "line1\nline2\nline3_modified\nline4\nline5", readTestFile(t, dir, "test.txt"))
}

func TestPatchFile_MultiHunk(t *testing.T) {
	content := "a\nb\nc\nd\ne\nf\ng"
	te, dir := setupEditTest(t, content)

	diff := `@@ -1,2 +1,2 @@
-a
+A
 b
@@ -6,2 +6,2 @@
-f
+F
 g`

	result := te.patchFile(map[string]any{"path": "test.txt", "diff": diff})
	assert.Contains(t, result, "2 hunk(s) applied")
	assert.Equal(t, "A\nb\nc\nd\ne\nF\ng", readTestFile(t, dir, "test.txt"))
}

func TestPatchFile_DryRun(t *testing.T) {
	te, dir := setupEditTest(t, "a\nb\nc")

	diff := `@@ -1,1 +1,1 @@
-a
+x`

	result := te.patchFile(map[string]any{"path": "test.txt", "diff": diff, "dry_run": true})
	assert.Contains(t, result, "dry-run")
	assert.Contains(t, result, "1 hunk(s)")
	// File unchanged
	assert.Equal(t, "a\nb\nc", readTestFile(t, dir, "test.txt"))
}

func TestPatchFile_ContextMismatch(t *testing.T) {
	te, _ := setupEditTest(t, "a\nb\nc")

	diff := `@@ -1,1 +1,1 @@
-wrong_line
+x`

	result := te.patchFile(map[string]any{"path": "test.txt", "diff": diff})
	assert.Contains(t, result, "Error:")
	assert.Contains(t, result, "mismatch")
}

func TestPatchFile_NoHunks(t *testing.T) {
	te, _ := setupEditTest(t, "content")

	result := te.patchFile(map[string]any{"path": "test.txt", "diff": "no hunks here"})
	assert.Contains(t, result, "Error:")
	assert.Contains(t, result, "no hunks")
}

func TestPatchFile_WithHeaders(t *testing.T) {
	te, dir := setupEditTest(t, "hello\nworld")

	diff := `--- a/test.txt
+++ b/test.txt
@@ -1,2 +1,2 @@
 hello
-world
+universe`

	result := te.patchFile(map[string]any{"path": "test.txt", "diff": diff})
	assert.Contains(t, result, "1 hunk(s)")
	assert.Equal(t, "hello\nuniverse", readTestFile(t, dir, "test.txt"))
}

// ─── undo_edit tests ─────────────────────────────────────────────────

func TestUndoEdit(t *testing.T) {
	te, dir := setupEditTest(t, "original content")

	// Make an edit
	te.editFile(map[string]any{"path": "test.txt", "old_text": "original", "new_text": "modified"})
	assert.Equal(t, "modified content", readTestFile(t, dir, "test.txt"))

	// Undo
	result := te.undoEdit(map[string]any{"path": "test.txt"})
	assert.Contains(t, result, "Reverted")
	assert.Equal(t, "original content", readTestFile(t, dir, "test.txt"))
}

func TestUndoEdit_NoBackup(t *testing.T) {
	te, _ := setupEditTest(t, "content")

	result := te.undoEdit(map[string]any{"path": "test.txt"})
	assert.Contains(t, result, "Error:")
	assert.Contains(t, result, "no edit history")
}

func TestUndoEdit_AfterLineEdit(t *testing.T) {
	te, dir := setupEditTest(t, "line1\nline2\nline3")

	te.editFileLines(map[string]any{
		"path": "test.txt", "start_line": 2, "end_line": 2, "content": "modified",
	})
	assert.Equal(t, "line1\nmodified\nline3", readTestFile(t, dir, "test.txt"))

	result := te.undoEdit(map[string]any{"path": "test.txt"})
	assert.Contains(t, result, "Reverted")
	assert.Equal(t, "line1\nline2\nline3", readTestFile(t, dir, "test.txt"))
}

func TestUndoEdit_AfterPatch(t *testing.T) {
	te, dir := setupEditTest(t, "a\nb\nc")

	te.patchFile(map[string]any{"path": "test.txt", "diff": "@@ -1,1 +1,1 @@\n-a\n+x"})
	assert.Equal(t, "x\nb\nc", readTestFile(t, dir, "test.txt"))

	result := te.undoEdit(map[string]any{"path": "test.txt"})
	assert.Contains(t, result, "Reverted")
	assert.Equal(t, "a\nb\nc", readTestFile(t, dir, "test.txt"))
}

// ─── applyUnifiedDiff tests ─────────────────────────────────────────

func TestApplyUnifiedDiff_AddLines(t *testing.T) {
	source := "a\nb\nc"
	diff := `@@ -1,2 +1,3 @@
 a
+new
 b`

	result, hunks, err := applyUnifiedDiff(source, diff)
	require.NoError(t, err)
	assert.Equal(t, 1, hunks)
	assert.Equal(t, "a\nnew\nb\nc", result)
}

func TestApplyUnifiedDiff_RemoveLines(t *testing.T) {
	source := "a\nb\nc\nd"
	diff := `@@ -2,2 +2,1 @@
-b
-c
+bc`

	result, hunks, err := applyUnifiedDiff(source, diff)
	require.NoError(t, err)
	assert.Equal(t, 1, hunks)
	assert.Equal(t, "a\nbc\nd", result)
}

func TestApplyUnifiedDiff_InvalidHunkHeader(t *testing.T) {
	_, _, err := applyUnifiedDiff("a", "@@ bad @@\n-a\n+b")
	assert.Error(t, err)
}

// ─── Backup system tests ────────────────────────────────────────────

func TestEditBackup_SaveAndGet(t *testing.T) {
	saveEditBackup("/tmp/test_backup_file", []byte("backup data"))
	data, ok := getEditBackup("/tmp/test_backup_file")
	assert.True(t, ok)
	assert.Equal(t, []byte("backup data"), data)

	// Cleanup
	editBackupsMu.Lock()
	delete(editBackups, "/tmp/test_backup_file")
	editBackupsMu.Unlock()
}

func TestEditBackup_NotFound(t *testing.T) {
	_, ok := getEditBackup("/tmp/nonexistent_backup_12345")
	assert.False(t, ok)
}
