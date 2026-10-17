package main

import (
	"math/rand"
	"testing"
)

func pids(ps []proc) []int {
	var out []int
	for _, p := range ps {
		out = append(out, p.pid)
	}
	return out
}

func TestSortByColumn(t *testing.T) {
	ps := sample()
	sortProcs(ps, colCPU, true)
	if ps[0].name != "compile" || ps[1].name != "go" {
		t.Fatalf("top by CPU = %s, %s", ps[0].name, ps[1].name)
	}
	sortProcs(ps, colName, false)
	if ps[0].name != "bluetoothd" {
		t.Fatalf("first by name = %s", ps[0].name)
	}
	sortProcs(ps, colPID, false)
	if ps[0].pid != 1 || ps[len(ps)-1].pid != 1630 {
		t.Fatalf("pid order wrong: %v", pids(ps))
	}
}

func TestSortTiesAreStable(t *testing.T) {
	ps := []proc{{pid: 9, cpu: 1}, {pid: 3, cpu: 1}, {pid: 5, cpu: 1}}
	sortProcs(ps, colCPU, true)
	if got := pids(ps); got[0] != 3 || got[1] != 5 || got[2] != 9 {
		t.Fatalf("equal CPU not ordered by pid: %v", got)
	}
}

func TestMatch(t *testing.T) {
	p := proc{pid: 702, name: "Code Helper", user: "mdawais"}
	for q, want := range map[string]bool{"": true, "code": true, "CODE help": true, "mdawais": true, "702": true, "postgres": false, "code 99": false} {
		if got := match(p, q); got != want {
			t.Errorf("match(%q) = %v, want %v", q, got, want)
		}
	}
}

func TestStepStaysInBounds(t *testing.T) {
	rng := rand.New(rand.NewSource(1)) // #nosec G404 -- fixed seed for a repeatable test
	ps := sample()
	for i := 0; i < 500; i++ {
		var cpu, mem float64
		ps, cpu, mem = step(ps, rng)
		if cpu < 0 || cpu > 100 || mem < 0 || mem > 100 {
			t.Fatalf("totals out of range: %v %v", cpu, mem)
		}
		for _, p := range ps {
			if p.cpu < 0 || p.cpu > 99 || p.mem < 0.05 || p.mem > 40 {
				t.Fatalf("process out of range: %+v", p)
			}
		}
	}
	if len(ps) != len(sample()) {
		t.Fatal("step changed the number of processes")
	}
}
