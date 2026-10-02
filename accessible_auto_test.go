package tui

import "testing"

func TestAccessibleAutoEnv(t *testing.T) {
	cases := []struct {
		acc, term string
		want      bool
	}{{"1", "xterm", true}, {"", "dumb", true}, {"", "xterm", false}, {"0", "xterm", false}}
	for _, c := range cases {
		t.Setenv("ACCESSIBLE", c.acc)
		t.Setenv("TERM", c.term)
		p := NewProgram(staticModel{}, WithAccessibleAuto())
		if p.Accessible() != c.want {
			t.Errorf("ACCESSIBLE=%q TERM=%q: Accessible=%v want %v", c.acc, c.term, p.Accessible(), c.want)
		}
		if c.want && (p.altScreen || !p.ReducedMotion()) {
			t.Error("auto-enabled accessible must force alt screen off and reduced motion on")
		}
	}
}

func TestAccessibleSuffixDiff(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(staticModel{view: "a\nhello"}, WithOutput(out), WithAccessible(true))
	p.render()
	before := len(read())
	p.model = staticModel{view: "a\nhello world"}
	p.render()
	if got := string(read()[before:]); got != " world\r\n" {
		t.Errorf("added = %q, want only suffix", got)
	}
}

func TestAnnounceOwnLine(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(staticModel{view: "a"}, WithOutput(out), WithAccessible(true))
	p.render()
	before := len(read())
	msg := Announce("saved")()
	if !p.handleAnnounce(msg) {
		t.Fatal("announceMsg not handled")
	}
	if got := string(read()[before:]); got != "saved\r\n" {
		t.Errorf("got %q", got)
	}
	q := NewProgram(staticModel{view: "a"}, WithOutput(out))
	b := len(read())
	q.handleAnnounce(msg)
	if len(read()) != b {
		t.Error("announce wrote outside accessible mode")
	}
}
