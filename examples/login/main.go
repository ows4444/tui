// Command login is a sign-in screen: a username, a masked password you can
// reveal, a remember-me box and a Sign in button. A wrong password shakes the
// panel, clears the password and keeps the username; three wrong tries lock
// the screen for a few seconds with a countdown. The sign-in is simulated, so
// nothing is sent anywhere. The demo account is ada / lovelace1815.
//
// Mouse: click a field or the box to focus it, click "show" or "hide" next to
// the password, click Sign in. Keyboard: Tab and Shift+Tab move between the
// fields, Space ticks the box, Enter signs in (from the username it moves on
// to the password), Ctrl+T shows or hides the password, Esc clears it, F1
// shows help, Ctrl+C quits. With reduced motion the shake is skipped.
//
// The Caps Lock hint is a guess: a terminal cannot report the lock, so the
// hint shows when the password has three or more letters and all are capitals.
//
//	go run ./examples/login
package main

import (
	"fmt"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/examples/internal/ui"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/motion"
	"github.com/ows4444/tui/passwordinput"
	"github.com/ows4444/tui/spinner"
	"github.com/ows4444/tui/textinput"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

const (
	demoUser     = "ada"
	demoPassword = "lovelace1815"
	maxTries     = 3
	lockSeconds  = 5
	labelW       = 10
	revealW      = 6 // the show/hide word and its gap
)

// shakeOffsets are the sideways steps of the panel after a wrong password.
var shakeOffsets = [...]int{-2, 2, -2, 2, -1, 1, 0}

type focus int

const (
	onUser focus = iota
	onPass
	onRemember
	onButton
	numFocus
)

type phase int

const (
	signingIn phase = iota
	signedIn
)

type authMsg struct {
	seq int
	ok  bool
}
type shakeMsg struct{ seq, step int }
type lockMsg struct{ seq int }

type model struct {
	th   ui.Theming // the theme the Program chose; see SetTheme
	user textinput.Model
	pass passwordinput.Model
	spin spinner.Model

	focus    focus
	phase    phase
	remember bool
	remName  string // the username kept for the next sign-in when Remember me was ticked
	reveal   bool
	busy     bool
	err      string
	fails    int
	lock     int // seconds of lock left
	shake    int // index into shakeOffsets, or len when still
	seq      int // tells a stale timer from the current one
	help     bool
	hover    string
	reduced  bool

	width, height int
}

func newModel(reduced bool) model {
	u := textinput.New()
	u.Placeholder = "username"
	p := passwordinput.New()
	p.Placeholder = "password"
	m := model{user: u, pass: p, spin: spinner.New(), reduced: reduced, shake: len(shakeOffsets)}
	m.user.Focus()
	return m
}

func (m model) Init() tui.Cmd {
	u := m.user
	return u.Focus()
}

// after returns a Cmd that sends the Msg fn makes once d has passed.
func after(d time.Duration, fn func() tui.Msg) tui.Cmd {
	return tui.FromCtx(motion.After(d, func(time.Time) tui.Msg { return fn() }))
}

// ---- update ----------------------------------------------------------------

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tui.Key:
		return m.key(msg)
	case tui.MouseEvent:
		return m.mouse(msg)
	case authMsg:
		return m.authDone(msg)
	case shakeMsg:
		return m.shakeStep(msg)
	case lockMsg:
		return m.lockTick(msg)
	default:
		var cmd tui.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd
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
	if m.phase == signedIn {
		return m.welcomeKey(k)
	}
	if m.busy {
		return m, nil
	}
	switch k.Type {
	case tui.KeyTab:
		if k.Mod.Shift() {
			return m.moveFocus(-1)
		}
		return m.moveFocus(1)
	case tui.KeyEnter:
		return m.enter()
	case tui.KeyEsc:
		m.pass.Reset()
		m.err = ""
		return m, nil
	case tui.KeyCtrl:
		if k.Code == 't' {
			m.reveal = !m.reveal
		}
		return m, nil
	}
	return m.typeKey(k)
}

func (m model) typeKey(k tui.Key) (tui.Model, tui.Cmd) {
	var cmd tui.Cmd
	switch m.focus {
	case onUser:
		m.user, cmd = m.user.Update(k)
	case onPass:
		m.pass, cmd = m.pass.Update(k)
	case onRemember:
		if k.Type == tui.KeyRunes && k.Text == " " {
			m.remember = !m.remember
		}
	case onButton:
		if k.Type == tui.KeyRunes && k.Text == " " {
			return m.submit()
		}
	}
	return m, cmd
}

func (m model) welcomeKey(k tui.Key) (tui.Model, tui.Cmd) {
	switch {
	case k.Type == tui.KeyRunes && k.Text == "q":
		return m, tui.Quit()
	case k.Type == tui.KeyEnter, k.Type == tui.KeyEsc, k.Type == tui.KeyRunes && k.Text == "o":
		return m.signOut()
	}
	return m, nil
}

func (m model) signOut() (tui.Model, tui.Cmd) {
	next := newModel(m.reduced)
	next.width, next.height = m.width, m.height
	next.remember = m.remName != ""
	next.remName = m.remName
	next.user.SetValue(m.remName)
	next.user.CursorEnd()
	next.focus = onPass
	next.user.Blur()
	return next, next.pass.Focus()
}

// moveFocus moves to the next (dir 1) or previous field.
func (m model) moveFocus(dir int) (tui.Model, tui.Cmd) {
	return m.focusOn((m.focus + focus(dir) + numFocus) % numFocus)
}

func (m model) focusOn(f focus) (tui.Model, tui.Cmd) {
	m.user.Blur()
	m.pass.Blur()
	m.focus = f
	switch f {
	case onUser:
		return m, m.user.Focus()
	case onPass:
		return m, m.pass.Focus()
	}
	return m, nil
}

func (m model) enter() (tui.Model, tui.Cmd) {
	if m.focus == onUser {
		return m.focusOn(onPass)
	}
	return m.submit()
}

// ---- signing in ------------------------------------------------------------

func (m model) submit() (tui.Model, tui.Cmd) {
	if m.lock > 0 {
		return m, nil
	}
	if strings.TrimSpace(m.user.Value()) == "" {
		m.err = "Enter your username"
		return m.focusOn(onUser)
	}
	if m.pass.Value() == "" {
		m.err = "Enter your password"
		return m.focusOn(onPass)
	}
	m.err, m.busy = "", true
	ok := strings.TrimSpace(m.user.Value()) == demoUser && m.pass.Value() == demoPassword
	m.seq++
	seq := m.seq
	if m.reduced {
		return m.authDone(authMsg{seq, ok})
	}
	return m, tui.Batch(m.spin.Start(), after(700*time.Millisecond, func() tui.Msg { return authMsg{seq, ok} }))
}

func (m model) authDone(a authMsg) (tui.Model, tui.Cmd) {
	if !m.busy || a.seq != m.seq {
		return m, nil
	}
	m.busy = false
	m.spin.Stop()
	if a.ok {
		m.phase, m.fails = signedIn, 0
		m.remName = ""
		if m.remember {
			m.remName = strings.TrimSpace(m.user.Value())
		}
		return m, nil
	}
	m.fails++
	m.pass.Reset()
	m.err = "Wrong username or password"
	next, cmd := m.focusOn(onPass)
	m = next.(model)
	if m.fails >= maxTries {
		m.lock, m.err = lockSeconds, ""
		m.seq++
		seq := m.seq
		cmd = tui.Batch(cmd, after(time.Second, func() tui.Msg { return lockMsg{seq} }))
	}
	if !m.reduced {
		m.shake = 0
		seq := m.seq
		cmd = tui.Batch(cmd, after(40*time.Millisecond, func() tui.Msg { return shakeMsg{seq, 1} }))
	}
	return m, cmd
}

func (m model) shakeStep(s shakeMsg) (tui.Model, tui.Cmd) {
	if s.seq != m.seq {
		return m, nil
	}
	m.shake = s.step
	if s.step >= len(shakeOffsets) {
		return m, nil
	}
	seq, step := m.seq, s.step+1
	return m, after(40*time.Millisecond, func() tui.Msg { return shakeMsg{seq, step} })
}

func (m model) lockTick(l lockMsg) (tui.Model, tui.Cmd) {
	if l.seq != m.seq || m.lock == 0 {
		return m, nil
	}
	m.lock--
	if m.lock == 0 {
		m.fails = 0
		return m, nil
	}
	seq := m.seq
	return m, after(time.Second, func() tui.Msg { return lockMsg{seq} })
}

// ---- mouse -----------------------------------------------------------------

func (m model) hit(x, y int) string {
	for _, id := range []string{"reveal", "user", "pass", "remember", "button", "signout"} {
		if r, ok := layout.RectOf(m.screen(), m.size(), id); ok && x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H {
			return id
		}
	}
	return ""
}

func (m model) mouse(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	id := m.hit(ev.X, ev.Y)
	if ev.Action == tui.MouseActionMotion {
		m.hover = id
		return m, nil
	}
	if m.help || m.busy || ev.Button != tui.MouseButtonLeft || ev.Action != tui.MouseActionPress {
		return m, nil
	}
	if m.phase == signedIn {
		if id == "signout" {
			return m.signOut()
		}
		return m, nil
	}
	switch id {
	case "reveal":
		m.reveal = !m.reveal
	case "user":
		return m.focusOn(onUser)
	case "pass":
		return m.focusOn(onPass)
	case "remember":
		m.remember = !m.remember
		return m.focusOn(onRemember)
	case "button":
		next, _ := m.focusOn(onButton)
		return next.(model).submit()
	}
	return m, nil
}

// ---- view ------------------------------------------------------------------

func (m model) size() layout.Size {
	if m.width == 0 {
		return layout.Size{W: 80, H: 24}
	}
	return layout.Size{W: m.width, H: m.height}
}

func (m model) inner() int { return max(16, min(40, m.size().W-6)) }

func (m model) compact() bool { return m.size().H < 16 }

func (m model) capsHint() bool {
	letters, upper := 0, 0
	for _, r := range m.pass.Value() {
		if unicode.IsLetter(r) {
			letters++
			if unicode.IsUpper(r) {
				upper++
			}
		}
	}
	return letters >= 3 && upper == letters
}

func block(s string) layout.FlexChild { return layout.FlexChild{Node: layout.Block(s)} }

func (m model) field(id, label string, in string, f focus, th theme.Theme) layout.FlexChild {
	st := ansi.NewStyle().Faint()
	if m.focus == f {
		st = ansi.NewStyle().Bold().Foreground(th.Primary)
	}
	tag := st.Render(fmt.Sprintf("%-*s", labelW, label))
	return layout.FlexChild{Node: layout.Named(id, layout.Block(tag+in))}
}

func (m model) inputWidth() int { return max(4, m.inner()-labelW-revealW) }

func (m model) fields(th theme.Theme) []layout.FlexChild {
	w := m.inputWidth()
	m.user.Width, m.pass.Width = w, w
	passView := m.pass.View()
	if m.reveal {
		passView = m.pass.Model.View()
	}
	word := "show"
	if m.reveal {
		word = "hide"
	}
	rv := ansi.NewStyle().Foreground(th.Muted)
	if m.hover == "reveal" {
		rv = ansi.NewStyle().Foreground(th.Primary).Underline()
	}
	passRow := layout.Row(1,
		m.field("pass", "Password", ansi.Truncate(passView, w+1), onPass, th),
		layout.FlexChild{Node: layout.Named("reveal", layout.Block(rv.Render(word)))},
	)
	box := "[ ]"
	if m.remember {
		box = "[x]"
	}
	rem := ansi.NewStyle()
	if m.focus == onRemember {
		rem = ansi.NewStyle().Bold().Foreground(th.Primary)
	}
	return []layout.FlexChild{
		m.field("user", "Username", ansi.Truncate(m.user.View(), w+1), onUser, th),
		{Node: passRow},
		{Node: layout.Block(m.notice(th))},
		{Node: layout.Named("remember", layout.Block(rem.Render(box+" Remember me")))},
	}
}

// notice is the one line under the fields: an error, a lock countdown, the
// caps hint, or nothing. A mark leads each so colour is not the only signal.
func (m model) notice(th theme.Theme) string {
	var s string
	switch {
	case m.lock > 0:
		s = ansi.NewStyle().Foreground(th.Error).Render(fmt.Sprintf("✗ Too many tries. Try again in %ds", m.lock))
	case m.err != "":
		s = ansi.NewStyle().Foreground(th.Error).Render("✗ " + m.err)
	case m.capsHint() && m.focus == onPass:
		s = ansi.NewStyle().Foreground(th.Warning).Render("⚠ Caps Lock may be on")
	}
	return ansi.Truncate(s, m.inner())
}

func (m model) button(th theme.Theme) layout.Node {
	label, st := " Sign in ", ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Primary)
	switch {
	case m.busy:
		label, st = " "+m.spin.View()+" Signing in… ", ansi.NewStyle().Faint()
		if m.reduced {
			label = " Signing in… "
		}
	case m.lock > 0:
		st = ansi.NewStyle().Faint()
	case m.hover == "button":
		st = st.Underline()
	case m.focus == onButton:
		st = st.Underline()
	}
	return layout.Named("button", layout.Block(st.Render(label)))
}

func (m model) form(th theme.Theme) []layout.FlexChild {
	out := m.fields(th)
	out = append(out, layout.FlexChild{Node: m.button(th)})
	if !m.compact() {
		f := widgets.Footer(m.inner(),
			widgets.Hint{Key: "tab", Action: "next"}, widgets.Hint{Key: "enter", Action: "sign in"},
			widgets.Hint{Key: "ctrl+t", Action: "show"}, widgets.Hint{Key: "f1", Action: "help"})
		out = append(out, block(ansi.NewStyle().Faint().Render(f)))
	}
	return out
}

func (m model) welcome(th theme.Theme) []layout.FlexChild {
	who := strings.TrimSpace(m.user.Value())
	note := "You will be asked for your username next time."
	if m.remName != "" {
		note = "Your username is remembered for next time."
	}
	so := ansi.NewStyle().Foreground(th.Primary)
	if m.hover == "signout" {
		so = so.Underline().Bold()
	}
	return []layout.FlexChild{
		block(ansi.NewStyle().Bold().Foreground(th.Success).Render("✓ Signed in")),
		block(ansi.Truncate("Welcome back, "+who+".", m.inner())),
		block(ansi.NewStyle().Faint().Render(ansi.Truncate(note, m.inner()))),
		{Node: layout.Named("signout", layout.Block(so.Render("[ Sign out ]")))},
		block(ansi.NewStyle().Faint().Render(widgets.Footer(m.inner(),
			widgets.Hint{Key: "enter", Action: "sign out"}, widgets.Hint{Key: "q", Action: "quit"}))),
	}
}

func (m model) helpBody(th theme.Theme) []layout.FlexChild {
	key := func(k, a string) layout.FlexChild {
		return block(ansi.Truncate(ansi.NewStyle().Bold().Foreground(th.Primary).Render(fmt.Sprintf("%-12s", k))+a, m.inner()))
	}
	return []layout.FlexChild{
		key("Tab", "next field"), key("Shift+Tab", "previous field"), key("Enter", "sign in"),
		key("Space", "tick the box"), key("Ctrl+T", "show or hide"), key("Esc", "clear the password"),
		key("F1", "close help"), key("Ctrl+C", "quit"),
	}
}

func (m model) screen() layout.Node {
	th := m.theme()
	var body []layout.FlexChild
	switch {
	case m.help:
		body = m.helpBody(th)
	case m.phase == signedIn:
		body = m.welcome(th)
	default:
		body = m.form(th)
	}
	gap := 1
	if m.compact() {
		gap = 0
	}
	if !m.compact() && !m.help && m.phase == signingIn {
		head := []layout.FlexChild{
			block(ansi.NewStyle().Bold().Foreground(th.Primary).Render("Sign in")),
			block(ansi.NewStyle().Faint().Render(ansi.Truncate("Demo account: "+demoUser+" / "+demoPassword, m.inner()))),
		}
		body = append(head, body...)
	}
	col := layout.Fixed(layout.Column(gap, body...), layout.Size{W: m.inner()})
	dx := 0
	if m.shake < len(shakeOffsets) {
		dx = shakeOffsets[m.shake]
	}
	if m.compact() {
		return layout.Stack(layout.Anchor(layout.Center, dx, 0, col))
	}
	box := layout.NewBox().Border(th.Border).Padding(1, 2, 1, 2)
	return layout.Stack(layout.Anchor(layout.Center, dx, 0, layout.BoxNode(box, col)))
}

func (m model) View() string { return layout.DrawTight(m.screen(), m.size()) }

func main() {
	opts := []tui.ProgramOption{tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion)}
	// ReducedMotion is known as soon as the Program exists, before Run.
	p := ui.NewProgram(newModel(false), opts...)
	if p.ReducedMotion() {
		p = ui.NewProgram(newModel(true), opts...)
	}
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
