package focus_test

import (
	"reflect"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/focus"
	"github.com/ows4444/tui/input"
	"github.com/ows4444/tui/passwordinput"
	"github.com/ows4444/tui/textarea"
	"github.com/ows4444/tui/textinput"
)

var (
	tab      = tui.Key{Type: tui.KeyTab}
	shiftTab = tui.Key{Type: tui.KeyTab, Mod: input.ModShift}
	letter   = tui.Key{Type: tui.KeyRunes, Text: "x"}
)

// widget is a fake that logs what it was asked to do, in a shared log.
type widget struct {
	name string
	log  *[]string
}

func (w *widget) Focus() tui.Cmd {
	*w.log = append(*w.log, "focus "+w.name)
	name := w.name
	return func() tui.Msg { return focusedMsg(name) }
}
func (w *widget) Blur() { *w.log = append(*w.log, "blur "+w.name) }
func (w *widget) update(msg tui.Msg) tui.Cmd {
	*w.log = append(*w.log, "update "+w.name)
	return func() tui.Msg { return updatedMsg(w.name) }
}

type focusedMsg string
type updatedMsg string

func (w *widget) field() focus.Field {
	return focus.Field{Focus: w.Focus, Blur: w.Blur, Update: w.update}
}

func setup(n int) ([]*widget, []focus.Field, *[]string) {
	var log []string
	ws := make([]*widget, n)
	fs := make([]focus.Field, n)
	for i := range ws {
		ws[i] = &widget{name: string(rune('a' + i)), log: &log}
		fs[i] = ws[i].field()
	}
	return ws, fs, &log
}

func msgOf(t *testing.T, cmd tui.Cmd) tui.Msg {
	t.Helper()
	if cmd == nil {
		t.Fatal("Cmd is nil")
	}
	return cmd()
}

func TestRouteTabBlursTheOldAndFocusesTheNew(t *testing.T) {
	_, fs, log := setup(3)
	r := focus.New(3)

	r, cmd := r.Route(tab, fs...)
	if r.Current() != 1 {
		t.Fatalf("after Tab Current = %d, want 1", r.Current())
	}
	if want := []string{"blur a", "focus b"}; !reflect.DeepEqual(*log, want) {
		t.Errorf("calls = %v, want %v", *log, want)
	}
	if got := msgOf(t, cmd); got != focusedMsg("b") {
		t.Errorf("Cmd delivered %v, want the new item's Focus Cmd", got)
	}

	*log = nil
	r, cmd = r.Route(shiftTab, fs...)
	if r.Current() != 0 {
		t.Fatalf("after Shift+Tab Current = %d, want 0", r.Current())
	}
	if want := []string{"blur b", "focus a"}; !reflect.DeepEqual(*log, want) {
		t.Errorf("calls = %v, want %v", *log, want)
	}
	if got := msgOf(t, cmd); got != focusedMsg("a") {
		t.Errorf("Cmd delivered %v", got)
	}

	// Wraps at both ends.
	*log = nil
	r, _ = r.Route(shiftTab, fs...)
	if r.Current() != 2 || !reflect.DeepEqual(*log, []string{"blur a", "focus c"}) {
		t.Errorf("wrap backwards: Current %d, calls %v", r.Current(), *log)
	}
	r, _ = r.Route(tab, fs...)
	if r.Current() != 0 {
		t.Errorf("wrap forwards: Current %d", r.Current())
	}
}

func TestRouteSendsOtherMessagesToTheFocusedItemOnly(t *testing.T) {
	_, fs, log := setup(3)
	r := focus.New(3).Set(2)

	r, cmd := r.Route(letter, fs...)
	if r.Current() != 2 {
		t.Errorf("a letter moved focus to %d", r.Current())
	}
	if want := []string{"update c"}; !reflect.DeepEqual(*log, want) {
		t.Errorf("calls = %v, want only the focused item updated", *log)
	}
	if got := msgOf(t, cmd); got != updatedMsg("c") {
		t.Errorf("Cmd delivered %v, want the focused item's Update Cmd", got)
	}

	// Non-key messages too.
	*log = nil
	r.Route(tui.ResizeMsg{Width: 10, Height: 5}, fs...)
	if want := []string{"update c"}; !reflect.DeepEqual(*log, want) {
		t.Errorf("resize calls = %v", *log)
	}
}

func TestRouteSkipsDisabledItems(t *testing.T) {
	_, fs, log := setup(4)
	r := focus.New(4).SetDisabled(1, true).SetDisabled(2, true)

	r, _ = r.Route(tab, fs...)
	if r.Current() != 3 {
		t.Fatalf("Tab skipped to %d, want 3", r.Current())
	}
	if want := []string{"blur a", "focus d"}; !reflect.DeepEqual(*log, want) {
		t.Errorf("calls = %v, want %v (never b or c)", *log, want)
	}
	*log = nil
	r, _ = r.Route(shiftTab, fs...)
	if r.Current() != 0 || !reflect.DeepEqual(*log, []string{"blur d", "focus a"}) {
		t.Errorf("back: Current %d calls %v", r.Current(), *log)
	}
}

func TestRouteWithOneEnabledItemChangesNothing(t *testing.T) {
	_, fs, log := setup(3)
	r := focus.New(3).SetDisabled(1, true).SetDisabled(2, true)
	r2, cmd := r.Route(tab, fs...)
	if r2.Current() != 0 || cmd != nil || len(*log) != 0 {
		t.Errorf("Current %d, cmd %v, calls %v; want no change", r2.Current(), cmd != nil, *log)
	}
}

func TestRouteEdgeCases(t *testing.T) {
	// An empty ring and empty fields never panic.
	r, cmd := focus.New(0).Route(tab)
	if r.Len() != 0 || cmd != nil {
		t.Error("empty ring changed")
	}
	_, cmd = focus.New(0).Route(letter)
	if cmd != nil {
		t.Error("a message to an empty ring produced a Cmd")
	}
	// Fewer fields than ring items: the missing ones are ignored.
	_, fs, log := setup(1)
	r = focus.New(3)
	r, _ = r.Route(tab, fs...) // blur a, focus index 1 (no field)
	if r.Current() != 1 || !reflect.DeepEqual(*log, []string{"blur a"}) {
		t.Errorf("Current %d calls %v", r.Current(), *log)
	}
	if _, cmd := r.Route(letter, fs...); cmd != nil {
		t.Error("a message to a missing field produced a Cmd")
	}
	// Nil funcs are skipped.
	nilFields := []focus.Field{{}, {}}
	if r2, cmd := focus.New(2).Route(tab, nilFields...); r2.Current() != 1 || cmd != nil {
		t.Error("nil funcs misbehaved")
	}
	if _, cmd := focus.New(2).Route(letter, nilFields...); cmd != nil {
		t.Error("nil Update produced a Cmd")
	}
}

func TestSyncFocusesTheCurrentAndBlursTheRest(t *testing.T) {
	_, fs, log := setup(3)
	cmd := focus.New(3).Set(1).Sync(fs...)
	if want := []string{"blur a", "blur c", "focus b"}; !reflect.DeepEqual(*log, want) {
		t.Errorf("calls = %v, want %v", *log, want)
	}
	if got := msgOf(t, cmd); got != focusedMsg("b") {
		t.Errorf("Cmd delivered %v", got)
	}
	if cmd := focus.New(0).Sync(); cmd != nil {
		t.Error("Sync on an empty ring returned a Cmd")
	}
}

// Route works with real widgets through Bind: Tab moves the cursor between
// inputs, and typed characters reach only the focused one.
func TestBindDrivesRealWidgets(t *testing.T) {
	name := textinput.New()
	pass := passwordinput.New()
	bio := textarea.New()
	ring := focus.New(3)

	fields := func() []focus.Field {
		return []focus.Field{focus.Bind(&name), focus.Bind(&pass), focus.Bind(&bio)}
	}
	ring.Sync(fields()...)
	if !name.Focused() || pass.Focused() || bio.Focused() {
		t.Fatalf("after Sync focused = %v %v %v, want only the first", name.Focused(), pass.Focused(), bio.Focused())
	}

	ring, _ = ring.Route(letter, fields()...)
	if name.Value() != "x" || pass.Value() != "" || bio.Value() != "" {
		t.Errorf("values after typing = %q %q %q, want only the first to change", name.Value(), pass.Value(), bio.Value())
	}

	ring, _ = ring.Route(tab, fields()...)
	if name.Focused() || !pass.Focused() {
		t.Errorf("after Tab focused = %v %v", name.Focused(), pass.Focused())
	}
	ring, _ = ring.Route(letter, fields()...)
	if pass.Value() != "x" || name.Value() != "x" {
		t.Errorf("values = %q %q", name.Value(), pass.Value())
	}

	ring, _ = ring.Route(tab, fields()...)
	ring, _ = ring.Route(tab, fields()...) // wraps to the first
	if !name.Focused() || ring.Current() != 0 {
		t.Errorf("wrap: name focused %v, Current %d", name.Focused(), ring.Current())
	}
}

// Existing Ring behaviour is untouched by Route: it only calls the same
// Update, Next and Prev.
func TestRouteAgreesWithRingUpdate(t *testing.T) {
	for _, msg := range []tui.Msg{tab, shiftTab, letter, tui.ResizeMsg{}} {
		a, _ := focus.New(4).SetDisabled(2, true).Set(1).Update(msg)
		b, _ := focus.New(4).SetDisabled(2, true).Set(1).Route(msg)
		if a.Current() != b.Current() {
			t.Errorf("%T: Update -> %d, Route -> %d", msg, a.Current(), b.Current())
		}
	}
}
