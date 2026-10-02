package accordion

import (
	"github.com/ows4444/tui/theme"
	"strings"
	"testing"
	"time"
)

// TestThinkingSectionMetadataAndContent proves criterion #526: given a label,
// reasoning, token count, and duration, ThinkingSection returns an
// Section (not a new Model type) whose Title includes the label
// plus formatted token-count/duration metadata, and whose Content is the
// reasoning text.
func TestThinkingSectionMetadataAndContent(t *testing.T) {
	sec := ThinkingSection("Reasoning", "step one\nstep two", 1234, 3200*time.Millisecond, false)

	if !strings.HasPrefix(sec.Title, "Reasoning") {
		t.Errorf("Title = %q, want prefix %q", sec.Title, "Reasoning")
	}
	if !strings.Contains(sec.Title, "1.2k tokens") {
		t.Errorf("Title = %q, want it to contain %q", sec.Title, "1.2k tokens")
	}
	if !strings.Contains(sec.Title, "3.2s") {
		t.Errorf("Title = %q, want it to contain %q", sec.Title, "3.2s")
	}
	if sec.Content != "step one\nstep two" {
		t.Errorf("Content = %q, want %q", sec.Content, "step one\nstep two")
	}
}

// TestThinkingSectionDefaultLabel proves criterion #527: an empty label
// defaults the title to "Reasoning".
func TestThinkingSectionDefaultLabel(t *testing.T) {
	sec := ThinkingSection("", "some reasoning", 10, time.Second, false)

	if !strings.HasPrefix(sec.Title, "Reasoning") {
		t.Errorf("Title = %q, want prefix %q", sec.Title, "Reasoning")
	}
}

// TestThinkingSectionEmptyReasoning proves criterion #528: empty reasoning
// still returns a valid Section with empty Content (modulo any streaming
// cursor), without panicking.
func TestThinkingSectionEmptyReasoning(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("ThinkingSection panicked on empty reasoning: %v", r)
		}
	}()

	sec := ThinkingSection("Reasoning", "", 0, 0, false)
	if sec.Content != "" {
		t.Errorf("Content = %q, want empty", sec.Content)
	}
	if sec.Title == "" {
		t.Error("Title = \"\", want non-empty")
	}
}

// TestThinkingSectionStreamingCursor proves criterion #529: streaming=true
// appends a trailing cursor indicator to Content matching ChatMessage's
// existing streaming-cursor convention (the cursorGlyph constant/style),
// not a new reveal-animation implementation.
func TestThinkingSectionStreamingCursor(t *testing.T) {
	sec := ThinkingSection("Reasoning", "partial thought", 5, time.Second, true)

	want := "partial thought" + cursorGlyph
	if sec.Content != want {
		t.Errorf("Content = %q, want %q", sec.Content, want)
	}

	notStreaming := ThinkingSection("Reasoning", "partial thought", 5, time.Second, false)
	if strings.Contains(notStreaming.Content, cursorGlyph) {
		t.Errorf("Content = %q, should not contain cursorGlyph when streaming=false", notStreaming.Content)
	}
}

// TestThinkingSectionStreamingEmptyReasoning proves the streaming cursor
// still appends cleanly (no panic, no unexpected content) even when
// reasoning is empty and streaming is true — combining criteria #528 and
// #529.
func TestThinkingSectionStreamingEmptyReasoning(t *testing.T) {
	sec := ThinkingSection("Reasoning", "", 0, 0, true)
	if sec.Content != cursorGlyph {
		t.Errorf("Content = %q, want %q", sec.Content, cursorGlyph)
	}
}

var cursorGlyph = theme.Theme{}.GlyphSet().Cursor
