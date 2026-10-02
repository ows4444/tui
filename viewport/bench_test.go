package viewport

import (
	"strings"
	"testing"
)

func benchContent(n int) string {
	line := "2026-09-29T12:00:00Z INFO request handled in 12ms path=/api/v1/items"
	return strings.Repeat(line+"\n", n-1) + line
}

// BenchmarkSetContent10k measures replacing the content with 10,000 lines.
func BenchmarkSetContent10k(b *testing.B) {
	m := New(120, 40)
	s := benchContent(10000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.SetContent(s)
	}
}

// BenchmarkView10k measures View() over 10,000 lines scrolled to the middle;
// cost should track the window height, not the line count.
func BenchmarkView10k(b *testing.B) {
	m := New(120, 40)
	m.SetContent(benchContent(10000))
	m.LineDown(5000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}

// BenchmarkSetContent10kRaw is SetContent10k with sanitizing off: the
// pre-sanitize cost, the reference for the fast path on clean text.
func BenchmarkSetContent10kRaw(b *testing.B) {
	m := New(120, 40)
	m.Raw = true
	s := benchContent(10000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.SetContent(s)
	}
}
