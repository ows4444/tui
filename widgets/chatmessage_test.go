package widgets

import (
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

var chatTS = time.Date(2026, 9, 29, 14, 30, 5, 0, time.UTC)

// TestChatMessageSenderStyling proves criterion #505: ChatMessage styles
// the header distinctly per sender, reusing widgets.Variant's color
// mapping (via Sender.Variant) rather than a parallel color scheme.
func TestChatMessageSenderStyling(t *testing.T) {
	dt := theme.DarkTheme()
	tests := []struct {
		name   string
		sender Sender
		want   Variant
	}{
		{"user", SenderUser, VariantNeutral},
		{"assistant", SenderAssistant, VariantSuccess},
		{"system", SenderSystem, VariantInfo},
		{"error", SenderError, VariantError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.sender.Variant(); got != tt.want {
				t.Errorf("Sender(%v).Variant() = %v, want %v", tt.sender, got, tt.want)
			}
			out := ChatMessage(tt.sender, "", "hi", chatTS, false, dt)
			wantColor := tt.want.Color(dt)
			if !strings.Contains(out, ansi.NewStyle().Foreground(wantColor).Render(tt.sender.label())) &&
				!strings.Contains(out, ansi.NewStyle().Bold().Foreground(wantColor).Render(tt.sender.label())) {
				t.Errorf("ChatMessage(%v, ...) = %q, want header styled with %v's color", tt.sender, out, tt.want)
			}
		})
	}

	// Distinct senders should render distinct output (different color
	// codes/labels), not the same styling collapsed into one scheme.
	u := ChatMessage(SenderUser, "", "hi", chatTS, false, dt)
	e := ChatMessage(SenderError, "", "hi", chatTS, false, dt)
	if u == e {
		t.Errorf("ChatMessage for SenderUser and SenderError rendered identically: %q", u)
	}
}

// TestChatMessageHeaderShowsNameAndTimestamp proves criterion #506: a
// non-empty name and the timestamp both appear in the rendered header
// alongside content.
func TestChatMessageHeaderShowsNameAndTimestamp(t *testing.T) {
	dt := theme.DarkTheme()
	out := ChatMessage(SenderUser, "Alice", "hello there", chatTS, false, dt)

	if !strings.Contains(out, "Alice") {
		t.Errorf("ChatMessage(...) = %q, want it to contain name %q", out, "Alice")
	}
	wantTS := chatTS.Format("15:04:05")
	if !strings.Contains(out, wantTS) {
		t.Errorf("ChatMessage(...) = %q, want it to contain formatted timestamp %q", out, wantTS)
	}
	if !strings.Contains(out, "hello there") {
		t.Errorf("ChatMessage(...) = %q, want it to contain content %q", out, "hello there")
	}

	// Empty name falls back to the sender's default label instead of
	// leaving the header blank.
	fallback := ChatMessage(SenderAssistant, "", "hi", chatTS, false, dt)
	if !strings.Contains(fallback, "Assistant") {
		t.Errorf("ChatMessage with empty name = %q, want fallback label %q", fallback, "Assistant")
	}
}

// TestChatMessageStreamingCursor proves criterion #507: streaming=true
// appends a trailing cursor matching streamtext's cursor convention (the
// same glyph, "▌", styled in the theme's Primary color) rather than any
// character-reveal timing.
func TestChatMessageStreamingCursor(t *testing.T) {
	dt := theme.DarkTheme()
	notStreaming := ChatMessage(SenderAssistant, "", "partial", chatTS, false, dt)
	streaming := ChatMessage(SenderAssistant, "", "partial", chatTS, true, dt)

	wantCursor := ansi.NewStyle().Foreground(dt.Primary).Render(streamCursor)
	if !strings.HasSuffix(streaming, wantCursor) {
		t.Errorf("ChatMessage(..., streaming=true, ...) = %q, want it to end with cursor %q (streamtext's convention)", streaming, wantCursor)
	}
	if strings.Contains(notStreaming, streamCursor) {
		t.Errorf("ChatMessage(..., streaming=false, ...) = %q, want no cursor glyph", notStreaming)
	}
	// The content itself must be shown in full, not partially revealed:
	// this widget does not implement character-reveal timing.
	if !strings.Contains(streaming, "partial") {
		t.Errorf("ChatMessage(..., streaming=true, ...) = %q, want full content %q present (no reveal timing)", streaming, "partial")
	}
}

// TestChatMessageEmptyContent proves criterion #508: empty content still
// renders the header without panicking.
func TestChatMessageEmptyContent(t *testing.T) {
	dt := theme.DarkTheme()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("ChatMessage with empty content panicked: %v", r)
		}
	}()

	out := ChatMessage(SenderSystem, "Bot", "", chatTS, false, dt)
	if !strings.Contains(out, "Bot") {
		t.Errorf("ChatMessage with empty content = %q, want name %q present", out, "Bot")
	}
	if !strings.Contains(out, chatTS.Format("15:04:05")) {
		t.Errorf("ChatMessage with empty content = %q, want timestamp present", out)
	}

	// Also verify streaming with empty content doesn't panic and still
	// appends the cursor.
	streamOut := ChatMessage(SenderSystem, "Bot", "", chatTS, true, dt)
	if !strings.HasSuffix(streamOut, ansi.NewStyle().Foreground(dt.Primary).Render(streamCursor)) {
		t.Errorf("ChatMessage with empty content and streaming=true = %q, want trailing cursor", streamOut)
	}
}
