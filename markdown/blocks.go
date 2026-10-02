package markdown

import "strings"

// The block tree. Leaf text (paragraph, heading) is raw inline markdown,
// parsed by parseInline at render time.
type (
	block interface{}

	paragraph struct{ text string } // lines joined by "\n"
	heading   struct {
		level int
		text  string
	}
	codeBlock struct{ text, lang string } // lang is the fence's first info word
	table     struct {
		align  []byte // per column: 'l', 'c' or 'r'
		header []string
		rows   [][]string // each padded or cut to len(header) cells
	}
	rule  struct{}
	quote struct{ children []block }
	list  struct {
		ordered bool
		start   int  // first number of an ordered list
		delim   byte // '.' or ')' for ordered lists, the bullet character otherwise
		items   [][]block
	}
)

// maxListDepth is how many list levels are recognised: a list, and one
// list nested inside it. Deeper markers are ordinary text.
const maxListDepth = 2

// maxQuoteDepth bounds block-quote nesting, which would otherwise make
// ">>>>…" cost quadratic time and space; deeper ">" markers are text.
const maxQuoteDepth = 8

// parseBlocks turns lines into blocks. listDepth and quoteDepth are how many
// lists and block quotes enclose these lines.
func parseBlocks(lines []string, listDepth, quoteDepth int) []block {
	var blocks []block
	for i := 0; i < len(lines); {
		line := lines[i]
		switch {
		case isBlank(line):
			i++

		case fenceOK(line):
			b, next := parseFence(lines, i)
			blocks = append(blocks, b)
			i = next

		case atxOK(line):
			level, text, _ := atxHeading(line)
			blocks = append(blocks, heading{level: level, text: text})
			i++

		case isRule(line):
			blocks = append(blocks, rule{})
			i++

		case tableAt(lines, i):
			var tb table
			tb, i = parseTable(lines, i, listDepth, quoteDepth)
			blocks = append(blocks, tb)

		case quoteDepth < maxQuoteDepth && isQuoteStart(line):
			var inner []string
			for i < len(lines) && isQuoteStart(lines[i]) {
				inner = append(inner, stripQuote(lines[i]))
				i++
			}
			blocks = append(blocks, quote{children: parseBlocks(inner, listDepth, quoteDepth+1)})

		case listDepth < maxListDepth && isListStart(line):
			var l list
			start := i
			l, i = parseList(lines, i, listDepth, quoteDepth)
			if i == start {
				i++ // defensive: always make progress
			}
			blocks = append(blocks, l)

		default:
			var ls []string
			j := i
			for j < len(lines) {
				ln := lines[j]
				if isBlank(ln) {
					break
				}
				if j > i {
					if level := setextLevel(ln); level > 0 {
						last := len(ls) - 1
						ls[last] = strings.TrimRight(ls[last], " ")
						blocks = append(blocks, heading{level: level, text: strings.Join(ls, "\n")})
						j++
						ls = nil
						break
					}
					if startsBlock(ln, listDepth, quoteDepth) || tableAt(lines, j) {
						break
					}
				}
				ls = append(ls, strings.TrimLeft(ln, " "))
				j++
			}
			if ls != nil {
				ls[len(ls)-1] = strings.TrimRight(ls[len(ls)-1], " ")
				blocks = append(blocks, paragraph{text: strings.Join(ls, "\n")})
			}
			i = j
		}
	}
	return blocks
}

func fenceOK(line string) bool { _, _, _, ok := fenceOpen(line); return ok }
func atxOK(line string) bool   { _, _, ok := atxHeading(line); return ok }

// startsBlock reports whether line ends a paragraph it interrupts.
func startsBlock(line string, listDepth, quoteDepth int) bool {
	if fenceOK(line) || atxOK(line) || isRule(line) || (quoteDepth < maxQuoteDepth && isQuoteStart(line)) {
		return true
	}
	if listDepth < maxListDepth {
		// Only a non-empty bullet, or an ordered item numbered 1, may
		// interrupt a paragraph — so "2019. was a year" stays text.
		if m, ok := listMarker(line); ok && !m.empty && (!m.ordered || m.num == 1) {
			return true
		}
	}
	return false
}

func isBlank(line string) bool { return strings.TrimSpace(line) == "" }

func leadingSpaces(line string) int {
	n := 0
	for n < len(line) && line[n] == ' ' {
		n++
	}
	return n
}

// stripIndent removes up to n leading spaces.
func stripIndent(line string, n int) string {
	if k := leadingSpaces(line); k < n {
		n = k
	}
	return line[n:]
}

// --- fenced code ---

// fenceOpen recognises an opening code fence: up to three spaces of indent,
// then three or more backticks or tildes. A backtick fence's info string
// may not contain a backtick.
func fenceOpen(line string) (indent int, ch byte, n int, ok bool) {
	indent = leadingSpaces(line)
	if indent > 3 {
		return 0, 0, 0, false
	}
	rest := line[indent:]
	if len(rest) < 3 || (rest[0] != '`' && rest[0] != '~') {
		return 0, 0, 0, false
	}
	ch = rest[0]
	for n < len(rest) && rest[n] == ch {
		n++
	}
	if n < 3 || (ch == '`' && strings.Contains(rest[n:], "`")) {
		return 0, 0, 0, false
	}
	return indent, ch, n, true
}

// parseFence reads a fenced block starting at lines[i]; it runs to the
// closing fence, or to the end of the input if there is none.
func parseFence(lines []string, i int) (codeBlock, int) {
	indent, ch, n, _ := fenceOpen(lines[i])
	var lang string
	if f := strings.Fields(lines[i][indent+n:]); len(f) > 0 {
		lang = f[0]
	}
	var content []string
	i++
	for i < len(lines) {
		if isFenceClose(lines[i], ch, n) {
			i++
			break
		}
		content = append(content, stripIndent(lines[i], indent))
		i++
	}
	return codeBlock{text: strings.Join(content, "\n"), lang: lang}, i
}

func isFenceClose(line string, ch byte, n int) bool {
	ind := leadingSpaces(line)
	if ind > 3 {
		return false
	}
	rest := strings.TrimRight(line[ind:], " ")
	if len(rest) < n {
		return false
	}
	for k := 0; k < len(rest); k++ {
		if rest[k] != ch {
			return false
		}
	}
	return true
}

// --- headings and rules ---

// atxHeading recognises "# text" through "###### text"; the marker needs a
// following space (or the end of the line), and a closing run of # is
// dropped.
func atxHeading(line string) (level int, text string, ok bool) {
	ind := leadingSpaces(line)
	if ind > 3 {
		return 0, "", false
	}
	rest := line[ind:]
	for level < len(rest) && rest[level] == '#' {
		level++
	}
	if level < 1 || level > 6 {
		return 0, "", false
	}
	rest = rest[level:]
	if rest != "" && rest[0] != ' ' && rest[0] != '\t' {
		return 0, "", false
	}
	text = strings.TrimSpace(rest)
	if strings.HasSuffix(text, "#") {
		if t := strings.TrimRight(text, "#"); t == "" {
			text = ""
		} else if strings.HasSuffix(t, " ") || strings.HasSuffix(t, "\t") {
			text = strings.TrimSpace(t)
		}
	}
	return level, text, true
}

// isRule reports whether line is a thematic break: three or more of the
// same -, * or _, with only spaces between.
func isRule(line string) bool {
	ind := leadingSpaces(line)
	if ind > 3 {
		return false
	}
	var ch byte
	n := 0
	for _, c := range []byte(line[ind:]) {
		switch {
		case c == ' ' || c == '\t':
		case (c == '-' || c == '*' || c == '_') && (ch == 0 || ch == c):
			ch = c
			n++
		default:
			return false
		}
	}
	return n >= 3
}

// setextLevel returns 1 for a line of "=" and 2 for a line of "-" (up to
// three spaces of indent, trailing spaces allowed), else 0. It is only
// consulted for a line that follows paragraph text.
func setextLevel(line string) int {
	ind := leadingSpaces(line)
	if ind > 3 {
		return 0
	}
	rest := strings.TrimRight(line[ind:], " ")
	if rest == "" || strings.Trim(rest, "=") != "" && strings.Trim(rest, "-") != "" {
		return 0
	}
	if rest[0] == '=' {
		return 1
	}
	return 2
}

// --- tables ---

// splitCells cuts a table row at unescaped pipes, dropping one leading and
// one trailing pipe and trimming each cell.
func splitCells(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	var cells []string
	start := 0
	for k := 0; k < len(line); k++ {
		switch line[k] {
		case '\\':
			k++
		case '|':
			cells = append(cells, strings.TrimSpace(line[start:k]))
			start = k + 1
		}
	}
	last := strings.TrimSpace(line[start:])
	if last != "" || len(cells) == 0 {
		cells = append(cells, last)
	}
	return cells
}

// delimiterAligns parses a table delimiter row ("|:--|:-:|--:|") into
// per-column alignments; ok is false unless every cell is dashes with an
// optional colon at either end and the row has a pipe.
func delimiterAligns(line string) ([]byte, bool) {
	if ind := leadingSpaces(line); ind > 3 || !strings.Contains(line, "|") {
		return nil, false
	}
	cells := splitCells(line)
	aligns := make([]byte, len(cells))
	for k, c := range cells {
		left, right := strings.HasPrefix(c, ":"), strings.HasSuffix(c, ":")
		core := strings.TrimSuffix(strings.TrimPrefix(c, ":"), ":")
		if core == "" || strings.Trim(core, "-") != "" {
			return nil, false
		}
		switch {
		case left && right:
			aligns[k] = 'c'
		case right:
			aligns[k] = 'r'
		default:
			aligns[k] = 'l'
		}
	}
	return aligns, true
}

// tableAt reports whether a table starts at lines[i]: a header row with a
// pipe, then a delimiter row with the same number of cells.
func tableAt(lines []string, i int) bool {
	if i+1 >= len(lines) || leadingSpaces(lines[i]) > 3 || !strings.Contains(lines[i], "|") {
		return false
	}
	aligns, ok := delimiterAligns(lines[i+1])
	return ok && len(aligns) == len(splitCells(lines[i]))
}

// parseTable reads the table at lines[i]; its rows run until a blank line
// or the start of another block.
func parseTable(lines []string, i, listDepth, quoteDepth int) (table, int) {
	tb := table{header: splitCells(lines[i])}
	tb.align, _ = delimiterAligns(lines[i+1])
	n := len(tb.header)
	for i += 2; i < len(lines); i++ {
		ln := lines[i]
		if isBlank(ln) || startsBlock(ln, listDepth, quoteDepth) {
			break
		}
		cells := splitCells(ln)
		for len(cells) < n {
			cells = append(cells, "")
		}
		tb.rows = append(tb.rows, cells[:n])
	}
	return tb, i
}

// --- block quotes ---

func isQuoteStart(line string) bool {
	ind := leadingSpaces(line)
	return ind <= 3 && ind < len(line) && line[ind] == '>'
}

// stripQuote removes the ">" marker and one optional following space.
func stripQuote(line string) string {
	s := line[leadingSpaces(line)+1:]
	return strings.TrimPrefix(s, " ")
}

// --- lists ---

type marker struct {
	ordered bool
	ch      byte // bullet character, or '.' / ')' for ordered
	num     int
	offset  int  // column where the item's content starts
	empty   bool // nothing after the marker
}

// listMarker parses a list item marker: up to three spaces of indent, a
// bullet (-, *, +) or one to nine digits and . or ), then a space or the
// end of the line.
func listMarker(line string) (marker, bool) {
	ind := leadingSpaces(line)
	if ind > 3 || ind >= len(line) {
		return marker{}, false
	}
	rest := line[ind:]
	var m marker
	width := 0
	switch c := rest[0]; {
	case c == '-' || c == '*' || c == '+':
		m.ch, width = c, 1
	case c >= '0' && c <= '9':
		d := 0
		for d < len(rest) && d < 10 && rest[d] >= '0' && rest[d] <= '9' {
			m.num = m.num*10 + int(rest[d]-'0')
			d++
		}
		if d > 9 || d >= len(rest) || (rest[d] != '.' && rest[d] != ')') {
			return marker{}, false
		}
		m.ordered, m.ch, width = true, rest[d], d+1
	default:
		return marker{}, false
	}
	after := rest[width:]
	if after != "" && after[0] != ' ' && after[0] != '\t' {
		return marker{}, false
	}
	sp := leadingSpaces(after)
	switch {
	case strings.TrimSpace(after) == "":
		m.empty = true
		m.offset = ind + width + 1
	case sp >= 5:
		m.offset = ind + width + 1
	default:
		m.offset = ind + width + max(sp, 1)
	}
	return m, true
}

func isListStart(line string) bool { _, ok := listMarker(line); return ok }

func sameKind(a, b marker) bool { return a.ordered == b.ordered && a.ch == b.ch }

// itemFirstLine is the text after the marker on an item's first line.
func itemFirstLine(line string, m marker) string {
	if m.empty {
		return ""
	}
	if len(line) <= m.offset {
		return ""
	}
	return line[m.offset:]
}

// parseList reads a list starting at lines[i]. Items continue with lines
// indented to their content column, a blank line followed by such a line,
// or a "lazy" unindented line continuing an open paragraph; a list marker
// of the same kind starts the next item.
func parseList(lines []string, i, listDepth, quoteDepth int) (list, int) {
	first, _ := listMarker(lines[i])
	l := list{ordered: first.ordered, start: first.num, delim: first.ch}

	for i < len(lines) {
		m, ok := listMarker(lines[i])
		if !ok || !sameKind(m, first) || isRule(lines[i]) {
			break
		}
		item := []string{itemFirstLine(lines[i], m)}
		i++

	body:
		for i < len(lines) {
			ln := lines[i]
			if isBlank(ln) {
				k := i
				for k < len(lines) && isBlank(lines[k]) {
					k++
				}
				if k < len(lines) && leadingSpaces(lines[k]) >= m.offset {
					for ; i < k; i++ {
						item = append(item, "")
					}
					continue
				}
				break body
			}
			if leadingSpaces(ln) >= m.offset {
				item = append(item, stripIndent(ln, m.offset))
				i++
				continue
			}
			if isListStart(ln) || startsBlock(ln, maxListDepth, quoteDepth) {
				break body
			}
			if strings.TrimSpace(item[len(item)-1]) == "" {
				break body // no open paragraph to continue lazily
			}
			item = append(item, strings.TrimLeft(ln, " "))
			i++
		}
		l.items = append(l.items, parseBlocks(item, listDepth+1, quoteDepth))

		// Blank lines between items don't end the list; anything else does.
		j := i
		for j < len(lines) && isBlank(lines[j]) {
			j++
		}
		if j < len(lines) {
			if m2, ok := listMarker(lines[j]); ok && sameKind(m2, first) && !isRule(lines[j]) {
				i = j
				continue
			}
		}
		break
	}
	return l, i
}
