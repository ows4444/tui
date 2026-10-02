package helpscreen

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/widgets"
)

func key(t tui.KeyType) tui.Key { return tui.Key{Type: t} }

// grid returns a w x h rectangle of '.' characters, one row per line, a
// fixed-size base to test HelpScreen placement against.
func grid(w, h int) string {
	row := strings.Repeat(".", w)
	rows := make([]string, h)
	for i := range rows {
		rows[i] = row
	}
	return strings.Join(rows, "\n")
}

// TestLifecycleMirrorsDialog proves criterion #367: New returns an
// already-open Model, Show/Hide/Open work as in dialog.Model, and Update
// dismisses on Enter/Esc while open and no-ops when closed.
func TestLifecycleMirrorsDialog(t *testing.T) {
	m := New(widgets.Hint{Key: "q", Action: "quit"})
	if !m.Open() {
		t.Fatal("New() should return an already-open help screen")
	}

	m.Hide()
	if m.Open() {
		t.Error("Hide() should close the help screen")
	}
	m.Show()
	if !m.Open() {
		t.Error("Show() should open the help screen")
	}

	next, cmd := m.Update(key(tui.KeyEnter))
	if next.Open() {
		t.Error("Enter should close the help screen")
	}
	if cmd == nil {
		t.Fatal("Enter should return a non-nil Cmd")
	}
	if _, ok := cmd().(DismissedMsg); !ok {
		t.Errorf("Cmd produced %T, want DismissedMsg", cmd())
	}

	m2 := New(widgets.Hint{Key: "q", Action: "quit"})
	next2, cmd2 := m2.Update(key(tui.KeyEsc))
	if next2.Open() {
		t.Error("Esc should close the help screen")
	}
	if cmd2 == nil {
		t.Fatal("Esc should return a non-nil Cmd")
	}

	m3 := New(widgets.Hint{Key: "q", Action: "quit"})
	next3, cmd3 := m3.Update(key(tui.KeyDown))
	if !next3.Open() {
		t.Error("an unrelated key should not close the help screen")
	}
	if cmd3 != nil {
		t.Error("an unrelated key should not return a Cmd")
	}

	m4 := New(widgets.Hint{Key: "q", Action: "quit"})
	m4.Hide()
	next4, cmd4 := m4.Update(key(tui.KeyEnter))
	if next4.Open() {
		t.Error("Update on a closed help screen should stay closed")
	}
	if cmd4 != nil {
		t.Error("Update on a closed help screen should not return a Cmd, even for Enter/Esc")
	}
}

// TestRenderReturnsBaseUnchangedWhenClosed proves criterion #368: Render
// returns base unchanged while closed.
func TestRenderReturnsBaseUnchangedWhenClosed(t *testing.T) {
	base := "some\nbackground\ncontent"
	m := New(widgets.Hint{Key: "q", Action: "quit"})
	m.Hide()
	if got := m.Render(base); got != base {
		t.Errorf("Render() with a closed help screen = %q, want base unchanged", got)
	}
}

// TestRenderListsEachHintOnItsOwnLine proves criterion #369: each Hint is
// rendered on its own line via widgets.KeyHint, unlike widgets.KeyHints
// which joins them into a single line.
func TestRenderListsEachHintOnItsOwnLine(t *testing.T) {
	base := grid(60, 20)
	hints := []widgets.Hint{
		{Key: "up/down", Action: "move"},
		{Key: "enter", Action: "select"},
		{Key: "q", Action: "quit"},
	}
	m := New(hints...)
	got := ansi.StripANSI(m.Render(base))

	for _, h := range hints {
		wantLine := ansi.StripANSI(widgets.KeyHint(h.Key, h.Action))
		found := false
		for _, line := range strings.Split(got, "\n") {
			if strings.Contains(line, wantLine) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Render() should contain %q on its own line:\n%s", wantLine, got)
		}
	}

	joined := ansi.StripANSI(widgets.KeyHints("  ", hints...))
	if strings.Contains(got, joined) {
		t.Errorf("Render() should not join hints into a single KeyHints line, got:\n%s", got)
	}
}

// TestRenderSpansFullWidthOfBase proves criterion #370: Render's box spans
// the full width of base, not a small anchored box like Popover/Tooltip or
// a content-sized centered box like dialog.Model.
func TestRenderSpansFullWidthOfBase(t *testing.T) {
	base := grid(50, 10)
	m := New(widgets.Hint{Key: "q", Action: "quit"})
	got := m.Render(base)

	lines := strings.Split(got, "\n")
	boxWidth := ansi.Width(lines[0])
	if boxWidth != 50 {
		t.Errorf("box width = %d, want 50 (full width of base)", boxWidth)
	}
}

// TestRenderBordersAndColorsViaTheme proves criterion #371: the box is
// bordered and colored via the given Theme, matching dialog.Model and
// popover.Model's bordered-box convention.
func TestRenderBordersAndColorsViaTheme(t *testing.T) {
	base := grid(30, 10)
	m := New(widgets.Hint{Key: "q", Action: "quit"})
	got := m.Render(base)

	content := widgets.KeyHint("q", "quit")
	wantBox := layout.NewBox().Border(m.Theme.Border).BorderColor(m.Theme.BorderColor).PaddingAll(1).Width(30 - 4).Render(content)
	want := layout.Overlay(base, wantBox, 0, 0)

	if got != want {
		t.Errorf("Render() did not composite a themed bordered box at (0,0):\ngot:\n%s\nwant:\n%s", got, want)
	}

	visible := ansi.StripANSI(got)
	if !strings.Contains(visible, m.Theme.Border.TopLeft) {
		t.Errorf("Render() should draw a border using the theme's Border style")
	}
}

// TestFromKeymap proves criterion #93 for helpscreen: a registered binding is
// listed with its keys and description, with no separate hint list.
func TestFromKeymap(t *testing.T) {
	var r keymap.Registry
	r.Add(keymap.Binding{Keys: []string{"q", "ctrl+c"}, Desc: "quit"})
	out := ansi.StripANSI(FromKeymap(&r, "").Render(grid(40, 5)))
	if !strings.Contains(out, "[q/ctrl+c] quit") {
		t.Errorf("binding not listed:\n%s", out)
	}
}
