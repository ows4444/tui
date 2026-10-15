package tui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
)

// stateModel states its ViewState from fields the test changes.
type stateModel struct {
	alt   *bool
	mouse *MouseMode
	title *string
}

func (m stateModel) Init() Cmd               { return nil }
func (m stateModel) Update(Msg) (Model, Cmd) { return m, nil }
func (m stateModel) View() string            { return "hello" }
func (m stateModel) ViewState() ViewState {
	return ViewState{AltScreen: m.alt, Mouse: m.mouse, Title: m.title}
}

func stateProgram(m stateModel) (*Program, *bytes.Buffer) {
	out := &bytes.Buffer{}
	p := NewProgram(m, WithOutput(out), WithAltScreen(false))
	p.width, p.height = 20, 5
	return p, out
}

func TestViewStateEntersAndLeavesTheAltScreen(t *testing.T) {
	on := true
	p, out := stateProgram(stateModel{alt: &on})
	p.render()
	if !strings.Contains(out.String(), ansi.AltScreenEnable) {
		t.Fatalf("alt screen not entered:\n%q", out.String())
	}
	out.Reset()
	p.render() // already there: nothing written to switch
	if strings.Contains(out.String(), ansi.AltScreenEnable) {
		t.Errorf("alt screen entered again:\n%q", out.String())
	}
	off := false
	p.model = stateModel{alt: &off}
	out.Reset()
	p.render()
	if !strings.Contains(out.String(), ansi.AltScreenDisable) {
		t.Errorf("alt screen not left:\n%q", out.String())
	}
}

func TestViewStateNilFieldsHaveNoOpinion(t *testing.T) {
	p, out := stateProgram(stateModel{})
	p.render()
	for _, s := range []string{ansi.AltScreenEnable, ansi.AltScreenDisable, "\x1b]0;", ansi.MouseSGREnable} {
		if strings.Contains(out.String(), s) {
			t.Errorf("a nil ViewState wrote %q", s)
		}
	}
}

func TestViewStateSetsMouseAndTitleOncePerChange(t *testing.T) {
	mode, title := MouseCellMotion, "editing a.go"
	p, out := stateProgram(stateModel{mouse: &mode, title: &title})
	p.render()
	got := out.String()
	if !strings.Contains(got, ansi.MouseSGREnable) || !strings.Contains(got, "\x1b]0;editing a.go\x07") {
		t.Fatalf("mouse and title not set:\n%q", got)
	}
	out.Reset()
	p.render()
	if strings.Contains(out.String(), "\x1b]0;") || strings.Contains(out.String(), ansi.MouseSGREnable) {
		t.Errorf("an unchanged state was written again:\n%q", out.String())
	}
	next := "done"
	p.model = stateModel{mouse: &mode, title: &next}
	out.Reset()
	p.render()
	if !strings.Contains(out.String(), "\x1b]0;done\x07") {
		t.Errorf("a changed title was not written:\n%q", out.String())
	}
}

func TestViewStateIsFoundThroughAnUnwrapper(t *testing.T) {
	on := true
	out := &bytes.Buffer{}
	p := NewProgram(wrapper{stateModel{alt: &on}}, WithOutput(out), WithAltScreen(false))
	p.width, p.height = 20, 5
	p.render()
	if !strings.Contains(out.String(), ansi.AltScreenEnable) {
		t.Errorf("not found through Unwrap:\n%q", out.String())
	}
}

type wrapper struct{ inner Model }

func (w wrapper) Init() Cmd               { return nil }
func (w wrapper) Update(Msg) (Model, Cmd) { return w, nil }
func (w wrapper) View() string            { return w.inner.View() }
func (w wrapper) Unwrap() Model           { return w.inner }

func TestViewStateIgnoredInAccessibleMode(t *testing.T) {
	on := true
	out := &bytes.Buffer{}
	p := NewProgram(stateModel{alt: &on}, WithOutput(out), WithAltScreen(false), WithAccessible(true))
	p.width, p.height = 20, 5
	p.render()
	if strings.Contains(out.String(), ansi.AltScreenEnable) {
		t.Errorf("accessible mode entered the alt screen:\n%q", out.String())
	}
}
