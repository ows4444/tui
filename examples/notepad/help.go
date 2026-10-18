package main

import (
	"github.com/ows4444/tui/examples/internal/ui"
	"github.com/ows4444/tui/layout"
)

// screen is the program's screen, or its help while help is open.
func (m model) screen() layout.Node {
	if m.help {
		return ui.HelpScreen(m.theme(), "Notepad",
			ui.Help{Key: "type", Does: "edit the text"},
			ui.Help{Key: "Ctrl+S", Does: "save"},
			ui.Help{Key: "Ctrl+O", Does: "open a file"},
			ui.Help{Key: "Ctrl+F", Does: "find"},
			ui.Help{Key: "Alt+letter  click", Does: "open a menu"},
			ui.Help{Key: "F1", Does: "this help"},
			ui.Help{Key: "Ctrl+Q", Does: "quit (asks first if there are unsaved changes)"})
	}
	return m.mainScreen()
}
