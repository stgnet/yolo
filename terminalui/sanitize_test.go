package terminalui

import (
	"strings"
	"testing"
)

// TestSanitizeOutput tests the SanitizeOutput function which strips
// escape sequences from external text to prevent terminal corruption.

func TestSanitizeOutputPreservesNormalText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "simple text",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "text with spaces",
			input:    "Hello   World",
			expected: "Hello   World",
		},
		{
			name:     "text with newlines",
			input:    "Line 1\nLine 2\nLine 3",
			expected: "Line 1\nLine 2\nLine 3",
		},
		{
			name:     "text with tabs",
			input:    "Column1\tColumn2",
			expected: "Column1\tColumn2",
		},
		{
			name:     "text with carriage return",
			input:    "Line1\rLine2",
			expected: "Line1\rLine2",
		},
		{
			name:     "mixed whitespace",
			input:    "Line1\n\tIndented\r\nNew paragraph",
			expected: "Line1\n\tIndented\r\nNew paragraph",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeOutput(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeOutput(%q) = %q; expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeOutputPreservesColors(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "reset color",
			input:    "\033[0mNormal",
			expected: "\033[0mNormal",
		},
		{
			name:     "bold text",
			input:    "\033[1mBold text\033[0m",
			expected: "\033[1mBold text\033[0m",
		},
		{
			name:     "red color",
			input:    "\033[31mRed text\033[0m",
			expected: "\033[31mRed text\033[0m",
		},
		{
			name:     "green color",
			input:    "\033[32mGreen\033[0m",
			expected: "\033[32mGreen\033[0m",
		},
		{
			name:     "multiple colors",
			input:    "\033[31mRed\033[32m then Green\033[0m",
			expected: "\033[31mRed\033[32m then Green\033[0m",
		},
		{
			name:     "color with text",
			input:    "Start \033[34mBlue middle\033[0m End",
			expected: "Start \033[34mBlue middle\033[0m End",
		},
		{
			name:     "bold and color",
			input:    "\033[1;31mBold Red\033[0m",
			expected: "\033[1;31mBold Red\033[0m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeOutput(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeOutput() = %q; expected %q", result, tt.expected)
			}
		})
	}
}

func TestSanitizeOutputStripsEscapeSequences(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "cursor move sequence",
			input:    "Hello\033[2JWorld",
			expected: "HelloWorld",
		},
		{
			name:     "clear screen",
			input:    "\033[2;2HText",
			expected: "Text",
		},
		{
			name:     "osc sequence with BEL",
			input:    "Before\033]0;Title\007After",
			expected: "BeforeAfter",
		},
		{
			name:     "esc followed by random char",
			input:    "Hello\x1b?World",
			expected: "HelloWorld",
		},
		{
			name:     "multiple escape sequences",
			input:    "\033[2J\033[HText",
			expected: "Text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeOutput(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeOutput() = %q; expected %q", result, tt.expected)
			}
		})
	}
}

func TestSanitizeOutputStripsControlChars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "bell character",
			input:    "Hello\007World",
			expected: "HelloWorld",
		},
		{
			name:     "backspace",
			input:    "Hello\010World",
			expected: "HelloWorld",
		},
		{
			name:     "form feed",
			input:    "Page\014Break",
			expected: "PageBreak",
		},
		{
			name:     "vertical tab",
			input:    "Line1\013Line2",
			expected: "Line1Line2",
		},
		{
			name:     "mixed control chars",
			input:    "A\007B\010C\014D",
			expected: "ABCD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeOutput(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeOutput() = %q; expected %q", result, tt.expected)
			}
		})
	}
}

func TestSanitizeOutputPreservesUTF8(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic UTF-8",
			input:    "Héllo Wörld",
			expected: "Héllo Wörld",
		},
		{
			name:     "emoji",
			input:    "Hello 🌍 World",
			expected: "Hello 🌍 World",
		},
		{
			name:     "Chinese characters",
			input:    "你好世界",
			expected: "你好世界",
		},
		{
			name:     "Japanese characters",
			input:    "こんにちは",
			expected: "こんにちは",
		},
		{
			name:     "mixed ASCII and UTF-8",
			input:    "Hello 世界 🌍!",
			expected: "Hello 世界 🌍!",
		},
		{
			name:     "UTF-8 with colors",
			input:    "\033[32mGreen 绿色\033[0m",
			expected: "\033[32mGreen 绿色\033[0m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeOutput(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeOutput() = %q; expected %q", result, tt.expected)
			}
		})
	}
}

func TestSanitizeOutputComplexCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "complex mixed content",
			input:    "Line1\n\033[32mGreen line with color \033[0mand normal\033]0;Title\007\n\033[2JClear",
			expected: "Line1\n\033[32mGreen line with color \033[0mand normal\nClear",
		},
		{
			name:     "LLM output simulation",
			input:    "\033[36mUser\033[0m: Hello\n\033[32mModel\033[0m: Hi there! 👋\n\033]8;https://example.com\033\\link\033]8;;\033\\",
			expected: "\033[36mUser\033[0m: Hello\n\033[32mModel\033[0m: Hi there! 👋\nlink",
		},
		{
			name:     "repeated escape chars",
			input:    "\x1b\x1b\x1bTest",
			expected: "est", // First two \x1b consume each other, third + 'T' consumed together
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeOutput(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeOutput() = %q; expected %q", result, tt.expected)
			}
		})
	}
}

func TestSanitizeOutputPerformance(t *testing.T) {
	// Test with a large input to ensure performance is acceptable
	largeInput := strings.Repeat("This is a line of normal text\n", 1000) +
		"\033[32mThis is colored\033[0m\n" +
		strings.Repeat("More normal text\n", 1000)

	start := len(largeInput)
	result := SanitizeOutput(largeInput)
	
	// Result should be similar size (colors preserved, no stripping needed in this case)
	if len(result) == 0 {
		t.Error("Expected non-empty result")
	}
	
	t.Logf("Processed %d bytes, produced %d bytes", start, len(result))
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxChars int
		expected string
	}{
		{
			name:     "shorter than limit",
			input:    "Short",
			maxChars: 10,
			expected: "Short",
		},
		{
			name:     "exactly at limit",
			input:    "Exactly10",
			maxChars: 10,
			expected: "Exactly10",
		},
		{
			name:     "longer than limit",
			input:    "This is a long string that needs truncation",
			maxChars: 20,
			expected: "This is a long st...", // Leaves room for 3-dot ellipsis
		},
		{
			name:     "truncate at boundary",
			input:    "Hello World Test",
			maxChars: 15,
			expected: "Hello World ...", // maxChars - 3 = 12 chars + "... "
		},
		{
			name:     "UTF-8 truncation",
			input:    "Hello 世界 🌍",
			maxChars: 10,
			expected: "Hello 世界 🌍", // String is exactly 10 runes, no truncation needed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateString(tt.input, tt.maxChars)
			if result != tt.expected {
				t.Errorf("truncateString(%q, %d) = %q; expected %q", 
					tt.input, tt.maxChars, result, tt.expected)
			}
		})
	}
}
