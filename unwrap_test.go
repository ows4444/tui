package tui

import (
	"testing"

	"github.com/ows4444/tui/theme"
)

type uBase struct{ name string }

func (uBase) Init() Cmd                 { return nil }
func (m uBase) Update(Msg) (Model, Cmd) { return m, nil }
func (m uBase) View() string            { return m.name }

type uCursor struct{ uBase }

func (uCursor) CursorPos() (int, int, bool) { return 1, 2, true }

type uThemed struct {
	uBase
	themed bool
}

func (m uThemed) SetTheme(theme.Theme) Model { m.themed = true; return m }

// uWrap wraps a model; it implements Rewrap only when canRewrap.
type uWrap struct {
	uBase
	inner Model
}

func (w uWrap) Unwrap() Model { return w.inner }

type uRewrap struct{ uWrap }

func (w uRewrap) Rewrap(inner Model) Model { w.inner = inner; return w }

func TestFindReachesTheWrappedModel(t *testing.T) {
	inner := uCursor{uBase{"in"}}
	got, ok := find[CursorPlacer](uWrap{uBase{"w"}, inner})
	if !ok {
		t.Fatal("CursorPlacer on the wrapped model was not found")
	}
	if x, y, _ := got.CursorPos(); x != 1 || y != 2 {
		t.Fatalf("CursorPos = %d,%d", x, y)
	}
}

func TestFindStopsAtTheFirstMatch(t *testing.T) {
	// The root implements CursorPlacer itself: it is the match even though
	// it also wraps one.
	m := wrapAndPlace{inner: uCursor{}}
	got, ok := find[CursorPlacer](m)
	if !ok || got.(wrapAndPlace).inner == nil {
		t.Fatalf("find returned %#v, %v; want the root", got, ok)
	}
}

type wrapAndPlace struct {
	uBase
	inner Model
}

func (w wrapAndPlace) Unwrap() Model             { return w.inner }
func (wrapAndPlace) CursorPos() (int, int, bool) { return 9, 9, true }

func TestFindThroughSeveralWrappers(t *testing.T) {
	m := uWrap{inner: uWrap{inner: uWrap{inner: uCursor{}}}}
	if _, ok := find[CursorPlacer](m); !ok {
		t.Fatal("not found through three wrappers")
	}
}

func TestFindMissing(t *testing.T) {
	if _, ok := find[CursorPlacer](uWrap{inner: uBase{}}); ok {
		t.Fatal("found a CursorPlacer that is not there")
	}
	if _, ok := find[CursorPlacer](uWrap{}); ok { // Unwrap returns nil
		t.Fatal("found a CursorPlacer under a nil inner model")
	}
}

type selfWrap struct{ uBase }

func (s selfWrap) Unwrap() Model { return s }

func TestFindSurvivesAnUnwrapCycle(t *testing.T) {
	if _, ok := find[CursorPlacer](selfWrap{}); ok {
		t.Fatal("a self-wrapping model has no CursorPlacer")
	}
}

func TestSetThemeRebuildsTheChain(t *testing.T) {
	m := uRewrap{uWrap{uBase{"w"}, uWrap{inner: uThemed{}}}}
	// The inner wrapper has no Rewrap, so the chain cannot be rebuilt.
	if _, ok := setTheme(m, theme.DarkTheme(), 0); ok {
		t.Fatal("a wrapper without Rewrap must stop the theme")
	}
	m = uRewrap{uWrap{uBase{"w"}, uRewrap{uWrap{inner: uThemed{}}}}}
	next, ok := setTheme(m, theme.DarkTheme(), 0)
	if !ok {
		t.Fatal("theme was not applied through two Rewrappers")
	}
	inner := next.(uRewrap).Unwrap().(uRewrap).Unwrap()
	if !inner.(uThemed).themed {
		t.Fatal("innermost model was not themed")
	}
	if m.Unwrap().(uRewrap).Unwrap().(uThemed).themed {
		t.Fatal("setTheme mutated the original chain")
	}
}

func TestSetThemeOnTheRootItself(t *testing.T) {
	next, ok := setTheme(uThemed{}, theme.DarkTheme(), 0)
	if !ok || !next.(uThemed).themed {
		t.Fatalf("root Themeable: ok=%v next=%#v", ok, next)
	}
}

func TestSetThemeWithNothingToTheme(t *testing.T) {
	m := uRewrap{uWrap{inner: uBase{}}}
	if next, ok := setTheme(m, theme.DarkTheme(), 0); ok || next.(uRewrap).Unwrap() == nil {
		t.Fatalf("ok=%v next=%#v", ok, next)
	}
	if _, ok := setTheme(selfWrapRewrap{}, theme.DarkTheme(), 0); ok {
		t.Fatal("cycle must end in not-found")
	}
}

type selfWrapRewrap struct{ selfWrap }

func (s selfWrapRewrap) Rewrap(Model) Model { return s }
