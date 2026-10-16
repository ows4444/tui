package maskedinput

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/cellbuf"
)

func TestDrawCellsMatchesView(t *testing.T) {
	m := New()
	m.SetValue("1234")
	m.Focus()
	want, err := cellbuf.Parse(m.View())
	if err != nil {
		t.Fatal(err)
	}
	got := cellbuf.New(30, 1)
	m.DrawCells(got, got.Bounds())
	g, w := strings.TrimRight(strings.Join(got.Lines(), ""), " "), strings.TrimRight(strings.Join(want.Lines(), ""), " ")
	if g != w {
		t.Errorf("DrawCells %q, View %q", g, w)
	}
}
