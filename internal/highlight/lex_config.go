package highlight

import (
	"strings"
	"unicode/utf8"
)

var iniLiterals = setOf("true", "false", "yes", "no", "on", "off", "none", "null")

// lexINI colours ; and # comment lines, [section] headers as types, the key
// of a key = value (or key: value) line as a key, and the value as a string
// when quoted, a number, a literal (true, yes, on, ...) or plain text.
func lexINI(s string) []Span {
	var e emitter
	eachLine(s, &e, func(l string) {
		trim := strings.TrimLeft(l, " \t")
		ind := l[:len(l)-len(trim)]
		switch {
		case trim == "":
			e.add(l, Plain)
		case trim[0] == ';' || trim[0] == '#':
			e.add(ind, Plain)
			e.add(trim, Comment)
		case trim[0] == '[':
			end := strings.IndexByte(trim, ']') + 1
			if end == 0 {
				end = len(trim)
			}
			e.add(ind, Plain)
			e.add(trim[:end], Type)
			e.add(trim[end:], Plain)
		default:
			sep := strings.IndexAny(trim, "=:")
			if sep < 0 {
				e.add(l, Plain)
				return
			}
			key := strings.TrimRight(trim[:sep], " \t")
			e.add(ind, Plain)
			e.add(key, Key)
			e.add(trim[len(key):sep+1], Plain)
			iniValue(&e, trim[sep+1:])
		}
	})
	return e.result()
}

func iniValue(e *emitter, v string) {
	val := strings.Trim(v, " \t\r")
	lead := v[:strings.Index(v, val)] // leading blanks (val is a substring of v)
	if val == "" {
		e.add(v, Plain)
		return
	}
	e.add(lead, Plain)
	switch {
	case len(val) >= 2 && (val[0] == '"' || val[0] == '\'') && val[len(val)-1] == val[0]:
		e.add(val, String)
	case isNumberWord(val):
		e.add(val, Number)
	case iniLiterals[strings.ToLower(val)]:
		e.add(val, Literal)
	default:
		e.add(val, Plain)
	}
	e.add(v[len(lead)+len(val):], Plain)
}

// isNumberWord reports whether w is a whole decimal or hex number such as
// 42, -3.5 or 0x1F.
func isNumberWord(w string) bool {
	if w != "" && (w[0] == '-' || w[0] == '+') {
		w = w[1:]
	}
	return w != "" && isDigit(w[0]) && scanNumber(w, 0) == len(w)
}

var makeDirectives = setOf("include", "-include", "sinclude", "ifeq", "ifneq", "ifdef", "ifndef",
	"else", "endif", "define", "endef", "export", "unexport", "override", "private", "vpath")

// lexMakefile colours # comments, directives (include, ifeq, define, ...) as
// keywords, the variable of an assignment and every $(VAR), ${VAR} and
// automatic variable ($@, $<, ...) as variables, rule targets as keys, and
// quoted strings. Recipe lines (those starting with a tab) keep only the
// variable and string colouring, since make hands them to the shell.
func lexMakefile(s string) []Span {
	var e emitter
	eachLine(s, &e, func(l string) {
		if strings.HasPrefix(l, "\t") {
			makeArgs(&e, l, true)
			return
		}
		trim := strings.TrimLeft(l, " ")
		ind := l[:len(l)-len(trim)]
		if strings.HasPrefix(trim, "#") {
			e.add(ind, Plain)
			e.add(trim, Comment)
			return
		}
		word := trim
		if k := strings.IndexAny(trim, " \t("); k >= 0 {
			word = trim[:k]
		}
		if makeDirectives[word] {
			e.add(ind, Plain)
			e.add(word, Keyword)
			makeArgs(&e, trim[len(word):], false)
			return
		}
		if name, op, ok := makeAssignment(trim); ok {
			e.add(ind, Plain)
			e.add(name, Variable)
			e.add(trim[len(name):op], Plain)
			makeArgs(&e, trim[op:], false)
			return
		}
		if colon := makeRuleColon(trim); colon > 0 {
			e.add(ind, Plain)
			makeTargets(&e, trim[:colon])
			makeArgs(&e, trim[colon:], false)
			return
		}
		makeArgs(&e, l, false)
	})
	return e.result()
}

// makeAssignment reports whether l assigns a variable (NAME = ..., :=, ::=,
// ?=, += or !=), returning the name and the index just past the operator.
func makeAssignment(l string) (name string, end int, ok bool) {
	j := 0
	for j < len(l) && (isIdentPart(l[j]) || l[j] == '.' || l[j] == '-' || l[j] == '/') {
		j++
	}
	if j == 0 {
		return "", 0, false
	}
	k := j
	for k < len(l) && (l[k] == ' ' || l[k] == '\t') {
		k++
	}
	for _, op := range []string{"::=", ":=", "?=", "+=", "!=", "="} {
		if strings.HasPrefix(l[k:], op) {
			return l[:j], k + len(op), true
		}
	}
	return "", 0, false
}

// makeRuleColon returns the index of the colon that ends a rule's target
// list, or 0 when l is not a rule. A colon inside $(...) does not count.
func makeRuleColon(l string) int {
	depth := 0
	for i := 0; i < len(l); i++ {
		switch l[i] {
		case '(', '{':
			depth++
		case ')', '}':
			if depth > 0 {
				depth--
			}
		case '=', '#':
			if depth == 0 {
				return 0
			}
		case ':':
			if depth == 0 && i > 0 {
				return i
			}
		}
	}
	return 0
}

// makeTargets colours the target names of a rule as keys and any variable
// references among them as variables.
func makeTargets(e *emitter, t string) {
	for i := 0; i < len(t); {
		if t[i] == '$' {
			j := makeVarEnd(t, i)
			e.add(t[i:j], Variable)
			i = j
			continue
		}
		if t[i] == ' ' || t[i] == '\t' {
			e.add(t[i:i+1], Plain)
			i++
			continue
		}
		j := i
		for j < len(t) && t[j] != ' ' && t[j] != '\t' && t[j] != '$' {
			j++
		}
		e.add(t[i:j], Key)
		i = j
	}
}

// makeArgs colours variable references and quoted strings in s, and #
// comments. Outside a recipe any # starts one, as make reads it; in a recipe
// (recipe set) only a # at the start of a word does, as the shell reads it.
func makeArgs(e *emitter, s string, recipe bool) {
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == '$':
			j := makeVarEnd(s, i)
			e.add(s[i:j], Variable)
			i = j
		case c == '#' && (!recipe || i == 0 || s[i-1] == ' ' || s[i-1] == '\t'):
			e.add(s[i:], Comment)
			return
		case c == '"' || c == '\'':
			j := scanQuoted(s, i, c, false)
			e.add(s[i:j], String)
			i = j
		default:
			j := i + 1 // a run of plain text, added at once so the emitter stays linear
			for j < len(s) && !strings.ContainsRune("$#\"'", rune(s[j])) {
				j++
			}
			e.add(s[i:j], Plain)
			i = j
		}
	}
}

// makeVarEnd returns the end of the variable reference at s[i] ('$'):
// $(NAME ...) or ${NAME} with nesting, a one-character name such as $@, or
// $$ (a dollar passed to the shell) together with the shell variable it
// starts, as in $${HOME} or $$PATH.
func makeVarEnd(s string, i int) int {
	if i+1 >= len(s) {
		return i + 1
	}
	if s[i+1] == '$' && i+2 < len(s) && (s[i+2] == '{' || s[i+2] == '(' || isIdentStart(s[i+2])) {
		if s[i+2] == '{' || s[i+2] == '(' {
			return makeVarEnd(s, i+1)
		}
		j := i + 2
		for j < len(s) && isIdentPart(s[j]) {
			j++
		}
		return j
	}
	open := s[i+1]
	if open != '(' && open != '{' {
		_, n := utf8.DecodeRuneInString(s[i+1:])
		return i + 1 + n // $@ $< $^ $* $$ $x, never splitting a rune
	}
	closer := byte(')')
	if open == '{' {
		closer = '}'
	}
	depth := 0
	for j := i + 1; j < len(s); j++ {
		switch s[j] {
		case open:
			depth++
		case closer:
			if depth--; depth == 0 {
				return j + 1
			}
		}
	}
	return len(s)
}
