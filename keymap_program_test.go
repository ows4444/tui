package tui_test

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/helpscreen"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/picker"
)

// app is a root model with one focused widget; its Bindings are the widget's,
// which is all an app writes to get help that follows the widget.
type app struct{ list picker.Model }

func (a app) Init() tui.Cmd                       { return nil }
func (a app) Update(tui.Msg) (tui.Model, tui.Cmd) { return a, nil }
func (a app) View() string                        { return a.list.View() }
func (a app) Bindings() []keymap.Binding          { return a.list.Bindings() }

// TestHelpscreenFollowsTheFocusedWidgetsKeyMap proves criterion #49: when the
// widget's KeyMap changes, the help screen shows the new keys, and the app
// does nothing more than hand Program.Keymap() to helpscreen.FromRegistry.
func TestHelpscreenFollowsTheFocusedWidgetsKeyMap(t *testing.T) {
	list := picker.NewStrings("a", "b")
	help := func(m tui.Model) string {
		p := tui.NewProgram(m)
		return helpscreen.FromRegistry(p.Keymap(), "").Render(strings.Repeat("\n", 12))
	}

	before := help(app{list: list})
	if !strings.Contains(before, "down") || !strings.Contains(before, "select") {
		t.Fatalf("default help missing the picker's bindings:\n%s", before)
	}

	list.KeyMap.Down = keymap.NewBinding("down", "j")
	after := help(app{list: list})
	if before == after {
		t.Fatal("help is identical after rebinding")
	}
	hints := tui.NewProgram(app{list: list}).Keymap().Hints("")
	var down keymap.Hint
	for _, h := range hints {
		if h.Desc == "down" {
			down = h
		}
	}
	if down.Key != "j" {
		t.Fatalf("Down hint = %+v, want key j (hints %v)", down, hints)
	}
}

func TestKeymapIsEmptyForAModelWithoutBindings(t *testing.T) {
	p := tui.NewProgram(plainModel{})
	if got := p.Keymap().Hints(""); len(got) != 0 {
		t.Fatalf("Hints = %v, want none", got)
	}
}

type plainModel struct{}

func (plainModel) Init() tui.Cmd                       { return nil }
func (plainModel) Update(tui.Msg) (tui.Model, tui.Cmd) { return plainModel{}, nil }
func (plainModel) View() string                        { return "" }
