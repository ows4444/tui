package tui

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
)

// tallVT is a minimal scrolling terminal: fixed height, CRLF at the bottom
// scrolls the top row into history, CursorUp clamps at the top of the screen.
type tallVT struct {
	h       int
	rows    []string // visible screen, len h
	history []string
	row     int
	col     int
	maxUp   int // largest requested CursorUp
}

func newTallVT(h int) *tallVT { return &tallVT{h: h, rows: make([]string, h)} }

func (v *tallVT) tfeed(data string) {
	for i := 0; i < len(data); {
		c := data[i]
		switch {
		case c == 0x1b && i+1 < len(data) && data[i+1] == '[':
			j := i + 2
			for j < len(data) && !(data[j] >= 'A' && data[j] <= 'Z' || data[j] >= 'a' && data[j] <= 'z') {
				j++
			}
			n, _ := strconv.Atoi(data[i+2 : j])
			switch data[j] {
			case 'A':
				if n > v.maxUp {
					v.maxUp = n
				}
				v.row -= n
				if v.row < 0 {
					v.row = 0
				}
			case 'B':
				v.row = min(v.row+max(n, 1), v.h-1)
			case 'C':
				v.col += max(n, 1)
			case 'D':
				v.col = max(v.col-max(n, 1), 0)
			case 'K':
				v.rows[v.row] = v.rows[v.row][:min(v.col, len(v.rows[v.row]))]
			case 'J':
				v.rows[v.row] = ""
				for k := v.row + 1; k < v.h; k++ {
					v.rows[k] = ""
				}
			}
			i = j + 1
		case c == '\r':
			v.col = 0
			i++
		case c == '\n':
			if v.row == v.h-1 {
				v.history = append(v.history, v.rows[0])
				copy(v.rows, v.rows[1:])
				v.rows[v.h-1] = ""
			} else {
				v.row++
			}
			v.col = 0
			i++
		default:
			r := v.rows[v.row]
			for len(r) < v.col {
				r += " "
			}
			if v.col < len(r) {
				r = r[:v.col] + string(c) + r[v.col+1:]
			} else {
				r += string(c)
			}
			v.rows[v.row] = r
			v.col++
			i++
		}
	}
}

func tallView(n int, tag string) string {
	var l []string
	for i := 0; i < n; i++ {
		l = append(l, fmt.Sprintf("line %d", i))
	}
	l[n-1] = tag
	return strings.Join(l, "\n")
}

func tallProgram(t *testing.T, h int, view string) (*Program, func() string) {
	out, read := captureOutput(t)
	p := NewProgram(staticModel{view: view}, WithOutput(out), WithAltScreen(false))
	p.width, p.height = 80, h
	return p, func() string { return string(read()) }
}

func combined(v *tallVT) []string {
	all := append([]string{}, v.history...)
	return append(all, v.rows...)
}

// Criteria #6, #7: overflow committed once; repeated updates leave no ghosts.
func TestInlineTallViewNoGhostRows(t *testing.T) {
	const h = 10
	p, read := tallProgram(t, h, tallView(15, "tag 0"))
	v := newTallVT(h)
	seen := 0
	for k := 0; k < 5; k++ {
		p.model = staticModel{view: tallView(15, fmt.Sprintf("tag %d", k))}
		p.render()
		all := read()
		v.tfeed(all[seen:])
		seen = len(all)
	}
	want := 15 - h
	if len(v.history) != want {
		t.Fatalf("history = %d rows, want %d: %q", len(v.history), want, v.history)
	}
	for i, l := range v.history {
		if l != fmt.Sprintf("line %d", i) {
			t.Fatalf("history[%d] = %q", i, l)
		}
	}
	for i := 0; i < h; i++ {
		w := fmt.Sprintf("line %d", 5+i)
		if i == h-1 {
			w = "tag 4"
		}
		if v.rows[i] != w {
			t.Fatalf("row %d = %q, want %q (screen %q)", i, v.rows[i], w, v.rows)
		}
	}
	if v.maxUp > h-1 {
		t.Fatalf("cursor up %d exceeds live region", v.maxUp)
	}
}

// Growing view commits each overflow row exactly once.
func TestInlineTallViewGrowsCommitsOnce(t *testing.T) {
	const h = 6
	p, read := tallProgram(t, h, "x")
	v := newTallVT(h)
	seen := 0
	for n := 1; n <= 14; n++ {
		p.model = staticModel{view: tallView(n, fmt.Sprintf("tail %d", n))}
		p.render()
		all := read()
		v.tfeed(all[seen:])
		seen = len(all)
	}
	got := combined(v)
	var content []string
	for _, l := range got {
		if l != "" {
			content = append(content, l)
		}
	}
	want := strings.Split(tallView(14, "tail 14"), "\n")
	if strings.Join(content, "|") != strings.Join(want, "|") {
		t.Fatalf("transcript = %q\nwant %q", content, want)
	}
}

// Criterion #8: shrinking below the height redraws without moving past the
// region top.
func TestInlineTallViewShrinkStaysInRegion(t *testing.T) {
	const h = 10
	p, read := tallProgram(t, h, tallView(15, "a"))
	p.render()
	before := len(read())
	p.model = staticModel{view: tallView(4, "b")}
	p.render()
	frame := read()[before:]
	v := newTallVT(h)
	v.tfeed(read()[:before])
	v.maxUp = 0
	v.tfeed(frame)
	if v.maxUp > h-1 {
		t.Fatalf("cursor up %d past live region top", v.maxUp)
	}
	if strings.Contains(frame, "line 0") {
		t.Fatalf("shrink re-emitted committed rows: %q", frame)
	}
}

// Criterion #9: one write per frame, inside the DEC 2026 bracket, including
// frames that commit rows.
func TestInlineTallViewSingleSyncedWrite(t *testing.T) {
	p, read := tallProgram(t, 10, tallView(15, "a"))
	prev := 0
	for k := 0; k < 3; k++ {
		p.model = staticModel{view: tallView(15+k, "z")}
		p.render()
		all := read()
		w := all[prev:]
		prev = len(all)
		if !strings.HasPrefix(w, ansi.SyncOutputEnable) || !strings.HasSuffix(w, ansi.SyncOutputDisable) ||
			strings.Count(w, ansi.SyncOutputEnable) != 1 || strings.Count(w, ansi.SyncOutputDisable) != 1 {
			t.Fatalf("frame %d not a single bracketed write: %q", k, w)
		}
	}
}
