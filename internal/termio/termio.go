// Package termio holds the implementations of tui.Terminal, the runtime's port
// to the terminal (size query, raw mode, output virtual-terminal processing):
// OSTerminal for a real terminal and Fake for tests. The interface lives in the
// root package; a compile-time assertion there keeps both in step. Bytes still
// flow through the io.Reader and io.Writer the Program is given; the port is
// the control plane only.
package termio

import (
	"io"
	"os"

	"github.com/ows4444/tui/term"
)

// OSTerminal is the OS-backed implementation of tui.Terminal for an input file
// and an output writer. The interface itself is defined once, in the root
// package; this package sits below it and only supplies implementations.
type OSTerminal struct {
	in  *os.File
	out io.Writer
}

// OS returns the terminal for in and out. Anything that is not an *os.File has
// no terminal behind it: it reports no size and enables nothing.
func OS(in *os.File, out io.Writer) OSTerminal { return OSTerminal{in: in, out: out} }

func (t OSTerminal) outFile() *os.File {
	f, _ := t.out.(*os.File)
	return f
}

func (t OSTerminal) IsTerminal() bool {
	return t.in != nil && term.IsTerminal(int(t.in.Fd()))
}

func (t OSTerminal) Size() (w, h int, ok bool) {
	f := t.outFile()
	if f == nil {
		return 0, 0, false
	}
	w, h, err := term.GetSize(int(f.Fd()))
	return w, h, err == nil
}

func (t OSTerminal) MakeRaw(mouse bool) (func() error, error) {
	fd := int(t.in.Fd())
	state, err := term.MakeRawMouse(fd, mouse)
	if err != nil {
		return nil, err
	}
	return func() error { return term.Restore(fd, state) }, nil
}

func (t OSTerminal) EnableOutputVT() (func() error, error) {
	f := t.outFile()
	if f == nil {
		return nil, nil
	}
	return term.EnableOutputVT(int(f.Fd()))
}
