// Command jsonview explores a JSON document as a tree you can search:
//
//	jsonview data.json
//	curl -s https://example.com/api | jsonview
//
// Up and Down move, Right and Left open and close a branch, and typing narrows
// the tree to the keys and values that match, with their parents opened. Esc
// clears the search; Ctrl+C quits.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/treeview"
)

type model struct {
	tree   treeview.Model
	source string
	height int
}

func newModel(data []byte, source string) (model, error) {
	tree, err := treeview.FromJSON(data)
	if err != nil {
		return model{}, fmt.Errorf("%s: %w", source, err)
	}
	tree.Filter = true
	tree.Mouse = true
	// Open the top level so the first screen shows something.
	if rows := tree.VisibleRows(); len(rows) > 0 {
		next, _ := tree.Update(tui.Key{Type: tui.KeyRight})
		tree = next
	}
	return model{tree: tree, source: source}, nil
}

func (m model) Init() tui.Cmd { return nil }

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.height = msg.Height
		return m, nil
	case tui.Key:
		if msg.Type == tui.KeyCtrlC {
			return m, tui.Quit()
		}
	}
	var cmd tui.Cmd
	m.tree, cmd = m.tree.Update(msg)
	return m, cmd
}

var (
	titleStyle = ansi.NewStyle().Bold().Foreground(ansi.BrightCyan)
	helpStyle  = ansi.NewStyle().Faint()
)

func (m model) View() string {
	rows := len(m.tree.VisibleRows())
	status := fmt.Sprintf("%d rows", rows)
	if q := m.tree.Query(); q != "" {
		status += fmt.Sprintf(" match %q", q)
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render(m.source) + "  " + helpStyle.Render(status) + "\n\n")
	b.WriteString(m.tree.View() + "\n\n")
	b.WriteString(helpStyle.Render("type to search   right/left open and close   esc clear   ctrl+c quit"))
	return b.String()
}

// read returns the document named by args: a file, or standard input when
// there is no argument or it is "-".
func read(args []string, stdin io.Reader) (data []byte, source string, err error) {
	if len(args) == 0 || args[0] == "-" {
		data, err = io.ReadAll(stdin)
		return data, "stdin", err
	}
	data, err = os.ReadFile(args[0]) // #nosec G304 G703 -- a viewer opens the file its user names
	return data, args[0], err
}

func main() {
	if len(os.Args) < 2 {
		if info, err := os.Stdin.Stat(); err == nil && info.Mode()&os.ModeCharDevice != 0 {
			fmt.Fprintln(os.Stderr, "usage: jsonview FILE   (or pipe JSON into it)")
			os.Exit(2)
		}
	}
	data, source, err := read(os.Args[1:], os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "jsonview:", err)
		os.Exit(1)
	}
	m, err := newModel(data, source)
	if err != nil {
		fmt.Fprintln(os.Stderr, "jsonview:", err)
		os.Exit(1)
	}
	// Input comes from the terminal even when the document was piped in.
	opts := []tui.ProgramOption{}
	if tty, err := os.Open("/dev/tty"); err == nil {
		defer tty.Close()
		opts = append(opts, tui.WithInput(tty))
	}
	if _, err := tui.NewProgram(m, opts...).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "jsonview:", err)
		os.Exit(1)
	}
}
