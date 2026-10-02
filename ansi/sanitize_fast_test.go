package ansi

import (
	"math/rand"
	"sort"
	"strings"
	"testing"
	"time"
)

func needsSanitizeRef(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c < 0x20 && c != '\n') || c == 0x7f || c == 0xc2 {
			return true
		}
	}
	return false
}

// The word-at-a-time scan must agree with the byte loop on every input,
// including each trigger byte at every offset of a word and bytes near the
// boundaries (0x1f, 0x20, 0x7e, 0x80, 0xc1, 0xc3).
func TestNeedsSanitizeMatchesReference(t *testing.T) {
	bytesToTry := []byte{0, 0x09, '\n', 0x1b, 0x1f, 0x20, 'a', 0x7e, 0x7f, 0x80, 0xc1, 0xc2, 0xc3, 0xff}
	for n := 0; n <= 20; n++ {
		for pos := 0; pos < n; pos++ {
			for _, c := range bytesToTry {
				b := []byte(strings.Repeat("a", n))
				b[pos] = c
				if got, want := needsSanitize(string(b)), needsSanitizeRef(string(b)); got != want {
					t.Fatalf("n=%d pos=%d byte=%#x: got %v, want %v", n, pos, c, got, want)
				}
			}
		}
	}
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 20000; i++ {
		b := make([]byte, rng.Intn(40))
		for j := range b {
			if rng.Intn(3) == 0 {
				b[j] = bytesToTry[rng.Intn(len(bytesToTry))]
			} else {
				b[j] = byte(0x20 + rng.Intn(0x5f))
			}
		}
		if got, want := needsSanitize(string(b)), needsSanitizeRef(string(b)); got != want {
			t.Fatalf("%q: got %v, want %v", b, got, want)
		}
	}
}

// Text with newlines but no control bytes is the common case and must not be
// meaningfully slower than the byte loop it replaced. Wall-clock comparisons
// are unreliable under the race detector and -short, so they are skipped there;
// the speed claim itself lives in BenchmarkNeedsSanitize.
func TestNeedsSanitizeNotSlowerThanByteLoop(t *testing.T) {
	if raceEnabled || testing.Short() {
		t.Skip("wall-clock comparison skipped under -race and -short; see BenchmarkNeedsSanitize")
	}
	s := needsSanitizeCorpus()
	median := func(f func(string) bool) time.Duration {
		const runs = 11
		ds := make([]time.Duration, 0, runs)
		for i := 0; i < runs; i++ {
			t0 := time.Now()
			if f(s) {
				t.Fatal("clean text reported dirty")
			}
			ds = append(ds, time.Since(t0))
		}
		sort.Slice(ds, func(a, b int) bool { return ds[a] < ds[b] })
		return ds[runs/2]
	}
	fast, ref := median(needsSanitize), median(needsSanitizeRef)
	t.Logf("word scan median %v, byte loop median %v", fast, ref)
	if fast > 2*ref {
		t.Errorf("word scan median %v more than 2x byte loop median %v", fast, ref)
	}
}

func needsSanitizeCorpus() string {
	line := "2026-09-29T12:00:00Z INFO request handled in 12ms path=/api/v1/items"
	return strings.Repeat(line+"\n", 10000)
}

func BenchmarkNeedsSanitize(b *testing.B) {
	s := needsSanitizeCorpus()
	for _, c := range []struct {
		name string
		f    func(string) bool
	}{{"word", needsSanitize}, {"byteloop", needsSanitizeRef}} {
		b.Run(c.name, func(b *testing.B) {
			b.SetBytes(int64(len(s)))
			for i := 0; i < b.N; i++ {
				if c.f(s) {
					b.Fatal("clean text reported dirty")
				}
			}
		})
	}
}
