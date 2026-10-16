package textarea

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

// BenchmarkHeap100kLines reports the memory a text area holds for 100,000 lines
// of about 60 bytes: text-B for the text itself and model-B for what the model
// holds beyond it, after drawing a frame.
func BenchmarkHeap100kLines(b *testing.B) {
	line := "line 000000 the quick brown fox jumps over the lazy dog 0123456\n"
	var textB, modelB float64
	for i := 0; i < b.N; i++ {
		before := liveBytes()
		text := strings.Repeat(line, 100000)
		withText := liveBytes()
		m := New()
		m.Height = 24
		m.SetValue(text)
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
