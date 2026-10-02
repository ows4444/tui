// Command login is a credential sign-in screen built on package form, with a
// live design-system panel beside it for trying every theming control the
// library has.
//
// The login panel has a username field, a masked password field and a
// "remember me" checkbox: Tab and Shift+Tab move between them, Space ticks
// the box, Enter signs in (Esc quits). Submitting a valid form starts a
// simulated authentication round-trip — a tui.Tick standing in for a network
// call — with a spinner in place of the form. Wrong credentials show an error
// alert, keep the username and clear the password; three failures lock the
// screen. The demo account is demoUser / demoPassword; there is no real
// credential storage here.
//
// Ctrl+T (or F2) moves the keyboard to the design panel (see design.go):
// Up/Down pick a setting, Left/Right change it, r resets, and Ctrl+T, F2 or
// Esc go back to the form. Every change rebuilds one theme.Theme — preset,
// light twin, border, border colour, component accent tokens, glyph set,
// colour depth, spacing, title typography and panel fill — and re-themes the
// form, spinner, alerts and boxes with it. On a terminal too narrow for both
// panels only the one with the keyboard is shown.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/form"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/motion"
	"github.com/ows4444/tui/spinner"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

const (
	// width is the content width of the login panel, or what is left of a
	// narrower terminal (see model.contentWidth).
	width = 44

	demoUser     = "ada"
	demoPassword = "lovelace"

	// maxAttempts is how many wrong passwords lock the screen.
	maxAttempts = 3
)

// authDelay stands in for network latency; tests set it to zero.
var authDelay = 800 * time.Millisecond

// state is which screen the login panel is on.
type state int

const (
	editing state = iota
	authenticating
	signedIn
	locked
)

// authResultMsg is the answer of the simulated authentication call.
type authResultMsg struct {
	user     string
	remember bool
	ok       bool
}

// authenticate checks the credentials after authDelay. A real app would call
// its backend here and return the result as a Msg the same way.
func authenticate(user, password string, remember bool) tui.Cmd {
	ok := user == demoUser && password == demoPassword
	res := authResultMsg{user: user, remember: remember, ok: ok}
	if authDelay <= 0 {
		return func() tui.Msg { return res }
	}
	return tui.FromCtx(motion.After(authDelay, func(time.Time) tui.Msg { return res }))
}

type model struct {
	t     theme.Theme // design.theme(), cached by restyle
	form  form.Model
	spin  spinner.Model
	state state

	design    design
	designing bool // the design panel has the keyboard

	user     string // the signed-in (or last tried) username
	remember bool
	failures int
	errMsg   string // last authentication error, "" when none

	termWidth, termHeight int // terminal size from the last ResizeMsg; 0 until known

	// focusCmd is the form's initial Focus Cmd, handed back by Init: Init can
	// only return a Cmd, so the Focus itself happens in initialModel.
	focusCmd tui.Cmd
}

// newForm builds the sign-in form, prefilled with user and remember.
func newForm(user string, remember bool) form.Model {
	rem := "false"
	if remember {
		rem = "true"
	}
	return form.New(
		form.Field{Name: "user", Label: "Username", Placeholder: demoUser, Value: user,
			Validators: []form.Validator{form.Required()}},
		form.Field{Name: "password", Label: "Password", Secret: true,
			Validators: []form.Validator{form.Required()}},
		form.Field{Name: "remember", Label: "Remember me", Kind: form.FieldCheckbox, Value: rem},
	)
}

func initialModel() model {
	sp := spinner.New()
	sp.Label = "Signing in…"
	m := model{form: newForm("", false), spin: sp}
	m.restyle()
	m.focusCmd = m.form.Focus()
	return m
}

// restyle rebuilds the theme from the design settings and hands it to every
// themed widget.
func (m *model) restyle() {
	m.t = m.design.theme()
	m.form = m.form.SetTheme(m.t)
	m.spin = m.spin.SetTheme(m.t)
}

func (m model) Init() tui.Cmd { return m.focusCmd }

// isToggle reports whether k moves the keyboard between the two panels.
func isToggle(k tui.Key) bool {
	return k.Type == tui.KeyF2 || (k.Type == tui.KeyCtrl && k.Code == 't')
}

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.termWidth, m.termHeight = msg.Width, msg.Height
		return m, nil
	case tui.Key:
		if msg.Type == tui.KeyCtrlC {
			return m, tui.Quit()
		}
		if isToggle(msg) || (m.designing && msg.Type == tui.KeyEsc) {
			return m.toggleDesign()
		}
		if m.designing {
			return m.updateDesign(msg), nil
		}
		if msg.Type == tui.KeyEsc {
			return m, tui.Quit()
		}
		if m.state == signedIn || m.state == locked {
			if msg.Type == tui.KeyEnter || (msg.Type == tui.KeyRunes && msg.Text == "q") {
				return m, tui.Quit()
			}
			return m, nil
		}
		if m.state == authenticating {
			return m, nil // ignore typing while the request is in flight
		}
	case form.SubmittedMsg:
		m.state = authenticating
		m.errMsg = ""
		m.user, m.remember = msg.Values["user"], msg.Values["remember"] == "true"
		m.form.Blur()
		return m, tui.Batch(m.spin.Start(),
			authenticate(m.user, msg.Values["password"], m.remember))
	case authResultMsg:
		m.spin.Stop()
		if msg.ok {
			m.state = signedIn
			return m, nil
		}
		m.failures++
		if m.failures >= maxAttempts {
			m.state = locked
			return m, nil
		}
		m.state = editing
		m.errMsg = fmt.Sprintf("Invalid username or password (%d of %d attempts)",
			m.failures, maxAttempts)
		// Rebuild the form to clear the password, keep the username, and move
		// focus on to the password field.
		m.form = newForm(m.user, m.remember).SetTheme(m.t)
		if m.designing {
			return m, nil
		}
		focus := m.form.Focus()
		var next tui.Cmd
		m.form, next = m.form.Update(tui.Key{Type: tui.KeyTab})
		return m, tui.Batch(focus, next)
	}

	if m.state == authenticating {
		var cmd tui.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd
	}
	if m.state != editing || m.designing {
		return m, nil
	}
	var cmd tui.Cmd
	m.form, cmd = m.form.Update(msg)
	return m, cmd
}

// toggleDesign moves the keyboard between the form and the design panel.
func (m model) toggleDesign() (tui.Model, tui.Cmd) {
	m.designing = !m.designing
	if m.designing {
		m.form.Blur()
		return m, nil
	}
	if m.state == editing {
		return m, m.form.Focus()
	}
	return m, nil
}

// updateDesign handles a key while the design panel has the keyboard.
func (m model) updateDesign(k tui.Key) model {
	switch {
	case k.Type == tui.KeyUp || (k.Type == tui.KeyTab && k.Mod.Shift()):
		m.design = m.design.move(-1)
	case k.Type == tui.KeyDown || k.Type == tui.KeyTab:
		m.design = m.design.move(1)
	case k.Type == tui.KeyLeft:
		m.design = m.design.change(-1)
	case k.Type == tui.KeyRight || k.Type == tui.KeySpace || k.Type == tui.KeyEnter:
		m.design = m.design.change(1)
	case k.Type == tui.KeyRunes && k.Text == "r":
		m.design = design{cursor: m.design.cursor}
	default:
		return m
	}
	m.restyle()
	return m
}

// contentWidth is the login panel's content width: width, or what is left of
// the terminal once the border and padding are paid for.
func (m model) contentWidth() int {
	if m.termWidth > 0 {
		return max(10, min(width, m.termWidth-2-2*m.design.padding(m.t)))
	}
	return width
}

// compact reports whether the terminal is too short for blank rows between
// sections and padding inside the box.
func (m model) compact() bool { return m.termHeight > 0 && m.termHeight < 14 }

func (m model) gap() int {
	if m.compact() {
		return 0
	}
	return 1
}

// sideBySide reports whether both panels fit next to each other.
func (m model) sideBySide() bool {
	pad := 2 + 2*m.design.padding(m.t)
	return m.termWidth == 0 || m.termWidth >= width+designWidth+2*pad+2
}

// hints is the key legend: with words when they fit in w, else the keys alone.
func hints(w int, hs ...widgets.Hint) string {
	full := widgets.KeyHints("  ", hs...)
	if ansi.Width(full) <= w {
		return full
	}
	for i := range hs {
		hs[i].Action = ""
	}
	return ansi.Truncate(widgets.KeyHints(" ", hs...), w)
}

func block(s string) layout.FlexChild { return layout.FlexChild{Node: layout.Block(s)} }

// title is the login panel's heading in the chosen typography, or "" when
// the title is drawn on the border instead.
func (m model) title() string {
	ty := m.t.ResolvedTypography()
	switch m.design.pick[setTitle] {
	case titleH2:
		return ty.H2.Render("Sign in")
	case titleBig:
		return widgets.BigText("LOGIN", widgets.FontSlim, m.t)
	case titleBorder:
		return ""
	}
	return ty.H1.Render("Sign in")
}

// body is the login panel's content for the current state.
func (m model) body() layout.Node {
	w := m.contentWidth()
	muted := ansi.NewStyle().Foreground(m.t.Muted)
	switch m.state {
	case signedIn:
		note := "Session ends when you quit."
		if m.remember {
			note = "You'll stay signed in on this device."
		}
		return layout.Column(m.gap(),
			block(widgets.Alert("Welcome back, "+m.user+"!", widgets.VariantSuccess, m.t, w)),
			block(muted.Render(ansi.Truncate(note, w))),
			block(hints(w, widgets.Hint{Key: "enter", Action: "quit"})))
	case locked:
		return layout.Column(m.gap(),
			block(widgets.Alert("Too many failed attempts. Try again later.", widgets.VariantError, m.t, w)),
			block(hints(w, widgets.Hint{Key: "enter", Action: "quit"})))
	}

	var children []layout.FlexChild
	if t := m.title(); t != "" {
		children = append(children, block(t))
	}
	children = append(children,
		block(muted.Render(ansi.Truncate("demo: "+demoUser+" / "+demoPassword, w))))
	if m.errMsg != "" {
		children = append(children, block(widgets.Alert(m.errMsg, widgets.VariantError, m.t, w)))
	}
	if m.state == authenticating {
		children = append(children, layout.FlexChild{Node: m.spin.LayoutNode()})
	} else {
		children = append(children,
			layout.FlexChild{Node: m.form.LayoutNode()},
			block(hints(w,
				widgets.Hint{Key: "tab", Action: "next"},
				widgets.Hint{Key: "enter", Action: "sign in"},
				widgets.Hint{Key: "ctrl+t", Action: "design"})))
	}
	return layout.Column(m.gap(), children...)
}

// frame wraps content in a box drawn with the theme's border, padding and
// fill. The panel with the keyboard has its border in the Focus colour.
func (m model) frame(content layout.Node, w int, active bool, title string) layout.Node {
	pad := m.design.padding(m.t)
	box := layout.NewBox().Border(m.t.Border).Padding(m.gap(), pad, m.gap(), pad)
	if active && m.sideBySide() {
		box = box.BorderColor(m.t.Focus)
	} else {
		box = box.BorderColor(m.t.BorderColor)
	}
	if c := m.design.fill(m.t); c != nil {
		box = box.Background(c)
	}
	if title != "" {
		box = box.Title(" "+title+" ", layout.AlignCenter)
	}
	return layout.BoxNode(box, layout.Fixed(content, layout.Size{W: w}))
}

func (m model) loginPanel() layout.Node {
	title := ""
	if m.design.pick[setTitle] == titleBorder {
		title = "Sign in"
	}
	return m.frame(m.body(), m.contentWidth(), !m.designing, title)
}

// designContentWidth is designWidth, or what is left of a narrower terminal.
func (m model) designContentWidth() int {
	if m.termWidth > 0 && !m.sideBySide() {
		return max(10, min(designWidth, m.termWidth-2-2*m.design.padding(m.t)))
	}
	return designWidth
}

func (m model) designPanel() layout.Node {
	w := m.designContentWidth()
	h := hints(w,
		widgets.Hint{Key: "↑↓", Action: "pick"},
		widgets.Hint{Key: "←→", Action: "change"}) + "\n" + hints(w,
		widgets.Hint{Key: "r", Action: "reset"},
		widgets.Hint{Key: "ctrl+t", Action: "back to form"})
	if !m.designing {
		h = hints(w, widgets.Hint{Key: "ctrl+t", Action: "edit design"})
	}
	return m.frame(layout.Block(m.design.panel(m.t, m.designing, h, w)), w, m.designing, "")
}

// screen is both panels side by side, or the one with the keyboard when the
// terminal is too narrow for both.
func (m model) screen() layout.Node {
	if m.sideBySide() {
		return layout.Row(2,
			layout.FlexChild{Node: m.loginPanel()},
			layout.FlexChild{Node: m.designPanel()})
	}
	if m.designing {
		return m.designPanel()
	}
	return m.loginPanel()
}

func (m model) View() string { return layout.Draw(m.screen(), layout.Unconstrained()) }

func main() {
	if _, err := tui.NewProgram(initialModel()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
