package toast

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// newFast returns a Model with a tiny Duration, for tests that actually
// call the Cmd Show returns — tui.Tick really sleeps for Duration, and
// New's 3-second default would make every such test take that long for no
// benefit.
func newFast(message string) Model {
	m := New(message)
	m.Duration = time.Millisecond
	return m
}

func TestNewIsNotOpen(t *testing.T) {
	m := New("hi")
	if m.Open() {
		t.Error("New() should not be open until Show()")
	}
}

func TestShowOpensAndReturnsCmd(t *testing.T) {
	m := newFast("hi")
	cmd := m.Show()
	if !m.Open() {
		t.Error("Show() should open the toast")
	}
	if cmd == nil {
		t.Fatal("Show() should return a non-nil Cmd")
	}
	msg, ok := tui.RunCmd(context.Background(), cmd).(dismissMsg)
	if !ok {
		t.Fatalf("Show()'s Cmd produced %T, want dismissMsg", tui.RunCmd(context.Background(), cmd))
	}
	if msg.id != m.id {
		t.Errorf("dismissMsg.id = %d, want %d (current generation)", msg.id, m.id)
	}
}

func TestHideClosesImmediately(t *testing.T) {
	m := New("hi")
	m.Show()
	m.Hide()
	if m.Open() {
		t.Error("Hide() should close the toast")
	}
}

func TestDismissMsgClosesAndEmitsDismissedMsg(t *testing.T) {
	m := New("hi")
	m.Show()
	next, cmd := m.Update(dismissMsg{id: m.id})
	if next.Open() {
		t.Error("a matching dismissMsg should close the toast")
	}
	if cmd == nil {
		t.Fatal("a matching dismissMsg should return a non-nil Cmd")
	}
	if _, ok := tui.RunCmd(context.Background(), cmd).(DismissedMsg); !ok {
		t.Errorf("Cmd produced %T, want DismissedMsg", tui.RunCmd(context.Background(), cmd))
	}
}

// TestStaleDismissIgnoredAfterReShow is the regression test for the whole
// reason Model tracks a generation id: if Show is called again before a
// previous timer fires, that earlier timer's dismissMsg must not close
// the toast the second Show() opened.
func TestStaleDismissIgnoredAfterReShow(t *testing.T) {
	m := newFast("hi")
	firstCmd := m.Show()
	firstMsg := tui.RunCmd(context.Background(), firstCmd).(dismissMsg) // id from generation 1

	secondCmd := m.Show() // re-shown before the first timer fired; generation 2
	if !m.Open() {
		t.Fatal("toast should still be open after re-Show")
	}

	next, cmd := m.Update(firstMsg) // the stale, generation-1 dismiss arrives
	if !next.Open() {
		t.Fatal("a stale dismissMsg from a previous Show() should not close the toast")
	}
	if cmd != nil {
		t.Error("a stale dismissMsg should not emit DismissedMsg")
	}

	secondMsg := tui.RunCmd(context.Background(), secondCmd).(dismissMsg) // id from generation 2
	next, cmd = next.Update(secondMsg)
	if next.Open() {
		t.Fatal("the current generation's dismissMsg should close the toast")
	}
	if cmd == nil {
		t.Fatal("the current generation's dismissMsg should emit DismissedMsg")
	}
}

func TestDismissMsgIsNoOpWhenAlreadyClosed(t *testing.T) {
	m := newFast("hi")
	cmd := m.Show()
	msg := tui.RunCmd(context.Background(), cmd).(dismissMsg)
	m.Hide()

	next, cmd2 := m.Update(msg)
	if next.Open() {
		t.Error("Update should leave an already-closed toast closed")
	}
	if cmd2 != nil {
		t.Error("a dismissMsg on an already-closed toast should not emit a Cmd")
	}
}

func TestNonDismissMsgIgnored(t *testing.T) {
	m := New("hi")
	m.Show()
	next, cmd := m.Update(tui.ResizeMsg{Width: 10, Height: 10})
	if !next.Open() || cmd != nil {
		t.Error("a non-dismissMsg should be a no-op")
	}
}

func TestRenderReturnsBaseUnchangedWhenClosed(t *testing.T) {
	base := "some\nbackground\ncontent"
	m := New("hi")
	if got := m.Render(base); got != base {
		t.Errorf("Render() with a closed toast = %q, want base unchanged", got)
	}
}

func TestRenderPreservesBaseDimensions(t *testing.T) {
	base := strings.TrimSuffix(strings.Repeat(strings.Repeat("a", 20)+"\n", 10), "\n")
	m := New("hi")
	m.Show()
	got := m.Render(base)
	if got2, want := len(strings.Split(got, "\n")), 10; got2 != want {
		t.Errorf("Render() produced %d lines, want %d (base's own height, unchanged)", got2, want)
	}
}

func TestRenderPositions(t *testing.T) {
	base := strings.TrimSuffix(strings.Repeat(strings.Repeat("a", 20)+"\n", 10), "\n")

	tests := []struct {
		name   string
		pos    Position
		inTop  bool
		inLeft bool
	}{
		{"TopLeft", TopLeft, true, true},
		{"TopRight", TopRight, true, false},
		{"BottomLeft", BottomLeft, false, true},
		{"BottomRight", BottomRight, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New("HI")
			m.Position = tt.pos
			m.Margin = 0 // flush to the edge, to test the underlying placement math directly
			m.Show()
			got := m.Render(base)
			lines := strings.Split(got, "\n")

			topRow := ansi.StripANSI(lines[0])
			bottomRow := ansi.StripANSI(lines[len(lines)-1])
			foundInTop := topRow != strings.Repeat("a", 20)
			foundInBottom := bottomRow != strings.Repeat("a", 20)

			if tt.inTop && !foundInTop {
				t.Errorf("%s: expected the box to touch the top row, top row = %q", tt.name, topRow)
			}
			if !tt.inTop && !foundInBottom {
				t.Errorf("%s: expected the box to touch the bottom row, bottom row = %q", tt.name, bottomRow)
			}

			leftCol := topRow
			if !tt.inTop {
				leftCol = bottomRow
			}
			startsAtLeft := strings.HasPrefix(leftCol, "a") == false // box border char, not 'a', at column 0
			if tt.inLeft && !startsAtLeft {
				t.Errorf("%s: expected the box to touch the left edge, row = %q", tt.name, leftCol)
			}
		})
	}
}

// TestDefaultMarginLeavesEdgesUntouched checks New's default Margin (1)
// actually insets the toast — the regression test for the visual
// roughness that showed up composing a BottomRight toast flush against a
// bordered base in examples/dashboard: the toast's own border ended up
// exactly overwriting the base's border at that corner. Margin exists
// specifically so New's default doesn't reproduce that.
func TestDefaultMarginLeavesEdgesUntouched(t *testing.T) {
	base := strings.TrimSuffix(strings.Repeat(strings.Repeat("a", 20)+"\n", 10), "\n")
	m := New("HI") // default Margin is 1
	m.Show()
	got := m.Render(base)
	lines := strings.Split(got, "\n")

	topRow := ansi.StripANSI(lines[0])
	bottomRow := ansi.StripANSI(lines[len(lines)-1])
	if topRow != strings.Repeat("a", 20) {
		t.Errorf("top row = %q, want untouched (Margin should keep a BottomRight toast off row 0)", topRow)
	}
	if bottomRow != strings.Repeat("a", 20) {
		t.Errorf("bottom row = %q, want untouched (Margin should keep the toast off the very last row)", bottomRow)
	}

	// The row(s) the box does occupy shouldn't reach column 19 (the last
	// column) either — a flush-right box would end its border exactly
	// there.
	for i, l := range lines {
		visible := ansi.StripANSI(l)
		if visible == strings.Repeat("a", 20) {
			continue // untouched row
		}
		if strings.HasSuffix(visible, "a") {
			continue // last column is still the base's own 'a', good
		}
		t.Errorf("row %d = %q, want its last column to still be base's 'a' (Margin should keep the toast off the right edge)", i, visible)
	}
}

func TestRenderDrawsBorderInThemeBorderColor(t *testing.T) {
	base := strings.TrimSuffix(strings.Repeat(strings.Repeat("a", 20)+"\n", 10), "\n")
	m := New("hi")
	m.Theme.BorderColor = ansi.Red
	m.Show()
	got := m.Render(base)

	red := ansi.NewStyle().Foreground(ansi.Red)
	b := m.Theme.Border
	// "hi" + 1 padding each side: inner width 4.
	top := b.TopLeft + strings.Repeat(b.Top, 4) + b.TopRight
	bottom := b.BottomLeft + strings.Repeat(b.Bottom, 4) + b.BottomRight
	for _, want := range []string{red.Render(top), red.Render(bottom)} {
		if !strings.Contains(got, want) {
			t.Errorf("Render() missing coloured border row %q", want)
		}
	}
	// Three body rows, each with a coloured side at both ends (the left and
	// right glyphs are the same character for the default border).
	sides := strings.Count(got, red.Render(b.Left))
	if b.Right != b.Left {
		sides += strings.Count(got, red.Render(b.Right))
	}
	if sides != 6 {
		t.Errorf("coloured side characters = %d, want 6", sides)
	}
	// The message keeps its variant colour.
	if want := ansi.NewStyle().Foreground(m.Variant.Color(m.Theme)).Render("hi"); !strings.Contains(got, want) {
		t.Errorf("message styling lost")
	}
}

func TestSwitchingThemeChangesBorderColor(t *testing.T) {
	base := strings.TrimSuffix(strings.Repeat(strings.Repeat("a", 20)+"\n", 10), "\n")
	render := func(th theme.Theme) string {
		m := New("hi")
		m.Theme = th
		m.Show()
		return m.Render(base)
	}
	for name, th := range map[string]theme.Theme{"dark": theme.DarkTheme(), "light": theme.LightTheme()} {
		b := th.Border
		top := b.TopLeft + strings.Repeat(b.Top, 4) + b.TopRight
		if want := ansi.NewStyle().Foreground(th.BorderColor).Render(top); !strings.Contains(render(th), want) {
			t.Errorf("%s: border not drawn in that theme's BorderColor", name)
		}
	}
	if render(theme.DarkTheme()) == render(theme.LightTheme()) {
		t.Error("Dark and Light render identically")
	}
}

func TestNilBorderColorAndZeroThemeAreSafe(t *testing.T) {
	base := strings.TrimSuffix(strings.Repeat(strings.Repeat("a", 20)+"\n", 10), "\n")

	m := New("hi")
	m.Theme.BorderColor = nil // border characters remain, uncoloured
	m.Show()
	out := m.Render(base)
	b := m.Theme.Border
	if !strings.Contains(out, b.TopLeft+strings.Repeat(b.Top, 4)+b.TopRight) {
		t.Errorf("uncoloured border row missing (or wrapped in a style)")
	}

	z := New("hi")
	z.Theme = theme.Theme{}
	z.Show()
	z.Render(base) // must not panic
}
