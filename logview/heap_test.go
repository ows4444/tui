//go:build !race

package logview

import (
	"fmt"
	"runtime"
	"testing"
)

// Criterion #86: appending 1M lines with Max=10000 keeps live heap below 32 MB.
func TestMillionLinesStayBounded(t *testing.T) {
	if testing.Short() {
		t.Skip("1M-line heap bound skipped in -short mode")
	}
	const total = 1_000_000
	m := New(120, 40)
	m.Max = 10000
	for i := 0; i < total; i++ {
		m.Append(fmt.Sprintf("2026-09-29T12:00:00Z INFO request %d handled", i))
	}
	if got := m.Viewport.LineCount(); got > m.Max {
		t.Fatalf("kept %d lines, want at most %d", got, m.Max)
	}
	var ms runtime.MemStats
	runtime.GC()
	runtime.GC()
	runtime.ReadMemStats(&ms)
	const limit = 32 << 20
	if ms.HeapAlloc >= limit {
		t.Fatalf("live heap %d bytes after %d appends, want below %d", ms.HeapAlloc, total, limit)
	}
	runtime.KeepAlive(m)
}
