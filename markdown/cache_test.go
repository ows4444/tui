package markdown

import (
	"runtime"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

const cacheDoc = "# Title\n\nSome *emphasis*, **bold** and `code` in a paragraph long enough to wrap.\n\n- a\n- b\n\n```go\nx := 1\n```\n"

func TestRenderCacheSecondCallAllocatesUnder10Percent(t *testing.T) {
	doc := strings.Repeat(cacheDoc, 20) + "unique-alloc-test"
	mallocs := func() uint64 {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		return m.Mallocs
	}
	before := mallocs()
	_ = Render(doc, 61, theme.DarkTheme())
	mid := mallocs()
	_ = Render(doc, 61, theme.DarkTheme())
	after := mallocs()
	first, second := mid-before, after-mid
	t.Logf("first=%d second=%d", first, second)
	if second*10 >= first {
		t.Fatalf("second call allocs %d, want < 10%% of first %d", second, first)
	}
}

func TestRenderCacheInvalidatesOnStyleAndWidth(t *testing.T) {
	doc := cacheDoc + "invalidate-test"
	a := Render(doc, 40, theme.DarkTheme())
	if b := Render(doc, 40, theme.DarkTheme()); a != b {
		t.Fatal("identical input differs")
	}
	th := theme.DarkTheme()
	th.Primary = ansi.Color256(200)
	if Render(doc, 40, th) == a {
		t.Fatal("style change served stale render")
	}
	if Render(doc, 20, theme.DarkTheme()) == a {
		t.Fatal("width change served stale render")
	}
	if Render(doc, 40, theme.DarkTheme()) != a {
		t.Fatal("original no longer reproducible")
	}
}
