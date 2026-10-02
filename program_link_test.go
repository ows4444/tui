package tui_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ows4444/tui"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

// osc8Balanced reports whether every OSC 8 open in s is closed and none is
// closed without being open. Copied from ansi/osc_test.go, which is
// unexported.
func osc8Balanced(s string) bool {
	const intro = "\x1b]8;"
	depth := 0
	for {
		i := strings.Index(s, intro)
		if i < 0 {
			return depth == 0
		}
		s = s[i+len(intro):]
		end, termLen := len(s), 0
		if j := strings.IndexByte(s, 0x07); j >= 0 {
			end, termLen = j, 1
		}
		if j := strings.Index(s, "\x1b\\"); j >= 0 && j < end {
			end, termLen = j, 2
		}
		if termLen == 0 {
			return false
		}
		body := s[:end]
		s = s[end+termLen:]
		if body[strings.IndexByte(body, ';')+1:] == "" {
			depth--
		} else {
			depth++
		}
		if depth < 0 || depth > 1 {
			return false
		}
	}
}

// linkModel shows a view, is resized to w x 24 by its first command, then quits.
type linkModel struct {
	view string
	w    int
}

func (m linkModel) Init() tui.Cmd {
	return tui.Sequence(func() tui.Msg { return tui.ResizeMsg{Width: m.w, Height: 24} }, tui.Quit())
}
func (m linkModel) Update(tui.Msg) (tui.Model, tui.Cmd) { return m, nil }
func (m linkModel) View() string                        { return m.view }

// A Link wider than the terminal is clipped by fit; the OSC 8 must stay
// balanced and the URL bytes must not be counted as columns. Criterion 4.
func TestFitClipsLinkWithBalancedOSC8(t *testing.T) {
	link := widgets.Link("click here for the documentation", "https://example.com/a/very/long/url/that/is/long", true, theme.DarkTheme())
	for _, cell := range []bool{false, true} {
		for _, w := range []int{5, 10, 20, 33, 60, 200} {
			var buf bytes.Buffer
			p := tui.NewProgram(linkModel{view: "head\n" + link + "\ntail", w: w},
				tui.WithOutput(&buf), tui.WithInput(strings.NewReader("")),
				tui.WithColorProfile(ansi.TrueColor), tui.WithCellRenderer(cell))
			if _, err := p.Run(); err != nil {
				t.Fatal(err)
			}
			out := buf.String()
			if !strings.Contains(out, "\x1b]8;") {
				t.Errorf("cell=%v width=%d: no OSC 8 drawn at all (test proves nothing): %q", cell, w, out)
			}
			if !osc8Balanced(out) {
				t.Errorf("cell=%v width=%d: unbalanced OSC 8 in %q", cell, w, out)
			}
		}
	}
}
