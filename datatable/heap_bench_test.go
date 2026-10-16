package datatable

import (
	"fmt"
	"runtime"
	"testing"
)

// liveBytes is the heap in use after a collection.
func liveBytes() int64 {
	runtime.GC()
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return int64(ms.HeapAlloc)
}

// BenchmarkHeap100kRows reports how much memory a 100,000-row, 4-column table
// holds: rows-B for the strings alone, and model-B for what the table adds on
// top of them (its width cache and the rest), after drawing a frame.
func BenchmarkHeap100kRows(b *testing.B) {
	const n = 100000
	var rowsB, modelB float64
	for i := 0; i < b.N; i++ {
		before := liveBytes()
		rows := make([][]string, n)
		for r := range rows {
			rows[r] = []string{fmt.Sprintf("r%06d", r), "name", "state", "a note about the row"}
		}
		withRows := liveBytes()
		m := New([]string{"ID", "Name", "State", "Notes"}, rows)
		m.Height = 22
		_ = m.View()
		withModel := liveBytes()
		rowsB += float64(withRows - before)
		modelB += float64(withModel - withRows)
		runtime.KeepAlive(m)
		runtime.KeepAlive(rows)
	}
	b.ReportMetric(rowsB/float64(b.N), "rows-B")
	b.ReportMetric(modelB/float64(b.N), "model-B")
}
