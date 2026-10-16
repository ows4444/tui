package tui

import "testing"

type byteCount struct{ n int }

func (c *byteCount) Write(p []byte) (int, error) { c.n += len(p); return len(p), nil }

// The cell renderer is the renderer a program uses, and the line renderer is
// the fallback it falls back to: this is the measurement that keeps that
// choice honest. On a frame whose values change every time, the cell renderer
// writes a fraction of the bytes (about a tenth on this 80x24 frame), which is
// what matters over a slow link; the line renderer uses a little less CPU
// (docs/rendering.md has the figures).
func TestCellRendererWritesFarFewerBytesThanTheLineRenderer(t *testing.T) {
	perFrame := func(cell bool) int {
		w := &byteCount{}
		p := NewProgram(frameModel{}, WithOutput(w), WithCellRenderer(cell))
		p.width, p.height = 80, 24
		p.render()
		w.n = 0
		const frames = 20
		for i := 1; i <= frames; i++ {
			p.model = frameModel{n: i}
			p.render()
		}
		return w.n / frames
	}
	line, cell := perFrame(false), perFrame(true)
	if cell*4 > line {
		t.Errorf("cell renderer wrote %d bytes per changing frame, line renderer %d: "+
			"expected the cell renderer to write under a quarter", cell, line)
	}
}
