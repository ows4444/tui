package tui

import (
	"testing"

	"github.com/ows4444/tui/keymap"
)

type bindModel struct{ key string }

func (m bindModel) Init() Cmd               { return nil }
func (m bindModel) Update(Msg) (Model, Cmd) { return m, nil }
func (m bindModel) View() string            { return "x" }
func (m bindModel) Bindings() []keymap.Binding {
	return []keymap.Binding{keymap.NewBinding("act", m.key)}
}

// TestKeymapTracksTheModelAcrossFrames: the registry is taken from the model
// as of the last frame, so a change of focus or KeyMap shows after the next
// render with no other bookkeeping.
func TestKeymapTracksTheModelAcrossFrames(t *testing.T) {
	out, _ := captureOutput(t)
	p := NewProgram(bindModel{key: "a"}, WithOutput(out))
	if got := p.Keymap().Hints(""); len(got) != 1 || got[0].Key != "a" {
		t.Fatalf("before any frame: %v", got)
	}
	p.model = bindModel{key: "b"}
	if got := p.Keymap().Hints(""); got[0].Key != "a" {
		t.Fatalf("Keymap changed without a frame: %v", got)
	}
	p.render()
	if got := p.Keymap().Hints(""); len(got) != 1 || got[0].Key != "b" {
		t.Fatalf("after a frame: %v", got)
	}
}
