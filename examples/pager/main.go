// Command pager reads a file or standard input the way less does: page and
// line scrolling, search with highlighted matches, a percent bar you can click,
// optional line numbers and wrapping, and a follow mode for files that grow.
// With no file and nothing piped in, it opens a sample.
//
// Mouse: scroll the wheel, drag to select text (y copies it), click the bar in
// the status line to jump to that place. Keyboard: j, k and the arrows scroll
// a line; space, f and PgDn a page; b and PgUp back a page; d and u half a
// page; g and G the top and bottom; Left and Right scroll sideways when lines
// are not wrapped; / searches, n and N go to the next and previous match, Esc
// clears the search; w toggles wrapping, # line numbers (with wrapping off),
// F following; ? or F1 shows help; q or Ctrl+C quits.
//
//	go run ./examples/pager [file]
//	some-command | go run ./examples/pager
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/examples/internal/ui"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/textinput"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/viewport"
	"github.com/ows4444/tui/widgets"
)

const (
	maxLines = 200000 // lines kept; older ones are dropped
	barW     = 12
)

type model struct {
	th      ui.Theming // the theme the Program chose; see SetTheme
	vp      viewport.Model
	name    string
	dropped int // lines dropped from the front, so numbers stay true
	eof     bool
	follow  *atomic.Bool
	sticky  bool // stay at the bottom while lines arrive
	numbers bool
	query   string
	typing  bool
	in      textinput.Model
	msg     string
	help    bool
	hover   bool
	err     error
	stop    context.CancelFunc
	start   tui.Cmd // the first read, run by Init

	width, height int
}

func newModel(name string) model {
	in := textinput.New()
	in.Prompt = "/ "
	in.Width = 40
	vp := viewport.New(80, 22)
	vp.SoftWrap, vp.Mouse = true, true
	return model{vp: vp, name: name, in: in, follow: &atomic.Bool{}}
}

// begin starts reading r. The first read is kept for Init to hand to the
// Program: Init can only return a Cmd, and this is where the model is built.
func (m model) begin(r io.Reader, following bool) model {
	m.follow.Store(following)
	m.sticky = following
	m.start, m.stop = startReading(r, m.follow)
	return m
}

func (m model) Init() tui.Cmd { return m.start }

// ---- geometry --------------------------------------------------------------

func (m model) size() layout.Size {
	if m.width == 0 {
		return layout.Size{W: 80, H: 24}
	}
	return layout.Size{W: m.width, H: m.height}
}

func (m model) bodyH() int { return max(1, m.size().H-2) }

func (m model) gutter() int {
	if !m.numbers {
		return 0
	}
	return len(strconv.Itoa(m.dropped+m.vp.LineCount())) + 1
}

func (m model) fit() model {
	m.vp.Width = max(1, m.size().W-m.gutter())
	m.vp.Height = m.bodyH()
	m.vp.Bounds = layout.Rect{X: m.gutter(), Y: 0, W: m.vp.Width, H: m.vp.Height}
	return m
}

// ---- update ----------------------------------------------------------------

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m.fit(), nil
	case batchMsg:
		return m.onBatch(msg)
	case eofMsg:
		m.eof, m.err = true, msg.err
		return m, nil
	case tui.Key:
		return m.key(msg)
	case tui.MouseEvent:
		return m.mouse(msg)
	}
	return m, nil
}

func (m model) onBatch(b batchMsg) (tui.Model, tui.Cmd) {
	atBottom := m.vp.AtBottom()
	for _, l := range b.lines {
		m.vp.AppendLine(l)
	}
	m.dropped += m.vp.TrimFront(maxLines)
	if m.sticky && atBottom {
		m.vp.GotoBottom()
	}
	return m.fit(), next(b.ch)
}

func (m model) quit() (tui.Model, tui.Cmd) {
	if m.stop != nil {
		m.stop()
	}
	return m, tui.Quit()
}

func (m model) key(k tui.Key) (tui.Model, tui.Cmd) {
	if k.Type == tui.KeyCtrlC {
		return m.quit()
	}
	if m.typing {
		return m.searchKey(k)
	}
	if k.Type == tui.KeyF1 {
		m.help = !m.help
		return m, nil
	}
	if m.help {
		m.help = k.Type != tui.KeyEsc && k.Type != tui.KeyEnter
		return m, nil
	}
	m.msg = ""
	if k.Type == tui.KeyRunes {
		return m.runeKey(k)
	}
	switch k.Type {
	case tui.KeyEsc:
		m.vp.Search("")
		m.query = ""
	case tui.KeyLeft:
		m.vp.LineLeft(8)
	case tui.KeyRight:
		m.vp.LineRight(8)
	default:
		var cmd tui.Cmd
		m.vp, cmd = m.vp.Update(k)
		return m, cmd
	}
	return m, nil
}

func (m model) runeKey(k tui.Key) (tui.Model, tui.Cmd) {
	switch k.Text {
	case "q":
		return m.quit()
	case "?":
		m.help = true
	case "j":
		m.vp.LineDown(1)
	case "k":
		m.vp.LineUp(1)
	case " ", "f":
		m.vp.PageDown()
	case "b":
		m.vp.PageUp()
	case "d":
		m.vp.HalfPageDown()
	case "u":
		m.vp.HalfPageUp()
	case "g":
		m.vp.GotoTop()
	case "G":
		m.vp.GotoBottom()
	case "h":
		m.vp.LineLeft(8)
	case "l":
		m.vp.LineRight(8)
	case "/":
		m.typing = true
		m.in.SetValue(m.query)
		m.in.CursorEnd()
		return m, m.in.Focus()
	case "w":
		m.vp.SoftWrap = !m.vp.SoftWrap
		if m.vp.SoftWrap {
			m.numbers = false
		}
		m.msg = map[bool]string{true: "Wrapping on", false: "Wrapping off"}[m.vp.SoftWrap]
	case "#":
		return m.toggleNumbers(), nil
	case "F":
		return m.toggleFollow(), nil
	default:
		var cmd tui.Cmd
		m.vp, cmd = m.vp.Update(k) // n, N and y belong to the viewport
		return m, cmd
	}
	return m.fit(), nil
}

func (m model) toggleNumbers() model {
	if m.vp.SoftWrap {
		m.msg = "Line numbers need wrapping off (press w)"
		return m
	}
	m.numbers = !m.numbers
	return m.fit()
}

func (m model) toggleFollow() model {
	m.sticky = !m.sticky
	m.follow.Store(m.sticky)
	if m.sticky {
		m.vp.GotoBottom()
		m.msg = "Following: the view stays at the end"
	} else {
		m.msg = "Not following"
	}
	return m
}

func (m model) searchKey(k tui.Key) (tui.Model, tui.Cmd) {
	switch k.Type {
	case tui.KeyEsc:
		m.typing = false
		m.in.Blur()
		return m, nil
	case tui.KeyEnter:
		m.typing = false
		m.in.Blur()
		m.query = m.in.Value()
		if n := m.vp.Search(m.query); m.query != "" {
			m.msg = fmt.Sprintf("%d matches", n)
			if n == 0 {
				m.msg = "Pattern not found: " + m.query
			}
		}
		return m, nil
	}
	var cmd tui.Cmd
	m.in, cmd = m.in.Update(k)
	return m, cmd
}

// ---- mouse -----------------------------------------------------------------

func inside(r layout.Rect, x, y int) bool { return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H }

func (m model) mouse(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	if m.typing || m.help {
		return m, nil
	}
	bar, haveBar := layout.RectOf(m.screen(), m.size(), "bar")
	if ev.Action == tui.MouseActionMotion {
		m.hover = haveBar && inside(bar, ev.X, ev.Y) // a drag also goes on to the viewport, below
	}
	if haveBar && inside(bar, ev.X, ev.Y) && ev.Button == tui.MouseButtonLeft && ev.Action == tui.MouseActionPress {
		return m.jumpTo(float64(ev.X-bar.X) / float64(max(1, bar.W-1))), nil
	}
	var cmd tui.Cmd
	m.vp, cmd = m.vp.Update(ev)
	return m, cmd
}

// jumpTo scrolls to a fraction of the way through the text.
func (m model) jumpTo(frac float64) model {
	span := max(0, m.vp.LineCount()-m.vp.Height)
	m.vp.SetYOffset(int(frac*float64(span) + 0.5))
	return m
}

// ---- view ------------------------------------------------------------------

func pad(s string, w int) string {
	if n := ansi.Width(s); n < w {
		return s + strings.Repeat(" ", w-n)
	}
	return s
}

func (m model) body(th theme.Theme) layout.Node {
	h := m.bodyH()
	lines := strings.Split(m.vp.View(), "\n")
	var rows []layout.FlexChild
	g := m.gutter()
	for i := 0; i < h; i++ {
		l := ""
		if i < len(lines) {
			l = lines[i]
		}
		if g > 0 {
			n := ""
			if idx := m.vp.YOffset() + i; idx < m.vp.LineCount() {
				n = strconv.Itoa(m.dropped + idx + 1)
			}
			l = ansi.NewStyle().Faint().Render(fmt.Sprintf("%*s ", g-1, n)) + l
		}
		rows = append(rows, layout.FlexChild{Node: layout.Block(pad(ansi.Truncate(l, m.size().W), m.size().W))})
	}
	return layout.Column(0, rows...)
}

func (m model) statusNode(th theme.Theme) layout.Node {
	w := m.size().W
	total := m.vp.LineCount()
	first := min(total, m.vp.YOffset()+1)
	last := min(total, m.vp.YOffset()+m.vp.Height)
	pct := int(m.vp.ScrollPercent()*100 + 0.5)
	parts := []string{ansi.NewStyle().Bold().Render(m.name)}
	if total == 0 {
		parts = append(parts, "empty")
	} else {
		parts = append(parts, fmt.Sprintf("lines %d–%d of %d", m.dropped+first, m.dropped+last, m.dropped+total))
	}
	if n := m.vp.MatchCount(); m.query != "" && n > 0 {
		parts = append(parts, fmt.Sprintf("match %d/%d", m.vp.CurrentMatch(), n))
	}
	switch {
	case !m.eof:
		parts = append(parts, "reading…")
	case m.err != nil:
		parts = append(parts, ansi.NewStyle().Foreground(th.Error).Render("✗ "+m.err.Error()))
	}
	if m.sticky {
		parts = append(parts, ansi.NewStyle().Foreground(th.Warning).Render("▶ following"))
	}
	left := strings.Join(parts, "  ")
	if avail := max(1, w-barW-6); ansi.Width(left) > avail && total > 0 {
		// Too narrow for the words: keep the numbers, and shorten the name to fit.
		nums := fmt.Sprintf("%d–%d/%d", m.dropped+first, m.dropped+last, m.dropped+total)
		name := ansi.Truncate(m.name, max(0, avail-ansi.Width(nums)-2))
		left = strings.TrimSpace(ansi.NewStyle().Bold().Render(name) + "  " + nums)
	}
	fill := barW * pct / 100
	st := ansi.NewStyle().Foreground(th.Primary)
	if m.hover {
		st = st.Underline()
	}
	bar := st.Render(strings.Repeat("━", fill)) + ansi.NewStyle().Faint().Render(strings.Repeat("─", barW-fill))
	return layout.Fixed(layout.Row(1,
		layout.FlexChild{Node: layout.Block(ansi.Truncate(left, max(1, w-barW-6))), Grow: 1},
		layout.FlexChild{Node: layout.Named("bar", layout.Block(bar))},
		layout.FlexChild{Node: layout.Block(fmt.Sprintf("%3d%%", pct))},
	), layout.Size{W: w, H: 1})
}

func (m model) bottom(th theme.Theme) string {
	w := m.size().W
	if m.typing {
		return ansi.Truncate(m.in.View(), w)
	}
	if m.msg != "" {
		return ansi.Truncate(m.msg, w)
	}
	return ansi.NewStyle().Faint().Render(ansi.Truncate(widgets.Footer(w,
		widgets.Hint{Key: "space", Action: "page"}, widgets.Hint{Key: "/", Action: "search"}, widgets.Hint{Key: "n", Action: "next"},
		widgets.Hint{Key: "w", Action: "wrap"}, widgets.Hint{Key: "#", Action: "numbers"}, widgets.Hint{Key: "F", Action: "follow"},
		widgets.Hint{Key: "?", Action: "help"}, widgets.Hint{Key: "q", Action: "quit"}), w))
}

func (m model) helpBody(th theme.Theme) layout.Node {
	key := func(k, a string) layout.FlexChild {
		return layout.FlexChild{Node: layout.Block(ansi.Truncate(" "+ansi.NewStyle().Bold().Foreground(th.Primary).Render(pad(k, 14))+a, m.size().W))}
	}
	return layout.Fixed(layout.Column(0,
		key("j k ↑ ↓", "scroll a line"), key("space f PgDn", "next page"), key("b PgUp", "previous page"),
		key("d u", "half a page down, up"), key("g G", "top, bottom"), key("← →", "scroll sideways (wrap off)"),
		key("/", "search; Enter runs it"), key("n N", "next, previous match"), key("Esc", "clear the search"),
		key("w # F", "wrap, line numbers, follow"), key("y", "copy the selected text"), key("q", "quit"),
	), layout.Size{W: m.size().W, H: m.bodyH()})
}

func (m model) screen() layout.Node {
	th := m.theme()
	var body layout.Node = m.body(th)
	if m.help {
		body = m.helpBody(th)
	}
	return layout.Column(0,
		layout.FlexChild{Node: body},
		layout.FlexChild{Node: m.statusNode(th)},
		layout.FlexChild{Node: layout.Block(m.bottom(th))},
	)
}

func (m model) View() string { return layout.DrawTight(m.screen(), m.size()) }

// open chooses what to read: a file, "-" or a pipe for standard input, or the
// sample.
func open(args []string, stdin *os.File) (r io.Reader, name string, err error) {
	switch {
	case len(args) > 0 && args[0] != "-":
		f, err := os.Open(args[0]) // #nosec G304 G703 -- a pager opens the file its user names
		return f, args[0], err
	case len(args) > 0:
		return stdin, "stdin", nil
	}
	if info, err := stdin.Stat(); err == nil && info.Mode()&os.ModeCharDevice == 0 {
		return stdin, "stdin", nil
	}
	return strings.NewReader(sample()), "sample", nil
}

func main() {
	r, name, err := open(os.Args[1:], os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "pager:", err)
		os.Exit(1)
	}
	m := newModel(name).begin(r, false)
	opts := []tui.ProgramOption{tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion)}
	if tty, err := os.Open("/dev/tty"); err == nil && name == "stdin" {
		defer tty.Close() // the keys come from the terminal, not from the pipe
		opts = append(opts, tui.WithInput(tty))
	}
	if _, err := ui.NewProgram(m, opts...).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
