package main

import "strings"

type status int

const (
	waiting status = iota
	running
	done
	failed
	skipped   // a step it needs failed
	cancelled // stopped by the user while running or waiting
)

// step is one job of the pipeline. dur is how many ticks it takes; out is the
// text it prints, spread over its run.
type step struct {
	name string
	deps []string
	dur  int
	out  []string
	bad  string // the line an injected failure prints
}

var pipeline = []step{
	{"fetch modules", nil, 8, []string{"go: downloading github.com/ows4444/ansi v0.2.0", "go: downloading golang.org/x/sys v0.25.0", "go: verifying go.sum"}, "go: github.com/ows4444/ansi: reading go.sum: checksum mismatch"},
	{"compile ansi", []string{"fetch modules"}, 10, []string{"compile internal/ansi", "compile ansi/color.go", "compile ansi/style.go"}, "ansi/style.go:41:9: undefined: Underline"},
	{"compile layout", []string{"fetch modules"}, 14, []string{"compile layout/box.go", "compile layout/flexnode.go", "compile layout/gridnode.go", "compile layout/wrap.go"}, "layout/box.go:88:2: declared and not used: pad"},
	{"compile widgets", []string{"fetch modules"}, 20, []string{"compile widgets/button.go", "compile widgets/table.go", "compile widgets/footer.go", "compile widgets/list.go"}, "widgets/table.go:130:14: cannot use w (type int) as string"},
	{"link tui", []string{"compile ansi", "compile layout", "compile widgets"}, 8, []string{"link tui.a", "link cmd/tui"}, "link: undefined reference to layout.Wrap"},
	{"vet", []string{"compile ansi", "compile layout", "compile widgets"}, 12, []string{"vet ./ansi", "vet ./layout", "vet ./widgets"}, "layout/wrap.go:22:3: unreachable code"},
	{"test", []string{"link tui"}, 24, []string{"=== RUN   TestBox", "=== RUN   TestFlex", "=== RUN   TestGrid", "=== RUN   TestWrap", "--- PASS: all (24 tests)"}, "--- FAIL: TestWrap (0.00s)\n    wrap_test.go:41: got 3 rows, want 2"},
	{"package", []string{"link tui", "vet", "test"}, 6, []string{"tar tui-v0.2.0.tar.gz", "sha256 written"}, "package: no space left on device"},
}

// job is the run state of one step.
type job struct {
	status   status
	progress int      // ticks spent
	started  int      // tick it started on
	ended    int      // tick it finished on
	log      []string // what it has printed so far
	reason   string   // why it was skipped
}

// build is the pipeline in motion. It has no clock of its own: step moves it one
// tick on, so the same inputs always give the same run.
type build struct {
	steps []step
	jobs  []job
	max   int    // steps allowed to run at once
	fail  string // step that fails at the end of its run, or ""
	tick  int
	over  bool
}

func newBuild(maxJobs int, fail string) build {
	b := build{steps: pipeline, jobs: make([]job, len(pipeline)), max: max(1, maxJobs), fail: fail}
	return b.startReady()
}

func (b build) index(name string) int {
	for i, s := range b.steps {
		if s.name == name {
			return i
		}
	}
	return -1
}

func (b build) clone() build {
	b.jobs = append([]job(nil), b.jobs...)
	return b
}

func (b build) count(s status) int {
	n := 0
	for _, j := range b.jobs {
		if j.status == s {
			n++
		}
	}
	return n
}

// startReady starts waiting steps whose needs are done, up to the limit, and
// skips those whose needs failed.
func (b build) startReady() build {
	b = b.clone()
	for i, s := range b.steps {
		if b.jobs[i].status != waiting {
			continue
		}
		ready := true
		for _, d := range s.deps {
			switch dj := b.jobs[b.index(d)]; dj.status {
			case failed, skipped, cancelled:
				b.jobs[i].status, b.jobs[i].reason, b.jobs[i].ended = skipped, d, b.tick
				ready = false
			case done:
			default:
				ready = false
			}
		}
		if ready && b.jobs[i].status == waiting && b.count(running) < b.max {
			b.jobs[i].status, b.jobs[i].started = running, b.tick
		}
	}
	return b
}

// advance runs one tick and returns the build with the names of the steps that
// finished on it.
func (b build) advance() (build, []int) {
	if b.over {
		return b, nil
	}
	b = b.clone()
	b.tick++
	var ended []int
	for i := range b.jobs {
		j := &b.jobs[i]
		if j.status != running {
			continue
		}
		s := b.steps[i]
		j.progress++
		// a line of output at each equal share of the run
		if n := len(s.out); n > 0 && len(j.log) < n && j.progress*n >= (len(j.log)+1)*s.dur {
			j.log = append(append([]string(nil), j.log...), s.out[len(j.log)])
		}
		if j.progress >= s.dur {
			j.ended = b.tick
			j.status = done
			if s.name == b.fail {
				j.status = failed
				j.log = append(append([]string(nil), j.log...), strings.Split(s.bad, "\n")...)
			}
			ended = append(ended, i)
		}
	}
	b = b.startReady()
	b.over = b.count(running)+b.count(waiting) == 0
	return b, ended
}

// cancel stops everything that has not finished.
func (b build) cancel() build {
	b = b.clone()
	for i := range b.jobs {
		if s := b.jobs[i].status; s == running || s == waiting {
			b.jobs[i].status, b.jobs[i].ended = cancelled, b.tick
		}
	}
	b.over = true
	return b
}

func (b build) ok() bool { return b.count(done) == len(b.steps) }

// seconds shows ticks as seconds with one decimal; a tick is a tenth.
func seconds(ticks int) string { return itoa(ticks/10) + "." + itoa(ticks%10) + "s" }
