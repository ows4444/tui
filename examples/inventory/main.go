// Command inventory is a stock table with a detail pane. The datatable reads
// its rows from a Source (nothing is stored), right-aligns and formats the
// numeric columns through Columns, lets Space mark rows, and the pane beside it
// shows the row under the cursor and how many are marked.
package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/datatable"
	"github.com/ows4444/tui/layout"
)

// stock is 5,000 items computed from the index alone.
type stock struct{}

var kinds = []string{"bolt", "nut", "washer", "bracket", "hinge"}

func (stock) Rows() int { return 5000 }
func (stock) Row(i int) []string {
	return []string{fmt.Sprintf("%s-%04d", kinds[i%len(kinds)], i), strconv.Itoa(i*37%900 + 10), strconv.Itoa(i * 125 % 9000)}
}

type model struct {
	table datatable.Model
	w, h  int // terminal size from the last ResizeMsg; 80x24 until known
}

func initialModel() model {
	t := datatable.NewSource([]string{"Item", "Qty", "Price"}, stock{})
	t.Multi = true
	t.Columns = []datatable.Column{
		{Min: 12},
		{Align: layout.AlignEnd, Min: 5},
		{Align: layout.AlignEnd, Min: 8, Format: price},
	}
	m := model{table: t, w: 80, h: 24}
	return m.fit()
}

// price turns cents into a dollar amount.
func price(cell string) string {
	c, err := strconv.Atoi(cell)
	if err != nil {
		return cell
	}
	return fmt.Sprintf("$%d.%02d", c/100, c%100)
}

func (m model) fit() model {
	m.table.Height = max(m.h-4, 1)
	m.table.Width = max(m.w*3/5, 20)
	return m
}

func (m model) Init() tui.Cmd { return nil }

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.w, m.h = msg.Width, msg.Height
		return m.fit(), nil
	case tui.Key:
		if msg.Type == tui.KeyCtrlC || msg.Type == tui.KeyEsc || (msg.Type == tui.KeyRunes && msg.Text == "q") {
			return m, tui.Quit()
		}
	}
	var cmd tui.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// detail describes the row under the cursor.
func (m model) detail() string {
	row := stock{}.Row(m.table.SourceRow(m.table.Cursor()))
	return fmt.Sprintf("Detail\n\nItem:  %s\nQty:   %s\nPrice: %s\n\nMarked: %d", row[0], row[1], price(row[2]), len(m.table.Marked()))
}

func (m model) screen() layout.Node {
	return layout.Column(1,
		layout.Fill(layout.Row(2,
			layout.FlexChild{Node: m.table.LayoutNode(), Basis: max(m.w*3/5, 20)},
			layout.Fill(layout.Block(m.detail())),
		)),
		layout.FlexChild{Node: layout.Text("up/down: move   space: mark   s: sort   q: quit", layout.WithEllipsis("…"))},
	)
}

func (m model) View() string {
	return layout.Draw(m.screen(), layout.Tight(layout.Size{W: m.w, H: m.h}))
}

func main() {
	if _, err := tui.NewProgram(initialModel(), tui.WithAltScreen(true)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
