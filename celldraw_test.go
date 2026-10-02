package tui

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
	"github.com/ows4444/tui/internal/vtscreen"
)

const cdW, cdH = 30, 8

// drawModel is a CellDrawer root whose View must never run.
type drawModel struct {
	draw  func(buf *cellbuf.Buffer, r cellbuf.Rect)
	views int
}

func (m *drawModel) Init() Cmd               { return nil }
func (m *drawModel) Update(Msg) (Model, Cmd) { return m, nil }
func (m *drawModel) View() string            { m.views++; return "VIEW-STRING-SHOULD-NOT-BE-BUILT" }
func (m *drawModel) DrawCells(buf *cellbuf.Buffer, r cellbuf.Rect) {
	m.draw(buf, r)
}

// stringChild is a Model with only a View.
type stringChild struct{ s string }

func (c stringChild) Init() Cmd               { return nil }
func (c stringChild) Update(Msg) (Model, Cmd) { return c, nil }
func (c stringChild) View() string            { return c.s }

// cellChild is a CellDrawer child that draws the same picture as the string
// "\x1b[1;31mhi\x1b[0m 世界" on its first row.
type cellChild struct{}

func (cellChild) Init() Cmd               { return nil }
func (cellChild) Update(Msg) (Model, Cmd) { return cellChild{}, nil }
func (cellChild) View() string            { return "\x1b[1;31mhi\x1b[0m 世界\n\x1b[44mbg\x1b[0m" }
func (cellChild) DrawCells(buf *cellbuf.Buffer, r cellbuf.Rect) {
	b := buf.Sub(r)
	st := b.StyleID(cellbuf.Style{Attrs: cellbuf.AttrBold, FG: cellbuf.Basic(1)})
	b.SetString(0, 0, "hi", st)
	b.SetString(2, 0, " 世界", 0)
	b.SetString(0, 1, "bg", b.StyleID(cellbuf.Style{BG: cellbuf.Basic(4)}))
}

func cdProgram(m Model, out *bytes.Buffer) *Program {
	p := NewProgram(m, WithOutput(out), WithCellRenderer(true), WithAltScreen(false))
	p.width, p.height = cdW, cdH
	p.colorProfile = ansi.TrueColor
	return p
}

func cdScreen(out []byte) *vtscreen.Screen {
	s := vtscreen.NewScreen(cdW, cdH)
	s.Write(out)
	return s
}

func sameScreens(t *testing.T, a, b *vtscreen.Screen) {
	t.Helper()
	if !reflect.DeepEqual(a.Lines(), b.Lines()) {
		t.Fatalf("text differs:\n%q\n%q", a.Lines(), b.Lines())
	}
	for y := 0; y < cdH; y++ {
		for x := 0; x < cdW; x++ {
			if a.Cell(x, y) != b.Cell(x, y) {
				t.Fatalf("cell %d,%d differs: %+v vs %+v", x, y, a.Cell(x, y), b.Cell(x, y))
			}
		}
	}
}

// When the root model implements CellDrawer, the system shall not parse a View
// string for that frame (#43): View is never called, no view is recorded, and
// the screen shows what DrawCells drew.
func TestCellDrawerRootSkipsView(t *testing.T) {
	m := &drawModel{draw: func(buf *cellbuf.Buffer, r cellbuf.Rect) {
		buf.SetString(r.X, r.Y, "hello", 0)
		buf.SetString(r.X+2, r.Y+2, "x", buf.StyleID(cellbuf.Style{FG: cellbuf.RGB(1, 2, 3)}))
	}}
	var out bytes.Buffer
	p := cdProgram(m, &out)
	p.renderIfChanged()
	p.renderIfChanged()
	if m.views != 0 || p.haveView || p.lastView != "" {
		t.Fatalf("View was used: calls=%d haveView=%v", m.views, p.haveView)
	}
	if got := cdScreen(out.Bytes()).Lines(); got[0] != "hello" || got[2] != "  x" {
		t.Fatalf("screen = %q", got)
	}
	if p.frameStats.fallback != "" || len(p.frameStats.rowFallbacks) > 0 {
		t.Fatalf("fell back: %+v", p.frameStats)
	}
}

// When a frame mixes CellDrawer and string children, the system shall produce
// the same screen as the all-string path (#44), across several frames
// including a change of what is drawn.
func TestCellDrawerMixedMatchesStringPath(t *testing.T) {
	const strChild = "\x1b[3;32mabc\x1b[0m\n\x1b]8;;https://x.test\x1b\\link\x1b]8;;\x1b\\ end"
	var body = []string{"title", "second row", "tail"}
	frame := func(n int) (draw func(*cellbuf.Buffer, cellbuf.Rect), str string) {
		body[2] = strings.Repeat("t", 1+n)
		draw = func(buf *cellbuf.Buffer, r cellbuf.Rect) {
			DrawView(buf, cellbuf.Rect{X: r.X, Y: r.Y, W: r.W, H: 1}, body[0])
			DrawChild(buf, cellbuf.Rect{X: 0, Y: 1, W: 10, H: 2}, stringChild{strChild})
			DrawChild(buf, cellbuf.Rect{X: 12, Y: 1, W: 10, H: 2}, cellChild{})
			DrawChild(buf, cellbuf.Rect{X: 0, Y: 4, W: 20, H: 1}, stringChild{body[2]})
		}
		// The all-string equivalent, composed by hand.
		rows := []string{body[0], "", "", "", body[2]}
		sc := strings.Split(strChild, "\n")
		cc := strings.Split(cellChild{}.View(), "\n")
		pad := func(s string, w int) string { return s + strings.Repeat(" ", max(0, w-ansi.Width(s))) }
		for i := 0; i < 2; i++ {
			rows[1+i] = pad(sc[i], 12) + cc[i]
		}
		str = strings.Join(rows, "\n")
		return
	}
	var dOut, sOut bytes.Buffer
	dm := &drawModel{}
	dp := cdProgram(dm, &dOut)
	sm := &stringModelHolder{}
	sp := cdProgram(sm, &sOut)
	for n := 0; n < 3; n++ {
		d, s := frame(n)
		dm.draw, sm.view = d, s
		dp.renderIfChanged()
		sp.renderIfChanged()
		ds := cdScreen(dOut.Bytes())
		sameScreens(t, ds, cdScreen(sOut.Bytes()))
		if l := ds.Lines(); !strings.HasPrefix(l[1], "abc") || !strings.Contains(l[1], "hi 世界") || l[0] != "title" {
			t.Fatalf("frame %d screen = %q", n, l)
		}
		if c := ds.Cell(12, 1); !c.Bold || c.FG == nil {
			t.Fatalf("style lost: %+v", c)
		}
	}
	if dm.views != 0 {
		t.Fatalf("View called %d times on the CellDrawer root", dm.views)
	}
}

type stringModelHolder struct{ view string }

func (m *stringModelHolder) Init() Cmd               { return nil }
func (m *stringModelHolder) Update(Msg) (Model, Cmd) { return m, nil }
func (m *stringModelHolder) View() string            { return m.view }

// With a profile below TrueColor the direct path is unavailable and the View
// string is drawn, so the colour downgrade still applies.
func TestCellDrawerFallsBackToViewWhenIneligible(t *testing.T) {
	m := &drawModel{draw: func(buf *cellbuf.Buffer, r cellbuf.Rect) { buf.SetString(0, 0, "drawn", 0) }}
	var out bytes.Buffer
	p := cdProgram(m, &out)
	p.colorProfile = ansi.ANSI16
	p.renderIfChanged()
	if m.views == 0 {
		t.Fatal("View not used when the direct path is unavailable")
	}
}

// A Program may switch between a direct frame and a string frame (the model
// changes, or the direct path becomes unavailable); the screen stays right.
func TestCellDrawerInterleavedWithStringFrames(t *testing.T) {
	dm := &drawModel{draw: func(buf *cellbuf.Buffer, r cellbuf.Rect) {
		buf.SetString(0, 0, "direct one", 0)
		buf.SetString(0, 1, "row2", 0)
	}}
	var out bytes.Buffer
	p := cdProgram(dm, &out)
	p.renderIfChanged()
	p.model = stringModel("string two")
	p.renderIfChanged()
	if got := cdScreen(out.Bytes()).Lines(); got[0] != "string two" || got[1] != "" {
		t.Fatalf("after string frame: %q", got)
	}
	p.model = dm
	p.renderIfChanged()
	if got := cdScreen(out.Bytes()).Lines(); got[0] != "direct one" || got[1] != "row2" {
		t.Fatalf("after direct frame: %q", got)
	}
}

func stringModel(s string) Model { return &stringModelHolder{view: s} }
