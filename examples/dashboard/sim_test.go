package main

import (
	"math"
	"testing"

	"github.com/ows4444/tui/theme"
)

func TestWorldIsDeterministic(t *testing.T) {
	a, b := newWorld(), newWorld()
	for i := 0; i < 50; i++ {
		a.step()
		b.step()
	}
	if a.cur != b.cur || a.load != b.load {
		t.Fatal("two worlds with the same seed differ")
	}
}

func TestWorldStaysInRange(t *testing.T) {
	w := newWorld()
	for i := 0; i < 5000; i++ {
		w.step()
		c := w.cur
		if c.cpu < 1 || c.cpu > 99 || c.memGB < 3 || c.memGB > memTotalGB || c.cached < 0 || c.swapGB < 0 || c.swapGB > swapTotalGB || c.rx < 2e3 || c.rx > 5e7 || c.tx < 1e3 || c.tx > 2e7 {
			t.Fatalf("step %d out of range: %+v", i, c)
		}
		for _, x := range c.core {
			if x < 0 || x > 100 || math.IsNaN(x) {
				t.Fatalf("core load %v", x)
			}
		}
		for _, d := range w.disks {
			if d.usedGB > d.totalGB {
				t.Fatalf("disk %s over full: %v of %v", d.mount, d.usedGB, d.totalGB)
			}
		}
	}
}

func TestHistoryIsBoundedAndStartsFull(t *testing.T) {
	w := newWorld()
	if len(w.hist) != history {
		t.Fatalf("history starts with %d samples", len(w.hist))
	}
	for i := 0; i < 300; i++ {
		w.step()
	}
	if len(w.hist) != history {
		t.Fatalf("history grew to %d", len(w.hist))
	}
	if got := w.series(func(s sample) float64 { return s.cpu }); len(got) != history || got[len(got)-1] != w.cur.cpu {
		t.Fatal("series does not end on the current sample")
	}
}

func TestTopIsSortedAndBounded(t *testing.T) {
	w := newWorld()
	top := w.top(4)
	if len(top) != 4 {
		t.Fatalf("top(4) = %d", len(top))
	}
	for i := 1; i < len(top); i++ {
		if top[i].cpu > top[i-1].cpu {
			t.Fatalf("not sorted: %+v", top)
		}
	}
	if len(w.top(100)) != len(w.procs) || len(w.top(0)) != 0 {
		t.Fatal("top does not clamp")
	}
	before := w.procs[0].pid
	_ = w.top(3)
	if w.procs[0].pid != before {
		t.Fatal("top reordered the process list")
	}
}

func TestFormatters(t *testing.T) {
	for in, want := range map[float64]string{12: "12 B/s", 1500: "2 KB/s", 2.5e6: "2.5 MB/s"} {
		if got := rate(in); got != want {
			t.Errorf("rate(%v) = %q, want %q", in, got, want)
		}
	}
	if bytesTotal(3e9) != "3.0 GB" || bytesTotal(2e6) != "2 MB" || bytesTotal(500) != "0 KB" {
		t.Errorf("bytesTotal: %s %s %s", bytesTotal(3e9), bytesTotal(2e6), bytesTotal(500))
	}
	if gb(500) != "500G" || gb(1710) != "1.7T" {
		t.Errorf("gb: %s %s", gb(500), gb(1710))
	}
}

func themeForTest() theme.Theme { return theme.DarkTheme() }
