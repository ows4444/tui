//go:build windows

package cancelreader

import (
	"io"
	"os"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

// Windows has no select(2) for pipes or the console, and a blocking
// ReadFile cannot be interrupted (os.File.SetReadDeadline is not supported
// on these handles). So Read never blocks in the kernel: it waits for the
// handle to have input, in short slices, checking for Cancel between them,
// and only calls ReadFile once input is there. A cancel therefore stops a
// waiting Read within one slice and never consumes input.
const (
	pipePollWait = 10 * time.Millisecond // sleep between PeekNamedPipe checks

	waitObject0 = 0x00000000
	waitTimeout = 0x00000102

	maxPeek = 64 // input records looked at in one PeekConsoleInputW
)

var (
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleMode   = kernel32.NewProc("GetConsoleMode")
	procPeekNamedPipe    = kernel32.NewProc("PeekNamedPipe")
	procWaitForSingleObj = kernel32.NewProc("WaitForSingleObject")
	procPeekConsoleInput = kernel32.NewProc("PeekConsoleInputW")
	procReadConsoleInput = kernel32.NewProc("ReadConsoleInputW")
)

// winConsole is the console interface over a real console input handle.
type winConsole struct{ h uintptr }

func (c winConsole) wait(d time.Duration) (bool, error) {
	ret, _, _ := procWaitForSingleObj.Call(c.h, uintptr(d/time.Millisecond))
	switch ret {
	case waitTimeout:
		return false, nil
	default: // signalled, or an error the following peek/ReadFile will report
		return true, nil
	}
}

func (c winConsole) peek() ([]consoleRecord, error) {
	var buf [maxPeek]inputRecord
	var n uint32
	ret, _, err := procPeekConsoleInput.Call(c.h, uintptr(unsafe.Pointer(&buf[0])), maxPeek, uintptr(unsafe.Pointer(&n))) // #nosec G103 -- Win32 out-parameters require unsafe.Pointer
	if ret == 0 {
		return nil, err
	}
	return decodeRecords(buf[:], int(n)), nil
}

func (c winConsole) discard(n int) error {
	var buf [maxPeek]inputRecord
	for n > 0 {
		want := min(n, maxPeek)
		var got uint32
		ret, _, err := procReadConsoleInput.Call(c.h, uintptr(unsafe.Pointer(&buf[0])), uintptr(want), uintptr(unsafe.Pointer(&got))) // #nosec G103 -- Win32 out-parameters require unsafe.Pointer
		if ret == 0 {
			return err
		}
		if got == 0 {
			return nil
		}
		n -= int(got)
	}
	return nil
}

// Reader reads from a file and can be cancelled. Read is meant to be called
// from one goroutine; Cancel may be called from any goroutine, any number of
// times.
type Reader struct {
	f        *os.File
	h        uintptr
	console  bool
	con      console
	onResize atomic.Pointer[func()]
	canceled atomic.Bool
	cancelCh chan struct{}
	once     sync.Once
}

// New returns a Reader for f. It does not take ownership of f, change its
// mode, or close it.
func New(f *os.File) (*Reader, error) {
	rc, err := f.SyscallConn()
	if err != nil {
		return nil, err
	}
	var h uintptr
	if err := rc.Control(func(u uintptr) { h = u }); err != nil {
		return nil, err
	}
	var mode uint32
	ret, _, _ := procGetConsoleMode.Call(h, uintptr(unsafe.Pointer(&mode))) // #nosec G103 -- Win32 out-parameter requires unsafe.Pointer
	return &Reader{f: f, h: h, console: ret != 0, con: winConsole{h}, cancelCh: make(chan struct{})}, nil
}

// SetResizeNotify registers fn to be called, from the reading goroutine, when
// the console reports a window-buffer-size event. fn must not block. Console
// input only; a pipe has no such events.
func (r *Reader) SetResizeNotify(fn func()) { r.onResize.Store(&fn) }

// Read waits until the file has input or the Reader is cancelled. After a
// cancel it returns ErrCanceled without reading, so pending input stays in
// the file for the next reader. A closed pipe reports io.EOF.
func (r *Reader) Read(p []byte) (int, error) {
	if r.console {
		// Wait for a key, discarding the events (resize, focus, ...) that
		// would leave ReadFile blocked, then read.
		notify := func() {
			if fn := r.onResize.Load(); fn != nil {
				(*fn)()
			}
		}
		if err := waitForKeyNotify(r.con, r.canceled.Load, notify); err != nil {
			return 0, err
		}
		return r.f.Read(p)
	}
	for {
		if r.canceled.Load() {
			return 0, ErrCanceled
		}
		ready, err := r.ready()
		if err != nil {
			return 0, err
		}
		if ready {
			return r.f.Read(p)
		}
	}
}

// ready reports whether a Read of a pipe or file would return without
// blocking, waiting up to one slice for that to become true. A console goes
// through waitForKey instead.
func (r *Reader) ready() (bool, error) {
	var avail uint32
	ret, _, callErr := procPeekNamedPipe.Call(r.h, 0, 0, 0, uintptr(unsafe.Pointer(&avail)), 0) // #nosec G103 -- Win32 out-parameter requires unsafe.Pointer
	if ret == 0 {
		switch callErr {
		case syscall.ERROR_BROKEN_PIPE, syscall.Errno(232), syscall.Errno(233): // ERROR_NO_DATA, ERROR_PIPE_NOT_CONNECTED
			return false, io.EOF
		}
		// Not a pipe (a regular file or NUL): reading it does not block.
		return true, nil
	}
	if avail > 0 {
		return true, nil
	}
	select {
	case <-r.cancelCh:
	case <-time.After(pipePollWait):
	}
	return false, nil
}

// Cancel makes a waiting Read return ErrCanceled. It is safe to call from any
// goroutine, any number of times, and after Close.
func (r *Reader) Cancel() {
	r.canceled.Store(true)
	r.once.Do(func() { close(r.cancelCh) })
}

// Close does nothing: the Reader owns no handle.
func (r *Reader) Close() {}
