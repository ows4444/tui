package tui

import "time"

// defaultMultiClick is the interval within which presses make a double click.
const defaultMultiClick = 400 * time.Millisecond

// clickCounter remembers the last counted press.
type clickCounter struct {
	at     time.Time
	x, y   int
	button MouseButton
	count  int
}

// count1 returns the click count for ev, and updates the counter. window <= 0
// turns counting off. A press within window of the last press of the same
// button on the same cell is the next click; a release or drag of that button
// repeats the count; anything else is 0.
func (c *clickCounter) count1(ev MouseEvent, now time.Time, window time.Duration) int {
	if window <= 0 {
		return 0
	}
	switch ev.Button {
	case MouseButtonLeft, MouseButtonMiddle, MouseButtonRight:
	default:
		return 0
	}
	if ev.Action != MouseActionPress {
		if c.count > 0 && ev.Button == c.button {
			return c.count
		}
		return 0
	}
	if c.count > 0 && ev.Button == c.button && ev.X == c.x && ev.Y == c.y && now.Sub(c.at) <= window {
		c.count++
		if c.count > 3 {
			c.count = 1
		}
	} else {
		c.count = 1
	}
	c.at, c.x, c.y, c.button = now, ev.X, ev.Y, ev.Button
	return c.count
}

// countClicks fills in MouseEvent.Clicks. It runs on the event loop.
func (p *Program) countClicks(ev MouseEvent) MouseEvent {
	window := defaultMultiClick
	if p.multiClickSet {
		window = p.multiClick
	}
	ev.Clicks = p.clicks.count1(ev, time.Now(), window)
	return ev
}
