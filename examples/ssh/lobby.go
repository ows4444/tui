package main

import (
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/examples/internal/ui"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

func itoa(n int) string { return strconv.Itoa(n) }

const (
	chromeRows = 4 // title, status, prompt, hints
	sideW      = 18
)

// lobby is the program every session runs: a shared chat room. It talks to the
// other sessions only through the hub.
type lobby struct {
	th     ui.Theming // the theme the Program chose; see SetTheme
	hub    *hub
	id     int
	nick   string
	remote string
	users  []string
	log    []event
	prompt []rune
	cur    int
	sent   []string // what this session typed, for Up and Down
	pos    int
	off    int // lines scrolled up from the newest
	help   bool
	msg    string
	hover  string

	w, h int
}

// newLobby joins the hub as nick and returns the model. send delivers a message
// to this session's program.
func newLobby(h *hub, nick, remote string, send func(tui.Msg) bool) (*lobby, error) {
	id, name, past, users, err := h.join(nick, send)
	if err != nil {
		return nil, err
	}
	return &lobby{hub: h, id: id, nick: name, remote: remote, users: users, log: past}, nil
}

func (l *lobby) Init() tui.Cmd { return nil }

// Update works on a copy so the model stays a value as far as the program can
// tell; the hub and id are shared on purpose.
func (l *lobby) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	c := *l
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		c.w, c.h = msg.Width, msg.Height
	case hubMsg:
		c.users = msg.users
		c.log = append(append([]event(nil), c.log...), msg.ev)
		if len(c.log) > maxHistory {
			c.log = c.log[len(c.log)-maxHistory:]
		}
		if c.off > 0 {
			c.off++ // stay where the reader is
		}
	case tui.PasteEvent:
		c.insert(cleanText(msg.Text))
	case tui.Key:
		return c.key(msg)
	case tui.MouseEvent:
		return c.mouse(msg)
	}
	return &c, nil
}

func (l *lobby) size() layout.Size {
	if l.w == 0 {
		return layout.Size{W: 80, H: 24}
	}
	return layout.Size{W: l.w, H: l.h}
}

func (l *lobby) insert(s string) {
	rs := []rune(s)
	p := make([]rune, 0, len(l.prompt)+len(rs))
	l.prompt = append(append(append(p, l.prompt[:l.cur]...), rs...), l.prompt[l.cur:]...)
	l.cur += len(rs)
	l.pos = len(l.sent)
}

func (l *lobby) key(k tui.Key) (tui.Model, tui.Cmd) {
	if l.help {
		if k.Type == tui.KeyCtrlC {
			return l, tui.Quit()
		}
		l.help = false
		return l, nil
	}
	l.msg = ""
	switch k.Type {
	case tui.KeyCtrlC:
		return l, tui.Quit()
	case tui.KeyCtrl:
		return l.ctrl(k.Code)
	case tui.KeyEnter:
		return l.submit()
	case tui.KeyEsc:
		l.prompt, l.cur = nil, 0
	case tui.KeyPgUp:
		l.scroll(l.page())
	case tui.KeyPgDown:
		l.scroll(-l.page())
	case tui.KeyUp:
		l.history(-1)
	case tui.KeyDown:
		l.history(1)
	case tui.KeyLeft:
		l.cur = max(0, l.cur-1)
	case tui.KeyRight:
		l.cur = min(len(l.prompt), l.cur+1)
	case tui.KeyHome:
		l.cur = 0
	case tui.KeyEnd:
		l.cur = len(l.prompt)
	case tui.KeyBackspace:
		if l.cur > 0 {
			l.prompt = append(append([]rune(nil), l.prompt[:l.cur-1]...), l.prompt[l.cur:]...)
			l.cur--
		}
	case tui.KeyDelete:
		if l.cur < len(l.prompt) {
			l.prompt = append(append([]rune(nil), l.prompt[:l.cur]...), l.prompt[l.cur+1:]...)
		}
	case tui.KeyTab:
		l.complete()
	case tui.KeyRunes, tui.KeySpace:
		if k.Text == "?" && len(l.prompt) == 0 {
			l.help = true
			return l, nil
		}
		l.insert(ansi.Sanitize(k.Text))
	}
	return l, nil
}

func (l *lobby) ctrl(c rune) (tui.Model, tui.Cmd) {
	switch c {
	case 'd':
		return l, tui.Quit()
	case 'u':
		l.prompt, l.cur = nil, 0
	}
	return l, nil
}

func (l *lobby) page() int { return max(1, l.size().H-chromeRows-1) }

func (l *lobby) scroll(d int) {
	l.off = max(0, min(max(0, len(l.rows())-l.page()-1), l.off+d))
}

func (l *lobby) history(d int) {
	if len(l.sent) == 0 {
		return
	}
	l.pos = max(0, min(len(l.sent), l.pos+d))
	text := ""
	if l.pos < len(l.sent) {
		text = l.sent[l.pos]
	}
	l.prompt = []rune(text)
	l.cur = len(l.prompt)
}

// complete finishes the @name being typed, or adds the first other name.
func (l *lobby) complete() {
	word := ""
	if i := strings.LastIndex(string(l.prompt[:l.cur]), "@"); i >= 0 {
		word = string(l.prompt[:l.cur])[i+1:]
	}
	for _, u := range l.users {
		if u != l.nick && strings.HasPrefix(strings.ToLower(u), strings.ToLower(word)) {
			if word != "" {
				l.insert(u[len(word):] + " ")
			} else {
				l.insert("@" + u + " ")
			}
			return
		}
	}
	l.msg = "No one to complete"
}

func (l *lobby) submit() (tui.Model, tui.Cmd) {
	text := strings.TrimSpace(string(l.prompt))
	if text == "" {
		return l, nil
	}
	l.prompt, l.cur = nil, 0
	l.sent = append(append([]string(nil), l.sent...), text)
	l.pos = len(l.sent)
	l.off = 0
	if !strings.HasPrefix(text, "/") {
		l.report(l.hub.say(l.id, text, false))
		return l, nil
	}
	arg := strings.TrimSpace(strings.TrimPrefix(text, strings.Fields(text)[0]))
	switch strings.Fields(text)[0] {
	case "/me":
		l.report(l.hub.say(l.id, arg, true))
	case "/nick":
		name, err := l.hub.rename(l.id, arg)
		if err != nil {
			l.msg = err.Error()
		} else {
			l.nick = name
		}
	case "/who":
		l.msg = itoa(len(l.users)) + " online: " + strings.Join(l.users, ", ")
	case "/help":
		l.help = true
	case "/quit":
		return l, tui.Quit()
	default:
		l.msg = "Unknown command " + strings.Fields(text)[0] + "; /help lists them"
	}
	return l, nil
}

func (l *lobby) report(err error) {
	if err != nil {
		l.msg = err.Error()
	}
}

func inside(r layout.Rect, x, y int) bool { return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H }

func (l *lobby) hit(x, y int) string {
	scr := l.screen()
	for i := range l.users {
		id := "user-" + itoa(i)
		if r, ok := layout.RectOf(scr, l.size(), id); ok && inside(r, x, y) {
			return id
		}
	}
	return ""
}

func (l *lobby) mouse(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	if l.help {
		return l, nil
	}
	id := l.hit(ev.X, ev.Y)
	switch {
	case ev.Action == tui.MouseActionMotion:
		l.hover = id
	case ev.Button == tui.MouseButtonWheelUp:
		l.scroll(3)
	case ev.Button == tui.MouseButtonWheelDown:
		l.scroll(-3)
	case ev.Button == tui.MouseButtonLeft && ev.Action == tui.MouseActionPress && strings.HasPrefix(id, "user-"):
		n, _ := strconv.Atoi(id[5:])
		if n < len(l.users) && l.users[n] != l.nick {
			l.insert("@" + l.users[n] + " ")
		}
	}
	return l, nil
}

// ---- view ------------------------------------------------------------------

func wrap(text string, w int) []string {
	w = max(8, w)
	var out []string
	line := ""
	for _, word := range strings.Fields(text) {
		switch {
		case line == "":
			line = word
		case ansi.Width(line)+1+ansi.Width(word) <= w:
			line += " " + word
		default:
			out = append(out, line)
			line = word
		}
	}
	return append(out, line)
}

func (l *lobby) tone(nick string, th theme.Theme) ansi.Style {
	cols := []ansi.Color{th.Primary, th.Secondary, th.Success, th.Warning, th.Error, th.Info}
	n := 0
	for _, r := range nick {
		n += int(r)
	}
	return ansi.NewStyle().Bold().Foreground(cols[n%len(cols)])
}

func (l *lobby) logWidth() int {
	if l.size().W >= 60 {
		return l.size().W - sideW - 1
	}
	return l.size().W
}

// rows is the log as wrapped, styled lines.
func (l *lobby) rows() []string {
	th := l.theme()
	w := l.logWidth()
	var out []string
	faint := ansi.NewStyle().Faint()
	for _, e := range l.log {
		at := faint.Render(e.at.Format("15:04") + " ")
		switch e.kind {
		case evJoin:
			out = append(out, at+faint.Render("→ "+e.who+" joined"))
		case evLeave:
			out = append(out, at+faint.Render("← "+e.who+" left"))
		case evNick:
			out = append(out, at+faint.Render("· "+e.text+" is now "+e.who))
		case evMe:
			for _, s := range wrap("* "+e.who+" "+e.text, w-6) {
				out = append(out, at+ansi.NewStyle().Italic().Render(s))
				at = "      "
			}
		default:
			prefix := l.tone(e.who, th).Render(e.who) + ": "
			lines := wrap(e.who+": "+e.text, w-6)
			for i, s := range lines {
				if i == 0 {
					s = prefix + s[len(e.who)+2:]
				}
				out = append(out, at+s)
				at = "      "
			}
		}
	}
	return out
}

func (l *lobby) sidebar(th theme.Theme, h int) layout.Node {
	rows := []layout.FlexChild{{Node: layout.Block(ansi.NewStyle().Bold().Render(pad("Online ("+itoa(len(l.users))+")", sideW)))}}
	for i, u := range l.users {
		id := "user-" + itoa(i)
		st := l.tone(u, th)
		label := "  " + u
		switch {
		case u == l.nick:
			label = "▸ " + u + " (you)"
		case l.hover == id:
			st = st.Underline()
		}
		rows = append(rows, layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render(pad(ansi.Truncate(label, sideW), sideW))))})
	}
	for len(rows) < h {
		rows = append(rows, layout.FlexChild{Node: layout.Block(strings.Repeat(" ", sideW))})
	}
	return layout.Fixed(layout.Column(0, rows[:h]...), layout.Size{W: sideW, H: h})
}

func pad(s string, w int) string {
	if n := ansi.Width(s); n < w {
		return s + strings.Repeat(" ", w-n)
	}
	return s
}

func (l *lobby) promptRow() string {
	w := l.size().W
	start := max(0, l.cur-(w-4))
	rs := l.prompt[min(start, len(l.prompt)):]
	cur := l.cur - start
	before := string(rs[:min(cur, len(rs))])
	at, after := " ", ""
	if cur < len(rs) {
		at, after = string(rs[cur]), string(rs[cur+1:])
	}
	label := ansi.NewStyle().Bold().Foreground(l.theme().Primary).Render(l.nick + "› ")
	if len(l.prompt) == 0 {
		return ansi.Truncate(label+ansi.NewStyle().Reverse().Render(" ")+ansi.NewStyle().Faint().Render("say something (/help)"), w)
	}
	return ansi.Truncate(label+before+ansi.NewStyle().Reverse().Render(at)+after, w)
}

func (l *lobby) helpRows(th theme.Theme) []string {
	key := func(k, a string) string {
		return " " + ansi.NewStyle().Bold().Foreground(th.Primary).Render(pad(k, 16)) + a
	}
	return []string{
		key("Enter", "send"), key("/me text", "an action, like \"* ada waves\""), key("/nick name", "change your name"), key("/who", "who is online"),
		key("@name  Tab", "mention someone; Tab completes a name"), key("click a name", "mention them"), key("PgUp PgDn", "scroll the log"),
		key("Up Down", "lines you sent before"), key("/quit  Ctrl+C", "leave"), "", " Press any key to close.",
	}
}

func (l *lobby) status() string {
	if l.msg != "" {
		return l.msg
	}
	s := "you are " + ansi.Sanitize(l.nick) + " from " + ansi.Sanitize(l.remote)
	if l.off > 0 {
		s += "  ·  ↓ " + itoa(l.off) + " lines below (PgDn)"
	}
	return ansi.NewStyle().Faint().Render(s)
}

func (l *lobby) screen() layout.Node {
	th := l.theme()
	s := l.size()
	n := max(1, s.H-chromeRows)
	hints := widgets.Footer(s.W, widgets.Hint{Key: "enter", Action: "send"}, widgets.Hint{Key: "tab", Action: "@name"}, widgets.Hint{Key: "pgup", Action: "scroll"},
		widgets.Hint{Key: "?", Action: "help"}, widgets.Hint{Key: "ctrl+c", Action: "leave"})
	var lines []layout.FlexChild
	if l.help {
		for _, r := range l.helpRows(th) {
			lines = append(lines, layout.FlexChild{Node: layout.Block(ansi.Truncate(r, s.W))})
		}
	} else {
		all := l.rows()
		end := len(all) - l.off
		start := max(0, end-n)
		if end > 0 {
			for _, r := range all[start:min(end, len(all))] {
				lines = append(lines, layout.FlexChild{Node: layout.Block(r)})
			}
		}
	}
	for len(lines) < n {
		lines = append(lines, layout.FlexChild{Node: layout.Block("")})
	}
	logCol := layout.Node(layout.Fixed(layout.Column(0, lines[:n]...), layout.Size{W: l.logWidth(), H: n}))
	body := logCol
	if s.W >= 60 && !l.help {
		sep := make([]layout.FlexChild, n)
		for i := range sep {
			sep[i] = layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render("│"))}
		}
		body = layout.Row(0, layout.FlexChild{Node: logCol}, layout.FlexChild{Node: layout.Fixed(layout.Column(0, sep...), layout.Size{W: 1, H: n})},
			layout.FlexChild{Node: l.sidebar(th, n)})
	}
	title := ansi.NewStyle().Bold().Foreground(th.Primary).Render("lobby") + ansi.NewStyle().Faint().Render("  "+itoa(len(l.users))+" online")
	return layout.Column(0,
		layout.FlexChild{Node: layout.Block(ansi.Truncate(title, s.W))},
		layout.FlexChild{Node: layout.Fixed(body, layout.Size{W: s.W, H: n})},
		layout.FlexChild{Node: layout.Block(ansi.Truncate(l.status(), s.W))},
		layout.FlexChild{Node: layout.Block(l.promptRow())},
		layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render(ansi.Truncate(hints, s.W)))})
}

func (l *lobby) View() string { return layout.DrawTight(l.screen(), l.size()) }
