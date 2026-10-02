package filepicker

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/internal/fsutil"
)

var _ tui.Linearizer = Model{}

func TestLinearize(t *testing.T) {
	m := Model{Dir: "/src"}
	if got := m.Linearize(); got != "/src, file picker, empty" {
		t.Errorf("empty = %q", got)
	}
	m.entries = []fsutil.Entry{{Name: "cmd", IsDir: true}, {Name: "main.go"}, {Name: "README.md"}}
	m.cursor = 1
	want := "/src, file picker, 3 entries\n" +
		"cmd, folder, entry 1 of 3\n" +
		"main.go, file, entry 2 of 3, selected\n" +
		"README.md, file, entry 3 of 3"
	if got := m.Linearize(); got != want {
		t.Errorf("Linearize =\n%s\nwant\n%s", got, want)
	}
}
