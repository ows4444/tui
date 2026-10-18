package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"reflect"
	"strings"
	"unicode"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

// demoWidget is one live widget in the demo: a tui.Model that can be given and
// taken away the keyboard and a theme. live adapts every widget package's value
// type to it.
type demoWidget interface {
	tui.Model
	focus() (demoWidget, tui.Cmd)
	blur() demoWidget
	withTheme(theme.Theme) demoWidget
	// started runs the widget's start command, if it has one, on the value that
	// is kept (a widget such as the spinner must be started where it will live,
	// not on a copy), and returns that value and the command.
	started() (demoWidget, tui.Cmd)
}

// live adapts a widget whose Update returns its own type. The widget gains focus
// and a theme through the optional methods it already has (Focus, Blur and
// SetTheme); a widget without them is left as it is.
type live[M interface {
	Update(tui.Msg) (M, tui.Cmd)
	View() string
}] struct {
	m     M
	start func(*M) tui.Cmd // run once when the widget is shown, for ones that animate
}

func newLive[M interface {
	Update(tui.Msg) (M, tui.Cmd)
	View() string
}](m M) live[M] {
	return live[M]{m: m}
}

func (l live[M]) Init() tui.Cmd { return nil }

func (l live[M]) started() (demoWidget, tui.Cmd) {
	if l.start != nil {
		return l, l.start(&l.m)
	}
	return l, nil
}

func (l live[M]) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	m, cmd := l.m.Update(msg)
	l.m = m
	return l, cmd
}

func (l live[M]) View() string { return l.m.View() }

func (l live[M]) focus() (demoWidget, tui.Cmd) {
	if f, ok := any(&l.m).(interface{ Focus() tui.Cmd }); ok {
		return l, f.Focus()
	}
	return l, nil
}

func (l live[M]) blur() demoWidget {
	if b, ok := any(&l.m).(interface{ Blur() }); ok {
		b.Blur()
	}
	return l
}

func (l live[M]) withTheme(t theme.Theme) demoWidget {
	if s, ok := any(l.m).(interface{ SetTheme(theme.Theme) M }); ok {
		l.m = s.SetTheme(t)
	}
	return l
}

// entry is one widget in the catalog.
type entry struct {
	name, group, summary string
	code                 []string
	build                func() demoWidget
}

// demoApp is the catalog: a filterable list of widgets on the left and the
// selected one, live, on the right.
type demoApp struct {
	entries []entry
	shown   []int // indexes into entries that match the filter
	cursor  int   // position in shown
	top     int   // first row of the list on screen
	query   string

	live    demoWidget
	liveOf  int     // entries index the live widget was built for; -1 for none
	initCmd tui.Cmd // the start command of the widget shown first
	inDemo  bool
	last    string // the last message the widget sent

	themes   []string
	themeIdx int
	th       theme.Theme

	w, h int
}

// demoCaps is what the terminal said it can draw, for the entries that show an
// image where it can; the zero value is cells only.
var demoCaps tui.Capabilities

// demoMinW and demoMinH are the smallest terminal the catalog is laid out for.
const (
	demoMinW = 60
	demoMinH = 14
	// demoListW is the width of the widget list, borders included.
	demoListW = 26
)

func newDemoApp(entries []entry, start string) *demoApp {
	a := &demoApp{entries: entries, liveOf: -1, themes: theme.PresetNames()}
	for i, n := range a.themes {
		if n == "dark" {
			a.themeIdx = i
		}
	}
	a.th = a.themeAt(a.themeIdx)
	a.filter()
	if start != "" {
		for pos, i := range a.shown {
			if strings.EqualFold(entries[i].name, start) {
				a.cursor = pos
			}
		}
	}
	a.initCmd = a.show()
	return a
}

func (a *demoApp) themeAt(i int) theme.Theme {
	if len(a.themes) == 0 {
		return theme.DarkTheme()
	}
	t, ok := theme.Preset(a.themes[i%len(a.themes)])
	if !ok {
		return theme.DarkTheme()
	}
	return t
}

// filter recomputes shown from the query: entries whose name or group contains it.
func (a *demoApp) filter() {
	q := strings.ToLower(a.query)
	a.shown = a.shown[:0]
	for i, e := range a.entries {
		if q == "" || strings.Contains(strings.ToLower(e.name+" "+e.group), q) {
			a.shown = append(a.shown, i)
		}
	}
	a.cursor = max(0, min(a.cursor, len(a.shown)-1))
}

// show builds a fresh widget for the entry under the cursor when it changed.
func (a *demoApp) show() tui.Cmd {
	if len(a.shown) == 0 {
		a.live, a.liveOf = nil, -1
		return nil
	}
	i := a.shown[a.cursor]
	if i == a.liveOf {
		return nil
	}
	var cmd tui.Cmd
	a.live, cmd = a.entries[i].build().withTheme(a.th).started()
	a.liveOf, a.last = i, ""
	return cmd
}

func (a *demoApp) Init() tui.Cmd { return a.initCmd }

func (a *demoApp) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch m := msg.(type) {
	case tui.ResizeMsg:
		a.w, a.h = m.Width, m.Height
		return a, nil
	case tui.MouseEvent:
		return a.mouse(m)
	case tui.CapabilitiesMsg:
		demoCaps = m.Capabilities
		return a, nil
	case tui.Key:
		switch {
		case m.Type == tui.KeyCtrlC:
			return a, tui.Quit()
		case m.Type == tui.KeyCtrl && m.Code == 't':
			a.themeIdx++
			a.th = a.themeAt(a.themeIdx)
			if a.live != nil {
				a.live = a.live.withTheme(a.th)
			}
			return a, nil
		}
		if a.inDemo {
			return a.updateDemo(m)
		}
		return a.updateList(m)
	}
	// Anything else is for the widget (a blink, a tick, a message it sent
	// earlier): hand it on, and remember what the application would be told.
	if a.live == nil {
		return a, nil
	}
	if t := reflect.TypeOf(msg); t != nil && t.Name() != "" && unicode.IsUpper([]rune(t.Name())[0]) {
		a.last = fmt.Sprintf("%s %+v", t.Name(), msg)
	}
	var cmd tui.Cmd
	a.live, cmd = updateWidget(a.live, msg)
	return a, cmd
}

// mouse follows the pointer. In the list a click selects the row and the wheel
// moves the cursor; on the widget a click gives it the keys, and a widget that
// takes the mouse (placeable) is told where it is drawn and sent the event.
func (a *demoApp) mouse(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	if a.w < demoMinW || a.h < demoMinH {
		return a, nil
	}
	press := ev.Action == tui.MouseActionPress
	if ev.X < demoListW-1 {
		switch {
		case press && ev.Button == tui.MouseButtonWheelUp:
			a.cursor--
		case press && ev.Button == tui.MouseButtonWheelDown:
			a.cursor++
		case press && ev.Button == tui.MouseButtonLeft && ev.Y >= 2 && a.top+ev.Y-2 < len(a.shown):
			a.cursor = a.top + ev.Y - 2
			if a.inDemo {
				a.inDemo = false
				a.live = a.live.blur()
			}
		default:
			return a, nil
		}
		a.cursor = max(0, min(a.cursor, len(a.shown)-1))
		a.scroll()
		return a, a.show()
	}
	if a.live == nil {
		return a, nil
	}
	var focus tui.Cmd
	if press && ev.Button == tui.MouseButtonLeft && !a.inDemo {
		a.live, focus = a.live.focus()
		a.inDemo = true
	}
	p, ok := a.live.(placeable)
	if !ok {
		return a, focus
	}
	// The widget starts below the box border, the summary and a blank line.
	summary := strings.Count(ansi.Wrap(a.entries[a.liveOf].summary, max(a.w-demoListW-3, 10)), "\n") + 1
	a.live = p.place(demoListW, 1+summary+1)
	var cmd tui.Cmd
	a.live, cmd = updateWidget(a.live, ev)
	return a, tui.Batch(focus, cmd)
}

func updateWidget(w demoWidget, msg tui.Msg) (demoWidget, tui.Cmd) {
	next, cmd := w.Update(msg)
	if dw, ok := next.(demoWidget); ok {
		return dw, cmd
	}
	return w, cmd
}

func (a *demoApp) updateDemo(k tui.Key) (tui.Model, tui.Cmd) {
	if k.Type == tui.KeyEsc || k.Type == tui.KeyTab {
		a.inDemo = false
		a.live = a.live.blur()
		return a, nil
	}
	var cmd tui.Cmd
	a.live, cmd = updateWidget(a.live, k)
	return a, cmd
}

func (a *demoApp) updateList(k tui.Key) (tui.Model, tui.Cmd) {
	rows := max(a.listRows(), 1)
	switch {
	case k.Type == tui.KeyUp:
		a.cursor--
	case k.Type == tui.KeyDown:
		a.cursor++
	case k.Type == tui.KeyPgUp:
		a.cursor -= rows
	case k.Type == tui.KeyPgDown:
		a.cursor += rows
	case k.Type == tui.KeyHome:
		a.cursor = 0
	case k.Type == tui.KeyEnd:
		a.cursor = len(a.shown) - 1
	case k.Type == tui.KeyEnter, k.Type == tui.KeyTab, k.Type == tui.KeyRight:
		if a.live == nil {
			return a, nil
		}
		var cmd tui.Cmd
		a.live, cmd = a.live.focus()
		a.inDemo = true
		return a, cmd
	case k.Type == tui.KeyEsc:
		if a.query != "" {
			a.query = ""
			a.filter()
			break
		}
		return a, tui.Quit()
	case k.Type == tui.KeyBackspace:
		if r := []rune(a.query); len(r) > 0 {
			a.query = string(r[:len(r)-1])
			a.filter()
		}
	case k.Type == tui.KeyRunes && k.Mod == 0 && k.Text != "":
		a.query += k.Text
		a.cursor = 0
		a.filter()
	case k.Type == tui.KeySpace:
		a.query += " "
		a.filter()
	}
	a.cursor = max(0, min(a.cursor, len(a.shown)-1))
	a.scroll()
	return a, a.show()
}

// listRows is how many entries the list shows at once.
func (a *demoApp) listRows() int { return a.h - 4 } // the footer, the borders and the filter line

// scroll keeps the cursor inside the visible part of the list.
func (a *demoApp) scroll() {
	rows := max(a.listRows(), 1)
	if a.cursor < a.top {
		a.top = a.cursor
	}
	if a.cursor >= a.top+rows {
		a.top = a.cursor - rows + 1
	}
	a.top = max(0, min(a.top, max(len(a.shown)-rows, 0)))
}

var (
	dim   = ansi.NewStyle().Faint()
	bold  = ansi.NewStyle().Bold()
	reset = ansi.Reset
)

func (a *demoApp) View() string {
	if a.w < demoMinW || a.h < demoMinH {
		return ansi.Wrap(fmt.Sprintf("tui demo needs a terminal of at least %dx%d (this one is %dx%d).", demoMinW, demoMinH, a.w, a.h), max(a.w, 1))
	}
	a.scroll()
	root := layout.Column(0,
		layout.Fill(layout.MergedRow(
			layout.FlexChild{Node: a.listBox(), Basis: demoListW},
			layout.Fill(layout.MergedColumn(
				layout.FlexChild{Node: a.demoBox(), Basis: max(a.h-12, 6), Grow: 1},
				layout.FlexChild{Node: a.codeBox(), Basis: 7},
			)),
		)),
		layout.FlexChild{Node: layout.Block(a.footer()), Basis: 1},
	)
	return layout.Draw(root, layout.Tight(layout.Size{W: a.w, H: a.h}))
}

func (a *demoApp) box(title string, focused bool) layout.Box {
	b := layout.NewBox().Border(layout.RoundedBorder()).BorderColor(a.th.BorderColor).Title(title, layout.AlignStart)
	if focused {
		b = b.BorderColor(a.th.Focus)
	}
	return b
}

func (a *demoApp) listBox() layout.Node {
	rows := a.listRows()
	var lines []string
	sel := a.th.ResolvedStates().Selected
	q := a.query
	if q == "" {
		q = dim.Render("type to filter")
	}
	lines = append(lines, "/ "+q)
	end := min(a.top+max(rows, 1), len(a.shown))
	for pos := a.top; pos < end; pos++ {
		e := a.entries[a.shown[pos]]
		line := "  " + e.name
		if pos == a.cursor {
			line = sel.Render("> " + e.name)
		}
		lines = append(lines, line)
	}
	if len(a.shown) == 0 {
		lines = append(lines, dim.Render("  nothing matches"))
	}
	return layout.BoxNode(a.box(fmt.Sprintf("Widgets %d", len(a.entries)), !a.inDemo), layout.Block(strings.Join(lines, "\n")))
}

func (a *demoApp) demoBox() layout.Node {
	title, body := "Nothing selected", ""
	if a.live != nil {
		e := a.entries[a.liveOf]
		title = e.name
		body = bold.Render(ansi.Wrap(e.summary, max(a.w-demoListW-3, 10))) + "\n\n" + a.live.View()
		if a.last != "" {
			body += "\n\n" + dim.Render("sent: "+a.last)
		}
	}
	return layout.BoxNode(a.box(title, a.inDemo), layout.Block(body))
}

func (a *demoApp) codeBox() layout.Node {
	var code string
	if a.live != nil {
		code = strings.Join(a.entries[a.liveOf].code, "\n")
	}
	return layout.BoxNode(a.box("Code", false), layout.Block(dim.Render(strings.ReplaceAll(code, "\t", "    "))))
}

func (a *demoApp) footer() string {
	name := "dark"
	if len(a.themes) > 0 {
		name = a.themes[a.themeIdx%len(a.themes)]
	}
	keys := "up/down choose  type to filter  Enter try it  Ctrl+T theme (" + name + ")  Esc quit  Ctrl+C quit"
	if a.inDemo {
		keys = "the widget has the keys  Tab or Esc back to the list  Ctrl+T theme (" + name + ")  Ctrl+C quit"
	}
	return dim.Render(ansi.Truncate(keys, a.w))
}

// runDemo is `tui demo [-list] [-theme name] [widget]`.
func runDemo(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("demo", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	list := fs.Bool("list", false, "print the widgets and exit")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("%v\n%s", err, usage)
	}
	if fs.NArg() > 1 {
		return errors.New(usage)
	}
	entries := demoEntries()
	if *list {
		for _, e := range entries {
			fmt.Fprintf(out, "%-14s %-8s %s\n", e.name, e.group, e.summary)
		}
		return nil
	}
	start := fs.Arg(0)
	if start != "" {
		found := false
		for _, e := range entries {
			found = found || strings.EqualFold(e.name, start)
		}
		if !found {
			return fmt.Errorf("no widget called %q; tui demo -list shows them", start)
		}
	}
	_, err := tui.NewProgram(newDemoApp(entries, start), tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion), tui.WithCapabilityProbe(0)).Run()
	return err
}

// custom is a demo widget written for the catalog: a view drawn from a theme and
// a small state that keys change. It is for the things that are functions in the
// library (a progress bar, a plot, a layout) and have no Model of their own.
type custom struct {
	th    theme.Theme
	state int
	draw  func(th theme.Theme, state int) string
	press func(state int, k tui.Key) int
}

func (c custom) Init() tui.Cmd { return nil }

func (c custom) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	if k, ok := msg.(tui.Key); ok && c.press != nil {
		c.state = c.press(c.state, k)
	}
	return c, nil
}

func (c custom) View() string                   { return c.draw(c.th, c.state) }
func (c custom) focus() (demoWidget, tui.Cmd)   { return c, nil }
func (c custom) blur() demoWidget               { return c }
func (c custom) started() (demoWidget, tui.Cmd) { return c, nil }

func (c custom) withTheme(t theme.Theme) demoWidget {
	c.th = t
	return c
}

// stepWith turns Left and Right (and +, -) into a change of the state in [lo, hi].
func stepWith(lo, hi, by int) func(int, tui.Key) int {
	return func(s int, k tui.Key) int {
		switch {
		case k.Type == tui.KeyRight, k.Type == tui.KeyUp, k.Type == tui.KeyRunes && k.Text == "+":
			s += by
		case k.Type == tui.KeyLeft, k.Type == tui.KeyDown, k.Type == tui.KeyRunes && k.Text == "-":
			s -= by
		case k.Type == tui.KeyHome:
			s = lo
		case k.Type == tui.KeyEnd:
			s = hi
		}
		return max(lo, min(s, hi))
	}
}
