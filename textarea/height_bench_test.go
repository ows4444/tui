package textarea

import (
	"strings"
	"testing"
)

// BenchmarkViewWindowed measures View() over a 100,000-line buffer with
// Height 40 and the cursor on the last line.
func BenchmarkViewWindowed(b *testing.B) {
	m := New()
	m.Height = 40
	m.SetValue(strings.Repeat("the quick brown fox jumps over the lazy dog\n", 100000))
	m.Focus()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}
