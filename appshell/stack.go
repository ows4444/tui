package appshell

import (
	"github.com/ows4444/tui"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/theme"
)

// Stack is a stack of screens: the top one is drawn and takes the keys, the
// ones under it wait. A screen pushes another over itself (a confirm, a
// picker, a form) and the pushed screen pops itself with a result that is
// delivered to the screen below, which is how a modal flow hands back its
// answer without every app wiring it by hand.
//
// A screen is a Child. It asks for the change by returning a Cmd from its
// Update (PushCmd, PopCmd); the Stack, which sees every Msg, makes it. A Stack
// is a value: every method returns the next one. The root screen cannot be
// popped.
//
// Routing, see Update: keys and mouse events go to the top screen only; a
// ResizeMsg and every other message go to every screen, so a screen under
// another still gets its ticks and task results and has the right size when it
// comes back to the top.
//
// Stability: experimental, like the rest of this package.
type Stack struct {
	screens []Child
	resize  *tui.ResizeMsg
	th      theme.Theme
	themed  bool
}

// NewStack returns a Stack whose only screen is root.
func NewStack(root Child) Stack { return Stack{screens: []Child{root}} }

// PushMsg asks the Stack to put Screen on top. A screen returns it from a Cmd,
// see PushCmd.
type PushMsg struct{ Screen Child }

// PopMsg asks the Stack to remove the top screen and deliver Result, if not
// nil, to the screens that remain. A screen returns it from a Cmd, see PopCmd.
type PopMsg struct{ Result tui.Msg }

// PushCmd is the Cmd that pushes c on the Stack it reaches.
func PushCmd(c Child) tui.Cmd { return func() tui.Msg { return PushMsg{Screen: c} } }

// PopCmd is the Cmd that pops the top screen. result, if not nil, is delivered
// to the screen below it as an ordinary Msg through Update: define a type for
// it, such as type ConfirmResult struct{ Yes bool }, and match it there.
func PopCmd(result tui.Msg) tui.Cmd { return func() tui.Msg { return PopMsg{Result: result} } }

// Depth is the number of screens, at least one.
func (s Stack) Depth() int { return len(s.screens) }

// Top is the screen that takes the keys.
func (s Stack) Top() Child { return s.screens[len(s.screens)-1] }

// Push puts c on top. If the Stack has seen a ResizeMsg c is given it, and
// the Cmd is whatever c returns for it. A nil c changes nothing.
func (s Stack) Push(c Child) (Stack, tui.Cmd) {
	if c == nil {
		return s, nil
	}
	if s.themed {
		c = giveTheme(c, s.th)
	}
	var cmd tui.Cmd
	if s.resize != nil {
		c, cmd = c.Update(*s.resize)
	}
	s.screens = append(append([]Child(nil), s.screens...), c)
	return s, cmd
}

// Pop removes the top screen, unless it is the only one, and returns the Cmd
// that delivers result to the Stack's Update, and so to the screens below, when
// result is not nil.
func (s Stack) Pop(result tui.Msg) (Stack, tui.Cmd) {
	if len(s.screens) < 2 {
		return s, nil
	}
	s.screens = append([]Child(nil), s.screens[:len(s.screens)-1]...)
	if result == nil {
		return s, nil
	}
	return s, func() tui.Msg { return result }
}

// Replace swaps the top screen for c: a wizard's next step, say. The old screen
// gets no result. Like Push, it hands c the last ResizeMsg.
func (s Stack) Replace(c Child) (Stack, tui.Cmd) {
	if c == nil {
		return s, nil
	}
	if s.themed {
		c = giveTheme(c, s.th)
	}
	var cmd tui.Cmd
	if s.resize != nil {
		c, cmd = c.Update(*s.resize)
	}
	s.screens = append([]Child(nil), s.screens...)
	s.screens[len(s.screens)-1] = c
	return s, cmd
}

// Update makes the changes PushMsg and PopMsg ask for, sends keys and mouse
// events to the top screen, and everything else to every screen from the
// bottom up, batching the Cmds.
func (s Stack) Update(msg tui.Msg) (Stack, tui.Cmd) {
	switch m := msg.(type) {
	case PushMsg:
		return s.Push(m.Screen)
	case PopMsg:
		return s.Pop(m.Result)
	case tui.Key, tui.MouseEvent:
		top := len(s.screens) - 1
		s.screens = append([]Child(nil), s.screens...)
		var cmd tui.Cmd
		s.screens[top], cmd = s.screens[top].Update(msg)
		return s, cmd
	case tui.ResizeMsg:
		r := m
		s.resize = &r
	}
	s.screens = append([]Child(nil), s.screens...)
	cmds := make([]tui.Cmd, len(s.screens))
	for i, c := range s.screens {
		s.screens[i], cmds[i] = c.Update(msg)
	}
	return s, tui.Batch(cmds...)
}

// View draws the top screen.
func (s Stack) View() string { return s.Top().View() }

// ViewAt draws the screen at depth, 0 being the root, or "" when there is none.
// An app that wants a modal drawn over the screen under it composes
// ViewAt(Depth()-2) and View itself.
func (s Stack) ViewAt(depth int) string {
	if depth < 0 || depth >= len(s.screens) {
		return ""
	}
	return s.screens[depth].View()
}

// SetTheme delivers t to every screen that takes a theme, and to screens
// pushed later when they are pushed. It makes Stack a tui.ThemeSetter[Stack].
func (s Stack) SetTheme(t theme.Theme) Stack {
	s.th, s.themed = t, true
	s.screens = append([]Child(nil), s.screens...)
	for i, c := range s.screens {
		s.screens[i] = giveTheme(c, t)
	}
	return s
}

// Bindings returns the top screen's bindings when it has them, so a help screen
// follows the stack. It makes Stack a tui.BindingsProvider.
func (s Stack) Bindings() []keymap.Binding {
	if b, ok := s.Top().(interface{ Bindings() []keymap.Binding }); ok {
		return b.Bindings()
	}
	return nil
}
