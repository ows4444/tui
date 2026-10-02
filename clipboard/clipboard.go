// Package clipboard is an interactive "copy to clipboard" button — Enter or
// Space activates it, writing an OSC52 clipboard-set escape sequence for the
// configured Text and showing a timed "Copied!" confirmation that reverts
// automatically. Reuses toast.Model's exact tui.Tick-driven auto-dismiss
// pattern with its id-disambiguation trick, so repeated activations don't
// race: each activation bumps an internal id, and a stale timer's message
// carrying an old id is ignored by Update.
//
// OSC52 support is write-only and best-effort: it depends on the terminal
// emulator recognizing the sequence, and under tmux it additionally
// depends on the "set-clipboard" option (on or external) in tmux.conf; with
// "off" tmux drops the escape sequence before it reaches the outer terminal, so
// Write appears to succeed (View shows "Copied!") but nothing lands on the
// system clipboard. There's no way to detect this from inside the program;
// it's a deployment/configuration caveat for anyone running under tmux, not
// a bug this package can work around.
//
// Stability: experimental. Its API may change in any minor release.
package clipboard

import (
	"os"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/motion"
	"github.com/ows4444/tui/theme"
)

// Model is a button that copies Text to the system clipboard via OSC52 when
// activated, showing Label normally and a "Copied!" confirmation for
// Timeout after activation.
type Model struct {
	// Text is the string copied to the clipboard on activation. Activating
	// with an empty Text is a no-op: nothing is written and no confirmation
	// is shown.
	Text string
	// Label is shown in View when not displaying the "Copied!" confirmation.
	Label string
	Theme theme.Theme
	// Timeout is how long the "Copied!" confirmation is shown before View
	// reverts to Label. Defaults to ~2000ms via New.
	Timeout time.Duration
	// Write is called with the OSC52 escape sequence on activation. Defaults
	// to os.Stdout.WriteString via New; tests can substitute their own to
	// capture the written bytes without touching a real terminal.
	Write func(string) (int, error)

	// Mouse, when true, makes Update handle tui.MouseEvent: a left click inside
	// Bounds activates the button as Enter does. Off (the default) ignores the
	// mouse.
	Mouse bool
	// Bounds is the screen rectangle where the app draws the button; clicks
	// outside it are ignored.
	Bounds hittest.Rect

	copied bool
	// id disambiguates activations the same way toast.Model's id does: each
	// activation starts a new dismiss timer carrying the id current at the
	// time. If activated again before a previous timer fires, that stale
	// timer's dismissMsg carries an id that no longer matches m.id, so
	// Update ignores it instead of clearing the new confirmation early.
	id int
}

// New returns a Model with Timeout defaulting to 2 seconds and Write
// defaulting to os.Stdout.WriteString.
func New(text, label string) Model {
	return Model{
		Text:    text,
		Label:   label,
		Theme:   theme.DarkTheme(),
		Timeout: 2000 * time.Millisecond,
		Write:   os.Stdout.WriteString,
	}
}

// Copied reports whether the "Copied!" confirmation is currently shown.
func (m Model) Copied() bool { return m.copied }

type dismissMsg struct{ id int }

// Update activates the clipboard copy on Enter or Space: with a non-empty
// Text it writes the OSC52 clipboard-set escape sequence via Write, shows
// the "Copied!" confirmation, and returns a Cmd that clears it after
// Timeout (ignored if a later activation has since started a newer timer).
// Activating with an empty Text is a no-op — no write, no confirmation. Any
// other Msg is a no-op except a matching dismissMsg, which clears the
// confirmation. With Mouse on, a left click inside Bounds activates it like Enter.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.MouseEvent:
		if !m.Mouse || msg.Action != tui.MouseActionPress || msg.Button != tui.MouseButtonLeft ||
			!m.Bounds.Contains(msg.X, msg.Y) {
			return m, nil
		}
		return m.activate()
	case tui.Key:
		if msg.Type != tui.KeyEnter && msg.Type != tui.KeySpace {
			return m, nil
		}
		return m.activate()
	case dismissMsg:
		if msg.id != m.id {
			return m, nil
		}
		m.copied = false
		return m, nil
	}
	return m, nil
}

// activate copies Text and starts the "Copied!" timer; a no-op with empty Text.
func (m Model) activate() (Model, tui.Cmd) {
	if m.Text == "" {
		return m, nil
	}
	if m.Write != nil {
		_, _ = m.Write(ansi.OSC52Copy(m.Text))
	}
	m.copied = true
	m.id++
	id := m.id
	return m, tui.FromCtx(motion.After(m.Timeout, func(time.Time) tui.Msg { return dismissMsg{id: id} }))
}

// View renders Label, or a "Copied!" confirmation in its place while the
// post-activation timeout hasn't yet elapsed.
func (m Model) View() string {
	if m.copied {
		return "Copied!"
	}
	return m.Label
}

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}
