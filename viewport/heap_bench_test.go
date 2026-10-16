package viewport

import (
	"runtime"
	"strings"
	"testing"
)

func liveBytes() int64 {
	runtime.GC()
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return int64(ms.HeapAlloc)
}

// BenchmarkHeap100kLogLines reports what a viewport holds for 100,000 log
// lines of about 60 bytes: text-B for the text, model-B for the viewport's own
// cost on top of it (the line slice), after drawing a frame.
func BenchmarkHeap100kLogLines(b *testing.B) {
	line := "2026-10-16 12:00:00 INFO request served path=/api/items id=0001234"
	var textB, modelB float64
	for i := 0; i < b.N; i++ {
		before := liveBytes()
		text := strings.Repeat(line+"\n", 100000)
		withText := liveBytes()
		m := New(80, 24)
		m.SetContent(text)
		_ = m.View()
		withModel := liveBytes()
		textB += float64(withText - before)
		modelB += float64(withModel - withText)
		runtime.KeepAlive(m)
		runtime.KeepAlive(text)
	}
	b.ReportMetric(textB/float64(b.N), "text-B")
	b.ReportMetric(modelB/float64(b.N), "model-B")
}
