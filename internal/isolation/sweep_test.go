package isolation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/accordion"
	"github.com/ows4444/tui/appshell"
	"github.com/ows4444/tui/autocomplete"
	"github.com/ows4444/tui/clipboard"
	"github.com/ows4444/tui/clockview"
	"github.com/ows4444/tui/colorpicker"
	"github.com/ows4444/tui/commandpalette"
	"github.com/ows4444/tui/confirm"
	"github.com/ows4444/tui/datatable"
	"github.com/ows4444/tui/datepicker"
	"github.com/ows4444/tui/dialog"
	"github.com/ows4444/tui/drawer"
	"github.com/ows4444/tui/emailinput"
	"github.com/ows4444/tui/errorretry"
	"github.com/ows4444/tui/faces"
	"github.com/ows4444/tui/filepicker"
	"github.com/ows4444/tui/focus"
	"github.com/ows4444/tui/form"
	"github.com/ows4444/tui/helpscreen"
	"github.com/ows4444/tui/internal/testutil"
	"github.com/ows4444/tui/loadingbar"
	logw "github.com/ows4444/tui/logview"
	"github.com/ows4444/tui/maskedinput"
	"github.com/ows4444/tui/menu"
	"github.com/ows4444/tui/multiselect"
	"github.com/ows4444/tui/numberinput"
	"github.com/ows4444/tui/passwordinput"
	"github.com/ows4444/tui/picker"
	"github.com/ows4444/tui/popover"
	"github.com/ows4444/tui/skeleton"
	"github.com/ows4444/tui/spinner"
	"github.com/ows4444/tui/streamtext"
	"github.com/ows4444/tui/tabs"
	"github.com/ows4444/tui/taginput"
	"github.com/ows4444/tui/textarea"
	"github.com/ows4444/tui/textinput"
	"github.com/ows4444/tui/toast"
	"github.com/ows4444/tui/toolapproval"
	"github.com/ows4444/tui/treeview"
	"github.com/ows4444/tui/viewport"
	"github.com/ows4444/tui/virtuallist"
	"github.com/ows4444/tui/widgets"
)

// sweep runs the shared copy-isolation check for one Model type.
func sweep[M interface{ Update(tui.Msg) (M, tui.Cmd) }](t *testing.T, name string, build func() M) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		testutil.CopyIsolation(t, build, func(m M, msg tui.Msg) M {
			n, _ := m.Update(msg)
			return n
		}, nil)
	})
}

func TestCopyIsolationSweep(t *testing.T) {
	// Text-editing family (built on internal/edit).
	sweep(t, "textinput", func() textinput.Model {
		m := textinput.New()
		m.Focus()
		m.SetValue("1234 5678")
		m.SetCursor(4)
		return m
	})
	sweep(t, "passwordinput", func() passwordinput.Model {
		m := passwordinput.New()
		m.Focus()
		m.SetValue("secret pw")
		return m
	})
	sweep(t, "maskedinput", func() maskedinput.Model {
		m := maskedinput.New()
		m.Focus()
		m.SetValue("1234")
		return m
	})
	sweep(t, "emailinput", func() emailinput.Model {
		m := emailinput.New()
		m.Focus()
		m.SetValue("a@b.co")
		return m
	})
	sweep(t, "numberinput", func() numberinput.Model {
		m := numberinput.New()
		m.Focus()
		m.SetValue("1234")
		return m
	})
	sweep(t, "searchinput", func() textinput.Model {
		m := textinput.NewSearch()
		m.Focus()
		m.SetValue("hello world")
		return m
	})
	sweep(t, "pathinput", func() textinput.Model {
		m := textinput.NewPath()
		m.Focus()
		m.SetValue("/usr/local/bin")
		return m
	})
	sweep(t, "taginput", func() taginput.Model {
		m := taginput.New()
		m.Input.Focus()
		m.Tags = []string{"x", "y"}
		m.Input.SetValue("abc")
		return m
	})
	sweep(t, "textarea", func() textarea.Model {
		m := textarea.New()
		m.Focus()
		m.SetValue("line one\nline two\nline three")
		return m
	})
	sweep(t, "autocomplete", func() autocomplete.Model {
		m := autocomplete.New("alpha", "alps", "beta")
		m.Focus()
		m.Input.SetValue("al")
		return m
	})
	sweep(t, "commandpalette", func() commandpalette.Model {
		m := commandpalette.New(
			commandpalette.Command{Name: "open", Description: "Open a file"},
			commandpalette.Command{Name: "close", Description: "Close it"},
		)
		m.Focus()
		return m
	})

	// Lists, trees, menus.
	items := []picker.Item{{Label: "A", Value: "a"}, {Label: "B", Value: "b"}, {Label: "C", Value: "c"}}
	sweep(t, "picker", func() picker.Model { return picker.New(items...) })
	sweep(t, "multiselect", func() multiselect.Model {
		its := make([]multiselect.Item, 3)
		for i := range its {
			its[i] = multiselect.Item{Label: fmt.Sprint("item", i), Value: fmt.Sprint(i)}
		}
		return multiselect.New(its...)
	})
	sweep(t, "menu", func() menu.Model {
		return menu.New([]menu.Item{
			{Label: "File", Children: []menu.Item{
				{Label: "New", Value: "new"},
				{Label: "Recent", Children: []menu.Item{{Label: "a.txt", Value: "a"}, {Label: "b.txt", Value: "b"}}},
			}},
			{Label: "Edit", Children: []menu.Item{{Label: "Undo", Value: "undo"}}},
			{Label: "Quit", Value: "quit"},
		})
	})
	sweep(t, "treeview", func() treeview.Model {
		return treeview.New(
			treeview.Node{Label: "src", Children: []treeview.Node{{Label: "a.go"}, {Label: "pkg", Children: []treeview.Node{{Label: "b.go"}}}}},
			treeview.Node{Label: "docs", Children: []treeview.Node{{Label: "x.md"}}},
		)
	})
	sweep(t, "accordion", func() accordion.Model {
		return accordion.New(
			accordion.Section{Title: "One", Content: "first\nbody"},
			accordion.Section{Title: "Two", Content: "second"},
		)
	})
	sweep(t, "tabs", func() tabs.Model { return tabs.New("one", "two", "three") })
	sweep(t, "datatable", func() datatable.Model {
		m := datatable.New([]string{"k", "v"}, [][]string{{"a", "1"}, {"b", "2"}, {"c", "3"}})
		m.Height = 2
		return m
	})
	sweep(t, "virtuallist", func() virtuallist.Model {
		return virtuallist.New(50, 5, func(i int) string { return fmt.Sprint("row ", i) })
	})
	sweep(t, "filepicker", func() filepicker.Model {
		dir := t.TempDir()
		for _, n := range []string{"a.txt", "b.txt", "c.txt"} {
			_ = os.WriteFile(filepath.Join(dir, n), nil, 0o600)
		}
		_ = os.Mkdir(filepath.Join(dir, "sub"), 0o700)
		_ = os.WriteFile(filepath.Join(dir, "sub", "d.txt"), nil, 0o600)
		return filepicker.New(dir)
	})
	sweep(t, "colorpicker", func() colorpicker.Model { return colorpicker.New() })
	sweep(t, "datepicker", func() datepicker.Model {
		return datepicker.New(time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC))
	})

	// Forms and prompts.
	sweep(t, "form", func() form.Model {
		m := form.New(
			form.Field{Name: "user", Label: "User", Value: "bob"},
			form.Field{Name: "pw", Label: "Password", Secret: true},
			form.Field{Name: "role", Label: "Role", Kind: form.FieldSelect, Options: []string{"a", "b", "c"}},
			form.Field{Name: "ok", Label: "OK", Kind: form.FieldCheckbox},
		)
		m.Focus()
		return m
	})
	sweep(t, "confirm", func() confirm.Model { return confirm.New("Sure?") })
	sweep(t, "errorretry", func() errorretry.Model { return errorretry.New("boom", 3) })
	sweep(t, "toolapproval", func() toolapproval.Model {
		return toolapproval.New("rm", "delete files", toolapproval.RiskMedium)
	})
	sweep(t, "clipboard", func() clipboard.Model { return clipboard.New("some text", "label") })

	// Overlays.
	sweep(t, "dialog", func() dialog.Model { m := dialog.New("T", "message"); m.Show(); return m })
	sweep(t, "drawer", func() drawer.Model { m := drawer.New("content\nmore"); m.Show(); return m })
	sweep(t, "popover", func() popover.Model { m := popover.New("content", 2, 2); m.Show(); return m })
	sweep(t, "toast", func() toast.Model { m := toast.New("hello"); m.Show(); return m })
	sweep(t, "helpscreen", func() helpscreen.Model {
		m := helpscreen.New(widgets.Hint{Key: "q", Action: "quit"}, widgets.Hint{Key: "?", Action: "help"})
		m.Show()
		return m
	})
	sweep(t, "appshell", func() appshell.Model { return appshell.New("title", 30, 6) })

	// Scrolling and streaming.
	sweep(t, "viewport", func() viewport.Model {
		m := viewport.New(10, 3)
		m.SetContent(strings.Repeat("a long line of text\n", 20))
		return m
	})
	sweep(t, "viewport-trimmed", func() viewport.Model {
		m := viewport.New(10, 3)
		for i := 0; i < 10; i++ {
			m.AppendLine(fmt.Sprint("line ", i))
		}
		m.TrimFront(4)
		return m
	})
	sweep(t, "log", func() logw.Model {
		m := logw.New(10, 3)
		for i := 0; i < 8; i++ {
			m.Append(fmt.Sprint("entry ", i))
		}
		return m
	})
	sweep(t, "streamtext", func() streamtext.Model {
		m := streamtext.New()
		m.SetText("streamed reply text")
		m.Append(" and more")
		return m
	})

	// Animations and timers.
	sweep(t, "spinner", func() spinner.Model { m := spinner.New(); m.Start(); return m })
	sweep(t, "loadingbar", func() loadingbar.Model { m := loadingbar.New(10); m.Start(); return m })
	sweep(t, "skeleton", func() skeleton.Model { m := skeleton.New(); m.Start(); return m })
	sweep(t, "clock", func() clockview.Model { m := clockview.New(clockview.ModeStopwatch); m.Start(); return m })
	sweep(t, "faces", func() faces.Model { return faces.New() })
}

// focus.Ring returns (Ring, bool) from Update, so it gets its own adapter.
func TestCopyIsolationFocusRing(t *testing.T) {
	testutil.CopyIsolation(t,
		func() focus.Ring { return focus.New(3) },
		func(r focus.Ring, msg tui.Msg) focus.Ring { n, _ := r.Update(msg); return n },
		nil)
}
