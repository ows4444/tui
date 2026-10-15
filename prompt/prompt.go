// Package prompt asks one question at the terminal and returns the answer,
// without a Model, Update or View of your own:
//
//	name, err := prompt.Input("Your name")
//	ok, err := prompt.Confirm("Deploy?", false)
//	i, err := prompt.Select("Region", []string{"eu", "us", "ap"})
//
// Each call runs a small inline program (no alternate screen), shows the
// question under the cursor, and returns when it is answered. Esc or Ctrl+C
// returns ErrCanceled. The prompts are built from textinput, passwordinput,
// confirm, picker, multiselect and spinner, so they handle paste, wide text,
// the mouse where it is turned on, and accessible output the same way.
//
// Stability: experimental. Its API may change in any minor release.
package prompt

import (
	"context"
	"errors"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/confirm"
	"github.com/ows4444/tui/multiselect"
	"github.com/ows4444/tui/passwordinput"
	"github.com/ows4444/tui/picker"
	"github.com/ows4444/tui/spinner"
	"github.com/ows4444/tui/textinput"
)

// ErrCanceled is returned when the person answers with Esc or Ctrl+C.
var ErrCanceled = errors.New("prompt: canceled")

// Config adjusts one prompt. Every field is optional, and a prompt takes at
// most one Config as its last argument.
type Config struct {
	// Program is passed to the tui.Program that runs the prompt: a different
	// input or output (tests), a colour profile, accessible mode.
	Program []tui.ProgramOption
	// Placeholder is the hint shown while Input or Password is empty.
	Placeholder string
	// Default is the text Input or Password starts with.
	Default string
	// Validate, when set, makes Input and Password refuse an answer it
	// rejects: the error text is shown under the question and the prompt stays
	// open.
	Validate func(string) error
	// NoFilter turns off typing to narrow Select and MultiSelect.
	NoFilter bool
}

type config = Config

func newConfig(cfg []Config) config {
	if len(cfg) > 0 {
		return cfg[0]
	}
	return Config{}
}

func run(m tui.Model, c config) (tui.Model, error) {
	return tui.NewProgram(m, c.Program...).Run()
}

// answer runs the Cmd a component returned for an answering key, and reports
// the message it carries. The widgets' answer Cmds are plain closures that
// return at once; reading them here, instead of waiting for the message to
// come back through the queue, means a prompt is answered even when its input
// ends right after the key (a pipe, a test).
func answer(cmd tui.Cmd) tui.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

func isKey(msg tui.Msg, t tui.KeyType) bool {
	k, ok := msg.(tui.Key)
	return ok && k.Type == t
}

func isCancel(msg tui.Msg) bool {
	k, ok := msg.(tui.Key)
	return ok && (k.Type == tui.KeyCtrlC || k.Type == tui.KeyEsc)
}

// ---- Input and Password ----

type textModel struct {
	label    string
	secret   bool
	in       textinput.Model
	pw       passwordinput.Model
	validate func(string) error
	err      string
	value    string
	done     bool
	canceled bool
	start    tui.Cmd
}

func (m *textModel) text() string {
	if m.secret {
		return m.pw.Value()
	}
	return m.in.Value()
}

func (m textModel) Init() tui.Cmd { return m.start }

func (m textModel) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	if isCancel(msg) {
		m.canceled = true
		return m, tui.Quit()
	}
	if k, ok := msg.(tui.Key); ok && k.Type == tui.KeyEnter {
		v := m.text()
		if m.validate != nil {
			if err := m.validate(v); err != nil {
				m.err = err.Error()
				return m, nil
			}
		}
		m.value, m.done = v, true
		return m, tui.Quit()
	}
	var cmd tui.Cmd
	if m.secret {
		m.pw, cmd = m.pw.Update(msg)
	} else {
		m.in, cmd = m.in.Update(msg)
	}
	if _, ok := msg.(tui.Key); ok {
		m.err = ""
	}
	return m, cmd
}

func (m textModel) View() string {
	v := m.in.View()
	if m.secret {
		v = m.pw.View()
	}
	if m.err != "" {
		v += "\n  " + ansi.NewStyle().Foreground(ansi.Red).Render(ansi.Sanitize(m.err))
	}
	return v
}

func askText(label string, secret bool, cfg []Config) (string, error) {
	c := newConfig(cfg)
	m := textModel{label: label, secret: secret, validate: c.Validate}
	prompt := ansi.Sanitize(label) + " "
	if secret {
		m.pw = passwordinput.New()
		m.pw.Prompt, m.pw.Placeholder = prompt, c.Placeholder
		m.pw.SetValue(c.Default)
		m.start = m.pw.Focus()
	} else {
		m.in = textinput.New()
		m.in.Prompt, m.in.Placeholder = prompt, c.Placeholder
		m.in.SetValue(c.Default)
		m.start = m.in.Focus()
	}
	out, err := run(m, c)
	if err != nil {
		return "", err
	}
	res := out.(textModel)
	if res.canceled || !res.done {
		return "", ErrCanceled
	}
	return res.value, nil
}

// Input asks for a line of text and returns it. An empty answer is valid
// unless WithValidate refuses it.
func Input(label string, cfg ...Config) (string, error) { return askText(label, false, cfg) }

// Password asks for a line of text and shows it masked.
func Password(label string, cfg ...Config) (string, error) { return askText(label, true, cfg) }

// ---- Confirm ----

type confirmModel struct {
	c        confirm.Model
	yes      bool
	done     bool
	canceled bool
}

func (m confirmModel) Init() tui.Cmd { return nil }

func (m confirmModel) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	if isCancel(msg) {
		m.canceled = true
		return m, tui.Quit()
	}
	var cmd tui.Cmd
	m.c, cmd = m.c.Update(msg)
	if k, ok := msg.(tui.Key); ok && cmd != nil && (k.Type == tui.KeyEnter || k.Type == tui.KeyRunes) {
		if cm, ok := answer(cmd).(confirm.ConfirmedMsg); ok {
			m.yes, m.done = cm.Yes, true
			return m, tui.Quit()
		}
	}
	return m, cmd
}

func (m confirmModel) View() string { return m.c.View() }

// Confirm asks a yes or no question. def is the answer Enter gives; y and n
// answer directly.
func Confirm(label string, def bool, cfg ...Config) (bool, error) {
	c := newConfig(cfg)
	cm := confirm.New(ansi.Sanitize(label))
	if cm.Highlighted() != def {
		cm, _ = cm.Update(tui.Key{Type: tui.KeyTab}) // Tab toggles the highlight
	}
	out, err := run(confirmModel{c: cm}, c)
	if err != nil {
		return false, err
	}
	res := out.(confirmModel)
	if res.canceled || !res.done {
		return false, ErrCanceled
	}
	return res.yes, nil
}

// ---- Select and MultiSelect ----

type selectModel struct {
	label    string
	p        picker.Model
	index    int
	done     bool
	canceled bool
}

func (m selectModel) Init() tui.Cmd { return nil }

func (m selectModel) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	// Esc clears a query first; with none it cancels.
	if k, ok := msg.(tui.Key); ok && (k.Type == tui.KeyCtrlC || (k.Type == tui.KeyEsc && m.p.Query() == "")) {
		m.canceled = true
		return m, tui.Quit()
	}
	var cmd tui.Cmd
	m.p, cmd = m.p.Update(msg)
	if isKey(msg, tui.KeyEnter) && cmd != nil {
		if sm, ok := answer(cmd).(picker.SelectedMsg); ok {
			m.index, m.done = sm.Index, true
			return m, tui.Quit()
		}
	}
	return m, cmd
}

func (m selectModel) View() string { return ansi.Sanitize(m.label) + "\n" + m.p.View() }

// Select asks the person to pick one of choices and returns its index. Typing
// narrows the list; Enter picks the one under the cursor.
func Select(label string, choices []string, cfg ...Config) (int, error) {
	c := newConfig(cfg)
	if len(choices) == 0 {
		return -1, errors.New("prompt: Select needs at least one choice")
	}
	p := picker.NewStrings(choices...)
	p.Filter = !c.NoFilter
	out, err := run(selectModel{label: label, p: p}, c)
	if err != nil {
		return -1, err
	}
	res := out.(selectModel)
	if res.canceled || !res.done {
		return -1, ErrCanceled
	}
	return res.index, nil
}

type multiModel struct {
	label    string
	m        multiselect.Model
	picked   []int
	done     bool
	canceled bool
}

func (m multiModel) Init() tui.Cmd { return nil }

func (m multiModel) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	if k, ok := msg.(tui.Key); ok && (k.Type == tui.KeyCtrlC || (k.Type == tui.KeyEsc && m.m.Query() == "")) {
		m.canceled = true
		return m, tui.Quit()
	}
	var cmd tui.Cmd
	m.m, cmd = m.m.Update(msg)
	if isKey(msg, tui.KeyEnter) && cmd != nil {
		if _, ok := answer(cmd).(multiselect.ConfirmedMsg); ok {
			m.picked, m.done = m.m.SelectedIndexes(), true
			return m, tui.Quit()
		}
	}
	return m, cmd
}

func (m multiModel) View() string { return ansi.Sanitize(m.label) + "\n" + m.m.View() }

// MultiSelect asks the person to tick any of choices (Space) and confirm
// (Enter). It returns the indexes ticked, ascending; none is a valid answer.
func MultiSelect(label string, choices []string, cfg ...Config) ([]int, error) {
	c := newConfig(cfg)
	if len(choices) == 0 {
		return nil, errors.New("prompt: MultiSelect needs at least one choice")
	}
	ms := multiselect.NewStrings(choices...)
	ms.Filter = !c.NoFilter
	out, err := run(multiModel{label: label, m: ms}, c)
	if err != nil {
		return nil, err
	}
	res := out.(multiModel)
	if res.canceled || !res.done {
		return nil, ErrCanceled
	}
	return res.picked, nil
}

// ---- Spin ----

type spinModel struct {
	label  string
	sp     spinner.Model
	fn     func(context.Context) error
	cancel context.CancelFunc
	ctx    context.Context
	err    error
	done   bool
	start  tui.Cmd
}

type spinDone struct{ err error }

func (m spinModel) Init() tui.Cmd {
	work := tui.Go(func(context.Context) tui.Msg { return spinDone{m.fn(m.ctx)} })
	return tui.Batch(m.start, work)
}

func (m spinModel) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	if d, ok := msg.(spinDone); ok {
		m.err, m.done = d.err, true
		m.sp.Stop()
		return m, tui.Quit()
	}
	if k, ok := msg.(tui.Key); ok && k.Type == tui.KeyCtrlC {
		m.cancel()
		return m, nil // fn returns, and spinDone quits
	}
	var cmd tui.Cmd
	m.sp, cmd = m.sp.Update(msg)
	return m, cmd
}

func (m spinModel) View() string { return m.sp.View() + " " + ansi.Sanitize(m.label) }

// Spin shows a spinner and label while fn runs, then returns fn's error. fn is
// given a context that is cancelled when the person presses Ctrl+C; it should
// return promptly when that happens.
func Spin(label string, fn func(ctx context.Context) error, cfg ...Config) error {
	c := newConfig(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sp := spinner.New()
	m := spinModel{label: label, sp: sp, fn: fn, ctx: ctx, cancel: cancel}
	m.start = m.sp.Start()
	out, err := run(m, c)
	if err != nil {
		return err
	}
	res := out.(spinModel)
	if !res.done {
		return ErrCanceled
	}
	return res.err
}
