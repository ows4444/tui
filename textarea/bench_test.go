package textarea

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
)

// BenchmarkTyping measures typing one rune into a focused 1,000-line buffer.
func BenchmarkTyping(b *testing.B) {
	m := New()
	m.SetValue(strings.Repeat("the quick brown fox jumps over the lazy dog\n", 1000))
	m.Focus()
	key := tui.Key{Type: tui.KeyRunes, Text: "x"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m, _ = m.Update(key)
	}
}

// BenchmarkView measures View() over a 1,000-line buffer.
func BenchmarkView(b *testing.B) {
	m := New()
	m.SetValue(strings.Repeat("the quick brown fox jumps over the lazy dog\n", 1000))
	m.Focus()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}
