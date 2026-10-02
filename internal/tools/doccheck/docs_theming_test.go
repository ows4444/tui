package main

import (
	"os"
	"strings"
	"testing"
)

// TestThemingDocDescribesWithTheme fails when docs/theming.md still says
// WithTheme is impossible or omits the API a reader needs to use it.
func TestThemingDocDescribesWithTheme(t *testing.T) {
	raw, err := os.ReadFile("docs/theming.md")
	if err != nil {
		t.Fatal(err)
	}
	doc := string(raw)
	for _, stale := range []string{"no `WithAutoTheme`", "cannot import `theme`", "import cycle"} {
		if strings.Contains(doc, stale) {
			t.Errorf("docs/theming.md still contains stale text %q", stale)
		}
	}
	for _, want := range []string{"WithTheme", "ThemeSetter", "SetTheme", "ResolvedStates", "Themeable"} {
		if !strings.Contains(doc, want) {
			t.Errorf("docs/theming.md does not mention %s", want)
		}
	}
}
