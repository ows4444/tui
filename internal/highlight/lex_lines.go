package highlight

import "strings"

// eachLine calls fn for every "\n"-separated line of s (without the newline)
// and re-emits the newlines, so the spans still reassemble the input.
func eachLine(s string, e *emitter, fn func(line string)) {
	for {
		i := strings.IndexByte(s, '\n')
		if i < 0 {
			fn(s)
			return
		}
		fn(s[:i])
		e.add("\n", Plain)
		s = s[i+1:]
	}
}

// lexDiff colours a unified diff: file headers are keywords, hunk headers
// types, added lines strings, removed lines numbers and the "\ No newline"
// marker a comment.
func lexDiff(s string) []Span {
	var e emitter
	eachLine(s, &e, func(l string) {
		switch {
		case strings.HasPrefix(l, "+++"), strings.HasPrefix(l, "---"),
			strings.HasPrefix(l, "diff "), strings.HasPrefix(l, "index "),
			strings.HasPrefix(l, "new file"), strings.HasPrefix(l, "deleted file"),
			strings.HasPrefix(l, "rename "), strings.HasPrefix(l, "similarity "),
			strings.HasPrefix(l, "old mode"), strings.HasPrefix(l, "new mode"):
			e.add(l, Keyword)
		case strings.HasPrefix(l, "@@"):
			end := strings.Index(l[2:], "@@")
			if end < 0 {
				e.add(l, Type)
				return
			}
			end += 4
			e.add(l[:end], Type)
			e.add(l[end:], Plain)
		case strings.HasPrefix(l, "+"):
			e.add(l, String)
		case strings.HasPrefix(l, "-"):
			e.add(l, Number)
		case strings.HasPrefix(l, "\\"):
			e.add(l, Comment)
		default:
			e.add(l, Plain)
		}
	})
	return e.result()
}

// mdListMarker returns the length of a list marker ("- ", "* ", "+ ", "12. ")
// at the start of l after its indentation, or 0.
func mdListMarker(l string) int {
	i := 0
	for i < len(l) && (l[i] == ' ' || l[i] == '\t') {
		i++
	}
	j := i
	switch {
	case j < len(l) && strings.IndexByte("-*+", l[j]) >= 0:
		j++
	case j < len(l) && isDigit(l[j]):
		for j < len(l) && isDigit(l[j]) {
			j++
		}
		if j >= len(l) || (l[j] != '.' && l[j] != ')') {
			return 0
		}
		j++
	default:
		return 0
	}
	if j < len(l) && l[j] == ' ' {
		return j + 1
	}
	return 0
}

// isMDRule reports whether l is a thematic break such as "---" or "***".
func isMDRule(l string) bool {
	t := strings.ReplaceAll(strings.TrimSpace(l), " ", "")
	if len(t) < 3 || strings.IndexByte("-*_", t[0]) < 0 {
		return false
	}
	return strings.Trim(t, t[:1]) == ""
}

// lexMarkdown colours headings and list markers as keywords, fences and
// rules as comments, inline code as strings, emphasis as types and links as
// a key (the text) and a variable (the target). Fenced code stays plain.
func lexMarkdown(s string) []Span {
	var e emitter
	fence := ""
	eachLine(s, &e, func(l string) {
		trim := strings.TrimLeft(l, " ")
		if fence != "" {
			if strings.HasPrefix(trim, fence) && strings.Trim(trim, fence[:1]) == "" {
				fence = ""
				e.add(l, Comment)
			} else {
				e.add(l, Plain)
			}
			return
		}
		switch {
		case strings.HasPrefix(trim, "```") || strings.HasPrefix(trim, "~~~"):
			fence = trim[:3]
			e.add(l, Comment)
		case strings.HasPrefix(trim, "#"):
			n := 0
			for n < len(trim) && trim[n] == '#' {
				n++
			}
			if n <= 6 && (n == len(trim) || trim[n] == ' ' || trim[n] == '\t') {
				e.add(l, Keyword)
				return
			}
			mdInline(&e, l)
		case isMDRule(l):
			e.add(l, Comment)
		case strings.HasPrefix(trim, ">"):
			k := len(l) - len(trim) + 1
			e.add(l[:k], Comment)
			mdInline(&e, l[k:])
		default:
			if n := mdListMarker(l); n > 0 {
				e.add(l[:n-1], Keyword)
				l = l[n-1:]
			}
			mdInline(&e, l)
		}
	})
	return e.result()
}

// mdInline colours the inline elements of one line.
func mdInline(e *emitter, l string) {
	for i := 0; i < len(l); {
		c := l[i]
		switch {
		case c == '\\' && i+1 < len(l):
			e.add(l[i:i+2], Plain)
			i += 2
		case c == '`':
			n := 0
			for i+n < len(l) && l[i+n] == '`' {
				n++
			}
			ticks := l[i : i+n]
			if k := strings.Index(l[i+n:], ticks); k >= 0 {
				e.add(l[i:i+n+k+n], String)
				i += n + k + n
			} else {
				e.add(ticks, Plain)
				i += n
			}
		case c == '*' || c == '_':
			n := 1
			if i+1 < len(l) && l[i+1] == c {
				n = 2
			}
			mark := l[i : i+n]
			rest := l[i+n:]
			k := strings.Index(rest, mark)
			if k > 0 && rest[0] != ' ' && rest[k-1] != ' ' && (c == '*' || mdWordEdge(l, i, i+n+k+n)) {
				e.add(l[i:i+n+k+n], Type)
				i += n + k + n
			} else {
				e.add(mark, Plain)
				i += n
			}
		case c == '[':
			end := strings.IndexByte(l[i:], ']')
			if end > 0 && i+end+1 < len(l) && l[i+end+1] == '(' {
				if close := strings.IndexByte(l[i+end+1:], ')'); close > 0 {
					e.add(l[i:i+end+1], Key)
					e.add(l[i+end+1:i+end+1+close+1], Variable)
					i += end + 1 + close + 1
					continue
				}
			}
			e.add("[", Plain)
			i++
		default:
			e.add(l[i:i+1], Plain)
			i++
		}
	}
}

// mdWordEdge reports whether the span l[from:to] is not glued to word
// characters, so snake_case names are not read as emphasis.
func mdWordEdge(l string, from, to int) bool {
	if from > 0 && isIdentPart(l[from-1]) {
		return false
	}
	return to >= len(l) || !isIdentPart(l[to])
}

var dockerInstructions = setOf("FROM", "RUN", "CMD", "LABEL", "MAINTAINER", "EXPOSE", "ENV",
	"ADD", "COPY", "ENTRYPOINT", "VOLUME", "USER", "WORKDIR", "ARG", "ONBUILD", "STOPSIGNAL",
	"HEALTHCHECK", "SHELL")

// lexDockerfile colours instructions as keywords, comments, quoted strings,
// $VAR and ${VAR} as variables and "AS" in a FROM line as a keyword. A line
// that follows a trailing backslash continues the previous instruction.
func lexDockerfile(s string) []Span {
	var e emitter
	cont := false
	from := false
	eachLine(s, &e, func(l string) {
		trim := strings.TrimLeft(l, " \t")
		if strings.HasPrefix(trim, "#") {
			e.add(l, Comment)
			return
		}
		rest := l
		if !cont {
			from = false
			end := 0
			for end < len(trim) && isIdentPart(trim[end]) {
				end++
			}
			if word := trim[:end]; dockerInstructions[strings.ToUpper(word)] {
				ind := len(l) - len(trim)
				e.add(l[:ind], Plain)
				e.add(word, Keyword)
				rest = trim[end:]
				from = strings.EqualFold(word, "FROM")
			}
		}
		cont = strings.HasSuffix(strings.TrimRight(l, " \t\r"), "\\")
		dockerArgs(&e, rest, from)
	})
	return e.result()
}

func dockerArgs(e *emitter, s string, from bool) {
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == '"' || c == '\'':
			j := scanQuoted(s, i, c, false)
			e.add(s[i:j], String)
			i = j
		case c == '$' && i+1 < len(s) && (s[i+1] == '{' || isIdentStart(s[i+1])):
			j := i + 1
			if s[j] == '{' {
				if k := strings.IndexByte(s[j:], '}'); k >= 0 {
					j += k + 1
				} else {
					j = lineEnd(s, j)
				}
			} else {
				for j < len(s) && isIdentPart(s[j]) {
					j++
				}
			}
			e.add(s[i:j], Variable)
			i = j
		case isIdentStart(c):
			j := i
			for j < len(s) && (isIdentPart(s[j]) || s[j] == '-' || s[j] == '.') {
				j++
			}
			if from && strings.EqualFold(s[i:j], "as") && i > 0 && s[i-1] == ' ' {
				e.add(s[i:j], Keyword)
			} else {
				e.add(s[i:j], Plain)
			}
			i = j
		default:
			e.add(s[i:i+1], Plain)
			i++
		}
	}
}
