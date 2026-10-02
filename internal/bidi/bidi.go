// Package bidi implements the Unicode Bidirectional Algorithm (UAX #9) for one
// line of text: it resolves the embedding level of every character and returns
// the visual order, for display only. The Bidi_Class, bracket and mirroring
// tables are generated from the Unicode Character Database by
// internal/tools/genbidi. The package uses only the standard library.
//
// It treats the whole input as a single paragraph: a line shown in a terminal is
// one row, never wrapped. It does not touch cursor logic or text editing, which
// stay in logical order.
package bidi

import "sort"

// Class is a Bidi_Class value.
type Class uint8

// The Bidi_Class values, in the order of UAX #9 table 4.
const (
	L   Class = iota // Left-to-Right
	R                // Right-to-Left
	AL               // Right-to-Left Arabic
	EN               // European Number
	ES               // European Number Separator
	ET               // European Number Terminator
	AN               // Arabic Number
	CS               // Common Number Separator
	NSM              // Nonspacing Mark
	BN               // Boundary Neutral
	B                // Paragraph Separator
	S                // Segment Separator
	WS               // Whitespace
	ON               // Other Neutrals
	LRE              // Left-to-Right Embedding
	LRO              // Left-to-Right Override
	RLE              // Right-to-Left Embedding
	RLO              // Right-to-Left Override
	PDF              // Pop Directional Format
	LRI              // Left-to-Right Isolate
	RLI              // Right-to-Left Isolate
	FSI              // First Strong Isolate
	PDI              // Pop Directional Isolate
)

type classRange struct {
	lo, hi rune
	class  Class
}

type bracketEntry struct {
	r, pair rune
	kind    byte // 'o' or 'c'
}

type mirrorEntry struct{ r, glyph rune }

// ClassOf returns the Bidi_Class of r.
func ClassOf(r rune) Class {
	i := sort.Search(len(classRanges), func(i int) bool { return classRanges[i].hi >= r })
	if i < len(classRanges) && classRanges[i].lo <= r {
		return classRanges[i].class
	}
	return L
}

// Mirror returns the glyph that replaces r in a right-to-left run (UAX #9 rule
// L4), and false when r has no exact mirror image.
func Mirror(r rune) (rune, bool) {
	i := sort.Search(len(mirrorTable), func(i int) bool { return mirrorTable[i].r >= r })
	if i < len(mirrorTable) && mirrorTable[i].r == r {
		return mirrorTable[i].glyph, true
	}
	return r, false
}

func bracketOf(r rune) (bracketEntry, bool) {
	i := sort.Search(len(bracketTable), func(i int) bool { return bracketTable[i].r >= r })
	if i < len(bracketTable) && bracketTable[i].r == r {
		return bracketTable[i], true
	}
	return bracketEntry{}, false
}

// canonicalBracket maps the two angle brackets with canonical decompositions to
// their equivalents, so U+2329 pairs with U+3009 as BD16 requires.
func canonicalBracket(r rune) rune {
	switch r {
	case 0x2329:
		return 0x3008
	case 0x232A:
		return 0x3009
	}
	return r
}

// Direction is the base direction of a paragraph.
type Direction uint8

const (
	// LeftToRight forces paragraph level 0.
	LeftToRight Direction = iota
	// RightToLeft forces paragraph level 1.
	RightToLeft
	// Auto takes the direction of the first strong character (rules P2 and
	// P3), or left to right when there is none.
	Auto
)

// maxDepth is the deepest explicit embedding level (BD2).
const maxDepth = 125

// Result is the outcome of Resolve.
type Result struct {
	// Levels is the resolved embedding level of each rune after rule L1. A
	// character removed by rule X9 (an embedding or override control, or BN)
	// takes the level of the character before it, or the paragraph level.
	Levels []uint8
	// Order lists logical indices in visual order, left to right (rule L2).
	Order []int
	// ParagraphLevel is the resolved paragraph embedding level, 0 or 1.
	ParagraphLevel uint8
}

// Removed reports whether rule X9 removes a character of class c from the
// text the later rules see.
func Removed(c Class) bool {
	switch c {
	case LRE, RLE, LRO, RLO, PDF, BN:
		return true
	}
	return false
}

func isIsolateInitiator(c Class) bool { return c == LRI || c == RLI || c == FSI }

// Resolve runs the algorithm over runes as one paragraph with base direction dir.
func Resolve(runes []rune, dir Direction) Result {
	n := len(runes)
	p := &paragraph{runes: runes, orig: make([]Class, n)}
	for i, r := range runes {
		p.orig[i] = ClassOf(r)
	}
	p.matchIsolates()
	switch dir {
	case LeftToRight:
		p.base = 0
	case RightToLeft:
		p.base = 1
	default:
		p.base = p.firstStrongLevel(0, n)
	}
	p.explicit()
	for _, seq := range p.sequences() {
		p.resolveSequence(seq)
	}
	p.reset()
	return Result{Levels: p.levels, Order: p.reorder(), ParagraphLevel: p.base}
}

type paragraph struct {
	runes []rune
	orig  []Class // original classes
	types []Class // classes after explicit overrides and the weak/neutral rules
	base  uint8

	levels   []uint8
	matchPDI []int // for an isolate initiator, the index of its PDI, or n when none
	matchIni []int // for a PDI, the index of its initiator, or -1
}

// matchIsolates pairs isolate initiators with PDIs (BD9).
func (p *paragraph) matchIsolates() {
	n := len(p.runes)
	p.matchPDI = make([]int, n)
	p.matchIni = make([]int, n)
	for i := range p.matchPDI {
		p.matchPDI[i], p.matchIni[i] = n, -1
	}
	var stack []int
	for i, c := range p.orig {
		switch {
		case isIsolateInitiator(c):
			stack = append(stack, i)
		case c == PDI && len(stack) > 0:
			j := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			p.matchPDI[j], p.matchIni[i] = i, j
		}
	}
}

// firstStrongLevel applies rules P2 and P3 to runes[from:to]: the level of the
// first L, R or AL, skipping what lies between an isolate initiator and its
// matching PDI; 0 when there is none.
func (p *paragraph) firstStrongLevel(from, to int) uint8 {
	for i := from; i < to; i++ {
		switch c := p.orig[i]; {
		case c == L:
			return 0
		case c == R || c == AL:
			return 1
		case isIsolateInitiator(c):
			if p.matchPDI[i] >= to {
				return 0 // no matching PDI in range: the rest is isolated
			}
			i = p.matchPDI[i]
		}
	}
	return 0
}

type stackEntry struct {
	level    uint8
	override Class // L, R, or ON for none
	isolate  bool
}

// explicit applies rules X1 to X8, setting levels and the overridden types.
func (p *paragraph) explicit() {
	n := len(p.runes)
	p.levels = make([]uint8, n)
	p.types = append([]Class(nil), p.orig...)
	stack := []stackEntry{{p.base, ON, false}}
	overflowIsolate, overflowEmbedding, validIsolate := 0, 0, 0
	for i, c := range p.orig {
		top := stack[len(stack)-1]
		switch c {
		case RLE, LRE, RLO, LRO, RLI, LRI, FSI:
			isolate := isIsolateInitiator(c)
			rtl := c == RLE || c == RLO || c == RLI
			if c == FSI {
				rtl = p.firstStrongLevel(i+1, p.matchPDI[i]) == 1
			}
			p.levels[i] = top.level
			if isolate && top.override != ON {
				p.types[i] = top.override
			}
			next := (top.level + 2) &^ 1 // least greater even level
			if rtl {
				next = (top.level + 1) | 1 // least greater odd level
			}
			if next <= maxDepth && overflowIsolate == 0 && overflowEmbedding == 0 {
				if isolate {
					validIsolate++
				}
				ov := ON
				switch c {
				case LRO:
					ov = L
				case RLO:
					ov = R
				}
				stack = append(stack, stackEntry{next, ov, isolate})
			} else if isolate {
				overflowIsolate++
			} else if overflowIsolate == 0 {
				overflowEmbedding++
			}
		case PDI:
			if overflowIsolate > 0 {
				overflowIsolate--
			} else if validIsolate > 0 {
				overflowEmbedding = 0
				for !stack[len(stack)-1].isolate {
					stack = stack[:len(stack)-1]
				}
				stack = stack[:len(stack)-1]
				validIsolate--
			}
			top = stack[len(stack)-1]
			p.levels[i] = top.level
			if top.override != ON {
				p.types[i] = top.override
			}
		case PDF:
			p.levels[i] = top.level
			if overflowIsolate > 0 {
			} else if overflowEmbedding > 0 {
				overflowEmbedding--
			} else if !top.isolate && len(stack) >= 2 {
				stack = stack[:len(stack)-1]
			}
		case B:
			p.levels[i] = p.base
		case BN:
			p.levels[i] = top.level
		default:
			p.levels[i] = top.level
			if top.override != ON {
				p.types[i] = top.override
			}
		}
	}
}

// sequence is an isolating run sequence (BD13): the indices of its characters in
// logical order, with its start and end of sequence types.
type sequence struct {
	idx      []int
	sos, eos Class
}

// sequences builds the isolating run sequences (rule X10).
func (p *paragraph) sequences() []sequence {
	n := len(p.runes)
	// Level runs over the characters X9 keeps.
	var runs [][]int
	runOf := make([]int, n)
	for i := range runOf {
		runOf[i] = -1
	}
	for i := 0; i < n; i++ {
		if Removed(p.orig[i]) {
			continue
		}
		last := len(runs) - 1
		if last >= 0 && p.levels[runs[last][len(runs[last])-1]] == p.levels[i] && p.prevKept(i) == runs[last][len(runs[last])-1] {
			runs[last] = append(runs[last], i)
		} else {
			runs = append(runs, []int{i})
		}
		runOf[i] = len(runs) - 1
	}
	var out []sequence
	for ri, run := range runs {
		first := run[0]
		if p.orig[first] == PDI && p.matchIni[first] >= 0 {
			continue // continues the sequence of its initiator
		}
		idx := append([]int(nil), run...)
		cur := ri
		for {
			last := runs[cur][len(runs[cur])-1]
			if !isIsolateInitiator(p.orig[last]) || p.matchPDI[last] >= n {
				break
			}
			cur = runOf[p.matchPDI[last]]
			if cur < 0 {
				break
			}
			idx = append(idx, runs[cur]...)
		}
		out = append(out, p.bounds(idx))
	}
	return out
}

// prevKept returns the index of the kept character before i, or -1.
func (p *paragraph) prevKept(i int) int {
	for j := i - 1; j >= 0; j-- {
		if !Removed(p.orig[j]) {
			return j
		}
	}
	return -1
}

func (p *paragraph) nextKept(i int) int {
	for j := i + 1; j < len(p.runes); j++ {
		if !Removed(p.orig[j]) {
			return j
		}
	}
	return -1
}

func dirOf(level uint8) Class {
	if level&1 == 1 {
		return R
	}
	return L
}

// bounds sets the start and end of sequence types of the sequence idx.
func (p *paragraph) bounds(idx []int) sequence {
	first, last := idx[0], idx[len(idx)-1]
	level := p.levels[first]
	prev := p.base
	if j := p.prevKept(first); j >= 0 {
		prev = p.levels[j]
	}
	next := p.base
	if !(isIsolateInitiator(p.orig[last]) && p.matchPDI[last] >= len(p.runes)) {
		if j := p.nextKept(last); j >= 0 {
			next = p.levels[j]
		}
	}
	return sequence{idx: idx, sos: dirOf(max(level, prev)), eos: dirOf(max(level, next))}
}

func isNeutralOrIsolate(c Class) bool {
	switch c {
	case B, S, WS, ON, FSI, LRI, RLI, PDI:
		return true
	}
	return false
}

// resolveSequence applies rules W1 to W7, N0 to N2 and I1 to I2 to seq.
func (p *paragraph) resolveSequence(seq sequence) {
	idx := seq.idx
	t := make([]Class, len(idx))
	for k, i := range idx {
		t[k] = p.types[i]
	}
	level := p.levels[idx[0]]

	// W1: a nonspacing mark takes the class of the previous character.
	for k := range t {
		if t[k] != NSM {
			continue
		}
		switch {
		case k == 0:
			t[k] = seq.sos
		case isIsolateInitiator(t[k-1]) || t[k-1] == PDI:
			t[k] = ON
		default:
			t[k] = t[k-1]
		}
	}
	// W2: European numbers after Arabic letters become Arabic numbers.
	for k := range t {
		if t[k] != EN {
			continue
		}
		for j := k - 1; j >= 0; j-- {
			if t[j] == L || t[j] == R || t[j] == AL {
				if t[j] == AL {
					t[k] = AN
				}
				break
			}
		}
	}
	// W3: Arabic letters become R.
	for k := range t {
		if t[k] == AL {
			t[k] = R
		}
	}
	// W4: a single separator between two numbers of one type joins them.
	for k := 1; k+1 < len(t); k++ {
		switch {
		case t[k] == ES && t[k-1] == EN && t[k+1] == EN:
			t[k] = EN
		case t[k] == CS && t[k-1] == EN && t[k+1] == EN:
			t[k] = EN
		case t[k] == CS && t[k-1] == AN && t[k+1] == AN:
			t[k] = AN
		}
	}
	// W5: a run of terminators next to a European number becomes European numbers.
	for k := 0; k < len(t); k++ {
		if t[k] != ET {
			continue
		}
		end := k
		for end < len(t) && t[end] == ET {
			end++
		}
		if (k > 0 && t[k-1] == EN) || (end < len(t) && t[end] == EN) {
			for j := k; j < end; j++ {
				t[j] = EN
			}
		}
		k = end - 1
	}
	// W6: remaining separators and terminators become neutral.
	for k := range t {
		if t[k] == ES || t[k] == ET || t[k] == CS {
			t[k] = ON
		}
	}
	// W7: European numbers after an L become L.
	for k := range t {
		if t[k] != EN {
			continue
		}
		strong := seq.sos
		for j := k - 1; j >= 0; j-- {
			if t[j] == L || t[j] == R {
				strong = t[j]
				break
			}
		}
		if strong == L {
			t[k] = L
		}
	}

	p.resolveBrackets(seq, t, level)

	// N1 and N2: runs of neutrals take the direction both sides share, else the
	// embedding direction.
	strong := func(c Class) Class { // L, R, or ON for neither; numbers count as R
		switch c {
		case L:
			return L
		case R, EN, AN:
			return R
		}
		return ON
	}
	for k := 0; k < len(t); k++ {
		if !isNeutralOrIsolate(t[k]) {
			continue
		}
		end := k
		for end < len(t) && isNeutralOrIsolate(t[end]) {
			end++
		}
		before, after := seq.sos, seq.eos
		if k > 0 {
			before = strong(t[k-1])
		}
		if end < len(t) {
			after = strong(t[end])
		}
		resolved := dirOf(level)
		if before == after && before != ON {
			resolved = before
		}
		for j := k; j < end; j++ {
			t[j] = resolved
		}
		k = end - 1
	}

	// I1 and I2: raise the levels of characters that disagree with their level.
	for k, i := range idx {
		if level&1 == 0 {
			switch t[k] {
			case R:
				p.levels[i] = level + 1
			case AN, EN:
				p.levels[i] = level + 2
			}
		} else if t[k] == L || t[k] == EN || t[k] == AN {
			p.levels[i] = level + 1
		}
	}
}

type bracketPair struct{ open, close int } // indices into the sequence

// resolveBrackets applies rule N0 to the sequence classes t.
func (p *paragraph) resolveBrackets(seq sequence, t []Class, level uint8) {
	idx := seq.idx
	const maxStack = 63
	type open struct {
		pair rune
		pos  int
	}
	var stack []open
	var pairs []bracketPair
scan:
	for k, i := range idx {
		if t[k] != ON {
			continue
		}
		b, ok := bracketOf(p.runes[i])
		if !ok {
			continue
		}
		switch b.kind {
		case 'o':
			if len(stack) == maxStack {
				break scan
			}
			stack = append(stack, open{canonicalBracket(b.pair), k})
		case 'c':
			me := canonicalBracket(p.runes[i])
			for s := len(stack) - 1; s >= 0; s-- {
				if stack[s].pair == me {
					pairs = append(pairs, bracketPair{stack[s].pos, k})
					stack = stack[:s]
					break
				}
			}
		}
	}
	sort.Slice(pairs, func(a, b int) bool { return pairs[a].open < pairs[b].open })

	embedding := dirOf(level)
	strong := func(c Class) Class {
		switch c {
		case L:
			return L
		case R, EN, AN:
			return R
		}
		return ON
	}
	for _, pr := range pairs {
		foundE, foundO := false, false
		for k := pr.open + 1; k < pr.close; k++ {
			switch s := strong(t[k]); {
			case s == embedding:
				foundE = true
			case s != ON:
				foundO = true
			}
			if foundE {
				break
			}
		}
		var set Class
		switch {
		case foundE:
			set = embedding
		case foundO:
			ctx := seq.sos
			for k := pr.open - 1; k >= 0; k-- {
				if s := strong(t[k]); s != ON {
					ctx = s
					break
				}
			}
			if ctx != embedding {
				set = ctx // the opposite direction
			} else {
				set = embedding
			}
		default:
			continue
		}
		t[pr.open], t[pr.close] = set, set
		// Nonspacing marks that followed a bracket take its new class.
		for _, k := range []int{pr.open, pr.close} {
			for j := k + 1; j < len(t) && p.orig[idx[j]] == NSM; j++ {
				t[j] = set
			}
		}
	}
}

// reset applies rule L1 (segment and paragraph separators, and the whitespace
// and isolate formatting characters before them and at the end of the line, go
// back to the paragraph level) and gives each character X9 removed the level of
// the character before it. Removed characters are transparent to L1.
func (p *paragraph) reset() {
	n := len(p.runes)
	trailing := true // everything after i, ignoring removed characters, is whitespace-like
	for i := n - 1; i >= 0; i-- {
		switch c := p.orig[i]; {
		case Removed(c):
		case c == S || c == B:
			p.levels[i] = p.base
			trailing = true
		case c == WS || isIsolateInitiator(c) || c == PDI:
			if trailing {
				p.levels[i] = p.base
			}
		default:
			trailing = false
		}
	}
	prev := p.base
	for i := 0; i < n; i++ {
		if Removed(p.orig[i]) {
			p.levels[i] = prev
		} else {
			prev = p.levels[i]
		}
	}
}

// reorder applies rule L2: from the highest level down to the lowest odd one,
// reverse every run of characters at that level or higher.
func (p *paragraph) reorder() []int {
	n := len(p.runes)
	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	var hi uint8
	lowOdd := uint8(255)
	for _, l := range p.levels {
		hi = max(hi, l)
		if l&1 == 1 {
			lowOdd = min(lowOdd, l)
		}
	}
	for lvl := hi; lvl >= lowOdd && lvl > 0; lvl-- {
		for i := 0; i < n; {
			if p.levels[order[i]] < lvl {
				i++
				continue
			}
			j := i
			for j < n && p.levels[order[j]] >= lvl {
				j++
			}
			for a, b := i, j-1; a < b; a, b = a+1, b-1 {
				order[a], order[b] = order[b], order[a]
			}
			i = j
		}
	}
	return order
}
