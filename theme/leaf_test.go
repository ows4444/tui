package theme

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ows4444/tui/internal/basetypes"
	"github.com/ows4444/tui/layout"
)

const modPrefix = "github.com/ows4444/tui/"

// theme must stay a dependency leaf: only ansi and internal packages.
func TestThemeIsDependencyLeaf(t *testing.T) {
	gobin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go tool not found")
	}
	out, err := exec.Command(gobin, "env", "GOMOD").Output()
	if err != nil {
		t.Skipf("go env GOMOD: %v", err)
	}
	gomod := strings.TrimSpace(string(out))
	if gomod == "" || gomod == "/dev/null" {
		t.Skip("not in a module")
	}
	cmd := exec.Command(gobin, "list", "-deps", "./theme")
	cmd.Dir = filepath.Dir(gomod)
	out, err = cmd.Output()
	if err != nil {
		t.Fatalf("go list -deps ./theme: %v", err)
	}
	var bad []string
	for _, p := range strings.Fields(string(out)) {
		if !strings.HasPrefix(p, modPrefix) || p == modPrefix+"theme" {
			continue
		}
		rest := strings.TrimPrefix(p, modPrefix)
		if rest == "ansi" || strings.HasPrefix(rest, "internal/") {
			continue
		}
		bad = append(bad, p)
	}
	if len(bad) > 0 {
		t.Fatalf("theme depends on forbidden packages: %v", bad)
	}
}

// The borders theme's presets use must equal layout's.
func TestThemeBordersMatchLayout(t *testing.T) {
	if basetypes.NormalBorder != layout.NormalBorder() ||
		basetypes.RoundedBorder != layout.RoundedBorder() ||
		basetypes.ASCIIBorder != layout.ASCIIBorder() {
		t.Fatal("themetypes borders drifted from layout borders")
	}
	var b layout.Border = DarkTheme().Border // alias: assignable both ways
	_ = b
}
