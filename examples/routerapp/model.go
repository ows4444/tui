package main

import (
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/examples/internal/ui"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/motion"
)

type kind int

const (
	rHome kind = iota
	rCustomers
	rCustomer
	rOrders
	rOrder
	rSettings
)

var sections = []struct {
	name string
	icon string
	kind kind
}{{"Home", "⌂", rHome}, {"Customers", "☺", rCustomers}, {"Orders", "▤", rOrders}, {"Settings", "⚙", rSettings}}

// route is one screen on the stack: what it shows, and where its selection is.
type route struct {
	kind kind
	id   int
	cur  int
}

// row is one selectable line of a screen: it opens a screen, jumps to a section
// or flips a setting.
type row struct {
	text   string
	to     *route
	reset  bool // jump to a section, clearing the stack, instead of pushing
	toggle int  // setting to flip, or -1
}

type toast struct {
	text string
	ttl  int
}

type tickMsg struct{}

const (
	toastTicks = 5 // ticks a notice stays
	tickEvery  = 500 * time.Millisecond
	chromeRows = 4 // title and crumbs, rule, status, hints
)

type model struct {
	th      ui.Theming // the theme the Program chose; see SetTheme
	stack   []route
	side    bool // the sidebar has the keys
	sideCur int
	starred map[int]bool
	shipped map[int]bool
	compact bool // sidebar shows icons only
	notify  bool
	toasts  []toast
	ticking bool
	hover   string
	help    bool

	width, height int
}

func newModel() model {
	return model{stack: []route{{kind: rHome}}, starred: map[int]bool{2: true}, shipped: map[int]bool{1001: true, 1002: true}, notify: true}
}

func (m model) Init() tui.Cmd { return nil }

func (m model) size() layout.Size {
	if m.width == 0 {
		return layout.Size{W: 100, H: 30}
	}
	return layout.Size{W: m.width, H: m.height}
}

func (m model) top() route { return m.stack[len(m.stack)-1] }

func (m model) withTop(r route) model {
	st := append([]route(nil), m.stack...)
	st[len(st)-1] = r
	m.stack = st
	return m
}

func (m model) push(r route) model {
	m.stack = append(append([]route(nil), m.stack...), r)
	return m
}

// back pops one screen, or says there is nowhere to go.
func (m model) back() (model, tui.Cmd) {
	if len(m.stack) == 1 {
		return m.notice("Already at the top")
	}
	m.stack = append([]route(nil), m.stack[:len(m.stack)-1]...)
	return m, nil
}

// popTo keeps the first n screens.
func (m model) popTo(n int) model {
	if n >= 1 && n < len(m.stack) {
		m.stack = append([]route(nil), m.stack[:n]...)
	}
	return m
}

func (m model) goSection(k kind) model {
	m.stack = []route{{kind: rHome}}
	if k != rHome {
		m.stack = append(m.stack, route{kind: k})
	}
	for i, s := range sections {
		if s.kind == k {
			m.sideCur = i
		}
	}
	return m
}

// notice shows a message for a few seconds, starting the clock if it is not
// already running.
func (m model) notice(text string) (model, tui.Cmd) {
	if !m.notify {
		return m, nil
	}
	m.toasts = append(append([]toast(nil), m.toasts...), toast{text, toastTicks})
	if len(m.toasts) > 3 {
		m.toasts = m.toasts[len(m.toasts)-3:]
	}
	if m.ticking {
		return m, nil
	}
	m.ticking = true
	return m, m.tick()
}

func (m model) tick() tui.Cmd {
	return tui.FromCtx(motion.After(tickEvery, func(time.Time) tui.Msg { return tickMsg{} }))
}

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tickMsg:
		var left []toast
		for _, t := range m.toasts {
			if t.ttl--; t.ttl > 0 {
				left = append(left, t)
			}
		}
		m.toasts = left
		if len(left) == 0 {
			m.ticking = false
			return m, nil
		}
		return m, m.tick()
	case tui.Key:
		return m.key(msg)
	case tui.MouseEvent:
		return m.mouse(msg)
	}
	return m, nil
}

// ---- screens ---------------------------------------------------------------

func (m model) crumb(r route) string {
	switch r.kind {
	case rCustomers:
		return "Customers"
	case rCustomer:
		return customerByID(r.id).name
	case rOrders:
		return "Orders"
	case rOrder:
		return "#" + itoa(r.id)
	case rSettings:
		return "Settings"
	}
	return "Home"
}

func (m model) rows(r route) []row {
	var out []row
	switch r.kind {
	case rHome:
		for _, s := range sections[1:] {
			out = append(out, row{text: s.icon + " " + s.name, to: &route{kind: s.kind}, reset: true, toggle: -1})
		}
	case rCustomers:
		for _, c := range customers {
			star := "  "
			if m.starred[c.id] {
				star = "★ "
			}
			out = append(out, row{text: star + padTo(c.name, 20) + padTo(c.city, 14) + itoa(len(ordersOf(c.id))) + " orders", to: &route{kind: rCustomer, id: c.id}, toggle: -1})
		}
	case rCustomer:
		for _, o := range ordersOf(r.id) {
			out = append(out, m.orderRow(o, false))
		}
	case rOrders:
		for _, o := range orders {
			out = append(out, m.orderRow(o, true))
		}
	case rSettings:
		out = append(out, row{text: check(m.compact) + " Compact sidebar", toggle: 0}, row{text: check(m.notify) + " Show notices", toggle: 1})
	}
	return out
}

func (m model) orderRow(o order, withCustomer bool) row {
	state := "new"
	if m.shipped[o.id] {
		state = "shipped"
	}
	who := ""
	if withCustomer {
		who = padTo(customerByID(o.cust).name, 20)
	}
	return row{text: "#" + itoa(o.id) + "  " + who + padTo(o.item, 12) + padTo(money(o.cents), 9) + state, to: &route{kind: rOrder, id: o.id}, toggle: -1}
}

func check(on bool) string {
	if on {
		return "[x]"
	}
	return "[ ]"
}
