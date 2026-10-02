package appshell

import (
	"github.com/ows4444/tui"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

// Child is what a Router holds: anything that can take a message and render.
// Build one from a widget with Wrap. A Child that also has
// SetTheme(theme.Theme) Child receives the Router's theme, and one with
// Bindings() []keymap.Binding lends its bindings to the Router's own.
type Child interface {
	Update(tui.Msg) (Child, tui.Cmd)
	View() string
}

// Wrap adapts a widget type T (any value with Update(tui.Msg) (T, tui.Cmd)
// and View() string) to a Child. If T has SetTheme(theme.Theme) T (see
// tui.ThemeSetter) or is a tui.Themeable whose SetTheme returns a T, the
// Router's theme reaches it; if T has Bindings() []keymap.Binding the Router
// reports them while the child is focused.
func Wrap[T interface {
	Update(tui.Msg) (T, tui.Cmd)
	View() string
}](v T) Child {
	return wrapped[T]{v: v}
}

type wrapped[T interface {
	Update(tui.Msg) (T, tui.Cmd)
	View() string
}] struct{ v T }

func (w wrapped[T]) Update(msg tui.Msg) (Child, tui.Cmd) {
	next, cmd := w.v.Update(msg)
	return wrapped[T]{v: next}, cmd
}

func (w wrapped[T]) View() string { return w.v.View() }

func (w wrapped[T]) Bindings() []keymap.Binding {
	if b, ok := any(w.v).(interface{ Bindings() []keymap.Binding }); ok {
		return b.Bindings()
	}
	return nil
}

// applyTheme reports false when T takes no theme.
func (w wrapped[T]) applyTheme(t theme.Theme) (Child, bool) {
	if s, ok := any(w.v).(tui.ThemeSetter[T]); ok {
		return wrapped[T]{v: s.SetTheme(t)}, true
	}
	if s, ok := any(w.v).(tui.Themeable); ok {
		if next, ok := s.SetTheme(t).(T); ok {
			return wrapped[T]{v: next}, true
		}
	}
	return w, false
}

// Router is an opt-in composition router for a screen of several children
// registered by id. It is a value: every method returns the next Router.
//
// Routing rules, see Update: keys go to the focused child only; mouse events
// go to the child under the pointer (see SetBounds, SetLayout); every other
// message, ResizeMsg included, is broadcast to all children. The theme is
// delivered with SetTheme, to each themeable child once.
type Router struct {
	ids    []string
	kids   []Child
	cur    int
	cycle  bool
	th     theme.Theme
	themed bool
	bounds map[string]layout.Rect
}

// NewRouter returns an empty Router. The first child added has focus.
func NewRouter() Router { return Router{} }

type themeChild interface {
	SetTheme(theme.Theme) Child
}

type themeWrapped interface {
	applyTheme(theme.Theme) (Child, bool)
}

func giveTheme(c Child, t theme.Theme) Child {
	switch v := c.(type) {
	case themeWrapped:
		n, _ := v.applyTheme(t)
		return n
	case themeChild:
		return v.SetTheme(t)
	}
	return c
}

func (r Router) index(id string) int {
	for i, s := range r.ids {
		if s == id {
			return i
		}
	}
	return -1
}

// Add registers c under id, after the children already added. A repeated id
// replaces that child in place. If SetTheme was called before, c receives the
// theme now.
func (r Router) Add(id string, c Child) Router {
	if r.themed {
		c = giveTheme(c, r.th)
	}
	if i := r.index(id); i >= 0 {
		r.kids = append([]Child(nil), r.kids...)
		r.kids[i] = c
		return r
	}
	r.ids = append(append([]string(nil), r.ids...), id)
	r.kids = append(append([]Child(nil), r.kids...), c)
	return r
}

// WithTabCycle opts in to Tab and Shift+Tab moving focus to the next and
// previous child (wrapping). Off by default, when Tab goes to the focused
// child like any other key.
func (r Router) WithTabCycle(on bool) Router { r.cycle = on; return r }

// Focus gives focus to the child registered as id. An unknown id changes
// nothing.
func (r Router) Focus(id string) Router {
	if i := r.index(id); i >= 0 {
		r.cur = i
	}
	return r
}

// Focused returns the id of the focused child, or "" for an empty Router.
func (r Router) Focused() string {
	if r.cur < 0 || r.cur >= len(r.ids) {
		return ""
	}
	return r.ids[r.cur]
}

// FocusNext moves focus to the next child, wrapping at the end.
func (r Router) FocusNext() Router {
	if n := len(r.ids); n > 0 {
		r.cur = (r.cur + 1) % n
	}
	return r
}

// FocusPrev moves focus to the previous child, wrapping at the start.
func (r Router) FocusPrev() Router {
	if n := len(r.ids); n > 0 {
		r.cur = (r.cur + n - 1) % n
	}
	return r
}

// View returns the View of the child registered as id, or "" if unknown. The
// Router does not lay children out itself; compose the views with
// package layout.
func (r Router) View(id string) string {
	if i := r.index(id); i >= 0 {
		return r.kids[i].View()
	}
	return ""
}

// Bindings returns the focused child's bindings when it has them, so a help
// screen follows focus. It makes Router a tui.BindingsProvider.
func (r Router) Bindings() []keymap.Binding {
	if r.cur < 0 || r.cur >= len(r.kids) {
		return nil
	}
	if b, ok := r.kids[r.cur].(interface{ Bindings() []keymap.Binding }); ok {
		return b.Bindings()
	}
	return nil
}

// SetTheme delivers t to every child that takes a theme, once each, and to
// children added later when they are added. Other children are untouched. It
// makes Router a tui.ThemeSetter[Router].
func (r Router) SetTheme(t theme.Theme) Router {
	r.th, r.themed = t, true
	kids := make([]Child, len(r.kids))
	for i, c := range r.kids {
		kids[i] = giveTheme(c, t)
	}
	r.kids = kids
	return r
}

// Update routes msg. A Tab or Shift+Tab moves focus when WithTabCycle is on.
// Any other tui.Key goes to the focused child only. A tui.MouseEvent is
// routed by position (see SetBounds). Everything else, ResizeMsg and ticks
// included, is broadcast to every child in order; the Cmds are batched.
func (r Router) Update(msg tui.Msg) (Router, tui.Cmd) {
	switch m := msg.(type) {
	case tui.Key:
		if r.cycle && m.Type == tui.KeyTab {
			if m.Mod.Shift() {
				return r.FocusPrev(), nil
			}
			return r.FocusNext(), nil
		}
		return r.updateAt(r.cur, msg)
	case tui.MouseEvent:
		return r.routeMouse(m)
	}
	kids := make([]Child, len(r.kids))
	cmds := make([]tui.Cmd, 0, len(r.kids))
	for i, c := range r.kids {
		var cmd tui.Cmd
		kids[i], cmd = c.Update(msg)
		cmds = append(cmds, cmd)
	}
	r.kids = kids
	return r, tui.Batch(cmds...)
}

// updateAt sends msg to child i only.
func (r Router) updateAt(i int, msg tui.Msg) (Router, tui.Cmd) {
	if i < 0 || i >= len(r.kids) {
		return r, nil
	}
	next, cmd := r.kids[i].Update(msg)
	r.kids = append([]Child(nil), r.kids...)
	r.kids[i] = next
	return r, cmd
}

var (
	_ tui.BindingsProvider    = Router{}
	_ tui.ThemeSetter[Router] = Router{}
)
