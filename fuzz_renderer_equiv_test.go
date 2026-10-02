package tui

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/vtscreen"
)

// FuzzRendererEquivalence drives the cell renderer and the line renderer with
// the same sequence of random styled views and asserts that the two screens
// (rune, attributes and colours of every cell, plus the cursor row) are equal
// after every frame.
//
// Input encoding: data[0] picks the terminal width (8..47); every later byte
// is an instruction: one of the fuzzTokens (text or an SGR), a row break
// (255), or a frame break (254). Content is ASCII plus one precomposed rune,
// because internal/vtscreen models one column per rune; it never emits a tab
// or a control byte, but "\x1b[2C" appears so the per-row fallback is hit.
//
// Smoke run:  go test -run xxx -fuzz FuzzRendererEquivalence -fuzztime 60s
// The criterion's 10-minute run is meant for a scheduled job; no fuzz
// workflow exists in .github/workflows, so it is not wired into CI here.
var fuzzTokens = []string{
	"a", "b", "c", "x", "Z", "1", " ", " ", " ", "é", "~", "hello", "  ",
	"\x1b[1m", "\x1b[3m", "\x1b[4m", "\x1b[4:3m", "\x1b[31m", "\x1b[38;5;123m", "\x1b[38;2;10;200;30m",
	"\x1b[44m", "\x1b[48;5;9m", "\x1b[92m", "\x1b[22m", "\x1b[39m", "\x1b[49m",
	"\x1b[0m", "\x1b[1;4;33m", "\x1b[7m", "\x1b[m", "\x1b[48;2;1;2;3m", "\x1b[9m", "\x1b[2m",
	"\x1b[2C", // not representable in the grid: the row falls back
}

func fuzzDecode(data []byte) (width int, frames [][]string) {
	if len(data) == 0 {
		return 20, nil
	}
	width = 8 + int(data[0])%40
	var rows []string
	var sb strings.Builder
	used := 0
	styled := false
	endRow := func() {
		if styled { // rows close their styles: the cell renderer deliberately does not let one bleed into the next row
			sb.WriteString("\x1b[0m")
			styled = false
		}
		rows = append(rows, sb.String())
		sb.Reset()
		used = 0
	}
	endFrame := func() {
		endRow()
		frames = append(frames, rows)
		rows = nil
	}
	for _, b := range data[1:] {
		switch {
		case b == 255:
			if len(rows) < 11 {
				endRow()
			}
		case b == 254:
			endFrame()
		default:
			tok := fuzzTokens[int(b)%len(fuzzTokens)]
			if tok == "\x1b[2C" {
				if used+2 > width {
					continue
				}
				used += 2
			} else if !strings.HasPrefix(tok, "\x1b") {
				w := ansi.Width(tok)
				if used+w > width {
					continue
				}
				used += w
			}
			if strings.HasPrefix(tok, "\x1b[") && tok != "\x1b[2C" {
				styled = true
			}
			sb.WriteString(tok)
		}
	}
	endFrame()
	return width, frames
}

func fuzzCompare(a, b *vtscreen.Screen, rows int) string {
	for y := 0; y < rows; y++ {
		for x := 0; ; x++ {
			ca, cb := a.Cell(x, y), b.Cell(x, y)
			if !reflect.DeepEqual(ca, cb) {
				return fmt.Sprintf("cell (%d,%d): %+v vs %+v", x, y, ca, cb)
			}
			if x >= 80 {
				break
			}
		}
	}
	_, ay, _ := a.Cursor()
	_, by, _ := b.Cursor()
	if ay != by {
		return fmt.Sprintf("cursor row %d vs %d", ay, by)
	}
	return ""
}

func FuzzRendererEquivalence(f *testing.F) {
	// Seeds: a few plain frames, style-heavy frames, near-identical successive
	// frames (diff and patch paths), shrinking/growing views and a fallback row.
	f.Add([]byte{12, 0, 1, 2, 255, 3, 4, 254, 0, 1, 2, 255, 3, 5})
	f.Add([]byte{20, 13, 0, 1, 26, 255, 17, 2, 3, 26, 254, 13, 0, 1, 26, 255, 17, 2, 4, 26})
	f.Add([]byte{8, 18, 0, 26, 19, 1, 26, 20, 2, 26, 255, 21, 3, 26, 254, 22, 0, 26})
	f.Add([]byte{30, 0, 1, 2, 3, 4, 255, 5, 6, 7, 255, 8, 9, 254, 0, 1, 255, 254, 0, 1, 2, 3, 4, 255, 5, 6, 7, 255, 8, 9, 255, 1})
	f.Add([]byte{25, 0, 32, 1, 255, 2, 254, 0, 32, 5, 255, 2, 254, 3, 1})
	f.Add([]byte{40, 11, 6, 6, 11, 255, 11, 254, 11, 6, 11, 255, 12, 12, 254})
	f.Add([]byte{16, 14, 9, 26, 255, 15, 9, 26, 254, 14, 9, 26, 255, 28, 9, 26, 254})
	f.Fuzz(func(t *testing.T, data []byte) {
		width, frames := fuzzDecode(data)
		if len(frames) > 24 {
			frames = frames[:24]
		}
		var lineOut, cellOut bytes.Buffer
		lp, cp := equivProgram(&lineOut, width, false), equivProgram(&cellOut, width, true)
		const screenRows = 100
		ls, cs := vtscreen.NewScreen(width, screenRows), vtscreen.NewScreen(width, screenRows)
		region := 0
		for fi, view := range frames {
			if len(view) > region {
				region = len(view)
			}
			lp.model = staticModel{view: strings.Join(view, "\n")}
			cp.model = lp.model
			lp.render()
			cp.render()
			ls.Write(lineOut.Bytes())
			cs.Write(cellOut.Bytes())
			lineOut.Reset()
			cellOut.Reset()
			if d := fuzzCompare(cs, ls, region+1); d != "" {
				t.Fatalf("frame %d width %d: cell vs line renderer: %s\nview %q", fi, width, d, view)
			}
		}
	})
}
