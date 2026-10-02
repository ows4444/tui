package drawer

import (
	"fmt"
	"strings"
	"time"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/motion"
)

// A drawer that slides in from its edge. In a real program SlideIn's Cmd is
// returned from Update and the Program delivers each tick back to the
// drawer's Update; here the ticks are fed by hand at fixed times, so the
// output does not depend on the clock. The first tick fixes the start; the
// drawer is fully in place exactly one SlideDuration later.
func Example() {
	base := strings.TrimRight(strings.Repeat(strings.Repeat(".", 24)+"\n", 5), "\n")

	m := New("menu")
	m.Hide()
	m.Edge = EdgeRight
	m.Width = 8
	m.SlideDuration = 100 * time.Millisecond
	m.Ease = motion.Linear

	m.SlideIn() // in a program: return this Cmd from Update

	start := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	for _, ms := range []int{0, 50, 100} {
		m, _ = m.Update(slideTickMsg{gen: m.gen, at: start.Add(time.Duration(ms) * time.Millisecond)})
		fmt.Printf("t=%dms, %d%% in:\n%s\n\n", ms, int(m.shown()*100+0.5), plainRows(m.Render(base)))
	}

	// Output:
	// t=0ms, 0% in:
	// ........................
	// ........................
	// ........................
	// ........................
	// ........................
	//
	// t=50ms, 50% in:
	// ..................┌─────
	// ..................│
	// ..................│ menu
	// ..................│
	// ..................└─────
	//
	// t=100ms, 100% in:
	// ............┌──────────┐
	// ............│          │
	// ............│ menu     │
	// ............│          │
	// ............└──────────┘
}

// plainRows strips the styling and trailing spaces so the output reads cleanly.
func plainRows(s string) string {
	rows := strings.Split(ansi.StripANSI(s), "\n")
	for i, r := range rows {
		rows[i] = strings.TrimRight(r, " ")
	}
	return strings.Join(rows, "\n")
}
