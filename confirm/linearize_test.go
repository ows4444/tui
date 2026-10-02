package confirm

import (
	"testing"

	"github.com/ows4444/tui"
)

var _ tui.Linearizer = Model{}

func TestLinearize(t *testing.T) {
	m := New("Delete the file?")
	want := "Delete the file?\nYes, option 1 of 2, selected\nNo, option 2 of 2"
	if got := m.Linearize(); got != want {
		t.Errorf("Linearize =\n%s\nwant\n%s", got, want)
	}
	next, _ := m.Update(tui.Key{Type: tui.KeyRight})
	m = next
	m.YesLabel, m.NoLabel = "Delete", "Keep"
	want = "Delete the file?\nDelete, option 1 of 2\nKeep, option 2 of 2, selected"
	if got := m.Linearize(); got != want {
		t.Errorf("after right arrow =\n%s\nwant\n%s", got, want)
	}
}
