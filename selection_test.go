package tui

import (
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/input"
)

func TestWordSpan(t *testing.T) {
	for _, c := range []struct {
		text     string
		col      int
		from, to int
	}{
		{"alpha beta", 1, 0, 5},
		{"alpha beta", 6, 6, 10},
		{"alpha beta", 5, 5, 6}, // the blank
		{"a.b", 1, 1, 2},
		{"日本 x", 1, 0, 4}, // a wide rune covers two columns
		{"日本 x", 3, 0, 4},
		{"ab", 9, 9, 10}, // past the end
	} {
		if from, to := wordSpan(c.text, c.col); from != c.from || to != c.to {
			t.Errorf("wordSpan(%q, %d) = %d, %d; want %d, %d", c.text, c.col, from, to, c.from, c.to)
		}
	}
}

func TestSelectionSpanIsInReadingOrder(t *testing.T) {
	s := selection{ar: 3, ac: 5, cr: 1, cc: 2}
	if r0, c0, r1, c1 := s.span(); r0 != 1 || c0 != 2 || r1 != 3 || c1 != 5 {
		t.Fatalf("span = %d,%d to %d,%d", r0, c0, r1, c1)
	}
	s = selection{ar: 2, ac: 7, cr: 2, cc: 3}
	if r0, c0, r1, c1 := s.span(); r0 != 2 || c0 != 3 || r1 != 2 || c1 != 7 {
		t.Fatalf("span = %d,%d to %d,%d", r0, c0, r1, c1)
	}
}

func TestPaintSelectionKeepsWidthAndStyle(t *testing.T) {
	p := &Program{selOn: true, selStyle: ansi.NewStyle().Reverse()}
	p.sel = selection{on: true, ar: 0, ac: 2, cr: 1, cc: 1}
	view := ansi.NewStyle().Bold().Render("hello") + "\nwor日ld"
	out := strings.Split(p.paintSelection(view), "\n")
	for i, l := range strings.Split(view, "\n") {
		if ansi.Width(out[i]) != ansi.Width(l) {
			t.Errorf("row %d is %d wide after painting; was %d", i, ansi.Width(out[i]), ansi.Width(l))
		}
		if ansi.StripANSI(out[i]) != ansi.StripANSI(l) {
			t.Errorf("row %d text changed: %q", i, ansi.StripANSI(out[i]))
		}
	}
	if !strings.Contains(out[0], "\x1b[7m") || !strings.Contains(out[1], "\x1b[7m") {
		t.Errorf("rows are not reversed: %q", out)
	}
	// A wide rune cut by the end of the selection keeps the columns aligned.
	p.sel = selection{on: true, ar: 1, ac: 0, cr: 1, cc: 3}
	row := strings.Split(p.paintSelection(view), "\n")[1]
	if ansi.Width(row) != ansi.Width("wor日ld") {
		t.Errorf("a cut wide rune moved the row to %d columns: %q", ansi.Width(row), row)
	}
}

type selModel struct {
	mu   *sync.Mutex
	got  *[]Msg
	text string
}

func (selModel) Init() Cmd { return nil }
func (m selModel) Update(msg Msg) (Model, Cmd) {
	switch msg.(type) {
	case SelectionMsg, MouseEvent, Key:
		m.mu.Lock()
		*m.got = append(*m.got, msg)
		m.mu.Unlock()
	}
	return m, nil
}
func (m selModel) View() string { return m.text }

// runSelection starts a Program with text selection on and returns what the
// model received and what was written.
func runSelection(t *testing.T) (p *Program, out *unsyncBuffer, got func() []Msg, stop func()) {
	t.Helper()
	pr, pw := io.Pipe()
	out = &unsyncBuffer{}
	var mu sync.Mutex
	var msgs []Msg
	p = NewProgram(selModel{mu: &mu, got: &msgs, text: "alpha beta\ngamma delta"},
		WithInput(pr), WithOutput(out), WithMouse(MouseCellMotion), WithAltScreen(true), WithTextSelection(ansi.Style{}))
	errc := make(chan error, 1)
	go func() { _, err := p.Run(); errc <- err }()
	p.Send(ResizeMsg{Width: 20, Height: 5})
	waitFor(t, func() bool { return strings.Contains(string(outBytes(p, out)), "gamma") })
	return p, out, func() []Msg {
			mu.Lock()
			defer mu.Unlock()
			return append([]Msg(nil), msgs...)
		}, func() {
			p.Quit()
			<-errc
			_ = pw.Close()
		}
}

func outBytes(p *Program, out *unsyncBuffer) []byte {
	p.outMu.Lock()
	defer p.outMu.Unlock()
	return append([]byte(nil), out.b...)
}

func altMouse(action MouseAction, x, y int) MouseEvent {
	return MouseEvent{X: x, Y: y, Button: MouseButtonLeft, Action: action, Mod: input.ModAlt}
}

func TestAltDragSelectsAndCopies(t *testing.T) {
	p, out, got, stop := runSelection(t)
	defer stop()
	mark := len(outBytes(p, out))
	p.Send(altMouse(MouseActionPress, 6, 0))
	p.Send(altMouse(MouseActionMotion, 4, 1))
	p.Send(altMouse(MouseActionRelease, 4, 1))
	waitFor(t, func() bool {
		for _, m := range got() {
			if _, ok := m.(SelectionMsg); ok {
				return true
			}
		}
		return false
	})
	var sel SelectionMsg
	for _, m := range got() {
		switch m := m.(type) {
		case SelectionMsg:
			sel = m
		case MouseEvent:
			t.Errorf("the model got a selection mouse event: %+v", m)
		}
	}
	if sel.Text != "beta\ngamma" {
		t.Fatalf("selected %q; want %q", sel.Text, "beta\ngamma")
	}
	written := string(outBytes(p, out)[mark:])
	if !strings.Contains(written, ansi.OSC52Copy("beta\ngamma")) {
		t.Errorf("the text was not copied with OSC 52: %q", written)
	}
	if !strings.Contains(written, "\x1b[7m") {
		t.Errorf("the selection was not drawn: %q", written)
	}
}

func TestSelectionClearsOnKeyAndPlainClicksPassThrough(t *testing.T) {
	p, out, got, stop := runSelection(t)
	defer stop()
	p.Send(altMouse(MouseActionPress, 0, 0))
	p.Send(altMouse(MouseActionMotion, 4, 0))
	p.Send(altMouse(MouseActionRelease, 4, 0))
	waitFor(t, func() bool { return strings.Contains(string(outBytes(p, out)), "\x1b[7m") })
	// The frame that draws the selection may still be arriving: wait until the
	// output has stopped growing, so what is checked below is the key's own frame.
	for last, quiet := -1, 0; quiet < 5; time.Sleep(10 * time.Millisecond) {
		if n := len(outBytes(p, out)); n == last {
			quiet++
		} else {
			last, quiet = n, 0
		}
	}
	mark := len(outBytes(p, out))
	p.Send(Key{Type: KeyRunes, Text: "x"})
	waitFor(t, func() bool { return len(outBytes(p, out)) > mark })
	if after := string(outBytes(p, out)[mark:]); strings.Contains(after, "\x1b[7m") {
		t.Errorf("a key left the selection drawn: %q", after)
	}
	plain := MouseEvent{X: 1, Y: 1, Button: MouseButtonLeft, Action: MouseActionPress}
	p.Send(plain)
	waitFor(t, func() bool {
		for _, m := range got() {
			if ev, ok := m.(MouseEvent); ok && ev.X == 1 && ev.Y == 1 {
				return true
			}
		}
		return false
	})
}

func TestDoubleClickSelectsAWordAndTripleARow(t *testing.T) {
	p, _, got, stop := runSelection(t)
	defer stop()
	selected := func() (texts []string) {
		for _, m := range got() {
			if s, ok := m.(SelectionMsg); ok {
				texts = append(texts, s.Text)
			}
		}
		return texts
	}
	// Three quick presses on one cell count as a single, double and triple click.
	// The first selects nothing (a bare click), so two selections are reported.
	for i := 0; i < 3; i++ {
		p.Send(altMouse(MouseActionPress, 8, 1))
		p.Send(altMouse(MouseActionRelease, 8, 1))
	}
	waitFor(t, func() bool { return len(selected()) == 2 })
	if got := selected(); got[0] != "delta" || got[1] != "gamma delta" {
		t.Fatalf("selected %q; want a word and then the row", got)
	}
}
