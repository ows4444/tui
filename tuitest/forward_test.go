package tuitest_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/tuitest"
)

type plain struct{ view string }

func (m plain) Init() tui.Cmd                       { return nil }
func (m plain) Update(tui.Msg) (tui.Model, tui.Cmd) { return m, nil }
func (m plain) View() string                        { return m.view }

// placer is a model that asks for the hardware cursor.
type placer struct {
	plain
	x, y int
}

func (m placer) Update(tui.Msg) (tui.Model, tui.Cmd) { return m, nil }
func (m placer) CursorPos() (int, int, bool)         { return m.x, m.y, true }

// spoken implements Linearizer.
type spoken struct{ plain }

func (m spoken) Update(tui.Msg) (tui.Model, tui.Cmd) { return m, nil }
func (spoken) Linearize() string                     { return "spoken line" }

// themed implements Themeable and shows whether it was themed.
type themed struct {
	plain
	got bool
}

func (m themed) Update(tui.Msg) (tui.Model, tui.Cmd) { return m, nil }
func (m themed) SetTheme(theme.Theme) tui.Model      { m.got = true; return m }
func (m themed) View() string {
	if m.got {
		return "themed"
	}
	return "unthemed"
}

// TestHostPlacesTheModelsHardwareCursor proves criterion #54: a model under
// tuitest that implements CursorPlacer gets the cursor where it asked.
func TestHostPlacesTheModelsHardwareCursor(t *testing.T) {
	s := tuitest.New(placer{plain: plain{view: "hello world"}, x: 4, y: 0}, 20, 3)
	defer s.Close()
	if x, y, vis := s.Cursor(); x != 4 || y != 0 || !vis {
		t.Fatalf("Cursor() = %d,%d visible=%v, want 4,0 visible", x, y, vis)
	}
}

func TestHostKeepsCursorHiddenWithoutCursorPlacer(t *testing.T) {
	s := tuitest.New(plain{view: "hello"}, 20, 3)
	defer s.Close()
	if _, _, vis := s.Cursor(); vis {
		t.Fatal("a model without CursorPlacer must leave the cursor hidden")
	}
}

func TestHostForwardsLinearizer(t *testing.T) {
	s := tuitest.New(spoken{plain{view: "drawn"}}, 30, 3, tui.WithAccessible(true))
	defer s.Close()
	got := strings.Join(s.Screen(), "\n")
	if !strings.Contains(got, "spoken line") || strings.Contains(got, "drawn") {
		t.Fatalf("accessible mode showed %q, want the Linearize text", got)
	}
}

func TestHostForwardsThemeable(t *testing.T) {
	s := tuitest.New(themed{}, 20, 2, tui.WithTheme(theme.DefaultAuto()))
	defer s.Close()
	if got := strings.Join(s.Screen(), "\n"); !strings.Contains(got, "themed") || strings.Contains(got, "unthemed") {
		t.Fatalf("SetTheme was not forwarded: %q", got)
	}
}

// outer gets CursorPos by embedding and keeps itself across Update.
type outer struct{ placer }

func (m outer) Update(msg tui.Msg) (tui.Model, tui.Cmd) { return m, nil }

// TestHostForwardsThroughEmbedding covers a model whose optional interface
// comes from an embedded type, the way real root models often get it.
func TestHostForwardsThroughEmbedding(t *testing.T) {
	s := tuitest.New(outer{placer{plain: plain{view: "x"}, x: 1, y: 0}}, 10, 2)
	defer s.Close()
	if x, _, vis := s.Cursor(); x != 1 || !vis {
		t.Fatalf("embedded CursorPlacer not forwarded: %d %v", x, vis)
	}
}

// TestCellReportsGraphemeAndStyle proves criterion #55.
func TestCellReportsGraphemeAndStyle(t *testing.T) {
	view := ansi.NewStyle().Bold().Foreground(ansi.Red).Render("x") + "y"
	s := tuitest.New(plain{view: view}, 10, 2)
	defer s.Close()
	c := s.Cell(0, 0)
	if c.Grapheme != "x" || !c.Bold || c.FG != ansi.Red || c.Italic || c.BG != nil {
		t.Fatalf("Cell(0,0) = %+v, want bold red x", c)
	}
	if c := s.Cell(1, 0); c.Grapheme != "y" || c.Bold || c.FG != nil {
		t.Fatalf("Cell(1,0) = %+v, want plain y (style must not bleed)", c)
	}
	if c := s.Cell(99, 99); c.Grapheme != " " {
		t.Fatalf("off-screen cell = %+v", c)
	}
}

func TestCellColourForms(t *testing.T) {
	view := ansi.NewStyle().Foreground(ansi.RGB{R: 1, G: 2, B: 3}).Background(ansi.Color256(200)).Render("a")
	s := tuitest.New(plain{view: view}, 10, 2)
	defer s.Close()
	c := s.Cell(0, 0)
	if c.FG != (ansi.RGB{R: 1, G: 2, B: 3}) || c.BG != ansi.Color256(200) {
		t.Fatalf("Cell = %+v", c)
	}
}

// mouseLog renders every mouse event it has received.
type mouseLog struct{ log []string }

func (m mouseLog) Init() tui.Cmd { return nil }
func (m mouseLog) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	if ev, ok := msg.(tui.MouseEvent); ok {
		kind := "press"
		if ev.Action == tui.MouseActionRelease {
			kind = "release"
		}
		name := map[tui.MouseButton]string{
			tui.MouseButtonLeft: "left", tui.MouseButtonWheelUp: "up", tui.MouseButtonWheelDown: "down",
		}[ev.Button]
		m.log = append(append([]string(nil), m.log...), fmt.Sprintf("%s %s %d,%d", name, kind, ev.X, ev.Y))
	}
	return m, nil
}
func (m mouseLog) View() string { return strings.Join(m.log, "\n") }

func TestClickAndWheelReachTheModel(t *testing.T) {
	s := tuitest.New(mouseLog{}, 20, 8)
	defer s.Close()
	s.Click(3, 1)
	s.Wheel(2, 2, -2)
	s.Wheel(5, 6, 1)
	want := []string{
		"left press 3,1", "left release 3,1",
		"up press 2,2", "up press 2,2",
		"down press 5,6",
	}
	got := s.Screen()
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("row %d = %q, want %q\nscreen: %q", i, got[i], w, got)
		}
	}
}
