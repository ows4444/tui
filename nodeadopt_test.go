package tui_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/accordion"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/appshell"
	"github.com/ows4444/tui/autocomplete"
	"github.com/ows4444/tui/clipboard"
	"github.com/ows4444/tui/clockview"
	"github.com/ows4444/tui/colorpicker"
	"github.com/ows4444/tui/confirm"
	"github.com/ows4444/tui/datatable"
	"github.com/ows4444/tui/datepicker"
	"github.com/ows4444/tui/dialog"
	"github.com/ows4444/tui/drawer"
	"github.com/ows4444/tui/errorretry"
	"github.com/ows4444/tui/faces"
	"github.com/ows4444/tui/filepicker"
	"github.com/ows4444/tui/form"
	"github.com/ows4444/tui/helpscreen"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/loadingbar"
	applog "github.com/ows4444/tui/logview"
	"github.com/ows4444/tui/menu"
	"github.com/ows4444/tui/multiselect"
	"github.com/ows4444/tui/notificationcenter"
	"github.com/ows4444/tui/picker"
	"github.com/ows4444/tui/popover"
	"github.com/ows4444/tui/skeleton"
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
	"github.com/ows4444/tui/virtuallist"
	"github.com/ows4444/tui/widgets"
	"github.com/ows4444/tui/wizard"
)

// TestWidgetNodesComposeAtEveryTerminalSize builds one realistic screen from
// the widgets' LayoutNode adapters (a table, a log viewport and a text
// input inside a header, a sidebar and a status bar) and renders it at sizes
// from a cramped 40x10 to a huge 300x80, plus awkward ones in between. Every
// render must be exactly the terminal's size, with nothing spilling over.
func TestWidgetNodesComposeAtEveryTerminalSize(t *testing.T) {
	rows := make([][]string, 40)
	for i := range rows {
		rows[i] = []string{fmt.Sprintf("job-%02d", i), "a fairly long description of what this job does", "running"}
	}
	table := datatable.New([]string{"Job", "Description", "State"}, rows)
	table.SetCursor(23)

	logs := viewport.New(1, 1) // its own size is irrelevant inside a layout
	var logLines []string
	for i := 0; i < 200; i++ {
		logLines = append(logLines, fmt.Sprintf("12:%02d:%02d INFO  worker %d finished a task", i/60, i%60, i%7))
	}
	logs.SetContent(strings.Join(logLines, "\n"))
	logs.LineDown(50)

	input := textarea.New()
	input.SetValue("first line of the note\nsecond line\nthird line with more text than the box is wide")
	input.Focus()

	screen := layout.Column(0,
		layout.FlexChild{Node: layout.BoxNode(layout.NewBox().Border(layout.NormalBorder()), layout.Block("Jobs dashboard"))},
		// Fill: scrollable content (a long table or log) measures to its full
		// length, so panels that should just take what is left use Fill.
		layout.Fill(layout.Row(1,
			layout.FlexChild{Node: layout.Block("nav\nnav\nnav"), Basis: 8, Shrink: 1, Min: 3},
			layout.FillWeight(table.LayoutNode(), 2),
			layout.FillWeight(layout.Column(0,
				layout.FillWeight(logs.LayoutNode(), 2),
				layout.FlexChild{Basis: 4, Shrink: 1, Node: layout.BoxNode(layout.NewBox().Border(layout.NormalBorder()), input.LayoutNode())},
			), 3),
		)),
		layout.FlexChild{Node: layout.RowJustify(2, layout.JustifySpaceBetween,
			layout.FlexChild{Node: layout.Block("ready")}, layout.FlexChild{Node: layout.Block("q: quit"), CrossAlign: layout.CrossEnd})},
	)

	for _, size := range [][2]int{{40, 10}, {80, 24}, {120, 40}, {300, 80}, {41, 11}, {57, 13}, {20, 6}, {5, 3}} {
		w, h := size[0], size[1]
		out := layout.Draw(screen, layout.Constraints{MinW: w, MaxW: w, MinH: h, MaxH: h})
		lines := strings.Split(out, "\n")
		if len(lines) != h {
			t.Errorf("%dx%d: %d rows, want %d", w, h, len(lines), h)
			continue
		}
		for i, l := range lines {
			if got := ansi.Width(l); got != w {
				t.Errorf("%dx%d: row %d is %d wide", w, h, i, got)
				break
			}
		}
	}

	// At a comfortable size the pieces really are all on screen. (The input's
	// box is two rows tall and the cursor is on its third line, so it shows
	// the window containing the cursor: lines two and three.)
	out := ansi.StripANSI(layout.Draw(screen, layout.Constraints{MinW: 120, MaxW: 120, MinH: 40, MaxH: 40}))
	for _, want := range []string{"Jobs dashboard", "job-23", "worker", "second line", "q: quit"} {
		if !strings.Contains(out, want) {
			t.Errorf("120x40 screen lacks %q", want)
		}
	}
}

// TestTabsTreeListAndInputNodesComposeAtEveryTerminalSize does the same for
// the second batch of adapters: a tab bar, a file tree, a choice list and a
// text input in one nested screen.
func TestTabsTreeListAndInputNodesComposeAtEveryTerminalSize(t *testing.T) {
	bar := tabs.New("Files", "Search", "Settings", "Help", "About the project")
	bar.SetActive(3)

	var kids []treeview.Node
	for i := 0; i < 30; i++ {
		kids = append(kids, treeview.Node{Label: fmt.Sprintf("file-%02d.go", i)})
	}
	tree := treeview.New(treeview.Node{Label: "src", Children: kids})
	tree.SetCursor(0)

	var items []string
	for i := 0; i < 25; i++ {
		items = append(items, fmt.Sprintf("choice %02d", i))
	}
	list := picker.NewStrings(items...)
	list.SetCursor(17)

	field := textinput.New()
	field.Prompt = "Search: "
	field.SetValue("a fairly long query that will not fit in a narrow terminal")
	field.Focus()

	screen := layout.Column(0,
		layout.FlexChild{Node: bar.LayoutNode()},
		layout.Fill(layout.Row(1,
			layout.FillWeight(tree.LayoutNode(), 1),
			layout.FillWeight(list.LayoutNode(), 1),
		)),
		layout.FlexChild{Node: field.LayoutNode()},
	)

	for _, size := range [][2]int{{40, 10}, {80, 24}, {120, 40}, {300, 80}, {41, 11}, {57, 13}, {20, 6}, {5, 3}} {
		w, h := size[0], size[1]
		out := layout.Draw(screen, layout.Constraints{MinW: w, MaxW: w, MinH: h, MaxH: h})
		lines := strings.Split(out, "\n")
		if len(lines) != h {
			t.Errorf("%dx%d: %d rows, want %d", w, h, len(lines), h)
			continue
		}
		for i, l := range lines {
			if got := ansi.Width(l); got != w {
				t.Errorf("%dx%d: row %d is %d wide", w, h, i, got)
				break
			}
		}
	}

	out := ansi.StripANSI(layout.Draw(screen, layout.Constraints{MinW: 100, MaxW: 100, MinH: 12, MaxH: 12}))
	for _, want := range []string{"Help", "choice 17", "Search:"} {
		if !strings.Contains(out, want) {
			t.Errorf("100x12 screen lacks %q:\n%s", want, out)
		}
	}
}

// TestRemainingWidgetNodesComposeAtEveryTerminalSize completes the set: the
// last twelve LayoutNode adapters in one nested screen, from a cramped
// 40x10 to a huge 300x80 and awkward sizes in between.
func TestRemainingWidgetNodesComposeAtEveryTerminalSize(t *testing.T) {
	list := virtuallist.New(100000, 5, func(i int) string { return fmt.Sprintf("row %d of a very long list", i) })
	list.LineDown(1234)

	logs := applog.New(40, 5)
	for i := 0; i < 500; i++ {
		logs.Append(fmt.Sprintf("12:00:%02d event %d happened", i%60, i))
	}

	var secs []accordion.Section
	for i := 0; i < 8; i++ {
		secs = append(secs, accordion.Section{Title: fmt.Sprintf("Section %d", i), Content: "first line\nsecond line"})
	}
	acc := accordion.New(secs...)

	files := filepicker.New(".")
	auto := autocomplete.New("alpha", "alpine", "alto", "altitude", "amber", "ample", "anchor", "angle")
	auto.Input.Prompt = "Go to: "
	auto.Input.SetValue("a")

	var choices []string
	for i := 0; i < 30; i++ {
		choices = append(choices, fmt.Sprintf("option %d", i))
	}
	multi := multiselect.NewStrings(choices...)
	multi.Toggle(3)
	multi.SetCursor(20)

	var kids []menu.Item
	for i := 0; i < 15; i++ {
		kids = append(kids, menu.Item{Label: fmt.Sprintf("child %d", i)})
	}
	nav := menu.New([]menu.Item{{Label: "Files", Children: kids}, {Label: "Edit"}})

	tags := taginput.New()
	tags.Tags = []string{"go", "tui", "layout", "accessibility"}
	tags.Input.SetValue("one more tag that is long")

	shimmer := skeleton.New()
	bar := loadingbar.New(8)
	bar.Start()
	stream := streamtext.New()
	stream.SetText("This text streams in and wraps to whatever width the layout gives it, keeping the newest lines.")
	stream.Skip()

	screen := layout.Column(0,
		layout.FlexChild{Node: bar.LayoutNode()},
		layout.Fill(layout.Row(1,
			layout.FillWeight(layout.Column(0,
				layout.Fill(list.LayoutNode()),
				layout.Fill(logs.LayoutNode()),
			), 2),
			layout.FillWeight(layout.Column(0,
				layout.Fill(acc.LayoutNode()),
				layout.Fill(files.LayoutNode()),
				layout.Fill(multi.LayoutNode()),
			), 2),
			layout.FillWeight(layout.Column(0,
				layout.Fill(nav.LayoutNode()),
				layout.Fill(shimmer.LayoutNode()),
				layout.Fill(stream.LayoutNode()),
			), 1),
		)),
		layout.FlexChild{Node: auto.LayoutNode()},
		layout.FlexChild{Node: tags.LayoutNode()},
	)

	for _, size := range [][2]int{{40, 10}, {80, 24}, {120, 40}, {300, 80}, {41, 11}, {57, 13}, {20, 6}, {5, 3}} {
		w, h := size[0], size[1]
		out := layout.Draw(screen, layout.Constraints{MinW: w, MaxW: w, MinH: h, MaxH: h})
		lines := strings.Split(out, "\n")
		if len(lines) != h {
			t.Errorf("%dx%d: %d rows, want %d", w, h, len(lines), h)
			continue
		}
		for i, l := range lines {
			if got := ansi.Width(l); got != w {
				t.Errorf("%dx%d: row %d is %d wide", w, h, i, got)
				break
			}
		}
	}

	out := ansi.StripANSI(layout.Draw(screen, layout.Constraints{MinW: 120, MaxW: 120, MinH: 30, MaxH: 30}))
	for _, want := range []string{"row 1234", "event 499", "Section 0", "Go to:", "one more tag"} {
		if !strings.Contains(out, want) {
			t.Errorf("120x30 screen lacks %q", want)
		}
	}
}

// TestEveryRemainingWidgetNodeComposesAtEveryTerminalSize finishes the set:
// the last seventeen adapters, overlay boxes included, in one nested screen.
func TestEveryRemainingWidgetNodeComposesAtEveryTerminalSize(t *testing.T) {
	sh := appshell.New("Dashboard", 40, 5)
	sh.Content.SetContent(strings.Repeat("a line of content\n", 60))
	sh.Hints = []widgets.Hint{{Key: "q", Action: "quit"}, {Key: "?", Action: "help"}}

	cal := datepicker.New(time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC))
	pal := colorpicker.New(ansi.Red, ansi.Green, ansi.Blue, ansi.Yellow, ansi.Magenta)
	pal.HexInput.SetValue("12ab34")
	face := faces.New()
	face.Size = faces.Large
	steps := wizard.New("Account", "Profile", "Preferences", "Billing", "Review", "Confirm")
	_ = steps.Next(nil)
	_ = steps.Next(nil)

	tick := clockview.New(clockview.ModeStopwatch)
	spin := spinner.New()
	spin.Label = "Syncing"
	copyBtn := clipboard.New("text", "Copy link")

	ask := confirm.New("Discard all unsaved changes to this document and close it?")
	fail := errorretry.New("The server rejected the upload because the file is too large", 3)
	tool := toolapproval.New("deploy", "Push the build to every production server in all regions", toolapproval.RiskHigh)

	dlg := dialog.New("Delete", "This cannot be undone and removes the file from every synced device.")
	dlg.Show()
	note := toast.New("Saved")
	note.Show()
	side := drawer.New("Settings\nTheme: dark\nFont: 14")
	side.Show()
	tip := popover.New("Press ? for help", 0, 0)
	tip.Show()
	help := helpscreen.New(widgets.Hint{Key: "q", Action: "quit"}, widgets.Hint{Key: "?", Action: "toggle help"})
	help.Show()
	inbox := notificationcenter.New(5, 0)
	inbox.Push(notificationcenter.Notification{Message: "The nightly build finished and every test passed", Variant: widgets.VariantSuccess})

	screen := layout.Column(0,
		layout.FlexChild{Node: steps.LayoutNode(theme.DarkTheme())},
		layout.Fill(layout.Row(1,
			layout.FillWeight(sh.LayoutNode(), 2),
			layout.FillWeight(layout.Column(0,
				layout.Fill(cal.LayoutNode()),
				layout.FlexChild{Node: pal.LayoutNode()},
				layout.Fill(face.LayoutNode()),
				layout.Fill(layout.Row(1, layout.Fill(dlg.LayoutNode()), layout.Fill(side.LayoutNode()))),
			), 2),
			layout.FillWeight(layout.Column(0,
				layout.Fill(ask.LayoutNode()),
				layout.Fill(fail.LayoutNode()),
				layout.Fill(tool.LayoutNode()),
				layout.Fill(layout.Row(1, layout.Fill(note.LayoutNode()), layout.Fill(tip.LayoutNode()))),
				layout.Fill(layout.Row(1, layout.Fill(help.LayoutNode()), layout.Fill(inbox.LayoutNode()))),
			), 2),
		)),
		layout.FlexChild{Node: layout.Row(2, layout.FlexChild{Node: tick.LayoutNode()}, layout.FlexChild{Node: spin.LayoutNode()}, layout.FlexChild{Node: copyBtn.LayoutNode()})},
	)

	for _, size := range [][2]int{{40, 10}, {80, 24}, {120, 40}, {300, 80}, {41, 11}, {57, 13}, {20, 6}, {5, 3}} {
		w, h := size[0], size[1]
		out := layout.Draw(screen, layout.Constraints{MinW: w, MaxW: w, MinH: h, MaxH: h})
		lines := strings.Split(out, "\n")
		if len(lines) != h {
			t.Errorf("%dx%d: %d rows, want %d", w, h, len(lines), h)
			continue
		}
		for i, l := range lines {
			if got := ansi.Width(l); got != w {
				t.Errorf("%dx%d: row %d is %d wide", w, h, i, got)
				break
			}
		}
	}
	out := ansi.StripANSI(layout.Draw(screen, layout.Constraints{MinW: 200, MaxW: 200, MinH: 60, MaxH: 60}))
	for _, want := range []string{"Review", "Dashboard", "Su Mo Tu", "Syncing", "Copy link", "Delete", "Settings", "Saved", "Approve"} {
		if !strings.Contains(out, want) {
			t.Errorf("200x60 screen lacks %q", want)
		}
	}
}

// TestFormNodeComposesAtEveryTerminalSize puts form.Model.LayoutNode in a
// composed screen beside a table and a log: a tall form with errors showing, a
// secret holding a value, and focus on its last field, so at small sizes it is
// taller than its slot and its window has to follow the focus.
func TestFormNodeComposesAtEveryTerminalSize(t *testing.T) {
	const secret = "s3cr3t-value-9f2"
	fields := []form.Field{
		{Name: "name", Label: "Name", Validators: []form.Validator{form.Required()}},
		{Name: "email", Label: "Email", Value: "ada@example.com", Validators: []form.Validator{form.Email()}},
		{Name: "password", Label: "Password", Secret: true, Value: secret},
		{Name: "plan", Label: "Plan", Kind: form.FieldSelect, Options: []string{"Free", "Pro", "Team"}},
		{Name: "terms", Label: "Accept terms", Kind: form.FieldCheckbox, Validators: []form.Validator{func(v string) error {
			if v != "true" {
				return fmt.Errorf("you must accept the terms")
			}
			return nil
		}}},
	}
	for i := 1; i <= 7; i++ {
		fields = append(fields, form.Field{Name: fmt.Sprintf("extra%d", i), Label: fmt.Sprintf("Extra field %d", i)})
	}
	fm := form.New(fields...)
	fm.Focus()
	fm, _ = fm.Submit() // errors for name and terms are now showing
	if fm.Err("name") == "" || fm.Err("terms") == "" {
		t.Fatal("test premise changed: the form should show two errors")
	}
	for i := 0; i < len(fields)-1; i++ { // focus the last field
		fm, _ = fm.Update(tui.Key{Type: tui.KeyTab})
	}
	if fm.Current() != "extra7" {
		t.Fatalf("focus = %q, want the last field", fm.Current())
	}

	rows := make([][]string, 30)
	for i := range rows {
		rows[i] = []string{fmt.Sprintf("job-%02d", i), "running"}
	}
	jobs := datatable.New([]string{"Job", "State"}, rows)
	logs := applog.New(40, 5)
	for i := 0; i < 100; i++ {
		logs.Append(fmt.Sprintf("12:00:%02d event %d", i%60, i))
	}

	screen := layout.Column(0,
		layout.FlexChild{Node: layout.BoxNode(layout.NewBox().Border(layout.NormalBorder()), layout.Block("Accounts"))},
		layout.Fill(layout.Row(1,
			layout.FillWeight(layout.Column(0,
				layout.Fill(jobs.LayoutNode()),
				layout.Fill(logs.LayoutNode()),
			), 1),
			layout.FillWeight(fm.LayoutNode(), 1),
		)),
		layout.FlexChild{Node: layout.Block("q: quit")},
	)

	for _, size := range [][2]int{{40, 10}, {80, 24}, {120, 40}, {300, 80}, {41, 11}, {57, 13}, {20, 6}, {5, 3}} {
		w, h := size[0], size[1]
		out := layout.Draw(screen, layout.Constraints{MinW: w, MaxW: w, MinH: h, MaxH: h})
		lines := strings.Split(out, "\n")
		if len(lines) != h {
			t.Errorf("%dx%d: %d rows, want %d", w, h, len(lines), h)
			continue
		}
		for i, l := range lines {
			if got := ansi.Width(l); got != w {
				t.Errorf("%dx%d: row %d is %d wide", w, h, i, got)
				break
			}
		}
		if strings.Contains(ansi.StripANSI(out), secret) || strings.Contains(out, "s3cr3t") {
			t.Errorf("%dx%d: the composed screen shows the secret", w, h)
		}
	}

	// The form is drawn, not just constructed: its labels and errors are in the
	// composed render, so removing form.LayoutNode from the screen fails here.
	big := ansi.StripANSI(layout.Draw(screen, layout.Constraints{MinW: 120, MaxW: 120, MinH: 40, MaxH: 40}))
	for _, want := range []string{"Name:", "Email:", "Password:", "Plan: <", "[ ] Accept terms", "required", "you must accept the terms", "Extra field 7"} {
		if !strings.Contains(big, want) {
			t.Errorf("the 120x40 composed screen lacks %q:\n%s", want, big)
		}
	}
	// The password is masked, not absent.
	if !strings.Contains(big, "•") && !strings.Contains(big, "*") {
		t.Errorf("the masked password is not drawn:\n%s", big)
	}

	// Taller than its slot: at 40x10 the form gets a few rows, and the window
	// follows the focus to the last field.
	small := ansi.StripANSI(layout.Draw(screen, layout.Constraints{MinW: 40, MaxW: 40, MinH: 10, MaxH: 10}))
	if !strings.Contains(small, "Extra field 7") {
		t.Errorf("the focused last field is not visible at 40x10:\n%s", small)
	}
	if strings.Contains(small, "Name:") {
		t.Errorf("the form was not scrolled at 40x10 (its first field is still shown):\n%s", small)
	}
}
