package markdown

import (
	"html"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/ows4444/tui/ansi"
)

// resolveRefs turns reference links into inline ones before the blocks are
// parsed: "[text][label]", "[text][]" and "[label]" become "[text](url)" when
// a definition "[label]: url" exists, and the definition lines are removed.
// Fenced code is left alone, and so are code spans and escaped brackets. A
// reference with no definition stays text. Titles are read and dropped.
func resolveRefs(lines []string) []string {
	defs := map[string]string{}
	keep := make([]string, 0, len(lines))
	var fenceCh byte
	var fenceN int
	fenced := make([]bool, 0, len(lines))
	for _, ln := range lines {
		if fenceCh != 0 {
			fenced = append(fenced, true)
			keep = append(keep, ln)
			if isFenceClose(ln, fenceCh, fenceN) {
				fenceCh = 0
			}
			continue
		}
		if _, ch, n, ok := fenceOpen(ln); ok {
			fenceCh, fenceN = ch, n
			fenced = append(fenced, true)
			keep = append(keep, ln)
			continue
		}
		if label, url, ok := refDefinition(ln); ok {
			if _, dup := defs[label]; !dup { // the first definition wins
				defs[label] = url
			}
			continue
		}
		fenced = append(fenced, false)
		keep = append(keep, ln)
	}
	if len(defs) == 0 {
		return keep
	}
	for i, ln := range keep {
		if !fenced[i] && strings.Contains(ln, "[") {
			keep[i] = rewriteRefs(ln, defs)
		}
	}
	return keep
}

// normLabel folds a reference label: case and runs of white space do not
// matter.
func normLabel(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}

// refDefinition reads "[label]: url" with an optional title, indented up to
// three spaces and with nothing after.
func refDefinition(line string) (label, url string, ok bool) {
	ind := leadingSpaces(line)
	if ind > 3 || ind >= len(line) || line[ind] != '[' {
		return "", "", false
	}
	rest := line[ind+1:]
	end := -1
	for i := 0; i < len(rest); i++ {
		switch rest[i] {
		case '\\':
			i++
		case '[':
			return "", "", false
		case ']':
			end = i
		}
		if end >= 0 {
			break
		}
	}
	if end <= 0 || end+1 >= len(rest) || rest[end+1] != ':' {
		return "", "", false
	}
	label = normLabel(rest[:end])
	if label == "" || label[0] == '^' { // a footnote definition is not a link
		return "", "", false
	}
	rest = strings.TrimLeft(rest[end+2:], " \t")
	switch {
	case rest == "":
		return "", "", false
	case rest[0] == '<':
		j := strings.IndexByte(rest, '>')
		if j < 0 {
			return "", "", false
		}
		url, rest = rest[1:j], rest[j+1:]
	default:
		j := strings.IndexAny(rest, " \t")
		if j < 0 {
			j = len(rest)
		}
		url, rest = rest[:j], rest[j:]
	}
	if url == "" || strings.ContainsAny(url, "<>") {
		return "", "", false
	}
	rest = strings.TrimSpace(rest)
	if rest != "" && !validTitle(rest) {
		return "", "", false
	}
	return label, url, true
}

// validTitle reports whether s is one quoted or parenthesised title.
func validTitle(s string) bool {
	if len(s) < 2 {
		return false
	}
	closer := map[byte]byte{'"': '"', '\'': '\'', '(': ')'}[s[0]]
	return closer != 0 && s[len(s)-1] == closer
}

// rewriteRefs replaces the reference links in one line of text.
func rewriteRefs(s string, defs map[string]string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		switch c := s[i]; c {
		case '\\':
			n := min(2, len(s)-i)
			b.WriteString(s[i : i+n])
			i += n
		case '`':
			j := i
			for j < len(s) && s[j] == '`' {
				j++
			}
			run := s[i:j]
			if k := strings.Index(s[j:], run); k >= 0 {
				b.WriteString(s[i : j+k+len(run)])
				i = j + k + len(run)
			} else {
				b.WriteString(run)
				i = j
			}
		case '[':
			if out, end, ok := refAt(s, i, defs); ok {
				b.WriteString(out)
				i = end
				continue
			}
			b.WriteByte(c)
			i++
		default:
			b.WriteByte(c)
			i++
		}
	}
	return b.String()
}

// refAt reads a reference link starting at s[i] == '['. It returns the inline
// link to write instead and where the reference ends.
func refAt(s string, i int, defs map[string]string) (string, int, bool) {
	depth, j := 0, i
	for ; j < len(s); j++ {
		if s[j] == '\\' {
			j++
			continue
		}
		if s[j] == '[' {
			depth++
		} else if s[j] == ']' {
			depth--
			if depth == 0 {
				break
			}
		}
	}
	if j >= len(s) {
		return "", 0, false
	}
	text, end := s[i+1:j], j+1
	if end < len(s) && s[end] == '(' {
		return "", 0, false // an inline link
	}
	label := text
	if end < len(s) && s[end] == '[' {
		if k := strings.IndexByte(s[end+1:], ']'); k >= 0 && !strings.ContainsAny(s[end+1:end+1+k], "[") {
			if l := s[end+1 : end+1+k]; l != "" {
				label = l
			}
			end += k + 2
		}
	} else if end < len(s) && s[end] == ':' {
		return "", 0, false
	}
	url, ok := defs[normLabel(label)]
	if !ok || strings.TrimSpace(text) == "" {
		return "", 0, false
	}
	if strings.ContainsAny(url, " ()") {
		url = "<" + url + ">"
	}
	return "[" + text + "](" + url + ")", end, true
}

// entity decodes the character reference at s[i] == '&', such as "&amp;" or
// "&#8212;", into plain text. It reports false for a reference that is unknown
// or that would decode to a control or format character, which stays as typed.
func entity(s string, i int) (text string, end int, ok bool) {
	j := i + 1
	if j < len(s) && s[j] == '#' {
		j++
		if j < len(s) && (s[j] == 'x' || s[j] == 'X') {
			j++
		}
	}
	start := j
	for j < len(s) && j-i <= 32 && (isASCIILetter(s[j]) || (s[j] >= '0' && s[j] <= '9')) {
		j++
	}
	if j == start || j >= len(s) || s[j] != ';' {
		return "", 0, false
	}
	raw := s[i : j+1]
	dec := html.UnescapeString(raw)
	if dec == raw || !utf8.ValidString(dec) || ansi.Sanitize(dec) != dec {
		return "", 0, false
	}
	for _, r := range dec {
		if unicode.IsControl(r) || r == utf8.RuneError {
			return "", 0, false
		}
	}
	return dec, j + 1, true
}
