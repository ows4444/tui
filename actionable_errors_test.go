package tui

import (
	"errors"
	"strings"
	"testing"

	"github.com/ows4444/tui/internal/termio"
)

func TestRunNonTerminalInputNamesWithInput(t *testing.T) {
	p := NewProgram(keyLogger{}, WithTerminal(&termio.Fake{Interactive: false}))
	_, err := p.Run()
	if err == nil {
		t.Fatal("Run succeeded on non-terminal input")
	}
	for _, want := range []string{"not a terminal", "WithInput"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q must contain %q", err, want)
		}
	}
}

func TestRunRawModeErrorNamesCauseAndFix(t *testing.T) {
	boom := errors.New("ioctl failed")
	p := NewProgram(keyLogger{}, WithTerminal(&termio.Fake{Interactive: true, RawErr: boom}))
	_, err := p.Run()
	if !errors.Is(err, boom) {
		t.Fatalf("Run = %v, want wrapped cause", err)
	}
	for _, want := range []string{"raw mode", "WithInput"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q must contain %q", err, want)
		}
	}
}

func TestRunReusedAndInterruptedErrorsNameTheFix(t *testing.T) {
	if !strings.Contains(ErrProgramReused.Error(), "NewProgram") {
		t.Errorf("ErrProgramReused %q must name NewProgram", ErrProgramReused)
	}
	if !strings.Contains(ErrInterrupted.Error(), "signal") {
		t.Errorf("ErrInterrupted %q must name the signal cause", ErrInterrupted)
	}
}
