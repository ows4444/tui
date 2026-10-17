// Command chat is a small chat client with channels, message history, an
// input line, a typing indicator and replies that stream in. The people are
// scripted and it is offline, so it is safe to run and the same every time.
//
// Mouse: click a channel to switch to it, scroll the wheel through history,
// click the input to type, click "new messages" to jump to the latest, click
// "? help" for the keys. On a narrow terminal the channel list folds away;
// click the channel name in the title to move to the next one.
// Keyboard: type and press Enter to send, Ctrl+N and Ctrl+P change channel,
// Up, Down, PgUp and PgDn scroll, Ctrl+L jumps to the latest, F1 or ? (with an
// empty input) shows help, Esc clears the input, Ctrl+Q or Ctrl+C quits.
// Every clickable thing is a named layout node, so the click targets are read
// from the same tree View draws.
//
//	go run ./examples/chat
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/motion"
	"github.com/ows4444/tui/textinput"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

const (
	sidebarW  = 18 // channel list, divider included
	wideFrom  = 60 // narrowest terminal that shows the channel list
	typingFor = 700 * time.Millisecond
	streamGap = 35 * time.Millisecond
	perTick   = 4 // characters revealed per tick
	maxLine   = 2000
)

type typingMsg struct{ seq int }
type streamMsg struct{ seq int }

// live is the reply being written: first only a typing indicator, then text
// arriving a few characters at a time.
type live struct {
	ch      int
	seq     int
	text    []rune
	shown   int
	started bool
}

type model struct {
	chans []channel
	cur   int
	input textinput.Model
	off   int // lines scrolled up from the newest; 0 follows the newest
	sent  int // messages sent, which picks the reply
	clock int // minutes after 09:00, for the time shown

	live    *live
	pending []int // channels still owed a reply, oldest first
	seq     int

	help    bool
	hover   string
	reduced bool

	width, height int
}

func newModel(reduced bool) model {
	in := textinput.New()
	in.Placeholder = "Write a message and press Enter"
	in.Prompt = ""
	in.Focus() // typing works at once; Init starts the cursor blink
	m := model{chans: sampleChannels(), input: in, reduced: reduced, clock: 22}
	return m
}

func (m model) Init() tui.Cmd { return m.input.Focus() }

// ---- layout numbers --------------------------------------------------------

func (m model) size() layout.Size {
	if m.width == 0 {
		return layout.Size{W: 80, H: 24}
	}
	return layout.Size{W: m.width, H: m.height}
}

func (m model) wide() bool { return m.size().W >= wideFrom }

// paneW is the width of the message pane.
func (m model) paneW() int {
	if m.wide() {
		return m.size().W - sidebarW
	}
	return m.size().W
}

// paneH is how many rows the message pane has: all but the title, the input
// and the key hints.
func (m model) paneH() int { return max(1, m.size().H-3) }

// ---- history ---------------------------------------------------------------

// styledName colours an author with a theme token chosen from the name, so a
// person keeps their colour; the name itself is always printed too.
func styledName(th theme.Theme, who string) string {
	if who == "you" {
		return ansi.NewStyle().Bold().Foreground(th.Primary).Render(who)
	}
	tones := []ansi.Color{th.Success, th.Warning, th.Info, th.Secondary}
	sum := 0
	for _, r := range who {
		sum += int(r)
	}
	return ansi.NewStyle().Bold().Foreground(tones[sum%len(tones)]).Render(who)
}

// historyLines is every line of the current channel at width w: a header per
// message, then its wrapped text indented two cells. A reply still arriving is
// the last message, with a cursor; before its first character it is the typing
// indicator.
func (m model) historyLines(w int) []string {
	th := theme.DarkTheme()
	ch := m.chans[m.cur]
	var out []string
	add := func(from, at, text string) {
		out = append(out, ansi.Truncate(styledName(th, from)+"  "+ansi.NewStyle().Faint().Render(at), w))
		for _, l := range strings.Split(ansi.Wrap(text, max(1, w-2)), "\n") {
			out = append(out, "  "+ansi.Truncate(l, max(1, w-2)))
		}
	}
	for _, msg := range ch.msgs {
		add(msg.from, msg.at, msg.text)
	}
	if lv := m.live; lv != nil && lv.ch == m.cur {
		if lv.started {
			add(ch.who, "now", string(lv.text[:lv.shown])+"▌")
		} else {
			out = append(out, ansi.Truncate(ansi.NewStyle().Faint().Italic().Render(ch.who+" is typing…"), w))
		}
	}
	if len(out) == 0 {
		out = append(out, ansi.NewStyle().Faint().Render("No messages yet. Say hello."))
	}
	return out
}

// maxOff is the furthest the history can be scrolled up.
func (m model) maxOff() int {
	return max(0, len(m.historyLines(m.paneW()))-m.paneH())
}

func (m model) clamp() model {
	m.off = max(0, min(m.off, m.maxOff()))
	return m
}

func (m model) stamp() string {
	return fmt.Sprintf("%02d:%02d", 9+m.clock/60, m.clock%60)
}

// ---- update ----------------------------------------------------------------

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m.clamp(), nil
	case tui.MouseEvent:
		return m.mouse(msg)
	case tui.Key:
		return m.key(msg)
	case typingMsg:
		return m.startStream(msg.seq)
	case streamMsg:
		return m.stream(msg.seq)
	}
	return m, nil
}

func (m model) key(k tui.Key) (tui.Model, tui.Cmd) {
	switch k.Type {
	case tui.KeyCtrlC:
		return m, tui.Quit()
	case tui.KeyCtrl:
		switch k.Code {
		case 'q':
			return m, tui.Quit()
		case 'n':
			return m.switchTo(m.cur + 1), nil
		case 'p':
			return m.switchTo(m.cur - 1), nil
		case 'l':
			m.off = 0
			return m, nil
		}
	case tui.KeyF1:
		m.help = !m.help
		return m, nil
	case tui.KeyEsc:
		if m.help {
			m.help = false
		} else {
			m.input.Reset()
		}
		return m, nil
	case tui.KeyUp:
		m.off++
		return m.clamp(), nil
	case tui.KeyDown:
		m.off--
		return m.clamp(), nil
	case tui.KeyPgUp:
		m.off += max(1, m.paneH()-1)
		return m.clamp(), nil
	case tui.KeyPgDown:
		m.off -= max(1, m.paneH()-1)
		return m.clamp(), nil
	case tui.KeyEnter:
		return m.send()
	case tui.KeyRunes:
		if k.Text == "?" && m.input.Value() == "" {
			m.help = !m.help
			return m, nil
		}
	}
	next, cmd := m.input.Update(k)
	m.input = next
	return m, cmd
}

// switchTo makes channel i (wrapping) the current one and reads it.
func (m model) switchTo(i int) model {
	n := len(m.chans)
	m.cur = ((i % n) + n) % n
	cs := append([]channel(nil), m.chans...)
	cs[m.cur].unread = 0
	m.chans = cs
	m.off, m.help = 0, false
	return m
}

func (m model) mouse(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	hitID := func() string {
		for _, id := range m.targets() {
			if rc, ok := layout.RectOf(m.screen(), m.size(), id); ok &&
				ev.X >= rc.X && ev.X < rc.X+rc.W && ev.Y >= rc.Y && ev.Y < rc.Y+rc.H {
				return id
			}
		}
		return ""
	}
	if ev.Action == tui.MouseActionMotion {
		m.hover = hitID()
		return m, nil
	}
	switch ev.Button {
	case tui.MouseButtonWheelUp:
		m.off += 3
		return m.clamp(), nil
	case tui.MouseButtonWheelDown:
		m.off -= 3
		return m.clamp(), nil
	case tui.MouseButtonLeft:
		if ev.Action != tui.MouseActionPress {
			return m, nil
		}
	default:
		return m, nil
	}
	id := hitID()
	switch {
	case strings.HasPrefix(id, "chan-"):
		n, _ := strconv.Atoi(strings.TrimPrefix(id, "chan-"))
		return m.switchTo(n), nil
	case id == "next":
		return m.switchTo(m.cur + 1), nil
	case id == "jump":
		m.off = 0
	case id == "help":
		m.help = !m.help
	case id == "input":
		return m, m.input.Focus()
	}
	return m, nil
}

// targets are the ids of every clickable node the screen can hold.
func (m model) targets() []string {
	ids := []string{"help", "next", "jump", "input"}
	for i := range m.chans {
		ids = append(ids, "chan-"+strconv.Itoa(i))
	}
	return ids
}

// ---- sending and replies ---------------------------------------------------

func (m model) send() (tui.Model, tui.Cmd) {
	text := strings.TrimSpace(m.input.Value())
	if text == "" {
		return m, nil
	}
	if r := []rune(text); len(r) > maxLine {
		text = string(r[:maxLine])
	}
	m.input.Reset()
	m.help, m.off = false, 0
	m = m.append(m.cur, message{"you", m.stamp(), text})
	m.sent++
	m.pending = append(append([]int(nil), m.pending...), m.cur)
	if m.live != nil {
		return m, nil // the reply under way ends, then the next begins
	}
	return m.nextReply()
}

// append files a message under channel i without touching the old slices.
func (m model) append(i int, msg message) model {
	cs := append([]channel(nil), m.chans...)
	cs[i].msgs = append(append([]message(nil), cs[i].msgs...), msg)
	m.chans = cs
	m.clock++
	return m
}

// nextReply starts the oldest owed reply, if any.
func (m model) nextReply() (tui.Model, tui.Cmd) {
	if len(m.pending) == 0 {
		return m, nil
	}
	ch := m.pending[0]
	m.pending = m.pending[1:]
	m.seq++
	c := m.chans[ch]
	m.live = &live{ch: ch, seq: m.seq, text: []rune(c.replies[(m.sent-1)%len(c.replies)])}
	if m.reduced {
		return m.finish()
	}
	seq := m.seq
	return m, tui.FromCtx(motion.After(typingFor, func(time.Time) tui.Msg { return typingMsg{seq} }))
}

func (m model) startStream(seq int) (tui.Model, tui.Cmd) {
	if m.live == nil || m.live.seq != seq {
		return m, nil
	}
	lv := *m.live
	lv.started = true
	m.live = &lv
	return m.clamp(), m.streamTick()
}

func (m model) streamTick() tui.Cmd {
	seq := m.live.seq
	return tui.FromCtx(motion.After(streamGap, func(time.Time) tui.Msg { return streamMsg{seq} }))
}

func (m model) stream(seq int) (tui.Model, tui.Cmd) {
	if m.live == nil || m.live.seq != seq {
		return m, nil
	}
	lv := *m.live
	lv.shown = min(len(lv.text), lv.shown+perTick)
	m.live = &lv
	if lv.shown >= len(lv.text) {
		return m.finish()
	}
	return m.clamp(), m.streamTick()
}

// finish files the reply under its channel and starts the next owed one.
func (m model) finish() (tui.Model, tui.Cmd) {
	lv := m.live
	m.live = nil
	m = m.append(lv.ch, message{m.chans[lv.ch].who, m.stamp(), string(lv.text)})
	if lv.ch != m.cur {
		cs := append([]channel(nil), m.chans...)
		cs[lv.ch].unread++
		m.chans = cs
	}
	return m.nextReply()
}

// ---- view ------------------------------------------------------------------

func pad(s string, w int) string {
	if n := ansi.Width(s); n < w {
		return s + strings.Repeat(" ", w-n)
	}
	return s
}

func (m model) button(id, label string, th theme.Theme) layout.Node {
	st := ansi.NewStyle().Foreground(th.Muted)
	if m.hover == id {
		st = ansi.NewStyle().Bold().Foreground(th.Primary).Underline()
	}
	return layout.Named(id, layout.Block(st.Render(label)))
}

func (m model) title(th theme.Theme) layout.Node {
	w := m.size().W
	left := layout.Block(ansi.NewStyle().Bold().Foreground(th.Primary).Render("chat"))
	if !m.wide() {
		label := "#" + m.chans[m.cur].name
		if n := m.totalUnread(); n > 0 {
			label += fmt.Sprintf(" (+%d unread)", n)
		}
		left = layout.Row(2, layout.FlexChild{Node: left},
			layout.FlexChild{Node: m.button("next", ansi.Truncate(label, max(4, w-14)), th)})
	}
	row := layout.Row(2,
		layout.FlexChild{Node: left, Grow: 1},
		layout.FlexChild{Node: m.button("help", "? help", th)},
	)
	return layout.Fixed(row, layout.Size{W: w, H: 1})
}

func (m model) totalUnread() int {
	n := 0
	for i, c := range m.chans {
		if i != m.cur {
			n += c.unread
		}
	}
	return n
}

func (m model) sidebar(th theme.Theme) layout.Node {
	h := m.paneH()
	rule := ansi.NewStyle().Foreground(th.Muted).Render("│")
	var rows []layout.FlexChild
	for i, c := range m.chans {
		count := ""
		if c.unread > 0 {
			count = " " + strconv.Itoa(c.unread) // a number, so colour is not the only signal
		}
		room := sidebarW - 2 - ansi.Width(count)
		text := pad(ansi.Truncate(" #"+c.name, room), room) + count + " "
		id := "chan-" + strconv.Itoa(i)
		st := ansi.NewStyle()
		switch {
		case i == m.cur:
			st = ansi.NewStyle().Bold().Background(th.Selection)
		case m.hover == id:
			st = ansi.NewStyle().Underline()
		case c.unread > 0:
			st = ansi.NewStyle().Bold()
		}
		rows = append(rows, layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render(text)+rule))})
	}
	for len(rows) < h {
		rows = append(rows, layout.FlexChild{Node: layout.Block(strings.Repeat(" ", sidebarW-1) + rule)})
	}
	return layout.Fixed(layout.Column(0, rows[:h]...), layout.Size{W: sidebarW, H: h})
}

func (m model) helpLines(th theme.Theme) []string {
	key := func(k, a string) string {
		return "  " + ansi.NewStyle().Bold().Foreground(th.Primary).Render(pad(k, 19)) + a
	}
	return []string{
		ansi.NewStyle().Bold().Render("Keys"),
		key("Enter", "send the message"),
		key("Ctrl+N / Ctrl+P", "next / previous channel"),
		key("Up Down PgUp PgDn", "scroll the history"),
		key("Ctrl+L", "jump to the latest message"),
		key("? or F1", "this help (? needs an empty input)"),
		key("Esc", "close help, or clear the input"),
		key("Ctrl+Q", "quit"),
		"",
		ansi.NewStyle().Bold().Render("Mouse"),
		key("click a channel", "switch to it"),
		key("wheel", "scroll the history"),
		key("click the input", "start typing"),
	}
}

// pane is the message area: exactly paneH rows of paneW cells.
func (m model) pane(th theme.Theme) layout.Node {
	h, w := m.paneH(), m.paneW()
	var shown []string
	if m.help {
		shown = m.helpLines(th)
		shown = shown[:min(len(shown), h)]
	} else {
		lines := m.historyLines(w)
		end := max(0, len(lines)-m.off)
		shown = lines[max(0, end-h):end]
	}
	rows := make([]layout.FlexChild, h)
	for i := range rows {
		line := ""
		if i < len(shown) {
			line = ansi.Truncate(shown[i], w)
		}
		rows[i] = layout.FlexChild{Node: layout.Block(pad(line, w))}
	}
	if !m.help && m.off > 0 {
		st := ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Primary)
		if m.hover == "jump" {
			st = st.Underline()
		}
		label := ansi.Truncate(" ↓ newer messages (click or Ctrl+L) ", w)
		rows[h-1] = layout.FlexChild{Node: layout.Named("jump", layout.Block(st.Render(label)))}
	}
	return layout.Fixed(layout.Column(0, rows...), layout.Size{W: w, H: h})
}

func (m model) inputRow(th theme.Theme) layout.Node {
	w := m.size().W
	m.input.Width = max(1, w-4)
	prompt := ansi.NewStyle().Foreground(th.Primary).Render("› ")
	return layout.Named("input", layout.Fixed(layout.Block(ansi.Truncate(prompt+m.input.View(), w)), layout.Size{W: w, H: 1}))
}

func (m model) hints() layout.Node {
	w := m.size().W
	f := widgets.Footer(w,
		widgets.Hint{Key: "Enter", Action: "send"},
		widgets.Hint{Key: "ctrl+n", Action: "channel"},
		widgets.Hint{Key: "PgUp/PgDn", Action: "scroll"},
		widgets.Hint{Key: "?", Action: "help"},
		widgets.Hint{Key: "ctrl+q", Action: "quit"})
	return layout.Block(ansi.NewStyle().Faint().Render(ansi.Truncate(f, w)))
}

func (m model) screen() layout.Node {
	th := theme.DarkTheme()
	body := m.pane(th)
	if m.wide() {
		body = layout.Row(0, layout.FlexChild{Node: m.sidebar(th)}, layout.FlexChild{Node: body})
	}
	return layout.Column(0,
		layout.FlexChild{Node: m.title(th)},
		layout.FlexChild{Node: body},
		layout.FlexChild{Node: m.inputRow(th)},
		layout.FlexChild{Node: m.hints()},
	)
}

func (m model) View() string { return layout.DrawTight(m.screen(), m.size()) }

func main() {
	opts := []tui.ProgramOption{tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion)}
	// ReducedMotion is known as soon as the Program exists, before Run.
	p := tui.NewProgram(newModel(false), opts...)
	if p.ReducedMotion() {
		p = tui.NewProgram(newModel(true), opts...)
	}
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
