// Package textarea is a multi-line text input widget built on top of the
// root tui package. It follows textinput's rune-buffer approach — the
// value is a single []rune buffer with embedded '\n' runes and the cursor
// is a rune offset into that buffer — but needs its own Update/View since
// textinput.Model has no multi-line support at all (it silently ignores
// tui.KeyEnter, KeyUp and KeyDown).
//
// Home/End operate on the current line (the line the cursor is on), not
// the whole buffer: that's the more useful textarea behavior and no
// criterion pins it to whole-buffer semantics, so this package picks the
// per-line interpretation and documents it here for consistency.
package textarea

import (
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/input"
	"github.com/ows4444/tui/internal/edit"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/motion"
	"github.com/ows4444/tui/theme"
)

// Model is a multi-line text input. Embed it as a field in a parent
// tui.Model, forward relevant Msgs to its Update, and render its View as
// part of the parent's own View.
type Model struct {
	Placeholder string
	Width       int // 0 = unlimited width, no horizontal scrolling per line
	// Height is the number of lines View shows. 0 = unlimited: every line is
	// shown, as before Height existed. With Height > 0 View shows a window of
	// at most Height lines that scrolls, by the least distance, to keep the
	// cursor line visible.
	Height    int
	CharLimit int // 0 = unlimited length, in grapheme clusters (a '\n' counts as one)

	// SoftWrap, when true and Width > 0, wraps a line longer than Width
	// columns onto further visual rows at a cluster boundary instead of
	// scrolling it horizontally. Up and Down then move between visual rows,
	// and Height (if set) counts visual rows. The zero value keeps the
	// per-line horizontal scrolling.
	SoftWrap bool

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
	// one step, as is a paste. SetValue and Reset forget the history.
	UndoLimit int

	// ClipboardWrite, when set, receives the OSC 52 sequence that Copy and
	// Cut send, in place of the Program's output, and Paste inserts only what
	// this Model copied. Leave it nil: Copy and Cut then return
	// tui.WriteClipboard and Paste returns tui.PasteCopied, so the Program
	// writes the sequence and holds the copied text.
	//
	// Deprecated: use tui.WithOutput to direct the Program's output.
	ClipboardWrite func(string) (int, error)

	// clip is the last text copied through the deprecated ClipboardWrite.
	clip string

	// Mouse, when true, makes Update handle tui.MouseEvent: a left click inside
	// Bounds moves the cursor to the clicked cell, a drag selects, and the wheel
	// moves the cursor (and with it the Height window) by three rows. Off (the
	// default) ignores the mouse.
	Mouse bool
	// Bounds is the screen rectangle where the app draws the text area;
	// clicks outside it do not start a selection. The app sets it each frame
	// it moves.
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

	value         doc // text and line index; persistent, see doc.go
	cursor        int
	focused       bool
	cursorVisible bool
	top           int    // first visible line when Height > 0; see scrollToCursor
	blinkID       uint64 // id of the one live blink loop; stale ticks are dropped

	hist     edit.History
	sel      int  // selection anchor as rune offset + 1; 0 = no selection
	quiet    bool // replace does not record history (a group records itself)
	dragging bool // a left-button drag that began inside Bounds is under way
}

// New returns a Model with sane default styling: a faint placeholder and a
// reverse-video block cursor, matching textinput.New.
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
	Newline        keymap.Binding // insert a line break
	Left           keymap.Binding // move the cursor one character left
	Right          keymap.Binding // move the cursor one character right
	WordLeft       keymap.Binding // move the cursor one word left
	WordRight      keymap.Binding // move the cursor one word right
	Up             keymap.Binding // move the cursor up one line
	Down           keymap.Binding // move the cursor down one line
	Home           keymap.Binding // move the cursor to the start of the line
	End            keymap.Binding // move the cursor to the end of the line
	DeleteBack     keymap.Binding // delete the character before the cursor
	DeleteForward  keymap.Binding // delete the character at the cursor
	DeleteToStart  keymap.Binding // delete from the cursor to the line start
	DeleteToEnd    keymap.Binding // delete from the cursor to the line end
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
		Newline:        keymap.NewBinding("new line", "enter"),
		Left:           keymap.NewBinding("move left", "left"),
		Right:          keymap.NewBinding("move right", "right"),
		WordLeft:       keymap.NewBinding("word left", "ctrl+left", "alt+left", "ctrl+alt+left"),
		WordRight:      keymap.NewBinding("word right", "ctrl+right", "alt+right", "ctrl+alt+right"),
		Up:             keymap.NewBinding("move up", "up"),
		Down:           keymap.NewBinding("move down", "down"),
		Home:           keymap.NewBinding("start of line", "home", "ctrl+a"),
		End:            keymap.NewBinding("end of line", "end", "ctrl+e"),
		DeleteBack:     keymap.NewBinding("delete back", "backspace"),
		DeleteForward:  keymap.NewBinding("delete forward", "delete"),
		DeleteToStart:  keymap.NewBinding("delete to line start", "ctrl+u"),
		DeleteToEnd:    keymap.NewBinding("delete to line end", "ctrl+k"),
		DeleteWordBack: keymap.NewBinding("delete word back", "ctrl+w"),
		Undo:           keymap.NewBinding("undo", "ctrl+z"),
		Redo:           keymap.NewBinding("redo", "ctrl+shift+z", "ctrl+y"),
		Copy:           keymap.NewBinding("copy", "ctrl+c"),
		Cut:            keymap.NewBinding("cut", "ctrl+x"),
		Paste:          keymap.NewBinding("paste", "ctrl+v"),
	}
}

func (km KeyMap) list() []keymap.Binding {
	return []keymap.Binding{km.Newline, km.Left, km.Right, km.WordLeft, km.WordRight, km.Up, km.Down,
		km.Home, km.End, km.DeleteBack, km.DeleteForward, km.DeleteToStart, km.DeleteToEnd, km.DeleteWordBack,
		km.Undo, km.Redo, km.Copy, km.Cut, km.Paste}
}

func (km KeyMap) zero() bool {
	for _, b := range km.list() {
		if len(b.Keys) > 0 {
			return false
		}
	}
	return true
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

// Value returns the current text, including any embedded newlines.
func (m Model) Value() string { return m.value.String() }

// SetValue replaces the current value and moves the cursor to its end,
// truncating to CharLimit if set.
func (m *Model) SetValue(s string) {
	rs := []rune(s)
	if m.CharLimit > 0 && len(rs) > m.CharLimit {
		cl := edit.Split(s)
		if len(cl) > m.CharLimit {
			rs = []rune(strings.Join(cl[:m.CharLimit], ""))
		}
	}
	m.value = newDoc(rs)
	m.cursor = m.value.len()
	m.hist.Clear()
	m.sel = 0
	m.scrollToCursor()
}

// Reset clears the value and moves the cursor to the start.
func (m *Model) Reset() {
	m.value = doc{}
	m.cursor = 0
	m.top = 0
	m.hist.Clear()
	m.sel = 0
}

// Cursor returns the cursor's current rune offset within Value.
func (m Model) Cursor() int { return m.cursor }

// SetCursor moves the cursor to pos, clamped within Value.
func (m *Model) SetCursor(pos int) {
	m.cursor = m.snap(clamp(pos, 0, m.value.len()))
	m.sel = 0
	m.scrollToCursor()
}

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

// Blur marks the input unfocused. A blink tick already in flight is dropped
// when it arrives (its id no longer matches) and does not reschedule.
func (m *Model) Blur() {
	m.focused = false
	m.cursorVisible = false
	m.blinkID = nextBlinkID.Add(1) // orphan any tick in flight
}

// blinkMsg carries the id of the loop that scheduled it, like textinput's.
// Focus issues a new id, so a tick from an earlier loop is dropped: Focus,
// Blur, Focus inside one blink interval leaves exactly one live loop.
type blinkMsg struct{ id uint64 }

var nextBlinkID atomic.Uint64

func blinkCmd(id uint64) tui.Cmd {
	return tui.FromCtx(motion.After(530*time.Millisecond, func(time.Time) tui.Msg { return blinkMsg{id: id} }))
}

// Update handles a key press or a blink tick. It returns a concrete Model
// (not tui.Model) so callers can chain without a type assertion.
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
			return m, nil // a stale loop: let it die
		}
		if m.Motion.Reduced() {
			m.cursorVisible = true
			return m, nil
		}
		m.cursorVisible = !m.cursorVisible
		return m, blinkCmd(m.blinkID)
	case tui.Key:
		cmd := m.handleKey(msg)
		m.scrollToCursor()
		m.cursorVisible = true
		return m, cmd
	case tui.PasteEvent:
		// Unlike textinput, a textarea keeps embedded newlines from a
		// multi-line paste.
		// Untrusted: expand tabs, strip escape sequences and controls
		// (newlines are kept).
		m.insertRunes(pasteRunes(msg.Text), false)
		m.scrollToCursor()
		m.cursorVisible = true
		return m, nil
	}
	return m, nil
}

// handleKey applies k and returns the Cmd a Copy, Cut or Paste needs the
// Program to run, if any.
func (m *Model) handleKey(k tui.Key) tui.Cmd {
	km := m.keys()
	// Redo is tested before Undo and neither ignores Shift: ctrl+shift+z must
	// not be taken for ctrl+z.
	switch {
	case keymap.Matches(k, km.Redo):
		m.redo()
		return nil
	case keymap.Matches(k, km.Undo):
		m.undo()
		return nil
	case keymap.Matches(k, km.Copy):
		_, cmd := m.copy()
		return cmd
	case keymap.Matches(k, km.Cut):
		ok, cmd := m.copy()
		if ok {
			m.deleteSelection()
		}
		return cmd
	case keymap.Matches(k, km.Paste):
		if m.ClipboardWrite != nil {
			m.insertRunes(pasteRunes(m.clip), false)
			return nil
		}
		return tui.PasteCopied()
	}
	// Moving keeps or extends the selection with Shift held and drops it
	// otherwise.
	moving := func() {
		if !k.Mod.Shift() {
			m.sel = 0
		} else if m.sel == 0 {
			m.sel = m.cursor + 1
		}
	}
	del := func() bool { return m.deleteSelection() }
	switch {
	case keyHit(k, km.WordLeft):
		moving()
		m.cursor = m.wordLeft(m.cursor)
	case keyHit(k, km.WordRight):
		moving()
		m.cursor = m.wordRight(m.cursor)
	case keyHit(k, km.Left):
		moving()
		m.cursor = m.prevBoundary(m.cursor)
	case keyHit(k, km.Right):
		moving()
		m.cursor = m.nextBoundary(m.cursor)
	case keyHit(k, km.Up):
		moving()
		m.moveVertical(-1)
	case keyHit(k, km.Down):
		moving()
		m.moveVertical(1)
	case keyHit(k, km.Home):
		moving()
		lineStart, _ := m.currentLineBounds()
		m.cursor = lineStart
	case keyHit(k, km.End):
		moving()
		_, lineEnd := m.currentLineBounds()
		m.cursor = lineEnd
	case keyHit(k, km.Newline):
		m.typeRune('\n')
	case keyHit(k, km.DeleteBack):
		if !del() {
			m.deleteBeforeCursor()
		}
	case keyHit(k, km.DeleteForward):
		if !del() {
			m.deleteAtCursor()
		}
	case keyHit(k, km.DeleteToStart):
		if !del() {
			lineStart, _ := m.currentLineBounds()
			m.replace(lineStart, m.cursor, nil)
			m.cursor = lineStart
		}
	case keyHit(k, km.DeleteToEnd):
		if !del() {
			_, lineEnd := m.currentLineBounds()
			m.replace(m.cursor, lineEnd, nil)
		}
	case keyHit(k, km.DeleteWordBack):
		if !del() {
			m.deleteWordBeforeCursor()
		}
	case k.Type == tui.KeyRunes:
		if k.Mod.Alt() {
			return nil
		}
		if r := []rune(k.Text); len(r) == 1 {
			m.typeRune(r[0])
		} else {
			m.insertRunes(r, false)
		}
	case k.Type == tui.KeySpace:
		m.typeRune(' ')
	}
	return nil
}

// pasteRunes is untrusted text made safe to insert: tabs expanded, escape
// sequences and controls stripped (newlines are kept), carriage returns dropped.
func pasteRunes(text string) []rune {
	rs := []rune(ansi.Sanitize(ansi.ExpandTabs(text)))
	out := rs[:0]
	for _, r := range rs {
		if r != '\r' {
			out = append(out, r)
		}
	}
	return out
}

// replace swaps value[from:to] for ins. The doc is persistent (copy-on-write):
// an edit inside one line rebuilds only that line's group, so Model copies
// holding the old value are undisturbed and the cost is O(line), not O(buffer).
// Unless quiet is set it records the edit as one undo step, with the cursor as
// it is on entry (callers move it afterwards).
func (m *Model) replace(from, to int, ins []rune) {
	if !m.quiet {
		removed := ""
		if to > from {
			removed = m.runesString(from, to)
		}
		if len(ins) > 0 || removed != "" {
			m.hist.Record(edit.Step{Pos: from, Del: len(ins), Ins: removed, Cursor: m.cursor}, false, m.UndoLimit)
		}
	}
	m.sel = 0
	m.value = m.value.replace(from, to, ins)
}

// oneByte holds the one-character strings of ASCII, so recording the removal
// of a single ASCII rune (Backspace over typed text) allocates nothing.
var oneByte = func() (t [128]string) {
	for i := range t {
		t[i] = string(rune(i))
	}
	return
}()

// runesString is value[from:to] as a string.
func (m *Model) runesString(from, to int) string {
	if to-from == 1 {
		if r := m.value.at(from); r < 0x80 {
			return oneByte[r]
		}
	}
	return m.value.str(from, to)
}

// typeRune inserts one typed rune; a run of them is one undo step.
func (m *Model) typeRune(r rune) {
	rs := [1]rune{r}
	m.insertRunes(rs[:], r != ' ' && r != '\n')
}

// insertRunes inserts rs at the cursor, in place of the selection if there is
// one, as one undo step. With coalesce set and no selection the step merges
// into the typing run before it.
func (m *Model) insertRunes(rs []rune, coalesce bool) {
	cur0, pos, removed := m.cursor, m.cursor, ""
	if lo, hi, ok := m.selection(); ok {
		removed, pos, coalesce = m.value.str(lo, hi), lo, false
		m.quiet = true
		m.replace(lo, hi, nil)
		m.quiet = false
		m.cursor = lo
	}
	m.sel = 0
	m.quiet = true
	for _, r := range rs {
		m.insertRune(r)
	}
	m.quiet = false
	if n := m.cursor - pos; n > 0 || removed != "" {
		m.hist.Record(edit.Step{Pos: pos, Del: n, Ins: removed, Cursor: cur0}, coalesce, m.UndoLimit)
	}
}

// selection returns the selected rune range [lo, hi).
func (m Model) selection() (lo, hi int, ok bool) {
	if m.sel == 0 {
		return 0, 0, false
	}
	a := min(m.sel-1, m.value.len())
	lo, hi = min(a, m.cursor), max(a, m.cursor)
	return lo, hi, lo != hi
}

// deleteSelection removes the selection as one undo step and reports whether
// there was one.
func (m *Model) deleteSelection() bool {
	lo, hi, ok := m.selection()
	m.sel = 0
	if !ok {
		return false
	}
	m.replace(lo, hi, nil)
	m.cursor = lo
	return true
}

// copy copies the selection and reports whether there was one. The Cmd it
// returns has the Program write the OSC 52 sequence and keep the text; with
// the deprecated ClipboardWrite set it writes there and returns no Cmd.
func (m *Model) copy() (bool, tui.Cmd) {
	lo, hi, ok := m.selection()
	if !ok {
		return false, nil
	}
	t := m.value.str(lo, hi)
	if m.ClipboardWrite != nil {
		_, _ = m.ClipboardWrite(ansi.OSC52Copy(t))
		m.clip = t
		return true, nil
	}
	return true, tui.WriteClipboard(t)
}

func (m *Model) undo() {
	if s, ok := m.hist.PopUndo(); ok {
		m.hist.PushRedo(m.applyStep(s))
	}
}

func (m *Model) redo() {
	if s, ok := m.hist.PopRedo(); ok {
		m.hist.PushUndo(m.applyStep(s), m.UndoLimit)
	}
}

// applyStep applies s to the text and returns the step that reverses it.
func (m *Model) applyStep(s edit.Step) edit.Step {
	pos := clamp(s.Pos, 0, m.value.len())
	end := min(pos+s.Del, m.value.len())
	ins := []rune(s.Ins)
	inv := edit.Step{Pos: pos, Del: len(ins), Ins: m.value.str(pos, end), Cursor: m.cursor}
	m.value = m.value.replace(pos, end, ins)
	m.sel = 0
	m.cursor = m.snap(clamp(s.Cursor, 0, m.value.len()))
	return inv
}

func (m *Model) insertRune(r rune) {
	if !m.hasRoomFor(r) {
		return
	}
	m.replace(m.cursor, m.cursor, []rune{r})
	m.cursor++
}

// hasRoomFor reports whether r may be inserted at the cursor under CharLimit,
// which counts grapheme clusters. A rune that joins the cluster before the
// cursor (a combining mark, a joiner) adds no cluster, so it fits even at the
// limit. There are never more clusters than runes, so the count is only taken
// once the rune count reaches the limit.
func (m Model) hasRoomFor(r rune) bool {
	if m.CharLimit <= 0 || m.value.len() < m.CharLimit {
		return true
	}
	if len(edit.Split(m.value.String())) < m.CharLimit {
		return true
	}
	if m.cursor == 0 || r == '\n' {
		return false
	}
	prev := m.value.slice(m.prevBoundary(m.cursor), m.cursor)
	return len(edit.Split(string(prev)+string(r))) == 1
}

// deleteBeforeCursor deletes the grapheme cluster immediately before the cursor. When
// that rune is a newline (i.e. the cursor sits at the start of a line,
// right after the previous line's newline), this joins the two lines —
// criterion #331 — which falls out naturally: a newline is a cluster of its own.
func (m *Model) deleteBeforeCursor() {
	if m.cursor == 0 {
		return
	}
	from := m.prevBoundary(m.cursor)
	m.replace(from, m.cursor, nil)
	m.cursor = from
}

func (m *Model) deleteAtCursor() {
	if m.cursor >= m.value.len() {
		return
	}
	m.replace(m.cursor, m.nextBoundary(m.cursor), nil)
}

func (m *Model) deleteWordBeforeCursor() {
	lineStart, _ := m.currentLineBounds()
	i := m.cursor
	for i > lineStart && m.value.at(i-1) == ' ' {
		i--
	}
	for i > lineStart && m.value.at(i-1) != ' ' {
		i--
	}
	m.replace(i, m.cursor, nil)
	m.cursor = i
}

// currentLineBounds returns the [start, end) rune-offset bounds of the
// line the cursor is currently on, excluding the delimiting newlines.
func (m Model) currentLineBounds() (start, end int) {
	start = m.cursor
	for start > 0 && m.value.at(start-1) != '\n' {
		start--
	}
	end = m.cursor
	for end < m.value.len() && m.value.at(end) != '\n' {
		end++
	}
	return start, end
}

// lineCol returns the cursor's current 0-based line index and column
// (rune offset within that line).
func (m Model) lineCol() (line, col int) {
	line = m.value.lineOf(m.cursor)
	return line, m.cursor - m.value.lineStart(line)
}

// boundsAt returns the [start, end) rune bounds of the line holding pos.
func (m Model) boundsAt(pos int) (start, end int) {
	start = pos
	for start > 0 && m.value.at(start-1) != '\n' {
		start--
	}
	end = pos
	for end < m.value.len() && m.value.at(end) != '\n' {
		end++
	}
	return start, end
}

// clusterSpan bounds how many runes around the cursor are segmented to find
// the neighbouring cluster; no real cluster is longer.
const clusterSpan = 64

// prevBoundary returns the rune offset of the cluster boundary before pos.
func (m Model) prevBoundary(pos int) int {
	if pos <= 0 {
		return 0
	}
	if m.value.at(pos-1) == '\n' {
		return pos - 1
	}
	// An ASCII rune after another ASCII rune (or at the start) is always a
	// cluster boundary: no ASCII rune joins to what precedes it.
	if r := m.value.at(pos - 1); r >= 0x20 && r < 0x7f && (pos == 1 || m.value.at(pos-2) < 0x80) {
		return pos - 1
	}
	s := pos
	for s > 0 && pos-s < clusterSpan && m.value.at(s-1) != '\n' {
		s--
	}
	cl := edit.Split(m.value.str(s, pos))
	return pos - utf8.RuneCountInString(cl[len(cl)-1])
}

// nextBoundary returns the rune offset of the cluster boundary after pos.
func (m Model) nextBoundary(pos int) int {
	if pos >= m.value.len() {
		return m.value.len()
	}
	if m.value.at(pos) == '\n' {
		return pos + 1
	}
	e := pos
	for e < m.value.len() && e-pos < clusterSpan && m.value.at(e) != '\n' {
		e++
	}
	cl := edit.Split(m.value.str(pos, e))
	return pos + utf8.RuneCountInString(cl[0])
}

// snap moves pos back to the start of the cluster containing it.
func (m Model) snap(pos int) int {
	s, e := m.boundsAt(pos)
	if pos == s || pos == e {
		return pos
	}
	at := s
	for _, c := range edit.Split(m.value.str(s, e)) {
		n := utf8.RuneCountInString(c)
		if at+n > pos {
			return at
		}
		at += n
	}
	return pos
}

func (m Model) wordLeft(pos int) int {
	s, _ := m.boundsAt(pos)
	if pos == s {
		return m.prevBoundary(pos)
	}
	var e edit.Editor
	e.Set(m.value.str(s, pos), 0)
	e.WordLeft()
	return s + utf8.RuneCountInString(joinTo(&e, e.Cursor()))
}

func (m Model) wordRight(pos int) int {
	_, end := m.boundsAt(pos)
	if pos == end {
		return m.nextBoundary(pos)
	}
	var e edit.Editor
	e.Set(m.value.str(pos, end), 0)
	e.Home()
	e.WordRight()
	return pos + utf8.RuneCountInString(joinTo(&e, e.Cursor()))
}

// joinTo returns the text of the first n clusters of e.
func joinTo(e *edit.Editor, n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteString(e.Cluster(i))
	}
	return b.String()
}

// colOf returns the display column of rune offset pos within [start, pos).
func (m Model) colOf(start, pos int) int { return ansi.Width(m.value.str(start, pos)) }

// offsetForCol returns how many runes of line to advance so the position is
// the last cluster boundary at or before display column col.
func offsetForCol(line []rune, col int) int {
	acc, n := 0, 0
	for _, c := range edit.Split(string(line)) {
		w := ansi.Width(c)
		if acc+w > col {
			break
		}
		acc += w
		n += utf8.RuneCountInString(c)
	}
	return n
}

// moveVertical moves the cursor to the equivalent display column on the row
// delta away (a logical line, or a visual row under SoftWrap), landing on a
// cluster boundary. It looks only at the current and the destination line, so
// its cost does not depend on the size of the buffer.
func (m *Model) moveVertical(delta int) {
	if m.wrapping() {
		m.moveVisual(delta)
		return
	}
	s, e := m.boundsAt(m.cursor)
	col := m.colOf(s, m.cursor)
	var ts, te int
	if delta < 0 {
		if s == 0 {
			return
		}
		ts, te = m.boundsAt(s - 1)
	} else {
		if e >= m.value.len() {
			return
		}
		ts, te = m.boundsAt(e + 1)
	}
	m.cursor = ts + offsetForCol(m.value.slice(ts, te), col)
}

func (m Model) wrapping() bool { return m.SoftWrap && m.Width > 0 }

// vrow is one visual row: value[s:e]. last marks the final row of its line.
type vrow struct {
	s, e int
	last bool
}

// lineRows wraps the line value[s:e] into rows of at most m.Width columns at
// cluster boundaries (a cluster wider than Width gets a row of its own). When
// cursorAtEnd is set and the last row is full, an empty row follows so the
// end-of-line cursor cell has somewhere to be.
func (m Model) lineRows(s, e int, cursorAtEnd bool) []vrow {
	var rows []vrow
	rs, pos, used := s, s, 0
	if e > s {
		for _, c := range edit.Split(m.value.str(s, e)) {
			w := ansi.Width(c)
			if used > 0 && used+w > m.Width {
				rows = append(rows, vrow{s: rs, e: pos})
				rs, used = pos, 0
			}
			used += w
			pos += utf8.RuneCountInString(c)
		}
	}
	rows = append(rows, vrow{s: rs, e: pos, last: true})
	if cursorAtEnd && used >= m.Width && e > s {
		rows[len(rows)-1].last = false
		rows = append(rows, vrow{s: pos, e: pos, last: true})
	}
	return rows
}

// rowIndex returns the row holding the cursor at pos: the last with s <= pos.
func rowIndex(rows []vrow, pos int) int {
	i := 0
	for j, r := range rows {
		if r.s <= pos {
			i = j
		}
	}
	return i
}

// colInRow returns the rune offset in row r for display column col: the last
// cluster boundary at or before col, never the row's end unless the end is a
// valid cursor position (the last row of a line with a spare column).
func (m Model) colInRow(r vrow, col int) int {
	n := offsetForCol(m.value.slice(r.s, r.e), col)
	if r.s+n == r.e && r.e > r.s && !(r.last && m.colOf(r.s, r.e) < m.Width) {
		n = m.prevBoundaryIn(r.s, r.e) - r.s
	}
	return r.s + n
}

// prevBoundaryIn returns the start of the last cluster of value[s:e].
func (m Model) prevBoundaryIn(s, e int) int {
	cl := edit.Split(m.value.str(s, e))
	return e - utf8.RuneCountInString(cl[len(cl)-1])
}

// moveVisual moves the cursor delta visual rows, keeping its display column.
func (m *Model) moveVisual(delta int) {
	s, e := m.boundsAt(m.cursor)
	rows := m.lineRows(s, e, m.cursor == e)
	ri := rowIndex(rows, m.cursor)
	col := m.colOf(rows[ri].s, m.cursor)
	if t := ri + delta; t >= 0 && t < len(rows) {
		m.cursor = m.colInRow(rows[t], col)
		return
	}
	var r vrow
	if delta < 0 {
		if s == 0 {
			return
		}
		ps, pe := m.boundsAt(s - 1)
		prev := m.lineRows(ps, pe, false)
		r = prev[len(prev)-1]
	} else {
		if e >= m.value.len() {
			return
		}
		ns, ne := m.boundsAt(e + 1)
		r = m.lineRows(ns, ne, false)[0]
	}
	m.cursor = m.colInRow(r, col)
}

// clipPlaceholder returns the placeholder with every row cut to Width columns
// so a long hint never makes the area wider than it was told to be; first is
// the room left on the first row after the cursor cell. With no Width set the
// placeholder is returned whole.
func (m Model) clipPlaceholder(first int) string {
	if m.Width <= 0 {
		return m.Placeholder
	}
	lines := strings.Split(m.Placeholder, "\n")
	for i, l := range lines {
		room := m.Width
		if i == 0 {
			room = max(first, 0)
		}
		lines[i] = ansi.Truncate(l, room)
	}
	return strings.Join(lines, "\n")
}

// View renders the value with the cursor position highlighted via
// CursorStyle, one row per line split on '\n' — criterion #330 — or, when
// the value is empty, a cursor cell followed by the styled placeholder —
// criterion #334. When Width is set, each line is independently scrolled
// horizontally to keep the cursor in view when it's long — criterion #333 —
// or, with SoftWrap, wrapped onto further rows. Text is handled as grapheme
// clusters measured in display columns, as textinput does.
func (m Model) View() string {
	if m.value.len() == 0 {
		var b strings.Builder
		room := m.Width // columns left on the first row; <= 0 means unbounded
		if m.focused && m.cursorVisible {
			b.WriteString(m.CursorStyle.Render(" "))
			room--
		}
		b.WriteString(m.PlaceholderStyle.Render(m.clipPlaceholder(room)))
		return b.String()
	}

	if _, _, ok := m.selection(); ok {
		return m.selectionView()
	}

	if m.wrapping() {
		return m.wrappedView()
	}

	if m.Height > 0 {
		return m.windowedView()
	}

	cursorLine, _ := m.lineCol()

	n := m.value.lineCount()
	var b strings.Builder
	b.Grow(m.value.len() + n)
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteByte('\n')
		}
		ls := m.value.lineStart(i)
		lineCursor := -1
		if i == cursorLine {
			lineCursor = m.cursor - ls
		}
		lr := m.value.slice(ls, m.value.lineEnd(i))
		if m.Width == 0 && isPrintableASCII(lr) {
			m.writeASCII(&b, lr, lineCursor)
		} else {
			b.WriteString(m.renderLine(lr, lineCursor))
		}
	}
	return b.String()
}

// CursorCell returns the display cell the cursor occupies in View: x columns
// from the left and y rows from the top of what View shows, after any
// horizontal scroll, soft wrap or Height window. ok is false while the field
// is not focused, so a parent can forward the result as its tui.CursorPlacer
// answer (after adding its own offset) and the terminal's real cursor follows
// the field, which an IME or a screen magnifier needs.
func (m Model) CursorCell() (x, y int, ok bool) {
	if m.value.len() == 0 {
		return 0, 0, m.focused
	}
	if m.wrapping() {
		rows, cur := m.allRows()
		top := 0
		if m.Height > 0 {
			top = windowTop(m.top, cur, len(rows), m.Height)
		}
		return m.colOf(rows[cur].s, m.cursor), cur - top, m.focused
	}
	line := m.value.lineOf(m.cursor)
	if m.Height > 0 {
		y = line - windowTop(m.top, line, m.value.lineCount(), m.Height)
	} else {
		y = line
	}
	ls := m.value.lineStart(line)
	lr := m.value.slice(ls, m.cursor)
	if m.Width == 0 {
		return ansi.Width(string(lr)), y, m.focused
	}
	var e edit.Editor
	e.Set(m.value.str(ls, m.value.lineEnd(line)), 0)
	e.SetCursor(len(edit.Split(string(lr))))
	first, _ := e.Window(m.Width)
	for i := first; i < e.Cursor(); i++ {
		x += ansi.Width(e.Cluster(i))
	}
	return x, y, m.focused
}

// allRows wraps the whole buffer and returns the rows and the cursor's row.
func (m Model) allRows() (rows []vrow, cursorRow int) {
	s := 0
	for s <= m.value.len() {
		_, e := m.boundsAt(s)
		atCursor := m.cursor >= s && m.cursor <= e
		lr := m.lineRows(s, e, atCursor && m.cursor == e)
		if atCursor {
			cursorRow = len(rows) + rowIndex(lr, m.cursor)
		}
		rows = append(rows, lr...)
		s = e + 1
	}
	return rows, cursorRow
}

// wrappedView renders the soft-wrapped rows, windowed to Height rows.
func (m Model) wrappedView() string {
	rows, cur := m.allRows()
	top, bottom := 0, len(rows)
	if m.Height > 0 {
		top = windowTop(m.top, cur, len(rows), m.Height)
		bottom = min(top+m.Height, len(rows))
	}
	out := make([]string, 0, bottom-top)
	for i := top; i < bottom; i++ {
		r := rows[i]
		cl := edit.Split(m.value.str(r.s, r.e))
		out = append(out, m.paint(cl, i == cur, r.s, r.last))
	}
	return strings.Join(out, "\n")
}

// paint styles clusters cl, which start at rune offset base. The cursor cell
// is drawn when the cursor is inside the row (own is true), and after the
// last cluster when the cursor is at the end of the line.
func (m Model) paint(cl []string, own bool, base int, lineEnd bool) string {
	var b strings.Builder
	show := own && m.focused && m.cursorVisible
	pos := base
	for _, c := range cl {
		if show && pos == m.cursor {
			b.WriteString(m.CursorStyle.Render(c))
		} else {
			b.WriteString(m.TextStyle.Render(c))
		}
		pos += utf8.RuneCountInString(c)
	}
	if show && lineEnd && pos == m.cursor {
		b.WriteString(m.CursorStyle.Render(" "))
	}
	return b.String()
}

// windowedView renders only the Height lines around the cursor. Only those
// lines are styled and joined, so the work that scales with the buffer is
// the group lookup for each line.
func (m Model) windowedView() string {
	total := m.value.lineCount()
	cursorLine := m.value.lineOf(m.cursor)
	top := windowTop(m.top, cursorLine, total, m.Height)
	bottom := min(top+m.Height, total)

	rendered := make([]string, 0, bottom-top)
	for i := top; i < bottom; i++ {
		ls := m.value.lineStart(i)
		lineCursor := -1
		if i == cursorLine {
			lineCursor = m.cursor - ls
		}
		rendered = append(rendered, m.renderLine(m.value.slice(ls, m.value.lineEnd(i)), lineCursor))
	}
	return strings.Join(rendered, "\n")
}

// renderLine renders one line. lineCursor is the cursor's rune offset within
// the line, or -1 when the cursor is on another line. Clusters are the unit
// of drawing and Width is a column count, windowed by edit.Editor.Window.
func (m Model) renderLine(lr []rune, lineCursor int) string {
	if m.Width == 0 && isPrintableASCII(lr) {
		return m.paintASCII(lr, lineCursor)
	}
	text := string(lr)
	var cl []string
	if m.Width > 0 {
		var e edit.Editor
		e.Set(text, 0)
		if lineCursor >= 0 {
			e.SetCursor(len(edit.Split(string(lr[:lineCursor]))))
		} else {
			e.SetCursor(0)
		}
		start, end := e.Window(m.Width)
		cl = make([]string, 0, end-start)
		for i := start; i < end; i++ {
			cl = append(cl, e.Cluster(i))
		}
		// The end-of-line cursor cell is drawn only if the window reaches the end.
		return m.paintLine(cl, lineCursor, end == e.Len(), len(lr), start, &e)
	}
	cl = split(text)
	return m.paintLine(cl, lineCursor, true, len(lr), 0, nil)
}

// paintLine draws clusters cl; startCl is the index of cl[0] within the line's
// clusters (used only to locate the window), e is non-nil when windowed.
func (m Model) paintLine(cl []string, lineCursor int, atEnd bool, runes, startCl int, e *edit.Editor) string {
	var b strings.Builder
	show := m.focused && m.cursorVisible && lineCursor >= 0
	pos := 0
	if e != nil {
		for i := 0; i < startCl; i++ {
			pos += utf8.RuneCountInString(e.Cluster(i))
		}
	}
	for _, c := range cl {
		if show && pos == lineCursor {
			b.WriteString(m.CursorStyle.Render(c))
		} else {
			b.WriteString(m.TextStyle.Render(c))
		}
		pos += utf8.RuneCountInString(c)
	}
	if show && lineCursor == runes && atEnd {
		b.WriteString(m.CursorStyle.Render(" "))
	}
	return b.String()
}

// windowTop returns the first visible line for a window of height lines over
// total lines: top moved by the least distance that puts cursorLine inside
// it, and clamped so the window never runs past the last line.
func windowTop(top, cursorLine, total, height int) int {
	if height <= 0 {
		return 0
	}
	top = clamp(top, 0, max(0, total-height))
	if cursorLine < top {
		return cursorLine
	}
	if cursorLine >= top+height {
		return cursorLine - height + 1
	}
	return top
}

// scrollToCursor records the window's first line after the cursor or value
// changed, so the next scroll is measured from where the window is now.
func (m *Model) scrollToCursor() {
	if m.Height <= 0 {
		m.top = 0
		return
	}
	if m.wrapping() {
		rows, cur := m.allRows()
		m.top = windowTop(m.top, cur, len(rows), m.Height)
		return
	}
	m.top = windowTop(m.top, m.value.lineOf(m.cursor), m.value.lineCount(), m.Height)
}

// visibleWindow returns a [start, end) window of width w into a sequence
// of length total, centered on pos, clamped to the sequence's bounds.
// Mirrors textinput's helper of the same name.
func visibleWindow(total, pos, w int) (start, end int) {
	start = pos - w/2
	if start < 0 {
		start = 0
	}
	end = start + w
	if end > total {
		end = total
		start = end - w
		if start < 0 {
			start = 0
		}
	}
	return start, end
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}

// Model is a tui.CursorProvider, so a Program places the hardware cursor.
var _ tui.CursorProvider = Model{}

// Linearize renders the text area as plain text for accessible output (see
// tui.Linearizer): a header line ("Text area, 3 lines", plus ", focused,
// cursor on line 2, column 5" when focused), then one line per text line,
// "Line 2 of 3: text" ("blank" for an empty line). An empty area reads "Text
// area, empty" with its placeholder. No cursor glyph or styling.
func (m Model) Linearize() string {
	head := "Text area"
	if m.value.len() == 0 {
		head += ", empty"
		if m.focused {
			head += ", focused"
		}
		if m.Placeholder != "" {
			head += ", placeholder: " + m.Placeholder
		}
		return head
	}
	lines := strings.Split(m.value.String(), "\n")
	n := strconv.Itoa(len(lines))
	if len(lines) == 1 {
		head += ", 1 line"
	} else {
		head += ", " + n + " lines"
	}
	if m.focused {
		line, _ := m.lineCol()
		ls, _ := m.boundsAt(m.cursor)
		col := len(edit.Split(m.value.str(ls, m.cursor)))
		head += ", focused, cursor on line " + strconv.Itoa(line+1) + ", column " + strconv.Itoa(col+1)
	}
	out := []string{head}
	for i, l := range lines {
		if l == "" {
			l = "blank"
		}
		out = append(out, "Line "+strconv.Itoa(i+1)+" of "+n+": "+l)
	}
	return strings.Join(out, "\n")
}

func isPrintableASCII(lr []rune) bool {
	for _, r := range lr {
		if r < 0x20 || r >= 0x7f {
			return false
		}
	}
	return true
}

// paintASCII is paintLine for a line of printable ASCII, where every rune is
// its own cluster; it allocates nothing but the result.
func (m Model) paintASCII(lr []rune, lineCursor int) string {
	var b strings.Builder
	b.Grow(len(lr))
	m.writeASCII(&b, lr, lineCursor)
	return b.String()
}

// writeASCII appends the painted line to b.
func (m Model) writeASCII(b *strings.Builder, lr []rune, lineCursor int) {
	show := m.focused && m.cursorVisible && lineCursor >= 0
	plain := m.TextStyle.Render("") == "" // no text styling: write runes as they are
	for j, r := range lr {
		if show && j == lineCursor {
			b.WriteString(m.CursorStyle.Render(string(r)))
		} else if plain {
			b.WriteByte(byte(r)) // #nosec G115 -- printable ASCII
		} else {
			b.WriteString(m.TextStyle.Render(string(r)))
		}
	}
	if show && lineCursor == len(lr) {
		b.WriteString(m.CursorStyle.Render(" "))
	}
}

// split is edit.Split with a fast path for printable ASCII, where every rune
// is its own cluster.
func split(s string) []string {
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] >= 0x7f {
			return edit.Split(s)
		}
	}
	out := make([]string, len(s))
	for i := range out {
		out[i] = s[i : i+1]
	}
	return out
}
