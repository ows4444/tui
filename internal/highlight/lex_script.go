package highlight

import "strings"

var (
	rubyKeywords = setOf("BEGIN", "END", "alias", "and", "begin", "break", "case", "class", "def",
		"defined?", "do", "else", "elsif", "end", "ensure", "for", "if", "in", "module", "next",
		"not", "or", "redo", "rescue", "retry", "return", "super", "then", "undef", "unless",
		"until", "when", "while", "yield", "require", "require_relative", "include", "extend",
		"attr_reader", "attr_writer", "attr_accessor", "private", "protected", "public", "raise",
		"lambda", "proc")
	rubyLiterals = setOf("true", "false", "nil", "self", "__FILE__", "__LINE__", "__method__")

	luaKeywords = setOf("and", "break", "do", "else", "elseif", "end", "for", "function", "goto",
		"if", "in", "local", "not", "or", "repeat", "return", "then", "until", "while")
	luaLiterals = setOf("nil", "true", "false", "self")
)

// lexRuby colours # and =begin/=end comments, '...' and "..." strings (which
// may span lines), :symbols as literals, key: hash keys as keys, @ivar,
// @@cvar and $global as variables, Constants as types, keywords and numbers.
func lexRuby(s string) []Span {
	var e emitter
	lineStart := true
	var prev byte // last byte emitted outside whitespace, 0 at the start
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case lineStart && strings.HasPrefix(s[i:], "=begin") && (i+6 == len(s) || s[i+6] == ' ' || s[i+6] == '\n' || s[i+6] == '\t'):
			j := len(s)
			if k := strings.Index(s[i:], "\n=end"); k >= 0 {
				j = lineEnd(s, i+k+1)
			}
			e.add(s[i:j], Comment)
			i, prev, lineStart = j, 'a', false
		case c == '\n':
			e.add("\n", Plain)
			i++
			lineStart = true
		case c == ' ' || c == '\t' || c == '\r':
			e.add(s[i:i+1], Plain)
			i++
		case c == '#':
			j := lineEnd(s, i)
			e.add(s[i:j], Comment)
			i, lineStart = j, false
		case c == '"' || c == '\'' || c == '`':
			j := scanQuoted(s, i, c, true)
			e.add(s[i:j], String)
			i, prev, lineStart = j, c, false
		case c == ':' && i+1 < len(s) && (isIdentStart(s[i+1]) || s[i+1] == '"') && (i == 0 || !isIdentPart(s[i-1]) && s[i-1] != ':'):
			j := i + 1
			if s[j] == '"' {
				j = scanQuoted(s, j, '"', false)
			} else {
				j = rubyIdentEnd(s, j)
			}
			e.add(s[i:j], Literal)
			i, prev, lineStart = j, 'a', false
		case (c == '@' || c == '$') && i+1 < len(s) && (isIdentStart(s[i+1]) || s[i+1] == '@' && c == '@'):
			j := i + 1
			if s[j] == '@' {
				j++
			}
			for j < len(s) && isIdentPart(s[j]) {
				j++
			}
			e.add(s[i:j], Variable)
			i, prev, lineStart = j, 'a', false
		case isDigit(c):
			j := scanNumber(s, i)
			e.add(s[i:j], Number)
			i, prev, lineStart = j, '0', false
		case isIdentStart(c):
			j := rubyIdentEnd(s, i)
			word := s[i:j]
			switch {
			case j < len(s) && s[j] == ':' && (j+1 == len(s) || s[j+1] != ':') && prev != '.':
				e.add(word+":", Key) // key: value in a hash or keyword argument
				j++
			case rubyKeywords[word] && prev != '.':
				e.add(word, Keyword)
			case rubyLiterals[word]:
				e.add(word, Literal)
			case word[0] >= 'A' && word[0] <= 'Z':
				e.add(word, Type)
			default:
				e.add(word, Plain)
			}
			i, prev, lineStart = j, 'a', false
		default:
			e.add(s[i:i+1], Plain)
			i, prev, lineStart = i+1, c, false
		}
	}
	return e.result()
}

// rubyIdentEnd returns the end of the identifier starting at s[i], taking a
// trailing ? or ! (empty?, save!) as part of it.
func rubyIdentEnd(s string, i int) int {
	j := i
	for j < len(s) && isIdentPart(s[j]) {
		j++
	}
	if j < len(s) && (s[j] == '?' || s[j] == '!') && (j+1 == len(s) || s[j+1] != '=') {
		j++
	}
	return j
}

// luaLongBracket returns the end of the long bracket [[...]] or [==[...]==]
// that opens at s[i] ('['), or -1 when s[i:] does not open one.
func luaLongBracket(s string, i int) int {
	j := i + 1
	for j < len(s) && s[j] == '=' {
		j++
	}
	if j >= len(s) || s[j] != '[' {
		return -1
	}
	closer := "]" + strings.Repeat("=", j-i-1) + "]"
	if k := strings.Index(s[j+1:], closer); k >= 0 {
		return j + 1 + k + len(closer)
	}
	return len(s)
}

// lexLua colours -- comments and --[[ ]] block comments, '...' "..." and
// [[ ]] strings, keywords, nil/true/false/self, and numbers.
func lexLua(s string) []Span {
	var e emitter
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == '-' && strings.HasPrefix(s[i:], "--"):
			j := -1
			if i+2 < len(s) && s[i+2] == '[' {
				j = luaLongBracket(s, i+2)
			}
			if j < 0 {
				j = lineEnd(s, i)
			}
			e.add(s[i:j], Comment)
			i = j
		case c == '[':
			if j := luaLongBracket(s, i); j >= 0 {
				e.add(s[i:j], String)
				i = j
				break
			}
			e.add("[", Plain)
			i++
		case c == '"' || c == '\'':
			j := scanQuoted(s, i, c, false)
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
			switch word := s[i:j]; {
			case luaKeywords[word]:
				e.add(word, Keyword)
			case luaLiterals[word]:
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
