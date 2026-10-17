// Package vi is the modal editing engine behind the Vi option of textarea and
// textinput. It knows nothing about widgets: a widget gives it a Buffer (the
// text as runes, a cursor, and edits that can be undone) and feeds it keys.
// In Normal and Visual modes it consumes every key; in Insert mode it returns
// the keys it does not own so the widget types them as usual.
//
// It works on runes, not grapheme clusters. Counts, the operators d c y with
// motions and text objects, p P, J, r, ~, ., u and Ctrl+R are supported; macros,
// registers other than the unnamed one, marks, searches, and block-wise Visual
// mode are not.
package vi

import (
	"strconv"
	"strings"
	"unicode"
)

// Mode is the editing mode.
type Mode uint8

// The modes.
const (
	Normal Mode = iota
	Insert
	Visual
	VisualLine
)

// Special is a key that has no rune.
type Special uint8

// The special keys the engine tells apart.
const (
	None Special = iota
	Esc
	Enter
	Backspace
	Delete
	Left
	Right
	Up
	Down
	Home
	End
	Redo // Ctrl+R
)

// Key is one key press: a rune, or a Special with R zero.
type Key struct {
	R rune
	S Special
}

// Buffer is the text being edited.
type Buffer interface {
	Len() int
	At(i int) rune
	Cursor() int
	SetCursor(i int)
	// Replace swaps [from, to) for ins as one undoable edit. It does not move
	// the cursor.
	Replace(from, to int, ins []rune)
	// Select shows [anchor, cursor) as selected, or clears the selection.
	Select(anchor int, on bool)
	// Begin and End bracket one command: the edits between them undo together.
	// An insert session stays open until the Esc that ends it.
	Begin()
	End()
	Undo()
	Redo()
}

// Result is what a key did beyond editing the buffer.
type Result struct {
	// Handled is false only for a key in Insert mode that the widget should type.
	Handled bool
	// Yank is text a y command copied, for the clipboard; Yanked says there is
	// one.
	Yank   string
	Yanked bool
}

// State is the engine's memory. The zero value is Normal mode with nothing
// pending. It is a plain value so a widget Model can copy it.
type State struct {
	mode    Mode
	count   int  // digits typed before the next motion or command
	opCount int  // digits typed before the operator
	op      rune // pending operator: d, c or y
	prefix  rune // pending first key of g, f, F, t, T, r, i, a
	reg     []rune
	regLine bool
	find    struct{ kind, ch rune }
	want    int // goal column + 1 for j and k; 0 when none
	keep    bool
	anchor  int // where Visual mode began

	cur       []Key // keys of the command being typed
	last      []Key // keys of the last change, for .
	changing  bool
	replaying bool

	yank   string
	yanked bool
}

// Mode returns the current mode.
func (s State) Mode() Mode { return s.mode }

// SetMode puts the engine in Normal or Insert mode, dropping anything pending.
func (s *State) SetMode(m Mode) {
	s.mode = m
	s.reset()
	s.cur, s.changing = nil, false
}

// Pending is the keys typed so far of a command that is not complete, such as
// "2d" or "g", for a status line.
func (s State) Pending() string {
	var b strings.Builder
	num := func(n int) {
		if n > 0 {
			b.WriteString(strconv.Itoa(n))
		}
	}
	num(s.opCount)
	if s.op != 0 {
		b.WriteRune(s.op)
	}
	num(s.count)
	if s.prefix != 0 {
		b.WriteRune(s.prefix)
	}
	return b.String()
}

// Handle applies k to b.
func (s *State) Handle(b Buffer, k Key) Result {
	if s.mode == Insert {
		return s.insert(b, k)
	}
	if !s.replaying {
		b.Begin()
		s.cur = appendKey(s.cur, k)
	}
	s.normal(b, k)
	s.finish(b)
	res := Result{Handled: true, Yank: s.yank, Yanked: s.yanked}
	s.yank, s.yanked = "", false
	return res
}

func appendKey(ks []Key, k Key) []Key {
	out := make([]Key, len(ks), len(ks)+1)
	copy(out, ks)
	return append(out, k)
}

func (s *State) reset() { s.count, s.opCount, s.op, s.prefix = 0, 0, 0, 0 }

// total is the count to use: the two counts multiplied, at least 1, and
// whether any was typed.
func (s *State) total() (n int, has bool) {
	return max(s.count, 1) * max(s.opCount, 1), s.count > 0 || s.opCount > 0
}

// finish tidies up after a key: the cursor stays on a character, the selection
// follows the mode, and a completed change is remembered for ".".
func (s *State) finish(b Buffer) {
	if s.mode != Insert {
		s.clamp(b)
	}
	s.sync(b)
	if !s.keep {
		s.want = 0
	}
	s.keep = false
	if s.replaying || s.mode == Insert {
		return
	}
	b.End()
	if s.op != 0 || s.prefix != 0 || s.count != 0 || s.opCount != 0 {
		return
	}
	if s.changing && s.mode == Normal {
		s.last = s.cur
	}
	s.cur, s.changing = nil, false
}

// clamp keeps the cursor on a character: not past the last one of its line.
func (s *State) clamp(b Buffer) {
	p := min(max(b.Cursor(), 0), b.Len())
	if ls := lineStart(b, p); p == lineEnd(b, p) && p > ls {
		p--
	}
	b.SetCursor(p)
}

func (s *State) sync(b Buffer) {
	switch s.mode {
	case Visual:
		b.Select(s.anchor, true)
	case VisualLine:
		a := lineStart(b, s.anchor)
		if b.Cursor() < a {
			a = lineEnd(b, s.anchor)
		}
		b.Select(a, true)
	default:
		b.Select(0, false)
	}
}

// insert handles a key in Insert mode: Esc leaves it; any other key is the
// widget's to type, and is remembered so "." can type it again.
func (s *State) insert(b Buffer, k Key) Result {
	if k.S == Esc {
		s.mode = Normal
		if p := b.Cursor(); p > lineStart(b, p) {
			b.SetCursor(p - 1)
		}
		if !s.replaying {
			b.End()
			if s.changing {
				s.last = appendKey(s.cur, k)
			}
			s.cur, s.changing = nil, false
		}
		b.Select(0, false)
		return Result{Handled: true}
	}
	if s.replaying {
		s.typeKey(b, k)
		return Result{Handled: true}
	}
	if s.changing {
		s.cur = appendKey(s.cur, k)
	}
	return Result{}
}

// typeKey types k into b, for a replayed insert session.
func (s *State) typeKey(b Buffer, k Key) {
	p := b.Cursor()
	switch {
	case k.S == None && k.R != 0:
		b.Replace(p, p, []rune{k.R})
		b.SetCursor(p + 1)
	case k.S == Enter:
		b.Replace(p, p, []rune{'\n'})
		b.SetCursor(p + 1)
	case k.S == Backspace && p > 0:
		b.Replace(p-1, p, nil)
		b.SetCursor(p - 1)
	}
}

// normal handles a key in Normal or Visual mode.
func (s *State) normal(b Buffer, k Key) {
	if k.S == Esc {
		s.reset()
		s.mode = Normal
		return
	}
	if s.prefix != 0 {
		p := s.prefix
		s.prefix = 0
		s.prefixed(b, p, k)
		return
	}
	r := k.R
	switch k.S {
	case Left, Backspace:
		r = 'h'
	case Right:
		r = 'l'
	case Up:
		r = 'k'
	case Down, Enter:
		r = 'j'
	case Home:
		r = '0'
	case End:
		r = '$'
	case Delete:
		r = 'x'
	case Redo:
		s.reset()
		b.Redo()
		return
	}
	if r == 0 {
		return
	}
	if r == ' ' {
		r = 'l'
	}
	if r >= '1' && r <= '9' || r == '0' && s.count > 0 {
		s.count = s.count*10 + int(r-'0')
		return
	}
	n, has := s.total()
	vis := s.mode == Visual || s.mode == VisualLine
	switch r {
	case 'd', 'c', 'y':
		switch {
		case vis:
			s.visualOp(b, r)
		case s.op == 0:
			s.op, s.opCount, s.count = r, s.count, 0
		case s.op == r:
			s.lines(b, n)
		default:
			s.reset()
		}
		return
	case 'g', 'f', 'F', 't', 'T':
		s.prefix = r
		return
	case 'r':
		if vis || s.op != 0 {
			s.reset()
			return
		}
		s.prefix = r
		return
	case 'i', 'a':
		if s.op != 0 || vis {
			s.prefix = r
			return
		}
	}
	if t, is := s.motion(b, r, n, has); is {
		s.move(b, t)
		return
	}
	if s.op != 0 {
		s.reset()
		return
	}
	s.command(b, r, n, vis)
}

// target is where a motion goes and how an operator treats it.
type target struct {
	pos  int
	line bool // the operator takes whole lines
	incl bool // the operator includes the character at pos
	ok   bool
}

// move goes to t, or applies the pending operator between the cursor and t.
func (s *State) move(b Buffer, t target) {
	op := s.op
	s.reset()
	if !t.ok {
		return
	}
	if op == 0 {
		b.SetCursor(t.pos)
		return
	}
	from, to := b.Cursor(), t.pos
	if from > to {
		from, to = to, from
	}
	if t.incl {
		to = min(to+1, b.Len())
	}
	s.apply(b, op, from, to, t.line)
}

// lines applies the pending operator to n lines from the cursor (dd, cc, yy).
func (s *State) lines(b Buffer, n int) {
	op := s.op
	s.reset()
	q := b.Cursor()
	for i := 1; i < n; i++ {
		nq, ok := stepLine(b, q, true)
		if !ok {
			break
		}
		q = nq
	}
	s.apply(b, op, b.Cursor(), q, true)
}

// apply runs operator op on [from, to), or on the lines holding from and to.
func (s *State) apply(b Buffer, op rune, from, to int, line bool) {
	n := b.Len()
	cur := b.Cursor()
	if line {
		la, lb := lineStart(b, from), lineEnd(b, to)
		text := slice(b, la, lb) + "\n"
		s.reg, s.regLine = []rune(text), true
		switch op {
		case 'y':
			s.yank, s.yanked = text, true
			b.SetCursor(min(from, cur))
		case 'd':
			s.changing = true
			at := la
			if lb < n {
				b.Replace(la, lb+1, nil)
			} else if la > 0 {
				b.Replace(la-1, lb, nil)
				at = la - 1
			} else {
				b.Replace(la, lb, nil)
			}
			b.SetCursor(firstNonBlank(b, min(at, b.Len())))
		case 'c':
			s.changing = true
			b.Replace(la, lb, nil)
			b.SetCursor(la)
			s.mode = Insert
		}
		return
	}
	text := slice(b, from, to)
	if to > from {
		s.reg, s.regLine = []rune(text), false
	}
	switch op {
	case 'y':
		if to > from {
			s.yank, s.yanked = text, true
		}
		b.SetCursor(from)
	case 'd':
		s.changing = true
		b.Replace(from, to, nil)
		b.SetCursor(from)
	case 'c':
		s.changing = true
		b.Replace(from, to, nil)
		b.SetCursor(from)
		s.mode = Insert
	}
}

// visualOp applies d, c or y to the Visual selection.
func (s *State) visualOp(b Buffer, op rune) {
	lo, hi := min(s.anchor, b.Cursor()), max(s.anchor, b.Cursor())
	line := s.mode == VisualLine
	if !line {
		hi = min(hi+1, b.Len())
	}
	s.mode = Normal
	s.reset()
	s.apply(b, op, lo, hi, line)
	s.changing = false
}

// prefixed completes a command whose first key was p.
func (s *State) prefixed(b Buffer, p rune, k Key) {
	r := k.R
	n, has := s.total()
	switch p {
	case 'g':
		if r == 'g' {
			ln := 1
			if has {
				ln = n
			}
			s.move(b, target{pos: firstNonBlank(b, lineAt(b, ln)), line: true, ok: true})
			return
		}
	case 'f', 'F', 't', 'T':
		if r != 0 {
			s.find.kind, s.find.ch = p, r
			s.move(b, findTarget(b, p, r, n, false))
			return
		}
	case 'r':
		if r != 0 {
			s.replaceChars(b, r, n)
			return
		}
	case 'i', 'a':
		if from, to, ok := textObject(b, p == 'a', r); ok {
			s.object(b, from, to)
			return
		}
	}
	s.reset()
}

// object applies a text object [from, to): the pending operator takes it, and
// in Visual mode it becomes the selection.
func (s *State) object(b Buffer, from, to int) {
	if s.op != 0 {
		op := s.op
		s.reset()
		s.apply(b, op, from, to, false)
		return
	}
	if s.mode == Visual || s.mode == VisualLine {
		s.reset()
		s.anchor = from
		b.SetCursor(max(to-1, from))
	}
}

func (s *State) replaceChars(b Buffer, ch rune, n int) {
	s.reset()
	p := b.Cursor()
	if p+n > lineEnd(b, p) {
		return
	}
	s.changing = true
	b.Replace(p, p+n, []rune(strings.Repeat(string(ch), n)))
	b.SetCursor(p + n - 1)
}

// command runs a command that is not a motion or an operator.
func (s *State) command(b Buffer, r rune, n int, vis bool) {
	s.reset()
	p := b.Cursor()
	if vis {
		s.visualCommand(b, r)
		return
	}
	switch r {
	case 'x':
		s.synth(b, 'd', 'l', n)
	case 'X':
		s.synth(b, 'd', 'h', n)
	case 'D':
		s.synth(b, 'd', '$', n)
	case 'C':
		s.synth(b, 'c', '$', n)
	case 's':
		s.synth(b, 'c', 'l', n)
	case 'S':
		s.op, s.opCount = 'c', n
		s.lines(b, n)
	case 'Y':
		s.op, s.opCount = 'y', n
		s.lines(b, n)
	case 'p', 'P':
		s.put(b, r == 'p', n)
	case 'J':
		s.join(b, max(n, 2))
	case '~':
		e := min(p+n, lineEnd(b, p))
		s.toggleCase(b, p, e)
		b.SetCursor(e)
	case 'u':
		b.Undo()
	case '.':
		s.repeat(b)
	case 'i':
		s.enterInsert(b, p)
	case 'a':
		if p < lineEnd(b, p) {
			p++
		}
		s.enterInsert(b, p)
	case 'I':
		s.enterInsert(b, firstNonBlank(b, p))
	case 'A':
		s.enterInsert(b, lineEnd(b, p))
	case 'o':
		e := lineEnd(b, p)
		b.Replace(e, e, []rune{'\n'})
		s.enterInsert(b, e+1)
	case 'O':
		ls := lineStart(b, p)
		b.Replace(ls, ls, []rune{'\n'})
		s.enterInsert(b, ls)
	case 'v':
		s.mode, s.anchor = Visual, p
	case 'V':
		s.mode, s.anchor = VisualLine, p
	}
}

func (s *State) enterInsert(b Buffer, at int) {
	b.SetCursor(at)
	s.mode, s.changing = Insert, true
}

// synth runs operator op over motion m with count n, for x, D and the like.
func (s *State) synth(b Buffer, op, m rune, n int) {
	s.op = op
	t, _ := s.motion(b, m, n, n > 1)
	s.move(b, t)
}

func (s *State) visualCommand(b Buffer, r rune) {
	p := b.Cursor()
	switch r {
	case 'v', 'V':
		to := Visual
		if r == 'V' {
			to = VisualLine
		}
		if s.mode == to {
			to = Normal
		}
		s.mode = to
	case 'o':
		s.anchor, p = p, s.anchor
		b.SetCursor(p)
	case 'x':
		s.visualOp(b, 'd')
	case 's':
		s.visualOp(b, 'c')
	case 'X', 'D':
		s.mode = VisualLine
		s.visualOp(b, 'd')
	case 'Y':
		s.mode = VisualLine
		s.visualOp(b, 'y')
	case 'C', 'S', 'R':
		s.mode = VisualLine
		s.visualOp(b, 'c')
	case '~':
		lo, hi := min(s.anchor, p), min(max(s.anchor, p)+1, b.Len())
		s.mode = Normal
		s.toggleCase(b, lo, hi)
		b.SetCursor(lo)
		s.changing = false
	}
}

// repeat runs the last change again.
func (s *State) repeat(b Buffer) {
	if len(s.last) == 0 || s.replaying {
		return
	}
	keys := s.last
	s.replaying = true
	for _, k := range keys {
		s.Handle(b, k)
	}
	if s.mode == Insert {
		s.insert(b, Key{S: Esc})
	}
	s.replaying = false
	s.changing = false
	s.reset()
}

// put pastes the register n times after or before the cursor.
func (s *State) put(b Buffer, after bool, n int) {
	if len(s.reg) == 0 {
		return
	}
	s.changing = true
	text := []rune(strings.Repeat(string(s.reg), n))
	p := b.Cursor()
	if s.regLine {
		if after {
			e := lineEnd(b, p)
			if e == b.Len() {
				ins := append([]rune{'\n'}, []rune(strings.TrimSuffix(string(text), "\n"))...)
				b.Replace(e, e, ins)
				b.SetCursor(firstNonBlank(b, e+1))
				return
			}
			b.Replace(e+1, e+1, text)
			b.SetCursor(firstNonBlank(b, e+1))
			return
		}
		ls := lineStart(b, p)
		b.Replace(ls, ls, text)
		b.SetCursor(firstNonBlank(b, ls))
		return
	}
	if after && p < lineEnd(b, p) {
		p++
	}
	b.Replace(p, p, text)
	b.SetCursor(p + len(text) - 1)
}

// join joins n lines from the cursor's line with single spaces.
func (s *State) join(b Buffer, n int) {
	at := b.Cursor()
	for i := 1; i < n; i++ {
		e := lineEnd(b, b.Cursor())
		if e >= b.Len() {
			break
		}
		s.changing = true
		q := e + 1
		for q < b.Len() && (b.At(q) == ' ' || b.At(q) == '\t') {
			q++
		}
		sep := []rune{' '}
		if e == lineStart(b, e) || q >= b.Len() || b.At(q) == '\n' || b.At(q) == ')' || b.At(e-1) == ' ' {
			sep = nil
		}
		b.Replace(e, q, sep)
		at = e
		b.SetCursor(e)
	}
	b.SetCursor(at)
}

func (s *State) toggleCase(b Buffer, from, to int) {
	if to <= from {
		return
	}
	s.changing = true
	rs := []rune(slice(b, from, to))
	for i, r := range rs {
		switch {
		case unicode.IsUpper(r):
			rs[i] = unicode.ToLower(r)
		case unicode.IsLower(r):
			rs[i] = unicode.ToUpper(r)
		}
	}
	b.Replace(from, to, rs)
}
