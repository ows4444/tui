package highlight

import "strings"

var sqlKeywords = setOf("add", "all", "alter", "and", "as", "asc", "begin", "between", "by",
	"case", "check", "column", "commit", "constraint", "create", "cross", "database", "default",
	"delete", "desc", "distinct", "drop", "else", "end", "exists", "foreign", "from", "full",
	"group", "having", "in", "index", "inner", "insert", "into", "is", "join", "key", "left",
	"like", "limit", "not", "offset", "on", "or", "order", "outer", "primary", "references",
	"returning", "right", "rollback", "select", "set", "table", "then", "transaction", "union",
	"unique", "update", "values", "view", "when", "where", "with")

var sqlTypes = setOf("bigint", "blob", "boolean", "char", "date", "datetime", "decimal",
	"double", "float", "int", "integer", "json", "numeric", "real", "serial", "smallint",
	"text", "time", "timestamp", "uuid", "varchar")

var sqlLiterals = setOf("true", "false", "null")

// lexSQL colours keywords case-insensitively, '...' strings, "..." and `...`
// identifiers as keys, numbers, -- and /* */ comments.
func lexSQL(s string) []Span {
	var e emitter
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == '-' && i+1 < len(s) && s[i+1] == '-':
			j := lineEnd(s, i)
			e.add(s[i:j], Comment)
			i = j
		case c == '/' && i+1 < len(s) && s[i+1] == '*':
			j := scanBlockComment(s, i, false)
			e.add(s[i:j], Comment)
			i = j
		case c == '\'':
			j := scanPlainQuoted(s, i, c)
			e.add(s[i:j], String)
			i = j
		case c == '"' || c == '`':
			j := scanPlainQuoted(s, i, c)
			e.add(s[i:j], Key)
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
			switch w := strings.ToLower(s[i:j]); {
			case sqlKeywords[w]:
				e.add(s[i:j], Keyword)
			case sqlTypes[w]:
				e.add(s[i:j], Type)
			case sqlLiterals[w]:
				e.add(s[i:j], Literal)
			default:
				e.add(s[i:j], Plain)
			}
			i = j
		default:
			e.add(s[i:i+1], Plain)
			i++
		}
	}
	return e.result()
}

// scanPlainQuoted is scanQuoted without backslash escapes: a doubled quote
// ends and restarts the string, which the emitter merges back together. An
// unterminated string stops at the newline.
func scanPlainQuoted(s string, i int, quote byte) int {
	j := i + 1
	for j < len(s) && s[j] != quote && s[j] != '\n' {
		j++
	}
	if j < len(s) && s[j] == quote {
		return j + 1
	}
	return j
}

// scanTOMLString returns the end of the string at s[i]: a triple-quoted
// string runs to its closing triple (basic strings honour backslash escapes),
// any other ends at its quote or the end of the line.
func scanTOMLString(s string, i int) int {
	q := s[i]
	if strings.HasPrefix(s[i:], string([]byte{q, q, q})) {
		triple := s[i : i+3]
		for j := i + 3; j < len(s); j++ {
			switch {
			case s[j] == '\\' && q == '"':
				j++
			case strings.HasPrefix(s[j:], triple):
				j += 3
				for j < len(s) && s[j] == q { // up to two quotes may precede the closer
					j++
				}
				return j
			}
		}
		return len(s)
	}
	if q == '\'' {
		return scanPlainQuoted(s, i, q)
	}
	return scanQuoted(s, i, q, false)
}

func isTOMLKeyByte(b byte) bool { return isIdentPart(b) || b == '-' }

// lexTOML colours tables as types, keys as keys, strings, numbers and dates,
// booleans as literals and # comments. A key is a bare or quoted (dotted)
// name at the start of a line that is followed by '='.
func lexTOML(s string) []Span {
	var e emitter
	lineStart := true
	for i := 0; i < len(s); {
		c := s[i]
		switch {
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
			i = j
		case c == '[' && lineStart:
			j := lineEnd(s, i)
			end := strings.IndexByte(s[i:j], ']')
			if end < 0 {
				e.add(s[i:j], Type)
				i = j
				break
			}
			end += i + 1
			if end < j && s[end] == ']' {
				end++
			}
			e.add(s[i:end], Type)
			i = end
			lineStart = false
		case lineStart && tomlKeyAt(s, i) > 0:
			j := tomlKeyAt(s, i)
			e.add(s[i:j], Key)
			i = j
			lineStart = false
		case c == '"' || c == '\'':
			j := scanTOMLString(s, i)
			e.add(s[i:j], String)
			i = j
			lineStart = false
		case isDigit(c) || (c == '+' || c == '-') && i+1 < len(s) && (isDigit(s[i+1]) || isIdentStart(s[i+1])):
			j := i + 1
			for j < len(s) && (isIdentPart(s[j]) || strings.IndexByte(".:+-", s[j]) >= 0) {
				j++
			}
			if w := s[i+1 : j]; !isDigit(c) && isIdentStart(s[i+1]) && w != "inf" && w != "nan" {
				e.add(s[i:i+1], Plain)
				i++
			} else {
				e.add(s[i:j], Number)
				i = j
			}
			lineStart = false
		case isIdentStart(c):
			j := i
			for j < len(s) && isIdentPart(s[j]) {
				j++
			}
			switch s[i:j] {
			case "true", "false":
				e.add(s[i:j], Literal)
			case "inf", "nan":
				e.add(s[i:j], Number)
			default:
				e.add(s[i:j], Plain)
			}
			i = j
			lineStart = false
		default:
			e.add(s[i:i+1], Plain)
			i++
			lineStart = false
		}
	}
	return e.result()
}

// tomlKeyAt returns the end of the (dotted) key starting at s[i] when it is
// followed by '=' on the same line, else 0.
func tomlKeyAt(s string, i int) int {
	j := i
	for j < len(s) {
		switch c := s[j]; {
		case c == '"' || c == '\'':
			j = scanPlainQuoted(s, j, c)
		case isTOMLKeyByte(c) || c == '.':
			j++
		default:
			goto done
		}
	}
done:
	if j == i {
		return 0
	}
	k := j
	for k < len(s) && (s[k] == ' ' || s[k] == '\t') {
		k++
	}
	if k < len(s) && s[k] == '=' {
		return j
	}
	return 0
}
