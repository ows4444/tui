package highlight

import "strings"

func lexHTML(s string) []Span { return lexMarkup(s, true) }
func lexXML(s string) []Span  { return lexMarkup(s, false) }

// lexMarkup colours HTML or XML: tag names and their < </ > /> as keywords,
// attribute names as keys, attribute values as strings, <!-- --> comments,
// <!DOCTYPE>, <?xml ?> and other declarations as keywords, CDATA sections as
// strings and &entities; as literals. In HTML the content of <script> and
// <style> elements goes to the JavaScript and CSS lexers.
func lexMarkup(s string, html bool) []Span {
	var e emitter
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case strings.HasPrefix(s[i:], "<!--"):
			j := len(s)
			if k := strings.Index(s[i+4:], "-->"); k >= 0 {
				j = i + 4 + k + 3
			}
			e.add(s[i:j], Comment)
			i = j
		case strings.HasPrefix(s[i:], "<![CDATA["):
			j := len(s)
			if k := strings.Index(s[i:], "]]>"); k >= 0 {
				j = i + k + 3
			}
			e.add(s[i:j], String)
			i = j
		case c == '<' && i+1 < len(s) && (s[i+1] == '!' || s[i+1] == '?'):
			j := len(s)
			if k := strings.IndexByte(s[i:], '>'); k >= 0 {
				j = i + k + 1
			}
			e.add(s[i:j], Keyword)
			i = j
		case c == '<' && i+1 < len(s) && (isIdentStart(s[i+1]) || s[i+1] == '/' && i+2 < len(s) && isIdentStart(s[i+2])):
			var name string
			i, name = markupTag(s, i, &e)
			if !html || strings.HasPrefix(name, "/") {
				break
			}
			var lex lexer
			switch strings.ToLower(name) {
			case "script":
				lex = clikeLexer("js")
			case "style":
				lex = lexCSS
			}
			if lex == nil || i >= 2 && s[i-2:i] == "/>" {
				break
			}
			end := len(s)
			if k := indexFold(s[i:], "</"+name); k >= 0 {
				end = i + k
			}
			for _, sp := range lex(s[i:end]) {
				e.add(sp.Text, sp.Class)
			}
			i = end
		case c == '&':
			j := i + 1
			for j < len(s) && j-i < 32 && (isIdentPart(s[j]) || s[j] == '#') {
				j++
			}
			if j < len(s) && s[j] == ';' && j > i+1 {
				e.add(s[i:j+1], Literal)
				i = j + 1
				break
			}
			e.add("&", Plain)
			i++
		default:
			j := i + 1
			for j < len(s) && s[j] != '<' && s[j] != '&' {
				j++
			}
			e.add(s[i:j], Plain)
			i = j
		}
	}
	return e.result()
}

// markupTag colours the tag that opens at s[i] ('<') up to and including its
// closing > or />, or to the end of input, and returns where it ended and the
// tag name (with a leading / for a closing tag).
func markupTag(s string, i int, e *emitter) (int, string) {
	j := i + 1
	if s[j] == '/' {
		j++
	}
	for j < len(s) && (isIdentPart(s[j]) || s[j] == '-' || s[j] == ':' || s[j] == '.') {
		j++
	}
	name := s[i+1 : j]
	e.add(s[i:j], Keyword)
	for j < len(s) {
		c := s[j]
		switch {
		case c == '>':
			e.add(">", Keyword)
			return j + 1, name
		case c == '/' && j+1 < len(s) && s[j+1] == '>':
			e.add("/>", Keyword)
			return j + 2, name
		case c == '"' || c == '\'':
			k := strings.IndexByte(s[j+1:], c)
			end := len(s)
			if k >= 0 {
				end = j + 1 + k + 1
			}
			e.add(s[j:end], String)
			j = end
		case c == '=':
			e.add("=", Plain)
			j++
			if j < len(s) && s[j] != '"' && s[j] != '\'' && s[j] != ' ' && s[j] != '>' && s[j] != '\n' {
				k := j
				for k < len(s) && !strings.ContainsRune(" \t\r\n>", rune(s[k])) && !(s[k] == '/' && k+1 < len(s) && s[k+1] == '>') {
					k++
				}
				e.add(s[j:k], String) // an unquoted value
				j = k
			}
		case isIdentStart(c):
			k := j
			for k < len(s) && (isIdentPart(s[k]) || s[k] == '-' || s[k] == ':' || s[k] == '.') {
				k++
			}
			e.add(s[j:k], Key)
			j = k
		default:
			e.add(s[j:j+1], Plain)
			j++
		}
	}
	return j, name
}

// indexFold is strings.Index ignoring ASCII case in sub.
func indexFold(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if strings.EqualFold(s[i:i+len(sub)], sub) {
			return i
		}
	}
	return -1
}

// lexCSS colours /* */ comments, strings, @rules and !important as keywords,
// selectors as types (with pseudo-classes as keywords), property names as
// keys, and numbers with their units and #hex colours as numbers. Text is a
// selector when a { comes before the next ; or }, which also holds for rules
// nested in @media; a name followed by : inside parentheses (an at-rule's
// feature query) is a key.
func lexCSS(s string) []Span {
	var e emitter
	depth, parens := 0, 0
	selUntil, selVal := -1, false // cssSelector's answer holds up to the next ; { or }
	for i := 0; i < len(s); {
		c := s[i]
		if depth > 0 && i > selUntil {
			selUntil, selVal = cssSelector(s, i, depth)
		}
		sel := depth == 0 || selVal
		switch {
		case c == '/' && i+1 < len(s) && s[i+1] == '*':
			j := scanBlockComment(s, i, false)
			e.add(s[i:j], Comment)
			i = j
		case c == '"' || c == '\'':
			j := scanQuoted(s, i, c, false)
			e.add(s[i:j], String)
			i = j
		case c == '{' || c == '}' || c == '(' || c == ')':
			switch c {
			case '{':
				depth++
				selUntil = -1
			case '}':
				depth = max(depth-1, 0)
				selUntil = -1
			case '(':
				parens++
			case ')':
				parens = max(parens-1, 0)
			}
			e.add(s[i:i+1], Plain)
			i++
		case c == '@' && i+1 < len(s) && isIdentStart(s[i+1]):
			j := cssIdentEnd(s, i+1)
			e.add(s[i:j], Keyword)
			i = j
		case c == '!' && strings.HasPrefix(strings.ToLower(s[i:]), "!important"):
			e.add(s[i:i+10], Keyword)
			i += 10
		case !sel && c == '#' && i+1 < len(s) && isHex(s[i+1]):
			j := i + 1
			for j < len(s) && isIdentPart(s[j]) {
				j++
			}
			e.add(s[i:j], Number)
			i = j
		case isDigit(c) || !sel && (c == '.' || c == '-') && i+1 < len(s) && isDigit(s[i+1]):
			j := scanNumber(s, i+1)
			if j < len(s) && s[j] == '%' {
				j++
			}
			e.add(s[i:j], Number)
			i = j
		case sel && (c == '.' || c == '#') && i+1 < len(s) && (isIdentStart(s[i+1]) || s[i+1] == '-'):
			j := cssIdentEnd(s, i+1)
			e.add(s[i:j], Type)
			i = j
		case sel && parens == 0 && c == ':' && i+1 < len(s) && (isIdentStart(s[i+1]) || s[i+1] == ':'):
			j := i + 1
			if s[j] == ':' {
				j++
			}
			j = cssIdentEnd(s, j)
			e.add(s[i:j], Keyword) // a pseudo-class or pseudo-element
			i = j
		case isIdentStart(c) || c == '-' && i+1 < len(s) && (isIdentStart(s[i+1]) || s[i+1] == '-'):
			j := cssIdentEnd(s, i)
			k := j
			for k < len(s) && (s[k] == ' ' || s[k] == '\t') {
				k++
			}
			colon := k < len(s) && s[k] == ':' && (k+1 == len(s) || s[k+1] != ':')
			switch {
			case j < len(s) && s[j] == '(':
				e.add(s[i:j], Plain) // a function: url(), var(), calc()
			case colon && (!sel || parens > 0):
				e.add(s[i:j], Key)
			case sel && parens == 0:
				e.add(s[i:j], Type)
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

// cssSelector reports whether the text at s[i] is part of a selector or an
// at-rule prelude (outside every block, or followed by a { before the next ;
// or }), and the index of that next ; { or } (len(s) when there is none), up
// to which the answer holds.
func cssSelector(s string, i, depth int) (until int, sel bool) {
	k := strings.IndexAny(s[i:], ";{}")
	if k < 0 {
		return len(s), depth == 0
	}
	return i + k, depth == 0 || s[i+k] == '{'
}

func cssIdentEnd(s string, i int) int {
	for i < len(s) && (isIdentPart(s[i]) || s[i] == '-') {
		i++
	}
	return i
}

func isHex(b byte) bool { return isDigit(b) || b >= 'a' && b <= 'f' || b >= 'A' && b <= 'F' }
