package main

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

var updateGolden = flag.Bool("update", false, "rewrite the golden files")

// portStates are the screens this example can show, by name.
func portStates() map[string]model {
	loading := initialModel()
	done, _ := loading.Update(loadedMsg{result: "42 widgets loaded from the warehouse"})
	return map[string]model{
		"loading": loading,
		"done":    done.(model),
	}
}

// The screen is built with layout.Node, not by joining strings.
func TestScreenUsesLayoutNode(t *testing.T) {
	b, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(b)
	if !strings.Contains(src, "layout.Draw(") {
		t.Error("main.go does not draw its view with layout.Draw")
	}
	for _, banned := range []string{"JoinHorizontal", "JoinVertical", `"\n\n"`} {
		if strings.Contains(src, banned) {
			t.Errorf("main.go still joins by hand: %s", banned)
		}
	}
}

// At any terminal size the screen fills exactly the space it is given.
func TestScreenRendersExactlyTheRequestedSize(t *testing.T) {
	for name, m := range portStates() {
		for _, s := range []layout.Size{{W: 40, H: 10}, {W: 80, H: 24}, {W: 300, H: 80}, {W: 5, H: 3}} {
			lines := strings.Split(m.screen().Render(s), "\n")
			if len(lines) != s.H {
				t.Fatalf("%s %v: %d rows, want %d", name, s, len(lines), s.H)
			}
			for i, l := range lines {
				if w := ansi.Width(l); w != s.W {
					t.Errorf("%s %v: row %d is %d wide, want %d", name, s, i, w, s.W)
				}
			}
		}
	}
}

// The natural view has no row wider than the rest: nothing spills.
func TestNaturalViewHasNoOverflowingRow(t *testing.T) {
	for name, m := range portStates() {
		lines := strings.Split(m.View(), "\n")
		want := ansi.Width(lines[0])
		for i, l := range lines {
			if w := ansi.Width(l); w != want {
				t.Errorf("%s: row %d is %d wide, the first row is %d: %q", name, i, w, want, ansi.StripANSI(l))
			}
		}
	}
}

// Golden views, ANSI stripped so they stay readable. Regenerate with -update.
func TestViewGolden(t *testing.T) {
	for name, m := range portStates() {
		got := ansi.StripANSI(m.View())
		golden := filepath.Join("testdata", strings.ReplaceAll(name, " ", "-")+".golden")
		if *updateGolden {
			if err := os.MkdirAll("testdata", 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		want, err := os.ReadFile(golden)
		if err != nil {
			t.Fatal(err)
		}
		if got != string(want) {
			t.Errorf("view differs from %s (go test -update to regenerate):\n%s", golden, got)
		}
	}
}
