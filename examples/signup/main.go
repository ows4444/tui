// Command signup is a sign-up wizard in three steps: account, plan (and
// billing for a paid plan), then a review you confirm. It is built on package
// form with groups, so each step is checked before the next opens, and the
// billing step only exists when the plan needs it.
//
// Mouse: click a field to edit it, click a finished step in the bar at the top
// to go back to it, click Back, Next or Create account. Keyboard: Tab and
// Shift+Tab move between fields, the arrows and Space change a choice, Enter
// checks the step and moves on, PgUp or Esc goes back, F1 shows help, Ctrl+C
// quits. Nothing is sent anywhere: the review shows the password's length
// only, and no file is written.
//
//	go run ./examples/signup
package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/form"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

// The groups, in the order the form shows them.
const (
	account = iota
	plan
	billing
)

var groupTitles = [...]string{"Account", "Plan", "Billing"}

type phase int

const (
	editing phase = iota
	reviewing
	finished
)

type model struct {
	form   form.Model
	phase  phase
	values map[string]string // what the form submitted, for the review
	help   bool
	hover  string

	width, height int
}

// mustBeTicked rejects a checkbox that is not ticked: a checkbox's value is the
// string "true" or "false".
func mustBeTicked(v string) error {
	if v != "true" {
		return errors.New("you must accept the terms")
	}
	return nil
}

// needsBilling is false for the free plan, which skips the billing step.
func needsBilling(values map[string]string) bool { return values["plan"] != "Free" }

func initialModel() model {
	text := func(name, label, hint string, v ...form.Validator) form.Field {
		return form.Field{Name: name, Label: label, Placeholder: hint, Width: 20, Validators: v}
	}
	f := form.NewGroups(
		form.Group{Title: "Account", Description: "Who is this for?", Fields: []form.Field{
			text("name", "Name", "Ada Lovelace", form.Required()),
			text("email", "Email", "ada@example.com", form.Required(), form.Email()),
			{Name: "password", Label: "Password", Secret: true, Width: 20,
				Description: "At least 8 characters",
				Validators:  []form.Validator{form.Required(), form.MinLen(8)}},
		}},
		form.Group{Title: "Plan", Description: "You can change it later.", Fields: []form.Field{
			{Name: "plan", Label: "Plan", Kind: form.FieldSelect, Options: []string{"Free", "Pro", "Team"}},
			{Name: "news", Label: "Newsletter", Kind: form.FieldCheckbox, Value: "true"},
			{Name: "terms", Label: "Accept terms", Kind: form.FieldCheckbox,
				Validators: []form.Validator{mustBeTicked}},
		}},
		form.Group{Title: "Billing", Description: "Only for paid plans.", Hide: func(v map[string]string) bool { return !needsBilling(v) },
			Fields: []form.Field{
				text("company", "Company", "Analytical Engines Ltd", form.Required()),
				form.SelectOf("seats", "Seats",
					form.Option[int]{Label: "1", Value: 1}, form.Option[int]{Label: "5", Value: 5},
					form.Option[int]{Label: "10", Value: 10}, form.Option[int]{Label: "25", Value: 25}),
			}},
	)
	f.Mouse = true
	f.Focus()
	return model{form: f}
}

func (m model) Init() tui.Cmd {
	f := m.form
	return f.Focus()
}

// ---- steps -----------------------------------------------------------------

// visible returns the group indexes that apply to the values entered so far.
func (m model) visible() []int {
	v := m.form.Values()
	out := []int{account, plan}
	if needsBilling(v) {
		out = append(out, billing)
	}
	return out
}

// steps are the names in the bar: each applicable group, then the review.
func (m model) steps() []string {
	var s []string
	for _, g := range m.visible() {
		s = append(s, groupTitles[g])
	}
	return append(s, "Review")
}

// current is the index in steps of where the user is.
func (m model) current() int {
	switch m.phase {
	case reviewing, finished:
		return len(m.steps()) - 1
	}
	for i, g := range m.visible() {
		if g == m.form.CurrentGroup() {
			return i
		}
	}
	return 0
}

// goTo moves back to step i; it never skips forward past what was checked.
func (m model) goTo(i int) model {
	cur := m.current()
	if m.phase == finished || i >= cur || i < 0 {
		return m
	}
	at := cur // the step the form itself is on
	if m.phase == reviewing {
		at-- // the review is not a group: the form waits on the last one
	}
	m.phase = editing
	for ; at > i; at-- {
		next, _ := m.form.Update(tui.Key{Type: tui.KeyPgUp})
		m.form = next
	}
	return m
}

// ---- update ----------------------------------------------------------------

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case form.SubmittedMsg:
		m.phase, m.values, m.help = reviewing, msg.Values, false
		return m, nil
	case tui.MouseEvent:
		return m.mouse(msg)
	case tui.Key:
		return m.key(msg)
	}
	return m, nil
}

func (m model) key(k tui.Key) (tui.Model, tui.Cmd) {
	switch {
	case k.Type == tui.KeyCtrlC:
		return m, tui.Quit()
	case k.Type == tui.KeyF1:
		m.help = !m.help
		return m, nil
	case m.help:
		if k.Type == tui.KeyEsc || k.Type == tui.KeyEnter {
			m.help = false
		}
		return m, nil
	}
	switch m.phase {
	case finished:
		if k.Type == tui.KeyEnter || k.Type == tui.KeyEsc || (k.Type == tui.KeyRunes && k.Text == "q") {
			return m, tui.Quit()
		}
		return m, nil
	case reviewing:
		switch k.Type {
		case tui.KeyEnter:
			m.phase = finished
		case tui.KeyEsc, tui.KeyPgUp:
			return m.goTo(m.current() - 1), nil
		}
		return m, nil
	}
	if k.Type == tui.KeyEsc {
		return m.goTo(m.current() - 1), nil
	}
	next, cmd := m.form.Update(k)
	m.form = next
	return m, cmd
}

func (m model) rect(id string) (layout.Rect, bool) {
	return layout.RectOf(m.screen(), m.size(), id)
}

func (m model) hit(ev tui.MouseEvent) string {
	ids := []string{"back", "next", "form"}
	for i := range m.steps() {
		ids = append(ids, "step-"+strconv.Itoa(i))
	}
	for _, id := range ids {
		if r, ok := m.rect(id); ok && ev.X >= r.X && ev.X < r.X+r.W && ev.Y >= r.Y && ev.Y < r.Y+r.H {
			return id
		}
	}
	return ""
}

func (m model) mouse(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	if m.help || m.phase == finished {
		return m, nil
	}
	id := m.hit(ev)
	if ev.Action == tui.MouseActionMotion {
		m.hover = id
		return m, nil
	}
	if ev.Button != tui.MouseButtonLeft || ev.Action != tui.MouseActionPress {
		if id == "form" && m.phase == editing {
			return m.toForm(ev)
		}
		return m, nil
	}
	switch {
	case strings.HasPrefix(id, "step-"):
		n, _ := strconv.Atoi(strings.TrimPrefix(id, "step-"))
		return m.goTo(n), nil
	case id == "back":
		return m.goTo(m.current() - 1), nil
	case id == "next":
		return m.key(tui.Key{Type: tui.KeyEnter})
	case id == "form" && m.phase == editing:
		return m.toForm(ev)
	}
	return m, nil
}

// toForm passes a mouse event to the form, telling it where it is drawn.
func (m model) toForm(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	if r, ok := m.rect("form"); ok {
		m.form.Bounds = r
	}
	next, cmd := m.form.Update(ev)
	m.form = next
	return m, cmd
}

// ---- view ------------------------------------------------------------------

func (m model) size() layout.Size {
	if m.width == 0 {
		return layout.Size{W: 80, H: 24}
	}
	return layout.Size{W: m.width, H: m.height}
}

// inner is the panel's content width.
func (m model) inner() int { return max(20, min(60, m.size().W-4)) }

// compact is a terminal too short for the airy layout.
func (m model) compact() bool { return m.size().H < 18 }

func (m model) stepBar(th theme.Theme) layout.Node {
	var kids []layout.FlexChild
	cur := m.current()
	for i, name := range m.steps() {
		id := "step-" + strconv.Itoa(i)
		label := fmt.Sprintf("%d %s", i+1, name)
		st := ansi.NewStyle().Faint()
		switch {
		case i < cur:
			label = "✓ " + label // done: a mark, so colour is not the only signal
			st = ansi.NewStyle().Foreground(th.Success)
			if m.hover == id {
				st = st.Underline()
			}
		case i == cur:
			st = ansi.NewStyle().Bold().Foreground(th.Primary)
		}
		kids = append(kids, layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render(label)))})
	}
	return layout.Row(2, kids...)
}

func (m model) button(id, label string, on bool, th theme.Theme) layout.FlexChild {
	st := ansi.NewStyle().Foreground(th.Muted)
	switch {
	case !on:
		st = ansi.NewStyle().Faint()
	case id == "next":
		st = ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Primary)
	case m.hover == id:
		st = ansi.NewStyle().Foreground(th.Primary).Underline()
	}
	node := layout.Block(st.Render(" " + label + " "))
	if on {
		node = layout.Named(id, node)
	}
	return layout.FlexChild{Node: node}
}

func (m model) buttons(th theme.Theme) layout.Node {
	next := "Next ›"
	switch m.phase {
	case reviewing:
		next = "Create account"
	case editing:
		if m.current() == len(m.steps())-2 {
			next = "Review ›"
		}
	}
	return layout.Row(2, m.button("back", "‹ Back", m.current() > 0, th), m.button("next", next, true, th))
}

func (m model) review(th theme.Theme) layout.Node {
	v := m.values
	row := func(k, val string) string {
		return ansi.NewStyle().Faint().Render(fmt.Sprintf("%-10s", k)) + val
	}
	lines := []string{
		row("Name", v["name"]),
		row("Email", v["email"]),
		row("Password", fmt.Sprintf("%d characters, not shown", len([]rune(v["password"])))),
		row("Plan", v["plan"]),
	}
	if needsBilling(v) {
		seats, _ := form.Value[int](m.form, "seats")
		lines = append(lines, row("Company", v["company"]), row("Seats", strconv.Itoa(seats)))
	}
	news := "no"
	if v["news"] == "true" {
		news = "yes"
	}
	lines = append(lines, row("Newsletter", news))
	var kids []layout.FlexChild
	for _, l := range lines {
		kids = append(kids, layout.FlexChild{Node: layout.Block(ansi.Truncate(l, m.inner()))})
	}
	return layout.Column(0, kids...)
}

func (m model) helpPanel(th theme.Theme) layout.Node {
	key := func(k, a string) layout.FlexChild {
		return layout.FlexChild{Node: layout.Block(ansi.Truncate(ansi.NewStyle().Bold().Foreground(th.Primary).Render(fmt.Sprintf("%-14s", k))+a, m.inner()))}
	}
	return layout.Column(0,
		key("Tab", "next field"), key("Shift+Tab", "previous field"),
		key("← → Space", "change a choice"), key("Enter", "check the step, go on"),
		key("PgUp, Esc", "go back a step"), key("click a ✓ step", "go back to it"),
		key("F1", "close this help"), key("Ctrl+C", "quit"))
}

func (m model) hints() layout.Node {
	f := widgets.Footer(m.inner(),
		widgets.Hint{Key: "tab", Action: "next field"}, widgets.Hint{Key: "enter", Action: "continue"},
		widgets.Hint{Key: "pgup", Action: "back"}, widgets.Hint{Key: "f1", Action: "help"},
		widgets.Hint{Key: "ctrl+c", Action: "quit"})
	return layout.Block(ansi.NewStyle().Faint().Render(f))
}

func (m model) screen() layout.Node {
	th := theme.DarkTheme()
	gap := 1
	if m.compact() {
		gap = 0
	}
	title := layout.Block(ansi.NewStyle().Bold().Foreground(th.Primary).Render("Create your account"))
	var body []layout.FlexChild
	switch {
	case m.phase == finished:
		name := m.values["name"]
		body = []layout.FlexChild{
			{Node: layout.Block(ansi.NewStyle().Bold().Foreground(th.Success).Render("✓ Account created"))},
			{Node: layout.Block(ansi.Truncate("Welcome, "+name+". Nothing was sent anywhere.", m.inner()))},
			{Node: m.review(th)},
			{Node: layout.Block(ansi.NewStyle().Faint().Render("Press Enter to quit."))},
		}
	case m.help:
		body = []layout.FlexChild{{Node: m.helpPanel(th)}}
	default:
		mid := layout.Named("form", m.form.LayoutNode())
		if m.phase == reviewing {
			mid = m.review(th)
		}
		body = []layout.FlexChild{{Node: m.stepBar(th)}, {Node: mid}, {Node: m.buttons(th)}, {Node: m.hints()}}
	}
	if !m.compact() {
		body = append([]layout.FlexChild{{Node: title}}, body...)
	}
	col := layout.Fixed(layout.Column(gap, body...), layout.Size{W: m.inner()})
	if m.compact() {
		return layout.Stack(layout.Anchor(layout.Center, 0, 0, col))
	}
	box := layout.NewBox().Border(th.Border).Padding(1, 1, 1, 1)
	return layout.Stack(layout.Anchor(layout.Center, 0, 0, layout.BoxNode(box, col)))
}

func (m model) View() string { return layout.DrawTight(m.screen(), m.size()) }

func main() {
	p := tui.NewProgram(initialModel(), tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion))
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
