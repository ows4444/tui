package main

import (
	"os"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
)

// View composes BigText and Header for a static logo/tagline screen
// (criterion #611).
func TestViewComposesLogoAndHeader(t *testing.T) {
	m := initialModel()
	view := ansi.StripANSI(m.View())

	if !strings.Contains(view, "Loading your workspace") {
		t.Errorf("View should render the Header tagline, got:\n%s", view)
	}
	if !strings.Contains(view, "press any key to continue") {
		t.Errorf("View should render the dismiss hint, got:\n%s", view)
	}
}

func TestAnyKeyDismisses(t *testing.T) {
	m := initialModel()
	for _, kt := range []tui.KeyType{tui.KeyEnter, tui.KeySpace, tui.KeyRunes} {
		_, cmd := m.Update(tui.Key{Type: kt})
		if cmd == nil {
			t.Errorf("key %v should return a quit Cmd", kt)
			continue
		}
		if _, ok := cmd().(tui.QuitMsg); !ok {
			t.Errorf("key %v Cmd delivered %T, want tui.QuitMsg", kt, cmd())
		}
	}
}

func TestNonKeyMsgDoesNotDismiss(t *testing.T) {
	m := initialModel()
	_, cmd := m.Update(struct{}{})
	if cmd != nil {
		t.Error("a non-Key Msg should not dismiss the splash screen")
	}
}

func TestDocCommentStatesNoNewPattern(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("reading main.go: %v", err)
	}
	if !strings.Contains(string(src), "no new pattern") {
		t.Error("doc comment should state this introduces no new pattern")
	}
}
