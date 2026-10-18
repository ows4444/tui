package main

import (
	"github.com/ows4444/tui/examples/internal/ui"
	"github.com/ows4444/tui/layout"
)

// screen is the program's screen, or its help while help is open.
func (m model) screen() layout.Node {
	if m.help {
		return ui.HelpScreen(m.theme(), "Hello, tui",
			ui.Help{Key: "type", Does: "your name in the field"},
			ui.Help{Key: "Enter", Does: "greet (or click Greet)"},
			ui.Help{Key: "Tab  Shift+Tab", Does: "move focus between the field and the buttons"},
			ui.Help{Key: "click", Does: "a button, or the field"},
			ui.Help{Key: "F1", Does: "this help"},
			ui.Help{Key: "Ctrl+C", Does: "quit (or click Quit)"})
	}
	return m.mainScreen()
}
