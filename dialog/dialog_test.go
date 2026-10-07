package dialog

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/boxdraw"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

func key(t tui.KeyType) tui.Key { return tui.Key{Type: t} }

func TestNewIsOpen(t *testing.T) {
	m := New("Title", "Message")
	if !m.Open() {
		t.Error("New() should return an already-open dialog")
	}
}

func TestShowHide(t *testing.T) {
	m := New("Title", "")
	m.Hide()
	if m.Open() {
		t.Error("Hide() should close the dialog")
	}
	m.Show()
	if !m.Open() {
		t.Error("Show() should open the dialog")
	}
}

func TestEnterDismisses(t *testing.T) {
	m := New("Title", "")
	next, cmd := m.Update(key(tui.KeyEnter))
	if next.Open() {
		t.Error("Enter should close the dialog")
	}
	if cmd == nil {
		t.Fatal("Enter should return a non-nil Cmd")
	}
	if _, ok := cmd().(DismissedMsg); !ok {
		t.Errorf("Cmd produced %T, want DismissedMsg", cmd())
	}
}

func TestEscDismisses(t *testing.T) {
	m := New("Title", "")
	next, cmd := m.Update(key(tui.KeyEsc))
	if next.Open() {
		t.Error("Esc should close the dialog")
	}
	if cmd == nil {
		t.Fatal("Esc should return a non-nil Cmd")
	}
}

func TestOtherKeysDoNotDismiss(t *testing.T) {
	m := New("Title", "")
	next, cmd := m.Update(key(tui.KeyDown))
	if !next.Open() {
		t.Error("an unrelated key should not close the dialog")
	}
	if cmd != nil {
		t.Error("an unrelated key should not return a Cmd")
	}
}

func TestUpdateIsNoOpWhenClosed(t *testing.T) {
	m := New("Title", "")
	m.Hide()
	next, cmd := m.Update(key(tui.KeyEnter))
	if next.Open() {
		t.Error("Update on a closed dialog should stay closed")
	}
	if cmd != nil {
		t.Error("Update on a closed dialog should not return a Cmd, even for Enter/Esc")
	}
}

func TestNonKeyMsgIgnored(t *testing.T) {
	m := New("Title", "")
	next, cmd := m.Update(tui.ResizeMsg{Width: 10, Height: 10})
	if !next.Open() || cmd != nil {
		t.Error("a non-Key Msg should be a no-op")
	}
}

func TestRenderReturnsBaseUnchangedWhenClosed(t *testing.T) {
	base := "some\nbackground\ncontent"
	m := New("Title", "")
	m.Hide()
	if got := m.Render(base); got != base {
		t.Errorf("Render() with a closed dialog = %q, want base unchanged", got)
	}
}

// TestRenderCentersDialog independently hand-verifies the box's expected
// dimensions before asserting Render's placement, rather than comparing
// against a re-derivation of the same centering formula Render itself
// uses — that would only prove Render is internally consistent with
// itself, not that the math is actually correct.
func TestRenderCentersDialog(t *testing.T) {
	base := strings.TrimSuffix(strings.Repeat(strings.Repeat("a", 20)+"\n", 10), "\n")

	m := New("Hi", "")
	got := m.Render(base)

	titleStyled := ansi.NewStyle().Bold().Foreground(m.Theme.Primary).Render("Hi")
	box := boxdraw.Draw(layout.NewBox().BorderColor(m.Theme.BorderColor), m.Theme.Border, 1, 0, titleStyled)

	// Hand-verified: "Hi" padded 1 on all sides has content width 2, inner
	// width 2+1+1=4, box width 4+2(border chars)=6, box height =
	// 1(top border) + 1(pad) + 1("Hi") + 1(pad) + 1(bottom border) = 5.
	// Centered in a 20-wide, 10-tall base: x=(20-6)/2=7, y=(10-5)/2=2.
	boxLines := strings.Split(box, "\n")
	if w := ansi.Width(boxLines[0]); w != 6 {
		t.Fatalf("box width = %d, want 6 (this test's centering math assumes this)", w)
	}
	if len(boxLines) != 5 {
		t.Fatalf("box height = %d, want 5", len(boxLines))
	}

	want := layout.Overlay(base, box, 7, 2)
	if got != want {
		t.Errorf("Render() did not composite at the hand-verified center (7, 2)")
	}
}

func TestRenderWithMessageIncludesBothLines(t *testing.T) {
	base := strings.TrimSuffix(strings.Repeat(strings.Repeat("a", 30)+"\n", 15), "\n")
	m := New("Confirm", "Are you sure?")
	got := m.Render(base)
	visible := ansi.StripANSI(got)
	if !strings.Contains(visible, "Confirm") {
		t.Error("Render() should contain the title")
	}
	if !strings.Contains(visible, "Are you sure?") {
		t.Error("Render() should contain the message")
	}
}

// Word-wrapping itself is tested in ansi.Wrap's own test suite;
// TestRenderWrapsLongMessageToFitWidth below tests that Render actually
// applies it when Width is set, which is the integration point that
// actually matters here.
func TestRenderWrapsLongMessageToFitWidth(t *testing.T) {
	base := strings.TrimSuffix(strings.Repeat(strings.Repeat("a", 40)+"\n", 15), "\n")
	m := New("Hi", "this message is definitely longer than the dialog width")
	m.Width = 15

	got := m.Render(base)
	lines := strings.Split(got, "\n")
	for _, l := range lines {
		if w := ansi.Width(l); w > 40 {
			t.Fatalf("row %q has width %d, wider than the 40-wide base — Width should have kept the dialog contained", l, w)
		}
	}
}

func TestRenderPreservesBaseDimensions(t *testing.T) {
	base := strings.TrimSuffix(strings.Repeat(strings.Repeat("a", 20)+"\n", 10), "\n")
	m := New("Hi", "message here")
	got := m.Render(base)
	if got2, want := len(strings.Split(got, "\n")), 10; got2 != want {
		t.Errorf("Render() produced %d lines, want %d (base's own height, unchanged)", got2, want)
	}
}

func TestRenderDrawsBorderInThemeBorderColor(t *testing.T) {
	base := strings.TrimSuffix(strings.Repeat(strings.Repeat("a", 20)+"\n", 10), "\n")
	m := New("Hi", "")
	m.Theme.BorderColor = ansi.Red
	got := m.Render(base)

	red := ansi.NewStyle().Foreground(ansi.Red)
	b := m.Theme.Border
	// "Hi" + 1 padding each side: inner width 4.
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
	// The title keeps its own styling.
	if want := ansi.NewStyle().Bold().Foreground(m.Theme.Primary).Render("Hi"); !strings.Contains(got, want) {
		t.Errorf("title styling lost")
	}
}

func TestSwitchingThemeChangesBorderColor(t *testing.T) {
	base := strings.TrimSuffix(strings.Repeat(strings.Repeat("a", 20)+"\n", 10), "\n")
	render := func(th theme.Theme) string {
		m := New("Hi", "")
		m.Theme = th
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

	m := New("Hi", "")
	m.Theme.BorderColor = nil // border characters remain, uncoloured
	out := m.Render(base)
	b := m.Theme.Border
	if !strings.Contains(out, b.TopLeft+strings.Repeat(b.Top, 4)+b.TopRight) {
		t.Errorf("uncoloured border row missing (or wrapped in a style)")
	}

	z := New("Hi", "")
	z.Theme = theme.Theme{}
	z.Render(base) // must not panic
}
