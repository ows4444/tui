package tui

import (
	"os"
	"strings"
	"testing"
)

// Criterion #74: WithCellRenderer(false) selects the line renderer.
func TestWithCellRendererFalseUsesTheLineRenderer(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(staticModel{view: "a\nb"}, WithOutput(out), WithAltScreen(false), WithCellRenderer(false))
	p.render()
	if p.cells != nil && p.cells.Valid() {
		t.Error("the cell renderer drew a frame although WithCellRenderer(false) was given")
	}
	if got := string(read()); !strings.Contains(got, "a") || !strings.Contains(got, "b") {
		t.Errorf("frame = %q", got)
	}
}

// Criterion #75: the migration guide shows what replaces WithLineRenderer.
func TestMigrationGuideShowsTheWithLineRendererReplacement(t *testing.T) {
	raw, err := os.ReadFile("docs/migrating-to-v1.md")
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	i := strings.Index(s, "### WithLineRenderer")
	if i < 0 {
		t.Fatal("the guide has no WithLineRenderer section")
	}
	sec := s[i:]
	if j := strings.Index(sec[3:], "\n### "); j >= 0 {
		sec = sec[:j+3]
	}
	if !strings.Contains(sec, "WithCellRenderer(false)") {
		t.Error("the WithLineRenderer section does not show WithCellRenderer(false)")
	}
}
