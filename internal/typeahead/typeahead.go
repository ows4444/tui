// Package typeahead is the "type letters to jump" behaviour of a list: typing
// "ba" moves the cursor to the next item whose label starts with "ba", and a
// pause of a second starts a new word. It is shared by the list-like widgets.
package typeahead

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ows4444/tui/input"
)

// Reset is how long a pause ends the word being typed.
const Reset = time.Second

// Now is the clock; a test replaces it.
var Now = time.Now

// State is the letters typed so far and where the word began. The zero value is
// an empty word. It is a small value, so a widget that is copied keeps its own.
type State struct {
	text   string
	at     time.Time
	origin int
}

// Printable reports whether msg is a typed character a list may take as part
// of a word: a rune key with no Ctrl or Alt, and not a release. It returns the
// text.
func Printable(msg any) (string, bool) {
	k, ok := msg.(input.Key)
	if !ok || k.Type != input.KeyRunes || k.Text == "" || k.Action == input.KeyRelease || k.Mod&^input.ModShift != 0 {
		return "", false
	}
	return k.Text, true
}

// Feed adds text to the word, starting a new one after a pause, and returns the
// prefix to look for and the index to start looking from. cursor is where the
// list's cursor is now.
//
// A word of one letter, or the same letter again and again, looks for the next
// item after the cursor that starts with that letter, so repeating the letter
// cycles through them. A longer, mixed word looks from where the word began,
// the cursor included, so typing "item 0000" on a list that starts there lands
// back on the first item, not the one after.
func (s *State) Feed(text string, cursor int) (prefix string, from int) {
	t := Now()
	if t.Sub(s.at) > Reset || s.text == "" {
		s.text, s.origin = "", cursor
	}
	s.at = t
	s.text += text
	first, n := utf8.DecodeRuneInString(s.text)
	if strings.Repeat(string(first), utf8.RuneCountInString(s.text)) == s.text {
		return s.text[:n], cursor + 1
	}
	return s.text, s.origin
}

// Clear forgets the word.
func (s *State) Clear() { s.text = "" }

// Find returns the index of the first of n items, starting at from and going
// round, whose label starts with prefix, ignoring case. label is called for as
// many items as it takes to find one.
func Find(prefix string, from, n int, label func(i int) string) (int, bool) {
	if n <= 0 || prefix == "" {
		return 0, false
	}
	want := strings.ToLower(prefix)
	for k := 0; k < n; k++ {
		i := ((from+k)%n + n) % n
		if strings.HasPrefix(strings.ToLower(label(i)), want) {
			return i, true
		}
	}
	return 0, false
}
