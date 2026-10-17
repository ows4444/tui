// Package hotkey reads the mnemonic marker of a label: "&Save" is the label
// "Save" with S as its hot key, drawn underlined and pressed with Alt+S.
// "&&" is a literal ampersand. It is shared by the widgets that take such a
// label (button, buttongroup, tabs).
package hotkey

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/input"
)

// Label is a label split into the text to draw and its hot key.
type Label struct {
	// Text is the label without its marker.
	Text string
	// Key is the hot key, lower-cased; 0 when the label has none.
	Key rune
	// Pos is the index in Text, in bytes, of the hot key's rune; -1 when none.
	Pos int
}

// Parse reads s, which the caller has already sanitised. The first "&" that is
// followed by a letter or digit marks the hot key and is removed; "&&" is one
// "&"; any other "&" stays. A label with no marker has no hot key.
func Parse(s string) Label {
	if !strings.Contains(s, "&") {
		return Label{Text: s, Pos: -1}
	}
	var b strings.Builder
	l := Label{Pos: -1}
	for i := 0; i < len(s); {
		if s[i] != '&' {
			_, n := utf8.DecodeRuneInString(s[i:])
			b.WriteString(s[i : i+n])
			i += n
			continue
		}
		if i+1 < len(s) && s[i+1] == '&' {
			b.WriteByte('&')
			i += 2
			continue
		}
		if l.Key == 0 && i+1 < len(s) {
			if r, n := utf8.DecodeRuneInString(s[i+1:]); unicode.IsLetter(r) || unicode.IsDigit(r) {
				l.Key, l.Pos = unicode.ToLower(r), b.Len()
				b.WriteString(s[i+1 : i+1+n])
				i += 1 + n
				continue
			}
		}
		b.WriteByte('&')
		i++
	}
	l.Text = b.String()
	return l
}

// Hit reports whether msg is Alt plus the label's hot key, pressed.
func (l Label) Hit(msg any) bool {
	k, ok := msg.(input.Key)
	if !ok || l.Key == 0 || k.Type != input.KeyRunes || k.Action == input.KeyRelease {
		return false
	}
	if k.Mod&^input.ModShift != input.ModAlt {
		return false
	}
	r, n := utf8.DecodeRuneInString(k.Text)
	return n == len(k.Text) && n > 0 && unicode.ToLower(r) == l.Key
}

// Render draws text in base with the rune at pos underlined (the whole of
// text in base when pos is negative or outside it).
func Render(text string, pos int, base ansi.Style) string {
	if pos < 0 || pos >= len(text) {
		return base.Render(text)
	}
	_, n := utf8.DecodeRuneInString(text[pos:])
	return base.Render(text[:pos]) + base.Underline().Render(text[pos:pos+n]) + base.Render(text[pos+n:])
}

// Spoken is the hot key as plain text for accessible output: "Alt+S", or "" for
// a label with none.
func (l Label) Spoken() string {
	if l.Key == 0 {
		return ""
	}
	return "Alt+" + strings.ToUpper(string(l.Key))
}
