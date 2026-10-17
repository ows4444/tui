package tui

import (
	"bytes"
	"strings"
	"testing"
)

type pointerModel struct{ shape *PointerShape }

func (pointerModel) Init() Cmd                 { return nil }
func (m pointerModel) Update(Msg) (Model, Cmd) { return m, nil }
func (pointerModel) View() string              { return "x" }
func (m pointerModel) ViewState() ViewState    { return ViewState{Pointer: m.shape} }

func TestSetPointerShapeWritesAndRestores(t *testing.T) {
	out := &bytes.Buffer{}
	p := NewProgram(pointerModel{}, WithOutput(out), WithAltScreen(false))
	p.width, p.height = 20, 5
	if m, ok := SetPointerShape(PointerPointer)().(modeMsg); ok {
		m.apply(p)
	}
	if !strings.Contains(out.String(), "\x1b]22;pointer\x1b\\") {
		t.Fatalf("not written: %q", out.String())
	}
	p.outMu.Lock()
	restore := p.titleAndShapeRestoreLocked()
	p.outMu.Unlock()
	if !strings.Contains(restore, "\x1b]22;default\x1b\\") {
		t.Fatalf("exit does not put the pointer back: %q", restore)
	}
	// Setting the default needs nothing restored.
	q := NewProgram(pointerModel{}, WithOutput(&bytes.Buffer{}), WithAltScreen(false))
	if m, ok := SetPointerShape(PointerDefault)().(modeMsg); ok {
		m.apply(q)
	}
	q.outMu.Lock()
	defer q.outMu.Unlock()
	if strings.Contains(q.titleAndShapeRestoreLocked(), "22;") {
		t.Fatal("a default pointer was 'restored'")
	}
}

func TestPointerSeqCannotEndEarly(t *testing.T) {
	for in, want := range map[PointerShape]string{
		"text":               "\x1b]22;text\x1b\\",
		"Not-Allowed":        "\x1b]22;not-allowed\x1b\\",
		"x\x1b\\\x1b]0;evil": "\x1b]22;xevil\x1b\\",
		"":                   "\x1b]22;default\x1b\\",
		"123":                "\x1b]22;default\x1b\\",
	} {
		if got := pointerSeq(in); got != want {
			t.Errorf("pointerSeq(%q) = %q; want %q", in, got, want)
		}
	}
}

func TestViewStatePointerIsAppliedOnce(t *testing.T) {
	out := &bytes.Buffer{}
	shape := PointerText
	p := NewProgram(pointerModel{shape: &shape}, WithOutput(out), WithAltScreen(false))
	p.width, p.height = 20, 5
	p.render()
	p.render()
	if n := strings.Count(out.String(), "\x1b]22;text\x1b\\"); n != 1 {
		t.Fatalf("the pointer shape was written %d times: %q", n, out.String())
	}
	shape = PointerWait
	p.model = pointerModel{shape: &shape}
	p.render()
	if !strings.Contains(out.String(), "\x1b]22;wait\x1b\\") {
		t.Fatal("a changed shape was not written")
	}
}
