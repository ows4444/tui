package tui

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// keyLogger records every Key and the first ResizeMsg, and quits on 'q'.
type keyLogger struct {
	keys    string
	resize  ResizeMsg
	gotSize bool
	errs    []error
}

func (m keyLogger) Init() Cmd { return nil }

func (m keyLogger) Update(msg Msg) (Model, Cmd) {
	switch msg := msg.(type) {
	case ResizeMsg:
		if !m.gotSize {
			m.resize, m.gotSize = msg, true
		}
	case InputErrorMsg:
		m.errs = append(m.errs, msg.Err)
	case Key:
		if msg.Type == KeyRunes {
			m.keys += msg.Text
			if msg.Text == "q" {
				return m, Quit()
			}
		}
	}
	return m, nil
}

func (m keyLogger) View() string { return "typed:" + m.keys }

// runReaderProgram runs p and fails the test if Run does not return in time.
func runReaderProgram(t *testing.T, p *Program) (keyLogger, error) {
	t.Helper()
	type res struct {
		m   Model
		err error
	}
	ch := make(chan res, 1)
	go func() {
		m, err := p.Run()
		ch <- res{m, err}
	}()
	select {
	case r := <-ch:
		return r.m.(keyLogger), r.err
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return in time")
		return keyLogger{}, nil
	}
}

func TestReaderAndWriterDriveAProgramWithoutATerminal(t *testing.T) {
	var out bytes.Buffer
	p := NewProgram(keyLogger{},
		WithInput(strings.NewReader("abq")),
		WithOutput(&out),
		WithAltScreen(false),
		WithCellRenderer(false), // the test looks for whole frames in the byte stream
	)
	m, err := runReaderProgram(t, p)
	if err != nil {
		t.Fatalf("Run returned %v", err)
	}
	if m.keys != "abq" {
		t.Errorf("model saw keys %q, want abq", m.keys)
	}
	// The frames went to the buffer, ending with the latest state.
	if !strings.Contains(out.String(), "typed:ab") {
		t.Errorf("output does not contain the rendered frames: %q", out.String())
	}
}

func TestInputReaderEOFQuitsAfterHandlingEarlierKeys(t *testing.T) {
	var out bytes.Buffer
	// No 'q': only EOF can end the program.
	p := NewProgram(keyLogger{},
		WithInput(strings.NewReader("xyz")),
		WithOutput(&out),
	)
	m, err := runReaderProgram(t, p)
	if err != nil {
		t.Fatalf("Run returned %v", err)
	}
	if m.keys != "xyz" {
		t.Errorf("keys before EOF were not all handled: %q", m.keys)
	}
	if len(m.errs) != 0 {
		t.Errorf("EOF surfaced as an InputErrorMsg: %v", m.errs)
	}
}

func TestEmptyInputReaderReturnsImmediately(t *testing.T) {
	p := NewProgram(keyLogger{}, WithInput(strings.NewReader("")), WithOutput(io.Discard))
	if _, err := runReaderProgram(t, p); err != nil {
		t.Fatalf("Run returned %v", err)
	}
}

type errReader struct{ err error }

func (r errReader) Read([]byte) (int, error) { return 0, r.err }

// quitOnError quits when it receives an InputErrorMsg.
type quitOnError struct{ errs []error }

func (m quitOnError) Init() Cmd    { return nil }
func (m quitOnError) View() string { return "" }
func (m quitOnError) Update(msg Msg) (Model, Cmd) {
	if e, ok := msg.(InputErrorMsg); ok {
		m.errs = append(m.errs, e.Err)
		return m, Quit()
	}
	return m, nil
}

func TestInputReaderErrorIsDeliveredToUpdate(t *testing.T) {
	boom := errors.New("boom")
	p := NewProgram(quitOnError{}, WithInput(errReader{boom}), WithOutput(io.Discard))
	type res struct {
		m   Model
		err error
	}
	ch := make(chan res, 1)
	go func() {
		m, err := p.Run()
		ch <- res{m, err}
	}()
	select {
	case r := <-ch:
		if r.err != nil {
			t.Fatalf("Run returned %v", r.err)
		}
		if errs := r.m.(quitOnError).errs; len(errs) != 1 || !errors.Is(errs[0], boom) {
			t.Errorf("errors delivered = %v, want [boom]", errs)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return in time")
	}
}

func TestPlainWriterGetsTheDefaultSize(t *testing.T) {
	p := NewProgram(keyLogger{}, WithInput(strings.NewReader("q")), WithOutput(io.Discard))
	m, err := runReaderProgram(t, p)
	if err != nil {
		t.Fatal(err)
	}
	if !m.gotSize || m.resize != (ResizeMsg{Width: 80, Height: 24}) {
		t.Errorf("initial size = %+v (seen=%v), want 80x24", m.resize, m.gotSize)
	}
}

// A writer that is not a file has no terminal to query, so the Program must
// not treat it as one: termSize reports "unknown" and Run succeeds where the
// default file-based path would reject a non-terminal input.
func TestTermSizeIsUnknownForAPlainWriter(t *testing.T) {
	p := NewProgram(keyLogger{}, WithOutput(&bytes.Buffer{}))
	if _, _, ok := p.termSize(); ok {
		t.Error("termSize reported a size for a bytes.Buffer")
	}
	f, err := os.CreateTemp(t.TempDir(), "out")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	p = NewProgram(keyLogger{}, WithOutput(f))
	if _, _, ok := p.termSize(); ok {
		t.Error("termSize reported a size for a regular file")
	}
}

// Without the new options nothing changes: a non-terminal input is still
// rejected, exactly as before.
func TestDefaultPathStillRequiresATerminal(t *testing.T) {
	r, w := mustPipe(t)
	defer r.Close()
	defer w.Close()
	p := NewProgram(keyLogger{}, WithInput(r), WithOutput(mustDevNull(t)))
	if _, err := p.Run(); err == nil || !strings.Contains(err.Error(), "not a terminal") {
		t.Errorf("Run = %v, want the not-a-terminal error", err)
	}
}

// Of several WithInput and WithOutput options the last one given wins: a file
// after a reader reads the file, a reader after a file reads the reader.
func TestInputAndOutputOptionsLastWins(t *testing.T) {
	r, w := mustPipe(t)
	defer r.Close()
	defer w.Close()
	p := NewProgram(keyLogger{}, WithInput(strings.NewReader("q")), WithInput(r))
	if p.inReader != nil || p.input != r {
		t.Errorf("a file after a reader: inReader=%v input=%v", p.inReader, p.input)
	}
	p = NewProgram(keyLogger{}, WithInput(r), WithInput(strings.NewReader("q")))
	if p.inReader == nil {
		t.Error("a reader after a file did not take effect")
	}
	var out bytes.Buffer
	p = NewProgram(keyLogger{}, WithOutput(&out), WithOutput(mustDevNull(t)))
	if _, isFile := p.output.(*os.File); !isFile {
		t.Error("the second WithOutput did not take effect")
	}
}
