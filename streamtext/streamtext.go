// Package streamtext reveals text progressively, a few characters at a
// time — InkUI's "StreamingText" and "Typewriter", which are one Model with
// different settings (New for fast streaming, NewTypewriter for pauses and
// a blinking cursor). It is the interactive counterpart to the stateless
// helpers in widgets: a Model driven by tui.Tick, in the same
// self-rescheduling style as spinner.
//
// Stability: experimental. Its API may change in any minor release.
package streamtext

import (
	"strings"
	"sync/atomic"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/motion"
	"github.com/ows4444/tui/theme"
)

var nextID atomic.Int64

// Model reveals its text CharsPerTick visible columns at a time, every
// Interval, while running. Text can be replaced (SetText) or grown as it
// arrives (Append), which makes it suitable for showing a streamed reply.
//
// Ticking is demand-driven: a tick Cmd is only in flight while there is
// unrevealed text, and there is never more than one — Start, Append and
// SetText return a Cmd only when they need to (re)start the chain, so the
// caller must return it from Update, as with spinner.Start. Once caught up
// the chain simply ends; Append starts it again. Stop ends the animation
// (a tick already in flight lands and does nothing).
//
// Create one with New: each Model gets its own id so it ignores the ticks
// of any other streamtext Model in the same program.
type Model struct {
	Theme theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens       theme.Tokens
	Interval     time.Duration // time between ticks
	CharsPerTick int           // visible columns revealed per tick (minimum 1)
	// Width, if > 0, word-wraps the text (ansi.Wrap, so runs of spaces
	// collapse). The wrap is computed from the full text and leaves room
	// for the cursor, so text already shown never reflows as more appears.
	Width int
	// Cursor is drawn in Theme.Primary after the revealed text, while the
	// reveal is incomplete (and afterwards if CursorWhenDone); empty
	// disables it.
	Cursor string

	// Pacing and cursor options, all off in New() and set by NewTypewriter.
	//
	// After a column showing one of the characters in PauseAfter is
	// revealed, the next PauseTicks ticks reveal nothing (the tick that
	// reveals it also stops there, whatever CharsPerTick is). No pause
	// follows the final character.
	PauseAfter string
	PauseTicks int
	// CursorWhenDone keeps the cursor after the reveal completes. If
	// BlinkInterval is also > 0 it then blinks at that interval for as long
	// as the model runs — the one case where ticks continue with nothing
	// left to reveal. The cursor is always solid while text is revealed,
	// so text added during a blink appears after at most one BlinkInterval.
	CursorWhenDone bool
	BlinkInterval  time.Duration

	// Motion is the reduced-motion preference. The zero value is
	// motion.Normal, the behaviour before this field existed. With
	// motion.Reduced, the text is revealed at once (as Skip does) and the
	// cursor stays solid, so no tick is scheduled.
	Motion motion.Preference

	// Raw, when true, stores and renders text unchanged. By default SetText
	// and Append expand tabs and strip every escape sequence and control
	// character except SGR styling (ansi.SanitizeKeepSGR), so untrusted
	// text cannot reach the terminal.
	Raw bool

	id       int64
	buf      *strings.Builder // text storage; text is buf's first n bytes
	n        int
	cache    *wrapCache
	shown    int // visible columns of the display text revealed so far
	running  bool
	ticking  bool // a tick Cmd is in flight
	wait     int  // pause ticks still to sit out
	blinkOff bool // cursor currently hidden by the blink
}

// New returns an idle Model with an empty text, a 30ms interval, two
// columns per tick and a block cursor.
func New() Model {
	return Model{
		Theme:        theme.DarkTheme(),
		Interval:     30 * time.Millisecond,
		CharsPerTick: 2,
		Cursor:       theme.UnicodeGlyphSet().Cursor,
		id:           nextID.Add(1),
		cache:        &wrapCache{},
	}
}

// NewTypewriter returns an idle Model tuned like a typewriter: one column
// every 55ms, a pause after sentence enders, and a solid block cursor that
// stays after the text and blinks.
func NewTypewriter() Model {
	m := New()
	m.Interval = 55 * time.Millisecond
	m.CharsPerTick = 1
	m.Cursor = theme.UnicodeGlyphSet().CursorBlock
	m.PauseAfter = ".!?\n"
	m.PauseTicks = 6
	m.CursorWhenDone = true
	m.BlinkInterval = 530 * time.Millisecond
	return m
}

type tickMsg struct{ id int64 }

func tickCmd(id int64, d time.Duration) tui.Cmd {
	return tui.FromCtx(motion.After(d, func(time.Time) tui.Msg { return tickMsg{id: id} }))
}

// Start begins revealing; return the Cmd it produces (nil when there is
// nothing to reveal yet) from your own Init or Update.
// Call it on the model your program keeps: in an Init with a value receiver
// it would run on a copy, and the kept model would ignore the ticks. Start
// where the model is built and have Init return that Cmd.
func (m *Model) Start() tui.Cmd {
	m.running = true
	return m.schedule()
}

// Stop ends the animation without discarding text or progress.
func (m *Model) Stop() { m.running = false }

// Running reports whether the reveal is currently running.
func (m Model) Running() bool { return m.running }

// SetText replaces the text and restarts the reveal from the beginning.
func (m *Model) SetText(s string) tui.Cmd {
	s = m.clean(s)
	m.buf = new(strings.Builder)
	m.buf.WriteString(s)
	m.n = len(s)
	m.cache = &wrapCache{}
	m.shown = 0
	m.wait = 0
	m.blinkOff = false
	return m.schedule()
}

// Append adds s to the end of the text, leaving progress where it is.
func (m *Model) Append(s string) tui.Cmd {
	s = m.clean(s)
	if m.buf == nil || m.buf.Len() != m.n {
		// Storage is shared with a copy that appended past us: fork it.
		nb := new(strings.Builder)
		nb.WriteString(m.txt())
		m.buf = nb
		m.cache = &wrapCache{}
	}
	m.buf.WriteString(s)
	m.n = m.buf.Len()
	m.blinkOff = false
	return m.schedule()
}

// clean applies the default sanitising unless Raw is set.
func (m Model) clean(s string) string {
	if m.Raw {
		return s
	}
	return ansi.SanitizeKeepSGR(ansi.ExpandTabs(s))
}

// Skip reveals all the text at once.
func (m *Model) Skip() { m.shown = m.total() }

// Text returns the full text, revealed or not.
func (m Model) Text() string { return m.txt() }

func (m Model) txt() string {
	if m.buf == nil {
		return ""
	}
	return m.buf.String()[:m.n]
}

// Done reports whether every character is revealed; true for empty text.
func (m Model) Done() bool { return m.shown >= m.total() }

// blinks reports whether a finished reveal keeps ticking to blink.
func (m Model) blinks() bool {
	return m.cursor() != "" && m.CursorWhenDone && m.BlinkInterval > 0
}

// schedule starts a tick chain if one is needed and none is running: while
// text remains to be revealed, or, once done, only to blink.
func (m *Model) schedule() tui.Cmd {
	if m.Motion.Reduced() {
		m.Skip()
		return nil
	}
	if !m.running || m.ticking {
		return nil
	}
	d := m.Interval
	if m.Done() {
		if !m.blinks() {
			return nil
		}
		d = m.BlinkInterval
	}
	m.ticking = true
	return tickCmd(m.id, d)
}

// Update reveals the next CharsPerTick columns on its own tick, honoring
// PauseAfter/PauseTicks and blinking a finished cursor; a no-op for any
// Msg but its own tick.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	tick, ok := msg.(tickMsg)
	if !ok || tick.id != m.id {
		return m, nil
	}
	m.ticking = false
	if !m.running {
		return m, nil
	}
	if m.wait > 0 {
		m.wait--
		return m, m.schedule()
	}
	if m.Done() {
		// Only a blinking cursor keeps the chain alive once done.
		if m.blinks() {
			m.blinkOff = !m.blinkOff
		}
		return m, m.schedule()
	}

	n := m.CharsPerTick
	if n < 1 {
		n = 1
	}
	total := m.total()
	var cols []rune
	if m.PauseTicks > 0 && m.PauseAfter != "" {
		cols = columnRunes(m.display())
	}
	for i := 0; i < n && m.shown < total; i++ {
		m.shown++
		if m.shown < total && m.shown <= len(cols) && cols[m.shown-1] != 0 &&
			strings.ContainsRune(m.PauseAfter, cols[m.shown-1]) {
			m.wait = m.PauseTicks
			break
		}
	}
	return m, m.schedule()
}

// View renders the text revealed so far, plus the cursor while the
// reveal is incomplete (or after, if CursorWhenDone).
func (m Model) View() string {
	out, done := m.revealed()
	if c, ok := m.cursorToDraw(done); ok {
		out += ansi.NewStyle().Foreground(m.themed().Primary).Render(c)
	}
	return out
}

// revealed returns the revealed part of the laid out text, without the
// cursor, and whether the whole text is revealed.
func (m Model) revealed() (out string, done bool) {
	d, width, plain := m.lay()
	out = d
	switch {
	case m.shown <= 0:
		out = ""
	case plain && m.shown >= width:
		// Everything is revealed and there is no styling for Truncate to
		// close, so d is returned as is instead of being re-measured.
	default:
		out = ansi.Truncate(d, m.shown)
	}
	return out, m.shown >= width
}

// cursorToDraw returns the cursor glyph to append to the revealed text now.
func (m Model) cursorToDraw(done bool) (string, bool) {
	c := m.cursor()
	return c, c != "" && (!done || (m.CursorWhenDone && !m.blinkOff))
}

// columnRunes maps each visible column of s to the rune that starts there
// (0 for the second column of a wide rune), skipping escape sequences and
// zero-width runes, so a column index can be checked against PauseAfter.
func columnRunes(s string) []rune {
	var cols []rune
	for _, r := range ansi.StripANSI(s) {
		w := ansi.Width(string(r))
		for k := 0; k < w; k++ {
			if k == 0 {
				cols = append(cols, r)
			} else {
				cols = append(cols, 0)
			}
		}
	}
	return cols
}

// wrapCache remembers the wrapped form of the text for one width, so an
// Append only lays out the words it added. It works incrementally, inside
// the last paragraph too: a line is committed as soon as the next word
// starts a new one, and only the open line plus the text after the last
// complete word (the word being typed may still grow) is laid out again.
//
// The wrapped text (ansi.Wrap of the whole text) is kept in disp, which is
// append-only, so View can hand out disp.String() without copying: the
// committed part, then the open part as of the last layout. When the open
// part changes by more than an extension (a growing word that no longer
// fits its line) disp is rebuilt once. The cache is shared by copies of a
// Model, so every use re-checks it against the text.
type wrapCache struct {
	w         int
	buf       *strings.Builder // text storage this was built from
	disp      *strings.Builder // committed output, then the open part
	commitLen int              // bytes of disp that are committed
	cw        int              // visible columns of the committed output
	consumed  int              // text[:consumed] is committed or in line
	parStart  int              // where the open paragraph starts in the text
	line      []byte           // open line: the complete words placed on it
	lineW     int              // visible columns of line
	add       []byte           // scratch: newly committed output, then the open part
	synced    int              // len(text) at the last sync
	syncedOK  bool             // synced, width describe the current layout
	width     int              // visible columns of the whole wrapped text
	started   bool             // a line has been committed (so "\n" precedes the next)
	esc       bool             // the text has an escape byte
	wrapped   int              // bytes examined by layout, for tests
}

func (c *wrapCache) reset(w int, buf *strings.Builder) {
	*c = wrapCache{w: w, buf: buf, disp: new(strings.Builder), wrapped: c.wrapped}
}

// lay returns the wrapped text and its visible width. plain reports that
// the text is cached and holds no escape byte, which lets View skip
// Truncate once everything is revealed.
func (m Model) lay() (out string, width int, plain bool) {
	text := m.txt()
	if m.Width <= 0 {
		return text, ansi.Width(text), false
	}
	w := m.Width - ansi.Width(m.cursor())
	if w < 1 {
		w = 1
	}
	c := m.cache
	if c == nil {
		out = ansi.Wrap(text, w)
		return out, ansi.Width(out), false
	}
	if c.disp == nil || c.w != w || c.buf != m.buf || c.consumed > len(text) || c.parStart > len(text) {
		c.reset(w, m.buf)
	}
	width = c.sync(text, w)
	return c.disp.String(), width, !c.esc
}

// nextField returns the first whitespace-separated word of s (the
// strings.Fields definition) and what follows it; word is "" when s has
// no more words. It allocates nothing, unlike strings.Fields.
func nextField(s string) (word, rest string) {
	i := 0
	for i < len(s) {
		r, n := utf8.DecodeRuneInString(s[i:])
		if !unicode.IsSpace(r) {
			break
		}
		i += n
	}
	s = s[i:]
	j := 0
	for j < len(s) {
		r, n := utf8.DecodeRuneInString(s[j:])
		if unicode.IsSpace(r) {
			break
		}
		j += n
	}
	return s[:j], s[j:]
}

// commit places one complete word on the open line, starting a new line
// (pushed onto c.add) when it does not fit within w columns.
func (c *wrapCache) commit(word string, w int) {
	ww := ansi.Width(word)
	switch {
	case len(c.line) == 0:
		c.line = append(c.line[:0], word...)
		c.lineW = ww
	case c.lineW+1+ww <= w:
		c.line = append(c.line, ' ')
		c.line = append(c.line, word...)
		c.lineW += 1 + ww
	default:
		c.push()
		c.line = append(c.line[:0], word...)
		c.lineW = ww
	}
}

// push closes the open line onto c.add (the newly committed output).
func (c *wrapCache) push() {
	if c.started {
		c.add = append(c.add, '\n')
	}
	c.add = append(c.add, c.line...)
	c.cw += c.lineW
	c.started = true
	c.line, c.lineW = c.line[:0], 0
}

// sync lays out text[c.consumed:] and updates disp; it returns the visible
// width of the whole wrapped text. It allocates only when disp, or the
// scratch buffers kept in c, have to grow; a call with no new text is free.
func (c *wrapCache) sync(text string, w int) int {
	if c.synced == len(text) && c.syncedOK {
		return c.width
	}
	rest := text[c.consumed:]
	c.wrapped += len(rest)
	if strings.IndexByte(rest, 0x1b) >= 0 {
		c.esc = true
	}
	c.add = c.add[:0]
	// Close every paragraph that ends in a newline.
	for {
		j := strings.IndexByte(rest, '\n')
		if j < 0 {
			break
		}
		for seg := rest[:j]; ; {
			var word string
			if word, seg = nextField(seg); word == "" {
				break
			}
			c.commit(word, w)
		}
		nl := len(text) - len(rest) + j
		if len(c.line) > 0 {
			c.push()
		} else if nl == c.parStart {
			// an empty paragraph is an empty line
			if c.started {
				c.add = append(c.add, '\n')
			}
			c.started = true
		}
		rest = rest[j+1:]
		c.parStart = nl + 1
		c.consumed = c.parStart
	}
	// The open paragraph: commit the complete words; the last one is
	// complete only if whitespace follows it.
	partial := ""
	body := rest
	if len(rest) > 0 {
		if r, _ := utf8.DecodeLastRuneInString(rest); !unicode.IsSpace(r) {
			k := len(rest)
			for k > 0 {
				r, n := utf8.DecodeLastRuneInString(rest[:k])
				if unicode.IsSpace(r) {
					break
				}
				k -= n
			}
			partial, body = rest[k:], rest[:k]
		}
	}
	for {
		var word string
		if word, body = nextField(body); word == "" {
			break
		}
		c.commit(word, w)
	}
	c.consumed = len(text) - len(partial)
	// The open part: the open line plus the word being typed.
	addLen := len(c.add)
	if c.started {
		c.add = append(c.add, '\n')
	}
	openW := 0
	switch pw := ansi.Width(partial); {
	case partial != "" && len(c.line) == 0:
		c.add = append(c.add, partial...)
		openW = pw
	case partial != "" && c.lineW+1+pw <= w:
		c.add = append(c.add, c.line...)
		c.add = append(c.add, ' ')
		c.add = append(c.add, partial...)
		openW = c.lineW + 1 + pw
	case partial != "":
		c.add = append(c.add, c.line...)
		c.add = append(c.add, '\n')
		c.add = append(c.add, partial...)
		openW = c.lineW + pw
	case len(c.line) > 0:
		c.add = append(c.add, c.line...)
		openW = c.lineW
	case c.parStart == len(text):
		// the last paragraph is empty: an empty line (just the separator)
	default:
		c.add = c.add[:addLen] // nothing open
	}
	// Bring disp up to date: extend it if the new open part continues the
	// old one, else rebuild it from the committed prefix.
	s := c.add
	old := c.disp.String()
	oldOpen := old[c.commitLen:]
	if len(s) >= len(oldOpen) && string(s[:len(oldOpen)]) == oldOpen {
		c.disp.Write(s[len(oldOpen):])
	} else {
		nb := new(strings.Builder)
		nb.Grow(c.commitLen + len(s))
		nb.WriteString(old[:c.commitLen])
		nb.Write(s)
		c.disp = nb
	}
	c.commitLen += addLen
	c.synced, c.syncedOK, c.width = len(text), true, c.cw+openW
	return c.width
}

// display is the text as laid out for drawing: wrapped when Width is set.
func (m Model) display() string {
	d, _, _ := m.lay()
	return d
}

func (m Model) total() int {
	_, w, _ := m.lay()
	return w
}

// cursor is the cursor to draw. A Cursor left at one of the two built-in
// defaults (the thin bar of New, the block of NewTypewriter) follows the
// theme's glyphs, so an ASCII theme gets an ASCII cursor; a Cursor the caller
// set to anything else is drawn as given.
func (m Model) cursor() string {
	g := m.themed().GlyphSet()
	switch m.Cursor {
	case theme.UnicodeGlyphSet().Cursor:
		return g.Cursor
	case theme.UnicodeGlyphSet().CursorBlock:
		return g.CursorBlock
	}
	return m.Cursor
}

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}
