package textarea

import (
	"math/rand"
	"strings"
	"testing"
)

// TestDocPieceTableMatchesFlat checks random single-line edits against a plain
// rune slice, and that an older doc is untouched by later edits.
func TestDocPieceTableMatchesFlat(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	flat := []rune(strings.Repeat("abcdefghij", 500))
	d := newDoc(flat)
	for step := 0; step < 3000; step++ {
		old, oldFlat := d, string(flat)
		from := rng.Intn(len(flat) + 1)
		to := min(len(flat), from+rng.Intn(4))
		ins := []rune(strings.Repeat("xyz", rng.Intn(3)))
		d = d.replace(from, to, ins)
		flat = append(append(append([]rune{}, flat[:from]...), ins...), flat[to:]...)
		if d.len() != len(flat) || d.String() != string(flat) {
			t.Fatalf("step %d: doc diverged from flat text", step)
		}
		if old.String() != oldFlat {
			t.Fatalf("step %d: earlier doc was modified", step)
		}
		if len(flat) > 0 {
			if i := rng.Intn(len(flat)); d.at(i) != flat[i] {
				t.Fatalf("step %d: at(%d) mismatch", step, i)
			}
		}
	}
}

// TestViewFullAllocs pins the full View of a 1k-line buffer to a constant
// number of allocations.
func TestViewFullAllocs(t *testing.T) {
	m := New()
	m.SetValue(strings.Repeat("the quick brown fox jumps over the lazy dog\n", 1000))
	m.Focus()
	if n := testing.AllocsPerRun(5, func() { _ = m.View() }); n > 8 {
		t.Fatalf("View allocs = %v, want <= 8", n)
	}
}
