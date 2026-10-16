package tui

import (
	"os"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
)

// quitOnInit draws one frame and asks to quit at once.
type quitOnInit struct{}

func (quitOnInit) Init() Cmd               { return Quit() }
func (quitOnInit) Update(Msg) (Model, Cmd) { return quitOnInit{}, nil }
func (quitOnInit) View() string            { return "hello" }

// BenchmarkStartup is the cost of a whole program's life when nothing happens:
// build the Program, take the terminal, draw the first frame, quit and restore
// it. It measures the library's fixed start-up cost, not a terminal's.
func BenchmarkStartup(b *testing.B) {
	out, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		b.Fatal(err)
	}
	defer out.Close()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		p := NewProgram(quitOnInit{}, WithInput(strings.NewReader("")), WithOutput(out), WithColorProfile(ansi.TrueColor), WithAltScreen(false))
		if _, err := p.Run(); err != nil {
			b.Fatal(err)
		}
	}
}
