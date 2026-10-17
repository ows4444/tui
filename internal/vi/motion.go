package vi

import "unicode"

// motion returns where motion r goes with count n; is is false when r is not a
// motion.
func (s *State) motion(b Buffer, r rune, n int, has bool) (t target, is bool) {
	p := b.Cursor()
	switch r {
	case 'h':
		return target{pos: max(p-n, lineStart(b, p)), ok: true}, true
	case 'l':
		return target{pos: min(p+n, lineEnd(b, p)), ok: true}, true
	case '0':
		return target{pos: lineStart(b, p), ok: true}, true
	case '^':
		return target{pos: firstNonBlank(b, p), ok: true}, true
	case '$':
		q := p
		for i := 1; i < n; i++ {
			e := lineEnd(b, q)
			if e >= b.Len() {
				break
			}
			q = e + 1
		}
		return target{pos: lineEnd(b, q), ok: true}, true
	case 'w', 'W':
		big := r == 'W'
		if s.op == 'c' && class(b, p, big) != 0 {
			return target{pos: wordEnd(b, p, big, n), incl: true, ok: true}, true
		}
		q := p
		for i := 0; i < n; i++ {
			q = wordFwd(b, q, big, s.op != 0 && i == n-1)
		}
		return target{pos: q, ok: true}, true
	case 'b', 'B':
		q := p
		for i := 0; i < n; i++ {
			q = wordBack(b, q, r == 'B')
		}
		return target{pos: q, ok: true}, true
	case 'e', 'E':
		return target{pos: wordEnd(b, p, r == 'E', n), incl: true, ok: true}, true
	case 'j', 'k':
		if s.want == 0 {
			s.want = p - lineStart(b, p) + 1
		}
		s.keep = true
		q, moved := p, 0
		for i := 0; i < n; i++ {
			nq, ok := stepLine(b, q, r == 'j')
			if !ok {
				break
			}
			q, moved = nq, moved+1
		}
		if moved == 0 && s.op != 0 {
			return target{}, true
		}
		return target{pos: min(lineStart(b, q)+s.want-1, lineEnd(b, q)), line: true, ok: true}, true
	case 'G':
		ln := 1 << 30
		if has {
			ln = n
		}
		return target{pos: firstNonBlank(b, lineAt(b, ln)), line: true, ok: true}, true
	case ';', ',':
		if s.find.kind == 0 {
			return target{}, true
		}
		kind := s.find.kind
		if r == ',' {
			kind = map[rune]rune{'f': 'F', 'F': 'f', 't': 'T', 'T': 't'}[kind]
		}
		return findTarget(b, kind, s.find.ch, n, true), true
	}
	return target{}, false
}

// findTarget is the f, F, t or T motion. repeat skips a match right next to the
// cursor for t and T, so ; keeps moving.
func findTarget(b Buffer, kind, ch rune, n int, repeat bool) target {
	p := b.Cursor()
	ls, le := lineStart(b, p), lineEnd(b, p)
	fwd := kind == 'f' || kind == 't'
	till := kind == 't' || kind == 'T'
	step := 1
	if !fwd {
		step = -1
	}
	pos := p
	for i := 0; i < n; i++ {
		q := pos
		for {
			q += step
			if q >= le || q < ls {
				return target{}
			}
			if b.At(q) == ch && !(till && repeat && i == 0 && q == pos+step) {
				break
			}
		}
		pos = q
	}
	switch kind {
	case 't':
		pos--
	case 'T':
		pos++
	}
	return target{pos: pos, incl: fwd, ok: true}
}

// slice is b[from:to] as a string.
func slice(b Buffer, from, to int) string {
	rs := make([]rune, 0, max(to-from, 0))
	for i := from; i < to; i++ {
		rs = append(rs, b.At(i))
	}
	return string(rs)
}

func lineStart(b Buffer, p int) int {
	for p > 0 && b.At(p-1) != '\n' {
		p--
	}
	return p
}

func lineEnd(b Buffer, p int) int {
	n := b.Len()
	for p < n && b.At(p) != '\n' {
		p++
	}
	return p
}

// firstNonBlank is the position of the first character of p's line that is not
// a space or tab (the line's end when it has none).
func firstNonBlank(b Buffer, p int) int {
	s, e := lineStart(b, p), lineEnd(b, p)
	for s < e && (b.At(s) == ' ' || b.At(s) == '\t') {
		s++
	}
	return s
}

// stepLine is the start of the line after (down) or before p's line.
func stepLine(b Buffer, p int, down bool) (int, bool) {
	if down {
		e := lineEnd(b, p)
		if e >= b.Len() {
			return p, false
		}
		return e + 1, true
	}
	ls := lineStart(b, p)
	if ls == 0 {
		return p, false
	}
	return lineStart(b, ls-1), true
}

// lineAt is the start of line number ln (1-based), or of the last line.
func lineAt(b Buffer, ln int) int {
	p := 0
	for i := 1; i < ln; i++ {
		nq, ok := stepLine(b, p, true)
		if !ok {
			break
		}
		p = nq
	}
	return p
}

// class is 0 for blanks, 1 for word characters (everything non-blank when big)
// and 2 for punctuation.
func class(b Buffer, p int, big bool) int {
	if p >= b.Len() {
		return 0
	}
	r := b.At(p)
	switch {
	case r == ' ' || r == '\t' || r == '\n':
		return 0
	case big, unicode.IsLetter(r), unicode.IsDigit(r), r == '_':
		return 1
	}
	return 2
}

// wordFwd is the start of the next word. An empty line counts as a word. With
// stopEOL it stops at the end of the line instead of going to the next one
// (dw on the last word of a line).
func wordFwd(b Buffer, p int, big, stopEOL bool) int {
	n := b.Len()
	if p >= n {
		return n
	}
	if b.At(p) == '\n' {
		if stopEOL {
			return p
		}
		p++
	} else if c := class(b, p, big); c != 0 {
		for p < n && class(b, p, big) == c {
			p++
		}
	}
	for p < n {
		switch r := b.At(p); {
		case r == '\n':
			if stopEOL || p > 0 && b.At(p-1) == '\n' {
				return p
			}
			p++
		case r == ' ' || r == '\t':
			p++
		default:
			return p
		}
	}
	return n
}

// wordBack is the start of the word before p.
func wordBack(b Buffer, p int, big bool) int {
	if p <= 0 {
		return 0
	}
	p--
	for p > 0 && class(b, p, big) == 0 {
		if b.At(p) == '\n' && b.At(p-1) == '\n' {
			return p
		}
		p--
	}
	c := class(b, p, big)
	for p > 0 && class(b, p-1, big) == c && c != 0 {
		p--
	}
	return p
}

// wordEnd is the last character of the nth word end from p.
func wordEnd(b Buffer, p int, big bool, n int) int {
	last := b.Len() - 1
	for i := 0; i < n; i++ {
		p++
		for p < last && class(b, p, big) == 0 {
			p++
		}
		c := class(b, p, big)
		for p < last && class(b, p+1, big) == c {
			p++
		}
	}
	return min(max(p, 0), max(last, 0))
}

// textObject returns [from, to) for the object named r around the cursor:
// iw aw iW aW, quotes, and brackets. around adds the delimiters, or the space
// after a word.
func textObject(b Buffer, around bool, r rune) (from, to int, ok bool) {
	p := b.Cursor()
	switch r {
	case 'w', 'W':
		return wordObject(b, p, around, r == 'W')
	case '"', '\'', '`':
		return quoteObject(b, p, around, r)
	}
	pairs := map[rune][2]rune{'(': {'(', ')'}, ')': {'(', ')'}, 'b': {'(', ')'}, '[': {'[', ']'}, ']': {'[', ']'},
		'{': {'{', '}'}, '}': {'{', '}'}, 'B': {'{', '}'}, '<': {'<', '>'}, '>': {'<', '>'}}
	pr, found := pairs[r]
	if !found {
		return 0, 0, false
	}
	return bracketObject(b, p, around, pr[0], pr[1])
}

func wordObject(b Buffer, p int, around, big bool) (int, int, bool) {
	if p >= b.Len() || b.At(p) == '\n' {
		return 0, 0, false
	}
	ls, le := lineStart(b, p), lineEnd(b, p)
	c := class(b, p, big)
	a, e := p, p+1
	for a > ls && class(b, a-1, big) == c {
		a--
	}
	for e < le && class(b, e, big) == c {
		e++
	}
	if !around {
		return a, e, true
	}
	if c == 0 { // blanks plus the word after them
		if e < le {
			c2 := class(b, e, big)
			for e < le && class(b, e, big) == c2 {
				e++
			}
		}
		return a, e, true
	}
	t := e
	for t < le && class(b, t, big) == 0 {
		t++
	}
	if t > e {
		return a, t, true
	}
	for a > ls && class(b, a-1, big) == 0 {
		a--
	}
	return a, e, true
}

func quoteObject(b Buffer, p int, around bool, q rune) (int, int, bool) {
	ls, le := lineStart(b, p), lineEnd(b, p)
	var at []int
	for i := ls; i < le; i++ {
		if b.At(i) == q {
			at = append(at, i)
		}
	}
	for i := 0; i+1 < len(at); i += 2 {
		if at[i+1] >= p {
			if around {
				return at[i], at[i+1] + 1, true
			}
			return at[i] + 1, at[i+1], true
		}
	}
	return 0, 0, false
}

func bracketObject(b Buffer, p int, around bool, open, cl rune) (int, int, bool) {
	o, d := -1, 0
	for i := min(p, b.Len()-1); i >= 0; i-- {
		switch c := b.At(i); {
		case c == cl && i != p:
			d++
		case c == open:
			if d == 0 {
				o = i
			} else {
				d--
			}
		}
		if o >= 0 {
			break
		}
	}
	if o < 0 {
		return 0, 0, false
	}
	d = 0
	for i := o + 1; i < b.Len(); i++ {
		switch c := b.At(i); {
		case c == open:
			d++
		case c == cl:
			if d == 0 {
				if around {
					return o, i + 1, true
				}
				return o + 1, i, true
			}
			d--
		}
	}
	return 0, 0, false
}
