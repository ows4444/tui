package tui_test

import (
	"testing"
	"time"

	"github.com/ows4444/tui/accordion"
	"github.com/ows4444/tui/autocomplete"
	"github.com/ows4444/tui/colorpicker"
	"github.com/ows4444/tui/commandpalette"
	"github.com/ows4444/tui/confirm"
	"github.com/ows4444/tui/datatable"
	"github.com/ows4444/tui/datepicker"
	"github.com/ows4444/tui/dialog"
	"github.com/ows4444/tui/drawer"
	"github.com/ows4444/tui/emailinput"
	"github.com/ows4444/tui/errorretry"
	"github.com/ows4444/tui/filepicker"
	"github.com/ows4444/tui/form"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/maskedinput"
	"github.com/ows4444/tui/menu"
	"github.com/ows4444/tui/multiselect"
	"github.com/ows4444/tui/numberinput"
	"github.com/ows4444/tui/passwordinput"
	"github.com/ows4444/tui/picker"
	"github.com/ows4444/tui/popover"
	"github.com/ows4444/tui/tabs"
	"github.com/ows4444/tui/taginput"
	"github.com/ows4444/tui/textarea"
	"github.com/ows4444/tui/textinput"
	"github.com/ows4444/tui/toolapproval"
	"github.com/ows4444/tui/treeview"
	"github.com/ows4444/tui/viewport"
	"github.com/ows4444/tui/virtuallist"
)

// hasBindings is the method every stateful widget that reacts to keys
// exposes so help text can list its active bindings.
type hasBindings interface {
	Bindings() []keymap.Binding
}

// TestKeyedWidgetsExposeBindings enumerates the interactive widgets (inputs,
// overlays, composites and the list-like widgets) and checks each one returns its
// active bindings, every one with keys and a description. A widget added to
// that family without Bindings fails to compile here once listed.
func TestKeyedWidgetsExposeBindings(t *testing.T) {
	widgets := map[string]hasBindings{
		"textinput":      textinput.New(),
		"textarea":       textarea.New(),
		"numberinput":    numberinput.New(),
		"passwordinput":  passwordinput.New(),
		"emailinput":     emailinput.New(),
		"maskedinput":    maskedinput.New(),
		"searchinput":    textinput.NewSearch(),
		"taginput":       taginput.New(),
		"autocomplete":   autocomplete.New("a"),
		"form":           form.New(form.Field{Name: "a", Label: "A"}),
		"confirm":        confirm.New("ok?"),
		"dialog":         dialog.New("t", "m"),
		"datepicker":     datepicker.New(time.Date(2024, time.March, 15, 0, 0, 0, 0, time.UTC)),
		"colorpicker":    colorpicker.New(),
		"popover":        popover.New("c", 0, 0),
		"drawer":         drawer.New("c"),
		"accordion":      accordion.New(accordion.Section{Title: "s"}),
		"viewport":       viewport.New(10, 3),
		"toolapproval":   toolapproval.New("rm", "delete", toolapproval.RiskLow),
		"errorretry":     errorretry.New("boom", 1),
		"menu":           menu.New([]menu.Item{{Label: "a"}}),
		"picker":         picker.NewStrings("a", "b"),
		"multiselect":    multiselect.NewStrings("a", "b"),
		"tabs":           tabs.New("a", "b"),
		"datatable":      datatable.New([]string{"h"}, [][]string{{"r"}}),
		"treeview":       treeview.New(treeview.Node{Label: "root"}),
		"virtuallist":    virtuallist.New(10, 3, func(i int) string { return "row" }),
		"filepicker":     filepicker.New("."),
		"commandpalette": commandpalette.New(commandpalette.Command{Name: "run"}),
	}
	for name, w := range widgets {
		bs := w.Bindings()
		if len(bs) == 0 {
			t.Errorf("%s: Bindings() is empty", name)
		}
		for _, b := range bs {
			if len(b.Keys) == 0 || b.Desc == "" {
				t.Errorf("%s: binding %+v lacks Keys or Desc", name, b)
			}
		}
	}
}
