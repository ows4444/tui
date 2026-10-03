// Package textinput is a single-line text input widget built on top of the
// root tui package — the first package in this module that depends on tui
// rather than the other way around, forming a widget layer above the
// core (term/ansi/input) -> tui layering described in the README.
//
// Unlike ansi.Style and layout.Box, Model here isn't an immutable
// value-builder: it's a small live component with both configuration
// (Placeholder, Width, ...) and state (the typed value, cursor position),
// so configuration is plain exported fields you set once after New,
// mirroring how a parent tui.Model holds its own fields — not the
// chainable pattern used by the purely-descriptive Style/Box types.
package textinput

import (
	"strings"
	"sync/atomic"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/input"
	"github.com/ows4444/tui/internal/a11y"
	"github.com/ows4444/tui/internal/edit"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/motion"
	"github.com/ows4444/tui/theme"
)

// Model is a single-line text input. Embed it as a field in a parent
// tui.Model, forward relevant Msgs to its Update, and render its View
// as part of the parent's own View.
type Model struct {
	Placeholder string
	Prompt      string
	Width       int // 0 = unlimited width, no horizontal scrolling
	CharLimit   int // 0 = unlimited length

	TextStyle        ansi.Style
	PlaceholderStyle ansi.Style
	CursorStyle      ansi.Style
	// SelectionStyle styles selected text. New sets a blue background; the
	// zero value falls back to reverse video.
	SelectionStyle ansi.Style

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens

	// UndoLimit is the number of undo steps kept (see KeyMap.Undo): 0 means
	// 100 and a negative value turns undo off. A run of typed characters is
	// one step. SetValue and Reset forget the history.
	UndoLimit int

	// ClipboardWrite, when set, receives the OSC 52 sequence that Copy and
	// Cut send, in place of the Program's output, and Paste inserts only what
	// this Model copied. Leave it nil: Copy and Cut then return
	// tui.WriteClipboard and Paste returns tui.PasteCopied, so the Program
	// writes the sequence and holds the copied text.
	//
	// Deprecated: use tui.WithOutput to direct the Program's output.
	ClipboardWrite func(string) (int, error)
	// DisableCopy makes Copy and Cut do nothing, for fields whose value must
	// not reach the clipboard (passwordinput sets it).
	DisableCopy bool

	// clip is the last text copied through the deprecated ClipboardWrite.
	clip string

	// Mouse, when true, makes Update handle tui.MouseEvent: a left click inside
	// Bounds moves the cursor to the clicked cell and a drag selects. Off (the
	// default) ignores the mouse.
	Mouse bool
	// Bounds is the screen rectangle where the app draws the field, prompt
	// included; clicks outside it do not start a selection. The app sets it
	// each frame it moves.
	Bounds hittest.Rect

	// Motion is the reduced-motion preference. The zero value is
	// motion.Normal, the behaviour before this field existed. With
	// motion.Reduced the cursor does not blink: it stays solid and Focus
	// schedules no tick.
	Motion motion.Preference

	// KeyMap holds the keys for the editing actions. New fills it with
	// DefaultKeyMap; a Model built as a struct literal with a zero KeyMap
	// behaves as if it held DefaultKeyMap. Typed characters are not actions:
	// they are always inserted.
	KeyMap KeyMap

	ed            edit.Editor
	focused       bool
	cursorVisible bool
	blinkID       uint64 // id of the one live blink loop; stale ticks are dropped
	dragging      bool   // a left-button drag that began inside Bounds is under way
}

// New returns a Model with sane default styling: a faint placeholder and a
// reverse-video block cursor (Program keeps the real terminal cursor
// hidden throughout, so the cursor is simulated by styling, not by
// positioning the terminal's own cursor).
func New() Model {
	return Model{
		PlaceholderStyle: ansi.NewStyle().Faint(),
		CursorStyle:      ansi.NewStyle().Reverse(),
		SelectionStyle:   ansi.NewStyle().Background(ansi.Blue).Foreground(ansi.BrightWhite),
		KeyMap:           DefaultKeyMap(),
	}
}

// KeyMap names the keys of each editing action of a Model.
type KeyMap struct {
	Left           keymap.Binding // move the cursor one character left
	Right          keymap.Binding // move the cursor one character right
	WordLeft       keymap.Binding // move the cursor one word left
	WordRight      keymap.Binding // move the cursor one word right
	Home           keymap.Binding // move the cursor to the start of the value
	End            keymap.Binding // move the cursor to the end of the value
	DeleteBack     keymap.Binding // delete the character before the cursor
	DeleteForward  keymap.Binding // delete the character at the cursor
	DeleteToStart  keymap.Binding // delete from the cursor to the start
	DeleteToEnd    keymap.Binding // delete from the cursor to the end
	DeleteWordBack keymap.Binding // delete the word before the cursor
	Undo           keymap.Binding // undo the last edit
	Redo           keymap.Binding // redo an undone edit
	Copy           keymap.Binding // copy the selection with OSC 52
	Cut            keymap.Binding // copy the selection with OSC 52 and delete it
	Paste          keymap.Binding // insert the text of the last Copy or Cut
}

// DefaultKeyMap returns the keys a Model used before KeyMap existed.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Left:           keymap.NewBinding("move left", "left"),
		Right:          keymap.NewBinding("move right", "right"),
		WordLeft:       keymap.NewBinding("word left", "ctrl+left", "alt+left", "ctrl+alt+left"),
		WordRight:      keymap.NewBinding("word right", "ctrl+right", "alt+right", "ctrl+alt+right"),
		Home:           keymap.NewBinding("start of line", "home", "ctrl+a"),
		End:            keymap.NewBinding("end of line", "end", "ctrl+e"),
		DeleteBack:     keymap.NewBinding("delete back", "backspace"),
		DeleteForward:  keymap.NewBinding("delete forward", "delete"),
		DeleteToStart:  keymap.NewBinding("delete to start", "ctrl+u"),
		DeleteToEnd:    keymap.NewBinding("delete to end", "ctrl+k"),
		DeleteWordBack: keymap.NewBinding("delete word back", "ctrl+w"),
		Undo:           keymap.NewBinding("undo", "ctrl+z"),
		Redo:           keymap.NewBinding("redo", "ctrl+shift+z", "ctrl+y"),
		Copy:           keymap.NewBinding("copy", "ctrl+c"),
		Cut:            keymap.NewBinding("cut", "ctrl+x"),
		Paste:          keymap.NewBinding("paste", "ctrl+v"),
	}
}

func (km KeyMap) zero() bool {
	for _, b := range km.list() {
		if len(b.Keys) > 0 {
			return false
		}
	}
	return true
}

func (km KeyMap) list() []keymap.Binding {
	return []keymap.Binding{km.Left, km.Right, km.WordLeft, km.WordRight, km.Home, km.End,
		km.DeleteBack, km.DeleteForward, km.DeleteToStart, km.DeleteToEnd, km.DeleteWordBack,
		km.Undo, km.Redo, km.Copy, km.Cut, km.Paste}
}

func (m Model) keys() KeyMap {
	if m.KeyMap.zero() {
		return DefaultKeyMap()
	}
	return m.KeyMap
}

// Bindings returns the editing actions the Model currently honours, with
// descriptions, for help text. Typed characters are not listed.
func (m Model) Bindings() []keymap.Binding { return m.keys().list() }

// keyHit reports whether k triggers b, ignoring Shift and Super so that, as
// before KeyMap existed, Shift+Left still moves left.
func keyHit(k tui.Key, b keymap.Binding) bool {
	if keymap.Matches(k, b) {
		return true
	}
	if k.Mod&(input.ModShift|input.ModSuper) != 0 {
		k.Mod &^= input.ModShift | input.ModSuper
		return keymap.Matches(k, b)
	}
	return false
}

// Value returns the current text.
func (m Model) Value() string { return m.ed.String() }

// SetValue replaces the current value and moves the cursor to its end,
// truncating to CharLimit if set.
func (m *Model) SetValue(s string) {
	m.ed.Set(s, m.CharLimit)
}

// Reset clears the value and moves the cursor to the start.
func (m *Model) Reset() {
	m.ed.Reset()
}

// Cursor returns the cursor's current position within Value as a grapheme
// cluster index (one cluster is one user-perceived character; for text
// without combining sequences this equals the rune index).
func (m Model) Cursor() int { return m.ed.Cursor() }

// SetCursor moves the cursor to pos, clamped within Value.
func (m *Model) SetCursor(pos int) { m.ed.SetCursor(pos) }

// CursorStart moves the cursor to the beginning of Value.
func (m *Model) CursorStart() { m.ed.Home() }

// CursorEnd moves the cursor to the end of Value.
func (m *Model) CursorEnd() { m.ed.End() }

// Focused reports whether the input is currently focused.
func (m Model) Focused() bool { return m.focused }

// Focus marks the input focused and starts its cursor blinking. The
// returned Cmd must be run (returned onward, or dispatched) for the blink
// to actually animate.
func (m *Model) Focus() tui.Cmd {
	m.focused = true
	m.cursorVisible = true
	if m.Motion.Reduced() {
		return nil // a solid cursor: nothing to tick
	}
	m.blinkID = nextBlinkID.Add(1)
	return blinkCmd(m.blinkID)
}

// Blur marks the input unfocused. Any blink tick already in flight from a
// prior Focus is a no-op once it arrives (see Update) and doesn't
// reschedule itself, so the blink loop stops on its own without needing to
// cancel the Cmd.
func (m *Model) Blur() {
	m.focused = false
	m.cursorVisible = false
	m.blinkID = nextBlinkID.Add(1) // orphan any tick in flight
}

// blinkMsg carries the id of the loop that scheduled it. Focus issues a
// fresh process-unique id, so a tick from an earlier Focus (or another
// input's loop) no longer matches and is dropped: Focus, Blur, Focus inside
// one blink interval leaves exactly one live loop.
type blinkMsg struct{ id uint64 }

var nextBlinkID atomic.Uint64

func blinkCmd(id uint64) tui.Cmd {
	return tui.FromCtx(motion.After(530*time.Millisecond, func(time.Time) tui.Msg { return blinkMsg{id: id} }))
}

// Update handles a key press or a blink tick. It returns a concrete Model
// (not tui.Model) so callers can chain without a type assertion, the same
// way ansi.Style and layout.Box avoid forcing one.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if ev, ok := msg.(tui.MouseEvent); ok {
		return m.updateMouse(ev), nil
	}
	if !m.focused {
		return m, nil
	}

	switch msg := msg.(type) {
	case blinkMsg:
		if msg.id != m.blinkID {
			return m, nil // stale loop
		}
		if m.Motion.Reduced() {
			m.cursorVisible = true
			return m, nil
		}
		m.cursorVisible = !m.cursorVisible
		return m, blinkCmd(m.blinkID)
	case tui.Key:
		cmd := m.handleKey(msg)
		m.cursorVisible = true
		return m, cmd
	case tui.PasteEvent:
		// This is a single-line field, so newlines in a multi-line paste
		// are dropped rather than inserted literally.
		// Untrusted: tabs become one space, then escape sequences and
		// controls are stripped before the text is stored.
		m.insert(singleLine(msg.Text))
		m.cursorVisible = true
		return m, nil
	}
	return m, nil
}

// handleKey applies k and returns the Cmd a Copy, Cut or Paste needs the
// Program to run, if any.
func (m *Model) handleKey(k tui.Key) tui.Cmd {
	km := m.keys()
	m.ed.SetHistoryLimit(m.UndoLimit)
	// Redo is tested before Undo and neither ignores Shift: ctrl+shift+z must
	// not be taken for ctrl+z.
	switch {
	case keymap.Matches(k, km.Redo):
		m.ed.Redo()
		return nil
	case keymap.Matches(k, km.Undo):
		m.ed.Undo()
		return nil
	case keymap.Matches(k, km.Copy):
		_, cmd := m.copy()
		return cmd
	case keymap.Matches(k, km.Cut):
		ok, cmd := m.copy()
		if ok {
			m.ed.DeleteSelection()
		}
		return cmd
	case keymap.Matches(k, km.Paste):
		if m.ClipboardWrite != nil {
			m.insert(singleLine(m.clip))
			return nil
		}
		return tui.PasteCopied()
	}
	move := func(f func()) {
		if k.Mod.Shift() {
			m.ed.SetAnchor()
		} else {
			m.ed.ClearSelection()
		}
		f()
	}
	switch {
	case keyHit(k, km.WordLeft):
		move(m.ed.WordLeft)
	case keyHit(k, km.WordRight):
		move(m.ed.WordRight)
	case keyHit(k, km.Left):
		move(m.ed.Left)
	case keyHit(k, km.Right):
		move(m.ed.Right)
	case keyHit(k, km.Home):
		move(m.ed.Home)
	case keyHit(k, km.End):
		move(m.ed.End)
	case keyHit(k, km.DeleteBack):
		m.ed.Backspace()
	case keyHit(k, km.DeleteForward):
		m.ed.Delete()
	case keyHit(k, km.DeleteToStart):
		m.ed.DeleteToStart()
	case keyHit(k, km.DeleteToEnd):
		m.ed.DeleteToEnd()
	case keyHit(k, km.DeleteWordBack):
		m.ed.DeleteWordBack()
	case k.Type == tui.KeyRunes:
		if k.Mod.Alt() {
			return nil
		}
		m.insert(k.Text)
	case k.Type == tui.KeySpace:
		m.insert(" ")
	}
	return nil
}

// copy copies the selection and reports whether there was one. The Cmd it
// returns has the Program write the OSC 52 sequence and keep the text; with
// the deprecated ClipboardWrite set it writes there and returns no Cmd.
func (m *Model) copy() (bool, tui.Cmd) {
	t := m.ed.SelectedText()
	if t == "" || m.DisableCopy {
		return false, nil
	}
	if m.ClipboardWrite != nil {
		_, _ = m.ClipboardWrite(ansi.OSC52Copy(t))
		m.clip = t
		return true, nil
	}
	return true, tui.WriteClipboard(t)
}

// singleLine is text as this field stores it: untrusted, so tabs become one
// space and escape sequences and controls are stripped, and newlines dropped.
func singleLine(text string) string {
	t := ansi.Sanitize(strings.ReplaceAll(text, "\t", " "))
	return strings.NewReplacer("\r\n", "", "\n", "", "\r", "").Replace(t)
}

// insert splices s in at the cursor, in place of the selection if there is
// one, in one operation, honoring CharLimit (counted in grapheme clusters).
func (m *Model) insert(s string) {
	m.ed.SetHistoryLimit(m.UndoLimit)
	m.ed.ReplaceSelection(s, m.CharLimit)
}

// window returns the cluster range View shows.
func (m Model) window() (start, end int) {
	n := m.ed.Len()
	if m.Width <= 0 {
		return 0, n
	}
	cols := m.Width
	if m.ed.Cursor() == n && !(m.focused && m.cursorVisible) {
		cols++
	}
	return m.ed.Window(cols)
}

// indexAt returns the cluster index under the cell x columns from the left of
// Bounds (prompt included); a cell past the text is the end of the visible
// text, one before the prompt the start of it.
func (m Model) indexAt(x int) int {
	n := m.ed.Len()
	if n == 0 {
		return 0
	}
	start, end := m.window()
	col := x - ansi.Width(m.Prompt)
	if col < 0 {
		return start
	}
	acc := 0
	for i := start; i < end; i++ {
		acc += ansi.Width(m.ed.Cluster(i))
		if col < acc {
			return i
		}
	}
	if end < n {
		return end - 1
	}
	return n
}

// updateMouse applies a mouse event when Mouse is on: a left press inside
// Bounds puts the cursor on the clicked cell and starts a selection there, a
// drag extends it, a release ends it.
func (m Model) updateMouse(ev tui.MouseEvent) Model {
	if !m.Mouse || ev.Button != tui.MouseButtonLeft {
		return m
	}
	switch ev.Action {
	case tui.MouseActionPress:
		if !m.Bounds.Contains(ev.X, ev.Y) {
			return m
		}
		m.ed.SetCursor(m.indexAt(ev.X - m.Bounds.X))
		m.ed.ClearSelection()
		m.ed.SetAnchor()
		m.dragging = true
		m.cursorVisible = true
	case tui.MouseActionMotion:
		if m.dragging {
			m.ed.SetCursor(m.indexAt(ev.X - m.Bounds.X))
			m.cursorVisible = true
		}
	case tui.MouseActionRelease:
		m.dragging = false
	}
	return m
}

func (m Model) selStyle() ansi.Style {
	if m.SelectionStyle == (ansi.Style{}) {
		return ansi.NewStyle().Reverse()
	}
	return m.SelectionStyle
}

// View renders the prompt, then the value with the cursor position
// highlighted via CursorStyle, or — when the value is empty — a cursor
// cell followed by the styled placeholder. When Width is set and the value
// is longer than it, the visible window scrolls to keep the cursor in
// view.
func (m Model) View() string {
	var b strings.Builder
	b.WriteString(m.Prompt)

	n := m.ed.Len()
	if n == 0 {
		if m.focused && m.cursorVisible {
			b.WriteString(m.CursorStyle.Render(" "))
		}
		b.WriteString(m.PlaceholderStyle.Render(m.Placeholder))
		return b.String()
	}

	// The end-of-text cursor cell is only drawn while the cursor is visible;
	// otherwise the text may use the column it would reserve (see window).
	start, end := m.window()
	cur := m.ed.Cursor()
	lo, hi, sel := m.ed.Selection()

	for i := start; i < end; i++ {
		c := m.ed.Cluster(i)
		switch {
		case m.focused && m.cursorVisible && i == cur:
			b.WriteString(m.CursorStyle.Render(c))
		case sel && i >= lo && i < hi:
			b.WriteString(m.selStyle().Render(c))
		default:
			b.WriteString(m.TextStyle.Render(c))
		}
	}
	if m.focused && m.cursorVisible && cur == n && cur == end {
		b.WriteString(m.CursorStyle.Render(" "))
	}
	return b.String()
}

// CursorCell returns the display cell the cursor occupies in View: x columns
// from the left of the prompt, y always 0. ok is false while the field is not
// focused, so a parent can forward the result as its tui.CursorPlacer answer
// (after adding its own offset) and the terminal's real cursor follows the
// field, which an IME or a screen magnifier needs.
func (m Model) CursorCell() (x, y int, ok bool) {
	x = ansi.Width(m.Prompt)
	n := m.ed.Len()
	if n == 0 {
		return x, 0, m.focused
	}
	start, _ := m.window()
	for i := start; i < m.ed.Cursor(); i++ {
		x += ansi.Width(m.ed.Cluster(i))
	}
	return x, 0, m.focused
}

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}

// Model is a tui.CursorProvider, so a Program places the hardware cursor.
var _ tui.CursorProvider = Model{}

// label is the prompt without surrounding punctuation and spaces ("Name:  "
// -> "Name", "> " -> ""), or "" if there is no wording in the prompt.
func (m Model) label() string {
	return a11y.PromptLabel(m.Prompt)
}

// Linearize renders the field as one plain-text line for accessible output
// (see tui.Linearizer): its label, "text field", ", focused" when it has
// focus, and its value in words, e.g. `Name, text field, focused, value:
// Ada`, or "empty" (with the placeholder) when nothing is typed. No cursor,
// prompt punctuation or styling.
//
// Widgets that embed Model and hold a secret must override this (see
// passwordinput and maskedinput), or the promoted method would speak the
// real value.
func (m Model) Linearize() string {
	head := "Text field"
	if l := m.label(); l != "" {
		head = l + ", text field"
	}
	if m.focused {
		head += ", focused"
	}
	if m.ed.Len() == 0 {
		if m.Placeholder != "" {
			return head + ", empty, placeholder: " + m.Placeholder
		}
		return head + ", empty"
	}
	return head + ", value: " + m.ed.String()
}

// visibleWindow returns a [start, end) window of width w into a sequence
// of length total, centered on pos, clamped to the sequence's bounds. View
// itself windows by display columns (edit.Editor.Window); this is the
// cell-count form.
func visibleWindow(total, pos, w int) (start, end int) {
	start = max(pos-w/2, 0)
	end = start + w
	if end > total {
		end = total
		start = max(end-w, 0)
	}
	return start, end
}
