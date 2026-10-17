package main

import (
	"math"
	"math/rand"
	"sort"
)

const (
	history = 120 // samples kept per series
	cores   = 8
)

// sample is the machine at one moment. The numbers are made up: the dashboard
// reads nothing from the system, so it is safe to run anywhere.
type sample struct {
	cpu    float64 // percent
	core   [cores]float64
	memGB  float64 // used
	cached float64
	swapGB float64
	rx, tx float64 // bytes a second
}

const (
	memTotalGB  = 16.0
	swapTotalGB = 4.0
)

type proc struct {
	pid  int
	name string
	cpu  float64
	memP float64
}

type volume struct {
	mount   string
	usedGB  float64
	totalGB float64
}

// world holds the simulated machine and its recent past.
type world struct {
	rng   *rand.Rand
	cur   sample
	hist  []sample
	procs []proc
	disks []volume
	rxSum float64 // bytes received so far
	txSum float64
	load  [3]float64
}

func newWorld() *world {
	w := &world{
		rng: rand.New(rand.NewSource(42)), // #nosec G404 -- made-up numbers, not security-sensitive
		cur: sample{cpu: 35, memGB: 6.1, cached: 3.0, swapGB: 0.2, rx: 4e5, tx: 8e4},
		procs: []proc{
			{1, "launchd", 0.1, 0.3}, {88, "kernel_task", 9, 4.1}, {142, "WindowServer", 6, 2.6},
			{420, "postgres", 4, 5.2}, {518, "node", 12, 7.9}, {733, "Safari", 5, 9.4},
			{801, "Slack", 3, 5.1}, {907, "go", 8, 2.2}, {1010, "docker", 2, 6.3}, {1203, "Terminal", 1, 0.9},
		},
		disks: []volume{{"/", 310, 500}, {"/home", 405, 500}, {"/var", 34, 100}, {"/data", 1710, 2000}},
	}
	for i := 0; i < history; i++ { // a past, so the charts start full
		w.step()
	}
	return w
}

func clamp(x, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, x)) }

// step moves the machine on one tick.
func (w *world) step() {
	r := w.rng
	s := w.cur
	s.cpu = clamp(s.cpu+(40-s.cpu)*0.1+r.NormFloat64()*7, 1, 99)
	if r.Intn(25) == 0 {
		s.cpu = clamp(s.cpu+30+r.Float64()*20, 1, 99) // a burst
	}
	for i := range s.core {
		s.core[i] = clamp(s.cpu+r.NormFloat64()*14, 0, 100)
	}
	s.memGB = clamp(s.memGB+r.NormFloat64()*0.08, 3, memTotalGB-0.5)
	s.cached = clamp(s.cached+r.NormFloat64()*0.05, 1, memTotalGB-s.memGB)
	s.swapGB = clamp(s.swapGB+(s.memGB-12)*0.01+r.NormFloat64()*0.01, 0, swapTotalGB)
	s.rx = clamp(s.rx*math.Exp(r.NormFloat64()*0.45), 2e3, 5e7)
	s.tx = clamp(s.tx*math.Exp(r.NormFloat64()*0.45), 1e3, 2e7)
	w.cur = s
	w.hist = append(w.hist, s)
	if len(w.hist) > history {
		w.hist = w.hist[len(w.hist)-history:]
	}
	w.rxSum += s.rx
	w.txSum += s.tx
	for i := range w.procs {
		p := &w.procs[i]
		p.cpu = clamp(p.cpu*math.Exp(r.NormFloat64()*0.35), 0, 80)
		p.memP = clamp(p.memP+r.NormFloat64()*0.05, 0.1, 25)
	}
	for i := range w.disks {
		w.disks[i].usedGB = clamp(w.disks[i].usedGB+r.Float64()*0.05, 0, w.disks[i].totalGB)
	}
	for i, k := range []float64{0.05, 0.015, 0.0055} { // 1, 5 and 15 minute averages
		w.load[i] += (s.cpu/100*cores - w.load[i]) * k * 4
	}
}

// series picks one number out of every sample in the history.
func (w *world) series(f func(sample) float64) []float64 {
	out := make([]float64, len(w.hist))
	for i, s := range w.hist {
		out[i] = f(s)
	}
	return out
}

// top lists the n busiest processes, busiest first.
func (w *world) top(n int) []proc {
	ps := append([]proc(nil), w.procs...)
	sort.SliceStable(ps, func(i, j int) bool { return ps[i].cpu > ps[j].cpu })
	if n < len(ps) {
		ps = ps[:n]
	}
	return ps
}
