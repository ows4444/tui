package main

import "testing"

func run(b build) (build, int) {
	for n := 0; !b.over && n < 1000; n++ {
		b, _ = b.advance()
	}
	return b, b.tick
}

func names(b build, s status) []string {
	var out []string
	for i, j := range b.jobs {
		if j.status == s {
			out = append(out, b.steps[i].name)
		}
	}
	return out
}

func TestOnlyStepsWithNoNeedsStartFirst(t *testing.T) {
	b := newBuild(3, "")
	if got := names(b, running); len(got) != 1 || got[0] != "fetch modules" {
		t.Fatalf("running %v", got)
	}
	if b.count(waiting) != 7 {
		t.Fatalf("waiting %d", b.count(waiting))
	}
}

func TestStepsStartTheTickTheirNeedsFinish(t *testing.T) {
	b := newBuild(3, "")
	var ended []int
	for range 7 {
		b, _ = b.advance()
	}
	if b.count(running) != 1 {
		t.Fatalf("running %d at tick 7", b.count(running))
	}
	b, ended = b.advance()
	if len(ended) != 1 || b.steps[ended[0]].name != "fetch modules" {
		t.Fatalf("ended %v", ended)
	}
	if got := names(b, running); len(got) != 3 {
		t.Fatalf("running %v after fetch", got)
	}
	if b.jobs[b.index("compile ansi")].started != 8 {
		t.Fatalf("started at %d", b.jobs[b.index("compile ansi")].started)
	}
}

func TestJobsLimitHoldsStepsBack(t *testing.T) {
	b := newBuild(1, "")
	most := 0
	for !b.over {
		b, _ = b.advance()
		most = max(most, b.count(running))
	}
	if most != 1 {
		t.Fatalf("%d ran at once with a limit of 1", most)
	}
	wide, tw := run(newBuild(5, ""))
	narrow, tn := run(newBuild(1, ""))
	if !wide.ok() || !narrow.ok() || tw >= tn {
		t.Fatalf("wide %d ticks, narrow %d", tw, tn)
	}
}

func TestEveryStepWaitsForAllItsNeeds(t *testing.T) {
	b := newBuild(8, "")
	for !b.over {
		b, _ = b.advance()
		for i, s := range b.steps {
			if b.jobs[i].status == waiting {
				continue
			}
			for _, d := range s.deps {
				if dj := b.jobs[b.index(d)]; dj.status != done {
					t.Fatalf("tick %d: %s has status %v but needs %s (%v)", b.tick, s.name, b.jobs[i].status, d, dj.status)
				}
			}
		}
	}
}

func TestFullRunIsDeterministicAndPrintsAllItsOutput(t *testing.T) {
	a, ta := run(newBuild(3, ""))
	_, tb := run(newBuild(3, ""))
	if !a.ok() || ta != tb || ta < 50 {
		t.Fatalf("ticks %d %d ok %v", ta, tb, a.ok())
	}
	for i, j := range a.jobs {
		if len(j.log) != len(a.steps[i].out) {
			t.Errorf("%s printed %d of %d lines", a.steps[i].name, len(j.log), len(a.steps[i].out))
		}
		if j.ended-j.started != a.steps[i].dur {
			t.Errorf("%s took %d ticks, want %d", a.steps[i].name, j.ended-j.started, a.steps[i].dur)
		}
	}
	if again, _ := a.advance(); again.tick != a.tick {
		t.Fatal("a finished build moved")
	}
}

func TestAFailureSkipsEverythingThatNeedsIt(t *testing.T) {
	b, _ := run(newBuild(3, "compile layout"))
	if b.ok() || b.count(failed) != 1 || names(b, failed)[0] != "compile layout" {
		t.Fatalf("failed %v", names(b, failed))
	}
	skipped := names(b, skipped)
	if len(skipped) != 4 {
		t.Fatalf("skipped %v", skipped)
	}
	if j := b.jobs[b.index("link tui")]; j.reason == "" {
		t.Fatal("no reason given for the skip")
	}
	last := b.jobs[b.index("compile layout")].log
	if len(last) == 0 || last[len(last)-1] != b.steps[b.index("compile layout")].bad {
		t.Fatalf("failure output %v", last)
	}
	if b.jobs[b.index("compile widgets")].status != done {
		t.Fatal("an unrelated step should still finish")
	}
}

func TestCancelStopsRunningAndWaitingSteps(t *testing.T) {
	b := newBuild(3, "")
	for range 12 {
		b, _ = b.advance()
	}
	b = b.cancel()
	if !b.over || b.count(cancelled) != 7 || b.count(done) != 1 || b.ok() {
		t.Fatalf("cancelled %d done %d", b.count(cancelled), b.count(done))
	}
}

func TestSecondsFormat(t *testing.T) {
	for in, want := range map[int]string{0: "0.0s", 7: "0.7s", 10: "1.0s", 123: "12.3s"} {
		if got := seconds(in); got != want {
			t.Errorf("seconds(%d) = %q, want %q", in, got, want)
		}
	}
}
