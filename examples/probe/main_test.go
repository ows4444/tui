package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
)

func update(m model, msgs ...tui.Msg) model {
	for _, msg := range msgs {
		next, _ := m.Update(msg)
		m = next.(model)
	}
	return m
}

func TestFocusEventsAreCounted(t *testing.T) {
	m := update(newModel(ansi.TrueColor),
		tui.FocusEvent{Focused: true}, tui.FocusEvent{Focused: false}, tui.FocusEvent{Focused: false})
	if m.focusIn != 1 || m.focusOut != 2 || m.focused != "out" {
		t.Errorf("focus state = in %d, out %d, last %q", m.focusIn, m.focusOut, m.focused)
	}
}

func TestBackgroundReplyPicksThemeAndTimeoutIsReported(t *testing.T) {
	m := update(newModel(ansi.TrueColor), tui.BackgroundColorEvent{R: 250, G: 250, B: 250})
	if m.bg != "rgb(250,250,250)" || m.theme != "light" {
		t.Errorf("light reply: bg=%q theme=%q", m.bg, m.theme)
	}
	m = update(newModel(ansi.TrueColor), tui.BackgroundColorEvent{R: 10, G: 10, B: 10})
	if m.theme != "dark" {
		t.Errorf("dark reply: theme=%q", m.theme)
	}
	m = update(newModel(ansi.TrueColor), tui.BackgroundUnknownMsg{})
	if m.bg != "no answer" || m.theme != "-" {
		t.Errorf("timeout: bg=%q theme=%q", m.bg, m.theme)
	}
}

func TestQuitKeys(t *testing.T) {
	for _, k := range []tui.Key{{Type: tui.KeyCtrlC}, {Type: tui.KeyRunes, Text: "q"}} {
		if _, cmd := newModel(ansi.TrueColor).Update(k); cmd == nil {
			t.Errorf("%v should quit", k)
		}
	}
	if _, cmd := newModel(ansi.TrueColor).Update(tui.Key{Type: tui.KeyRunes, Text: "x"}); cmd != nil {
		t.Error("x should not quit")
	}
}

func TestViewAndSummaryReportEverything(t *testing.T) {
	m := update(newModel(ansi.ANSI256),
		tui.FocusEvent{Focused: true}, tui.BackgroundColorEvent{R: 30, G: 30, B: 30})
	v := ansi.StripANSI(m.View())
	for _, want := range []string{"256-colour", "in=1 out=0", "rgb(30,30,30)", "theme=dark", "probe: profile=256-colour focus_in=1 focus_out=0 background=rgb(30,30,30) theme=dark"} {
		if !strings.Contains(v, want) {
			t.Errorf("View lacks %q:\n%s", want, v)
		}
	}
	if got, want := m.summary(), "probe: profile=256-colour focus_in=1 focus_out=0 background=rgb(30,30,30) theme=dark"; got != want {
		t.Errorf("summary = %q", got)
	}
}

func TestProfileNames(t *testing.T) {
	want := map[ansi.Profile]string{ansi.NoColor: "none", ansi.ANSI16: "16-colour", ansi.ANSI256: "256-colour", ansi.TrueColor: "truecolor"}
	for p, name := range want {
		if profileName(p) != name {
			t.Errorf("profileName(%v) = %q, want %q", p, profileName(p), name)
		}
	}
}

func TestGradientIsTruecolorStrip(t *testing.T) {
	g := gradient(10)
	if strings.Count(g, "\x1b[48;2;") != 10 || ansi.Width(g) != 10 {
		t.Errorf("gradient = %q (width %d)", g, ansi.Width(g))
	}
}

// fakeTerm answers DSR 6 like a terminal that advances by adv[text] columns.
type fakeTerm struct {
	out   strings.Builder
	adv   map[string]int
	reply bool
	last  string
}

func (f *fakeTerm) Write(p []byte) (int, error) {
	s := string(p)
	f.out.WriteString(s)
	if strings.HasSuffix(s, "\x1b[6n") {
		f.last = strings.TrimSuffix(strings.TrimPrefix(s, "\r\x1b[2K"), "\x1b[6n")
	}
	return len(p), nil
}

func (f *fakeTerm) read(time.Duration) (string, bool) {
	if !f.reply {
		return "", false
	}
	return fmt.Sprintf("\x1b[1;%dR", f.adv[f.last]+1), true
}

func TestClustersReportMeasuredAndWidth(t *testing.T) {
	f := &fakeTerm{reply: true, adv: map[string]int{}}
	for i, s := range samples {
		f.adv[s.text] = i + 1
	}
	res := measureClusters(f, f.read, time.Millisecond)
	if len(res) != len(samples) {
		t.Fatalf("got %d results", len(res))
	}
	out := formatClusters(res)
	for i, s := range samples {
		if !res[i].answered || res[i].measured != i+1 || res[i].width != ansi.Width(s.text) {
			t.Errorf("%s: %+v", s.name, res[i])
		}
		want := fmt.Sprintf("measured=%d ansi.Width=%d", i+1, ansi.Width(s.text))
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
}

func TestClustersSilentTerminalIsReportedNotGuessed(t *testing.T) {
	f := &fakeTerm{adv: map[string]int{}}
	res := measureClusters(f, f.read, time.Millisecond)
	out := formatClusters(res)
	if strings.Contains(out, "measured=") {
		t.Errorf("guessed a value:\n%s", out)
	}
	if strings.Count(out, "no answer") != len(samples) {
		t.Errorf("want no answer per sample:\n%s", out)
	}
	for _, r := range res {
		if r.answered || r.measured != 0 {
			t.Errorf("%+v", r)
		}
	}
}

func TestParseCPR(t *testing.T) {
	if c, ok := parseCPR("\x1b[12;7R"); !ok || c != 7 {
		t.Errorf("got %d %v", c, ok)
	}
	if _, ok := parseCPR("junk"); ok {
		t.Error("junk parsed")
	}
}

func TestFocusLogTimestampsEventsFromFakeSource(t *testing.T) {
	base := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	var tick int
	clock := func() time.Time { tick++; return base.Add(time.Duration(tick) * 10 * time.Millisecond) }
	m := newModel(ansi.TrueColor)
	m.log = newFocusLog(clock)
	m.log.markEnabled() // +10ms
	// A fake event source: an initial focus-in, another focus-in, then out.
	m = update(m, tui.FocusEvent{Focused: true}, tui.FocusEvent{Focused: true}, tui.FocusEvent{Focused: false})
	want := []string{
		"focus reporting enabled at 03:04:05.010",
		"#1 03:04:05.020 focus-in (+10ms)",
		"#2 03:04:05.030 focus-in (+20ms)",
		"#3 03:04:05.040 focus-out (+30ms)",
	}
	got := m.log.lines()
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("lines = %q, want %q", got, want)
	}
	if m.focusIn != 2 || m.focusOut != 1 {
		t.Errorf("counts in=%d out=%d", m.focusIn, m.focusOut)
	}
	if v := m.View(); !strings.Contains(v, "focus-in (+10ms)") {
		t.Errorf("view lacks timeline:\n%s", v)
	}
}

func TestKeysAndSizeAreRecordedForExitSummary(t *testing.T) {
	m := update(newModel(ansi.TrueColor),
		tui.Key{Type: tui.KeyUp}, tui.ResizeMsg{Width: 100, Height: 40})
	if got := strings.Join(m.keys, ","); got != "up" {
		t.Errorf("keys = %q, want up", got)
	}
	if m.size != "100x40" {
		t.Errorf("size = %q, want 100x40", m.size)
	}
	if v := ansi.StripANSI(m.View()); !strings.Contains(v, "100x40") {
		t.Errorf("View lacks the size:\n%s", v)
	}
}
