package main

import (
	"fmt"
	"strings"
	"time"
)

// focusLog records each focus report with a timestamp relative to the moment
// focus reporting was enabled, so a person can see whether the terminal sends
// an initial focus-in right after the mode is turned on.
type focusLog struct {
	now       func() time.Time // injectable clock
	enabledAt time.Time
	entries   []focusEntry
}

type focusEntry struct {
	at      time.Time
	focused bool
}

func newFocusLog(now func() time.Time) focusLog {
	if now == nil {
		now = time.Now
	}
	return focusLog{now: now}
}

// markEnabled stamps the time focus reporting is switched on.
func (l *focusLog) markEnabled() { l.enabledAt = l.now() }

func (l *focusLog) record(focused bool) {
	l.entries = append(l.entries, focusEntry{at: l.now(), focused: focused})
}

// lines formats the enable time and every event: a wall-clock timestamp and the
// offset from the enable time.
func (l focusLog) lines() []string {
	const stamp = "15:04:05.000"
	if l.enabledAt.IsZero() {
		return []string{"focus reporting enabled: (not yet)"}
	}
	out := []string{"focus reporting enabled at " + l.enabledAt.Format(stamp)}
	for i, e := range l.entries {
		state := "out"
		if e.focused {
			state = "in"
		}
		out = append(out, fmt.Sprintf("#%d %s focus-%s (+%dms)", i+1, e.at.Format(stamp), state, e.at.Sub(l.enabledAt).Milliseconds()))
	}
	return out
}

func (l focusLog) String() string { return strings.Join(l.lines(), "; ") }
