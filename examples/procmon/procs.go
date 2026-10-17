package main

import (
	"math/rand"
	"sort"
	"strconv"
	"strings"
)

type proc struct {
	pid   int
	name  string
	user  string
	state string // R running, S sleeping, I idle
	cpu   float64
	mem   float64
}

// column indexes, also the order of the header cells.
const (
	colPID = iota
	colName
	colUser
	colCPU
	colMem
	colState
	numCols
)

var colNames = [numCols]string{"PID", "NAME", "USER", "CPU%", "MEM%", "STATE"}

// sample is the process table the program starts with and the tests use.
func sample() []proc {
	return []proc{
		{1, "launchd", "root", "S", 0.1, 0.3},
		{88, "kernel_task", "root", "R", 12.4, 4.1},
		{142, "WindowServer", "_window", "R", 9.8, 2.6},
		{311, "sshd", "root", "S", 0.0, 0.1},
		{420, "postgres", "postgres", "S", 2.3, 5.2},
		{421, "postgres: writer", "postgres", "I", 0.2, 1.1},
		{507, "redis-server", "redis", "S", 1.1, 0.8},
		{612, "docker", "mdawais", "S", 3.7, 6.4},
		{688, "node", "mdawais", "R", 24.5, 3.9},
		{702, "Code Helper", "mdawais", "R", 17.2, 7.7},
		{703, "Code Helper (GPU)", "mdawais", "S", 4.8, 2.2},
		{745, "Safari", "mdawais", "R", 14.1, 9.3},
		{790, "Slack", "mdawais", "S", 3.2, 4.4},
		{811, "Terminal", "mdawais", "S", 0.9, 0.6},
		{812, "zsh", "mdawais", "S", 0.0, 0.1},
		{930, "go", "mdawais", "R", 31.6, 2.1},
		{931, "compile", "mdawais", "R", 42.0, 3.3},
		{1002, "mds_stores", "root", "I", 0.6, 1.4},
		{1044, "Spotlight", "mdawais", "I", 0.4, 0.9},
		{1201, "cupsd", "root", "I", 0.0, 0.2},
		{1305, "bluetoothd", "root", "S", 0.1, 0.2},
		{1410, "Music", "mdawais", "S", 1.5, 2.8},
		{1522, "ssh-agent", "mdawais", "S", 0.0, 0.1},
		{1630, "htop", "mdawais", "R", 0.8, 0.1},
	}
}

// sortProcs orders ps by column col; ties fall back to PID so the order is
// stable between ticks.
func sortProcs(ps []proc, col int, desc bool) {
	less := func(a, b proc) bool {
		switch col {
		case colName:
			return strings.ToLower(a.name) < strings.ToLower(b.name)
		case colUser:
			return a.user < b.user
		case colCPU:
			return a.cpu < b.cpu
		case colMem:
			return a.mem < b.mem
		case colState:
			return a.state < b.state
		}
		return a.pid < b.pid
	}
	sort.SliceStable(ps, func(i, j int) bool {
		a, b := ps[i], ps[j]
		if desc {
			a, b = b, a
		}
		if less(a, b) != less(b, a) {
			return less(a, b)
		}
		return ps[i].pid < ps[j].pid
	})
}

// match reports whether p matches a filter: every word must be in the name or
// user, ignoring case, or equal the PID.
func match(p proc, query string) bool {
	hay := strings.ToLower(p.name + " " + p.user)
	for _, w := range strings.Fields(strings.ToLower(query)) {
		if !strings.Contains(hay, w) && w != strconv.Itoa(p.pid) {
			return false
		}
	}
	return true
}

// step moves every process's numbers a little: a bounded random walk, so the
// table looks alive. It returns the new slice and the machine totals.
func step(ps []proc, rng *rand.Rand) ([]proc, float64, float64) {
	out := make([]proc, len(ps))
	var cpu, mem float64
	for i, p := range ps {
		p.cpu = clamp(p.cpu+(rng.Float64()-0.5)*p.cpu*0.6+(rng.Float64()-0.5)*0.4, 0, 99)
		p.mem = clamp(p.mem+(rng.Float64()-0.5)*0.2, 0.05, 40)
		out[i] = p
		cpu += p.cpu
		mem += p.mem
	}
	return out, clamp(cpu/4, 0, 100), clamp(mem, 0, 100)
}

func clamp(v, lo, hi float64) float64 { return max(lo, min(hi, v)) }

// totals are the machine-wide CPU and memory percentages for ps.
func totals(ps []proc) (cpu, mem float64) {
	for _, p := range ps {
		cpu += p.cpu
		mem += p.mem
	}
	return clamp(cpu/4, 0, 100), clamp(mem, 0, 100)
}
