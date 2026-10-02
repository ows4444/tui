package tui

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// When all example goldens are rendered with the cell renderer, the frame log
// shall record zero fallbacks (spec S11, decision #5).
func TestCellRendererZeroFallbacksOnExampleGoldens(t *testing.T) {
	files, err := filepath.Glob("examples/*/testdata/*.golden")
	if err != nil || len(files) == 0 {
		t.Fatalf("no goldens found: %v", err)
	}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var out, log bytes.Buffer
		p := NewProgram(staticModel{view: strings.TrimSuffix(string(data), "\n")},
			WithOutput(&out), WithFrameLog(&log), WithCellRenderer(true))
		p.width, p.height = 400, 200
		p.render()
		if strings.Contains(log.String(), "kind=fallback") {
			t.Errorf("%s: %s", f, strings.TrimSpace(log.String()))
		}
	}
}
