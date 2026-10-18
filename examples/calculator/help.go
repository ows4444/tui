package main

import (
	"github.com/ows4444/tui/examples/internal/ui"
	"github.com/ows4444/tui/layout"
)

// screen is the program's screen, or its help while help is open.
func (m model) screen() layout.Node {
	if m.help {
		return ui.HelpScreen(m.theme(), "Calculator",
			ui.Help{Key: "0-9 . + - * /", Does: "type a calculation"},
			ui.Help{Key: "Enter  =", Does: "evaluate"},
			ui.Help{Key: "Backspace", Does: "delete the last character"},
			ui.Help{Key: "c  Esc", Does: "clear (Esc on an empty screen quits)"},
			ui.Help{Key: "y", Does: "copy the result"},
			ui.Help{Key: "click", Does: "the keys, or an entry in the history"},
			ui.Help{Key: "?", Does: "this help"},
			ui.Help{Key: "q  Ctrl+C", Does: "quit"})
	}
	return m.mainScreen()
}
