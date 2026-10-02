package tui

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// runErrPrint runs a loop that receives one stderr print and returns after
// the model has seen two messages.
func runErrPrint(t *testing.T, opts ...ProgramOption) {
	t.Helper()
	pr, pw := mustPipe(t)
	defer pr.Close()
	defer pw.Close()
	out, _ := captureOutput(t)

	p := NewProgram(printModel{view: "v", quitAfter: 2}, append([]ProgramOption{WithInput(pr), WithOutput(out), WithAltScreen(false)}, opts...)...)
	done := make(chan struct{})
	go func() {
		p.runLoop()
		close(done)
	}()
	p.msgs <- printMsg{text: "stderr-text", stderr: true}
	p.msgs <- Key{Type: KeyRunes, Text: "x", Code: 'x'}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("runLoop did not return in time")
	}
}

// WithErrOutput accepts any io.Writer, not only a file.
func TestWithErrOutputReceivesEprintlnOutput(t *testing.T) {
	var buf bytes.Buffer
	runErrPrint(t, WithErrOutput(&buf))
	if !strings.Contains(buf.String(), "stderr-text") {
		t.Fatalf("writer got %q, want it to contain the committed text", buf.String())
	}
}

// Given an *os.File it receives the text like any writer.
func TestWithErrOutputAcceptsAFile(t *testing.T) {
	errOut, read := captureOutput(t)
	runErrPrint(t, WithErrOutput(errOut))
	if !strings.Contains(string(read()), "stderr-text") {
		t.Fatalf("file got %q", read())
	}
}

// The last WithErrOutput wins.
func TestLastErrOutputWins(t *testing.T) {
	var a, b bytes.Buffer
	errOut, read := captureOutput(t)
	runErrPrint(t, WithErrOutput(&a), WithErrOutput(errOut), WithErrOutput(&b))
	if a.Len() != 0 || len(read()) != 0 || !strings.Contains(b.String(), "stderr-text") {
		t.Fatalf("first=%q file=%q last=%q", a.String(), read(), b.String())
	}
}
