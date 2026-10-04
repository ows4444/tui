package tuitest_test

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/internal/termio"
	"github.com/ows4444/tui/tuitest"
)

// fill draws the size it was told as a full block: h rows of w cells, the
// last cell of every row a different rune so a clipped row shows.
type fill struct{ w, h int }

func (m fill) Init() tui.Cmd { return nil }
func (m fill) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	if r, ok := msg.(tui.ResizeMsg); ok {
		m.w, m.h = r.Width, r.Height
	}
	return m, nil
}
func (m fill) View() string {
	if m.w < 1 || m.h < 1 {
		return ""
	}
	row := strings.Repeat("x", m.w-1) + "|"
	return strings.TrimSuffix(strings.Repeat(row+"\n", m.h), "\n")
}

// wantFull fails unless the screen is h rows, each w cells ending in "|".
func wantFull(t *testing.T, s *tuitest.Session, w, h int) {
	t.Helper()
	got := s.Screen()
	if len(got) != h {
		t.Fatalf("screen has %d rows, want %d", len(got), h)
	}
	want := strings.Repeat("x", w-1) + "|"
	for i, row := range got {
		if row != want {
			t.Fatalf("row %d is %d cells %q..., want %d cells ending in |", i, len(row), row[:min(len(row), 12)], w)
		}
	}
}

// A session larger than the 80x24 a plain writer gets must draw frames of its
// own size: every row and every column the model drew reaches the screen.
func TestNewDrawsFramesAtTheSessionSize(t *testing.T) {
	for _, size := range [][2]int{{30, 6}, {80, 24}, {100, 30}, {140, 40}, {160, 50}} {
		w, h := size[0], size[1]
		s := tuitest.New(fill{}, w, h)
		wantFull(t, s, w, h)
		s.Close()
	}
}

func TestResizeChangesTheFrameSize(t *testing.T) {
	s := tuitest.New(fill{}, 60, 20)
	t.Cleanup(s.Close)
	wantFull(t, s, 60, 20)
	s.Resize(160, 50)
	wantFull(t, s, 160, 50)
	s.Resize(50, 15)
	wantFull(t, s, 50, 15)
	s.Resize(100, 30)
	wantFull(t, s, 100, 30)
}

// counter counts the ResizeMsgs it receives and shows the count and the size.
type counter struct{ n, w, h int }

func (m counter) Init() tui.Cmd { return nil }
func (m counter) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	if r, ok := msg.(tui.ResizeMsg); ok {
		m.n, m.w, m.h = m.n+1, r.Width, r.Height
	}
	return m, nil
}
func (m counter) View() string { return "n:" + itoa(m.n) + " size:" + itoa(m.w) + "x" + itoa(m.h) }

// One Resize is one ResizeMsg for the model, carrying the new size.
func TestResizeDeliversOneResizeMsg(t *testing.T) {
	s := tuitest.New(counter{}, 100, 30)
	t.Cleanup(s.Close)
	before := s.Screen()[0]
	if !strings.HasSuffix(before, "size:100x30") {
		t.Fatalf("first frame = %q, want the session size", before)
	}
	s.Resize(120, 35)
	after := s.Screen()[0]
	var n0, n1 int
	for _, c := range strings.TrimPrefix(strings.Fields(before)[0], "n:") {
		n0 = n0*10 + int(c-'0')
	}
	for _, c := range strings.TrimPrefix(strings.Fields(after)[0], "n:") {
		n1 = n1*10 + int(c-'0')
	}
	if n1 != n0+1 || !strings.HasSuffix(after, "size:120x35") {
		t.Fatalf("after Resize = %q (before %q): want one more ResizeMsg and size 120x35", after, before)
	}
}

// A test that brings its own terminal keeps control of the size: the Program
// follows that terminal, as it does outside tuitest.
func TestOwnTerminalDecidesTheSize(t *testing.T) {
	own := &termio.Fake{W: 40, H: 5, SizeKnown: true}
	s := tuitest.New(counter{}, 100, 30, tui.WithTerminal(own))
	t.Cleanup(s.Close)
	own.SetSize(44, 6)
	s.Resize(120, 35)
	if got := s.Screen()[0]; !strings.HasSuffix(got, "size:44x6") {
		t.Fatalf("after Resize with an own terminal = %q, want the size that terminal reports", got)
	}
}
