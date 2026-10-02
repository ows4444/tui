// Package errorretry is a small interactive widget for showing an error
// with a retry/dismiss keybinding — Enter or 'r' retries (up to MaxRetries),
// Esc always dismisses.
//
// Stability: experimental. Its API may change in any minor release.
package errorretry

import (
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/input"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/theme"
)

// Model shows an error Message and lets the user retry (Enter or 'r', up
// to MaxRetries times) or dismiss (Esc). Retrying past MaxRetries is a
// no-op; Esc always works regardless of retryCount.
type Model struct {
	Message    string
	MaxRetries int
	Theme      theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
	// KeyMap holds the keys for each action. New fills it with
	// DefaultKeyMap; a Model built as a struct literal with a zero KeyMap
	// behaves as if it held DefaultKeyMap. The hint line View draws names the
	// default keys.
	KeyMap KeyMap

	// Mouse, when true, makes Update handle tui.MouseEvent: a left click on the
	// "retry" part of the hint line retries and on the "dismiss" part dismisses,
	// as the keys would. Off (the default) ignores the mouse.
	Mouse bool
	// Bounds is the screen rectangle where the app draws the widget (its first
	// row is the first line of Message); clicks outside it are ignored.
	Bounds hittest.Rect

	retryCount int // number of retries already requested
}

// New returns a Model for message, allowing up to maxRetries retries.
func New(message string, maxRetries int) Model {
	return Model{Message: message, MaxRetries: maxRetries, Theme: theme.DarkTheme(), KeyMap: DefaultKeyMap()}
}

// KeyMap names the keys of each action of a Model.
type KeyMap struct {
	Retry   keymap.Binding // request a retry while retries remain
	Dismiss keymap.Binding // dismiss the error
}

// DefaultKeyMap returns the keys a Model used before KeyMap existed.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Retry:   keymap.NewBinding("retry", "enter", "r", "R"),
		Dismiss: keymap.NewBinding("dismiss", "esc"),
	}
}

func (m Model) keys() KeyMap {
	if len(m.KeyMap.Retry.Keys)+len(m.KeyMap.Dismiss.Keys) == 0 {
		return DefaultKeyMap()
	}
	return m.KeyMap
}

// Bindings returns the actions the widget currently honours, with
// descriptions, for help text. Retry is left out once retries are exhausted.
func (m Model) Bindings() []keymap.Binding {
	km := m.keys()
	if m.Exhausted() {
		return []keymap.Binding{km.Dismiss}
	}
	return []keymap.Binding{km.Retry, km.Dismiss}
}

// hit reports whether msg triggers b, ignoring Shift so a shifted letter
// such as a capital 'R' still matches as it did before KeyMap existed.
func hit(msg tui.Msg, b keymap.Binding) bool {
	if keymap.Matches(msg, b) {
		return true
	}
	if k, ok := msg.(tui.Key); ok && k.Mod.Shift() {
		k.Mod &^= input.ModShift
		return keymap.Matches(k, b)
	}
	return false
}

// RetryCount reports how many retries have been requested so far.
func (m Model) RetryCount() int { return m.retryCount }

// Exhausted reports whether retries have been used up — further Enter/'r'
// presses are a no-op once true.
func (m Model) Exhausted() bool { return m.retryCount >= m.MaxRetries }

// RetryMsg is delivered (via the Cmd Update returns) when the user
// requests a retry while retries remain.
type RetryMsg struct{}

// DismissedMsg is delivered (via the Cmd Update returns) when the user
// dismisses the error with Esc.
type DismissedMsg struct{}

// Update handles Enter/'r' (retry, while retryCount < MaxRetries) and Esc
// (dismiss, always). Any other Msg, or Enter/'r' once retries are
// exhausted, is a no-op. With Mouse on, a left click on the retry or dismiss
// part of the hint line does the same as the key.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if ev, ok := msg.(tui.MouseEvent); ok {
		return m.updateMouse(ev)
	}
	km := m.keys()
	switch {
	case hit(msg, km.Dismiss):
		return m, func() tui.Msg { return DismissedMsg{} }
	case hit(msg, km.Retry):
		return m.retry()
	}
	return m, nil
}

// updateMouse retries or dismisses when a left press lands on the matching
// part of the hint line, the row under Message.
func (m Model) updateMouse(ev tui.MouseEvent) (Model, tui.Cmd) {
	if !m.Mouse || ev.Action != tui.MouseActionPress || ev.Button != tui.MouseButtonLeft ||
		!m.Bounds.Contains(ev.X, ev.Y) {
		return m, nil
	}
	lx, ly := m.Bounds.Local(ev.X, ev.Y)
	if ly != strings.Count(m.Message, "\n")+1 {
		return m, nil
	}
	retryPart, dismissPart := m.hintParts()
	rw := ansi.Width(retryPart)
	switch {
	case rw > 0 && lx < rw:
		return m.retry()
	case lx >= rw && lx < rw+ansi.Width(dismissPart):
		return m, func() tui.Msg { return DismissedMsg{} }
	}
	return m, nil
}

// retry increments retryCount and emits RetryMsg, unless retries are
// already exhausted, in which case it's a no-op.
func (m Model) retry() (Model, tui.Cmd) {
	if m.Exhausted() {
		return m, nil
	}
	m.retryCount++
	return m, func() tui.Msg { return RetryMsg{} }
}

// View renders the error message and a hint line. The hint reflects
// whether retries remain: "press Enter/r to retry" while they do, and a
// visibly distinct "no retries left" (dimmed, no retry mention) once
// exhausted — Esc to dismiss is always offered either way.
func (m Model) View() string {
	errStyle := ansi.NewStyle().Foreground(m.themed().Error).Bold()
	hintStyle := ansi.NewStyle().Foreground(m.themed().Muted)
	return errStyle.Render(m.Message) + "\n" + hintStyle.Render(m.hint())
}

// hint is the plain-text line under the message: how many retries are used and
// which keys are available.
func (m Model) hint() string {
	retry, dismiss := m.hintParts()
	return retry + dismiss
}

// hintParts splits hint into the part that names retrying (empty once retries
// are exhausted) and the part that names dismissing, so a click can tell them
// apart.
func (m Model) hintParts() (retry, dismiss string) {
	if m.Exhausted() {
		return "", "no retries left (" + strconv.Itoa(m.retryCount) + "/" + strconv.Itoa(m.MaxRetries) + ") " + m.themed().GlyphSet().Dash + " press Esc to dismiss"
	}
	return "press Enter or r to retry (" + strconv.Itoa(m.retryCount) + "/" + strconv.Itoa(m.MaxRetries) + ")", ", Esc to dismiss"
}

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}
