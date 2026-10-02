package appshell

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/widgets"
)

var _ tui.Linearizer = Model{}

func TestLinearize(t *testing.T) {
	m := New("My App", 40, 5)
	m.Input.Prompt = "Search: "
	m.Content.SetContent("result one\nresult two")
	m.Hints = []widgets.Hint{{Key: "q", Action: "quit"}, {Key: "?", Action: "help"}}
	want := "My App\nSearch, text field, focused, empty\nresult one\nresult two\nKey: q, quit\nKey: ?, help"
	if got := m.Linearize(); got != want {
		t.Errorf("Linearize =\n%s\nwant\n%s", got, want)
	}
	if strings.Contains(m.Linearize(), "\x1b") {
		t.Error("no escapes allowed")
	}
}
