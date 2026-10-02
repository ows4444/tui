package cellcheck

import (
	"testing"

	"github.com/ows4444/tui"
)

type viewModel string

func (v viewModel) Init() tui.Cmd                       { return nil }
func (v viewModel) Update(tui.Msg) (tui.Model, tui.Cmd) { return v, nil }
func (v viewModel) View() string                        { return string(v) }

func TestFallbacksDetects(t *testing.T) {
	if fb := Fallbacks(viewModel("\x1b[31mok\x1b[0m"), 40, 10); len(fb) != 0 {
		t.Errorf("clean view: %v", fb)
	}
	if fb := Fallbacks(viewModel("a\x01b"), 40, 10); len(fb) == 0 {
		t.Errorf("control char view: %v", fb)
	}
}
