package tui

import (
	"strings"
	"sync"
	"testing"
)

// #20: EnableMouse / EnterAltScreen change mode state on the loop goroutine
// while a SIGTERM restore (restoreFixed) reads it from another. Run with
// -race: the restore path must snapshot that state under outMu.
func TestModeChangesRacingSignalRestoreAreRaceFree(t *testing.T) {
	for i := 0; i < 50; i++ {
		out := &strings.Builder{}
		p := NewProgram(staticModel{view: "v"}, WithOutput(out), WithAltScreen(false))
		mouse := EnableMouse(MouseAllMotion)().(modeMsg)
		enter := EnterAltScreen()().(modeMsg)
		exit := ExitAltScreen()().(modeMsg)
		off := EnableMouse(MouseOff)().(modeMsg)

		var wg sync.WaitGroup
		wg.Add(2)
		go func() { // the loop goroutine applying queued mode changes
			defer wg.Done()
			for _, m := range []modeMsg{mouse, enter, exit, off, mouse, enter} {
				m.apply(p)
			}
		}()
		go func() { // the signal watcher's timeout fallback
			defer wg.Done()
			p.restoreFixed()
		}()
		wg.Wait()
	}
}

// Whatever mode state the restore saw, nothing is written after it.
func TestNoBytesAfterFixedRestoreEvenWhenModesChange(t *testing.T) {
	out := &strings.Builder{}
	p := NewProgram(staticModel{view: "v"}, WithOutput(out))
	p.restoreFixed()
	before := out.Len()
	EnableMouse(MouseCellMotion)().(modeMsg).apply(p)
	ExitAltScreen()().(modeMsg).apply(p)
	if out.Len() != before {
		t.Fatalf("wrote %d bytes after the restore", out.Len()-before)
	}
}
