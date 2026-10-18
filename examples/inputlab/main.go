// Command inputlab shows every key, mouse, paste, focus and resize event as it
// arrives, decoded the way the library hands it to Update, plus what the
// terminal said it can do. It is a debugging tool: run it in the terminal that
// has the problem and see what your program would really get.
//
// Everything you type or click is an event here, so q is just a key. The keys
// the tool keeps for itself are Ctrl+C (quit), Ctrl+P (pause the log), Ctrl+L
// (clear it), F1 (help), F2 to F5 (show or hide keys, mouse, paste, other) and
// F6 (join runs of mouse motion into one line); they are logged too. The wheel
// is an event as well, so the log scrolls with PgUp and PgDn while it is
// paused. Mouse: click the filter buttons in the top bar.
//
// By default it asks the terminal for the kitty keyboard protocol, so key
// release and repeat show up where the terminal supports them; -legacy turns
// that off to see what an older terminal sends.
//
//	go run ./examples/inputlab
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ows4444/tui"
)

func main() {
	legacy := flag.Bool("legacy", false, "do not ask for the kitty keyboard protocol")
	flag.Parse()
	opts := []tui.ProgramOption{tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion), tui.WithBracketedPaste(true),
		tui.WithFocusReporting(true), tui.WithCapabilityProbe(500 * time.Millisecond), tui.WithColorSchemeReports(true)}
	if !*legacy {
		opts = append(opts, tui.WithKeyboard(tui.KeyboardDisambiguate|tui.KeyboardReportEvents|tui.KeyboardReportAllKeys))
	}
	if _, err := tui.NewProgram(newModel(), opts...).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
