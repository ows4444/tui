package tui

import (
	"time"
)

// chordExpireMsg fires when a pending chord prefix's timeout passes, so its
// keys can be flushed without waiting for another keypress. It is internal:
// never forwarded to Update.
type chordExpireMsg struct{}

// handleChordMsg routes Keys (and the internal expiry message) through the
// chord matcher. It reports true when it consumed msg, having already
// delivered anything Update should see and repainted; false means msg is
// not chord-related and takes the normal path.
func (p *Program) handleChordMsg(msg Msg, done <-chan struct{}) bool {
	var deliver []Msg
	switch m := msg.(type) {
	case Key:
		res := p.chords.Feed(m, time.Now())
		for _, k := range res.Flush {
			deliver = append(deliver, k)
		}
		if res.Msg != nil {
			deliver = append(deliver, *res.Msg)
		}
		if res.Pending {
			p.armChordExpiry(done)
		}
	case chordExpireMsg:
		for _, k := range p.chords.Expire(time.Now()) {
			deliver = append(deliver, k)
		}
	default:
		return false
	}
	for _, d := range deliver {
		var cmd Cmd
		p.model, cmd = p.model.Update(d)
		if cmd != nil {
			p.spawn(cmd, done)
		}
	}
	if len(deliver) > 0 && p.allowRenderFor(done, true) {
		p.render()
	}
	return true
}

// armChordExpiry (re)starts the timer that sends chordExpireMsg once the
// pending prefix has waited out the chord timeout.
func (p *Program) armChordExpiry(done <-chan struct{}) {
	// A little past the timeout: ChordMatcher.Expire treats "exactly the
	// timeout" as not yet expired.
	d := p.chords.Timeout + 10*time.Millisecond
	if p.chordTimer != nil {
		p.chordTimer.Stop()
	}
	p.chordTimer = time.AfterFunc(d, func() {
		select {
		case p.msgs <- chordExpireMsg{}:
		case <-done:
		}
	})
}
