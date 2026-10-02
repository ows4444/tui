package tui

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
)

// providerModel exposes a focused widget's cell only through CursorCell.
type providerModel struct {
	view    string
	x, y    int
	focused bool
}

func (m providerModel) Init() Cmd               { return nil }
func (m providerModel) Update(Msg) (Model, Cmd) { return m, nil }
func (m providerModel) View() string            { return m.view }
func (m providerModel) CursorCell() (int, int, bool) {
	return m.x, m.y, m.focused
}

var _ CursorProvider = providerModel{}

// Criterion #76: a focused widget's CursorCell becomes the hardware cursor
// with no CursorPlacer in the app.
func TestProgramPlacesCursorFromCursorProvider(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(providerModel{view: "title\n> abc", x: 5, y: 1, focused: true}, WithOutput(out))
	p.render()
	want := "\r" + ansi.CursorForward(5) + ansi.CursorShow + ansi.SyncOutputDisable
	if !strings.HasSuffix(string(read()), want) {
		t.Fatalf("frame ends %q, want the cursor at col 5 of the last row", string(read()))
	}
}

// An unfocused provider keeps the cursor hidden, and CursorPlacer wins when
// a model has both.
func TestCursorProviderUnfocusedAndPlacerPrecedence(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(providerModel{view: "a\nb", x: 1, y: 0}, WithOutput(out))
	p.render()
	if strings.Contains(string(read()), ansi.CursorShow) {
		t.Fatal("cursor shown for an unfocused widget")
	}

	out2, read2 := captureOutput(t)
	p2 := NewProgram(bothModel{providerModel{view: "a\nb", x: 1, y: 1, focused: true}}, WithOutput(out2))
	p2.render()
	if want := "\r" + ansi.CursorShow + ansi.SyncOutputDisable; !strings.HasSuffix(string(read2()), want) {
		t.Fatalf("frame ends %q, want CursorPlacer's col 0", string(read2()))
	}
}

type bothModel struct{ providerModel }

func (bothModel) CursorPos() (int, int, bool) { return 0, 1, true }
