package tui

import (
	"bytes"
	"math/rand"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/vtscreen"
)

// patchView builds a view of styled ASCII rows, sometimes with the bytes the
// patch path refuses (spaces at the end, non-ASCII, tabs) so both paths run.
func patchView(rng *rand.Rand, rows, cols int, base []string) []string {
	out := make([]string, rows)
	for r := range out {
		if base != nil && rng.Intn(3) != 0 {
			// Mutate a few printable bytes of the previous row outside escapes.
			b := []byte(base[r])
			for m := rng.Intn(6); m > 0; m-- {
				k := rng.Intn(len(b))
				if inEscape(b, k) || b[k] < 0x21 || b[k] > 0x7e {
					continue
				}
				b[k] = byte('!' + rng.Intn(90))
			}
			out[r] = string(b)
			continue
		}
		var sb strings.Builder
		n := 0
		for n < cols {
			seg := strings.Repeat(string(rune('a'+rng.Intn(26))), 1+rng.Intn(6))
			if rng.Intn(9) == 0 {
				seg += " "
			}
			if rng.Intn(25) == 0 {
				seg = "café"
			}
			if rng.Intn(40) == 0 {
				seg = "a\tb"
			}
			if n+len(seg) > cols {
				break
			}
			if rng.Intn(3) == 0 {
				sb.WriteString("\x1b[3" + string(rune('1'+rng.Intn(6))) + "m" + seg + "\x1b[0m")
			} else {
				sb.WriteString(seg)
			}
			n += len(seg)
		}
		out[r] = sb.String()
	}
	return out
}

// inEscape reports whether byte k of b lies inside an escape sequence.
func inEscape(b []byte, k int) bool {
	for i := 0; i < len(b) && i <= k; i++ {
		if b[i] != 0x1b {
			continue
		}
		j := i + 1
		if j < len(b) && b[j] == '[' {
			for j++; j < len(b) && !(b[j] >= 0x40 && b[j] <= 0x7e); j++ {
			}
		}
		if k <= j {
			return true
		}
		i = j
	}
	return false
}

// TestCellPatchDrawsTheScreenOfAFullParse: patching rows from the previous
// frame and copying the changed spans from the new line must leave the
// emulated screen, every cell with its attributes and the cursor, exactly as
// parsing every row does, frame after frame, and the fast path must actually
// be taken on the rows it is meant for.
func TestCellPatchDrawsTheScreenOfAFullParse(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	const rows, cols = 12, 60
	var bufA, bufB bytes.Buffer
	mk := func(buf *bytes.Buffer, noPatch bool) *Program {
		p := NewProgram(staticModel{}, WithOutput(buf), WithColorProfile(ansi.TrueColor), WithCellRenderer(true), WithAltScreen(false))
		p.width, p.height = cols, 100
		p.render()
		p.cells.SetNoPatch(noPatch)
		return p
	}
	a, b := mk(&bufA, false), mk(&bufB, true)
	bufA.Reset()
	bufB.Reset()
	screenA, screenB := vtscreen.NewScreen(cols, 100), vtscreen.NewScreen(cols, 100)
	var base []string
	patchedRows := 0
	for frame := 0; frame < 400; frame++ {
		lines := patchView(rng, rows, cols, base)
		base = lines
		v := strings.Join(lines, "\n")
		a.model, b.model = staticModel{view: v}, staticModel{view: v}
		a.render()
		b.render()
		screenA.Write(bufA.Bytes())
		screenB.Write(bufB.Bytes())
		for y := 0; y < rows+2; y++ {
			for x := 0; x < cols; x++ {
				if ca, cb := screenA.Cell(x, y), screenB.Cell(x, y); ca != cb {
					t.Fatalf("frame %d cell (%d,%d): patched %+v, parsed %+v\nrow: %q", frame, x, y, ca, cb, lines[min(y, rows-1)])
				}
			}
		}
		if ax, ay, av := screenA.Cursor(); true {
			if bx, by, bv := screenB.Cursor(); ax != bx || ay != by || av != bv {
				t.Fatalf("frame %d cursor: patched %d,%d,%v parsed %d,%d,%v", frame, ax, ay, av, bx, by, bv)
			}
		}
		for _, ok := range a.cells.PatchedRows()[:rows] {
			if ok {
				patchedRows++
			}
		}
		bufA.Reset()
		bufB.Reset()
	}
	if patchedRows < 200 {
		t.Fatalf("only %d rows took the patch path; the test is not exercising it", patchedRows)
	}
}
