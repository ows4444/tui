package passwordinput

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/cellbuf"
)

func TestDrawCellsNeverDrawsTheSecret(t *testing.T) {
	m := New()
	m.SetValue("hunter2")
	m.Focus()
	buf := cellbuf.New(30, 1)
	m.DrawCells(buf, buf.Bounds())
	row := strings.Join(buf.Lines(), "")
	if strings.Contains(row, "hunter") || strings.Contains(row, "2") {
		t.Errorf("DrawCells drew the secret: %q", row)
	}
	if strings.TrimSpace(row) == "" {
		t.Error("DrawCells drew nothing")
	}
}
