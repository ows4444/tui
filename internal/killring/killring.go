// Package killring is the ring of killed text that readline and Emacs call the
// kill ring: Ctrl+K, Ctrl+U and Ctrl+W save what they delete, Ctrl+Y yanks the
// newest back and Alt+Y replaces it with an older one. A Ring is a small value;
// every method returns the new Ring and leaves the receiver alone, so a widget
// that is copied keeps its own history.
package killring

import "slices"

// max is how many kills a Ring keeps.
const max = 16

type kind uint8

const (
	none kind = iota
	killed
	yanked
)

// Ring holds the kills, newest first, and what the last command was, since
// consecutive kills join into one entry and Alt+Y only works right after a
// yank.
type Ring struct {
	items []string
	last  kind
	pick  int // the entry the last yank inserted
}

// Kill saves text. Right after another kill it joins that entry instead of
// starting a new one: after it when backward is false (Ctrl+K), before it when
// true (Ctrl+U, Ctrl+W), so a run of kills reads in document order. Empty text
// saves nothing but keeps a run going.
func (r Ring) Kill(text string, backward bool) Ring {
	if text == "" {
		if r.last == killed {
			return r
		}
		r.last = killed
		return r
	}
	items := slices.Clone(r.items)
	switch {
	case r.last == killed && len(items) > 0 && backward:
		items[0] = text + items[0]
	case r.last == killed && len(items) > 0:
		items[0] += text
	default:
		items = append([]string{text}, items...)
		if len(items) > max {
			items = items[:max]
		}
	}
	return Ring{items: items, last: killed}
}

// Len is the number of entries.
func (r Ring) Len() int { return len(r.items) }

// Yank returns the newest entry and the Ring noting that a yank happened, or
// false when the ring is empty.
func (r Ring) Yank() (string, Ring, bool) {
	if len(r.items) == 0 {
		return "", r, false
	}
	r.last, r.pick = yanked, 0
	return r.items[0], r, true
}

// Pop returns the next older entry than the one the last yank inserted, going
// round to the newest after the oldest, and the Ring noting it. It is false
// unless the last command was a yank (or a pop) and there is another entry.
func (r Ring) Pop() (string, Ring, bool) {
	if r.last != yanked || len(r.items) < 2 {
		return "", r, false
	}
	r.pick = (r.pick + 1) % len(r.items)
	return r.items[r.pick], r, true
}

// Other ends a run of kills or yanks: call it after any command that is not a
// kill, a yank or a pop.
func (r Ring) Other() Ring {
	r.last = none
	return r
}

// Last reports whether the last command was a kill, a yank, or neither, for a
// caller that needs to know before it decides what a key means.
func (r Ring) Last() (wasKill, wasYank bool) { return r.last == killed, r.last == yanked }
