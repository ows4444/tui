// Package widgetshots draws one picture of each component with a small, real
// program, checks the text of every picture against a golden file, and, when
// TUI_WIDGET_SHOTS_DIR is set, writes each as an SVG for the documentation
// site's widget gallery (site/scripts/sync-docs.mjs). A component that changes
// how it looks fails here until its golden is rewritten with TUITEST_UPDATE=1,
// so the gallery cannot drift from the code.
package widgetshots

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/accordion"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/avatar"
	"github.com/ows4444/tui/breadcrumb"
	"github.com/ows4444/tui/button"
	"github.com/ows4444/tui/buttongroup"
	"github.com/ows4444/tui/carousel"
	"github.com/ows4444/tui/checkbox"
	"github.com/ows4444/tui/colorpicker"
	"github.com/ows4444/tui/confirm"
	"github.com/ows4444/tui/datatable"
	"github.com/ows4444/tui/datepicker"
	"github.com/ows4444/tui/emailinput"
	"github.com/ows4444/tui/loadingbar"
	"github.com/ows4444/tui/menu"
	"github.com/ows4444/tui/multiselect"
	"github.com/ows4444/tui/numberinput"
	"github.com/ows4444/tui/otpinput"
	"github.com/ows4444/tui/pagination"
	"github.com/ows4444/tui/passwordinput"
	"github.com/ows4444/tui/picker"
	"github.com/ows4444/tui/radiogroup"
	"github.com/ows4444/tui/rating"
	"github.com/ows4444/tui/scrollbar"
	"github.com/ows4444/tui/skeleton"
	"github.com/ows4444/tui/slider"
	"github.com/ows4444/tui/tabs"
	"github.com/ows4444/tui/taginput"
	"github.com/ows4444/tui/textinput"
	"github.com/ows4444/tui/toggle"
	"github.com/ows4444/tui/transferlist"
	"github.com/ows4444/tui/treeview"
	"github.com/ows4444/tui/tuitest"
)

// comp is the shape of a component Model: it updates into its own type.
type comp[T any] interface {
	Update(tui.Msg) (T, tui.Cmd)
	View() string
}

// host runs a component as the root of a Program.
type host[T comp[T]] struct{ m T }

func (host[T]) Init() tui.Cmd { return nil }
func (h host[T]) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	var c tui.Cmd
	h.m, c = h.m.Update(msg)
	return h, c
}
func (h host[T]) View() string { return h.m.View() }

// shot is one picture: a package, a model, a size and the keys pressed first.
type shot struct {
	pkg   string // the component's package, and the picture's name
	title string
	w, h  int
	model func() tui.Model
	keys  []string
}

func add[T comp[T]](out *[]shot, pkg, title string, w, h int, m func() T, keys ...string) {
	*out = append(*out, shot{pkg: pkg, title: title, w: w, h: h, keys: keys, model: func() tui.Model { return host[T]{m()} }})
}

func shots() []shot {
	var s []shot
	add(&s, "button", "Button", 30, 3, func() button.Model { return button.New("Save changes") })
	add(&s, "buttongroup", "Button group", 40, 3, func() buttongroup.Model {
		return buttongroup.New(buttongroup.ModeSingle, "Day", "Week", "Month")
	})
	add(&s, "checkbox", "Checkbox", 30, 3, func() checkbox.Model { return checkbox.New("Remember me") }, "space")
	add(&s, "toggle", "Toggle", 30, 3, func() toggle.Model { return toggle.New("Dark mode") }, "space")
	add(&s, "radiogroup", "Radio group", 30, 5, func() radiogroup.Model { return radiogroup.New("Small", "Medium", "Large") }, "down")
	add(&s, "slider", "Slider", 40, 3, func() slider.Model { m := slider.New(0, 100); m.SetValue(40); return m })
	add(&s, "rating", "Rating", 30, 3, func() rating.Model { return rating.New(5) }, "3")
	add(&s, "tabs", "Tabs", 44, 3, func() tabs.Model { return tabs.New("Overview", "Logs", "Settings") }, "right")
	add(&s, "breadcrumb", "Breadcrumb", 44, 3, func() breadcrumb.Model { return breadcrumb.New("Home", "Docs", "Widgets") })
	add(&s, "pagination", "Pagination", 44, 3, func() pagination.Model { return pagination.New(12) }, "right", "right")
	add(&s, "accordion", "Accordion", 44, 7, func() accordion.Model {
		return accordion.New(accordion.Section{Title: "Install", Content: "go get github.com/ows4444/tui"}, accordion.Section{Title: "Run", Content: "go run ."})
	}, "enter")
	add(&s, "picker", "Picker", 40, 7, func() picker.Model {
		return picker.New(picker.Item{Label: "Berlin", Value: "BER"}, picker.Item{Label: "Lisbon", Value: "LIS"}, picker.Item{Label: "Oslo", Value: "OSL"})
	}, "down")
	add(&s, "multiselect", "Multi-select", 40, 6, func() multiselect.Model {
		return multiselect.New(multiselect.Item{Label: "Tests", Value: "t"}, multiselect.Item{Label: "Lint", Value: "l"}, multiselect.Item{Label: "Docs", Value: "d"})
	}, "space", "down", "down", "space")
	add(&s, "treeview", "Tree view", 40, 8, func() treeview.Model {
		return treeview.New(treeview.Node{Label: "src", Children: []treeview.Node{{Label: "main.go"}, {Label: "util", Children: []treeview.Node{{Label: "a.go"}}}}}, treeview.Node{Label: "README.md"})
	}, "enter")
	add(&s, "datatable", "Data table", 40, 7, func() datatable.Model {
		m := datatable.New([]string{"name", "qty"}, [][]string{{"apple", "3"}, {"pear", "5"}, {"plum", "8"}})
		m.Height = 4
		return m
	}, "down")
	add(&s, "textinput", "Text input", 40, 3, func() textinput.Model { m := textinput.NewSearch(); m.Focus(); return m }, "h", "e", "l", "l", "o")
	add(&s, "passwordinput", "Password input", 40, 3, func() passwordinput.Model { m := passwordinput.New(); m.Focus(); return m }, "s", "e", "c", "r", "e", "t")
	add(&s, "emailinput", "Email input", 40, 3, func() emailinput.Model { m := emailinput.New(); m.Focus(); return m }, "a", "d", "a", "@", "x", ".", "i", "o")
	add(&s, "numberinput", "Number input", 40, 3, func() numberinput.Model { m := numberinput.New(); m.Focus(); return m }, "4", "2")
	add(&s, "otpinput", "One-time code", 40, 3, func() otpinput.Model { m := otpinput.New(6); m.Focus(); return m }, "1", "2", "3")
	add(&s, "taginput", "Tag input", 40, 3, func() taginput.Model { m := taginput.New(); m.Tags = []string{"go", "tui", "terminal"}; return m })
	add(&s, "loadingbar", "Loading bar", 40, 3, func() loadingbar.Model { return loadingbar.New(30) })
	add(&s, "scrollbar", "Scrollbar", 20, 6, func() scrollbar.Model { m := scrollbar.New(100, 10); m.Length, m.Offset = 5, 30; return m })
	add(&s, "skeleton", "Skeleton", 40, 4, func() skeleton.Model { m := skeleton.New(); m.Width, m.Lines = 30, 3; return m })
	add(&s, "datepicker", "Date picker", 40, 9, func() datepicker.Model {
		return datepicker.New(time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC))
	})
	add(&s, "colorpicker", "Color picker", 40, 4, func() colorpicker.Model {
		return colorpicker.New(ansi.RGB{R: 0xe0, G: 0x40, B: 0x40}, ansi.RGB{R: 0x40, G: 0xa0, B: 0x40}, ansi.RGB{R: 0x40, G: 0x60, B: 0xe0})
	})
	add(&s, "confirm", "Confirm", 40, 3, func() confirm.Model { return confirm.New("Delete this file?") })
	add(&s, "avatar", "Avatar", 24, 5, func() avatar.Model { return avatar.New("Ada Lovelace") })
	add(&s, "carousel", "Carousel", 40, 5, func() carousel.Model { return carousel.New("first slide", "second slide", "third slide") }, "right")
	add(&s, "transferlist", "Transfer list", 44, 7, func() transferlist.Model { return transferlist.New("alpha", "beta", "gamma", "delta") }, "space")
	add(&s, "menu", "Menu", 30, 6, func() menu.Model {
		return menu.New([]menu.Item{{Label: "File", Value: "f"}, {Label: "Edit", Value: "e"}, {Label: "View", Value: "v"}})
	}, "down")
	return s
}

func TestWidgetShots(t *testing.T) {
	dir := os.Getenv("TUI_WIDGET_SHOTS_DIR")
	if dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil { // #nosec G301 -- generated documentation output
			t.Fatal(err)
		}
	}
	type entry struct {
		Package string `json:"package"`
		Title   string `json:"title"`
		SVG     string `json:"svg"`
	}
	var index []entry
	for _, sh := range shots() {
		t.Run(sh.pkg, func(t *testing.T) {
			s := tuitest.New(sh.model(), sh.w, sh.h)
			defer s.Close()
			s.Keys(sh.keys...)
			blank := true
			for y := 0; y < sh.h && blank; y++ {
				for x := 0; x < sh.w; x++ {
					if c := s.Cell(x, y); (c.Grapheme != " " && c.Grapheme != "") || c.BG != nil {
						blank = false
						break
					}
				}
			}
			if blank {
				t.Fatal("the component drew nothing")
			}
			s.Golden(t, sh.pkg)
			if dir != "" {
				name := sh.pkg + ".svg"
				if err := os.WriteFile(filepath.Join(dir, name), []byte(s.SVG(tuitest.SVGOptions{Title: sh.pkg, TrimRows: true})), 0o644); err != nil { // #nosec G306 -- generated documentation output
					t.Fatal(err)
				}
				index = append(index, entry{Package: sh.pkg, Title: sh.title, SVG: name})
			}
		})
	}
	if dir != "" {
		raw, err := json.MarshalIndent(index, "", "\t")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "index.json"), append(raw, '\n'), 0o644); err != nil { // #nosec G306 -- generated documentation output
			t.Fatal(err)
		}
	}
}
