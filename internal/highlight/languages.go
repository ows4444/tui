package highlight

import "strings"

var goKeywords = setOf("break", "case", "chan", "const", "continue", "default", "defer", "else",
	"fallthrough", "for", "func", "go", "goto", "if", "import", "interface", "map", "package",
	"range", "return", "select", "struct", "switch", "type", "var")

var goTypes = setOf("any", "bool", "byte", "comparable", "complex64", "complex128", "error",
	"float32", "float64", "int", "int8", "int16", "int32", "int64", "rune", "string",
	"uint", "uint8", "uint16", "uint32", "uint64", "uintptr")

var goLiterals = setOf("true", "false", "nil", "iota")

var shellKeywords = setOf("if", "then", "else", "elif", "fi", "for", "while", "until", "do",
	"done", "case", "esac", "in", "function", "select", "time", "return", "break", "continue")

func setOf(words ...string) map[string]bool {
	m := make(map[string]bool, len(words))
	for _, w := range words {
		m[w] = true
	}
	return m
}

func lexGo(s string) []Span {
	var e emitter
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == '/' && i+1 < len(s) && s[i+1] == '/':
			j := strings.IndexByte(s[i:], '\n')
			if j < 0 {
				j = len(s) - i
			}
			e.add(s[i:i+j], Comment)
			i += j
		case c == '/' && i+1 < len(s) && s[i+1] == '*':
			j := strings.Index(s[i+2:], "*/")
			end := len(s)
			if j >= 0 {
				end = i + 2 + j + 2
			}
			e.add(s[i:end], Comment)
			i = end
		case c == '"' || c == '\'':
			j := scanQuoted(s, i, c, false)
			e.add(s[i:j], String)
			i = j
		case c == '`':
			j := scanQuoted(s, i, c, true)
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
			case goKeywords[word]:
				e.add(word, Keyword)
			case goTypes[word]:
				e.add(word, Type)
			case goLiterals[word]:
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

func lexJSON(s string) []Span {
	var e emitter
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == '"':
			j := scanQuoted(s, i, c, false)
			k := j
			for k < len(s) && (s[k] == ' ' || s[k] == '\t') {
				k++
			}
			if k < len(s) && s[k] == ':' {
				e.add(s[i:j], Key)
			} else {
				e.add(s[i:j], String)
			}
			i = j
		case isDigit(c) || c == '-' && i+1 < len(s) && isDigit(s[i+1]):
			j := scanNumber(s, i+1)
			e.add(s[i:j], Number)
			i = j
		case isIdentStart(c):
			j := i
			for j < len(s) && isIdentPart(s[j]) {
				j++
			}
			switch word := s[i:j]; word {
			case "true", "false", "null":
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

func lexShell(s string) []Span {
	var e emitter
	wordStart := true // at the start of a word, where '#' begins a comment
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == '#' && wordStart:
			j := strings.IndexByte(s[i:], '\n')
			if j < 0 {
				j = len(s) - i
			}
			e.add(s[i:i+j], Comment)
			i += j
		case c == '"' || c == '\'':
			j := scanQuoted(s, i, c, true)
			e.add(s[i:j], String)
			i, wordStart = j, false
		case c == '$' && i+1 < len(s):
			j := i + 1
			switch n := s[j]; {
			case n == '{':
				if k := strings.IndexByte(s[j:], '}'); k >= 0 {
					j += k + 1
				} else {
					j = len(s)
				}
			case isIdentStart(n):
				for j < len(s) && isIdentPart(s[j]) {
					j++
				}
			case isDigit(n) || strings.IndexByte("?$!#@*-", n) >= 0:
				j++
			default:
				j = i + 1
			}
			e.add(s[i:j], Variable)
			i, wordStart = j, false
		case isIdentStart(c):
			j := i
			for j < len(s) && (isIdentPart(s[j]) || s[j] == '-' || s[j] == '.') {
				j++
			}
			if word := s[i:j]; shellKeywords[word] && wordStart {
				e.add(word, Keyword)
			} else {
				e.add(word, Plain)
			}
			i, wordStart = j, false
		case isDigit(c):
			j := scanNumber(s, i)
			e.add(s[i:j], Plain)
			i, wordStart = j, false
		default:
			e.add(s[i:i+1], Plain)
			wordStart = c == ' ' || c == '\t' || c == '\n' || c == ';' || c == '|' || c == '&' || c == '('
			i++
		}
	}
	return e.result()
}
