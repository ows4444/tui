package main

import (
	"github.com/ows4444/tui/examples/internal/ui"
	"github.com/ows4444/tui/layout"
)

// screen is the program's screen, or its help while help is open.
func (m model) screen() layout.Node {
	if m.help {
		return ui.HelpScreen(m.theme(), "Tasks",
			ui.Help{Key: "click", Does: "a tab to filter, a box to tick, the cross to delete, the field to type"},
			ui.Help{Key: "Up Down  j k", Does: "move"},
			ui.Help{Key: "Space  Enter", Does: "tick or untick"},
			ui.Help{Key: "d  Delete", Does: "delete"},
			ui.Help{Key: "Tab", Does: "next filter"},
			ui.Help{Key: "a  /", Does: "add a task (Enter adds, Esc stops)"},
			ui.Help{Key: "?", Does: "this help"},
			ui.Help{Key: "q  Esc  Ctrl+C", Does: "quit"})
	}
	return m.mainScreen()
}
