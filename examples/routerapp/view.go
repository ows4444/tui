package main

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/avatar"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

func fit(s string, w int) string {
	s = ansi.Truncate(s, w)
	if n := ansi.Width(s); n < w {
		s += strings.Repeat(" ", w-n)
	}
	return s
}

func (m model) compactSide() bool { return m.compact || m.size().W < 60 }

func (m model) sideW() int {
	if m.compactSide() {
		return 5
	}
	return 16
}

func (m model) titleBar(th theme.Theme) layout.Node {
	back := ansi.NewStyle().Faint()
	if len(m.stack) > 1 {
		back = ansi.NewStyle().Foreground(th.Primary)
		if m.hover == "back" {
			back = back.Underline()
		}
	}
	kids := []layout.FlexChild{{Node: layout.Named("back", layout.Block(back.Render("‹ Back")))}}
	for i, r := range m.stack {
		id := "crumb-" + itoa(i)
		st := ansi.NewStyle().Foreground(th.Muted)
		switch {
		case i == len(m.stack)-1:
			st = ansi.NewStyle().Bold()
		case m.hover == id:
			st = ansi.NewStyle().Foreground(th.Primary).Underline()
		}
		if i > 0 {
			kids = append(kids, layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render("›"))})
		}
		kids = append(kids, layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render(m.crumb(r))))})
	}
	return layout.Fixed(layout.Row(1, kids...), layout.Size{W: m.size().W, H: 1})
}

func (m model) sidebar(th theme.Theme, h int) layout.Node {
	w := m.sideW()
	var rows []layout.FlexChild
	for i, s := range sections {
		id := "side-" + itoa(i)
		label := " " + s.icon + " " + s.name
		if m.compactSide() {
			label = " " + s.icon
		}
		st := ansi.NewStyle()
		switch {
		case i == sectionOf(m.stack[min(1, len(m.stack)-1)].kind) && !(m.side && i == m.sideCur):
			st = ansi.NewStyle().Bold().Foreground(th.Primary)
			label = "▎" + label[1:]
		case m.side && i == m.sideCur:
			st = ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Primary)
		case m.hover == id:
			st = ansi.NewStyle().Underline()
		}
		rows = append(rows, layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render(fit(label, w))))})
	}
	for len(rows) < h {
		rows = append(rows, layout.FlexChild{Node: layout.Block(strings.Repeat(" ", w))})
	}
	return layout.Fixed(layout.Column(0, rows[:h]...), layout.Size{W: w, H: h})
}

// sectionOf is the sidebar index of the section that owns kind k.
func sectionOf(k kind) int {
	switch k {
	case rCustomers, rCustomer:
		return 1
	case rOrders, rOrder:
		return 2
	case rSettings:
		return 3
	}
	return 0
}

// faceW and faceH are the cells of a customer's avatar.
const faceW, faceH = 8, 4

// face is the avatar of the customer on route r: drawn from the name, so the
// same customer always looks the same, and smiling once starred. It sits beside
// the name, so a screen reader is not told the name twice.
func (m model) face(r route, th theme.Theme) string {
	if r.kind != rCustomer {
		return ""
	}
	return m.avatarOf(r).SetTheme(th).View()
}

// avatarOf is the kept avatar if it is the customer's, else a fresh one placed
// where the screen draws it.
func (m model) avatarOf(r route) avatar.Model {
	name := customerByID(r.id).name
	a := m.faceModel
	if a.Name != name {
		a = avatar.New(name)
		a.Width, a.Height, a.Decorative = faceW, faceH, true
	}
	a.X, a.Y = m.sideW()+2, 2 // beside the sidebar and its rule, below the title and its rule
	if m.starred[r.id] && a.Expression == avatar.ExpressionNone {
		a.Expression = avatar.ExpressionHappy
	} else if !m.starred[r.id] {
		a.Expression = avatar.ExpressionNone
	}
	return a
}

// head is the lines above the rows of a screen.
func (m model) head(r route, th theme.Theme) []string {
	bold := func(s string) string { return ansi.NewStyle().Bold().Render(s) }
	faint := func(s string) string { return ansi.NewStyle().Faint().Render(s) }
	switch r.kind {
	case rHome:
		return []string{bold("Welcome back"), faint(itoa(len(customers)) + " customers, " + itoa(len(orders)) + " orders, " + itoa(len(m.shipped)) + " shipped"), "", faint("Go to")}
	case rCustomers:
		return []string{bold("Customers") + faint("  Enter opens a customer, s stars one"), ""}
	case rCustomer:
		c := customerByID(r.id)
		star := "not starred"
		if m.starred[r.id] {
			star = "★ starred"
		}
		return []string{bold(c.name), faint(c.city + "  ·  " + strings.ToLower(strings.Fields(c.name)[0]) + "@example.com  ·  " + star), "", faint("Orders")}
	case rOrders:
		return []string{bold("Orders") + faint("  Enter opens an order"), ""}
	case rSettings:
		return []string{bold("Settings"), ""}
	}
	o := orderByID(r.id)
	state := "waiting to ship"
	if m.shipped[o.id] {
		state = "shipped"
	}
	return []string{bold("Order #" + itoa(o.id)), "", "Item      " + o.item, "Customer  " + customerByID(o.cust).name, "Total     " + money(o.cents), "Status    " + state, "", faint("m ships it, Esc goes back")}
}

func (m model) content(th theme.Theme, w, h int) layout.Node {
	r := m.top()
	head := m.head(r, th)
	rows := m.rows(r)
	var kids []layout.FlexChild
	if face := m.face(r, th); r.kind == rCustomer && face != "" && w >= 30 && h >= len(head)+2 {
		// The avatar stands beside the four lines that name the customer.
		text := make([]layout.FlexChild, len(head))
		for i, l := range head {
			text[i] = layout.FlexChild{Node: layout.Block(fit(l, w-faceW-2))}
		}
		kids = append(kids, layout.FlexChild{Node: layout.Fixed(layout.Row(0,
			layout.FlexChild{Node: layout.Block(" ")},
			layout.FlexChild{Node: layout.Fixed(layout.Block(face), layout.Size{W: faceW, H: faceH})},
			layout.FlexChild{Node: layout.Block(" ")},
			layout.FlexChild{Node: layout.Column(0, text...)}), layout.Size{W: w, H: len(head)})})
	} else {
		for _, l := range head {
			kids = append(kids, layout.FlexChild{Node: layout.Block(fit(" "+l, w))})
		}
	}
	vis := max(0, h-len(head))
	off := max(0, r.cur-(vis-1))
	for i := off; i < len(rows) && i < off+vis; i++ {
		id := "row-" + itoa(i)
		st := ansi.NewStyle()
		text := "  " + rows[i].text
		switch {
		case i == r.cur && !m.side:
			st, text = ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Primary), "▸ "+rows[i].text
		case i == r.cur:
			text = "▸ " + rows[i].text
		case m.hover == id:
			st = ansi.NewStyle().Foreground(th.Primary)
		}
		kids = append(kids, layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render(fit(text, w))))})
	}
	for len(kids) < h {
		kids = append(kids, layout.FlexChild{Node: layout.Block(strings.Repeat(" ", w))})
	}
	return layout.Fixed(layout.Column(0, kids[:h]...), layout.Size{W: w, H: h})
}

func (m model) toastColumn(th theme.Theme) layout.Node {
	var kids []layout.FlexChild
	for i, t := range m.toasts {
		st := ansi.NewStyle().Foreground(th.TextInverse).Background(th.Success)
		kids = append(kids, layout.FlexChild{Node: layout.Named("toast-"+itoa(i), layout.Block(st.Render(" ✓ "+t.text+"  ×")))})
	}
	return layout.Column(0, kids...)
}

func (m model) helpLines(th theme.Theme) []string {
	key := func(k, a string) string {
		return " " + ansi.NewStyle().Bold().Foreground(th.Primary).Render(fit(k, 16)) + a
	}
	return []string{
		key("Tab", "move between the sidebar and the screen"), key("↑ ↓  j k", "move the selection"), key("Enter", "open the item, or go to the section"),
		key("Esc  Backspace", "go back one screen"), key("s", "star the open customer"), key("m", "ship the open order"),
		key("click a crumb", "jump back to it"), key("?  Esc", "close this help"), key("q  Ctrl+C", "quit"),
	}
}

func (m model) status() string {
	if len(m.stack) == 1 {
		return ansi.NewStyle().Faint().Render(m.crumb(m.top()))
	}
	return ansi.NewStyle().Faint().Render("Depth " + itoa(len(m.stack)) + "  ·  Esc goes back to " + m.crumb(m.stack[len(m.stack)-2]))
}

func (m model) screen() layout.Node {
	th := m.theme()
	s := m.size()
	bodyH := max(1, s.H-chromeRows)
	sw := m.sideW()
	hints := widgets.Footer(s.W, widgets.Hint{Key: "tab", Action: "sidebar"}, widgets.Hint{Key: "enter", Action: "open"}, widgets.Hint{Key: "esc", Action: "back"},
		widgets.Hint{Key: "s", Action: "star"}, widgets.Hint{Key: "m", Action: "ship"}, widgets.Hint{Key: "?", Action: "help"}, widgets.Hint{Key: "q", Action: "quit"})
	var body layout.Node
	if m.help {
		var lines []layout.FlexChild
		for _, l := range m.helpLines(th) {
			lines = append(lines, layout.FlexChild{Node: layout.Block(ansi.Truncate(l, s.W))})
		}
		body = layout.Column(0, lines...)
	} else {
		sep := make([]layout.FlexChild, bodyH)
		for i := range sep {
			sep[i] = layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render("│"))}
		}
		body = layout.Row(0,
			layout.FlexChild{Node: m.sidebar(th, bodyH)},
			layout.FlexChild{Node: layout.Fixed(layout.Column(0, sep...), layout.Size{W: 1, H: bodyH})},
			layout.FlexChild{Node: m.content(th, s.W-sw-1, bodyH)})
		if len(m.toasts) > 0 {
			body = layout.Stack(body, layout.Anchor(layout.TopRight, 1, 0, m.toastColumn(th)))
		}
	}
	return layout.Column(0,
		layout.FlexChild{Node: m.titleBar(th)},
		layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render(strings.Repeat("─", s.W)))},
		layout.FlexChild{Node: layout.Fixed(body, layout.Size{W: s.W, H: bodyH})},
		layout.FlexChild{Node: layout.Block(ansi.Truncate(m.status(), s.W))},
		layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render(ansi.Truncate(hints, s.W)))})
}

func (m model) View() string { return layout.DrawTight(m.screen(), m.size()) }
