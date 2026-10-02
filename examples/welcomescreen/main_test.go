package main

import (
	"os"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
)

// View composes Header, a BigText logo Panel and a KeyValue Info Panel
// side by side (criterion #609).
func TestViewComposesHeaderLogoAndInfoPanels(t *testing.T) {
	m := initialModel()
	view := ansi.StripANSI(m.View())

	if !strings.Contains(view, "Welcome to TUI") {
		t.Errorf("View should render the Header title, got:\n%s", view)
	}
	if !strings.Contains(view, "Info") {
		t.Errorf("View should render the Info panel title, got:\n%s", view)
	}
	if !strings.Contains(view, "Version:") || !strings.Contains(view, "v1.0.0") {
		t.Errorf("View should render KeyValue metadata, got:\n%s", view)
	}
}

// Enter, Esc and Ctrl+C all quit this static screen.
func TestKeysQuit(t *testing.T) {
	for _, kt := range []tui.KeyType{tui.KeyEnter, tui.KeyEsc, tui.KeyCtrlC} {
		m := initialModel()
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

// The package doc comment states this is composition-only, introducing no
// new widget capability (criterion #608).
func TestDocCommentStatesCompositionOnly(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("reading main.go: %v", err)
	}
	text := string(src)
	if !strings.Contains(text, "no new widget capability") {
		t.Error("doc comment should state this introduces no new widget capability")
	}
}
