package tui_test

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/breadcrumb"
	"github.com/ows4444/tui/button"
	"github.com/ows4444/tui/buttongroup"
	"github.com/ows4444/tui/checkbox"
	"github.com/ows4444/tui/otpinput"
	"github.com/ows4444/tui/pagination"
	"github.com/ows4444/tui/radiogroup"
	"github.com/ows4444/tui/rating"
	"github.com/ows4444/tui/slider"
	"github.com/ows4444/tui/toggle"
)

// The pressable and adjustable controls show keyboard focus one way: the
// control's outer brackets become angle brackets. It reads with no colour
// and no bold, and does not change the control's width. A control added to
// this family joins the table.
func TestControlsShowFocusWithAngleBrackets(t *testing.T) {
	plain := ansi.StripANSI
	cases := map[string]func(focused bool) string{
		"breadcrumb": func(f bool) string {
			m := breadcrumb.New("Home", "Docs")
			if f {
				m.Focus()
			}
			return plain(m.View())
		},
		"otpinput": func(f bool) string {
			m := otpinput.New(4)
			if f {
				m.Focus()
			}
			return plain(m.View())
		},
		"pagination": func(f bool) string {
			m := pagination.New(9)
			if f {
				m.Focus()
			}
			return plain(m.View())
		},
		"button": func(f bool) string {
			m := button.New("Save")
			if f {
				m.Focus()
			}
			return plain(m.View())
		},
		"buttongroup": func(f bool) string {
			m := buttongroup.New(buttongroup.ModeSingle, "A", "B")
			if f {
				m.Focus()
			}
			return plain(m.View())
		},
		"checkbox": func(f bool) string {
			m := checkbox.New("Keep")
			if f {
				m.Focus()
			}
			return plain(m.View())
		},
		"radiogroup": func(f bool) string {
			m := radiogroup.New("A", "B")
			if f {
				m.Focus()
			}
			return plain(m.View())
		},
		"rating": func(f bool) string {
			m := rating.New(5)
			if f {
				m.Focus()
			}
			return plain(m.View())
		},
		"slider": func(f bool) string {
			m := slider.New(0, 10)
			if f {
				m.Focus()
			}
			return plain(m.View())
		},
		"toggle": func(f bool) string {
			m := toggle.New("Overwrite")
			if f {
				m.Focus()
			}
			return plain(m.View())
		},
	}
	for name, view := range cases {
		rest, focused := view(false), view(true)
		if strings.ContainsAny(rest, "<>") {
			t.Errorf("%s without focus shows an angle bracket: %q", name, rest)
		}
		if strings.Count(focused, "<") != 1 || strings.Count(focused, ">") != 1 || strings.Index(focused, "<") > strings.Index(focused, ">") {
			t.Errorf("%s with focus is not in one pair of angle brackets: %q", name, focused)
		}
		if ansi.Width(rest) != ansi.Width(focused) {
			t.Errorf("%s changes width with focus: %q is %d cells, %q is %d", name, rest, ansi.Width(rest), focused, ansi.Width(focused))
		}
	}
}
