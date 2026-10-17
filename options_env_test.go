package tui

import (
	"io"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ows4444/tui/ansi"
)

type nopModel struct{}

func (nopModel) Init() Cmd               { return nil }
func (nopModel) Update(Msg) (Model, Cmd) { return nopModel{}, nil }
func (nopModel) View() string            { return "x" }

func TestWithEnvironment(t *testing.T) {
	p := NewProgram(nopModel{}, WithInput(strings.NewReader("")), WithOutput(&unsyncBuffer{}),
		WithEnvironment([]string{"TERM=xterm", "FOO=1", "FOO=2", "NO_COLOR=1"}))
	if p.Getenv("FOO") != "2" || p.Getenv("TERM") != "xterm" || p.Getenv("MISSING") != "" {
		t.Fatalf("Getenv: FOO=%q TERM=%q", p.Getenv("FOO"), p.Getenv("TERM"))
	}
	if p.ColorProfile() != ansi.NoColor {
		t.Fatalf("NO_COLOR in the program's environment gave %v", p.ColorProfile())
	}
}

func TestEnvironmentDoesNotLeakTheProcess(t *testing.T) {
	t.Setenv("TERM", "dumb")
	t.Setenv("ACCESSIBLE", "1")
	t.Setenv("NO_ANIMATION", "1")
	p := NewProgram(nopModel{}, WithInput(strings.NewReader("")), WithOutput(&unsyncBuffer{}),
		WithEnvironment([]string{"TERM=xterm-256color"}), WithAccessibleAuto())
	if p.Accessible() || p.ReducedMotion() {
		t.Fatalf("the process environment reached a Program with its own: accessible=%v reduced=%v", p.Accessible(), p.ReducedMotion())
	}
	q := NewProgram(nopModel{}, WithInput(strings.NewReader("")), WithOutput(&unsyncBuffer{}),
		WithEnvironment([]string{"ACCESSIBLE=1", "REDUCE_MOTION=1"}), WithAccessibleAuto())
	if !q.Accessible() || !q.ReducedMotion() {
		t.Fatalf("the program's own variables were ignored: accessible=%v reduced=%v", q.Accessible(), q.ReducedMotion())
	}
	// Without the option the process environment still decides.
	r := NewProgram(nopModel{}, WithInput(strings.NewReader("")), WithOutput(&unsyncBuffer{}), WithAccessibleAuto())
	if !r.Accessible() || !r.ReducedMotion() {
		t.Fatalf("the process environment was lost: accessible=%v reduced=%v", r.Accessible(), r.ReducedMotion())
	}
}

func TestOptionOrderStillDecidesAccessibleAndMotion(t *testing.T) {
	env := WithEnvironment([]string{"ACCESSIBLE=1", "NO_ANIMATION=1"})
	in, out := WithInput(strings.NewReader("")), WithOutput(&unsyncBuffer{})
	if NewProgram(nopModel{}, in, out, WithAccessibleAuto(), env, WithAccessible(false)).Accessible() {
		t.Error("a later WithAccessible(false) lost to the environment")
	}
	if !NewProgram(nopModel{}, in, out, WithAccessible(false), WithAccessibleAuto(), env).Accessible() {
		t.Error("a later WithAccessibleAuto lost to an earlier WithAccessible")
	}
	if NewProgram(nopModel{}, in, out, env, WithReducedMotion(false)).ReducedMotion() {
		t.Error("WithReducedMotion(false) lost to the environment")
	}
}

// quitterModel records what reaches Update.
type quitterModel struct{ got *atomic.Int32 }

func (quitterModel) Init() Cmd { return nil }
func (m quitterModel) Update(msg Msg) (Model, Cmd) {
	if _, ok := msg.(Key); ok {
		m.got.Add(1)
	}
	return m, nil
}
func (quitterModel) View() string { return "x" }

func TestWithFilterCanHoldBackQuitAndDropMessages(t *testing.T) {
	pr, pw := io.Pipe()
	defer pw.Close()
	var got atomic.Int32
	var quits atomic.Int32
	p := NewProgram(quitterModel{got: &got}, WithInput(pr), WithOutput(&unsyncBuffer{}),
		WithFilter(func(_ Model, m Msg) Msg {
			switch v := m.(type) {
			case QuitMsg:
				if quits.Add(1) == 1 {
					return nil // the first quit is held back
				}
			case Key:
				if v.String() == "x" {
					return nil // dropped
				}
			}
			return m
		}))
	errc := make(chan error, 1)
	go func() { _, err := p.Run(); errc <- err }()
	p.Send(ResizeMsg{Width: 10, Height: 2})
	p.Send(Key{Type: KeyRunes, Text: "x"})
	p.Send(Key{Type: KeyRunes, Text: "y"})
	p.Send(QuitMsg{})
	select {
	case <-errc:
		t.Fatal("the first QuitMsg got through the filter")
	case <-time.After(200 * time.Millisecond):
	}
	p.Send(QuitMsg{})
	select {
	case <-errc:
	case <-time.After(3 * time.Second):
		t.Fatal("the second QuitMsg did not end Run")
	}
	if got.Load() != 1 {
		t.Fatalf("Update saw %d keys; want 1 (x was dropped)", got.Load())
	}
}

func TestWithoutRendererDrawsNothing(t *testing.T) {
	pr, pw := io.Pipe()
	defer pw.Close()
	out := &unsyncBuffer{}
	var got atomic.Int32
	p := NewProgram(quitterModel{got: &got}, WithInput(pr), WithOutput(out), WithoutRenderer())
	errc := make(chan error, 1)
	go func() { _, err := p.Run(); errc <- err }()
	p.Send(ResizeMsg{Width: 10, Height: 2})
	p.Send(Key{Type: KeyRunes, Text: "a"})
	time.Sleep(150 * time.Millisecond)
	p.Quit()
	<-errc
	if got.Load() != 1 {
		t.Fatalf("Update saw %d keys; want 1", got.Load())
	}
	if strings.Contains(string(out.b), "x") {
		t.Fatalf("a frame was written: %q", out.b)
	}
}

func TestWithWindowSize(t *testing.T) {
	var w, h atomic.Int32
	pr, pw := io.Pipe()
	defer pw.Close()
	p := NewProgram(startSizeModel{w: &w, h: &h}, WithInput(pr), WithOutput(&unsyncBuffer{}), WithWindowSize(132, 43))
	errc := make(chan error, 1)
	go func() { _, err := p.Run(); errc <- err }()
	deadline := time.Now().Add(3 * time.Second)
	for w.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	p.Quit()
	<-errc
	if w.Load() != 132 || h.Load() != 43 {
		t.Fatalf("the model started at %dx%d; want 132x43", w.Load(), h.Load())
	}
	// A non-positive size is ignored.
	q := NewProgram(nopModel{}, WithWindowSize(0, 50))
	if q.initW != 0 {
		t.Fatal("a width of 0 was taken")
	}
}

type startSizeModel struct{ w, h *atomic.Int32 }

func (startSizeModel) Init() Cmd { return nil }
func (m startSizeModel) Update(msg Msg) (Model, Cmd) {
	if r, ok := msg.(ResizeMsg); ok {
		m.w.Store(int32(r.Width))
		m.h.Store(int32(r.Height))
	}
	return m, nil
}
func (startSizeModel) View() string { return "x" }
