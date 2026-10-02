package textarea

import (
	"math/rand"
	"slices"
	"strings"
	"testing"

	"github.com/ows4444/tui"
)

// randomKey picks an edit or a cursor move, weighted toward edits that stay
// inside a line, which is the case the incremental index handles.
func randomKey(r *rand.Rand) tui.Msg {
	switch n := r.Intn(20); {
	case n < 7:
		return tui.Key{Type: tui.KeyRunes, Text: string([]rune{rune('a' + r.Intn(26))})}
	case n < 8:
		return tui.Key{Type: tui.KeyRunes, Text: "é日x"[:1+r.Intn(3)]}
	case n < 9:
		return tui.Key{Type: tui.KeySpace}
	case n < 10:
		return tui.Key{Type: tui.KeyEnter}
	case n < 13:
		return tui.Key{Type: tui.KeyBackspace}
	case n < 15:
		return tui.Key{Type: tui.KeyDelete}
	case n < 16:
		return tui.Key{Type: tui.KeyCtrl, Code: rune("ukw"[r.Intn(3)])}
	case n < 17:
		return tui.PasteEvent{Text: []string{"ab", "x\ny", "\n\n", "one two"}[r.Intn(4)]}
	case n < 18:
		return tui.Key{Type: tui.KeyUp}
	case n < 19:
		return tui.Key{Type: tui.KeyDown}
	default:
		return tui.Key{Type: []tui.KeyType{tui.KeyLeft, tui.KeyRight, tui.KeyHome, tui.KeyEnd}[r.Intn(4)]}
	}
}

// flatStarts is the reference line index: the rune offset of each line of s.
func flatStarts(s string) []int {
	starts := []int{0}
	for i, r := range []rune(s) {
		if r == '\n' {
			starts = append(starts, i+1)
		}
	}
	return starts
}

// docStarts reads the line starts back out of d.
func docStarts(d doc) []int {
	out := make([]int, d.lineCount())
	for i := range out {
		out[i] = d.lineStart(i)
	}
	return out
}

// After every edit of a random sequence the incremental index equals a full
// rebuild, and an earlier Model copy keeps the index it had.
func TestIncrementalIndexEqualsFullRebuild(t *testing.T) {
	for seed := int64(0); seed < 200; seed++ {
		r := rand.New(rand.NewSource(seed))
		m := New()
		m.Height = 1 + r.Intn(6)
		m.CharLimit = []int{0, 0, 40}[r.Intn(3)]
		m.SetValue(strings.Repeat("hello world\nfoo\n\nbar baz", 1+r.Intn(3)))
		m.Focus()
		type snap struct {
			m      Model
			starts []int
			text   string
		}
		var snaps []snap
		for step := 0; step < 150; step++ {
			if step%25 == 0 {
				snaps = append(snaps, snap{m, docStarts(m.value), m.Value()})
			}
			m, _ = m.Update(randomKey(r))
			want := flatStarts(m.Value())
			if got := docStarts(m.value); !slices.Equal(got, want) {
				t.Fatalf("seed %d step %d: index %v, full rebuild %v, value %q", seed, step, got, want, m.Value())
			}
			for l, s := range want {
				if got := m.value.lineOf(s); got != l {
					t.Fatalf("seed %d step %d: lineOf(%d) = %d, want %d", seed, step, s, got, l)
				}
			}
		}
		for i, s := range snaps {
			if !slices.Equal(docStarts(s.m.value), s.starts) || s.m.Value() != s.text {
				t.Fatalf("seed %d: copy %d's index was changed by later edits", seed, i)
			}
		}
	}
}

func TestSetValueAndResetDropTheIndex(t *testing.T) {
	m := New()
	m.Height = 3
	m.SetValue("a\nb\nc")
	m.SetValue("x")
	if got := docStarts(m.value); !slices.Equal(got, []int{0}) {
		t.Errorf("after SetValue starts = %v, want [0]", got)
	}
	m.Reset()
	if got := docStarts(m.value); !slices.Equal(got, []int{0}) || m.value.len() != 0 {
		t.Errorf("after Reset starts = %v, want [0]", got)
	}
}

const bigLines = 100_000

func bigModel() Model {
	m := New()
	m.Height = 20
	m.SetValue(strings.Repeat("the quick brown fox jumps over the lazy dog\n", bigLines))
	m.SetCursor(m.value.len() / 2)
	m.Focus()
	return m
}

// BenchmarkLineStarts100k is the full rebuild the index used to pay per edit.
func BenchmarkLineStarts100k(b *testing.B) {
	rs := []rune(bigModel().Value())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = newDoc(rs)
	}
}

// BenchmarkIndexEdit100k is the incremental index update for one in-line edit.
func BenchmarkIndexEdit100k(b *testing.B) {
	m := bigModel()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.value.replace(m.cursor, m.cursor, []rune{'x'})
	}
}

// BenchmarkTypingWindowed100k is a whole keystroke into a 100k-line buffer.
func BenchmarkTypingWindowed100k(b *testing.B) {
	m := bigModel()
	key := tui.Key{Type: tui.KeyRunes, Text: "x"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m, _ = m.Update(key)
	}
}
