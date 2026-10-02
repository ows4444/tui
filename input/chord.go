package input

import "time"

// DefaultChordTimeout is how long ChordMatcher waits for the next key of a
// still-possible chord before giving up on it. Vim-style multi-key
// sequences (g g, d d) and Emacs-style prefix bindings (ctrl+x ctrl+s) both
// use a window in roughly this range.
const DefaultChordTimeout = 500 * time.Millisecond

// ChordMsg is delivered by ChordMatcher.Feed when a configured chord
// completes: Name is whichever ChordDef.Name matched.
type ChordMsg struct{ Name string }

// ChordDef configures one multi-key chord ChordMatcher recognizes. Keys is
// the ordered sequence of Key.String() forms (e.g. "g", "ctrl+x") that must
// arrive one after another for the chord to fire.
type ChordDef struct {
	Name string
	Keys []string
}

// ChordMatcher accumulates Key events against a set of configured
// ChordDefs. It has no framework dependency: an application feeds it every
// Key event from tui.Update and acts on the returned ChordResult, so it
// works the same whether or not the caller happens to be a tui.Program.
//
// ChordMatcher is not goroutine-safe; use it from a single Update-driven
// call site, the same assumption every other stateful piece of this
// library makes.
type ChordMatcher struct {
	defs []ChordDef
	// Timeout overrides DefaultChordTimeout; the zero value (before any
	// Feed call sets it) means DefaultChordTimeout.
	Timeout time.Duration

	buf  []Key
	last time.Time
}

// NewChordMatcher returns a ChordMatcher recognizing defs, using
// DefaultChordTimeout.
func NewChordMatcher(defs ...ChordDef) *ChordMatcher {
	return &ChordMatcher{defs: defs, Timeout: DefaultChordTimeout}
}

// ChordResult is Feed's outcome for one Key.
type ChordResult struct {
	// Msg is non-nil when k completed a configured chord.
	Msg *ChordMsg
	// Pending is true when k extends a still-possible chord prefix: the
	// caller should wait rather than act on k directly. If the chord never
	// completes, a later Feed or Expire call returns the buffered keys via
	// Flush.
	Pending bool
	// Flush holds keys that are NOT part of any completed chord — a
	// previously buffered prefix that turned out to be a dead end, or (when
	// no chord could ever match) k itself — in the order they should be
	// delivered to the application, as if chord matching had never seen
	// them.
	Flush []Key
}

// Feed advances the matcher by one Key, arriving at time now. now is a
// parameter (rather than time.Now()) so callers, and this package's own
// tests, can drive it deterministically.
func (c *ChordMatcher) Feed(k Key, now time.Time) ChordResult {
	if k.Action != KeyPress {
		// A kitty release or repeat is not a keystroke of a chord: pass it
		// through untouched and leave any pending prefix as it is.
		return ChordResult{Flush: []Key{k}}
	}
	if c.Timeout <= 0 {
		c.Timeout = DefaultChordTimeout
	}
	var expired []Key
	if len(c.buf) > 0 && now.Sub(c.last) > c.Timeout {
		expired = c.buf
		c.buf = nil
	}

	res := c.feedFresh(k, now)
	if len(expired) > 0 {
		res.Flush = append(append([]Key{}, expired...), res.Flush...)
	}
	return res
}

// Expire flushes a pending prefix that has timed out even though no
// further key has arrived to trigger Feed's own expiry check — e.g. driven
// by an application's own idle/tick timer. It returns nil if nothing is
// pending or the pending prefix hasn't timed out yet.
func (c *ChordMatcher) Expire(now time.Time) []Key {
	if len(c.buf) == 0 || now.Sub(c.last) <= c.Timeout {
		return nil
	}
	old := c.buf
	c.buf = nil
	return old
}

func (c *ChordMatcher) feedFresh(k Key, now time.Time) ChordResult {
	trial := append(append([]Key{}, c.buf...), k)
	if name, ok := c.exactMatch(trial); ok {
		c.buf = nil
		return ChordResult{Msg: &ChordMsg{Name: name}}
	}
	if c.hasPrefixMatch(trial) {
		c.buf = trial
		c.last = now
		return ChordResult{Pending: true}
	}

	// trial (buf+k) doesn't lead anywhere. If buf was empty, k alone simply
	// isn't part of any chord.
	if len(c.buf) == 0 {
		return ChordResult{Flush: []Key{k}}
	}

	// buf was a dead end once k extended it; flush buf and retry k as a
	// fresh start of its own, since k might begin a different chord (e.g.
	// buf="g" (pending), k="x": "g x" matches nothing, but "x" alone might).
	old := c.buf
	c.buf = nil
	fresh := c.feedFresh(k, now)
	fresh.Flush = append(append([]Key{}, old...), fresh.Flush...)
	return fresh
}

func (c *ChordMatcher) exactMatch(seq []Key) (name string, ok bool) {
	for _, d := range c.defs {
		if len(d.Keys) == len(seq) && keysMatch(d.Keys, seq) {
			return d.Name, true
		}
	}
	return "", false
}

func (c *ChordMatcher) hasPrefixMatch(seq []Key) bool {
	for _, d := range c.defs {
		if len(d.Keys) > len(seq) && keysMatch(d.Keys[:len(seq)], seq) {
			return true
		}
	}
	return false
}

func keysMatch(want []string, got []Key) bool {
	for i, k := range got {
		if k.String() != want[i] {
			return false
		}
	}
	return true
}
