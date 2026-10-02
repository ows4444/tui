// Package cancelreader is an io.Reader over an *os.File whose blocked Read
// can be cancelled from another goroutine.
//
// Program needs this to stop its input reader when Run returns. The obvious
// tool, File.SetReadDeadline, does not work for a terminal: os.Stdin is a
// blocking descriptor Go never registers with its poller (so there is no
// deadline support at all), and calling File.Fd() — which Run does — switches
// even a pollable file to blocking mode, after which SetReadDeadline returns
// nil yet no longer interrupts a Read. The stranded reader then outlives Run
// and steals keys from whatever reads the terminal next.
//
// On darwin and linux the Reader waits with select(2) on the file and on an
// internal cancel pipe, and only reads once the file is readable, so Cancel
// never leaves a Read half-done and never consumes input. On windows it polls
// the handle (PeekNamedPipe for pipes, WaitForSingleObject for a console) and
// reads only once input is there, so a Cancel lands within a short slice. On
// other platforms it falls back to the read-deadline behaviour Program used
// before, which is best effort and unverified there.
//
// The file must stay open until the Reader is closed: the Reader waits on
// the raw descriptor, and closing a descriptor under a waiting select is
// undefined (and its number can be reused).
package cancelreader

import "errors"

// ErrCanceled is returned by Read after Cancel.
var ErrCanceled = errors.New("cancelreader: read canceled")
