package tui

import (
	"errors"
	"strings"
	"testing"
)

// #19: a Program runs once; the second Run reports it and writes nothing.
func TestSecondRunReturnsErrProgramReusedAndWritesNothing(t *testing.T) {
	out := &strings.Builder{}
	p := NewProgram(keyLogger{}, WithInput(strings.NewReader("q")), WithOutput(out))
	if _, err := p.Run(); err != nil {
		t.Fatalf("first Run: %v", err)
	}
	written := out.Len()
	if written == 0 {
		t.Fatal("first Run wrote nothing; the test would prove nothing")
	}

	m, err := p.Run()
	if !errors.Is(err, ErrProgramReused) {
		t.Fatalf("second Run error = %v, want ErrProgramReused", err)
	}
	if m == nil {
		t.Error("second Run returned a nil model")
	}
	if out.Len() != written {
		t.Fatalf("second Run wrote %d bytes, want 0", out.Len()-written)
	}
}

// Even a Run that failed to start uses the Program up, so a retry cannot
// half-initialise a second event loop over the first one's state.
func TestRunAfterFailedRunIsReused(t *testing.T) {
	p := NewProgram(keyLogger{}, WithOutput(&strings.Builder{})) // stdin is not a terminal under go test
	if _, err := p.Run(); err == nil {
		t.Skip("stdin is a terminal here")
	}
	if _, err := p.Run(); !errors.Is(err, ErrProgramReused) {
		t.Fatalf("second Run error = %v, want ErrProgramReused", err)
	}
}
