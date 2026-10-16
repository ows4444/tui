// Command diskuse shows what takes the space in a directory:
//
//	diskuse            # the current directory
//	diskuse ~/Projects
//
// Each entry is a row with its size, largest first. Enter goes into a
// directory, Backspace goes up, "s" sorts by the column the cursor is on
// (Left and Right choose it) and "S" puts the order back, "/" filters by name,
// and Ctrl+C quits.
package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/datatable"
)

// maxFiles caps how many files one scan reads, so a huge tree does not hang
// the screen; a directory that hits it is marked "~".
const maxFiles = 200_000

type entry struct {
	name   string
	dir    bool
	bytes  int64
	capped bool
}

// scan lists the entries of dir with their sizes: a file's own size, a
// directory's total, largest first.
func scan(dir string) ([]entry, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]entry, 0, len(ents))
	seen := 0
	for _, e := range ents {
		en := entry{name: e.Name(), dir: e.IsDir()}
		if !e.IsDir() {
			if info, err := e.Info(); err == nil {
				en.bytes = info.Size()
			}
			out = append(out, en)
			continue
		}
		_ = filepath.WalkDir(filepath.Join(dir, e.Name()), func(_ string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil // unreadable: count what can be read
			}
			if seen++; seen > maxFiles {
				en.capped = true
				return fs.SkipAll
			}
			if !d.IsDir() {
				if info, err := d.Info(); err == nil {
					en.bytes += info.Size()
				}
			}
			return nil
		})
		out = append(out, en)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].bytes > out[j].bytes })
	return out, nil
}

// human is a size in the usual units: 1.5 MB.
func human(n int64) string {
	const unit = 1024
	if n < unit {
		return strconv.FormatInt(n, 10) + " B"
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}

type model struct {
	dir     string
	table   datatable.Model
	entries []entry
	err     error
	width   int
}

func newModel(dir string) model {
	m := model{dir: dir}
	m.load()
	return m
}

// load scans m.dir and rebuilds the table.
func (m *model) load() {
	ents, err := scan(m.dir)
	m.entries, m.err = ents, err
	rows := make([][]string, len(ents))
	for i, e := range ents {
		kind, size := "file", human(e.bytes)
		if e.dir {
			kind = "dir"
			if e.capped {
				size = "~" + size
			}
		}
		rows[i] = []string{size, e.name, kind}
	}
	t := datatable.New([]string{"Size", "Name", "Kind"}, rows)
	t.Filter = true
	t.CellStyle = func(row, col int, v string) ansi.Style {
		if col == 2 && v == "dir" {
			return ansi.NewStyle().Foreground(ansi.BrightBlue)
		}
		return ansi.Style{}
	}
	m.table = t
}

func (m model) Init() tui.Cmd { return nil }

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width = msg.Width
		m.table, _ = m.table.Update(msg)
		return m, nil
	case datatable.SelectedMsg:
		if e := m.entries[msg.SourceRow]; e.dir {
			m.dir = filepath.Join(m.dir, e.name)
			m.load()
		}
		return m, nil
	case tui.Key:
		switch {
		case msg.Type == tui.KeyCtrlC:
			return m, tui.Quit()
		case msg.Type == tui.KeyBackspace && !m.table.Filtering() && m.table.Query() == "":
			m.dir = filepath.Dir(m.dir)
			m.load()
			return m, nil
		}
	}
	var cmd tui.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

var (
	titleStyle = ansi.NewStyle().Bold().Foreground(ansi.BrightCyan)
	helpStyle  = ansi.NewStyle().Faint()
)

func (m model) View() string {
	if m.err != nil {
		return titleStyle.Render(m.dir) + "\n\n" + m.err.Error() + "\n\n" + helpStyle.Render("backspace up   ctrl+c quit")
	}
	var total int64
	for _, e := range m.entries {
		total += e.bytes
	}
	head := fmt.Sprintf("%s  %s", titleStyle.Render(m.dir), helpStyle.Render(human(total)+" in "+strconv.Itoa(len(m.entries))+" entries"))
	return head + "\n\n" + m.table.View() + "\n\n" +
		helpStyle.Render("enter open   backspace up   s sort   S unsort   / filter   ctrl+c quit")
}

func main() {
	dir := "."
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "diskuse:", err)
		os.Exit(1)
	}
	if _, err := tui.NewProgram(newModel(abs)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "diskuse:", err)
		os.Exit(1)
	}
}
