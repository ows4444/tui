package logview

import "testing"

// BenchmarkAppend100k measures appending 100,000 lines to an empty log, the
// "streaming logs" case. It must stay linear in the line count.
func BenchmarkAppend100k(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m := New(120, 40)
		for j := 0; j < 100000; j++ {
			m.Append("2026-09-29T12:00:00Z INFO request handled in 12ms path=/api/v1/items")
		}
	}
}

// BenchmarkView measures View() over a 100,000-line log pinned to the bottom.
func BenchmarkView(b *testing.B) {
	m := New(120, 40)
	for j := 0; j < 100000; j++ {
		m.Append("2026-09-29T12:00:00Z INFO request handled in 12ms path=/api/v1/items")
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}
