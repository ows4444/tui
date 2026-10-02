package tui_test

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/commandpalette"
	"github.com/ows4444/tui/datatable"
	"github.com/ows4444/tui/dialog"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/menu"
	"github.com/ows4444/tui/notificationcenter"
	"github.com/ows4444/tui/toast"
	"github.com/ows4444/tui/treeview"
)

// The two payloads from the audit: an OSC 52 clipboard write and a clear
// screen. If either survives into a widget's output, a caller string can act
// on the user's terminal.
const (
	evilClip  = "\x1b]52;c;ZXZpbA==\x07"
	evilClear = "\x1b[2J"
)

// #27, #28 for the widgets that render through View/Render, LayoutNode and
// Linearize: no ESC byte of the caller's string survives by default, and Raw
// draws it unchanged.
func TestCallerStringsAreSanitisedUnlessRaw(t *testing.T) {
	type build func(raw bool, payload string) (outputs map[string]string)
	size := layout.Size{W: 60, H: 12}
	widgets := map[string]build{
		"datatable": func(raw bool, p string) map[string]string {
			m := datatable.New([]string{"h" + p}, [][]string{{"cell" + p}})
			m.Raw = raw
			return map[string]string{"View": m.View(), "LayoutNode": m.LayoutNode().Render(size), "Linearize": m.Linearize()}
		},
		"treeview": func(raw bool, p string) map[string]string {
			m := treeview.New(treeview.Node{Label: "node" + p})
			m.Raw = raw
			return map[string]string{"View": m.View(), "LayoutNode": m.LayoutNode().Render(size), "Linearize": m.Linearize()}
		},
		"menu": func(raw bool, p string) map[string]string {
			m := menu.New([]menu.Item{{Label: "item" + p, Value: "v"}})
			m.Raw = raw
			return map[string]string{"View": m.View(), "LayoutNode": m.LayoutNode().Render(size), "Linearize": m.Linearize()}
		},
		"commandpalette": func(raw bool, p string) map[string]string {
			m := commandpalette.New(commandpalette.Command{Name: "x" + p, Description: "d" + p})
			m.Raw = raw
			m.Input.SetValue("x")
			return map[string]string{"View": m.View(), "LayoutNode": m.LayoutNode().Render(size), "Linearize": m.Linearize()}
		},
		"toast": func(raw bool, p string) map[string]string {
			m := toast.New("msg" + p)
			m.Raw = raw
			m.Show()
			return map[string]string{"Render": m.Render("base\nbase\nbase\nbase\nbase"), "LayoutNode": m.LayoutNode().Render(size), "Linearize": m.Linearize()}
		},
		"dialog": func(raw bool, p string) map[string]string {
			m := dialog.New("title"+p, "body"+p)
			m.Raw = raw
			return map[string]string{"Render": m.Render(strings.Repeat(strings.Repeat(".", 40)+"\n", 8)), "LayoutNode": m.LayoutNode().Render(size), "Linearize": m.Linearize()}
		},
		"notificationcenter": func(raw bool, p string) map[string]string {
			m := notificationcenter.New(3, 30)
			m.Raw = raw
			m.Push(notificationcenter.Notification{Message: "note" + p})
			return map[string]string{"Render": m.Render(strings.Repeat(strings.Repeat(".", 40)+"\n", 8)), "LayoutNode": m.LayoutNode().Render(size), "Linearize": m.Linearize()}
		},
	}
	for name, mk := range widgets {
		for _, payload := range []string{evilClip, evilClear} {
			for path, out := range mk(false, payload) {
				if strings.Contains(out, "\x1b]52") || strings.Contains(out, "\x1b[2J") {
					t.Errorf("%s %s: a caller escape reached the output: %q", name, path, out)
				}
			}
			// Raw: the string is drawn unchanged, so at least the Render and
			// View paths still carry it.
			raw := mk(true, payload)
			found := false
			for _, out := range raw {
				if strings.Contains(out, payload) {
					found = true
				}
			}
			if !found {
				t.Errorf("%s with Raw: none of %v carried the payload %q", name, keys(raw), payload)
			}
		}
	}
}

func keys(m map[string]string) []string {
	var k []string
	for name := range m {
		k = append(k, name)
	}
	return k
}

// A caller string cannot smuggle a second escape by splitting it around one
// the widget strips.
func TestNestedEscapeIsFullyRemoved(t *testing.T) {
	m := treeview.New(treeview.Node{Label: "a\x1b\x1b]52;c;x\x07\x1b[2Jb"})
	if out := m.View(); strings.Contains(out, "\x1b]") || strings.Contains(out, "\x1b[2J") {
		t.Fatalf("got %q", out)
	}
	_ = tui.Key{}
}
