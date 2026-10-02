package tui_test

import (
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/ows4444/tui"
)

// remoteTerm is what a remote session would supply: it answers for the far
// end and records what the Program asked of it.
type remoteTerm struct {
	mu      sync.Mutex
	raw, vt int
	unraw   int
	size    [2]int
}

func (*remoteTerm) IsTerminal() bool         { return true }
func (r *remoteTerm) Size() (int, int, bool) { return r.size[0], r.size[1], true }
func (r *remoteTerm) MakeRaw(bool) (func() error, error) {
	r.mu.Lock()
	r.raw++
	r.mu.Unlock()
	return func() error { r.mu.Lock(); r.unraw++; r.mu.Unlock(); return nil }, nil
}
func (r *remoteTerm) EnableOutputVT() (func() error, error) {
	r.mu.Lock()
	r.vt++
	r.mu.Unlock()
	return nil, nil
}

type sizeModel struct{ w, h int }

func (sizeModel) Init() tui.Cmd { return nil }
func (m sizeModel) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.w, m.h = msg.Width, msg.Height
	case tui.Key:
		return m, tui.Quit()
	}
	return m, nil
}
func (sizeModel) View() string { return "x" }

// An application outside this module can supply its own Terminal: here the far
// end reports 120x40, input and output are plain streams.
func TestWithTerminalFromOutsideThePackage(t *testing.T) {
	rt := &remoteTerm{size: [2]int{120, 40}}
	p := tui.NewProgram(sizeModel{},
		tui.WithTerminal(rt),
		tui.WithInput(strings.NewReader("q")),
		tui.WithOutput(io.Discard),
	)
	if _, err := p.Run(); err != nil {
		t.Fatal(err)
	}
	// With WithInput Run does not ask IsTerminal or enter raw mode, but
	// it does enable output VT through the port.
	if rt.raw != 0 || rt.vt != 1 {
		t.Errorf("raw=%d vt=%d, want 0 and 1", rt.raw, rt.vt)
	}
}
