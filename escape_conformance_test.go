package tui_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/ows4444/tui/accordion"
	"github.com/ows4444/tui/appshell"
	"github.com/ows4444/tui/autocomplete"
	"github.com/ows4444/tui/avatar"
	"github.com/ows4444/tui/clipboard"
	"github.com/ows4444/tui/commandpalette"
	"github.com/ows4444/tui/confirm"
	"github.com/ows4444/tui/contextmenu"
	"github.com/ows4444/tui/datatable"
	"github.com/ows4444/tui/dialog"
	"github.com/ows4444/tui/drawer"
	"github.com/ows4444/tui/errorretry"
	"github.com/ows4444/tui/filepicker"
	"github.com/ows4444/tui/form"
	"github.com/ows4444/tui/helpscreen"
	tuiimage "github.com/ows4444/tui/imageview"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/logview"
	"github.com/ows4444/tui/menu"
	"github.com/ows4444/tui/menubar"
	"github.com/ows4444/tui/multiselect"
	"github.com/ows4444/tui/notificationcenter"
	"github.com/ows4444/tui/picker"
	"github.com/ows4444/tui/popover"
	"github.com/ows4444/tui/spinner"
	"github.com/ows4444/tui/streamtext"
	"github.com/ows4444/tui/tabs"
	"github.com/ows4444/tui/taginput"
	"github.com/ows4444/tui/textarea"
	"github.com/ows4444/tui/textinput"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/toast"
	"github.com/ows4444/tui/toolapproval"
	"github.com/ows4444/tui/treeview"
	"github.com/ows4444/tui/viewport"
	"github.com/ows4444/tui/widgets"
	"github.com/ows4444/tui/wizard"
)

// evil is text a widget is given by its caller (or reads from disk, a
// network peer or a model): a visible marker around an OSC 52 clipboard write
// and a clear-screen. A widget conforms when the marker is drawn and neither
// escape survives.
const evil = "MARK\x1b]52;c;ZXZpbA==\x07\x1b[2JEND"

// conformance cases: each builds the widget with evil as caller text and
// returns everything it renders, by path.
var conformance = map[string]func() map[string]string{
	"accordion": func() map[string]string {
		m := accordion.New(accordion.Section{Title: evil, Content: evil})
		m.Toggle(0)
		return map[string]string{"View": m.View()}
	},
	"appshell": func() map[string]string {
		m := appshell.New(evil, 60, 6)
		m.Hints = []widgets.Hint{{Key: evil, Action: evil}}
		m.Content.SetContent(evil)
		return map[string]string{"View": m.View()}
	},
	"autocomplete": func() map[string]string {
		m := autocomplete.New(evil)
		m.Input.SetValue("M")
		return map[string]string{"View": m.View()}
	},
	"avatar": func() map[string]string {
		// The name is hashed, never drawn; Linearize is the one place it is shown.
		m := avatar.New(evil)
		return map[string]string{"View": m.View(), "Linearize": m.Linearize()}
	},
	"clipboard": func() map[string]string {
		m := clipboard.New(evil, evil)
		return map[string]string{"View": m.View()}
	},
	"commandpalette": func() map[string]string {
		m := commandpalette.New(commandpalette.Command{Name: evil, Description: evil})
		m.Input.SetValue("M")
		return map[string]string{"View": m.View(), "Linearize": m.Linearize()}
	},
	"confirm": func() map[string]string {
		m := confirm.New(evil)
		m.YesLabel, m.NoLabel = evil, evil
		return map[string]string{"View": m.View()}
	},
	"datatable": func() map[string]string {
		m := datatable.New([]string{evil}, [][]string{{evil}})
		return map[string]string{"View": m.View(), "Linearize": m.Linearize()}
	},
	"dialog": func() map[string]string {
		m := dialog.New(evil, evil)
		return map[string]string{"Render": m.Render(strings.Repeat(strings.Repeat(".", 50)+"\n", 8)), "Linearize": m.Linearize()}
	},
	"drawer": func() map[string]string {
		m := drawer.New(evil)
		m.Show()
		return map[string]string{"Render": m.Render(strings.Repeat(strings.Repeat(".", 50)+"\n", 8))}
	},
	"errorretry": func() map[string]string {
		m := errorretry.New(evil, 3)
		return map[string]string{"View": m.View()}
	},
	"filepicker": nil, // needs a file on disk; covered by TestFilepickerConformance
	"form": func() map[string]string {
		m := form.New(form.Field{Name: "n", Label: evil, Placeholder: evil})
		m.Focus()
		return map[string]string{"View": m.View()}
	},
	"imageview": func() map[string]string {
		m := tuiimage.New(nil, 40, 5, evil)
		return map[string]string{"View": m.View(), "Linearize": m.Linearize()}
	},
	"helpscreen": func() map[string]string {
		m := helpscreen.New(widgets.Hint{Key: evil, Action: evil})
		return map[string]string{"Render": m.Render(strings.Repeat(strings.Repeat(".", 60)+"\n", 10))}
	},
	"logview": func() map[string]string {
		m := logview.New(60, 4)
		m.Append(evil)
		return map[string]string{"View": m.View()}
	},
	"contextmenu": func() map[string]string {
		m := contextmenu.New(contextmenu.Item{Label: evil, Value: "v", Shortcut: evil})
		m.Show(0, 0)
		return map[string]string{"View": m.View(), "Linearize": m.Linearize()}
	},
	"menubar": func() map[string]string {
		m := menubar.New(menubar.Menu{Title: evil, Items: []menubar.Item{{Label: evil, Value: "v"}}})
		m.Show(0)
		return map[string]string{"View": m.View(), "Linearize": m.Linearize()}
	},
	"menu": func() map[string]string {
		m := menu.New([]menu.Item{{Label: evil, Value: "v"}})
		return map[string]string{"View": m.View(), "Linearize": m.Linearize()}
	},
	"multiselect": func() map[string]string {
		m := multiselect.New(multiselect.Item{Label: evil, Value: "v"})
		return map[string]string{"View": m.View()}
	},
	"notificationcenter": func() map[string]string {
		m := notificationcenter.New(3, 40)
		m.Push(notificationcenter.Notification{Message: evil})
		return map[string]string{"Render": m.Render(strings.Repeat(strings.Repeat(".", 50)+"\n", 8)), "Linearize": m.Linearize()}
	},
	"picker": func() map[string]string {
		m := picker.New(picker.Item{Label: evil, Value: "v"})
		return map[string]string{"View": m.View(), "Linearize": m.Linearize()}
	},
	"popover": func() map[string]string {
		m := popover.New(evil, 2, 2)
		return map[string]string{"Render": m.Render(strings.Repeat(strings.Repeat(".", 50)+"\n", 8))}
	},
	"spinner": func() map[string]string {
		m := spinner.New()
		m.Label = evil
		m.Frames = []string{evil}
		return map[string]string{"View": m.View()}
	},
	"streamtext": func() map[string]string {
		m := streamtext.New()
		m.SetText(evil)
		m.Skip()
		return map[string]string{"View": m.View()}
	},
	"tabs": func() map[string]string {
		m := tabs.New(evil)
		return map[string]string{"View": m.View()}
	},
	"taginput": func() map[string]string {
		m := taginput.New()
		m.Tags = []string{evil}
		return map[string]string{"View": m.View()}
	},
	"textarea": func() map[string]string {
		m := textarea.New()
		m.Placeholder = evil
		return map[string]string{"View": m.View()}
	},
	"textinput": func() map[string]string {
		m := textinput.New()
		m.Placeholder, m.Prompt = evil, evil
		return map[string]string{"View": m.View()}
	},
	"toast": func() map[string]string {
		m := toast.New(evil)
		m.Show()
		return map[string]string{"Render": m.Render(strings.Repeat(strings.Repeat(".", 50)+"\n", 8)), "Linearize": m.Linearize()}
	},
	"toolapproval": func() map[string]string {
		m := toolapproval.New(evil, evil, toolapproval.RiskLow)
		return map[string]string{"View": m.View()}
	},
	"treeview": func() map[string]string {
		m := treeview.New(treeview.Node{Label: evil})
		return map[string]string{"View": m.View(), "Linearize": m.Linearize()}
	},
	"viewport": func() map[string]string {
		m := viewport.New(60, 4)
		m.SetContent(evil)
		return map[string]string{"View": m.View()}
	},
	"wizard": func() map[string]string {
		m := wizard.New(evil, "b")
		return map[string]string{"View": m.View(theme.DarkTheme())}
	},
}

// exempt lists widget packages that take no caller text, with the reason.
var exempt = map[string]string{
	"clockview":     "renders a time it computes; Layout is a time format, not display text",
	"colorpicker":   "takes colours, not text",
	"datepicker":    "takes a time.Time",
	"faces":         "fixed face set; no caller text",
	"loadingbar":    "no caller text",
	"skeleton":      "no caller text",
	"virtuallist":   "RenderItem returns the row: the caller's own rendering, styled by design (Raw by construction)",
	"maskedinput":   "renders the user's typed value under a mask rune",
	"passwordinput": "renders a mask of the typed value",
	"emailinput":    "a textinput.Model wrapper; covered by textinput",
	"numberinput":   "a textinput.Model wrapper; covered by textinput",
	"scrollbar":     "draws a thumb and track from numbers; no caller text",
	"splitpane":     "draws only a divider; the panes are caller layout.Nodes that sanitise their own content",
	"markdown":      "Render sanitises its input (markdown/render.go); has its own tests",
}

// knownUnsanitised lists widgets that draw a caller string as given. It is
// the explicit "Raw" list: each entry is a gap, not a design, and a widget
// leaves it by being fixed (the test then fails until the entry is removed).
var knownUnsanitised = map[string]string{
	"accordion":    "section Title and Content",
	"appshell":     "Title and key-hint text (Content is a viewport, which sanitises)",
	"autocomplete": "suggestion text",
	"clipboard":    "Text and Label",
	"confirm":      "Prompt, YesLabel, NoLabel",
	"drawer":       "Content",
	"errorretry":   "Message",
	"form":         "field Label and Placeholder",
	"helpscreen":   "hint Key and Action",
	"multiselect":  "item Label",
	"picker":       "item Label (menu sanitises its own copy)",
	"popover":      "Content",
	"spinner":      "Label and Frames",
	"tabs":         "tab labels",
	"taginput":     "tags",
	"textarea":     "Placeholder",
	"textinput":    "Placeholder and Prompt",
	"toolapproval": "ToolName and Description, often supplied by a model",
	"wizard":       "step titles",
}

// widgetPackages returns every top-level library package that defines a
// Model with View or Render, found by parsing the tree, so a new widget
// cannot be added without being put in conformance or exempt.
func widgetPackages(t *testing.T) []string {
	t.Helper()
	dirs, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, d := range dirs {
		switch d.Name() {
		case "internal", "examples", "tools", "bench", "docs", "tuitest", "widgets", "layout", "ansi", "theme", "term", "input", "motion", "focus", "hittest", "keymap", "faces", "testdata":
			if d.Name() != "faces" {
				continue
			}
		}
		if !d.IsDir() || strings.HasPrefix(d.Name(), ".") {
			continue
		}
		fset := token.NewFileSet()
		pkgs, err := parser.ParseDir(fset, d.Name(), func(fi os.FileInfo) bool { return !strings.HasSuffix(fi.Name(), "_test.go") }, 0)
		if err != nil {
			t.Fatal(err)
		}
		has := false
		for _, p := range pkgs {
			for _, f := range p.Files {
				for _, decl := range f.Decls {
					fn, ok := decl.(*ast.FuncDecl)
					if !ok || fn.Recv == nil || (fn.Name.Name != "View" && fn.Name.Name != "Render") {
						continue
					}
					has = true
				}
			}
		}
		if has {
			out = append(out, d.Name())
		}
	}
	sort.Strings(out)
	return out
}

func hasEscape(out string) bool {
	return strings.Contains(out, "\x1b]52") || strings.Contains(out, "\x1b[2J")
}

// #31: every library widget that accepts caller text is exercised with an
// ESC-bearing string and either passes or is on the explicit list.
func TestEveryWidgetSanitisesCallerText(t *testing.T) {
	for _, pkg := range widgetPackages(t) {
		build, tested := conformance[pkg]
		reason, isExempt := exempt[pkg]
		if !tested && !isExempt {
			t.Errorf("widget package %q is in neither conformance nor exempt: add a case that feeds it an ESC-bearing string", pkg)
			continue
		}
		if isExempt && reason == "" {
			t.Errorf("exempt %q needs a reason", pkg)
		}
		if build == nil {
			continue
		}
		outs := build()
		rendered := false
		leaked := ""
		for path, out := range outs {
			if strings.Contains(out, "MARK") {
				rendered = true
			}
			if hasEscape(out) {
				leaked = path
			}
		}
		if !rendered {
			t.Errorf("%s: the marker text was not drawn by %v, so the case proves nothing", pkg, keysOf(outs))
			continue
		}
		note, listed := knownUnsanitised[pkg]
		switch {
		case leaked != "" && !listed:
			t.Errorf("%s %s: a caller escape reached the output; sanitise it (ansi.Clean) or list it in knownUnsanitised", pkg, leaked)
		case leaked == "" && listed:
			t.Errorf("%s now sanitises; remove it from knownUnsanitised (%s)", pkg, note)
		}
	}
	for name := range conformance {
		found := false
		for _, p := range widgetPackages(t) {
			if p == name {
				found = true
			}
		}
		if !found {
			t.Errorf("conformance case %q is not a widget package", name)
		}
	}
}

func keysOf(m map[string]string) []string {
	var k []string
	for name := range m {
		k = append(k, name)
	}
	sort.Strings(k)
	return k
}

// The filepicker reads names from disk.
func TestFilepickerConformance(t *testing.T) {
	dir := t.TempDir()
	name := "MARK\x1b]52;c;ZXZpbA==\x07\x1b[2JEND"
	if err := os.WriteFile(dir+"/"+name, nil, 0o644); err != nil {
		t.Skipf("this file system cannot hold ESC in a name: %v", err)
	}
	m := filepicker.New(dir)
	for path, out := range map[string]string{"View": m.View(), "Linearize": m.Linearize(), "LayoutNode": m.LayoutNode().Render(layout.Size{W: 60, H: 4})} {
		if !strings.Contains(out, "MARK") || hasEscape(out) {
			t.Errorf("filepicker %s: %q", path, out)
		}
	}
}
