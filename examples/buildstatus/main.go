// Command buildstatus runs a made-up build pipeline in the terminal's own
// scrollback (no alternate screen): eight steps with dependencies, several
// running at once, each with a spinner, a progress bar and its latest line of
// output. A finished step is written into the scrollback as one line and stays
// there; a failed one also prints its output. The live region at the bottom is
// replaced by a summary when the build ends. Nothing real is run: the steps and
// their times are fixed, so every run is the same.
//
// Keyboard: Ctrl+C cancels the build (what has finished stays printed), + and -
// change how many steps may run at once, v shows or hides the output lines,
// Enter or q leaves once it is over. There is no mouse: an inline program
// leaves the mouse to the terminal so you can select and copy from scrollback.
//
//	go run ./examples/buildstatus
//	go run ./examples/buildstatus -fail test      # one step fails, others are skipped
//	go run ./examples/buildstatus -jobs 1         # run the steps one at a time
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/examples/internal/ui"
)

func main() {
	fail := flag.String("fail", "", "name of a step that fails (fetch modules, compile ansi, link tui, vet, test, ...)")
	jobs := flag.Int("jobs", 3, "steps to run at once")
	flag.Parse()
	if *fail != "" && newBuild(1, "").index(*fail) < 0 {
		fmt.Fprintf(os.Stderr, "no step named %q\n", *fail)
		os.Exit(2)
	}
	m := newModel(*jobs, *fail)
	final, err := ui.NewProgram(m, tui.WithAltScreen(false)).Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if fm, ok := final.(model); ok && !fm.b.ok() {
		os.Exit(1)
	}
}
