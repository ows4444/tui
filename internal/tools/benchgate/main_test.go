package main

import (
	"os"
	"strings"
	"testing"
)

const sampleOut = `goos: darwin
pkg: example.com/p
BenchmarkA-10   100   50.0 ns/op   100 B/op   4 allocs/op
BenchmarkA-10   100   52.0 ns/op   100 B/op   4 allocs/op
BenchmarkA-10   100   51.0 ns/op   100 B/op   4 allocs/op
BenchmarkNoMem-10   100   50.0 ns/op
PASS
`

func TestParseGroupsRunsAndStripsTheProcsSuffix(t *testing.T) {
	m, err := parse(strings.NewReader(sampleOut))
	if err != nil {
		t.Fatal(err)
	}
	s, ok := m["example.com/p.BenchmarkA"]
	if !ok || len(s.allocs) != 3 || median(s.allocs) != 4 || median(s.bytes) != 100 {
		t.Fatalf("parsed = %+v", m)
	}
	if _, ok := m["example.com/p.BenchmarkNoMem"]; ok {
		t.Error("a line without -benchmem columns was kept")
	}
}

func TestMedian(t *testing.T) {
	if got := median([]float64{5, 1, 3}); got != 3 {
		t.Errorf("odd median = %v", got)
	}
	if got := median([]float64{4, 1, 3, 2}); got != 2.5 {
		t.Errorf("even median = %v", got)
	}
}

func mk(allocs, bytes float64) sample {
	return sample{allocs: []float64{allocs}, bytes: []float64{bytes}}
}

func TestCompareFlagsGrowthPastTheThreshold(t *testing.T) {
	base := map[string]sample{"p.Same": mk(10, 1000), "p.Alloc": mk(10, 1000), "p.Bytes": mk(10, 1000), "p.Gone": mk(1, 1), "p.Zero": mk(0, 0)}
	cur := map[string]sample{
		"p.Same":  mk(11, 1100), // exactly +10%: allowed
		"p.Alloc": mk(12, 1000), // +20% allocs
		"p.Bytes": mk(10, 1300), // +30% bytes
		"p.Zero":  mk(1, 0),     // 0 -> 1 alloc
		"p.New":   mk(1, 1),
	}
	problems, notes := compare(base, cur, 10)
	joined := strings.Join(problems, "\n")
	for _, want := range []string{"p.Alloc allocs/op 10 -> 12", "p.Bytes B/op 1000 -> 1300", "p.Gone is in the baseline", "p.Zero allocs/op 0 -> 1"} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing problem %q in:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, "p.Same") || len(problems) != 4 {
		t.Errorf("unexpected problems:\n%s", joined)
	}
	if len(notes) != 1 || !strings.Contains(notes[0], "p.New") {
		t.Errorf("notes = %v", notes)
	}
}

func TestSmallByteNoiseIsAllowed(t *testing.T) {
	problems, _ := compare(map[string]sample{"p.X": mk(0, 0)}, map[string]sample{"p.X": mk(0, 60)}, 10)
	if len(problems) != 0 {
		t.Errorf("60 B of noise flagged: %v", problems)
	}
}

func TestLoadRejectsEmptyOutput(t *testing.T) {
	if _, err := load("/nonexistent/file"); err == nil {
		t.Error("want an error for a missing file")
	}
}

// TestBenchWorkflowInstallsNothingOutsideTheRepo proves the workflow runs no
// `go install` and no external benchstat, so CI depends only on this module.
func TestBenchWorkflowInstallsNothingOutsideTheRepo(t *testing.T) {
	b, err := os.ReadFile("../../../.github/workflows/bench.yml")
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range strings.Split(string(b), "\n") {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "#") {
			continue
		}
		for _, bad := range []string{"go install", "go get", "benchstat", "golang.org/x/perf"} {
			if strings.Contains(l, bad) {
				t.Errorf("bench.yml line %q uses %q", l, bad)
			}
		}
	}
}
