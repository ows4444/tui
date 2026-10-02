package markdown

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/theme"
)

// BenchmarkRender renders a 200-block document at 80 columns.
func BenchmarkRender(b *testing.B) {
	doc := strings.Repeat("# Heading\n\nSome *emphasis*, **bold** and `code` in a paragraph that is long enough to wrap across the width.\n\n- item one\n- item two\n\n```go\nfmt.Println(\"hi\")\n```\n\n", 50)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Render(doc, 80, theme.DarkTheme())
	}
}
