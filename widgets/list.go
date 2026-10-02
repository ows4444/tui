package widgets

import (
	"strconv"
	"strings"

	"github.com/ows4444/tui/theme"
)

// Marker selects how List prefixes each item.
type Marker int

const (
	// MarkerNone leaves each item unprefixed. It is the zero value.
	MarkerNone Marker = iota
	// MarkerBullet prefixes each item with "• ".
	MarkerBullet
	// MarkerNumber prefixes each item with its 1-based number and ". ",
	// right-aligned so the items line up past nine entries.
	MarkerNumber
)

// List renders items one per line with the given marker prefix. It is a
// plain, stateless render — no cursor, no Update/Model — distinct from the
// interactive picker.Model and multiselect.Model widgets. An empty items
// slice renders as "".
func List(items []string, marker Marker) string { return ListWith(items, marker, theme.Theme{}) }

// ListWith is List with the bullet drawn from t (theme.Glyphs.Bullet), so an
// ASCII theme gets "* ". List is ListWith on the default theme.
func ListWith(items []string, marker Marker, t theme.Theme) string {
	if len(items) == 0 {
		return ""
	}

	lines := make([]string, len(items))
	switch marker {
	case MarkerBullet:
		for i, item := range items {
			lines[i] = t.GlyphSet().Bullet + " " + item
		}
	case MarkerNumber:
		width := len(strconv.Itoa(len(items)))
		for i, item := range items {
			ord := strconv.Itoa(i + 1)
			lines[i] = strings.Repeat(" ", width-len(ord)) + ord + ". " + item
		}
	default:
		copy(lines, items)
	}

	return strings.Join(lines, "\n")
}
