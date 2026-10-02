package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/dialog"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/loadingbar"
	"github.com/ows4444/tui/spinner"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/toast"
	"github.com/ows4444/tui/widgets"
	"github.com/ows4444/tui/widgets/chart"
)

type model struct {
	cpu  []float64
	mem  []float64
	disk float64
	rng  *rand.Rand

	width int // terminal width from ResizeMsg; 0 until known (treated as narrow)

	spin    spinner.Model
	loading loadingbar.Model
	about   dialog.Model
	notice  toast.Model
}

func initialModel() model {
	m := model{rng: rand.New(rand.NewSource(1))} // #nosec G404 -- fixed-seed demo data, not security-sensitive
	for i := 0; i < 20; i++ {
		m.cpu = append(m.cpu, m.rng.Float64()*100)
		m.mem = append(m.mem, m.rng.Float64()*100)
	}
	m.disk = m.rng.Float64()

	m.spin = spinner.New()
	m.loading = loadingbar.New(15)
	// Called here so the persisted model starts running (Init below only
	// returns the Cmds; it can't mutate the model that gets kept — the
	// same reason examples/form captures a focusCmd at construction).
	m.spin.Start()
	m.loading.Start()

	m.about = dialog.New("About", "tui — a from-scratch terminal UI framework.\n\nEnter/Esc to close.")
	m.about.Width = 30 // narrower than boxWidth so it stays comfortably inside the dashboard, with margin
	m.about.Hide()     // starts closed; 'd' opens it

	m.notice = toast.New("Refreshed!")
	m.notice.Variant = widgets.VariantSuccess
	m.notice.Duration = 2 * time.Second

	return m
}

func (m model) Init() tui.Cmd {
	return tui.Batch(m.spin.Start(), m.loading.Start())
}

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width = msg.Width
		return m, nil
	case tui.Key:
		if msg.Type == tui.KeyCtrlC {
			return m, tui.Quit()
		}
		// Modal: while the dialog is open, it gets every key exclusively
		// (only Ctrl+C above still force-quits) — 'q'/'r'/'d' below don't
		// fire until it's dismissed.
		if m.about.Open() {
			next, cmd := m.about.Update(msg)
			m.about = next
			return m, cmd
		}
		switch msg.Type {
		case tui.KeyEsc:
			return m, tui.Quit()
		case tui.KeyRunes:
			if msg.Text == "" {
				break
			}
			switch []rune(msg.Text)[0] {
			case 'q':
				return m, tui.Quit()
			case 'r':
				m.cpu = append(m.cpu[1:], m.rng.Float64()*100)
				m.mem = append(m.mem[1:], m.rng.Float64()*100)
				m.disk = m.rng.Float64()
				return m, m.notice.Show()
			case 'd':
				m.about.Show()
			}
		}
		return m, nil
	}

	// None of spinner/loadingbar/toast's tick message types are exported,
	// so forwarding every non-Key Msg to all three is how each decides for
	// itself whether it means anything to it (a no-op for anything but its
	// own ticks) — the same pattern autocomplete uses to let textinput's
	// blink ticks reach its embedded Input.
	var spinCmd, loadCmd, noticeCmd tui.Cmd
	m.spin, spinCmd = m.spin.Update(msg)
	m.loading, loadCmd = m.loading.Update(msg)
	m.notice, noticeCmd = m.notice.Update(msg)
	return m, tui.Batch(spinCmd, loadCmd, noticeCmd)
}

var label = ansi.NewStyle().Bold()

// contentWidth is the width of the dashboard's inner content: the dividers and
// header, which are pre-rendered strings that need a width, are drawn at it.
const contentWidth = 32

// wideMin is the narrowest width that gets the side-by-side grid; wideContentWidth
// is the wide layout's inner width (box border and padding take 4 more).
const (
	wideMin            = 100
	mediumMin          = 60
	mediumContentWidth = 70
	wideContentWidth   = 96
)

// statusRow is one line of the status table: a name with its indicator, and a
// badge. The grid aligns the badges, so no name needs padding by hand.
type statusRow struct {
	name    string
	variant widgets.Variant
	badge   string
}

var statusRows = []statusRow{
	{"API", widgets.VariantSuccess, "healthy"},
	{"Worker Queue", widgets.VariantWarning, "degraded"},
	{"Database", widgets.VariantError, "down"},
}

// metricRow is one line of the metrics table: a label, a graph and its value.
type metricRow struct {
	name  string
	graph string
	value float64 // percent
}

func (m model) metricRows(t theme.Theme) []metricRow {
	return []metricRow{
		{"CPU", chart.Sparkline(m.cpu), m.cpu[len(m.cpu)-1]},
		{"Mem", chart.Sparkline(m.mem), m.mem[len(m.mem)-1]},
		{"Disk", widgets.ProgressBar(m.disk, 15, t), m.disk * 100},
	}
}

// screen builds the dashboard as a layout.Node tree. Each group is a Column
// and the groups are separated by one blank row; the two tables are grids whose
// columns line up whatever the cells contain, styled or not.
func (m model) screen() layout.Node {
	t := theme.DarkTheme()
	child := func(n layout.Node) layout.FlexChild { return layout.FlexChild{Node: n} }

	// Two aligned tables. GridNodeGaps sizes each column to its widest cell, so
	// the columns line up without padding any text; one blank column between
	// columns and no blank rows between rows.
	var statusCells []layout.Node
	for _, r := range statusRows {
		statusCells = append(statusCells,
			layout.Block(widgets.StatusIndicator(r.name, r.variant, t)),
			layout.Block(widgets.Badge(r.badge, r.variant, t)))
	}
	var metricCells []layout.Node
	for _, r := range m.metricRows(t) {
		metricCells = append(metricCells,
			layout.Block(label.Render(r.name)),
			layout.Block(r.graph),
			layout.Block(fmt.Sprintf("%5.1f%%", r.value)))
	}

	header := func(w int) layout.Node {
		return layout.Column(0,
			child(layout.Block(widgets.HeaderWithAccessory("Service Dashboard", "v2.1.0", w, t))),
			child(layout.Block(widgets.DividerLabel(w, "status"))),
		)
	}
	status := layout.GridNodeGaps([]layout.Track{{}, {}}, 1, 0, statusCells...)
	deploy := layout.Column(0,
		child(layout.Block(widgets.Stepper([]string{"Build", "Test", "Deploy"}, 1, t))),
		child(layout.Row(0,
			child(m.spin.LayoutNode()),
			child(layout.Block(" Deploying build  ")),
			child(m.loading.LayoutNode()),
		)),
	)
	metrics := layout.GridNodeGaps([]layout.Track{{}, {}, {}}, 1, 0, metricCells...)
	footer := func(w int) layout.Node {
		return layout.Column(0,
			child(layout.Block(widgets.Divider(w))),
			child(layout.Block(widgets.KeyHints("  ",
				widgets.Hint{Key: "r", Action: "refresh"},
				widgets.Hint{Key: "d", Action: "about"},
				widgets.Hint{Key: "q", Action: "quit"}))),
		)
	}
	box := func(body layout.Node) layout.Node {
		return layout.BoxNode(layout.NewBox().Border(t.Border).PaddingAll(1), body)
	}

	// Below wideMin columns the panels stack in one column; from it up they sit
	// side by side in a grid.
	narrow := box(layout.Column(1,
		child(header(contentWidth)),
		child(status),
		child(deploy),
		child(layout.Block(widgets.DividerLabel(contentWidth, "metrics"))),
		child(metrics),
		child(footer(contentWidth)),
	))
	wide := box(layout.Column(1,
		child(header(wideContentWidth)),
		child(layout.Row(6,
			child(layout.Column(1, child(status), child(deploy))),
			child(layout.Column(1, child(layout.Block(label.Render("metrics"))), child(metrics))),
		)),
		child(footer(wideContentWidth)),
	))
	// From mediumMin up to wideMin-1 columns the status and deploy panels sit
	// side by side above the metrics, in a box wider than the narrow one.
	medium := box(layout.Column(1,
		child(header(mediumContentWidth)),
		child(layout.Row(4, child(status), child(deploy))),
		child(layout.Block(widgets.DividerLabel(mediumContentWidth, "metrics"))),
		child(metrics),
		child(footer(mediumContentWidth)),
	))
	return layout.Responsive([]layout.Break{{MaxW: mediumMin - 1}, {MaxW: wideMin - 1}}, narrow, medium, wide)
}

func (m model) View() string {
	maxW := m.width
	if maxW <= 0 {
		maxW = mediumMin - 1
	}
	base := layout.Draw(m.screen(), layout.Constraints{MaxW: maxW, MaxH: layout.Unbounded})
	return m.notice.Render(m.about.Render(base))
}

func main() {
	if _, err := tui.NewProgram(initialModel()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
