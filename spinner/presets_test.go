package spinner

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
	"github.com/ows4444/tui/theme"
)

func TestEveryPresetIsWellFormed(t *testing.T) {
	seen := map[string]bool{}
	ps := Presets()
	if len(ps) < 15 {
		t.Fatalf("only %d presets", len(ps))
	}
	for _, p := range ps {
		if p.Name == "" || seen[strings.ToLower(p.Name)] {
			t.Errorf("name %q is empty or repeated", p.Name)
		}
		seen[strings.ToLower(p.Name)] = true
		if len(p.Frames) < 2 || p.Interval < 20*time.Millisecond || p.Interval > time.Second {
			t.Errorf("%s: %d frames at %v", p.Name, len(p.Frames), p.Interval)
		}
		for _, set := range [][]string{p.Frames, p.ASCII} {
			for _, fr := range set {
				if fr != ansi.Sanitize(fr) || strings.Contains(fr, "\n") {
					t.Errorf("%s: frame %q holds control characters", p.Name, fr)
				}
				if w := ansi.Width(fr); w != ansi.Width(set[0]) || w == 0 {
					t.Errorf("%s: frame %q is %d wide, the first is %d: the line would jitter", p.Name, fr, w, ansi.Width(set[0]))
				}
			}
		}
		if isASCII(p.Frames) != (p.ASCII == nil) {
			t.Errorf("%s: ASCII fallback %v for frames that are ASCII=%v", p.Name, p.ASCII, isASCII(p.Frames))
		}
		if p.ASCII != nil && !isASCII(p.ASCII) {
			t.Errorf("%s: the ASCII fallback is not ASCII: %q", p.Name, p.ASCII)
		}
	}
}

func TestPresetsAreCopies(t *testing.T) {
	a := Presets()
	a[0].Frames[0] = "changed"
	a[0].Name = "changed"
	if b := Presets(); b[0].Frames[0] == "changed" || b[0].Name == "changed" {
		t.Error("Presets handed out the set itself")
	}
	p, ok := Lookup("DOTS")
	if !ok || p.Name != "dots" {
		t.Fatalf("Lookup(DOTS) = %+v, %v", p, ok)
	}
	p.Frames[0] = "changed"
	if q, _ := Lookup("dots"); q.Frames[0] == "changed" {
		t.Error("Lookup handed out the set itself")
	}
	if _, ok := Lookup("nope"); ok {
		t.Error("Lookup found a preset that does not exist")
	}
}

func TestWithPresetSetsFramesIntervalAndRestartsAtTheFirstFrame(t *testing.T) {
	m := New()
	m.frame = 5
	p, _ := Lookup("arrow")
	m = m.WithPreset(p)
	if !slices.Equal(m.Frames, p.Frames) || m.Interval != p.Interval || m.frame != 0 {
		t.Errorf("Frames %q Interval %v frame %d", m.Frames, m.Interval, m.frame)
	}
	p.Frames[0] = "changed" // a later change to the caller's copy must not reach the model
	if m.Frames[0] == "changed" {
		t.Error("WithPreset kept the caller's slice")
	}
	if got := ansi.StripANSI(m.View()); got != "←" {
		t.Errorf("View = %q", got)
	}
}

func TestPresetFallsBackToAsciiOnAnAsciiTheme(t *testing.T) {
	p, _ := Lookup("arc")
	m := New().WithPreset(p)
	if got := ansi.StripANSI(m.View()); got != "◜" {
		t.Errorf("unicode theme View = %q", got)
	}
	m.Theme = theme.DarkTheme().ASCII()
	if got := ansi.StripANSI(m.View()); got != "-" {
		t.Errorf("ascii theme View = %q, want the plain spinner's first frame", got)
	}
	buf := cellbuf.New(4, 1)
	m.DrawCells(buf, buf.Bounds())
	if got := buf.At(0, 0).Cluster; got != "-" {
		t.Errorf("DrawCells on an ascii theme drew %q", got)
	}
	// A preset that is already ASCII is drawn as it is on either theme.
	line, _ := Lookup("simpleDots")
	m = New().WithPreset(line)
	m.Theme = theme.DarkTheme().ASCII()
	if got := ansi.StripANSI(m.View()); got != ".  " {
		t.Errorf("simpleDots on an ascii theme = %q", got)
	}
}

func TestDefaultAndCustomFramesAreUnchanged(t *testing.T) {
	m := New()
	if got := ansi.StripANSI(m.View()); got != "⠋" {
		t.Errorf("default View = %q", got)
	}
	m.Theme = theme.DarkTheme().ASCII()
	if got := ansi.StripANSI(m.View()); got != "|" && got != "-" && got != "/" && got != `\` {
		t.Errorf("default on an ascii theme = %q", got)
	}
	m = New()
	m.Frames = []string{"a", "b"}
	m.Theme = theme.DarkTheme().ASCII()
	if got := ansi.StripANSI(m.View()); got != "a" {
		t.Errorf("custom frames = %q", got)
	}
}

func TestAPresetAnimatesThroughAllItsFrames(t *testing.T) {
	p, _ := Lookup("toggle")
	m := New().WithPreset(p)
	m.Motion = 0
	m.Start()
	var seen []string
	for i := 0; i < 4; i++ {
		seen = append(seen, ansi.StripANSI(m.View()))
		m, _ = m.Update(tickMsg{id: m.id})
	}
	if !slices.Equal(seen, []string{"⊶", "⊷", "⊶", "⊷"}) {
		t.Errorf("frames = %q", seen)
	}
}
