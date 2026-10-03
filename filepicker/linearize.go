package filepicker

import (
	"github.com/ows4444/tui/ansi"
	"strconv"
)

// Linearize renders the listing as plain text for accessible output (see
// tui.Linearizer): a header with the directory and entry count, then one
// line per entry with its kind ("folder" or "file"), its position and
// ", selected" on the cursor row. No prefixes or trailing slashes. A
// directory that could not be listed says so in the header.
func (m Model) Linearize() string {
	if m.err != nil {
		return ansi.Clean(m.Raw, m.Dir) + ", file picker, cannot be read"
	}
	if len(m.entries) == 0 {
		return ansi.Clean(m.Raw, m.Dir) + ", file picker, empty"
	}
	n := strconv.Itoa(len(m.entries))
	out := ansi.Clean(m.Raw, m.Dir) + ", file picker, " + n + " entries"
	for i, e := range m.entries {
		kind := "file"
		if e.IsDir {
			kind = "folder"
		}
		line := ansi.Clean(m.Raw, e.Name) + ", " + kind + ", entry " + strconv.Itoa(i+1) + " of " + n
		if i == m.cursor {
			line += ", selected"
		}
		out += "\n" + line
	}
	return out
}
