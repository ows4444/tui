// Command gallery walks through seven pages, each showing one of the newer
// components: otpinput, inputgroup, transferlist, carousel, widgets.Timeline,
// widgets.Empty, and the three overlays (tooltip, hovercard, and a dialog
// over a backdrop). The gallery's own navigation is two more: a breadcrumb
// on the first row and a pagination row under the page.
//
// Tab moves between the page and the pager, PgUp and PgDn change page from
// anywhere, and the mouse clicks, scrolls and hovers. Esc quits.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/backdrop"
	"github.com/ows4444/tui/breadcrumb"
	"github.com/ows4444/tui/button"
	"github.com/ows4444/tui/carousel"
	"github.com/ows4444/tui/dialog"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/hovercard"
	"github.com/ows4444/tui/inputgroup"
	"github.com/ows4444/tui/otpinput"
	"github.com/ows4444/tui/pagination"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/tooltip"
	"github.com/ows4444/tui/transferlist"
	"github.com/ows4444/tui/widgets"
)

// The pages, counted from 1 as the pager counts them.
const (
	pageCode = iota + 1
	pageAddress
	pageColumns
	pageTips
	pageHistory
	pageEmpty
	pageOverlays
	pageCount = pageOverlays
)

var names = [pageCount + 1]string{"", "Code", "Address", "Columns", "Tips", "History", "Empty", "Overlays"}

// The two things on the overlays page that the pointer or the keys can be
// on.
const (
	onDelete = iota
	onName
)

// nameText is the name the hover card describes, with the cell either side
// that takes angle brackets when the keys are on it.
const nameText = "@ada"

type model struct {
	crumb breadcrumb.Model
	pager pagination.Model

	code otpinput.Model
	site inputgroup.Model
	cols transferlist.Model
	tips carousel.Model

	del  button.Model
	tip  tooltip.Model
	card hovercard.Model
	dim  backdrop.Model
	ask  dialog.Model

	onPager       bool   // the keys go to the pager, not to the page
	target        int    // on the overlays page: onDelete or onName
	note          string // the status line
	width, height int    // terminal size from the last ResizeMsg; 0 until known
}

func initialModel() model {
	m := model{
		crumb: breadcrumb.New("Gallery", names[pageCode]),
		pager: pagination.New(pageCount),
		code:  otpinput.New(6),
		site:  inputgroup.New("https://", ".com"),
		cols:  transferlist.New("Name", "Size", "Modified", "Owner", "Mode", "Links"),
		tips: carousel.New(
			"Press ? for help\nfrom any screen.",
			"Press / to search\nthe current view.",
			"Press q to quit\nwithout saving.",
		),
		del:  button.New("Delete"),
		tip:  tooltip.New("Asks before it deletes", hittest.Rect{}),
		card: hovercard.New("Ada Lovelace", "Wrote the first program,\nfor a machine never built.", hittest.Rect{}),
		dim:  backdrop.New(),
		ask:  dialog.New("Delete the file?", "Enter or Esc closes this."),
		note: "Type six digits, or paste them.",
	}
	m.pager.Siblings = pageCount // seven pages: room to show every number
	m.code.Group = 3
	m.site.Width = 16
	m.cols.LeftTitle, m.cols.RightTitle = "Hidden", "Shown"
	m.cols.ColumnWidth, m.cols.Height = 16, 4
	m.tips.Loop = true
	m.del.Variant = button.VariantDestructive
	m.dim.Hide()
	m.ask.Hide()
	for _, c := range []*bool{&m.crumb.Mouse, &m.pager.Mouse, &m.code.Mouse, &m.site.Mouse, &m.cols.Mouse, &m.tips.Mouse,
		&m.del.Mouse, &m.tip.Mouse, &m.card.Mouse, &m.ask.Mouse} {
		*c = true
	}
	m.setFocus()
	return m
}

// Init has nothing to start: the first page's field has no cursor that
// blinks. The address field's blink starts when its page is reached.
func (m model) Init() tui.Cmd { return nil }

func (m model) page() int { return m.pager.Page() }

// hasBody reports whether the page holds something that takes keys.
func (m model) hasBody() bool { return m.page() != pageHistory && m.page() != pageEmpty }

// setFocus blurs everything and gives focus to the pager or to what the page
// holds. The address field's Focus returns the Cmd that blinks its cursor.
func (m *model) setFocus() tui.Cmd {
	m.pager.Blur()
	m.code.Blur()
	m.site.Blur()
	m.cols.Blur()
	m.tips.Blur()
	m.del.Blur()
	m.tip.Hide()
	if !m.hasBody() {
		m.onPager = true
	}
	if m.onPager {
		return m.pager.Focus()
	}
	switch m.page() {
	case pageCode:
		return m.code.Focus()
	case pageAddress:
		return m.site.Focus()
	case pageColumns:
		return m.cols.Focus()
	case pageTips:
		return m.tips.Focus()
	case pageOverlays:
		if m.target == onDelete {
			m.tip.Show() // a hint for the keys, as the pointer gets one
			return m.del.Focus()
		}
	}
	return nil
}

// goTo shows page p: the pager, the breadcrumb and the status line follow.
func (m *model) goTo(p int) tui.Cmd {
	m.pager.SetPage(p)
	m.crumb.Items = []string{"Gallery", names[m.page()]}
	m.card.Hide()
	m.note = [pageCount + 1]string{
		pageCode:     "Type six digits, or paste them.",
		pageAddress:  "Type a name; the ends stay put.",
		pageColumns:  "Enter moves a column across.",
		pageTips:     "Left and Right change the tip.",
		pageHistory:  "A timeline only draws.",
		pageEmpty:    "What a view with nothing shows.",
		pageOverlays: "Hover, or use Left, Right and Enter.",
	}[m.page()]
	return m.setFocus()
}

// contentWidth is the width the gallery draws in: 60 columns, or the
// terminal's when that is narrower.
func (m model) contentWidth() int {
	if m.width > 0 {
		return max(30, min(60, m.width))
	}
	return 60
}

// compact reports whether the terminal is too short for blank rows between
// the sections.
func (m model) compact() bool { return m.height > 0 && m.height < 15 }

// bodyRows is the number of rows a page draws in; every page takes the same,
// so the pager under it does not move.
func (m model) bodyRows() int {
	if m.compact() {
		return 6
	}
	return 8
}

// bodyY, pagerY and rows are the screen rows of the page and of the pager,
// and the number of rows in all.
func (m model) bodyY() int {
	if m.compact() {
		return 1
	}
	return 2
}

func (m model) pagerY() int {
	if m.compact() {
		return m.bodyY() + m.bodyRows() + 1
	}
	return m.bodyY() + m.bodyRows() + 2
}

func (m model) rows() int {
	if m.compact() {
		return m.pagerY() + 2
	}
	return m.pagerY() + 3
}

// nameRect is where the name on the overlays page is drawn: after the
// button, three blanks and "by ".
func (m model) nameRect() hittest.Rect {
	return hittest.Rect{X: 2 + m.del.Width() + 6, Y: m.bodyY(), W: ansi.Width(nameText) + 2, H: 1}
}

// place tells every component where it is drawn, so each can read the
// pointer. It runs before a mouse event is handed on, since the layout
// follows the terminal's size.
func (m *model) place() {
	y := m.bodyY()
	whole := hittest.Rect{W: m.contentWidth(), H: m.rows()}
	m.crumb.Bounds = hittest.Rect{W: m.crumb.Width(), H: 1}
	m.pager.Bounds = hittest.Rect{Y: m.pagerY(), W: m.pager.Width(), H: 1}
	m.code.Bounds = hittest.Rect{X: 6, Y: y, W: m.code.Width(), H: 1}
	m.site.Bounds = hittest.Rect{X: ansi.Width(m.site.Prefix), Y: y, W: m.site.Width, H: 1}
	m.cols.Bounds = hittest.Rect{Y: y, W: m.cols.Width(), H: m.cols.Rows()}
	tw, th := m.tips.Size()
	m.tips.Bounds = hittest.Rect{Y: y, W: tw, H: th}
	m.del.Bounds = hittest.Rect{X: 2, Y: y, W: m.del.Width(), H: 1}
	m.tip.Target, m.tip.Bounds = m.del.Bounds, whole
	m.card.Target, m.card.Bounds = m.nameRect(), whole
	m.dim.Bounds, m.ask.Bounds = whole, whole
}

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.place()
		return m, nil
	case tui.Key:
		return m.updateKey(msg)
	case tui.MouseEvent:
		return m.updateMouse(msg)
	case pagination.ChangedMsg:
		return m, m.goTo(msg.Page)
	case breadcrumb.ChosenMsg:
		if msg.Index == 0 {
			return m, m.goTo(pageCode)
		}
		return m, nil
	case otpinput.ChangedMsg:
		m.note = "Code so far: " + msg.Value
		if msg.Complete {
			m.note = "Complete: " + msg.Value
		}
		return m, nil
	case transferlist.ChangedMsg:
		to := "hidden"
		if msg.To == transferlist.Right {
			to = "shown"
		}
		m.note = strings.Join(msg.Moved, ", ") + " is now " + to + "."
		return m, nil
	case carousel.ChangedMsg:
		m.note = fmt.Sprintf("Tip %d of %d.", msg.Index+1, len(m.tips.Slides))
		return m, nil
	case button.PressedMsg:
		m.tip.Hide()
		m.dim.Show()
		m.ask.Show()
		return m, nil
	case dialog.DismissedMsg:
		m.dim.Hide()
		m.note = "Nothing was deleted."
		return m, nil
	case hovercard.ClosedMsg:
		return m, nil
	}
	// Anything else is a tick for the address field's cursor.
	var cmd tui.Cmd
	m.site, cmd = m.site.Update(msg)
	return m, cmd
}

func (m model) updateKey(k tui.Key) (tui.Model, tui.Cmd) {
	var cmd tui.Cmd
	if m.ask.Open() { // a dialog takes every key until it closes
		m.ask, cmd = m.ask.Update(k)
		return m, cmd
	}
	switch k.String() {
	case "ctrl+c":
		return m, tui.Quit()
	case "esc":
		if m.card.Open() {
			m.card, cmd = m.card.Update(k)
			return m, cmd
		}
		return m, tui.Quit()
	case "tab", "shift+tab":
		m.onPager = !m.onPager
		return m, m.setFocus()
	case "pgdown":
		return m, m.goTo(min(m.page()+1, pageCount))
	case "pgup":
		return m, m.goTo(max(m.page()-1, 1))
	}
	if m.onPager {
		m.pager, cmd = m.pager.Update(k)
		return m, cmd
	}
	switch m.page() {
	case pageCode:
		m.code, cmd = m.code.Update(k)
	case pageAddress:
		m.site, cmd = m.site.Update(k)
		m.note = "Address: " + m.site.FullValue()
	case pageColumns:
		m.cols, cmd = m.cols.Update(k)
	case pageTips:
		m.tips, cmd = m.tips.Update(k)
	case pageOverlays:
		switch {
		case k.String() == "left" || k.String() == "right":
			m.target = onName - m.target
			m.card.Hide()
			return m, m.setFocus()
		case m.target == onDelete:
			m.del, cmd = m.del.Update(k)
		case k.String() == "enter" || k.String() == "space":
			m.card.Show()
		}
	}
	return m, cmd
}

// updateMouse moves focus to the part a press lands on and then gives the
// event to the navigation and to what the page holds: each ignores what is
// outside its own rectangle.
func (m model) updateMouse(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	m.place()
	var cmds [5]tui.Cmd
	if m.ask.Open() { // nothing behind a dialog is reached
		m.ask, cmds[0] = m.ask.Update(ev)
		return m, cmds[0]
	}
	if ev.Action == tui.MouseActionPress && ev.Button == tui.MouseButtonLeft {
		inBody := ev.Y >= m.bodyY() && ev.Y < m.bodyY()+m.bodyRows()
		if onPager := ev.Y == m.pagerY(); (onPager || inBody) && onPager != m.onPager {
			m.onPager = onPager
			cmds[0] = m.setFocus()
		}
	}
	m.crumb, cmds[1] = m.crumb.Update(ev)
	m.pager, cmds[2] = m.pager.Update(ev)
	switch m.page() {
	case pageCode:
		m.code, cmds[3] = m.code.Update(ev)
	case pageAddress:
		m.site, cmds[3] = m.site.Update(ev)
	case pageColumns:
		m.cols, cmds[3] = m.cols.Update(ev)
	case pageTips:
		m.tips, cmds[3] = m.tips.Update(ev)
	case pageOverlays:
		m.del, cmds[3] = m.del.Update(ev)
		m.tip, _ = m.tip.Update(ev)
		m.card, cmds[4] = m.card.Update(ev)
	}
	return m, tui.Batch(cmds[:]...)
}

var (
	muted = ansi.NewStyle().Faint()
	label = ansi.NewStyle().Bold()
)

var history = []widgets.TimelineItem{
	{Time: "09:40", Title: "Build passed", Detail: "139 packages", Status: widgets.TimelineDone},
	{Time: "10:02", Title: "Deploying", Status: widgets.TimelineCurrent},
	{Time: "10:05", Title: "Tests failed", Detail: "two on windows", Status: widgets.TimelineFailed},
	{Title: "Verify", Status: widgets.TimelinePending},
}

// body is what the current page draws, before it is fitted to its rows.
func (m model) body() string {
	th := theme.DarkTheme()
	w := m.contentWidth()
	switch m.page() {
	case pageCode:
		return label.Render("Code") + "  " + m.code.View()
	case pageAddress:
		return m.site.View()
	case pageColumns:
		return m.cols.View()
	case pageTips:
		return m.tips.View()
	case pageHistory:
		return widgets.Timeline(history, th, w)
	case pageEmpty:
		return widgets.Empty("No results", "Nothing matches the filter.", "Press / to search again", th, w)
	}
	name := " " + nameText + " "
	if !m.onPager && m.target == onName {
		name = label.Render("<" + nameText + ">")
	}
	// Nothing under the two: that is where the hint and the card are drawn.
	return "  " + m.del.View() + "   by" + name
}

// fit cuts s to n rows of the content width and pads it to both, so the
// frame is a rectangle that an overlay can be placed on.
func (m model) fit(s string, n int) []string {
	w := m.contentWidth()
	lines := strings.Split(s, "\n")
	out := make([]string, n)
	for i := range out {
		if i < len(lines) {
			out[i] = ansi.Truncate(lines[i], w)
		}
		out[i] += strings.Repeat(" ", w-ansi.Width(out[i]))
	}
	return out
}

// hints is the key legend: with words when they fit, else the keys alone.
func (m model) hints() string {
	hs := []widgets.Hint{{Key: "tab", Action: "page or pager"}, {Key: "pgup/pgdn", Action: "change page"}, {Key: "esc", Action: "quit"}}
	full := widgets.KeyHints("  ", hs...)
	if ansi.Width(full) <= m.contentWidth() {
		return full
	}
	for i := range hs {
		hs[i].Action = ""
	}
	return widgets.KeyHints(" ", hs...)
}

func (m model) View() string {
	m.place() // on the copy View has: the overlays follow the size
	var lines []string
	add := func(s string, n int) { lines = append(lines, m.fit(s, n)...) }
	blank := func() {
		if !m.compact() {
			add("", 1)
		}
	}
	add(m.crumb.View(), 1)
	blank()
	add(m.body(), m.bodyRows())
	blank()
	add(muted.Render(m.note), 1)
	add(m.pager.View(), 1)
	blank()
	add(m.hints(), 1)
	frame := strings.Join(lines, "\n")
	if m.page() == pageOverlays {
		frame = m.card.Render(m.tip.Render(frame))
	}
	return m.ask.Render(m.dim.Render(frame))
}

func main() {
	p := tui.NewProgram(initialModel(), tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion))
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
