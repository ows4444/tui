package tui

import (
	"io"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/announce"
	"github.com/ows4444/tui/internal/render"
)

// render diffs the new View() against the last painted frame line-by-line
// and repaints only the rows that changed. The live region is addressed
// with RELATIVE cursor movement — CursorUp back to the top of the region
// last drawn, then a top-to-bottom sweep — rather than absolute
// ansi.CursorPosition ("row 1 of the terminal"). Absolute addressing broke
// as soon as anything (a Println) could scroll real terminal history: "row
// 1" no longer meant "the top of the live region" once prior output had
// pushed it down. Relative addressing keeps working regardless of how much
// has scrolled, at the cost of needing p.liveLines to know how far up "the
// top of the region" currently is.
//
// The line-diff optimization is preserved: unchanged rows still skip
// ClearLine/content entirely, only advancing the cursor down past them via
// "\r\n" to keep the sweep positioned for the next row.
func (p *Program) render() {
	p.publishKeymap()
	// Any render satisfies a throttled one still waiting.
	p.stopFlushTimer()
	if p.accessible {
		p.renderAccessible()
		return
	}
	start := p.frameClock()
	pre := p.unparkCursor()
	body := p.renderBody()
	// Wrapped in synchronized-output mode (CSI ?2026) so the terminal
	// paints the whole diffed frame atomically instead of character by
	// character — a terminal that doesn't support mode 2026 just ignores
	// both sequences, falling back to the previous unwrapped write.
	// The frame is assembled in a buffer the Program keeps, so a frame costs
	// no allocation beyond the body itself.
	frame := append(p.frameBuf[:0], p.syncEnable()...)
	frame = append(frame, pre...)
	frame = append(frame, body...)
	frame = append(frame, p.parkCursor()...)
	frame = append(frame, p.syncDisable()...)
	p.frameBuf = frame
	p.writeBytes(frame)
	p.logFrame(len(frame), start)
}

// downgradeString is ansi.DowngradeString; a variable so a test can prove it
// is not called when it could not change anything.
var downgradeString = ansi.DowngradeString

// transformView applies the whole-view transforms that can change a frame:
// colour downgrade for the profile and the styled-underline gate. Each is
// skipped when it provably cannot change s: no downgrade at TrueColor or when
// s has no CSI (so no SGR), no underline gate without a ':' in s.
func (p *Program) transformView(s string) string {
	if p.colorProfile < ansi.TrueColor && strings.Contains(s, "\x1b[") {
		s = downgradeString(s, p.colorProfile)
	}
	return p.gateUnderlines(s)
}

// renderBody builds the diffed-frame bytes render() writes (and, prefixed
// with repaintFresh's own clear sequence, that repaintFresh writes), and
// updates p.lastFrame/p.liveLines as a side effect. Split out from render
// so repaintFresh can wrap its clear-then-repaint as a single synchronized
// write instead of two separate ones (which would let the terminal paint
// the clear before the redraw catches up, the exact tearing mode 2026 is
// meant to prevent).
func (p *Program) renderBody() string {
	if d, ok := p.directDraw(); ok {
		p.peekedView, p.hasPeeked = "", false
		if out, ok := p.renderDirect(d); ok {
			return out
		}
	}
	raw := p.peekedView
	if !p.hasPeeked {
		raw = p.model.View()
	}
	p.peekedView, p.hasPeeked = "", false
	p.lastView, p.haveView = raw, true
	view := p.transformView(p.withAnnounceRegion(p.withInspector(raw)))
	lines := p.splitView(view)
	if p.bidi {
		m := p.Measurer()
		for i, l := range lines {
			lines[i] = render.BidiLine(l, m)
		}
	}
	// The cell renderer can often prove a line still fits (it is the previous
	// frame's line with a few characters replaced), which is cheaper than
	// measuring it, so it fits lines itself when nothing can overflow into
	// scrollback and its previous frame was drawn at this width.
	lazyFit := p.cellRender && p.cells != nil && p.cells.Valid() && p.cells.Width() == p.width && p.width > 0 &&
		(p.altScreen || p.height <= 0 || len(lines)-p.height <= p.committed)
	if lazyFit {
		lines = p.fitRows(lines)
	} else {
		lines = p.fit(lines)
	}
	newLines := lines

	// Inline view taller than the terminal: rows that no longer fit are
	// committed to scrollback once, and only the visible tail is diffed.
	var commit string
	newLines, commit = p.splitOverflow(newLines)

	max := len(newLines)
	if len(p.lastFrame) > max {
		max = len(p.lastFrame)
	}

	in := frameInput{
		lines: newLines, rows: max, lazyFit: lazyFit,
		width: p.width, height: p.height, prev: p.lastFrame, commit: commit,
		regionTop: p.toRegionTop, fitLine: p.fitLine, measurer: p.Measurer(),
	}
	var (
		out      string
		st       frameStats
		fallback string
	)
	switch {
	case commit != "":
		// The live region is redrawn below the commit, so the cell grid of the
		// previous frame no longer matches the screen.
		if p.cells != nil {
			cellFrames{p.cells}.reset()
		}
		in.prev = nil
		out, st, _ = lineFrames{}.draw(in)
	case p.cellRender:
		if p.cells == nil {
			p.cells = render.New()
		}
		cr := cellFrames{p.cells}
		var ok bool
		if out, st, ok = cr.draw(in); !ok {
			fallback = cr.reason()
			if lazyFit {
				in.lines = p.fit(in.lines) // the line renderer needs every line fitted
			}
			out, st, _ = lineFrames{}.draw(in)
		}
	default:
		out, st, _ = lineFrames{}.draw(in)
	}
	st.fallback = fallback
	if commit != "" {
		st.full = true
	}

	// padded becomes the new p.lastFrame: it's kept at length max (not
	// len(newLines)) so that a shrinking view still leaves the
	// now-blanked trailing rows tracked as part of the live region —
	// otherwise the next render wouldn't know it still needs to move the
	// cursor up past them.
	// The two frame slices alternate (the old lastFrame is dead once draw has
	// returned), so a steady-state frame allocates neither.
	padded := p.frameSpare
	if cap(padded) >= max {
		padded = padded[:max]
		clear(padded)
	} else {
		padded = make([]string, max)
	}
	copy(padded, in.lines)
	p.frameSpare = p.lastFrame
	p.lastFrame = padded
	p.liveLines = max
	p.frameStats = st
	return out
}

// splitView is strings.Split(view, "\n") into a slice the Program reuses.
func (p *Program) splitView(view string) []string {
	lines := p.splitBuf[:0]
	for {
		i := strings.IndexByte(view, '\n')
		if i < 0 {
			break
		}
		lines = append(lines, view[:i])
		view = view[i+1:]
	}
	lines = append(lines, view)
	p.splitBuf = lines
	return lines
}

// splitOverflow implements the inline tall-view rule. It returns the live
// tail of lines (at most p.height rows) and, when rows newly overflowed, the
// bytes that commit them: cursor to the region top, erase down, then the rows
// each ended by CRLF so the terminal scrolls them into history. Rows are
// committed once (p.committed counts them); they are no longer live-updatable.
// It returns lines unchanged when nothing overflows.
func (p *Program) splitOverflow(lines []string) ([]string, string) {
	if p.altScreen || p.height <= 0 {
		return lines, ""
	}
	n := len(lines)
	if p.committed > n {
		p.committed = n
	}
	target := n - p.height
	if target <= p.committed {
		return lines[p.committed:], ""
	}
	var b strings.Builder
	if p.liveLines > 0 {
		b.WriteString(p.toRegionTop())
		b.WriteString(ansi.EraseDown)
	}
	for _, l := range lines[p.committed:target] {
		b.WriteString(l)
		b.WriteString("\r\n")
	}
	p.committed = target
	return lines[target:], b.String()
}

// toRegionTop returns the sequence that moves the cursor from where
// render() leaves it — on the LAST row of the live region, with no trailing
// newline — to column 0 of the region's first row: liveLines-1 rows up, not
// liveLines (which lands one row above the region and overwrites whatever
// is there, e.g. the shell prompt). CursorUp(0) still moves a row, so it's
// omitted, and the distance never exceeds what the screen can scroll back.
func (p *Program) toRegionTop() string {
	up := p.liveLines - 1
	if p.height > 0 && up > p.height-1 {
		up = p.height - 1
	}
	if up > 0 {
		if p.topUp != up || p.topSeq == "" {
			p.topUp, p.topSeq = up, ansi.CursorUp(up)+"\r"
		}
		return p.topSeq
	}
	return "\r"
}

// toReflowedTop is toRegionTop for a region the terminal has re-wrapped: a
// terminal that reflows on resize shows each committed row over
// ceil(width/p.width) physical rows, so the region is taller than liveLines
// and the cursor must climb that many rows to reach its real top. The width
// of every row is taken from lastFrame. Without a known width (or row
// widths) it is toRegionTop.
func (p *Program) toReflowedTop() string {
	if p.width <= 0 || len(p.lastFrame) == 0 {
		return p.toRegionTop()
	}
	rows := 0
	for _, l := range p.lastFrame {
		rows += max(1, (p.Measurer().Width(l)+p.width-1)/p.width)
	}
	up := rows - 1
	if p.height > 0 && up > p.height-1 {
		up = p.height - 1
	}
	if up > 0 {
		if p.topUp != up || p.topSeq == "" {
			p.topUp, p.topSeq = up, ansi.CursorUp(up)+"\r"
		}
		return p.topSeq
	}
	return "\r"
}

// cursorCell is a screen cell a CursorPlacer asked for; ok false means none.
type cursorCell struct {
	x, y int
	ok   bool
}

// unparkCursor returns the sequence that undoes parkCursor: it hides the
// hardware cursor and moves it back to the last live row, where render and
// printLines expect it. It is empty when no cursor was parked, so a Model
// without a CursorPlacer writes exactly what it did before.
func (p *Program) unparkCursor() string {
	var s string
	if p.curShown {
		s = ansi.CursorHide
		p.curShown = false
	}
	if p.curUp > 0 {
		s += ansi.CursorDown(p.curUp)
		p.curUp = 0
	}
	return s
}

// cursorSource returns the model's cursor query: its CursorPlacer, else its
// CursorProvider, searching the Unwrap chain for each in turn.
func cursorSource(m Model) (func() (int, int, bool), bool) {
	if cp, ok := find[CursorPlacer](m); ok {
		return cp.CursorPos, true
	}
	if cp, ok := find[CursorProvider](m); ok {
		return cp.CursorCell, true
	}
	return nil, false
}

// cursorTarget asks the Model where the hardware cursor should sit, clamped
// to the live region as it stands now.
func (p *Program) cursorTarget() cursorCell {
	pos, ok := cursorSource(p.model)
	if !ok || p.accessible || p.liveLines == 0 {
		return cursorCell{}
	}
	x, y, ok := pos()
	if !ok {
		return cursorCell{}
	}
	y -= p.committed // rows already in scrollback are not part of the region
	x = max(x, 0)
	if p.width > 0 {
		x = min(x, p.width-1)
	}
	return cursorCell{x: x, y: min(max(y, 0), p.liveLines-1), ok: true}
}

// parkCursor returns the sequence that, from the last live row, moves the
// cursor to the cell the Model asked for and shows it, using relative
// movement like the rest of render. It is empty when the Model asked for
// none. The cursor then rests above the last row, which unparkCursor undoes.
func (p *Program) parkCursor() string {
	p.curWant = p.cursorTarget()
	if !p.curWant.ok {
		return ""
	}
	up := p.liveLines - 1 - p.curWant.y
	s := "\r"
	if up > 0 {
		s = ansi.CursorUp(up) + s
	}
	if p.curWant.x > 0 {
		s += ansi.CursorForward(p.curWant.x)
	}
	p.curUp, p.curShown = up, true
	return s + ansi.CursorShow
}

// followCursor moves the hardware cursor when the Model's answer changed but
// the View did not, so no frame is being drawn.
func (p *Program) followCursor() {
	if _, ok := cursorSource(p.model); !ok {
		return
	}
	if want := p.cursorTarget(); want == p.curWant {
		return
	}
	p.write(p.syncEnable() + p.unparkCursor() + p.parkCursor() + p.syncDisable())
}

// fit clips lines to the terminal so that what render() counts in
// p.liveLines is what the terminal actually draws. A line wider than the
// terminal wraps onto extra physical rows, and (on the alternate screen) a
// view taller than the terminal scrolls it; either desynchronizes the
// relative cursor addressing render() relies on, and a resize is exactly
// when a view stops fitting. Inline mode keeps its full height — rows that
// scroll away there belong to the terminal's scrollback. An unknown size
// (0) clips nothing.
func (p *Program) fit(lines []string) []string {
	lines = p.fitRows(lines)
	for i, l := range lines {
		lines[i] = p.fitLine(l)
	}
	return lines
}

// fitRows is the part of fit that drops the rows an alternate-screen view has
// beyond the terminal's height.
func (p *Program) fitRows(lines []string) []string {
	if p.altScreen && p.height > 0 && len(lines) > p.height {
		lines = lines[:p.height]
	}
	return lines
}

// fitLine is the part of fit that applies to one line: tabs are expanded and a
// line wider than the terminal is cut to it.
func (p *Program) fitLine(l string) string {
	if strings.IndexByte(l, '\t') >= 0 {
		l = ansi.ExpandTabs(l)
	}
	if m := p.Measurer(); p.width > 0 && m.Width(l) > p.width {
		l = m.Truncate(l, p.width)
	}
	return l
}

// repaintFresh repaints the live region from scratch, without diffing
// against what render() last painted. After a resize the terminal has
// cropped, re-wrapped or scrolled what was on screen, so lastFrame no
// longer describes it: a diff would skip every row whose text is
// unchanged and leave those rows blank or garbled. The alternate screen
// is simply cleared. Inline, the region is erased from its top (moving up
// no further than the screen allows) and drawn again below that point.
func (p *Program) repaintFresh() {
	if p.accessible {
		return // append-only output: there is nothing to repaint on resize
	}
	pre := p.unparkCursor()
	var clear string
	if p.altScreen {
		clear = ansi.CursorHome + ansi.ClearScreen
	} else if p.liveLines > 0 {
		clear = p.toReflowedTop() + ansi.EraseDown
	}
	p.lastFrame = nil
	p.liveLines = 0
	start := p.frameClock()
	body := p.renderBody()
	// One synchronized write covering both the clear and the redraw — see
	// renderBody's comment on why these can't be two separate wrapped
	// writes without reintroducing the tearing this is meant to prevent.
	frame := p.syncEnable() + pre + clear + body + p.parkCursor() + p.syncDisable()
	p.write(frame)
	p.logFrame(len(frame), start)
}

// printLines commits text to the terminal's real, permanent scrollback,
// above the live region: it moves the cursor up to the top of the
// currently-drawn live region (a no-op if render() has never run),
// erases everything below that point with ansi.EraseDown so no stale
// live-region content is left behind if text is shorter than the region
// it displaces, then writes text terminated with real "\r\n" line by
// line so the terminal's own scrolling absorbs it into history rather
// than it being treated as a live-region repaint. It resets
// p.lastFrame/p.liveLines so the render() that follows (see runLoop's
// printMsg case) repaints the live region fresh, from scratch, at its
// new position below the committed text.
//
// dest is where the commit is written — p.output for Println, p.errOutput
// for Eprintln — but the cursor-up/erase math above is always computed
// from p.liveLines, the STDOUT live region's size, regardless of dest:
// terminal cursor position belongs to the terminal, not to whichever file
// descriptor happens to write to it, so this only produces a correct
// result when dest shares a terminal with p.output (the ordinary case,
// and the one Eprintln exists for).
// maxAltPrints bounds how many Println/Eprintln calls are held while the
// alternate screen is up; beyond it the oldest are dropped.
const maxAltPrints = 10000

// holdAltPrint keeps a Println made on the alternate screen until it is left.
func (p *Program) holdAltPrint(pm printMsg) {
	if len(p.altPrints) >= maxAltPrints {
		p.altPrints = append(p.altPrints[:0], p.altPrints[1:]...)
	}
	p.altPrints = append(p.altPrints, pm)
}

// flushAltPrints writes the held prints to their streams, in order, on the
// main screen. Call it right after the alternate screen has been left.
func (p *Program) flushAltPrints() {
	held := p.altPrints
	p.altPrints = nil
	if len(held) == 0 {
		return
	}
	p.lastFrame, p.liveLines = nil, 0
	for _, pm := range held {
		dest := p.output
		if pm.stderr {
			dest = p.errOutput
		}
		p.printLines(pm.text, dest)
	}
}

func (p *Program) printLines(text string, dest io.Writer) {
	text = ansi.DowngradeString(text, p.colorProfile)
	var buf strings.Builder
	buf.WriteString(p.unparkCursor())
	if p.liveLines > 0 {
		buf.WriteString(p.toRegionTop())
		buf.WriteString(ansi.EraseDown)
	}
	for _, line := range strings.Split(text, "\n") {
		buf.WriteString(line)
		buf.WriteString("\r\n")
	}

	if !p.accessible {
		p.lastFrame = nil
		p.liveLines = 0
	}
	p.writeTo(dest, buf.String())
}

// stripOSC removes OSC strings, keeping the text between them; see announce.StripOSC.
func stripOSC(s string) string { return announce.StripOSC(s) }
