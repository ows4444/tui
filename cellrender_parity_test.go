package tui

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// When a frame contains colon-parameter SGR or OSC 8 links, the cell renderer
// shall render it without falling back.
func TestCellRendererDrawsColonSGRAndLinksWithoutFallback(t *testing.T) {
	const bel = "\x1b]8;;http://x\x07"
	const st = "\x1b]8;id=1;http://y\x1b\\"
	const end = "\x1b]8;;\x07"
	for _, view := range []string{
		"\x1b[4:3mcurly\x1b[0m",
		"\x1b[38:2::1:2:3mrgb\x1b[0m",
		"\x1b[48:2:0:9:8:7mrgb+cs\x1b[0m",
		"\x1b[38:5:99;1mx256\x1b[0m",
		"\x1b[4:0mplain\x1b[m",
		bel + "link" + end,
		st + "link é 世" + end + " after",
		"a " + bel + "b\x1b[1mc" + end + "d\x1b[0me",
	} {
		var buf bytes.Buffer
		p := equivProgram(&buf, 40, true)
		p.model = staticModel{view: view}
		p.render()
		if !p.cells.Valid() || p.cells.Reason() != "" {
			t.Errorf("%q fell back (%q)", view, p.cells.Reason())
		}
		if got, want := ansiStrip(buf.String()), ansiStrip(view); !strings.Contains(got, want) {
			t.Errorf("%q: visible text %q, want %q", view, got, want)
		}
	}
}

func ansiStrip(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != 0x1b {
			b.WriteByte(s[i])
			continue
		}
		i++
		if i < len(s) && s[i] == ']' {
			for i < len(s) && s[i] != 0x07 && !(s[i] == 0x1b && i+1 < len(s) && s[i+1] == '\\') {
				i++
			}
			if i < len(s) && s[i] == 0x1b {
				i++
			}
			continue
		}
		for i+1 < len(s) && (s[i+1] < 0x40 || s[i+1] > 0x7e) {
			i++
		}
		i++
		for i < len(s) && (s[i] < 0x40 || s[i] > 0x7e) && s[i] != 0x1b {
			i++
		}
		// consume final byte of the CSI
	}
	return b.String()
}

func TestCellRendererEmitsColonSGRAndLinkAndClosesThem(t *testing.T) {
	var buf bytes.Buffer
	p := equivProgram(&buf, 40, true)
	p.model = staticModel{view: "\x1b[4:3;38:2::1:2:3mab\x1b[0m \x1b]8;;http://x\x07go\x1b]8;;\x07"}
	p.render()
	out := buf.String()
	for _, want := range []string{"4:3", "38;2;1;2;3", "\x1b]8;;http://x\x07go", "\x1b]8;;\x1b\\"} {
		if !strings.Contains(out, want) {
			t.Errorf("output %q lacks %q", out, want)
		}
	}
	// Changing one cell of the link rewrites only that cell, and the
	// link is closed again before the frame ends.
	buf.Reset()
	p.model = staticModel{view: "\x1b[4:3;38:2::1:2:3mab\x1b[0m \x1b]8;;http://x\x07gX\x1b]8;;\x07"}
	p.render()
	out = buf.String()
	if !p.cells.Valid() || strings.Contains(out, "ab") || !strings.Contains(out, "X") {
		t.Errorf("second frame %q is not a one-cell diff", out)
	}
	if strings.Count(out, "\x1b]8;http") != 0 && !strings.Contains(out, "\x1b]8;;\x1b\\") {
		t.Errorf("link left open: %q", out)
	}
}

// When a whole frame falls back, the frame log shall record kind=fallback and
// the reason.
func TestFrameLogRecordsCellRendererFallbackReason(t *testing.T) {
	var out, log bytes.Buffer
	p := NewProgram(staticModel{}, WithOutput(&out), WithFrameLog(&log), WithCellRenderer(true))
	p.frameStats = frameStats{rows: 3, changed: 3, full: true, fallback: "view_taller_than_terminal"}
	p.logFrame(0, time.Now())
	p.frameStats = frameStats{rows: 3, changed: 1}
	p.logFrame(0, time.Now())
	lines := strings.Split(strings.TrimSpace(log.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("log = %q", log.String())
	}
	if !strings.Contains(lines[0], " kind=fallback ") || !strings.HasSuffix(lines[0], " reason=view_taller_than_terminal") {
		t.Errorf("fallback line = %q", lines[0])
	}
	if strings.Contains(lines[1], "fallback") || strings.Contains(lines[1], "reason=") {
		t.Errorf("non-fallback line = %q", lines[1])
	}
}

// When one row falls back, the frame log shall name the row and the reason,
// and the frame is still a cell frame.
func TestFrameLogNamesFallbackRowAndReason(t *testing.T) {
	var out, log bytes.Buffer
	p := NewProgram(staticModel{view: "ok\na\x01b\nok\n\x1b[6mx"}, WithOutput(&out), WithFrameLog(&log), WithCellRenderer(true))
	p.width, p.height = 40, 100
	p.render()
	p.model = staticModel{view: "OK\na\x01b\nok\n\x1b[6mx"} // only row 0 changes: the bad rows are not redrawn
	p.render()
	p.model = staticModel{view: "OK\na\x01c\nok\n\x1b[6mx"}
	p.render()
	lines := strings.Split(strings.TrimSpace(log.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("log = %q", log.String())
	}
	if !strings.Contains(lines[0], " kind=full ") || !strings.HasSuffix(lines[0], " fallback_rows=1:control_character,3:unsupported_SGR") {
		t.Errorf("first frame line = %q", lines[0])
	}
	if strings.Contains(lines[1], "fallback") {
		t.Errorf("a frame that redraws no bad row names one: %q", lines[1])
	}
	if !strings.Contains(lines[2], " kind=diff ") || !strings.HasSuffix(lines[2], " fallback_rows=1:control_character") {
		t.Errorf("third frame line = %q", lines[2])
	}
}
