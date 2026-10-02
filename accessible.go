package tui

import (
	"os"
	"strings"
	"time"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/announce"
)

// announceMsg is a renderer instruction produced by Announce; the event loop
// intercepts it and never hands it to Update.
type announceMsg struct {
	text  string
	level AnnounceLevel
}

// AnnounceLevel is how urgently an announcement is spoken.
type AnnounceLevel int

const (
	// Polite announcements are de-duplicated: the same text sent again within
	// one second is written once.
	Polite AnnounceLevel = iota
	// Assertive announcements always write, interrupting any de-duplication.
	Assertive
)

// AnnounceWith is Announce with an explicit politeness level.
func AnnounceWith(text string, level AnnounceLevel) Cmd {
	return func() Msg { return announceMsg{text: text, level: level} }
}

// Announce returns a Cmd that, in accessible mode, writes text to the
// transcript as its own line, so a screen reader speaks it once. Outside
// accessible mode it does nothing: the visible UI is expected to carry the
// same information.
func Announce(text string) Cmd {
	return AnnounceWith(text, Polite)
}

// WithAccessibleAuto turns accessible mode on when the environment asks for
// it: ACCESSIBLE=1 or TERM=dumb. Otherwise it leaves accessible mode off.
// Like every option, the last of WithAccessible / WithAccessibleAuto wins.
func WithAccessibleAuto() ProgramOption {
	return func(p *Program) { p.accessible = accessibleFromEnv() }
}

func accessibleFromEnv() bool {
	return os.Getenv("ACCESSIBLE") == "1" || os.Getenv("TERM") == "dumb"
}

// AnnounceTTL is how long WithAnnounceRegion keeps an announcement visible.
const AnnounceTTL = announce.TTL

// alertPrefix starts every Assertive transcript line in accessible mode.
const alertPrefix = announce.AlertPrefix

// announceExpireMsg is a renderer instruction: an announcement's time is up.
type announceExpireMsg struct{}

// WithAnnounceRegion reserves rows rows below the view (inline: appended
// below it; alternate screen: the view is cut to leave them) in which the most
// recent announcements made with Announce or AnnounceWith are shown for
// AnnounceTTL each, outside accessible mode. The region is blank when there is
// nothing to show, and is redrawn once an announcement expires. The model's
// View is not changed. rows <= 0 turns it off (the default); accessible mode
// ignores it, since announcements there go to the transcript.
//
// When a capability probe (WithCapabilityProbe) reports OSC 99 support
// (Capabilities.Notifications), Assertive announcements are also sent as a
// desktop notification, whether or not a region is configured.
func WithAnnounceRegion(rows int) ProgramOption {
	return func(p *Program) { p.announce.Rows = max(rows, 0) }
}

// withAnnounceRegion composes the announcement rows under view.
func (p *Program) withAnnounceRegion(view string) string {
	if p.accessible {
		return view
	}
	return p.announce.Compose(view, p.altScreen, p.height)
}

// showAnnouncement adds text to the region and arms the expiry timer.
func (p *Program) showAnnouncement(text string) {
	p.announce.Show(text, func() {
		select {
		case p.msgs <- announceExpireMsg{}:
		case <-p.announce.Done:
		}
	})
}

// notificationsOK reports whether a finished probe found OSC 99 support.
func (p *Program) notificationsOK() bool {
	if !p.capProbe.on {
		return false
	}
	c, known := p.capProbe.result()
	return known && c.Notifications
}

// flushPolite writes the queued polite lines. "Pending" means queued by
// handleAnnounce and not yet written: polite lines wait in the queue while
// more messages are already waiting in the event queue, and are written
// before the next non-announcement message is handled or once the queue of
// waiting messages is empty. An Assertive announcement arriving in between
// discards them.
func (p *Program) flushPolite() {
	for _, l := range p.polite.Take() {
		p.printLines(l, p.output)
	}
}

// handleAnnounce handles announcements and their expiry; it reports whether
// msg was one. Any other message first flushes the pending polite lines.
func (p *Program) handleAnnounce(msg Msg) bool {
	if _, ok := msg.(announceExpireMsg); ok {
		if !p.accessible && p.announce.Rows > 0 && p.announce.Prune() {
			p.render()
		}
		return true
	}
	am, ok := msg.(announceMsg)
	if !ok {
		p.flushPolite()
		return false
	}
	text := stripEscapes(am.text)
	if p.accessible {
		if am.level == Polite {
			if p.polite.Polite(am.text, text, time.Now()) && len(p.msgs) == 0 {
				p.flushPolite()
			}
		} else {
			p.polite.Assertive()
			p.printLines(alertPrefix+text, p.output)
		}
		return true
	}
	if am.level == Assertive && p.notificationsOK() {
		p.write(p.announce.Notify(am.text))
	}
	if p.announce.Rows > 0 {
		p.showAnnouncement(text)
		p.render()
	}
	return true
}

// stripEscapes removes OSC strings and CSI/SGR sequences from s.
func stripEscapes(s string) string { return announce.Strip(s) }

// WithLinearizeFullLine makes accessible mode write a transcript line in full
// again, as a new line, when it grows, instead of only the appended suffix.
// Default: only the suffix.
func WithLinearizeFullLine() ProgramOption {
	return func(p *Program) { p.fullLine = true }
}

// renderAccessible is render() for accessible mode: it writes, in order,
// only the lines that differ from the previous frame (by row index), each
// followed by CRLF, with no cursor movement, clearing, styling or
// synchronized-output wrapper. Rows that vanish are not erased; the output
// is an append-only transcript. p.liveLines stays 0 so nothing else tries
// to move the cursor back over it.
func (p *Program) renderAccessible() {
	start := p.frameClock()
	var text string
	if l, ok := find[Linearizer](p.model); ok {
		text = l.Linearize()
	} else {
		text = p.model.View()
	}
	text = stripOSC(ansi.StripANSI(stripOSC(text)))
	lines := strings.Split(text, "\n")
	if text == "" {
		lines = nil
	}

	// Item identity: only when the new frame is a shifted window of the
	// previous one (see isShiftedWindow) is a non-empty line that was in the
	// previous frame at a different row treated as the same item and not
	// announced again. Rows equal at the same index are matched first so a
	// duplicated line is counted per occurrence. Otherwise every changed row
	// is announced.
	var prevCount map[string]int
	if isShiftedWindow(p.lastFrame, lines) {
		prevCount = make(map[string]int, len(p.lastFrame))
		for _, l := range p.lastFrame {
			if l != "" {
				prevCount[l]++
			}
		}
		for i, line := range lines {
			if line != "" && i < len(p.lastFrame) && p.lastFrame[i] == line {
				prevCount[line]--
			}
		}
	}

	var buf strings.Builder
	written := 0
	for i, line := range lines {
		if i < len(p.lastFrame) && p.lastFrame[i] == line {
			continue
		}
		if line != "" && prevCount != nil && prevCount[line] > 0 {
			prevCount[line]--
			continue
		}
		if i < len(p.lastFrame) && p.lastFrame[i] != "" && strings.HasPrefix(line, p.lastFrame[i]) {
			// A transcript line grew (wherever it sits): write only the new
			// suffix, unless WithLinearizeFullLine asks for the whole line.
			if !p.fullLine {
				line = line[len(p.lastFrame[i]):]
			}
		}
		buf.WriteString(line)
		buf.WriteString("\r\n")
		written++
	}
	p.lastFrame = lines
	p.write(buf.String())
	p.frameStats = frameStats{rows: len(lines), changed: written, accessible: true}
	p.logFrame(buf.Len(), start)
}

// isShiftedWindow reports whether next is prev scrolled by a constant,
// non-zero row offset s (next[i] == prev[i+s]). The rule is deliberately
// conservative: every row where both frames have a row under that offset must
// match exactly, at least one such row must be non-empty, and the overlap must
// be at least two rows (or, for frames of two or three rows, all but one row
// of the shorter frame). The smallest |s| that qualifies is used. When no
// offset qualifies the caller announces every changed row, so genuinely new
// content that merely repeats text elsewhere in the previous frame is never
// dropped.
func isShiftedWindow(prev, next []string) bool {
	n := len(prev)
	if len(next) > n {
		n = len(next)
	}
	short := len(prev)
	if len(next) < short {
		short = len(next)
	}
	for d := 1; d < n; d++ {
		for _, s := range [2]int{d, -d} {
			overlap, nonEmpty, ok := 0, false, true
			for i, line := range next {
				j := i + s
				if j < 0 || j >= len(prev) {
					continue
				}
				if prev[j] != line {
					ok = false
					break
				}
				overlap++
				if line != "" {
					nonEmpty = true
				}
			}
			if ok && nonEmpty && (overlap >= 2 || overlap >= short-1) {
				return true
			}
		}
	}
	return false
}
