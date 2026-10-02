package ansi

import (
	"math/rand"
	"strings"
	"testing"
	"unicode/utf8"
)

// refTruncate is the original Truncate (with the unterminated-CSI fix and OSC/DCS/APC/SOS/PM skipping),
// kept verbatim as the reference the allocation-free version must match.
func refTruncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	var b strings.Builder
	visible := 0
	open, link := false, false
	i := 0
	for i < len(s) {
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
				break
			}
			seq := s[i:j]
			b.WriteString(seq)
			open = seq != Reset
			i = j
			continue
		}
		if s[i] == 0x1b && i+1 < len(s) && strings.IndexByte("]PX^_", s[i+1]) >= 0 {
			j, end := i+2, -1
			for j < len(s) && end < 0 {
				if s[j] == 0x07 {
					end = j + 1
				} else if s[j] == 0x1b && j+1 < len(s) && s[j+1] == '\\' {
					end = j + 2
				}
				j++
			}
			if end < 0 {
				break
			}
			b.WriteString(s[i:end])
			if isLink, opens := osc8(s[i:end]); isLink {
				link = opens
			}
			i = end
			continue
		}
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] >= 0x30 && s[i+1] <= 0x7e {
			b.WriteString(s[i : i+2]) // two-byte sequence: ESC 7, ESC c, ...
			i += 2
			continue
		}
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] >= 0x20 && s[i+1] <= 0x2f {
			j := i + 2 // nF sequence: intermediates, then a final byte
			for j < len(s) && s[j] >= 0x20 && s[j] <= 0x2f {
				j++
			}
			if j < len(s) && s[j] >= 0x30 && s[j] <= 0x7e {
				b.WriteString(s[i : j+1])
				i = j + 1
				continue
			}
			if j >= len(s) {
				break // unterminated: runs to the end
			}
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		w := runeWidth(r)
		if visible+w > width {
			break
		}
		b.WriteString(s[i : i+size])
		visible += w
		i += size
	}
	if open {
		b.WriteString(Reset)
	}
	if link {
		b.WriteString(linkClose)
	}
	return b.String()
}

var truncPieces = []string{"", "a", "hello world", "你好", "🎉", "é", "é", "\u200b", "│", " ", "\t",
	"\x1b[1m", "\x1b[0m", "\x1b[m", "\x1b[38;2;1;2;3m", "\x1b[", "\x1b", "\x1b[31", "\x1b7", "\x1b(B", "\x1b(", "\x1b]8;;u\x07", "\xe2", "\x82\xac", "\xff", "\n"}

func TestTruncateMatchesReferenceOnRandomInputs(t *testing.T) {
	r := rand.New(rand.NewSource(3))
	for i := 0; i < 300000; i++ {
		var s string
		for j, n := 0, r.Intn(9); j < n; j++ {
			s += truncPieces[r.Intn(len(truncPieces))]
		}
		width := r.Intn(14) - 2 // includes zero and negative
		if got, want := Truncate(s, width), refTruncate(s, width); got != want {
			t.Fatalf("Truncate(%q, %d) = %q, want %q", s, width, got, want)
		}
	}
}

func FuzzTruncateMatchesReference(f *testing.F) {
	for _, p := range truncPieces {
		f.Add(p, 3)
	}
	f.Add("\x1b[1mhello\x1b[0m world", 7)
	f.Add("\x1b[31mred text that is cut", 5)
	// The reference counts rune by rune, so compare with clusters off.
	SetClusterWidth(false)
	f.Cleanup(func() { SetClusterWidth(true) })
	f.Fuzz(func(t *testing.T, s string, width int) {
		width %= 200
		if got, want := Truncate(s, width), refTruncate(s, width); got != want {
			t.Fatalf("Truncate(%q, %d) = %q, want %q", s, width, got, want)
		}
	})
}

// Text that already fits, with no styling left open, must come back without
// allocating: it is a substring of the input.
func TestTruncateFittingTextDoesNotAllocate(t *testing.T) {
	s := "\x1b[1mhello\x1b[0m world, this fits"
	if n := testing.AllocsPerRun(100, func() { _ = Truncate(s, 80) }); n != 0 {
		t.Errorf("Truncate of fitting text allocated %v times, want 0", n)
	}
}
