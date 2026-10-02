package tui

import (
	"io"
	"strings"
	"testing"
	"time"
)

// ringModel counts messages and sleeps a little in Update so the log has a
// non-zero duration to show.
type ringModel struct{ N int }

func (m ringModel) Init() Cmd { return nil }
func (m ringModel) Update(msg Msg) (Model, Cmd) {
	if _, ok := msg.(ringMsg); ok {
		m.N++
		time.Sleep(time.Millisecond)
	}
	return m, nil
}
func (m ringModel) View() string { return "hello" }

type ringMsg int

func TestMsgRingKeepsLast200(t *testing.T) {
	var r msgRing
	for i := 1; i <= 450; i++ {
		r.add(ringMsg(i), time.Duration(i))
	}
	es := r.entries()
	if len(es) != 200 || es[0].n != 251 || es[199].n != 450 {
		t.Fatalf("got %d entries, first %d last %d", len(es), es[0].n, es[len(es)-1].n)
	}
}

// Criterion #116: with the message pane open, the last messages show with
// their Update duration; the model pane shows a %#v dump.
func TestInspectorMessagesAndModelDump(t *testing.T) {
	pr, pw := io.Pipe()
	defer pw.Close()
	out := &unsyncBuffer{}
	p := NewProgram(ringModel{}, WithInput(pr), WithOutput(out),
		WithInspector(InspectorKeys{Panel: InspectorOff, Layout: InspectorOff, Messages: "f10", Model: "f9"}))
	errc := make(chan error, 1)
	go func() { _, err := p.Run(); errc <- err }()
	p.Send(ResizeMsg{Width: 80, Height: 20})
	for i := 1; i <= 3; i++ {
		p.Send(ringMsg(i))
	}
	snapshot := func() string {
		time.Sleep(150 * time.Millisecond)
		p.outMu.Lock()
		defer p.outMu.Unlock()
		return screenOf(out, 80, 20)
	}
	snapshot()
	_, _ = io.WriteString(pw, "\x1b[21~") // F10
	_, _ = io.WriteString(pw, "\x1b[20~") // F9
	on := snapshot()
	for _, want := range []string{"messages (f10 closes)", "tui.ringMsg 3", " us", "model (f9 closes)", "tui.ringModel{N:3}"} {
		if !strings.Contains(on, want) {
			t.Errorf("missing %q:\n%s", want, on)
		}
	}
	_, _ = io.WriteString(pw, "\x1b[21~")
	_, _ = io.WriteString(pw, "\x1b[20~")
	if off := snapshot(); strings.Contains(off, "messages (") || strings.Contains(off, "model (") {
		t.Fatalf("panes still shown:\n%s", off)
	}
	p.Send(QuitMsg{})
	<-errc
}
