package main

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
)

var update = flag.Bool("update", false, "rewrite the golden files")

// The view is built with layout.Node, not the string layout helpers.
func TestViewUsesLayoutNode(t *testing.T) {
	b, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(b)
	if !strings.Contains(src, "layout.Draw(") {
		t.Error("main.go does not build its view with layout.Draw")
	}
	for _, banned := range []string{"JoinHorizontal", "JoinVertical", ".View()"} {
		if strings.Contains(src, banned) {
			t.Errorf("main.go still uses the string layout: %s", banned)
		}
	}
}

// Golden view of the expanded JSON tab, ANSI stripped. Regenerate with -update.
func TestViewGolden(t *testing.T) {
	m := send(initialModel(), tui.Key{Type: tui.KeyRight})
	m = send(m, tui.Key{Type: tui.KeyEnter})
	got := ansi.StripANSI(m.View())
	golden := filepath.Join("testdata", "json-tab.golden")
	if *update {
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
