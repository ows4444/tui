package ansi

import (
	"strings"
	"testing"
)

// BenchmarkAnsiWidth measures Width on a representative mix of plain ASCII,
// wide (CJK) runes, and embedded ANSI styling — the hot path every render
// diff and layout join calls per line.
func BenchmarkAnsiWidth(b *testing.B) {
	s := NewStyle().Bold().Foreground(BasicColor(1)).Render("hello, 世界! " + strings.Repeat("x", 40))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Width(s)
	}
}

// BenchmarkAnsiWidthASCII measures Width on plain printable ASCII, the fast
// path that must stay at zero allocations (gated by internal/tools/benchgate).
func BenchmarkAnsiWidthASCII(b *testing.B) {
	s := strings.Repeat("hello, world! ", 8)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Width(s)
	}
}

// BenchmarkStyleRender measures Style.Render building a single styled,
// possibly multi-line string — called once per line by the diffed renderer
// and by every widget that styles text.
func BenchmarkStyleRender(b *testing.B) {
	style := NewStyle().Bold().Foreground(RGB{200, 100, 50}).Background(BasicColor(4))
	text := "line one of a styled block\nline two of a styled block\nline three"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		style.Render(text)
	}
}

// BenchmarkAnsiWidthWide measures Width over text that exercises the width
// tables: CJK, emoji, combining marks and a Khitan character, where the table
// lookup (not the ASCII fast path) decides each column.
func BenchmarkAnsiWidthWide(b *testing.B) {
	s := strings.Repeat("中文字符 😀🎉 éä 한국어 \U00018B00 ྙ", 4)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Width(s)
	}
}
