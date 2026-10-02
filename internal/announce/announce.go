// Package announce is the state behind a Program's announcements: the
// on-screen region that shows recent ones for a few seconds, the de-duplicating
// queue that feeds the accessible-mode transcript, and the sanitising and
// OSC 99 notification encoding both share. It owns no goroutine, writes no
// output and does not know about Programs; the runtime decides when to call it
// and where the bytes go, so the rules here can be tested with a fake clock.
package announce

import (
	"strconv"
	"strings"
	"time"

	"github.com/ows4444/tui/ansi"
)

// TTL is how long the region keeps an announcement visible.
const TTL = 3 * time.Second

// PoliteDedupWindow is how long the same polite text is written only once.
const PoliteDedupWindow = time.Second

// AlertPrefix starts every assertive transcript line in accessible mode.
const AlertPrefix = "Alert: "

// Region is the announcement rows below a view. The zero value is off (Rows 0).
// Not safe for concurrent use: the runtime's loop goroutine owns it, except
// Done, Now and After, which are set before the loop starts.
type Region struct {
	Rows  int                             // rows reserved below the view; <= 0 is off
	Done  <-chan struct{}                 // closed when the loop ends; expiry callbacks should stop on it
	Now   func() time.Time                // nil: time.Now
	After func(d time.Duration, f func()) // nil: time.AfterFunc

	items []item
	seq   int // notification ids
}

type item struct {
	text    string
	expires time.Time
}

func (r *Region) clock() time.Time {
	if r.Now != nil {
		return r.Now()
	}
	return time.Now()
}

// Len is the number of announcements currently held.
func (r *Region) Len() int { return len(r.items) }

// Compose appends the region's rows to view: blank when there is nothing to
// show. On the alternate screen with a known height the view is cut to leave
// the rows. With Rows <= 0 it returns view unchanged.
func (r *Region) Compose(view string, altScreen bool, height int) string {
	rows := r.Rows
	if rows <= 0 {
		return view
	}
	lines := strings.Split(view, "\n")
	if altScreen && height > rows && len(lines) > height-rows {
		lines = lines[:height-rows]
	}
	items := r.items
	if len(items) > rows {
		items = items[len(items)-rows:]
	}
	for i := 0; i < rows; i++ {
		if i < len(items) {
			lines = append(lines, items[i].text)
		} else {
			lines = append(lines, "")
		}
	}
	return strings.Join(lines, "\n")
}

// Prune drops expired items and reports whether any were dropped.
func (r *Region) Prune() bool {
	now := r.clock()
	kept := r.items[:0]
	for _, it := range r.items {
		if now.Before(it.expires) {
			kept = append(kept, it)
		}
	}
	changed := len(kept) != len(r.items)
	r.items = kept
	return changed
}

// Show adds text, keeping only the newest Rows items, and arms a timer that
// calls expire a little after TTL, so a following Prune sees the item as
// expired. expire runs on the timer's goroutine.
func (r *Region) Show(text string, expire func()) {
	r.items = append(r.items, item{text: text, expires: r.clock().Add(TTL)})
	if len(r.items) > r.Rows {
		r.items = r.items[len(r.items)-r.Rows:]
	}
	d := TTL + 10*time.Millisecond
	if r.After != nil {
		r.After(d, expire)
	} else {
		time.AfterFunc(d, expire)
	}
}

// Notify returns text as a kitty desktop notification (OSC 99, urgency
// critical) with the next notification id.
func (r *Region) Notify(text string) string {
	r.seq++
	return "\x1b]99;i=" + strconv.Itoa(r.seq) + ":u=2;" + sanitizeNotification(text) + "\x1b\\"
}

// sanitizeNotification strips escapes and control characters from s.
func sanitizeNotification(s string) string {
	s = Strip(s)
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r < 0xa0) {
			return ' '
		}
		return r
	}, s)
}

// Queue is the accessible-mode transcript's announcement queue: it drops a
// polite announcement that repeats within PoliteDedupWindow and holds polite
// lines until the runtime flushes them. The zero value is empty.
type Queue struct {
	last    string
	lastAt  time.Time
	pending []string
}

// Polite offers a polite announcement. raw is the text as sent (what repeats
// are compared on) and line is what would be written (raw without escapes). It
// reports false when it repeats the previous polite one inside the window and
// must not be written. Otherwise line is queued.
func (q *Queue) Polite(raw, line string, now time.Time) bool {
	if q.last == raw && !q.lastAt.IsZero() && now.Sub(q.lastAt) < PoliteDedupWindow {
		return false
	}
	q.last, q.lastAt = raw, now
	q.pending = append(q.pending, line)
	return true
}

// Assertive records an assertive announcement: it resets the de-duplication
// and discards polite lines that were still waiting, which it overrides.
func (q *Queue) Assertive() {
	q.last, q.lastAt = "", time.Time{}
	q.pending = nil
}

// Take returns the queued polite lines, oldest first, and empties the queue.
func (q *Queue) Take() []string {
	lines := q.pending
	q.pending = nil
	return lines
}

// Pending reports whether polite lines are waiting.
func (q *Queue) Pending() bool { return len(q.pending) > 0 }

// Strip removes OSC strings and CSI/SGR sequences from s.
func Strip(s string) string {
	if !strings.Contains(s, "\x1b") {
		return s
	}
	return StripOSC(ansi.StripANSI(StripOSC(s)))
}

// StripOSC removes OSC strings (ESC ] ... terminated by BEL or ESC \), such
// as ansi.Hyperlink's OSC 8 wrappers, keeping the text between them. An
// unterminated OSC is dropped through the end of the string.
func StripOSC(s string) string {
	if !strings.Contains(s, "\x1b]") {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == ']' {
			j := i + 2
			for j < len(s) {
				if s[j] == 0x07 {
					j++
					break
				}
				if s[j] == 0x1b && j+1 < len(s) && s[j+1] == '\\' {
					j += 2
					break
				}
				j++
			}
			i = j
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}
