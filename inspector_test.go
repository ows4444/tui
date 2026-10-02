package tui

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui/internal/vtscreen"
	"github.com/ows4444/tui/layout"
)

// inspModel draws two named panes and reports them to the inspector.
type inspModel struct{ keys *[]string }

func (m inspModel) Init() Cmd { return nil }
func (m inspModel) Update(msg Msg) (Model, Cmd) {
	if k, ok := msg.(Key); ok {
		*m.keys = append(*m.keys, k.String())
	}
	return m, nil
}
func (m inspModel) tree() layout.Node {
	return layout.Row(1,
		layout.FlexChild{Node: layout.Named("sidebar", layout.Block("nav")), Basis: 6},
		layout.FlexChild{Node: layout.Named("content", layout.Block("body")), Grow: 1},
	)
}
func (m inspModel) View() string {
	return layout.Draw(m.tree(), layout.Tight(layout.Size{W: 60, H: 10}))
}
func (m inspModel) InspectLayout() layout.Node { return m.tree() }
func (m inspModel) FocusedID() string          { return "content" }

// screenOf replays the bytes the Program wrote through the VT emulator.
func screenOf(out *unsyncBuffer, w, h int) string {
	s := vtscreen.NewScreen(w, h)
	s.Write(out.b)
	return strings.Join(s.Lines(), "\n")
}

// Criterion #58: the inspector key overlays the named-rect tree, the focused
// id and the last frame's size and time, and the next press removes it.
func TestInspectorOverlaysAndTogglesOff(t *testing.T) {
	pr, pw := io.Pipe()
	defer pw.Close()
	out := &unsyncBuffer{}
	keys := []string{}
	p := NewProgram(inspModel{keys: &keys}, WithInput(pr), WithOutput(out), WithInspector(InspectorKeys{Panel: "f12", Layout: InspectorOff, Messages: InspectorOff, Model: InspectorOff}))
	errc := make(chan error, 1)
	go func() { _, err := p.Run(); errc <- err }()
	p.Send(ResizeMsg{Width: 60, Height: 10})

	snapshot := func() string {
		time.Sleep(150 * time.Millisecond)
		p.outMu.Lock()
		defer p.outMu.Unlock()
		return screenOf(out, 60, 10)
	}
	if got := snapshot(); strings.Contains(got, "inspector") {
		t.Fatalf("overlay shown before the key:\n%s", got)
	}
	_, _ = io.WriteString(pw, "\x1b[24~") // F12
	on := snapshot()
	for _, want := range []string{"inspector (f12 closes)", "focus: content", "sidebar 0,0 6x", "content 7,0", "last frame:", "bytes"} {
		if !strings.Contains(on, want) {
			t.Errorf("overlay missing %q:\n%s", want, on)
		}
	}
	_, _ = io.WriteString(pw, "\x1b[24~")
	if off := snapshot(); strings.Contains(off, "inspector") {
		t.Fatalf("overlay still shown after the second press:\n%s", off)
	}
	p.Send(QuitMsg{})
	select {
	case <-errc:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return")
	}
	if len(keys) != 0 {
		t.Fatalf("the inspector key reached Update: %v", keys)
	}
}

func TestInspectorOffWithoutTheOption(t *testing.T) {
	pr, pw := io.Pipe()
	defer pw.Close()
	out := &unsyncBuffer{}
	keys := []string{}
	p := NewProgram(inspModel{keys: &keys}, WithInput(pr), WithOutput(out))
	errc := make(chan error, 1)
	go func() { _, err := p.Run(); errc <- err }()
	p.Send(ResizeMsg{Width: 60, Height: 10})
	_, _ = io.WriteString(pw, "\x1b[24~")
	time.Sleep(150 * time.Millisecond)
	p.outMu.Lock()
	got := screenOf(out, 60, 10)
	p.outMu.Unlock()
	if strings.Contains(got, "inspector") {
		t.Fatalf("overlay drawn without WithInspector:\n%s", got)
	}
	p.Send(QuitMsg{})
	<-errc
	if len(keys) != 1 || keys[0] != "f12" {
		t.Fatalf("without the option F12 must reach Update as \"f12\"; got %v", keys)
	}
}

// outlineModel names three panes wide enough for their labels to fit; each
// pane's content starts on its second row and column, inside the outline.
type outlineModel struct{ keys *[]string }

func (m outlineModel) Init() Cmd { return nil }
func (m outlineModel) Update(msg Msg) (Model, Cmd) {
	if k, ok := msg.(Key); ok {
		*m.keys = append(*m.keys, k.String())
	}
	return m, nil
}
func (m outlineModel) tree() layout.Node {
	return layout.Column(0,
		layout.FlexChild{Node: layout.Named("header", layout.Block("\n title")), Basis: 3},
		layout.FlexChild{Grow: 1, Node: layout.Row(0,
			layout.FlexChild{Node: layout.Named("sidebar", layout.Block("\n nav")), Basis: 20},
			layout.FlexChild{Node: layout.Named("content", layout.Block("\n body text")), Grow: 1},
		)},
	)
}
func (m outlineModel) View() string {
	return layout.Draw(m.tree(), layout.Tight(layout.Size{W: 60, H: 10}))
}
func (m outlineModel) InspectLayout() layout.Node { return m.tree() }

// Criterion #59: with the layout overlay on, every named node is outlined with
// its name and size, the content stays visible, and the key toggles it off.
func TestInspectorLayoutOutlinesEveryNamedNode(t *testing.T) {
	pr, pw := io.Pipe()
	defer pw.Close()
	out := &unsyncBuffer{}
	keys := []string{}
	p := NewProgram(outlineModel{keys: &keys}, WithInput(pr), WithOutput(out), WithInspector(InspectorKeys{Panel: InspectorOff, Messages: InspectorOff, Model: InspectorOff}))
	errc := make(chan error, 1)
	go func() { _, err := p.Run(); errc <- err }()
	p.Send(ResizeMsg{Width: 60, Height: 10})
	snapshot := func() string {
		time.Sleep(150 * time.Millisecond)
		p.outMu.Lock()
		defer p.outMu.Unlock()
		return screenOf(out, 60, 10)
	}
	if got := snapshot(); strings.Contains(got, "header 60x3") {
		t.Fatalf("outlines shown before the key:\n%s", got)
	}
	_, _ = io.WriteString(pw, "\x1b[23~") // F11
	on := snapshot()
	for _, want := range []string{"+header 60x3", "+sidebar 20x7", "+content 40x7", "title", "nav", "body text"} {
		if !strings.Contains(on, want) {
			t.Errorf("overlay missing %q:\n%s", want, on)
		}
	}
	lines := strings.Split(on, "\n")
	if !strings.HasPrefix(lines[3], "+sidebar 20x7") || !strings.HasPrefix(lines[9], "+---") || lines[4][0] != '|' || lines[4][19] != '|' {
		t.Errorf("sidebar box is not outlined at its rectangle:\n%s", on)
	}
	_, _ = io.WriteString(pw, "\x1b[23~")
	if off := snapshot(); strings.Contains(off, "sidebar 20x7") {
		t.Fatalf("outlines still shown after the second press:\n%s", off)
	}
	p.Send(QuitMsg{})
	select {
	case <-errc:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return")
	}
	if len(keys) != 0 {
		t.Fatalf("the layout key reached Update: %v", keys)
	}
}

func TestInspectorLayoutOutlineEdgeCases(t *testing.T) {
	p := NewProgram(outlineModel{})
	view := strings.Repeat("..........\n", 3) + ".........."
	got := p.outline(view, layout.Rect{X: 1, Y: 0, W: 1, H: 3}, "n")
	if want := ".|........\n.|........\n.|........\n.........."; got != want {
		t.Errorf("1-wide:\n%s\nwant\n%s", got, want)
	}
	got = p.outline(view, layout.Rect{X: 0, Y: 1, W: 6, H: 1}, "abcdefgh")
	if want := "..........\n+abcd+....\n..........\n.........."; got != want {
		t.Errorf("1-tall:\n%s\nwant\n%s", got, want)
	}
}
