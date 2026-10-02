package highlight

import "strings"

var pyKeywords = setOf("and", "as", "assert", "async", "await", "break", "class", "continue",
	"def", "del", "elif", "else", "except", "finally", "for", "from", "global", "if", "import",
	"in", "is", "lambda", "match", "case", "nonlocal", "not", "or", "pass", "raise", "return",
	"try", "while", "with", "yield")

var pyTypes = setOf("bool", "bytes", "bytearray", "complex", "dict", "float", "frozenset", "int",
	"list", "object", "set", "str", "tuple", "type")

var pyLiterals = setOf("True", "False", "None")

// pyStringPrefix reports whether word is a string prefix such as r, b, f or rb.
func pyStringPrefix(word string) bool {
	if len(word) > 2 {
		return false
	}
	for _, c := range strings.ToLower(word) {
		if c != 'r' && c != 'b' && c != 'u' && c != 'f' {
			return false
		}
	}
	return true
}

// scanPyString returns the end of the string literal whose opening quote is at
// s[i]. A triple-quoted string runs to its closing triple quote or the end of
// input; any other string ends at its closing quote or at the end of the line.
func scanPyString(s string, i int) int {
	q := s[i]
	if strings.HasPrefix(s[i:], string([]byte{q, q, q})) {
		triple := s[i : i+3]
		for j := i + 3; j < len(s); j++ {
			switch {
			case s[j] == '\\':
				j++
			case strings.HasPrefix(s[j:], triple):
				return j + 3
			}
		}
		return len(s)
	}
	return scanQuoted(s, i, q, false)
}

func lexPython(s string) []Span {
	var e emitter
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == '#':
			j := strings.IndexByte(s[i:], '\n')
			if j < 0 {
				j = len(s) - i
			}
			e.add(s[i:i+j], Comment)
			i += j
		case c == '"' || c == '\'':
			j := scanPyString(s, i)
			e.add(s[i:j], String)
			i = j
		case isDigit(c) || c == '.' && i+1 < len(s) && isDigit(s[i+1]):
			j := scanNumber(s, i)
			e.add(s[i:j], Number)
			i = j
		case isIdentStart(c):
			j := i
			for j < len(s) && isIdentPart(s[j]) {
				j++
			}
			word := s[i:j]
			switch {
			case j < len(s) && (s[j] == '"' || s[j] == '\'') && pyStringPrefix(word):
				k := scanPyString(s, j)
				e.add(s[i:k], String)
				j = k
			case pyKeywords[word]:
				e.add(word, Keyword)
			case pyTypes[word]:
				e.add(word, Type)
			case pyLiterals[word]:
				e.add(word, Literal)
			default:
				e.add(word, Plain)
			}
			i = j
		default:
			e.add(s[i:i+1], Plain)
			i++
		}
	}
	return e.result()
}
