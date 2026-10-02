package ansi

import (
	"math/rand"
	"strings"
	"testing"
	"unicode/utf8"
)

// slowTruncate and slowTrimLeftWidth are Truncate and TrimLeftWidth without
// the printable-ASCII run fast path (the segmenter on every rune), kept as the
// oracle the fast path must match byte for byte.
func slowTruncate(s string, width int) string {
	if width <= 0 {
		return ""
	}

	// Everything Truncate keeps (escape sequences and the runes that fit) is a
	// verbatim prefix of s, so scan to find where it ends and slice, instead
	// of copying into a new string. Only leaving styling open costs an
	// allocation, for the Reset appended after it.
	clusters := clusterWidthOn.Load()
	var st clusterState
	visible, pending := 0, 0 // finished clusters, and the cluster in progress
	open, openAtStart := false, false
	clusterStart := 0
	cut := len(s)
	for i := 0; i < len(s); {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			terminated := false
			for j < len(s) {
				c := s[j]
				j++
				if c >= 0x40 && c <= 0x7E {
					terminated = true
					break
				}
			}
			if !terminated {
				// A CSI cut off by the end of the string: keeping it would
				// fuse with whatever follows (e.g. the Reset appended
				// below), so drop it.
				cut = i
				break
			}
			open = s[i:j] != Reset
			i = j
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		rw := runeWidth(r)
		if clusters && st.next(r) {
			// r joins the cluster in progress; it stays whole or goes whole.
			grown := st.joinWidth(pending, rw, r)
			if visible+grown > width {
				cut, open = clusterStart, openAtStart
				break
			}
			pending = grown
		} else {
			visible += pending
			if visible+rw > width {
				cut = i
				break
			}
			pending, clusterStart, openAtStart = rw, i, open
		}
		i += size
	}
	if open {
		return s[:cut] + Reset
	}
	return s[:cut]
}

func slowTrimLeftWidth(s string, width int) string {
	if width <= 0 {
		return s
	}

	clusters := clusterWidthOn.Load()
	var st clusterState
	var active string
	done, pending := 0, 0 // finished clusters, and the cluster in progress
	i := 0
	for i < len(s) {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) {
				c := s[j]
				j++
				if c >= 0x40 && c <= 0x7E {
					break
				}
			}
			seq := s[i:j]
			if seq == Reset {
				active = ""
			} else {
				active = seq
			}
			i = j
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		rw := runeWidth(r)
		if clusters && st.next(r) {
			// The rest of a cluster that began inside the trimmed part goes
			// with it.
			pending = st.joinWidth(pending, rw, r)
		} else {
			if done+pending >= width {
				break
			}
			done += pending
			pending = rw
		}
		i += size
	}

	pad := ""
	if over := done + pending - width; over > 0 {
		pad = strings.Repeat(" ", over)
	}
	return active + pad + s[i:]
}

var fastPieces = []string{"", "a", "hello world", "x", "  ", "~", "\x7f", "\x1f", "你好", "🎉", "é", "e\u0301",
	"\u200d", "\u0600", "\u200b", "👋🏽", "🇺🇸", "\r\n", "\t",
	"\x1b[1m", "\x1b[0m", "\x1b[38;2;1;2;3m", "\x1b[", "\x1b[31", "\xe2", "\xff"}

func checkFast(t testing.TB, s string, width int) {
	t.Helper()
	for _, on := range []bool{true, false} {
		SetClusterWidth(on)
		if got, want := Truncate(s, width), slowTruncate(s, width); got != want {
			t.Fatalf("clusters=%v Truncate(%q, %d) = %q, want %q", on, s, width, got, want)
		}
		if got, want := TrimLeftWidth(s, width), slowTrimLeftWidth(s, width); got != want {
			t.Fatalf("clusters=%v TrimLeftWidth(%q, %d) = %q, want %q", on, s, width, got, want)
		}
	}
	SetClusterWidth(true)
}

func TestASCIIFastPathMatchesSlowPath(t *testing.T) {
	defer SetClusterWidth(true)
	r := rand.New(rand.NewSource(28))
	for i := 0; i < 200000; i++ {
		var sb strings.Builder
		for j, n := 0, r.Intn(10); j < n; j++ {
			if r.Intn(3) == 0 {
				sb.WriteString(strings.Repeat("ab ~", r.Intn(6)))
			}
			sb.WriteString(fastPieces[r.Intn(len(fastPieces))])
		}
		s := sb.String()
		if !utf8.ValidString(s) && r.Intn(2) == 0 {
			continue
		}
		checkFast(t, s, r.Intn(30)-2)
	}
	for w := -1; w < 100; w++ {
		checkFast(t, strings.Repeat("abcdefgh", 12), w)
		checkFast(t, "\x1b[1m"+strings.Repeat("abcdefgh", 12)+"\x1b[0m", w)
	}
}

func FuzzASCIIFastPathMatchesSlowPath(f *testing.F) {
	for _, p := range fastPieces {
		f.Add(p+"abc"+p, 2)
	}
	f.Add("\u0600abc", 2)
	f.Add("a\u200d🎉bcd", 3)
	f.Fuzz(func(t *testing.T, s string, width int) {
		checkFast(t, s, width%200)
	})
}

func BenchmarkTruncateASCII(b *testing.B) {
	s := strings.Repeat("abcdefgh", 12) // 96 bytes
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		Truncate(s, 80)
	}
}

func BenchmarkTrimLeftWidthASCII(b *testing.B) {
	s := strings.Repeat("abcdefgh", 12)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		TrimLeftWidth(s, 40)
	}
}
