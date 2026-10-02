package testutil

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
)

var updateSizes = flag.Bool("update-sizes", false, "rewrite the size-matrix golden files")

// NoHeightCheck, passed in SizeMatrix's known list, disables the check that
// the frame is no taller than the terminal. It is for inline-mode examples,
// whose frame is not clipped to the window.
const NoHeightCheck = "noheight"

// MatrixSizes are the terminal sizes SizeMatrix renders every model at.
var MatrixSizes = [][2]int{{40, 10}, {80, 24}, {120, 40}, {300, 80}}

func updatingSizes() bool {
	if *updateSizes || os.Getenv("TUITEST_UPDATE") == "1" {
		return true
	}
	f := flag.Lookup("update")
	return f != nil && f.Value.String() == "true"
}

// SizeMatrix renders the model build returns at each of MatrixSizes and
// compares the colour-stripped View with testdata/size-<W>x<H>.golden in the
// calling package's directory. It also fails if the render panics or any line
// is wider than the terminal (measured with ansi.Width).
//
// The render is deterministic by construction: the model only receives a
// ResizeMsg, no Cmd is run (so no clock, spinner or timer advances), and ANSI
// is stripped before comparing, so the colour profile does not matter.
//
// Unless known contains NoHeightCheck, a frame taller than the terminal also
// fails, which is the right contract for alt-screen examples.
//
// known also takes "h40x10"-style entries naming a size at which the frame is
// known to be taller than the terminal; the golden is still written and
// compared, and the subtest fails once the overflow is gone.
//
// known lists sizes ("40x10") at which the model is known to overflow. Those
// subtests are skipped with the measured width, and fail once the overflow is
// gone, so the entry has to be removed and the golden generated. No golden is
// ever written or compared for a frame that overflows.
//
// Regenerate goldens with `go test -update-sizes` (or -update, or
// TUITEST_UPDATE=1).
func SizeMatrix(t *testing.T, build func() tui.Model, known ...string) {
	t.Helper()
	for _, sz := range MatrixSizes {
		w, h := sz[0], sz[1]
		name := fmt.Sprintf("%dx%d", w, h)
		t.Run(name, func(t *testing.T) {
			got := renderAt(t, build, w, h)
			widest := 0
			lines := strings.Split(strings.TrimSuffix(got, "\n"), "\n")
			for _, l := range lines {
				widest = max(widest, ansi.Width(l))
			}
			if slices.Contains(known, name) {
				if widest <= w {
					t.Fatalf("no longer overflows at %s: remove it from the known-overflow list and generate its golden", name)
				}
				t.Logf("frame:\n%s", got)
				t.Skipf("known overflow: widest line is %d, terminal is %d", widest, w)
			}
			if widest > w {
				t.Logf("frame:\n%s", got)
				t.Fatalf("a line is %d wide, terminal is %d", widest, w)
			}
			knownTall := slices.Contains(known, "h"+name)
			switch tall := len(lines) > h; {
			case knownTall && !tall:
				t.Fatalf("no longer taller than the terminal at %s: remove %q from the known list", name, "h"+name)
			case tall && !knownTall && !slices.Contains(known, NoHeightCheck):
				t.Logf("frame:\n%s", got)
				t.Fatalf("frame is %d lines tall, terminal is %d (pass testutil.NoHeightCheck for inline examples that intentionally exceed it, or %q to record a known overflow)", len(lines), h, "h"+name)
			}
			path := filepath.Join("testdata", "size-"+name+".golden")
			if updatingSizes() {
				if err := os.MkdirAll("testdata", 0o755); err != nil { // #nosec G301 -- a golden directory committed to the repository, world-readable like any source file
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(got), 0o644); err != nil { // #nosec G306 -- a golden file committed to the repository, world-readable like any source file
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(path) // #nosec G304 -- path is testdata/size-<name>.golden built from the test author's own name
			if err != nil {
				t.Fatalf("reading golden (run with -update-sizes to create): %v", err)
			}
			if got != string(want) {
				t.Errorf("view differs from %s (run with -update-sizes to accept)\n--- want\n%s\n--- got\n%s", path, want, got)
			}
		})
	}
}

func renderAt(t *testing.T, build func() tui.Model, w, h int) (out string) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic at %dx%d: %v", w, h, r)
		}
	}()
	m, _ := build().Update(tui.ResizeMsg{Width: w, Height: h})
	return ansi.StripANSI(m.View())
}
