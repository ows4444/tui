package markdown

import (
	"sync"

	"github.com/ows4444/tui/theme"
)

// cacheMax bounds the number of memoised renders; cacheMaxInput skips
// inputs too large to be worth holding on to.
const (
	cacheMax      = 64
	cacheMaxInput = 64 << 10
)

// cacheKey identifies a render. The whole Theme is part of the key, so any
// style change misses instead of returning stale output.
type cacheKey struct {
	md      string
	width   int
	t       theme.Theme
	builtin bool // Options.Lexer is DefaultLexer() (false: nil)
}

var renderCache = struct {
	sync.Mutex
	m map[cacheKey]string
}{m: make(map[cacheKey]string)}

// cacheable reports whether o renders deterministically from the key:
// only a nil Lexer or DefaultLexer(), since custom lexers may be stateful.
func cacheable(md string, o Options) (builtin, ok bool) {
	if len(md) > cacheMaxInput {
		return false, false
	}
	switch o.Lexer.(type) {
	case nil:
		return false, true
	case builtinLexer:
		return true, true
	}
	return false, false
}

func cacheGet(k cacheKey) (string, bool) {
	renderCache.Lock()
	s, ok := renderCache.m[k]
	renderCache.Unlock()
	return s, ok
}

func cachePut(k cacheKey, s string) {
	renderCache.Lock()
	if len(renderCache.m) >= cacheMax {
		clear(renderCache.m)
	}
	renderCache.m[k] = s
	renderCache.Unlock()
}
