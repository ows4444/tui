package tui

import (
	"testing"
	"time"
)

// capSeenModel records p.Capabilities() as seen from Update when the
// CapabilitiesMsg arrives.
type capSeenModel struct {
	p    **Program
	msg  *CapabilitiesMsg
	seen *Capabilities
}

func (capSeenModel) Init() Cmd { return nil }
func (m capSeenModel) Update(msg Msg) (Model, Cmd) {
	if c, ok := msg.(CapabilitiesMsg); ok {
		*m.msg, *m.seen = c, (*m.p).Capabilities()
		return m, Quit()
	}
	return m, nil
}
func (capSeenModel) View() string { return "" }

func TestProgramCapabilitiesMatchesCapabilitiesMsg(t *testing.T) {
	t.Cleanup(func() { setClusterWidth(true) })
	var msg CapabilitiesMsg
	var seen Capabilities
	var p *Program
	pr, pw := mustPipe(t)
	t.Cleanup(func() { pr.Close(); pw.Close() })
	out, _ := captureOutput(t)
	p = NewProgram(capSeenModel{p: &p, msg: &msg, seen: &seen}, WithInput(pr), WithOutput(out), WithCapabilityProbe(time.Hour))
	if p.CapabilitiesKnown() {
		t.Fatal("known before the probe finished")
	}
	done := make(chan struct{})
	go func() { defer close(done); _, _ = p.runLoop() }()
	_, _ = pw.Write([]byte("\x1b[?2026;1$y\x1b[?2027;2$y\x1b[?62;c"))
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("no CapabilitiesMsg")
	}
	if !msg.Capabilities.SyncOutput || !msg.Capabilities.GraphemeClusters {
		t.Fatalf("msg = %#v", msg.Capabilities)
	}
	if seen != msg.Capabilities || p.Capabilities() != msg.Capabilities || !p.CapabilitiesKnown() {
		t.Errorf("Capabilities() = %#v (in Update %#v), msg %#v", p.Capabilities(), seen, msg.Capabilities)
	}
}
