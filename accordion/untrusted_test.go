package accordion

import (
	"strings"
	"testing"
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

// Criterion: default ToolCall drops non-SGR sequences; Raw is unchanged.
func TestToolCallSanitisesAndRaw(t *testing.T) {
	sec := ToolCall("t\x1b]0;x\x07", evil, ToolCallSuccess, evil)
	noEscapesButSGR(t, sec.Content)
	if strings.Contains(sec.Title, "\x1b") {
		t.Errorf("title %q has escape", sec.Title)
	}
	raw := ToolCallRaw("n", evil, ToolCallSuccess, evil)
	if raw.Content != "Args: "+evil+"\nResult: "+evil {
		t.Errorf("raw content changed: %q", raw.Content)
	}
}
