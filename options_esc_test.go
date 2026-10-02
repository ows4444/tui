package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui/input"
)

func TestWithEscTimeout(t *testing.T) {
	p := NewProgram(inspModel{keys: new([]string)}, WithOutput(&strings.Builder{}))
	if got := p.newReaderSet(nil).rd.EscTimeout(); got != input.DefaultEscTimeout {
		t.Fatalf("default = %v, want %v", got, input.DefaultEscTimeout)
	}
	for _, d := range []time.Duration{0, -time.Second} {
		p = NewProgram(inspModel{keys: new([]string)}, WithEscTimeout(d), WithOutput(&strings.Builder{}))
		if got := p.newReaderSet(nil).rd.EscTimeout(); got != input.DefaultEscTimeout {
			t.Fatalf("d=%v: got %v, want default", d, got)
		}
	}
	p = NewProgram(inspModel{keys: new([]string)}, WithEscTimeout(120*time.Millisecond), WithOutput(&strings.Builder{}))
	if got := p.newReaderSet(nil).rd.EscTimeout(); got != 120*time.Millisecond {
		t.Fatalf("got %v, want 120ms", got)
	}
}

func TestWithResizePoll(t *testing.T) {
	p := NewProgram(inspModel{keys: new([]string)}, WithOutput(&strings.Builder{}))
	if got := p.resizePollEvery(); got != 250*time.Millisecond {
		t.Fatalf("default = %v", got)
	}
	p = NewProgram(inspModel{keys: new([]string)}, WithResizePoll(-1), WithOutput(&strings.Builder{}))
	if got := p.resizePollEvery(); got != 250*time.Millisecond {
		t.Fatalf("d<=0 = %v", got)
	}
	p = NewProgram(inspModel{keys: new([]string)}, WithResizePoll(40*time.Millisecond), WithOutput(&strings.Builder{}))
	if got := p.resizePollEvery(); got != 40*time.Millisecond {
		t.Fatalf("got %v", got)
	}
}
