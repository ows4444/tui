package appshell

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/theme"
)

type confirmResult struct{ Yes bool }

// screen is a test Child that records what reaches it.
type screen struct {
	name  string
	log   []string
	push  Child // pushed on "p"
	size  tui.ResizeMsg
	theme bool
}

func (s *screen) Update(msg tui.Msg) (Child, tui.Cmd) {
	switch m := msg.(type) {
	case tui.Key:
		s.log = append(s.log, "key:"+m.Text)
		switch m.Text {
		case "p":
			return s, PushCmd(s.push)
		case "y":
			return s, PopCmd(confirmResult{Yes: true})
		case "n":
			return s, PopCmd(nil)
		}
	case tui.ResizeMsg:
		s.size = m
		s.log = append(s.log, "resize")
	case confirmResult:
		s.log = append(s.log, "result:"+map[bool]string{true: "yes", false: "no"}[m.Yes])
	default:
		s.log = append(s.log, "msg")
	}
	return s, nil
}

func (s *screen) View() string { return s.name }

func (s *screen) SetTheme(theme.Theme) Child { c := *s; c.theme = true; return &c }

func (s *screen) Bindings() []keymap.Binding {
	return []keymap.Binding{keymap.NewBinding(s.name+" help", "?")}
}

func rune_(t string) tui.Key { return tui.Key{Type: tui.KeyRunes, Text: t} }

// settle feeds a Cmd's message back to the stack until no Cmd is left.
func settle(s Stack, cmd tui.Cmd) Stack {
	for cmd != nil {
		msg := cmd()
		s, cmd = s.Update(msg)
	}
	return s
}

func TestModalFlowHandsBackTheAnswer(t *testing.T) {
	home := &screen{name: "home", push: &screen{name: "confirm"}}
	s := NewStack(home)
	s, cmd := s.Update(rune_("p"))
	s = settle(s, cmd)
	if s.Depth() != 2 || s.View() != "confirm" || s.ViewAt(0) != "home" {
		t.Fatalf("depth %d view %q below %q", s.Depth(), s.View(), s.ViewAt(0))
	}
	s, cmd = s.Update(rune_("y"))
	s = settle(s, cmd)
	if s.Depth() != 1 || s.View() != "home" {
		t.Fatalf("after pop: depth %d view %q", s.Depth(), s.View())
	}
	if got := strings.Join(home.log, ","); got != "key:p,result:yes" {
		t.Fatalf("home saw %q; want its key and the answer", got)
	}
}

func TestKeysGoToTheTopOnly(t *testing.T) {
	under, over := &screen{name: "under"}, &screen{name: "over"}
	s, _ := NewStack(under).Push(over)
	s, _ = s.Update(rune_("x"))
	if len(under.log) != 0 || len(over.log) != 1 {
		t.Fatalf("under %v over %v", under.log, over.log)
	}
	s, _ = s.Update(tui.MouseEvent{X: 1, Y: 1, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if len(under.log) != 0 {
		t.Fatal("a mouse event reached the screen under the top")
	}
	_, _ = s.Update(struct{}{})
	if len(under.log) != 1 || under.log[0] != "msg" {
		t.Fatalf("other messages must reach every screen: %v", under.log)
	}
}

func TestPopWithoutResultAndRootStays(t *testing.T) {
	s, _ := NewStack(&screen{name: "a"}).Push(&screen{name: "b"})
	s, cmd := s.Update(rune_("n"))
	if cmd == nil {
		t.Fatal("the popping screen's Cmd is gone")
	}
	s = settle(s, cmd)
	if s.Depth() != 1 {
		t.Fatalf("depth %d after a pop without result", s.Depth())
	}
	if s2, cmd := s.Pop(confirmResult{}); s2.Depth() != 1 || cmd != nil {
		t.Fatal("the root screen was popped")
	}
}

func TestPushedScreenGetsTheLastResize(t *testing.T) {
	s := NewStack(&screen{name: "a"})
	s, _ = s.Update(tui.ResizeMsg{Width: 80, Height: 24})
	pushed := &screen{name: "b"}
	s, _ = s.Push(pushed)
	if pushed.size.Width != 80 {
		t.Fatalf("pushed screen has size %+v", pushed.size)
	}
	next := &screen{name: "c"}
	s, _ = s.Replace(next)
	if next.size.Height != 24 || s.Depth() != 2 || s.View() != "c" {
		t.Fatalf("replace: depth %d view %q size %+v", s.Depth(), s.View(), next.size)
	}
}

func TestPushNilAndReplaceNil(t *testing.T) {
	s := NewStack(&screen{name: "a"})
	if s2, _ := s.Push(nil); s2.Depth() != 1 {
		t.Fatal("pushed nil")
	}
	if s2, _ := s.Replace(nil); s2.View() != "a" {
		t.Fatal("replaced with nil")
	}
	if s.ViewAt(-1) != "" || s.ViewAt(5) != "" {
		t.Fatal("ViewAt out of range is not empty")
	}
}

func TestStackIsAValue(t *testing.T) {
	s := NewStack(&screen{name: "a"})
	s2, _ := s.Push(&screen{name: "b"})
	if s.Depth() != 1 || s2.Depth() != 2 {
		t.Fatal("Push changed the receiver")
	}
	s3, _ := s2.Pop(nil)
	if s2.Depth() != 2 || s3.Depth() != 1 {
		t.Fatal("Pop changed the receiver")
	}
}

func TestStackThemeAndBindings(t *testing.T) {
	s := NewStack(&screen{name: "a"}).SetTheme(theme.DarkTheme())
	s, _ = s.Push(&screen{name: "b"})
	if !s.Top().(*screen).theme || !s.screens[0].(*screen).theme {
		t.Fatal("the theme did not reach every screen, pushed ones included")
	}
	if b := s.Bindings(); len(b) != 1 || b[0].Desc != "b help" {
		t.Fatalf("bindings %+v follow the top screen", b)
	}
	plain := NewStack(Wrap(plainWidget{}))
	if plain.Bindings() != nil {
		t.Fatal("a screen with no bindings reported some")
	}
}

type plainWidget struct{}

func (p plainWidget) Update(tui.Msg) (plainWidget, tui.Cmd) { return p, nil }
func (plainWidget) View() string                            { return "w" }
