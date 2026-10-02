package tui

import (
	"io"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
)

type printOnceModel struct{ cmds []Cmd }

func (m printOnceModel) Init() Cmd               { return Sequence(m.cmds...) }
func (m printOnceModel) Update(Msg) (Model, Cmd) { return m, nil }
func (printOnceModel) View() string              { return "live" }

func runPrinting(t *testing.T, cmds []Cmd, opts ...ProgramOption) (out, errOut string) {
	t.Helper()
	pr, pw := io.Pipe()
	t.Cleanup(func() { pw.Close() })
	o, e := &strings.Builder{}, &strings.Builder{}
	opts = append(opts, WithInput(pr), WithOutput(o), WithErrOutput(e))
	if _, err := NewProgram(printOnceModel{cmds}, opts...).Run(); err != nil {
		t.Fatal(err)
	}
	return o.String(), e.String()
}

// #24: a Println made while the alternate screen is up is written to the main
// screen, after the alternate screen has been left, in order.
func TestPrintlnOnAltScreenLandsOnMainScreenAfterExit(t *testing.T) {
	out, _ := runPrinting(t, []Cmd{Println("first"), Println("second\nthird"), Quit()})
	leave := strings.LastIndex(out, ansi.AltScreenDisable)
	if leave < 0 {
		t.Fatalf("alternate screen never left: %q", out)
	}
	after := out[leave+len(ansi.AltScreenDisable):]
	if !strings.Contains(after, "first\r\nsecond\r\nthird\r\n") {
		t.Fatalf("after leaving the alt screen got %q, want the three lines in order", after)
	}
	if strings.Contains(out[:leave], "first") {
		t.Errorf("text was written onto the alternate screen, where the next frame erases it: %q", out[:leave])
	}
}

// ExitAltScreen mid-run flushes what was held, before the next frame.
func TestPrintlnHeldUntilExitAltScreen(t *testing.T) {
	out, _ := runPrinting(t, []Cmd{Println("held"), ExitAltScreen(), Quit()})
	leave := strings.Index(out, ansi.AltScreenDisable)
	if leave < 0 || !strings.Contains(out[leave:], "held\r\n") || strings.Contains(out[:leave], "held") {
		t.Fatalf("held text not written right after the alt screen was left: %q", out)
	}
}

func TestEprintlnOnAltScreenGoesToErrOutputAfterExit(t *testing.T) {
	out, errOut := runPrinting(t, []Cmd{Eprintln("oops"), Quit()})
	if !strings.Contains(errOut, "oops\r\n") {
		t.Fatalf("stderr got %q", errOut)
	}
	if strings.Contains(out, "oops") {
		t.Errorf("stderr text leaked to stdout: %q", out)
	}
}

// Inline (no alternate screen) behaviour is unchanged: printed at once.
func TestPrintlnInlineStillPrintsImmediately(t *testing.T) {
	out, _ := runPrinting(t, []Cmd{Println("now"), Quit()}, WithAltScreen(false))
	if !strings.Contains(out, "now\r\n") {
		t.Fatalf("got %q", out)
	}
}

func TestAltPrintQueueIsBounded(t *testing.T) {
	p := NewProgram(staticModel{view: "v"})
	for i := 0; i < maxAltPrints+5; i++ {
		p.holdAltPrint(printMsg{text: "x"})
	}
	if len(p.altPrints) != maxAltPrints {
		t.Fatalf("held %d prints, want at most %d", len(p.altPrints), maxAltPrints)
	}
}
