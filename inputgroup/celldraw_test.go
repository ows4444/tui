package inputgroup

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/cellbuf"
)

func TestDrawCellsShowsThePrefixAndSuffix(t *testing.T) {
	m := New("https://", ".com")
	m.SetValue("example")
	got := cellbuf.New(40, 1)
	m.DrawCells(got, got.Bounds())
	row := strings.Join(got.Lines(), "")
	if !strings.Contains(row, "https://") || !strings.Contains(row, "example") || !strings.Contains(row, ".com") {
		t.Errorf("DrawCells = %q", row)
	}
}
