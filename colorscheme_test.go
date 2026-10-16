package tui

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
)

func TestColorSchemeAndInBandResizeOffEmitNothing(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(staticModel{}, WithOutput(out))
	p.enterModes()
	p.leaveModes()
	if got := string(read()); strings.Contains(got, "?2031") || strings.Contains(got, "?2048") {
		t.Errorf("mode sequences emitted without the options: %q", got)
	}
}

func TestColorSchemeAndInBandResizeEnterAndLeave(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(staticModel{}, WithOutput(out), WithColorSchemeReports(true), WithInBandResize(true))
	p.enterModes()
	got := string(read())
	for _, seq := range []string{ansi.ColorSchemeReportingEnable, ansi.InBandResizeEnable} {
		if !strings.Contains(got, seq) {
			t.Errorf("enterModes did not send %q: %q", seq, got)
		}
	}
	before := len(read())
	p.leaveModes()
	left := string(read()[before:])
	for _, seq := range []string{ansi.ColorSchemeReportingDisable, ansi.InBandResizeDisable} {
		if !strings.Contains(left, seq) {
			t.Errorf("leaveModes did not send %q: %q", seq, left)
		}
	}
	if strings.Index(left, ansi.InBandResizeDisable) > strings.Index(left, ansi.ColorSchemeReportingDisable) {
		t.Errorf("modes should be undone in reverse order: %q", left)
	}
}

func TestColorSchemeReportSwapsTheAutoTheme(t *testing.T) {
	write, _, wait := bgRun(t, themedModel{recorderModel: recorderModel{quitAfter: 3}}, WithTheme(testAuto))
	write("\x1b[?997;2n") // light
	write("\x1b[?997;1n") // dark again
	m := wait().model.(themedModel)
	if len(m.themes) != 3 || m.themes[0] != testDark || m.themes[1] != testLight || m.themes[2] != testDark {
		t.Fatalf("themes delivered = %v, want [Dark, Light, Dark]", m.themes)
	}
	var seen []ColorSchemeEvent
	for _, msg := range m.received {
		if ev, ok := msg.(ColorSchemeEvent); ok {
			seen = append(seen, ev)
		}
	}
	if len(seen) != 2 || seen[0].Dark || !seen[1].Dark {
		t.Errorf("ColorSchemeEvents reaching Update = %v, want [light, dark]", seen)
	}
}

func TestInBandResizeReportBecomesAResizeMsg(t *testing.T) {
	write, _, wait := bgRun(t, recorderModel{quitAfter: 3})
	write("\x1b[48;33;111;0;0t")
	write("qq")
	var got []ResizeMsg
	for _, m := range wait().model.(recorderModel).received {
		if r, ok := m.(ResizeMsg); ok {
			got = append(got, r)
		}
	}
	found := false
	for _, r := range got {
		if r.Width == 111 && r.Height == 33 {
			found = true
		}
	}
	if !found {
		t.Errorf("no ResizeMsg of 111x33 after the in-band report; resizes seen: %v", got)
	}
}
