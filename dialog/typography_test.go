package dialog

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
)

func TestTitleUsesTypographyH2(t *testing.T) {
	m := New("Hi", "")
	m.Theme.Typography.H2 = ansi.NewStyle().Foreground(ansi.RGB{R: 7, G: 7, B: 7}).Italic()
	if !strings.Contains(m.Render(strings.Repeat(strings.Repeat("x", 40)+"\n", 9)), m.Theme.Typography.H2.Render("Hi")) {
		t.Fatal("Render ignores Typography.H2")
	}
}
