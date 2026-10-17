package prompt

import (
	"errors"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

// Choice is one entry of Choose: the text shown, the value it stands for, and
// optionally a group it belongs to.
type Choice[T any] struct {
	// Label is what the person sees and types to find the entry.
	Label string
	// Value is what Choose returns when this entry is picked.
	Value T
	// Group, when set, is shown before the label as "Group > Label", and typing
	// the group narrows to its entries. Entries are listed in the order given, so
	// give a group's entries together.
	Group string
}

// Choose is Select for values of any type: it asks the person to pick one of
// choices and returns that choice's Value. Typing narrows the list, Enter picks,
// Config.Start sets where the cursor begins and Config.PageSize limits how many
// entries show at once.
//
//	region, err := prompt.Choose("Region", []prompt.Choice[Region]{
//		{Label: "Frankfurt", Value: euCentral, Group: "Europe"},
//		{Label: "Oregon", Value: usWest, Group: "Americas"},
//	})
func Choose[T any](label string, choices []Choice[T], cfg ...Config) (T, error) {
	var zero T
	if len(choices) == 0 {
		return zero, errors.New("prompt: Choose needs at least one choice")
	}
	labels := make([]string, len(choices))
	for i, ch := range choices {
		labels[i] = ansi.Sanitize(ch.Label)
		if ch.Group != "" {
			labels[i] = ansi.Sanitize(ch.Group) + " > " + labels[i]
		}
	}
	i, err := Select(label, labels, cfg...)
	if err != nil {
		return zero, err
	}
	return choices[i].Value, nil
}

// pageOf cuts the picker's lines to a window of page entries around the cursor.
// filter says the picker's first line is the query, which stays on top.
func pageOf(view string, cursor, page int, filter bool) string {
	if page <= 0 {
		return view
	}
	lines := strings.Split(view, "\n")
	head := 0
	if filter && len(lines) > 0 {
		head = 1
	}
	if len(lines)-head <= page {
		return view
	}
	body := layout.Window(lines[head:], cursor, page)
	return strings.Join(append(lines[:head:head], body...), "\n")
}
