package main

import (
	"testing"
)

func TestFilterToolActivityMarkers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"no markers",
			"hello world\nfoo bar",
			"hello world\nfoo bar",
		},
		{
			"opening marker",
			"before\n[tool activity] something\nafter",
			"before\nafter",
		},
		{
			"closing marker",
			"before\n[/tool activity]\nafter",
			"before\nafter",
		},
		{
			"both markers",
			"text\n[tool activity] read_file\n  working...\n[/tool activity]\nmore text",
			"text\n  working...\nmore text",
		},
		{
			"marker with leading whitespace",
			"  [tool activity] test\n  [/tool activity]",
			"",
		},
		{
			"empty input",
			"",
			"",
		},
		{
			"only regular text",
			"line1\nline2\nline3",
			"line1\nline2\nline3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterToolActivityMarkers(tt.input)
			if got != tt.expected {
				t.Errorf("filterToolActivityMarkers(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestBreakWordAtVisibleLength(t *testing.T) {
	tests := []struct {
		name      string
		word      string
		maxLen    int
		wantTrunc string
		wantRem   string
	}{
		{
			"short word no break",
			"hello", 10,
			"hello", "",
		},
		{
			"exact length",
			"hello", 5,
			"hello", "",
		},
		{
			"break plain word",
			"hello", 3,
			"hel", "lo",
		},
		{
			"word with ansi codes",
			"\033[31mhello\033[0m", 3,
			"\033[31mhel", "lo\033[0m",
		},
		{
			"break at 1",
			"abcdef", 1,
			"a", "bcdef",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTrunc, gotRem := breakWordAtVisibleLength(tt.word, tt.maxLen)
			if gotTrunc != tt.wantTrunc {
				t.Errorf("truncated = %q, want %q", gotTrunc, tt.wantTrunc)
			}
			if gotRem != tt.wantRem {
				t.Errorf("remainder = %q, want %q", gotRem, tt.wantRem)
			}
		})
	}
}

func TestStripAnsiCodesExtended(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"no codes", "hello", "hello"},
		{"color code", "\033[31mred\033[0m", "red"},
		{"bold", "\033[1mbold\033[0m", "bold"},
		{"multiple", "\033[31m\033[1mhello\033[0m", "hello"},
		{"empty", "", ""},
		{"only codes", "\033[31m\033[0m", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripAnsiCodes(tt.input)
			if got != tt.expected {
				t.Errorf("stripAnsiCodes(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestTruncateStringExtended(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"short", "hi", 10, "hi"},
		{"exact", "hello", 5, "hello"},
		{"truncated", "hello world", 5, "hello..."},
		{"one char", "abcdef", 1, "a..."},
		{"unicode", "héllo", 3, "hél..."},
		{"empty", "", 5, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateString(tt.input, tt.maxLen)
			if got != tt.expected {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.expected)
			}
		})
	}
}

func TestTrackCursorMovementExtended(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		rows    int
		cols    int
		wantRow int
		wantCol int
	}{
		{"simple text", "abc", 24, 80, 1, 4},
		{"with newline", "ab\ncd", 24, 80, 2, 3},
		{"carriage return", "abc\rde", 24, 80, 1, 3},
		{"wrap at col boundary", "abcde", 24, 3, 2, 3},
		{"multiple wraps", "abcdefgh", 24, 3, 3, 3},
		{"newline at scroll boundary", "a\nb\nc", 4, 80, 2, 2}, // rows-2 = 2
		{"empty", "", 24, 80, 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ui := &TerminalUI{rows: tt.rows, cols: tt.cols, outRow: 1, outCol: 1, scrollEnd: tt.rows - 2}
			ui.trackCursorMovement(tt.text)
			if ui.outRow != tt.wantRow || ui.outCol != tt.wantCol {
				t.Errorf("after %q: got row=%d col=%d, want row=%d col=%d",
					tt.text, ui.outRow, ui.outCol, tt.wantRow, tt.wantCol)
			}
		})
	}
}

func TestWrapTextZeroCols(t *testing.T) {
	ui := &TerminalUI{cols: 0}
	result := ui.wrapText("hello world")
	if result != "hello world" {
		t.Errorf("wrapText with 0 cols should pass through, got: %q", result)
	}
}
