// Package logview is an append-only scrolling log view. It composes
// viewport.Model rather than reimplementing scrolling: Log's own job is
// just appending lines and auto-scrolling to bottom on append when the
// reader is already at the bottom.
package logview

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
	"github.com/ows4444/tui/viewport"
)

// Model is an append-only log built on top of viewport.Model.
type Model struct {
	Viewport viewport.Model

	// Max, if positive, caps the number of lines kept: an Append that would
	// exceed it drops the oldest lines. Zero (the default) means unbounded.
	Max int

	// NewLinesIndicator, when true, makes View show an "N new lines" row
	// over the last row while the reader is scrolled up and lines arrived.
	// Off by default so View is exactly the viewport's rows; NewLines is
	// always tracked.
	NewLinesIndicator bool

	// Follow, when true, pins the view to the newest line: every Append
	// scrolls to the bottom, even if the Viewport was moved off it by code.
	// Scrolling away with the keys or mouse wheel (Update), or jumping to a
	// search match, turns it off, like pressing a key in `tail -f`; set it
	// again, or call SetFollow, to resume. With Follow false an Append still
	// stays pinned while the reader is already at the bottom.
	Follow bool

	unseen int // lines appended while scrolled up, not yet seen
}

// SetFollow turns follow-tail mode on or off. Turning it on scrolls to the
// bottom and clears NewLines.
func (m *Model) SetFollow(on bool) {
	m.Follow = on
	if on {
		m.Viewport.GotoBottom()
		m.unseen = 0
	}
}

// Search highlights every match of q and jumps to the first one at or below the
// top row, returning the match count; see viewport.Model.Search. Jumping to a
// match leaves follow-tail mode unless the match is on screen at the bottom.
func (m *Model) Search(q string) int {
	n := m.Viewport.Search(q)
	m.leaveFollowIfScrolled()
	return n
}

// NextMatch and PrevMatch move between search matches (also bound to n and N
// in Update while a query is set).
func (m *Model) NextMatch() bool {
	ok := m.Viewport.NextMatch()
	m.leaveFollowIfScrolled()
	return ok
}

// PrevMatch moves to the previous search match.
func (m *Model) PrevMatch() bool {
	ok := m.Viewport.PrevMatch()
	m.leaveFollowIfScrolled()
	return ok
}

func (m *Model) leaveFollowIfScrolled() {
	if m.Follow && !m.Viewport.AtBottom() {
		m.Follow = false
	}
}

// NewLines returns how many lines were appended while the reader was
// scrolled up. It resets to 0 once the view is back at the bottom (End).
func (m Model) NewLines() int { return m.unseen }

// New returns a Model sized to width x height.
func New(width, height int) Model {
	return Model{Viewport: viewport.New(width, height)}
}

// Append adds line to the log. If the viewport was scrolled to the bottom
// before the append, it stays pinned to the bottom so the new line is
// visible. If the reader had scrolled up, the scroll position is left
// unchanged (and, when Max drops lines above it, keeps showing the same
// content).
func (m *Model) Append(line string) {
	atBottom := m.Follow || m.Viewport.AtBottom()
	m.Viewport.AppendLine(line) // incremental: SetContent(join(lines)) made n appends O(n^2)
	if m.Max > 0 {
		m.Viewport.TrimFront(m.Max)
	}
	if atBottom {
		m.Viewport.GotoBottom()
		m.unseen = 0
	} else {
		m.unseen++
	}
}

// Update delegates to the embedded viewport.Model's Update.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	var cmd tui.Cmd
	m.Viewport, cmd = m.Viewport.Update(msg)
	m.leaveFollowIfScrolled()
	if m.Viewport.AtBottom() {
		m.unseen = 0
	}
	return m, cmd
}

// View renders the viewport. While NewLines is positive the last row shows
// an "N new lines" indicator in place of that row's content.
func (m Model) View() string {
	v := m.Viewport.View()
	if !m.NewLinesIndicator || m.unseen <= 0 || m.Viewport.Height <= 0 {
		return v
	}
	rows := strings.Split(v, "\n")
	word := "lines"
	if m.unseen == 1 {
		word = "line"
	}
	ind := fmt.Sprintf("%d new %s", m.unseen, word)
	rows[len(rows)-1] = ansi.Truncate(ind, m.Viewport.Width)
	return strings.Join(rows, "\n")
}

// DrawCells draws the log into r of buf as View shows it: the viewport's rows,
// with the "N new lines" indicator replacing the last row when it is on. It
// implements tui.CellDrawer.
func (m Model) DrawCells(buf *cellbuf.Buffer, r cellbuf.Rect) {
	m.Viewport.DrawCells(buf, r)
	if !m.NewLinesIndicator || m.unseen <= 0 || m.Viewport.Height <= 0 || m.Viewport.Width <= 0 {
		return
	}
	r.W, r.H = min(r.W, m.Viewport.Width), min(r.H, m.Viewport.Height)
	if r.W <= 0 || r.H <= 0 {
		return
	}
	row := buf.Sub(cellbuf.Rect{X: r.X, Y: r.Y + r.H - 1, W: r.W, H: 1})
	row.Fill(row.Bounds(), " ", 0)
	word := "lines"
	if m.unseen == 1 {
		word = "line"
	}
	row.SetString(0, 0, fmt.Sprintf("%d new %s", m.unseen, word), 0)
}

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}
