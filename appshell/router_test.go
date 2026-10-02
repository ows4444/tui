package appshell

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/input"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/theme"
)

// rec records what reaches it in a shared log.
type rec struct {
	name string
	log  *[]string
}

func (r rec) Update(msg tui.Msg) (rec, tui.Cmd) {
	*r.log = append(*r.log, fmt.Sprintf("%s:%v", r.name, msg))
	return r, nil
}
func (r rec) View() string { return r.name }

// themed is a rec that also takes a theme.
type themed struct {
	rec
	n *int
}

func (t themed) Update(msg tui.Msg) (themed, tui.Cmd) {
	r, c := t.rec.Update(msg)
	t.rec = r
	return t, c
}

func (t themed) SetTheme(theme.Theme) themed { *t.n++; return t }

// keyed is a rec with bindings.
type keyed struct{ rec }

func (k keyed) Update(msg tui.Msg) (keyed, tui.Cmd) {
	r, c := k.rec.Update(msg)
	k.rec = r
	return k, c
}

func (keyed) Bindings() []keymap.Binding { return []keymap.Binding{keymap.NewBinding("go", "g")} }

func three(log *[]string) Router {
	r := NewRouter()
	for _, n := range []string{"a", "b", "c"} {
		r = r.Add(n, Wrap(rec{n, log}))
	}
	return r
}

func TestRouterKeyReachesOnlyFocused(t *testing.T) {
	var log []string
	r := three(&log).Focus("b")
	r, _ = r.Update(tui.Key{Type: tui.KeyRunes, Text: "x"})
	if len(log) != 1 || !strings.HasPrefix(log[0], "b:") {
		t.Fatalf("log = %v", log)
	}
	r = r.Focus("c")
	log = nil
	_, _ = r.Update(tui.Key{Type: tui.KeyRunes, Text: "y"})
	if len(log) != 1 || !strings.HasPrefix(log[0], "c:") {
		t.Fatalf("after refocus log = %v", log)
	}
	if r.Focused() != "c" {
		t.Fatal(r.Focused())
	}
}

func TestRouterTheme(t *testing.T) {
	var log []string
	var n1, n2 int
	r := NewRouter().
		Add("a", Wrap(themed{rec{"a", &log}, &n1})).
		Add("plain", Wrap(rec{"p", &log})).
		Add("b", Wrap(themed{rec{"b", &log}, &n2}))
	r = r.SetTheme(theme.DarkTheme())
	if n1 != 1 || n2 != 1 {
		t.Fatalf("n1=%d n2=%d", n1, n2)
	}
	if len(log) != 0 {
		t.Fatalf("theme leaked as message: %v", log)
	}
}

func TestRouterFocusCycleOptIn(t *testing.T) {
	var log []string
	tab := tui.Key{Type: tui.KeyTab}
	r := three(&log)
	if r.Focused() != "a" {
		t.Fatal("first added child should be focused by default")
	}
	r, _ = r.Update(tab)
	if r.Focused() != "a" || len(log) != 1 {
		t.Fatalf("tab without opt-in must go to child: %v %s", log, r.Focused())
	}
	r = r.WithTabCycle(true)
	log = nil
	r, _ = r.Update(tab)
	if r.Focused() != "b" || len(log) != 0 {
		t.Fatalf("got %s %v", r.Focused(), log)
	}
	back := tui.Key{Type: tui.KeyTab, Mod: input.ModShift}
	r, _ = r.Update(back)
	r, _ = r.Update(back)
	if r.Focused() != "c" {
		t.Fatalf("wrap back: %s", r.Focused())
	}
	if r.FocusNext().Focused() != "a" {
		t.Fatal("next wrap")
	}
}

func TestRouterBroadcastAndResize(t *testing.T) {
	var log []string
	r := three(&log)
	_, _ = r.Update(tui.ResizeMsg{Width: 5, Height: 6})
	if len(log) != 3 {
		t.Fatalf("resize log = %v", log)
	}
	log = nil
	_, _ = r.Update(struct{ tick int }{1})
	if len(log) != 3 {
		t.Fatalf("broadcast log = %v", log)
	}
}

func TestRouterBindingsFollowFocus(t *testing.T) {
	var log []string
	r := NewRouter().Add("a", Wrap(rec{"a", &log})).Add("k", Wrap(keyed{rec{"k", &log}}))
	if len(r.Bindings()) != 0 {
		t.Fatal("rec has no bindings")
	}
	if b := r.Focus("k").Bindings(); len(b) != 1 || b[0].Desc != "go" {
		t.Fatalf("%v", b)
	}
	var _ tui.BindingsProvider = r
}

func TestRouterViewAndUnknownFocus(t *testing.T) {
	var log []string
	r := three(&log)
	if r.Focus("zzz").Focused() != "a" {
		t.Fatal("unknown id must not change focus")
	}
	if r.View("b") != "b" || r.View("zzz") != "" {
		t.Fatal("view")
	}
}
