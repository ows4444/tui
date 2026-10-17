package main

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// sqlError is a problem with the query, with where in it the problem is.
type sqlError struct {
	msg string
	pos int // byte offset into the query, or -1
}

func (e *sqlError) Error() string { return e.msg }

func errAt(pos int, f string, a ...any) *sqlError { return &sqlError{fmt.Sprintf(f, a...), pos} }

type tokKind int

const (
	tEOF tokKind = iota
	tIdent
	tNumber
	tString
	tSym // punctuation and operators
)

type token struct {
	kind tokKind
	text string // idents keep their case; symbols are as written
	pos  int
	num  float64
}

// upper is the token's text in capitals, for matching keywords.
func (t token) upper() string { return strings.ToUpper(t.text) }

// lex splits a query into tokens. Strings use single quotes, with ” for a
// quote inside; -- starts a comment to the end of the line.
func lex(q string) ([]token, *sqlError) {
	var out []token
	r := []rune(q)
	// positions are byte offsets, so walk the string by bytes of its runes
	offsets := make([]int, len(r)+1)
	for i, off := 0, 0; i < len(r); i++ {
		offsets[i] = off
		off += len(string(r[i]))
		offsets[i+1] = off
	}
	for i := 0; i < len(r); {
		c := r[i]
		switch {
		case unicode.IsSpace(c):
			i++
		case c == '-' && i+1 < len(r) && r[i+1] == '-':
			for i < len(r) && r[i] != '\n' {
				i++
			}
		case unicode.IsLetter(c) || c == '_':
			j := i
			for j < len(r) && (unicode.IsLetter(r[j]) || unicode.IsDigit(r[j]) || r[j] == '_') {
				j++
			}
			out = append(out, token{kind: tIdent, text: string(r[i:j]), pos: offsets[i]})
			i = j
		case unicode.IsDigit(c) || (c == '.' && i+1 < len(r) && unicode.IsDigit(r[i+1])):
			j := i
			for j < len(r) && (unicode.IsDigit(r[j]) || r[j] == '.') {
				j++
			}
			n, err := strconv.ParseFloat(string(r[i:j]), 64)
			if err != nil {
				return nil, errAt(offsets[i], "%q is not a number", string(r[i:j]))
			}
			out = append(out, token{kind: tNumber, text: string(r[i:j]), pos: offsets[i], num: n})
			i = j
		case c == '\'':
			var b strings.Builder
			j := i + 1
			for ; j < len(r); j++ {
				if r[j] == '\'' {
					if j+1 < len(r) && r[j+1] == '\'' {
						b.WriteRune('\'')
						j++
						continue
					}
					break
				}
				b.WriteRune(r[j])
			}
			if j >= len(r) {
				return nil, errAt(offsets[i], "the text starting here has no closing quote")
			}
			out = append(out, token{kind: tString, text: b.String(), pos: offsets[i]})
			i = j + 1
		default:
			sym := string(c)
			if i+1 < len(r) {
				switch two := string(r[i : i+2]); two {
				case "<=", ">=", "<>", "!=":
					sym = two
				}
			}
			if !strings.Contains("(),*=<>+-/%;.", string(c)) && sym == string(c) {
				return nil, errAt(offsets[i], "unexpected %q", string(c))
			}
			out = append(out, token{kind: tSym, text: sym, pos: offsets[i]})
			i += len([]rune(sym))
		}
	}
	return append(out, token{kind: tEOF, pos: len(q)}), nil
}
