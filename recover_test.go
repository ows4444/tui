//go:build darwin || linux

package tui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
)

// recoverModel panics in the stage named by where.
type recoverModel struct{ where string }

func (m recoverModel) Init() Cmd {
	switch m.where {
	case "init":
		panic("boom-init")
	case "cmd":
		return func() Msg { panic("boom-cmd") }
	}
	return nil
}

func (m recoverModel) Update(msg Msg) (Model, Cmd) {
	if m.where == "update" {
		if _, ok := msg.(ResizeMsg); ok {
			panic("boom-update")
		}
	}
	return m, nil
}

func (m recoverModel) View() string {
	if m.where == "view" {
		panic("boom-view")
	}
	return "x"
}

func runRecoverModel(t *testing.T, where string, opts ...ProgramOption) (string, error) {
	t.Helper()
	master, slave := openPTY(t, 80, 24)
	defer master.Close()
	defer slave.Close()
	out, err := os.Create(filepath.Join(t.TempDir(), "out"))
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	opts = append([]ProgramOption{WithInput(slave), WithOutput(out), WithAltScreen(true)}, opts...)
	p := NewProgram(recoverModel{where: where}, opts...)
	_, err = p.Run()
	b, _ := os.ReadFile(out.Name())
	return string(b), err
}

func TestWithRecoverReturnsPanicError(t *testing.T) {
	for _, where := range []string{"init", "update", "view", "cmd"} {
		t.Run(where, func(t *testing.T) {
			out, err := runRecoverModel(t, where, WithRecover(true))
			var pe *PanicError
			if !errors.As(err, &pe) {
				t.Fatalf("err = %v, want *PanicError", err)
			}
			if pe.Value != "boom-"+where {
				t.Errorf("Value = %v", pe.Value)
			}
			if len(pe.Stack) == 0 || !strings.Contains(pe.Error(), "boom-"+where) {
				t.Errorf("Stack empty or Error() = %q", pe.Error())
			}
			if !strings.Contains(out, ansi.AltScreenDisable) || !strings.Contains(out, ansi.CursorShow) {
				t.Errorf("terminal not restored: %q", out)
			}
		})
	}
}

func TestWithoutRecoverRepanicsAfterRestore(t *testing.T) {
	var got any
	func() {
		defer func() { got = recover() }()
		_, _ = runRecoverModel(t, "update")
	}()
	if got != "boom-update" {
		t.Fatalf("recovered %v, want the original panic value", got)
	}
}
