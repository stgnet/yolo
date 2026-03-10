package main

import (
	"testing"
)

func TestUtf8ByteLen(t *testing.T) {
	tests := []struct {
		name string
		lead byte
		want int
	}{
		{"2-byte (latin)", 0xC3, 2},      // e.g., é
		{"2-byte low", 0xC0, 2},
		{"2-byte high", 0xDF, 2},
		{"3-byte (CJK)", 0xE4, 3},        // e.g., 中
		{"3-byte low", 0xE0, 3},
		{"3-byte high", 0xEF, 3},
		{"4-byte (emoji)", 0xF0, 4},       // e.g., 😀
		{"4-byte high", 0xF4, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := utf8ByteLen(tt.lead)
			if got != tt.want {
				t.Errorf("utf8ByteLen(0x%02X) = %d, want %d", tt.lead, got, tt.want)
			}
		})
	}
}

func TestInputLineStruct(t *testing.T) {
	// Test the InputLine struct is properly constructed
	line := InputLine{Text: "hello", OK: true}
	if line.Text != "hello" || !line.OK {
		t.Errorf("InputLine mismatch: %+v", line)
	}

	eof := InputLine{OK: false}
	if eof.OK {
		t.Error("EOF line should have OK=false")
	}
}

func TestNewInputManagerChannelCapacity(t *testing.T) {
	agent := &YoloAgent{}
	im := NewInputManager(agent)

	if cap(im.Lines) != 8 {
		t.Errorf("Lines channel capacity = %d, want 8", cap(im.Lines))
	}
	if cap(im.rawBytes) != 64 {
		t.Errorf("rawBytes channel capacity = %d, want 64", cap(im.rawBytes))
	}
	if cap(im.rawErr) != 1 {
		t.Errorf("rawErr channel capacity = %d, want 1", cap(im.rawErr))
	}
}
