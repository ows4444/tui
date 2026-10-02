package ansi

import (
	"strings"
	"testing"
)

const kittySeq = "\x1b_Ga=T,f=100;AAAA\x1b\\"

func TestPassthroughTmuxWrapsAndDoublesEscapes(t *testing.T) {
	got := Passthrough(kittySeq, env(map[string]string{"TMUX": "/tmp/tmux-1000/default,1,0"}))
	want := "\x1bPtmux;\x1b\x1b_Ga=T,f=100;AAAA\x1b\x1b\\\x1b\\"
	if got != want {
		t.Errorf("tmux: got %q, want %q", got, want)
	}
	// Every ESC of the payload is doubled: the wrapper adds exactly one ESC
	// in the opener and one in the closer.
	if n, m := strings.Count(got, "\x1b"), 2*strings.Count(kittySeq, "\x1b")+2; n != m {
		t.Errorf("ESC count = %d, want %d", n, m)
	}
}

func TestPassthroughScreenWrapsInDCS(t *testing.T) {
	got := Passthrough(kittySeq, env(map[string]string{"STY": "1234.pts-0.host"}))
	if want := "\x1bP" + kittySeq + "\x1b\\"; got != want {
		t.Errorf("screen: got %q, want %q", got, want)
	}
}

func TestPassthroughScreenSplitsLongPayloads(t *testing.T) {
	seq := strings.Repeat("x", 2*screenChunk+5)
	got := Passthrough(seq, env(map[string]string{"STY": "1"}))
	if n := strings.Count(got, "\x1bP"); n != 3 {
		t.Errorf("pieces = %d, want 3", n)
	}
	if strings.ReplaceAll(strings.ReplaceAll(got, "\x1bP", ""), "\x1b\\", "") != seq {
		t.Error("splitting changed the payload")
	}
}

func TestPassthroughLeavesPlainTerminalsAlone(t *testing.T) {
	for _, vars := range []map[string]string{{}, {"TMUX": "", "STY": ""}, {"TERM": "xterm-256color"}} {
		if got := Passthrough(kittySeq, env(vars)); got != kittySeq {
			t.Errorf("%v: got %q, want it unchanged", vars, got)
		}
	}
	if got := Passthrough("", env(map[string]string{"TMUX": "x"})); got != "" {
		t.Errorf("empty sequence wrapped: %q", got)
	}
}

func TestPassthroughTmuxBeatsScreen(t *testing.T) {
	got := Passthrough("a", env(map[string]string{"TMUX": "x", "STY": "y"}))
	if !strings.HasPrefix(got, "\x1bPtmux;") {
		t.Errorf("got %q, want the tmux wrapper", got)
	}
}

func TestPassthroughNilGetenvReadsTheProcessEnvironment(t *testing.T) {
	t.Setenv("TMUX", "x")
	if got := Passthrough("a", nil); !strings.HasPrefix(got, "\x1bPtmux;") {
		t.Errorf("got %q, want the tmux wrapper", got)
	}
}
