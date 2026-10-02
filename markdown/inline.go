package markdown

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// attrs are the inline formatting flags a piece of text carries.
type attrs struct {
	bold, italic bool
	code         bool // inline code span
	link         bool // link text
	muted        bool // a link's "(url)"
}

// merge returns a with b's flags added (inheritance from an enclosing span).
func (a attrs) merge(b attrs) attrs {
	a.bold = a.bold || b.bold
	a.italic = a.italic || b.italic
	a.code = a.code || b.code
	a.link = a.link || b.link
	a.muted = a.muted || b.muted
	return a
}

// piece is a run of text with uniform attrs, or (brk) a hard line break.
type piece struct {
	text string
	a    attrs
	brk  bool
}

type nodeKind int

const (
	nText   nodeKind = iota // plain text
	nPieces                 // finished pieces (code span, link, hard break)
	nDelim                  // a run of * or _ awaiting emphasis matching
	nEmph                   // matched emphasis wrapping children
)

type node struct {
	kind   nodeKind
	text   string
	pieces []piece

	ch                byte
	n, orig           int // delimiters left / originally in the run
	canOpen, canClose bool
	strong            bool

	first, last  *node // an nEmph's children, as a sibling list
	prev, next   *node // siblings
	dprev, dnext *node // links among the unmatched delimiter runs (nDelim only)
}

// nodeList is a doubly linked list, so wrapping a run of nodes in an
// emphasis node is O(1) instead of copying slices.
type nodeList struct{ head, tail *node }

func (l *nodeList) push(n *node) {
	n.prev = l.tail
	if l.tail != nil {
		l.tail.next = n
	} else {
		l.head = n
	}
	l.tail = n
}

// Limits that keep hostile input from making the parser super-linear: link
// text, destination and title scans give up beyond these many bytes (the
// brackets then stay literal).
const (
	maxLinkText  = 1000
	maxLinkDest  = 2000
	maxLinkTitle = 1000
)

// parseInline turns inline markdown (paragraph text with "\n" for line
// ends) into styled pieces. It implements a simplified CommonMark inline
// pass: escapes, code spans, autolinks, inline links and images, and
// emphasis by the delimiter-run algorithm with flanking rules. Anything it
// doesn't recognise stays literal text.
func parseInline(s string) []piece {
	l, firstDelim := tokenize(s)
	processEmphasis(firstDelim)
	out := make([]piece, 0, 16)
	flatten(l.head, attrs{}, &out)
	return out
}

// tokenize splits s into nodes, and returns the first delimiter run (the
// runs are also chained through dprev/dnext for processEmphasis).
func tokenize(s string) (l nodeList, firstDelim *node) {
	// Pending text is a substring of s while contiguous (no allocation); it
	// moves into sb once a second, non-adjacent piece is appended.
	var text string
	var sb strings.Builder
	building := false
	var lastDelim *node
	flush := func() {
		if building {
			l.push(&node{kind: nText, text: sb.String()})
			sb.Reset()
			building = false
		} else if text != "" {
			l.push(&node{kind: nText, text: text})
		}
		text = ""
	}
	add := func(x string) {
		switch {
		case building:
			sb.WriteString(x)
		case text == "":
			text = x
		default:
			sb.WriteString(text)
			sb.WriteString(x)
			building = true
		}
	}
	hardBreak := func() *node { return &node{kind: nPieces, pieces: []piece{{brk: true}}} }
	pieces := func(ps ...piece) *node { return &node{kind: nPieces, pieces: ps} }

	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == '\\':
			switch {
			case i+1 < len(s) && s[i+1] == '\n':
				flush()
				l.push(hardBreak())
				i += 2
			case i+1 < len(s) && isASCIIPunct(s[i+1]):
				add(s[i+1 : i+2])
				i += 2
			default:
				add(s[i : i+1])
				i++
			}

		case c == '\n':
			t := text
			if building {
				t = sb.String()
			}
			trimmed := strings.TrimRight(t, " ")
			sb.Reset()
			building = false
			text = trimmed
			flush()
			if len(t)-len(trimmed) >= 2 {
				l.push(hardBreak())
			} else {
				l.push(&node{kind: nText, text: " "})
			}
			i++

		case c == '`':
			content, end, ok := codeSpan(s, i)
			if !ok {
				add(s[i:end])
				i = end
				continue
			}
			flush()
			l.push(pieces(piece{text: content, a: attrs{code: true}}))
			i = end

		case c == '*' || c == '_':
			j := i
			for j < len(s) && s[j] == c {
				j++
			}
			flush()
			d := delimNode(s, i, j)
			d.dprev = lastDelim
			if lastDelim != nil {
				lastDelim.dnext = d
			} else {
				firstDelim = d
			}
			lastDelim = d
			l.push(d)
			i = j

		case c == '!' && i+1 < len(s) && s[i+1] == '[':
			if ps, end, ok := parseLink(s, i+1); ok {
				flush()
				l.push(pieces(ps...))
				i = end
				continue
			}
			add(s[i : i+1])
			i++

		case c == '[':
			if ps, end, ok := parseLink(s, i); ok {
				flush()
				l.push(pieces(ps...))
				i = end
				continue
			}
			add(s[i : i+1])
			i++

		case c == '<':
			if url, end, ok := autolink(s, i); ok {
				flush()
				l.push(pieces(piece{text: url, a: attrs{link: true}}))
				i = end
				continue
			}
			add(s[i : i+1])
			i++

		default:
			j := i + 1
			for j < len(s) && !isInlineSpecial(s[j]) {
				j++
			}
			add(s[i:j])
			i = j
		}
	}
	flush()
	return l, firstDelim
}

// isInlineSpecial reports whether c can start a construct in tokenize.
func isInlineSpecial(c byte) bool {
	switch c {
	case '\\', '\n', '`', '*', '_', '!', '[', '<':
		return true
	}
	return false
}

// delimNode classifies the delimiter run s[i:j] by CommonMark's flanking
// rules; the start and end of the text count as whitespace.
func delimNode(s string, i, j int) *node {
	before, after := ' ', ' '
	if i > 0 {
		before, _ = utf8.DecodeLastRuneInString(s[:i])
	}
	if j < len(s) {
		after, _ = utf8.DecodeRuneInString(s[j:])
	}
	beforeWS, afterWS := unicode.IsSpace(before), unicode.IsSpace(after)
	beforeP, afterP := isPunct(before), isPunct(after)

	left := !afterWS && (!afterP || beforeWS || beforeP)
	right := !beforeWS && (!beforeP || afterWS || afterP)

	n := &node{kind: nDelim, ch: s[i], n: j - i, orig: j - i}
	if s[i] == '*' {
		n.canOpen, n.canClose = left, right
	} else { // '_' does not open or close inside a word
		n.canOpen = left && (!right || beforeP)
		n.canClose = right && (!left || afterP)
	}
	return n
}

// processEmphasis matches delimiter runs into emphasis and strong nodes,
// innermost first, per CommonMark's "process emphasis" (including the rule
// of three). Unmatched delimiters stay as literal text. Delimiters live in
// a linked list and a failed search records a per-kind "openers bottom",
// so the whole pass is close to linear.
func processEmphasis(firstDelim *node) {
	type bkey struct {
		ch   byte
		open bool
		mod  int
	}
	bottoms := map[bkey]*node{} // presence means "searched down to this node"
	remove := func(d *node) {
		if d.dprev != nil {
			d.dprev.dnext = d.dnext
		}
		if d.dnext != nil {
			d.dnext.dprev = d.dprev
		}
	}

	for c := firstDelim; c != nil; {
		if !c.canClose {
			c = c.dnext
			continue
		}
		k := bkey{c.ch, c.canOpen, c.orig % 3}
		bottom, searched := bottoms[k]

		var o *node
		for p := c.dprev; p != nil && !(searched && p == bottom); p = p.dprev {
			if p.ch != c.ch || !p.canOpen {
				continue
			}
			if (p.canClose || c.canOpen) && (p.orig+c.orig)%3 == 0 &&
				!(p.orig%3 == 0 && c.orig%3 == 0) {
				continue
			}
			o = p
			break
		}
		if o == nil {
			bottoms[k] = c.dprev
			next := c.dnext
			if !c.canOpen {
				remove(c)
			}
			c = next
			continue
		}

		use := 1
		if o.n >= 2 && c.n >= 2 {
			use = 2
		}
		emph := &node{kind: nEmph, strong: use == 2}
		if o.next != c { // detach everything between o and c as the children
			emph.first, emph.last = o.next, c.prev
			emph.first.prev, emph.last.next = nil, nil
		}
		o.next, emph.prev, emph.next, c.prev = emph, o, c, emph
		o.dnext, c.dprev = c, o // delimiters in between are now literal
		o.n -= use
		c.n -= use
		if o.n == 0 {
			remove(o)
		}
		if c.n == 0 {
			next := c.dnext
			remove(c)
			c = next
		}
	}
}

// flatten appends the pieces for the sibling list starting at first,
// applying inherited attrs.
func flatten(first *node, inherit attrs, out *[]piece) {
	for n := first; n != nil; n = n.next {
		switch n.kind {
		case nText:
			*out = append(*out, piece{text: n.text, a: inherit})
		case nPieces:
			for _, p := range n.pieces {
				p.a = p.a.merge(inherit)
				*out = append(*out, p)
			}
		case nDelim:
			if n.n > 0 {
				*out = append(*out, piece{text: strings.Repeat(string(n.ch), n.n), a: inherit})
			}
		case nEmph:
			a := inherit
			if n.strong {
				a.bold = true
			} else {
				a.italic = true
			}
			flatten(n.first, a, out)
		}
	}
}

// codeSpan parses a code span starting at s[i] (a backtick run). On a
// match it returns the normalised content and the index after the closing
// run; otherwise ok is false and end is the index after the opening run,
// which is then literal.
func codeSpan(s string, i int) (content string, end int, ok bool) {
	j := i
	for j < len(s) && s[j] == '`' {
		j++
	}
	n := j - i
	for k := j; k < len(s); {
		if s[k] != '`' {
			k++
			continue
		}
		m := k
		for m < len(s) && s[m] == '`' {
			m++
		}
		if m-k == n {
			c := strings.ReplaceAll(s[j:k], "\n", " ")
			if len(c) >= 2 && c[0] == ' ' && c[len(c)-1] == ' ' && strings.Trim(c, " ") != "" {
				c = c[1 : len(c)-1]
			}
			return c, m, true
		}
		k = m
	}
	return "", j, false
}

// parseLink parses an inline link or image starting at the '[' at s[i]:
// [text](destination "title"). It returns the pieces to show — the link
// text (underlined) and, unless it would repeat the text, the url in
// parentheses — and the index after the closing ')'.
func parseLink(s string, i int) (ps []piece, end int, ok bool) {
	depth := 0
	j := i
	closeAt := -1
scan:
	for j < len(s) && j-i <= maxLinkText {
		switch s[j] {
		case '\\':
			j += 2
			continue
		case '`':
			if _, e, ok := codeSpan(s, j); ok {
				j = e
			} else {
				j = e // skip the unmatched run
			}
			continue
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				closeAt = j
				break scan
			}
		}
		j++
	}
	if closeAt < 0 || closeAt+1 >= len(s) || s[closeAt+1] != '(' {
		return nil, 0, false
	}
	url, end, ok := parseDest(s, closeAt+2)
	if !ok {
		return nil, 0, false
	}

	text := parseInline(s[i+1 : closeAt])
	plain := plainText(text)
	for k := range text {
		text[k].a.link = true
	}
	switch {
	case url == "":
		// Nothing to show after the text.
	case plain == "":
		text = []piece{{text: url, a: attrs{link: true}}}
	case plain == url:
		// The text already is the url.
	default:
		text = append(text,
			piece{text: " "},
			piece{text: "(" + url + ")", a: attrs{muted: true}})
	}
	return text, end, true
}

// parseDest reads a link destination and optional title starting at s[p]
// (just after the "(") through the closing ')'.
func parseDest(s string, p int) (url string, end int, ok bool) {
	skipWS := func() {
		for p < len(s) && (s[p] == ' ' || s[p] == '\n') {
			p++
		}
	}
	skipWS()
	if p < len(s) && s[p] == '<' {
		q := p + 1
		for q < len(s) && s[q] != '>' && s[q] != '<' && s[q] != '\n' {
			q++
		}
		if q >= len(s) || s[q] != '>' {
			return "", 0, false
		}
		url = s[p+1 : q]
		p = q + 1
	} else {
		start, depth := p, 0
	dest:
		for p < len(s) {
			c := s[p]
			switch {
			case c == '\\' && p+1 < len(s) && isASCIIPunct(s[p+1]):
				p += 2
				continue
			case c == ' ' || c == '\n' || c < 0x20 || p-start > maxLinkDest:
				break dest
			case c == '(':
				depth++
			case c == ')':
				if depth == 0 {
					break dest
				}
				depth--
			}
			p++
		}
		if depth != 0 || p-start > maxLinkDest {
			return "", 0, false
		}
		url = s[start:p]
	}
	url = unescape(url)

	skipWS()
	if p < len(s) && (s[p] == '"' || s[p] == '\'' || s[p] == '(') {
		closer := s[p]
		if closer == '(' {
			closer = ')'
		}
		q := p + 1
		for q < len(s) && s[q] != closer && q-p <= maxLinkTitle {
			if s[q] == '\\' {
				q++
			}
			q++
		}
		if q >= len(s) || q-p > maxLinkTitle {
			return "", 0, false
		}
		p = q + 1
		skipWS()
	}
	if p < len(s) && s[p] == ')' {
		return url, p + 1, true
	}
	return "", 0, false
}

// autolink parses <scheme:rest> at s[i].
func autolink(s string, i int) (url string, end int, ok bool) {
	j := i + 1
	if j >= len(s) || !isASCIILetter(s[j]) {
		return "", 0, false
	}
	k := j
	for k < len(s) && (isASCIILetter(s[k]) || (s[k] >= '0' && s[k] <= '9') || s[k] == '+' || s[k] == '.' || s[k] == '-') {
		k++
	}
	if n := k - j; n < 2 || n > 32 || k >= len(s) || s[k] != ':' {
		return "", 0, false
	}
	for m := k + 1; m < len(s); m++ {
		switch c := s[m]; {
		case c == '>':
			return s[j:m], m + 1, true
		case c == '<' || c == ' ' || c == '\n' || c < 0x20:
			return "", 0, false
		}
	}
	return "", 0, false
}

func plainText(ps []piece) string {
	var b strings.Builder
	for _, p := range ps {
		b.WriteString(p.text)
	}
	return b.String()
}

// unescape removes the backslash from backslash-escaped ASCII punctuation.
func unescape(s string) string {
	if !strings.Contains(s, "\\") {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) && isASCIIPunct(s[i+1]) {
			i++
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func isASCIILetter(c byte) bool { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') }

func isASCIIPunct(c byte) bool {
	return (c >= '!' && c <= '/') || (c >= ':' && c <= '@') || (c >= '[' && c <= '`') || (c >= '{' && c <= '~')
}

func isPunct(r rune) bool { return unicode.IsPunct(r) || unicode.IsSymbol(r) }
