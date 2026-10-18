// Command filedialog is an Open and Save As dialog built on package filepicker:
// browse folders, filter by kind, hide or show the dotfiles, jump to a folder by
// clicking it in the path bar, and type the first letters of a name to go to it.
// When you confirm, the chosen path is printed to standard output after the
// screen closes, and the exit status is 0; Cancel exits with status 1 and prints
// nothing. It only reads: Save As chooses a name and writes nothing.
//
// Mouse: click a row to select it and click it again, or double-click, to open
// it; click a folder of the path bar to go there; click a filter or the Show
// hidden box; click Open (or Save) and Cancel; scroll the list. Keyboard: arrows
// move, Enter or Right opens a folder or chooses a file, Left or Backspace goes
// up a folder, letters jump to a name, F3 changes the filter, "." shows or hides
// dotfiles, F2 switches between Open and Save As, in Save As Tab moves between
// the list and the name field, Esc cancels, ? shows help.
//
//	go run ./examples/filedialog [dir]
//	go run ./examples/filedialog -save [dir]
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ows4444/tui"
)

func main() {
	save := flag.Bool("save", false, "start as Save As instead of Open")
	flag.Parse()
	dir := "."
	if flag.NArg() > 0 {
		dir = flag.Arg(0)
	}
	m := newModel(dir, *save)
	// the dialog is drawn on the terminal's own screen, so the path can be printed after
	final, err := tui.NewProgram(m, tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion)).Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fm, ok := final.(model)
	if !ok || fm.result == "" {
		os.Exit(1)
	}
	fmt.Println(fm.result)
}
