package textarea

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
)

// BenchmarkTextareaTypeLongLine1MB measures typing one rune into the middle of
// a single 1 MiB line.
func BenchmarkTextareaTypeLongLine1MB(b *testing.B) {
	m := New()
	m.SetValue(strings.Repeat("the quick brown fox ", 1<<20/20))
	m.Focus()
	m.SetCursor(1 << 19)
	key := tui.Key{Type: tui.KeyRunes, Text: "x"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m, _ = m.Update(key)
	}
}
