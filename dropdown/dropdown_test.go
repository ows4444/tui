package dropdown

import (
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/input"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

func key(t tui.KeyType) tui.Key { return tui.Key{Type: t} }

func runes(s string) tui.Key { return tui.Key{Type: tui.KeyRunes, Text: s, Code: []rune(s)[0]} }

func changed(cmd tui.Cmd) (ChangedMsg, bool) {
	if cmd == nil {
		return ChangedMsg{}, false
	}
	c, ok := cmd().(ChangedMsg)
	return c, ok
}

func plain(m Model) string { return ansi.StripANSI(m.View()) }

func plan() Model {
	m := New("Plan", "Free", "Pro", "Team")
	m.ID = "plan"
	m.Focus()
	return m
}

func press(m Model, keys ...tui.Key) (Model, tui.Cmd) {
	var cmd tui.Cmd
	for _, k := range keys {
		m, cmd = m.Update(k)
	}
	return m, cmd
}

func TestClosedViewShowsThePlaceholderThenTheValue(t *testing.T) {
	m := plan()
	if got := plain(m); got != "Plan: Select ▸" && !strings.HasPrefix(got, "Plan: Select") {
		t.Errorf("View = %q", got)
	}
	m.Placeholder = "Pick one"
	if got := plain(m); !strings.Contains(got, "Pick one") {
		t.Errorf("placeholder missing in %q", got)
	}
	m.SetSelected(1)
	got := plain(m)
	if !strings.HasPrefix(got, "Plan: Pro  ") || strings.Contains(got, "\n") {
		t.Errorf("closed View = %q", got)
	}
	if m.Value() != "Pro" || m.Selected() != 1 {
		t.Errorf("Value %q Selected %d", m.Value(), m.Selected())
	}
}

func TestTheControlKeepsItsWidthWhateverIsChosen(t *testing.T) {
	m := plan()
	w := ansi.Width(m.View())
	for i := -1; i < 3; i++ {
		m.SetSelected(i)
		if got := ansi.Width(m.View()); got != w || got != m.Width() {
			t.Errorf("selected %d: width %d, want %d (Width() %d)", i, got, w, m.Width())
		}
	}
	m.MinWidth = 20
	if ansi.Width(m.View()) <= w {
		t.Error("MinWidth did not widen the control")
	}
}

func TestEnterOpensAndEnterChooses(t *testing.T) {
	m, _ := press(plan(), key(tui.KeyEnter))
	if !m.Open() {
		t.Fatal("Enter did not open")
	}
	m, cmd := press(m, key(tui.KeyDown))
	if cmd != nil || m.Selected() != -1 {
		t.Fatal("moving the highlight chose something")
	}
	m, cmd = press(m, key(tui.KeyEnter))
	msg, ok := changed(cmd)
	if !ok || msg != (ChangedMsg{ID: "plan", Index: 1, Value: "Pro"}) {
		t.Fatalf("ChangedMsg = %+v, %v", msg, ok)
	}
	if m.Open() || m.Value() != "Pro" {
		t.Errorf("open %v value %q", m.Open(), m.Value())
	}
}

func TestOpenStartsOnTheChosenOption(t *testing.T) {
	m := plan()
	m.SetSelected(2)
	m, _ = press(m, key(tui.KeySpace))
	lines := strings.Split(plain(m), "\n")
	if len(lines) != 4 || !strings.Contains(lines[3], "✓ Team") {
		t.Fatalf("open View:\n%s", strings.Join(lines, "\n"))
	}
	m, cmd := press(m, key(tui.KeyEnter))
	if cmd != nil {
		t.Error("choosing the option already chosen sent a ChangedMsg")
	}
	if m.Open() {
		t.Error("Enter left the list open")
	}
}

func TestEscClosesWithoutChanging(t *testing.T) {
	m := plan()
	m.SetSelected(0)
	m, _ = press(m, key(tui.KeyEnter), key(tui.KeyDown), key(tui.KeyDown))
	m, cmd := press(m, key(tui.KeyEsc))
	if m.Open() || cmd != nil || m.Selected() != 0 {
		t.Errorf("Esc: open %v selected %d cmd %v", m.Open(), m.Selected(), cmd != nil)
	}
}

func TestClosedArrowsStepThroughTheOptions(t *testing.T) {
	m := plan()
	for _, want := range []struct {
		k    tui.KeyType
		idx  int
		sent bool
	}{
		{tui.KeyRight, 0, true}, {tui.KeyRight, 1, true}, {tui.KeyRight, 2, true}, {tui.KeyRight, 2, false},
		{tui.KeyLeft, 1, true}, {tui.KeyUp, 0, true}, {tui.KeyLeft, 0, false},
	} {
		var cmd tui.Cmd
		m, cmd = press(m, key(want.k))
		if _, sent := changed(cmd); sent != want.sent || m.Selected() != want.idx {
			t.Fatalf("after %v: selected %d (want %d), sent %v (want %v)", want.k, m.Selected(), want.idx, sent, want.sent)
		}
	}
}

func TestOpenTakesKeysAndDoesNotChangeOnTheOthers(t *testing.T) {
	m, _ := press(plan(), key(tui.KeyEnter))
	m, cmd := press(m, key(tui.KeyTab), key(tui.KeyF1), tui.Key{Type: tui.KeyRunes, Text: "q", Mod: input.ModCtrl})
	if cmd != nil || !m.Open() {
		t.Error("an unbound key changed something")
	}
}

func TestMovementClampsAndPages(t *testing.T) {
	opts := make([]string, 20)
	for i := range opts {
		opts[i] = "opt" + string(rune('a'+i))
	}
	m := New("", opts...)
	m.MaxRows = 5
	m.Focus()
	m, _ = press(m, key(tui.KeyEnter), key(tui.KeyEnd))
	if got := plain(m); !strings.Contains(got, "opts") || !strings.Contains(got, "15 above") {
		t.Errorf("after End:\n%s", got)
	}
	m, _ = press(m, key(tui.KeyUp)) // no wrap, no overflow
	m, _ = press(m, key(tui.KeyHome))
	if got := plain(m); !strings.Contains(got, "opta") || !strings.Contains(got, "15 below") || strings.Contains(got, "above") {
		t.Errorf("after Home:\n%s", got)
	}
	m, _ = press(m, key(tui.KeyUp)) // stays on the first
	m, _ = press(m, key(tui.KeyPgDown))
	if m.cursor != 4 {
		t.Errorf("PgDn cursor = %d, want 4", m.cursor)
	}
	m, _ = press(m, key(tui.KeyPgUp))
	if m.cursor != 0 {
		t.Errorf("PgUp cursor = %d, want 0", m.cursor)
	}
}

func TestTypeAheadJumpsToTheNextMatch(t *testing.T) {
	clock := time.Unix(100, 0)
	now = func() time.Time { return clock }
	defer func() { now = time.Now }()
	m := New("", "Apple", "Avocado", "Banana", "Blueberry", "Cherry")
	m.Focus()
	m, _ = press(m, key(tui.KeyEnter))
	m, _ = press(m, runes("b"))
	if m.cursor != 2 {
		t.Fatalf("b -> %d, want 2", m.cursor)
	}
	m, _ = press(m, runes("b")) // "bb" matches nothing; cursor stays
	if m.cursor != 2 {
		t.Fatalf("bb -> %d, want 2", m.cursor)
	}
	clock = clock.Add(2 * time.Second) // a pause ends the word
	m, _ = press(m, runes("b"))
	if m.cursor != 3 {
		t.Fatalf("b again -> %d, want 3 (the next b)", m.cursor)
	}
	clock = clock.Add(2 * time.Second)
	m, _ = press(m, runes("B"), runes("l")) // ignores case, prefix grows in place
	if m.cursor != 3 {
		t.Fatalf("Bl -> %d, want 3", m.cursor)
	}
	clock = clock.Add(2 * time.Second)
	m, _ = press(m, runes("a")) // wraps round to the top
	if m.cursor != 0 {
		t.Fatalf("a -> %d, want 0 (wrapped)", m.cursor)
	}
}

func TestDisabledAndEmpty(t *testing.T) {
	m := plan()
	m.Disabled = true
	if m2, cmd := press(m, key(tui.KeyEnter), key(tui.KeyRight)); m2.Open() || cmd != nil || m2.Selected() != -1 {
		t.Error("a disabled dropdown reacted")
	}
	m.SetOpen(true)
	if m.Open() {
		t.Error("SetOpen opened a disabled dropdown")
	}
	if m.Bindings() != nil {
		t.Error("a disabled dropdown has bindings")
	}
	e := New("Empty")
	e.Focus()
	e, cmd := press(e, key(tui.KeyEnter), key(tui.KeyRight))
	if e.Open() || cmd != nil || e.Value() != "" {
		t.Error("an empty dropdown reacted")
	}
}

func TestFocusAndBlur(t *testing.T) {
	m := New("Plan", "Free")
	if m2, _ := press(m, key(tui.KeyEnter)); m2.Open() {
		t.Error("an unfocused dropdown opened")
	}
	m.Focus()
	m, _ = press(m, key(tui.KeyEnter))
	m.Blur()
	if m.Open() || m.Focused() {
		t.Error("Blur left it open or focused")
	}
	if plain(New("a", "b")) == "" {
		t.Error("empty view")
	}
}

func TestMouse(t *testing.T) {
	m := plan()
	m.Mouse = true
	m.Bounds = hittest.Rect{X: 2, Y: 3, W: 12, H: 1}
	click := func(m Model, x, y int) (Model, tui.Cmd) {
		return m.Update(tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	}
	m, _ = click(m, 5, 3)
	if !m.Open() {
		t.Fatal("a click did not open")
	}
	m, cmd := click(m, 5, 5) // the second option is two rows below the control
	if msg, ok := changed(cmd); !ok || msg.Index != 1 || m.Open() {
		t.Fatalf("click on an option: %+v %v open %v", msg, ok, m.Open())
	}
	m, _ = click(m, 5, 3)
	m, cmd = click(m, 40, 20) // outside closes
	if m.Open() || cmd != nil || m.Selected() != 1 {
		t.Error("an outside click changed something")
	}
	m, _ = click(m, 5, 3)
	m, _ = click(m, 5, 3) // the control again closes
	if m.Open() {
		t.Error("clicking the control did not close it")
	}
	m.Mouse = false
	if m2, _ := click(m, 5, 3); m2.Open() {
		t.Error("the mouse worked with Mouse off")
	}
	m.Mouse = true
	if m2, _ := m.Update(tui.MouseEvent{X: 5, Y: 3, Button: tui.MouseButtonRight, Action: tui.MouseActionPress}); m2.Open() {
		t.Error("a right click opened")
	}
	m.Disabled = true
	if m2, _ := click(m, 5, 3); m2.Open() {
		t.Error("a disabled dropdown took a click")
	}
}

func TestPopupIsTheListWithoutTheControlRow(t *testing.T) {
	m := plan()
	if m.Popup() != "" {
		t.Error("a closed dropdown has a popup")
	}
	m, _ = press(m, key(tui.KeyEnter))
	pop := ansi.StripANSI(m.Popup())
	_, list, _ := strings.Cut(plain(m), "\n")
	if strings.TrimLeft(list, " ") == "" || !strings.Contains(pop, "Free") || strings.Contains(pop, "Plan") {
		t.Errorf("Popup = %q", pop)
	}
}

func TestLayoutNodeLinearizeAndTheme(t *testing.T) {
	m := plan()
	m.SetSelected(1)
	if w := m.LayoutNode().Measure(layout.Constraints{MaxW: 80, MaxH: 24}); w.W != m.Width() || w.H != 1 {
		t.Errorf("Measure = %+v", w)
	}
	for open, want := range map[bool]string{false: "Plan, dropdown, Pro, 2 of 3", true: "Plan, dropdown, Pro, 2 of 3, expanded"} {
		m.SetOpen(open)
		if got := m.Linearize(); got != want {
			t.Errorf("Linearize = %q, want %q", got, want)
		}
	}
	if got := New("", "a").Linearize(); got != "dropdown, Select" {
		t.Errorf("Linearize = %q", got)
	}
	m.Disabled = true
	if !strings.HasSuffix(m.Linearize(), ", unavailable") {
		t.Errorf("Linearize = %q", m.Linearize())
	}
	if m.SetTheme(theme.LightTheme()).Theme != theme.LightTheme() {
		t.Error("SetTheme did not apply")
	}
	if len(plan().Bindings()) != 3 {
		t.Error("closed bindings")
	}
	o, _ := press(plan(), key(tui.KeyEnter))
	if len(o.Bindings()) != 4 {
		t.Error("open bindings")
	}
	if plan().keys().Open.Keys == nil || (Model{}).keys().Open.Keys == nil {
		t.Error("a zero KeyMap did not fall back to the default")
	}
}

func TestOptionTextCannotCarryControlCodes(t *testing.T) {
	m := New("La\x1b[2Jbel", "a\x1b]0;t\x07b")
	m.Focus()
	m.SetSelected(0)
	m, _ = press(m, key(tui.KeyEnter))
	if s := ansi.StripANSI(m.View()); strings.ContainsAny(s, "\x1b\x07") || !strings.Contains(s, "ab") {
		t.Errorf("View = %q", s)
	}
	m.SetSelected(5)
	if m.Selected() != -1 {
		t.Error("SetSelected out of range chose something")
	}
}

func TestTypeAheadKeepsTheHighlightWhileItStillMatches(t *testing.T) {
	m := New("", "Blue", "Bluebird", "Green")
	m.Focus()
	m, _ = press(m, key(tui.KeyEnter))
	m, _ = press(m, runes("b")) // from Blue, the next match is Bluebird
	if m.cursor != 1 {
		t.Fatalf("b -> %d, want 1", m.cursor)
	}
	m, _ = press(m, runes("l")) // "bl" still matches Bluebird: stay, do not wrap to Blue
	if m.cursor != 1 {
		t.Fatalf("bl -> %d, want 1", m.cursor)
	}
}

func TestOptionsChangedWhileOpenCannotBreakTheView(t *testing.T) {
	opts := make([]string, 20)
	for i := range opts {
		opts[i] = "o" + string(rune('a'+i))
	}
	m := New("", opts...)
	m.MaxRows = 5
	m.Focus()
	m, _ = press(m, key(tui.KeyEnter), key(tui.KeyEnd))
	m.Options = m.Options[:7] // the app replaced the list under the open dropdown
	if got := plain(m); strings.Count(got, "\n") < 5 {
		t.Errorf("View after the list shrank:\n%s", got)
	}
	m.Mouse = true
	m.Bounds = hittest.Rect{X: 0, Y: 0, W: 10, H: 1}
	m, cmd := m.Update(tui.MouseEvent{X: 1, Y: 1, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if msg, ok := changed(cmd); !ok || msg.Index != 2 {
		t.Errorf("click on the first visible row chose %+v, %v; want index 2", msg, ok)
	}
}
