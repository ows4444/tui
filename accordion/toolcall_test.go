package accordion

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
)

// TestToolCallTitleAndContent proves criterion #562: ToolCall returns an
// Section whose Title includes the name plus a status indicator,
// and whose Content includes the args and result.
func TestToolCallTitleAndContent(t *testing.T) {
	sec := ToolCall("read_file", `{"path":"a.go"}`, ToolCallRunning, "")

	if !strings.Contains(sec.Title, "read_file") {
		t.Errorf("Title = %q, want it to contain %q", sec.Title, "read_file")
	}
	if !strings.Contains(sec.Title, "running") {
		t.Errorf("Title = %q, want it to contain a status indicator for running", sec.Title)
	}
	if !strings.Contains(sec.Content, `{"path":"a.go"}`) {
		t.Errorf("Content = %q, want it to contain the args", sec.Content)
	}
}

// TestToolCallStatusMarkersDistinguishable proves criterion #564: each
// ToolCallStatus value renders a distinguishably different indicator, still
// as plain text (no embedded ANSI/Render calls, per the previous
// criterion's no-embedded-Render rule).
func TestToolCallStatusMarkersDistinguishable(t *testing.T) {
	statuses := []ToolCallStatus{ToolCallPending, ToolCallRunning, ToolCallSuccess, ToolCallError}
	seen := map[string]bool{}

	for _, status := range statuses {
		sec := ToolCall("tool", "args", status, "result")
		if strings.Contains(sec.Title, "\x1b[") {
			t.Errorf("status %v: Title = %q contains an embedded ANSI escape, want plain text", status, sec.Title)
		}
		if seen[sec.Title] {
			t.Errorf("status %v produced a Title already seen for another status: %q", status, sec.Title)
		}
		seen[sec.Title] = true
	}

	if len(seen) != len(statuses) {
		t.Errorf("got %d distinct titles, want %d (one per status)", len(seen), len(statuses))
	}
}

// TestToolCallEmptyArgsAndResult proves criterion #565: empty args and/or
// result still return a valid Section without panicking.
func TestToolCallEmptyArgsAndResult(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("ToolCall panicked on empty args/result: %v", r)
		}
	}()

	sec := ToolCall("noop", "", ToolCallPending, "")
	if sec.Title == "" {
		t.Error("Title = \"\", want non-empty")
	}
	if sec.Content == "" {
		t.Error("Content = \"\", want non-empty (should contain the Args:/Result: labels even when values are empty)")
	}
}

// TestToolCallCursorHighlightNotClobbered proves criterion #563: when an
// Model renders a ToolCall-produced Section on the cursor row,
// the ENTIRE title text is wrapped in the cursor's highlight style, not
// just a leading segment. This is the empirical check for the
// picker.Item/color-picker-style bug: if ToolCall had embedded a
// separately-Render-ed sub-style (e.g. widgets.Badge) into Title, that
// sub-style's own trailing ansi.Reset would terminate the outer
// cursorStyle's styling partway through the line, leaving a plain-text
// (unstyled) segment in the middle of what should be a fully highlighted
// row.
func TestToolCallCursorHighlightNotClobbered(t *testing.T) {
	sec := ToolCall("read_file", "args", ToolCallRunning, "result")

	m := New(sec)
	m.SetCursor(0)

	out := m.View()

	// Isolate the rendered header line (accordion.View writes exactly one
	// line per collapsed section).
	line := strings.SplitN(out, "\n", 2)[0]

	cursorStyle := m.Theme.ResolvedStates().Selected.Bold()
	wantTitle := cursorStyle.Render(sec.Title)

	if !strings.Contains(line, wantTitle) {
		t.Fatalf("rendered cursor line = %q, want it to contain the whole title wrapped in one cursorStyle.Render call: %q", line, wantTitle)
	}

	// Empirically confirm there is no reset sequence embedded BEFORE the
	// end of the rendered title (which would mean some sub-style inside
	// the title closed the highlight early). The only reset allowed is the
	// one terminating the single outer cursorStyle.Render call, i.e. it
	// must be the last thing in the line, not something in the middle
	// followed by more plain-text title content.
	idx := strings.Index(line, wantTitle)
	if idx == -1 {
		t.Fatalf("could not locate wantTitle within rendered line %q", line)
	}
	afterTitle := line[idx+len(wantTitle):]
	if strings.Contains(afterTitle, ansi.Reset) {
		t.Errorf("found an extra reset after the expected single styled title in %q; title may contain an embedded pre-terminated sub-style", line)
	}

	// The rendered title itself must contain exactly one reset (the outer
	// cursorStyle's own trailing Reset), proving no inner Render call
	// snuck an early reset into the middle of the title text.
	if got := strings.Count(wantTitle, ansi.Reset); got != 1 {
		t.Fatalf("wantTitle = %q contains %d resets, want exactly 1 (title must carry no embedded ANSI of its own)", wantTitle, got)
	}
	if got := strings.Count(sec.Title, "\x1b["); got != 0 {
		t.Fatalf("sec.Title = %q contains %d embedded escape sequences, want 0 (accordion applies all title styling itself)", sec.Title, got)
	}
}
