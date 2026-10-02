package tui

import (
	"os"
	"testing"

	"github.com/ows4444/tui/ansi"
)

// frameBudgetAllocs and frameBudgetNs are the budget of a steady-state frame
// of BenchmarkFrame200x60Styled with the cell renderer: a 200x60 screen of
// SGR-styled rows whose text changes on every row.
const (
	frameBudgetAllocs = 20
	frameBudgetNs     = 60_000
)

// TestFrame200x60StyledAllocs proves the allocation half of criterion #73: a
// steady-state styled frame makes at most 20 allocations.
func TestFrame200x60StyledAllocs(t *testing.T) {
	if raceEnabled {
		t.Skip("the race detector changes allocation counts")
	}
	r := testing.Benchmark(BenchmarkFrame200x60Styled)
	if got := r.AllocsPerOp(); got > frameBudgetAllocs {
		t.Fatalf("a steady-state 200x60 styled frame makes %d allocs, want <= %d", got, frameBudgetAllocs)
	}
}

// TestFrame200x60StyledTime proves the time half of criterion #73: at most
// 60 microseconds per frame. It is wall-clock time on the machine running it,
// so like the other timing tests it runs only when TUI_TIMING_TESTS is set.
func TestFrame200x60StyledTime(t *testing.T) {
	if testing.Short() || raceEnabled {
		t.Skip("timing test")
	}
	if os.Getenv("TUI_TIMING_TESTS") == "" {
		t.Skip("wall-clock test; set TUI_TIMING_TESTS=1 to run it")
	}
	best := int64(1 << 62)
	for i := 0; i < 7; i++ {
		if ns := testing.Benchmark(BenchmarkFrame200x60Styled).NsPerOp(); ns < best {
			best = ns
		}
	}
	if best > frameBudgetNs {
		t.Fatalf("a steady-state 200x60 styled frame takes %d ns, want <= %d", best, frameBudgetNs)
	}
	t.Logf("%d ns per frame (budget %d)", best, frameBudgetNs)
}

// TestFrame200x60StyledAllocsCriterion34 proves criterion #34: at most 7 allocs/op.
func TestFrame200x60StyledAllocsCriterion34(t *testing.T) {
	if raceEnabled {
		t.Skip("the race detector changes allocation counts")
	}
	if got := testing.Benchmark(BenchmarkFrame200x60Styled).AllocsPerOp(); got > 7 {
		t.Fatalf("BenchmarkFrame200x60Styled makes %d allocs/op, want <= 7", got)
	}
}

// TestFrame300x80StyledAllocs proves criterion #39: at most 8 allocs/op.
func TestFrame300x80StyledAllocs(t *testing.T) {
	if raceEnabled {
		t.Skip("the race detector changes allocation counts")
	}
	if got := testing.Benchmark(BenchmarkFrame300x80Styled).AllocsPerOp(); got > 8 {
		t.Fatalf("BenchmarkFrame300x80Styled makes %d allocs/op, want <= 8", got)
	}
}

// TestDowngradeStringSkippedWhenUnneeded proves criterion #33: no call at
// TrueColor, none for a view without CSI, but one when it can change output.
func TestDowngradeStringSkippedWhenUnneeded(t *testing.T) {
	calls := 0
	orig := downgradeString
	downgradeString = func(s string, pr ansi.Profile) string { calls++; return orig(s, pr) }
	t.Cleanup(func() { downgradeString = orig })

	styled := "\x1b[38;2;1;2;3mhi\x1b[0m"
	p := NewProgram(staticModel{}, WithColorProfile(ansi.TrueColor))
	if got := p.transformView(styled); got != styled || calls != 0 {
		t.Fatalf("TrueColor: got %q, %d calls; want unchanged, 0 calls", got, calls)
	}
	p = NewProgram(staticModel{}, WithColorProfile(ansi.ANSI256))
	if got := p.transformView("plain\ntext"); got != "plain\ntext" || calls != 0 {
		t.Fatalf("no SGR: %d calls, want 0", calls)
	}
	if got := p.transformView(styled); got == styled || calls != 1 {
		t.Fatalf("ANSI256 styled: got %q, %d calls; want rewritten, 1 call", got, calls)
	}
}
