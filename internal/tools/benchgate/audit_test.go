package main

import "testing"

// The audit benchmarks must be in the committed baseline so the gate compares
// them on every bench.yml run.
func TestBaselineHasAuditBenchmarks(t *testing.T) {
	base, err := load("../../../bench/baseline.txt")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"github.com/ows4444/tui.BenchmarkFrame200x60Styled",
		"github.com/ows4444/tui.BenchmarkOneCellChange200x60",
		"github.com/ows4444/tui/streamtext.BenchmarkStream1kTokensInline",
		"github.com/ows4444/tui/virtuallist.BenchmarkList10kWheelScroll",
		"github.com/ows4444/tui/textarea.BenchmarkTextareaTypeLongLine1MB",
	} {
		if len(base[name].allocs) == 0 {
			t.Errorf("%s missing from bench/baseline.txt", name)
		}
	}
}
