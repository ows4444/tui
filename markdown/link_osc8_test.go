package markdown

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/theme"
)

// Markdown links render as underlined text with the destination shown in
// parentheses; they never emit OSC 8, so no target can reach one.
func TestLinkNeverEmitsOSC8(t *testing.T) {
	for _, dest := range []string{"https://ok.test", "javascript:alert(1)", "https://x\x1b]8;;evil\x07", "data:text/html,x"} {
		out := Render("[text]("+dest+")", 80, theme.DarkTheme())
		if strings.Contains(out, "\x1b]8;") {
			t.Errorf("dest %q: output %q contains OSC 8", dest, out)
		}
		if strings.Contains(out, "\x07") {
			t.Errorf("dest %q: output %q contains BEL", dest, out)
		}
	}
}
