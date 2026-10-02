package widgets

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/theme"
)

const evil = "a\x1b]52;c;ZXZpbA==\x07b\x1bP1$rx\x1b\\c\x1b_apc\x1b\\d\x1b[2Je\x1b[31mred\x1b[0m\tf"

func noEscapesButSGR(t *testing.T, s string) {
	t.Helper()
	for _, bad := range []string{"\x1b]", "\x1bP", "\x1b_", "\x1b[2J", "\x07", "\t"} {
		if strings.Contains(s, bad) {
			t.Errorf("output %q contains %q", s, bad)
		}
	}
	if !strings.Contains(s, "\x1b[31m") {
		t.Errorf("output %q lost SGR", s)
	}
}

// Criterion: default ChatMessage drops non-SGR sequences.
func TestChatMessageSanitisesByDefault(t *testing.T) {
	noEscapesButSGR(t, ChatMessage(SenderAssistant, "n\x1b]0;x\x07", evil, chatTS, false, theme.DarkTheme()))
}

// Criterion: Raw variant renders content unchanged.
func TestChatMessageRawUnchanged(t *testing.T) {
	if got := ChatMessageRaw(SenderUser, "n", evil, chatTS, false, theme.DarkTheme()); !strings.Contains(got, evil) {
		t.Errorf("raw output %q lost content", got)
	}
}
