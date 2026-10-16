package celldraw

import (
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
)

func TestStyleIDOfThePlainStyleIsZero(t *testing.T) {
	buf := cellbuf.New(4, 1)
	id, ok := StyleID(buf, ansi.Style{})
	if !ok || id != 0 {
		t.Errorf("zero style = %d, %v", id, ok)
	}
	bold, ok := StyleID(buf, ansi.NewStyle().Bold())
	if !ok || bold == 0 {
		t.Errorf("bold = %d, %v", bold, ok)
	}
	again, _ := StyleID(buf, ansi.NewStyle().Bold())
	if again != bold {
		t.Errorf("the same style got two ids: %d and %d", bold, again)
	}
}

func TestStyleIDReportsAStyleTheGridCannotHold(t *testing.T) {
	saved := parse
	defer func() { parse = saved }()
	styleCache.Lock() // a conversion stored by an earlier test would answer instead of parse
	styleCache.m = nil
	styleCache.Unlock()
	parse = func(string) (*cellbuf.Buffer, error) { return nil, cellbuf.ErrUnsupported }
	if _, ok := StyleID(cellbuf.New(4, 1), ansi.NewStyle().Bold()); ok {
		t.Error("StyleID said ok for a style the grid cannot hold")
	}
}

func TestPlain(t *testing.T) {
	if !Plain("hello", "wörld 日本語 😀", "") {
		t.Error("printable text reported as not plain")
	}
	for _, s := range []string{"a\x1b[31mb", "tab\there", "nl\nx", "del\x7f", "\x00"} {
		if Plain("ok", s) {
			t.Errorf("Plain accepted %q", s)
		}
	}
}

func TestStyleIDCachesTheConversionAndBoundsTheCache(t *testing.T) {
	styleCache.Lock()
	styleCache.m = nil
	styleCache.Unlock()
	calls := 0
	saved := parse
	defer func() { parse = saved }()
	parse = func(s string) (*cellbuf.Buffer, error) { calls++; return cellbuf.Parse(s) }
	buf := cellbuf.New(2, 1)
	st := ansi.NewStyle().Italic()
	a, _ := StyleID(buf, st)
	b, _ := StyleID(buf, st)
	if a != b || calls != 1 {
		t.Errorf("ids %d and %d, %d parses; want equal ids and one parse", a, b, calls)
	}
	for i := 0; i < styleCacheMax+50; i++ {
		StyleID(buf, ansi.NewStyle().Foreground(ansi.RGB{R: uint8(i), G: uint8(i >> 8), B: 7}))
	}
	styleCache.RLock()
	n := len(styleCache.m)
	styleCache.RUnlock()
	if n > styleCacheMax {
		t.Errorf("cache holds %d entries, cap %d", n, styleCacheMax)
	}
}
