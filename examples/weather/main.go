// Command weather shows a seven-day forecast for a city you search for. The
// work happens in the background (tui.Tasks): a search as you type, then the
// forecast, with a spinner while it loads, a message and a Retry when it fails,
// and Esc to give up on a slow one. By default the answers come from a small
// built-in table, after a short wait, so it works offline; three places in it
// misbehave on purpose: Reykjavik fails the first time, Atlantis does not
// exist, and Slowville never answers (cancel it with Esc).
//
// Mouse: click a search result, a recent city, a day, Retry, or the units.
// Keyboard: type to search; Up, Down and Enter choose a result; Tab swaps
// between the search box and the days; Left and Right pick a day; r reloads or
// retries; u switches °C and °F; 1 to 5 reopen a recent city; Esc cancels a
// load; F1 shows help; Ctrl+C quits (q too, outside the search box).
//
//	go run ./examples/weather             # built-in fixture, offline
//	go run ./examples/weather -live       # real forecasts from open-meteo.com
//	go run ./examples/weather Oslo        # open a city at start
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/examples/internal/ui"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/spinner"
	"github.com/ows4444/tui/textinput"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

type state int

const (
	idle state = iota
	loading
	failed
	loaded
)

const (
	chrome     = 5  // title, search, recents, status, hints
	cardW      = 13 // a day card
	cardH      = 7
	maxRecents = 5
)

// searchedMsg and loadedMsg are what the background work comes back with.
type searchedMsg struct {
	q   string
	res []place
	err error
}

type loadedMsg struct {
	p   place
	fc  forecast
	err error
}

type model struct {
	th       ui.Theming // the theme the Program chose; see SetTheme
	prov     provider
	tasks    *tui.Tasks
	spin     spinner.Model
	debounce time.Duration
	now      func() time.Time

	in      textinput.Model
	typing  bool // the search box has the keys
	results []place
	resCur  int
	showRes bool

	state   state
	target  place
	started time.Time
	err     string
	fc      forecast
	have    bool // fc holds a forecast
	sel     int
	recents []place
	units   units

	startCity string
	msg       string
	help      bool
	hover     string

	width, height int
}

func newModel(p provider, debounce time.Duration, city string) model {
	in := textinput.New()
	in.Prompt = "Search ▸ "
	in.Placeholder = "type a city"
	in.Width = 30
	in.Focus()
	return model{prov: p, tasks: tui.NewTasks(), spin: calm(spinner.New()), debounce: debounce, now: time.Now, in: in, typing: true, startCity: city}
}

func (m model) Init() tui.Cmd {
	if m.startCity == "" {
		return m.in.Focus()
	}
	q := m.startCity
	return m.tasks.Start("search", func(ctx context.Context) tui.Msg {
		res, err := m.prov.search(ctx, q)
		return searchedMsg{q: q, res: res, err: err}
	})
}

// ---- background work -------------------------------------------------------

func (m model) searchCmd() tui.Cmd {
	q := strings.TrimSpace(m.in.Value())
	if q == "" {
		m.tasks.Cancel("search")
		return nil
	}
	wait, prov := m.debounce, m.prov
	return m.tasks.Start("search", func(ctx context.Context) tui.Msg {
		if wait > 0 { // do not ask the service on every keystroke
			select {
			case <-time.After(wait):
			case <-ctx.Done():
				return nil
			}
		}
		res, err := prov.search(ctx, q)
		return searchedMsg{q: q, res: res, err: err}
	})
}

func (m model) load(p place) (model, tui.Cmd) {
	m.state, m.target, m.started, m.err, m.msg = loading, p, m.now(), "", ""
	m.showRes = false
	prov := m.prov
	return m, tui.Batch(m.spin.Start(), m.tasks.Start("forecast", func(ctx context.Context) tui.Msg {
		fc, err := prov.forecast(ctx, p)
		return loadedMsg{p: p, fc: fc, err: err}
	}))
}

func (m model) onLoaded(l loadedMsg) model {
	if m.state != loading || l.p != m.target {
		return m // an answer for a load that is no longer the one wanted
	}
	m.spin.Stop()
	if l.err != nil {
		m.state, m.err = failed, sentence(l.err)
		return m
	}
	m.state, m.fc, m.have, m.sel = loaded, l.fc, true, 0
	rec := []place{l.p}
	for _, r := range m.recents {
		if r != l.p && len(rec) < maxRecents {
			rec = append(rec, r)
		}
	}
	m.recents = rec
	return m
}

func (m model) onSearched(s searchedMsg) (model, tui.Cmd) {
	if s.q != strings.TrimSpace(m.in.Value()) && m.startCity == "" {
		return m, nil // typed on since
	}
	if s.err != nil {
		m.msg = "Search failed: " + sentence(s.err)
		return m, nil
	}
	if m.startCity != "" { // the city given on the command line: open the first match
		m.startCity = ""
		if len(s.res) == 0 {
			m.msg = "No city called " + s.q
			return m, nil
		}
		return m.load(s.res[0])
	}
	m.results, m.resCur, m.showRes = s.res, 0, true
	return m, nil
}

func (m model) cancel() model {
	if m.state != loading {
		return m
	}
	m.tasks.Cancel("forecast")
	m.spin.Stop()
	m.msg = "Cancelled loading " + m.target.Name
	m.state = idle
	if m.have {
		m.state = loaded
	}
	return m
}

func sentence(err error) string {
	s := err.Error()
	return strings.ToUpper(s[:1]) + s[1:]
}

// ---- update ----------------------------------------------------------------

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tui.TaskDoneMsg:
		if msg.Canceled || msg.Msg == nil {
			return m, nil
		}
		return m.Update(msg.Msg)
	case searchedMsg:
		return m.onSearched(msg)
	case loadedMsg:
		return m.onLoaded(msg), nil
	case tui.Key:
		return m.key(msg)
	case tui.MouseEvent:
		return m.mouse(msg)
	default:
		if m.state == loading {
			var cmd tui.Cmd
			m.spin, cmd = m.spin.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m model) key(k tui.Key) (tui.Model, tui.Cmd) {
	if k.Type == tui.KeyCtrlC {
		return m, tui.Quit()
	}
	if k.Type == tui.KeyF1 {
		m.help = !m.help
		return m, nil
	}
	if m.help {
		m.help = k.Type != tui.KeyEsc && k.Type != tui.KeyEnter
		return m, nil
	}
	if k.Type == tui.KeyEsc {
		return m.escape(), nil
	}
	if k.Type == tui.KeyTab {
		m.typing = !m.typing
		if m.typing {
			return m, m.in.Focus()
		}
		m.in.Blur()
		return m, nil
	}
	if m.typing {
		return m.searchKey(k)
	}
	return m.dayKey(k)
}

func (m model) escape() model {
	switch {
	case m.state == loading:
		return m.cancel()
	case m.showRes:
		m.showRes = false
	}
	return m
}

func (m model) searchKey(k tui.Key) (tui.Model, tui.Cmd) {
	switch k.Type {
	case tui.KeyUp:
		m.resCur = max(0, m.resCur-1)
		return m, nil
	case tui.KeyDown:
		m.resCur = min(len(m.results)-1, m.resCur+1)
		return m, nil
	case tui.KeyEnter:
		if m.showRes && m.resCur < len(m.results) {
			p := m.results[m.resCur]
			m.in.Reset()
			return m.load(p)
		}
		return m, nil
	}
	before := m.in.Value()
	var cmd tui.Cmd
	m.in, cmd = m.in.Update(k)
	if m.in.Value() != before {
		m.msg = ""
		if m.in.Value() == "" {
			m.showRes, m.results = false, nil
		}
		return m, tui.Batch(cmd, m.searchCmd())
	}
	return m, cmd
}

func (m model) dayKey(k tui.Key) (tui.Model, tui.Cmd) {
	switch k.Type {
	case tui.KeyLeft:
		m.sel = max(0, m.sel-1)
	case tui.KeyRight:
		m.sel = min(len(m.fc.Days)-1, m.sel+1)
	case tui.KeyRunes:
		return m.dayRune(k.Text)
	}
	return m, nil
}

func (m model) dayRune(r string) (tui.Model, tui.Cmd) {
	switch r {
	case "q":
		return m, tui.Quit()
	case "?":
		m.help = true
	case "/":
		m.typing = true
		return m, m.in.Focus()
	case "u":
		m.units = 1 - m.units
	case "r":
		return m.reload()
	default:
		if n, err := strconv.Atoi(r); err == nil && n >= 1 && n <= len(m.recents) {
			return m.load(m.recents[n-1])
		}
	}
	return m, nil
}

func (m model) reload() (tui.Model, tui.Cmd) {
	if m.target == (place{}) {
		m.msg = "Nothing to reload yet"
		return m, nil
	}
	return m.load(m.target)
}

// ---- mouse -----------------------------------------------------------------

func inside(r layout.Rect, x, y int) bool { return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H }

func (m model) ids() []string {
	ids := []string{"input", "retry", "units"}
	for i := range m.results {
		ids = append(ids, "res-"+strconv.Itoa(i))
	}
	for i := range m.recents {
		ids = append(ids, "rec-"+strconv.Itoa(i))
	}
	for i := range m.fc.Days {
		ids = append(ids, "day-"+strconv.Itoa(i))
	}
	return ids
}

func (m model) hit(x, y int) string {
	for _, id := range m.ids() {
		if r, ok := layout.RectOf(m.screen(), m.size(), id); ok && inside(r, x, y) {
			return id
		}
	}
	return ""
}

func (m model) mouse(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	if m.help {
		return m, nil
	}
	id := m.hit(ev.X, ev.Y)
	if ev.Action == tui.MouseActionMotion {
		m.hover = id
		return m, nil
	}
	if ev.Button != tui.MouseButtonLeft || ev.Action != tui.MouseActionPress {
		return m, nil
	}
	n := 0
	if i := strings.LastIndexByte(id, '-'); i >= 0 {
		n, _ = strconv.Atoi(id[i+1:])
	}
	switch {
	case id == "input":
		m.typing = true
		return m, m.in.Focus()
	case id == "retry":
		return m.reload()
	case id == "units":
		m.units = 1 - m.units
	case strings.HasPrefix(id, "res-"):
		p := m.results[n]
		m.in.Reset()
		return m.load(p)
	case strings.HasPrefix(id, "rec-"):
		return m.load(m.recents[n])
	case strings.HasPrefix(id, "day-"):
		m.sel, m.typing = n, false
		m.in.Blur()
	}
	return m, nil
}

// ---- view ------------------------------------------------------------------

func (m model) size() layout.Size {
	if m.width == 0 {
		return layout.Size{W: 100, H: 28}
	}
	return layout.Size{W: m.width, H: m.height}
}

func (m model) bodyH() int { return max(1, m.size().H-chrome) }

func pad(s string, w int) string {
	if n := ansi.Width(s); n < w {
		return s + strings.Repeat(" ", w-n)
	}
	return s
}

func block(s string) layout.FlexChild { return layout.FlexChild{Node: layout.Block(s)} }

func (m model) button(id, label string, th theme.Theme) layout.Node {
	st := ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Primary)
	if m.hover == id {
		st = st.Underline()
	}
	return layout.Named(id, layout.Block(st.Render(" "+label+" ")))
}

func (m model) recentsNode(th theme.Theme) layout.Node {
	w := m.size().W
	kids := []layout.FlexChild{block(ansi.NewStyle().Faint().Render("Recent"))}
	if len(m.recents) == 0 {
		kids = append(kids, block(ansi.NewStyle().Faint().Render("none yet")))
	}
	for i, p := range m.recents {
		st := ansi.NewStyle().Foreground(th.Primary)
		id := "rec-" + strconv.Itoa(i)
		if m.hover == id {
			st = st.Underline()
		}
		kids = append(kids, layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render(fmt.Sprintf("%d %s", i+1, p.Name))))})
	}
	unit := "°C"
	if m.units == imperial {
		unit = "°F"
	}
	ust := ansi.NewStyle().Foreground(th.Muted)
	if m.hover == "units" {
		ust = ust.Underline()
	}
	kids = append(kids, layout.FlexChild{Node: layout.Block(""), Grow: 1}, layout.FlexChild{Node: layout.Named("units", layout.Block(ust.Render("["+unit+"]")))})
	return layout.Fixed(layout.Row(2, kids...), layout.Size{W: w, H: 1})
}

func (m model) resultsNode(th theme.Theme) layout.Node {
	w := m.size().W
	var kids []layout.FlexChild
	if len(m.results) == 0 {
		kids = append(kids, block(ansi.NewStyle().Faint().Render(" No city matches “"+strings.TrimSpace(m.in.Value())+"”.")))
	}
	for i, p := range m.results {
		line := "  " + p.String()
		st := ansi.NewStyle()
		if i == m.resCur {
			line, st = "› "+p.String(), ansi.NewStyle().Bold().Background(th.Selection)
		}
		kids = append(kids, layout.FlexChild{Node: layout.Named("res-"+strconv.Itoa(i), layout.Block(st.Render(pad(ansi.Truncate(line, w), w))))})
	}
	return layout.Column(0, kids...)
}

func (m model) stateNode(th theme.Theme) layout.Node {
	w := m.size().W
	switch m.state {
	case loading:
		secs := int(m.now().Sub(m.started).Seconds())
		lines := []layout.FlexChild{
			{Node: layout.Row(1, layout.FlexChild{Node: m.spin.LayoutNode()}, block(fmt.Sprintf("Loading the forecast for %s… %ds", m.target.String(), secs)))},
		}
		if secs >= 3 {
			lines = append(lines, block(ansi.NewStyle().Foreground(th.Warning).Render("Taking a while. Press Esc to give up.")))
		}
		if m.have {
			lines = append(lines, block(ansi.NewStyle().Faint().Render("Still showing "+m.fc.Place.Name+" below.")))
			lines = append(lines, layout.FlexChild{Node: m.forecastNode(th, max(1, m.bodyH()-len(lines)-1))})
		}
		return layout.Column(1, lines...)
	case failed:
		return layout.Column(1,
			block(ansi.NewStyle().Bold().Foreground(th.Error).Render("✗ Could not load "+m.target.String())),
			block(ansi.Truncate(m.err, w)),
			layout.FlexChild{Node: layout.Row(2, layout.FlexChild{Node: m.button("retry", "Retry (r)", th)}, block(ansi.NewStyle().Faint().Render("or search for another city")))},
		)
	case loaded:
		return m.forecastNode(th, m.bodyH())
	}
	lines := []layout.FlexChild{
		block("Search for a city above and press Enter."),
		block(ansi.NewStyle().Faint().Render("Open a recent one with its number, or press F1 for help.")),
	}
	if m.prov.name() == "built-in fixture" {
		lines = append(lines, block(ansi.NewStyle().Faint().Render("Try Berlin, Cairo, Tokyo… or Reykjavik (fails once), Atlantis (does not exist), Slowville (never answers).")))
	}
	return layout.Column(1, lines...)
}

func (m model) forecastNode(th theme.Theme, h int) layout.Node {
	w := m.size().W
	fc := m.fc
	glyph, words := sky(fc.NowCode)
	head := []layout.FlexChild{
		block(ansi.NewStyle().Bold().Foreground(th.Primary).Render(fc.Place.String()) + ansi.NewStyle().Faint().Render("  "+m.fetchedAt())),
		block(fmt.Sprintf("Now %s  %s %s  wind %s", m.units.temp(fc.NowTemp), glyph, words, m.units.wind(fc.NowWind))),
	}
	room := h - len(head) - 2
	per := max(1, w/cardW)
	var days layout.Node
	switch {
	case w < 2*cardW || ((len(fc.Days)+per-1)/per)*cardH > room: // the cards, wrapped, would not fit
		days = m.dayList(th)
	default:
		days = m.dayCards(th, w)
	}
	rows := append(head, layout.FlexChild{Node: days})
	rows = append(rows, block(ansi.Truncate(m.detail(), w)))
	return layout.Fixed(layout.Column(0, rows...), layout.Size{W: w, H: h})
}

func (m model) fetchedAt() string {
	f := m.fc
	return "fetched " + f.Fetched.Format("15:04:05") + " from " + f.Provider
}

func (m model) detail() string {
	if m.sel >= len(m.fc.Days) {
		return ""
	}
	d := m.fc.Days[m.sel]
	_, words := sky(d.Code)
	return fmt.Sprintf("%s: %s, high %s, low %s, rain %s, wind up to %s", d.Date.Format("Mon 2 Jan"), words, m.units.temp(d.Hi), m.units.temp(d.Lo), m.units.rain(d.Rain), m.units.wind(d.Wind))
}

func (m model) dayList(th theme.Theme) layout.Node {
	var kids []layout.FlexChild
	for i, d := range m.fc.Days {
		g, words := sky(d.Code)
		line := fmt.Sprintf("%s %s %-13s %s/%s", d.Date.Format("Mon 2"), g, ansi.Truncate(words, 13), m.units.temp(d.Hi), m.units.temp(d.Lo))
		st := ansi.NewStyle()
		if i == m.sel {
			st = ansi.NewStyle().Bold().Background(th.Selection)
		}
		kids = append(kids, layout.FlexChild{Node: layout.Named("day-"+strconv.Itoa(i), layout.Block(st.Render(ansi.Truncate(line, m.size().W))))})
	}
	return layout.Column(0, kids...)
}

func (m model) dayCards(th theme.Theme, w int) layout.Node {
	per := max(1, w/cardW)
	var rows []layout.FlexChild
	for start := 0; start < len(m.fc.Days); start += per {
		var cards []layout.FlexChild
		for i := start; i < min(start+per, len(m.fc.Days)); i++ {
			cards = append(cards, layout.FlexChild{Node: m.card(i, th)})
		}
		rows = append(rows, layout.FlexChild{Node: layout.Row(0, cards...)})
	}
	return layout.Column(0, rows...)
}

func (m model) card(i int, th theme.Theme) layout.Node {
	d := m.fc.Days[i]
	g, words := sky(d.Code)
	edge := ansi.NewStyle().Foreground(th.Muted)
	title := ansi.NewStyle().Faint()
	if i == m.sel {
		edge, title = ansi.NewStyle().Bold().Foreground(th.Primary), ansi.NewStyle().Bold()
	}
	inner := cardW - 2
	body := []string{
		title.Render(d.Date.Format("Mon 2 Jan")),
		g + " " + ansi.Truncate(words, inner-2),
		"▲ " + m.units.temp(d.Hi),
		"▼ " + m.units.temp(d.Lo),
		"☂ " + m.units.rain(d.Rain),
	}
	lines := []layout.FlexChild{block(edge.Render("╭" + strings.Repeat("─", inner) + "╮"))}
	for _, b := range body {
		lines = append(lines, block(edge.Render("│")+pad(ansi.Truncate(b, inner), inner)+edge.Render("│")))
	}
	lines = append(lines, block(edge.Render("╰"+strings.Repeat("─", inner)+"╯")))
	return layout.Named("day-"+strconv.Itoa(i), layout.Fixed(layout.Column(0, lines...), layout.Size{W: cardW, H: cardH}))
}

func (m model) helpNode(th theme.Theme) layout.Node {
	key := func(k, a string) layout.FlexChild {
		return block(ansi.Truncate(" "+ansi.NewStyle().Bold().Foreground(th.Primary).Render(pad(k, 14))+a, m.size().W))
	}
	return layout.Column(0,
		key("type, Enter", "search for a city, open the chosen one"), key("Up Down", "choose a result"), key("Tab", "search box or days"),
		key("Left Right", "pick a day"), key("r", "reload, or retry after a failure"), key("u", "°C or °F"),
		key("1-5", "reopen a recent city"), key("Esc", "cancel a load, close the results"), key("F1", "close help"),
	)
}

func (m model) statusLine(th theme.Theme) string {
	w := m.size().W
	switch {
	case m.msg != "":
		return ansi.Truncate(m.msg, w)
	case m.state == loading:
		return ansi.NewStyle().Foreground(th.Warning).Render("● loading")
	}
	return ansi.NewStyle().Faint().Render(m.prov.name())
}

func (m model) screen() layout.Node {
	th := m.theme()
	w := m.size().W
	var body layout.Node
	switch {
	case m.help:
		body = m.helpNode(th)
	case m.showRes && m.typing:
		body = m.resultsNode(th)
	default:
		body = m.stateNode(th)
	}
	hints := widgets.Footer(w, widgets.Hint{Key: "enter", Action: "open"}, widgets.Hint{Key: "tab", Action: "days"},
		widgets.Hint{Key: "r", Action: "reload"}, widgets.Hint{Key: "u", Action: "units"}, widgets.Hint{Key: "esc", Action: "cancel"},
		widgets.Hint{Key: "f1", Action: "help"})
	title := ansi.NewStyle().Bold().Foreground(th.Primary).Render("weather")
	return layout.Column(0,
		layout.FlexChild{Node: layout.Block(title)},
		layout.FlexChild{Node: layout.Named("input", layout.Block(ansi.Truncate(m.in.View(), w)))},
		layout.FlexChild{Node: m.recentsNode(th)},
		layout.FlexChild{Node: layout.Fixed(body, layout.Size{W: w, H: m.bodyH()})},
		layout.FlexChild{Node: layout.Block(m.statusLine(th))},
		layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render(ansi.Truncate(hints, w)))},
	)
}

func (m model) View() string { return layout.DrawTight(m.screen(), m.size()) }

func main() {
	live := flag.Bool("live", false, "fetch real forecasts from open-meteo.com instead of the built-in fixture")
	flag.Parse()
	var p provider = newFixture()
	debounce := time.Duration(0)
	if *live {
		p, debounce = newOpenMeteo(), 250*time.Millisecond
	}
	city := strings.Join(flag.Args(), " ")
	if _, err := ui.NewProgram(newModel(p, debounce, city), tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
