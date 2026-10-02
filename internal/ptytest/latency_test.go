//go:build darwin || linux

package ptytest

import (
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestPercentile(t *testing.T) {
	ms := func(n ...int) []time.Duration {
		out := make([]time.Duration, len(n))
		for i, v := range n {
			out[i] = time.Duration(v) * time.Millisecond
		}
		return out
	}
	hundred := make([]time.Duration, 100)
	for i := range hundred {
		hundred[i] = time.Duration(i+1) * time.Millisecond
	}
	cases := []struct {
		name string
		in   []time.Duration
		p    float64
		want time.Duration
	}{
		{"empty", nil, 50, 0},
		{"single", ms(7), 99, 7 * time.Millisecond},
		{"p50 odd", ms(30, 10, 20), 50, 20 * time.Millisecond},
		{"p50 even nearest rank", ms(40, 10, 30, 20), 50, 20 * time.Millisecond},
		{"p99 of 100", hundred, 99, 99 * time.Millisecond},
		{"p0 clamps to min", ms(5, 9), 0, 5 * time.Millisecond},
		{"p100 is max", ms(5, 9), 100, 9 * time.Millisecond},
	}
	for _, c := range cases {
		orig := append([]time.Duration(nil), c.in...)
		if got := Percentile(c.in, c.p); got != c.want {
			t.Errorf("%s: Percentile = %v, want %v", c.name, got, c.want)
		}
		for i := range orig {
			if orig[i] != c.in[i] {
				t.Errorf("%s: Percentile mutated its input", c.name)
				break
			}
		}
	}
}

func TestSummarize(t *testing.T) {
	s := Summarize([]time.Duration{3 * time.Millisecond, time.Millisecond, 2 * time.Millisecond})
	if s.N != 3 || s.Min != time.Millisecond || s.Max != 3*time.Millisecond || s.P50 != 2*time.Millisecond {
		t.Errorf("Summarize = %+v", s)
	}
	if got := Summarize(nil); got.N != 0 || got.String() == "" {
		t.Errorf("empty Summarize = %+v %q", got, got.String())
	}
}

// TestMeasureShell measures a trivial process, proving the harness mechanics
// (pty, first byte, key echo) without the repo's toolchain. It asserts that
// measurements are positive, never that they are fast.
func TestMeasureShell(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not found")
	}
	m, s, err := Open()
	if err != nil {
		t.Skipf("cannot open a pseudo-terminal (/dev/ptmx unavailable in this environment): %v", err)
	}
	_ = m.Close()
	_ = s.Close()
	cfg := Config{Path: sh, Args: []string{"-c", "printf ready; exec cat"}, Timeout: 20 * time.Second}
	d, err := ColdStart(cfg, nil)
	if err != nil {
		t.Fatalf("ColdStart: %v", err)
	}
	if d <= 0 {
		t.Errorf("cold start = %v, want > 0", d)
	}
	k, err := KeyLatency(cfg, []byte("k"), 50*time.Millisecond)
	if err != nil {
		t.Fatalf("KeyLatency: %v", err)
	}
	if k <= 0 {
		t.Errorf("key latency = %v, want > 0", k)
	}
}

// TestCounterLatencyReport builds examples/counter and reports cold-start and
// key-to-byte p50/p99 over a few runs. It reports with t.Log (run with -v);
// it never asserts a time bound, so a slow machine cannot fail it.
func TestCounterLatencyReport(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs the counter example")
	}
	bin := filepath.Join(t.TempDir(), "counter")
	build := exec.Command("go", "build", "-o", bin, "../../examples/counter")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	cfg := Config{Path: bin, Cols: 80, Rows: 24, Timeout: 15 * time.Second}
	cold, key, err := Collect(cfg, []byte("\x1b[A"), 15, 100*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if len(cold) == 0 || len(key) == 0 {
		t.Fatalf("no samples: cold=%d key=%d", len(cold), len(key))
	}
	t.Logf("counter cold start to first frame: %s", Summarize(cold))
	t.Logf("counter key to first byte:         %s", Summarize(key))
}
