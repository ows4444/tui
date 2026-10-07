package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui/ansi"
)

const zwjFamily = "\U0001F468\u200d\U0001F469\u200d\U0001F467" // 3 emoji joined: 2 cols as a cluster, 6 per codepoint

// setClusterWidth sets ansi's process-wide cluster width. The probe tests
// exercise that switch and must put it back, and nothing else can set it.
func setClusterWidth(on bool) {
	//lint:ignore SA1019 the deprecated process-wide switch is what these tests cover and restore
	ansi.SetClusterWidth(on)
}

// probeResult is what a probe run left behind: the output the terminal saw,
// the Program (for Measurer) and the ResizeMsgs the model received.
type probeResult struct {
	out     string
	p       *Program
	resizes []ResizeMsg
}

// measurerModel records every ResizeMsg and quits once it has seen the
// CapabilitiesMsg and a ResizeMsg carrying a non-zero Measurer.
type measurerModel struct {
	resizes []ResizeMsg
	caps    bool
}

func (measurerModel) Init() Cmd { return nil }
func (m measurerModel) Update(msg Msg) (Model, Cmd) {
	switch msg := msg.(type) {
	case ResizeMsg:
		m.resizes = append(m.resizes, msg)
	case CapabilitiesMsg:
		m.caps = true
	}
	if m.caps && len(m.resizes) > 0 && m.resizes[len(m.resizes)-1].Measurer != (ansi.Measurer{}) {
		return m, Quit()
	}
	return m, nil
}
func (measurerModel) View() string { return "" }

// runProbeWith runs a Program whose terminal answers the probe with replies and
// waits until the model has been told the probed Measurer in a ResizeMsg. The
// probe can finish before or after the first ResizeMsg, so that is either the
// first message or a second one.
func runProbeWith(t *testing.T, replies string) probeResult {
	t.Helper()
	pr, pw := mustPipe(t)
	t.Cleanup(func() { pr.Close(); pw.Close() })
	out, readOut := captureOutput(t)
	p := NewProgram(measurerModel{}, WithInput(pr), WithOutput(out), WithCapabilityProbe(time.Hour))
	ch := make(chan runResult, 1)
	go func() { m, err := p.runLoop(); ch <- runResult{m, err} }()
	if _, err := pw.Write([]byte(replies)); err != nil {
		t.Fatal(err)
	}
	var res runResult
	select {
	case res = <-ch:
	case <-time.After(3 * time.Second):
		t.Fatal("runLoop did not return in time")
	}
	return probeResult{out: string(readOut()), p: p, resizes: res.model.(measurerModel).resizes}
}

// The probe's answer about clusters belongs to the Program, not the process:
// ansi.Width keeps its own setting, and the model hears about it in a ResizeMsg.
func TestProbeMode2027SettableEnablesClusterWidth(t *testing.T) {
	t.Cleanup(func() { setClusterWidth(true) })
	for _, val := range []string{"1", "2"} {
		setClusterWidth(false)
		r := runProbeWith(t, "\x1b[?2027;"+val+"$y\x1b[?62;c")
		if !strings.Contains(r.out, "\x1b[?2027h") {
			t.Errorf("value %s: ESC[?2027h not sent in %q", val, r.out)
		}
		if w := r.p.Measurer().Width(zwjFamily); w != 2 {
			t.Errorf("value %s: Program cluster width = %d, want 2", val, w)
		}
		if w := ansi.Width(zwjFamily); w != 6 {
			t.Errorf("value %s: the probe changed the process-wide width to %d, want it left at 6", val, w)
		}
		if last := r.resizes[len(r.resizes)-1]; last.Measurer != ansi.ClusterMeasurer(true) {
			t.Errorf("value %s: last ResizeMsg = %+v, want the cluster Measurer", val, last)
		}
	}
}

func TestProbeMode2027UnsupportedUsesPerCodepointWidth(t *testing.T) {
	t.Cleanup(func() { setClusterWidth(true) })
	for name, replies := range map[string]string{
		"reset":       "\x1b[?2027;0$y\x1b[?62;c",
		"unanswered":  "\x1b[?62;c",
		"unsupported": "\x1b[?2027;4$y\x1b[?62;c",
	} {
		setClusterWidth(true)
		r := runProbeWith(t, replies)
		if strings.Contains(r.out, "\x1b[?2027h") {
			t.Errorf("%s: ESC[?2027h sent", name)
		}
		if w := r.p.Measurer().Width(zwjFamily); w != 6 {
			t.Errorf("%s: Program per-codepoint width = %d, want 6", name, w)
		}
		if w := ansi.Width(zwjFamily); w != 2 {
			t.Errorf("%s: the probe changed the process-wide width to %d, want it left at 2", name, w)
		}
		if last := r.resizes[len(r.resizes)-1]; last.Measurer != ansi.ClusterMeasurer(false) {
			t.Errorf("%s: last ResizeMsg = %+v, want the per-codepoint Measurer", name, last)
		}
	}
}

// Before any probe the Measurer is the zero value, which follows the process
// setting, so a Program without WithCapabilityProbe measures as it always did.
func TestMeasurerIsZeroWithoutAProbe(t *testing.T) {
	p := NewProgram(staticModel{})
	if p.Measurer() != (ansi.Measurer{}) {
		t.Fatalf("Measurer = %+v, want the zero value", p.Measurer())
	}
}

// Two Programs in one process keep their own answers.
func TestProgramsDoNotShareAMeasurer(t *testing.T) {
	a, b := NewProgram(staticModel{}), NewProgram(staticModel{})
	a.applyCapabilities(Capabilities{GraphemeClusters: false})
	if a.Measurer().Clusters() {
		t.Fatal("a: clusters on after a probe that said no")
	}
	if b.Measurer() != (ansi.Measurer{}) {
		t.Fatalf("b: Measurer = %+v after a's probe, want untouched", b.Measurer())
	}
}

func TestMode2027ResetOnRestore(t *testing.T) {
	t.Cleanup(func() { setClusterWidth(true) })
	p := NewProgram(staticModel{}, WithCapabilityProbe(time.Hour))
	if strings.Contains(p.restoreFixedString(), "\x1b[?2027l") {
		t.Error("reset emitted although 2027 was never set")
	}
	out, _ := captureOutput(t)
	p.output = out
	p.applyCapabilities(Capabilities{GraphemeClusters: true})
	if !strings.Contains(p.restoreFixedString(), "\x1b[?2027l") {
		t.Error("restore does not reset mode 2027")
	}
}

// signalModel closes started on its first ResizeMsg, then behaves as measurerModel.
type signalModel struct {
	measurerModel
	started chan struct{}
}

func (m signalModel) Update(msg Msg) (Model, Cmd) {
	if _, ok := msg.(ResizeMsg); ok && len(m.resizes) == 0 {
		close(m.started)
	}
	next, cmd := m.measurerModel.Update(msg)
	m.measurerModel = next.(measurerModel)
	return m, cmd
}

// A probe that finishes after the first frame re-sends the size with the new
// Measurer, so a model that already measured with the default hears the change.
func TestProbeAfterStartSendsASecondResizeMsg(t *testing.T) {
	pr, pw := mustPipe(t)
	t.Cleanup(func() { pr.Close(); pw.Close() })
	out, _ := captureOutput(t)
	started := make(chan struct{})
	p := NewProgram(signalModel{started: started}, WithInput(pr), WithOutput(out), WithCapabilityProbe(time.Hour))
	ch := make(chan runResult, 1)
	go func() { m, err := p.runLoop(); ch <- runResult{m, err} }()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("no first ResizeMsg")
	}
	if _, err := pw.Write([]byte("\x1b[?62;c")); err != nil { // DA1 only: nothing supported
		t.Fatal(err)
	}
	select {
	case res := <-ch:
		got := res.model.(signalModel).resizes
		if len(got) != 2 || got[0].Measurer != (ansi.Measurer{}) || got[1].Measurer != ansi.ClusterMeasurer(false) {
			t.Fatalf("ResizeMsgs = %+v, want the default Measurer then the per-codepoint one", got)
		}
		if got[0].Width != got[1].Width || got[0].Height != got[1].Height {
			t.Fatalf("size changed between the two: %+v", got)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("runLoop did not return in time")
	}
}
