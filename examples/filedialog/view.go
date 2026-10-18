package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

func pad(s string, w int) string {
	if n := ansi.Width(s); n < w {
		return s + strings.Repeat(" ", w-n)
	}
	return ansi.Truncate(s, w)
}

func (m model) pathBar(th theme.Theme) layout.Node {
	names, _ := m.crumbs()
	var kids []layout.FlexChild
	for i, n := range names {
		id := "crumb-" + strconv.Itoa(i)
		st := ansi.NewStyle().Foreground(th.Muted)
		switch {
		case i == len(names)-1:
			st = ansi.NewStyle().Bold()
		case m.hover == id:
			st = ansi.NewStyle().Foreground(th.Primary).Underline()
		}
		if i > 0 {
			kids = append(kids, layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render("›"))})
		}
		kids = append(kids, layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render(ansi.Sanitize(n))))})
	}
	return layout.Fixed(layout.Row(1, kids...), layout.Size{W: m.size().W, H: 1})
}

func (m model) filterBar(th theme.Theme) layout.Node {
	kids := []layout.FlexChild{{Node: layout.Block(ansi.NewStyle().Faint().Render("Show "))}}
	for i, f := range filters {
		id := "filter-" + strconv.Itoa(i)
		label, st := " "+f.name+" ", ansi.NewStyle().Foreground(th.Muted)
		switch {
		case i == m.filter:
			label, st = "▸"+f.name+" ", ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Primary)
		case m.hover == id:
			st = ansi.NewStyle().Foreground(th.Primary).Underline()
		}
		kids = append(kids, layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render(label)))})
	}
	box := "[ ]"
	if m.showHidden {
		box = "[x]"
	}
	st := ansi.NewStyle()
	if m.hover == "hidden" {
		st = ansi.NewStyle().Foreground(th.Primary).Underline()
	}
	kids = append(kids, layout.FlexChild{Node: layout.Block(""), Grow: 1}, layout.FlexChild{Node: layout.Named("hidden", layout.Block(st.Render(box+" Show hidden (.)")))})
	return layout.Fixed(layout.Row(0, kids...), layout.Size{W: m.size().W, H: 1})
}

func (m model) listWidth() int {
	if m.size().W >= 70 {
		return m.size().W - 32
	}
	return m.size().W
}

func (m model) list(th theme.Theme, h int) []layout.FlexChild {
	vis := m.visible()
	w := m.listWidth()
	entries := m.fp.Entries()
	cur := -1
	for k, i := range vis {
		if i == m.fp.Cursor() {
			cur = k
		}
	}
	top := max(0, cur-(h-1))
	if cur >= 0 && cur < h {
		top = 0
	}
	var rows []layout.FlexChild
	if err := m.fp.Err(); err != nil {
		return []layout.FlexChild{{Node: layout.Block(ansi.NewStyle().Foreground(th.Error).Render(" Cannot read this folder: " + ansi.Sanitize(err.Error())))}}
	}
	if len(vis) == 0 {
		rows = append(rows, layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render(" Nothing here" + m.emptyHint()))})
	}
	for k := top; k < len(vis) && len(rows) < h; k++ {
		i := vis[k]
		e := entries[i]
		id := "row-" + strconv.Itoa(i)
		label := ansi.Sanitize(e.Name)
		icon := "  "
		if e.IsDir {
			label, icon = label+"/", "▸ "
		}
		st := ansi.NewStyle()
		switch {
		case i == m.fp.Cursor() && !m.onName:
			st = ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Primary)
			icon = "> "
		case i == m.fp.Cursor():
			st = ansi.NewStyle().Bold().Foreground(th.Primary)
			icon = "> "
		case hidden(e.Name):
			st = ansi.NewStyle().Faint()
		case m.hover == id:
			st = ansi.NewStyle().Foreground(th.Primary)
		}
		rows = append(rows, layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render(pad(" "+icon+label, w))))})
	}
	return rows
}

func (m model) emptyHint() string {
	if m.filter > 0 {
		return " with the " + filters[m.filter].name + " filter; F3 changes it"
	}
	return ""
}

// info describes the entry under the cursor.
func (m model) info(th theme.Theme) []string {
	name, isDir, ok := m.current()
	if !ok {
		return []string{ansi.NewStyle().Faint().Render("Nothing selected")}
	}
	out := []string{ansi.NewStyle().Bold().Render(ansi.Sanitize(name))}
	fi, err := os.Stat(m.path(name))
	if err != nil {
		return append(out, ansi.NewStyle().Foreground(th.Error).Render(ansi.Sanitize(err.Error())))
	}
	kind := "File"
	if isDir {
		kind = "Folder"
	}
	out = append(out, kind, "")
	if !isDir {
		out = append(out, "Size      "+size(fi.Size()))
	}
	return append(out, "Modified  "+m.when(fi.ModTime()), "Mode      "+fi.Mode().String())
}

// when formats a modification time in the dialog's time zone.
func (m model) when(t time.Time) string {
	if m.loc != nil {
		t = t.In(m.loc)
	}
	return t.Format(time.DateTime)
}

func size(n int64) string {
	switch {
	case n < 1024:
		return fmt.Sprintf("%d B", n)
	case n < 1<<20:
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
}

func (m model) button(id, label string, th theme.Theme, primary bool) layout.FlexChild {
	st := ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Muted)
	if primary {
		st = ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Success)
	}
	if m.hover == id {
		st = st.Underline()
	}
	return layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render(" "+label+" ")))}
}

func (m model) footerRow(th theme.Theme) layout.Node {
	label := "Open"
	var kids []layout.FlexChild
	if m.save {
		label = "Save"
		field := string(m.name)
		st := ansi.NewStyle().Underline()
		if m.onName {
			field += "▏"
			st = ansi.NewStyle().Bold().Foreground(th.Primary).Underline()
		}
		if field == "" {
			field = " "
		}
		kids = append(kids, layout.FlexChild{Node: layout.Block("Name ")}, layout.FlexChild{Node: layout.Named("name", layout.Block(st.Render(pad(field, 28))))}, layout.FlexChild{Node: layout.Block("  ")})
	}
	kids = append(kids, layout.FlexChild{Node: layout.Block(""), Grow: 1}, m.button("btn-ok", "Enter "+label, th, true), layout.FlexChild{Node: layout.Block(" ")}, m.button("btn-cancel", "Esc Cancel", th, false))
	return layout.Fixed(layout.Row(0, kids...), layout.Size{W: m.size().W, H: 1})
}

func (m model) helpLines(th theme.Theme) []string {
	key := func(k, a string) string {
		return " " + ansi.NewStyle().Bold().Foreground(th.Primary).Render(pad(k, 16)) + a
	}
	return []string{
		key("arrows", "move; Right or Enter opens a folder"), key("Left  Backspace", "up one folder"), key("letters", "jump to the next name starting with them"),
		key("Enter", "choose the selected file"), key(".", "show or hide dotfiles"), key("F3", "next filter"), key("F2", "Open or Save As"),
		key("Tab", "Save As: list and name field"), key("Esc", "cancel"), key("click twice", "open the row"), key("?", "close this help"),
	}
}

func (m model) status() string {
	switch {
	case m.msg != "":
		return m.msg
	case m.confirm != "":
		return "Replace " + ansi.Sanitize(m.confirm) + "?  y to replace, n to go back"
	}
	n := len(m.visible())
	word := "entries"
	if n == 1 {
		word = "entry"
	}
	return ansi.NewStyle().Faint().Render(fmt.Sprintf("%d %s  ·  %s", n, word, filters[m.filter].name))
}

func (m model) screen() layout.Node {
	th := theme.DarkTheme()
	s := m.size()
	h := m.rows()
	title := "Open file"
	if m.save {
		title = "Save file as"
	}
	hints := widgets.Footer(s.W, widgets.Hint{Key: "↑↓", Action: "move"}, widgets.Hint{Key: "enter", Action: "open"}, widgets.Hint{Key: "←", Action: "up"},
		widgets.Hint{Key: ".", Action: "hidden"}, widgets.Hint{Key: "f3", Action: "filter"}, widgets.Hint{Key: "f2", Action: "mode"}, widgets.Hint{Key: "esc", Action: "cancel"})
	var body layout.Node
	switch {
	case m.help:
		var lines []layout.FlexChild
		for _, l := range m.helpLines(th) {
			lines = append(lines, layout.FlexChild{Node: layout.Block(ansi.Truncate(l, s.W))})
		}
		body = layout.Fixed(layout.Column(0, lines...), layout.Size{W: s.W, H: h})
	default:
		rows := m.list(th, h)
		for len(rows) < h {
			rows = append(rows, layout.FlexChild{Node: layout.Block("")})
		}
		list := layout.Fixed(layout.Column(0, rows[:h]...), layout.Size{W: m.listWidth(), H: h})
		body = list
		if s.W >= 70 {
			var side []layout.FlexChild
			for _, l := range m.info(th) {
				side = append(side, layout.FlexChild{Node: layout.Block(" " + ansi.Truncate(l, 30))})
			}
			sep := make([]layout.FlexChild, h)
			for i := range sep {
				sep[i] = layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render("│"))}
			}
			body = layout.Row(0, layout.FlexChild{Node: list}, layout.FlexChild{Node: layout.Fixed(layout.Column(0, sep...), layout.Size{W: 1, H: h})},
				layout.FlexChild{Node: layout.Fixed(layout.Column(0, side...), layout.Size{W: 31, H: h})})
		}
	}
	return layout.Column(0,
		layout.FlexChild{Node: layout.Block(ansi.NewStyle().Bold().Foreground(th.Primary).Render(title))},
		layout.FlexChild{Node: m.pathBar(th)},
		layout.FlexChild{Node: m.filterBar(th)},
		layout.FlexChild{Node: body},
		layout.FlexChild{Node: layout.Block(ansi.Truncate(m.status(), s.W))},
		layout.FlexChild{Node: m.footerRow(th)},
		layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render(ansi.Truncate(hints, s.W)))})
}

func (m model) View() string { return layout.DrawTight(m.screen(), m.size()) }
