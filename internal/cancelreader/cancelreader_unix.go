//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package cancelreader

import (
	"errors"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"
)

// fdSetSize is select(2)'s FD_SETSIZE: descriptors at or above it can't be
// waited on. It is the number of bits in the platform's FdSet (1024 on Linux,
// macOS, FreeBSD, OpenBSD and DragonFly, 256 on NetBSD), so a descriptor the set
// cannot hold is refused instead of indexing past it.
var fdSetSize = fdSetBits(&syscall.FdSet{})

// selectFn is selectRead, replaceable so tests can inject EINTR and spurious
// wakeups that a real select rarely produces.
var selectFn = selectRead

// Reader reads from a file and can be cancelled. Read is meant to be called
// from one goroutine; Cancel may be called from any goroutine, any number of
// times. Close releases the cancel pipe and must only be called once Read
// has returned (the reading goroutine defers it).
type Reader struct {
	fd       int
	cancelR  *os.File
	cancelW  *os.File
	rfd      int
	canceled atomic.Bool

	mu     sync.Mutex
	closed bool
}

// New returns a Reader for f. It does not take ownership of f, change its
// blocking mode, or close it.
func New(f *os.File) (*Reader, error) {
	fd, err := descriptor(f)
	if err != nil {
		return nil, err
	}
	cr, cw, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	rfd, err := descriptor(cr)
	if err != nil {
		_ = cr.Close()
		_ = cw.Close()
		return nil, err
	}
	if fd >= fdSetSize || rfd >= fdSetSize {
		_ = cr.Close()
		_ = cw.Close()
		return nil, errors.New("cancelreader: descriptor too large for select")
	}
	return &Reader{fd: fd, cancelR: cr, cancelW: cw, rfd: rfd}, nil
}

// descriptor returns f's descriptor without changing its blocking mode
// (unlike File.Fd).
func descriptor(f *os.File) (int, error) {
	rc, err := f.SyscallConn()
	if err != nil {
		return 0, err
	}
	fd := -1
	if err := rc.Control(func(u uintptr) { fd = int(u) }); err != nil {
		return 0, err
	}
	return fd, nil
}

// Read waits until the file is readable or the Reader is cancelled. After a
// cancel it returns ErrCanceled without reading, so pending input stays in
// the file for the next reader.
func (r *Reader) Read(p []byte) (int, error) {
	for {
		if r.canceled.Load() {
			return 0, ErrCanceled
		}
		var set syscall.FdSet
		fdSetAdd(&set, r.fd)
		fdSetAdd(&set, r.rfd)
		n := r.fd
		if r.rfd > n {
			n = r.rfd
		}
		if err := selectFn(n+1, &set); err != nil {
			if err == syscall.EINTR {
				continue
			}
			return 0, err
		}
		if fdIsSet(&set, r.rfd) || r.canceled.Load() {
			return 0, ErrCanceled // cancel wins over pending input
		}
		if !fdIsSet(&set, r.fd) {
			continue
		}
		n2, err := syscall.Read(r.fd, p)
		switch {
		case err == syscall.EINTR || err == syscall.EAGAIN:
			continue
		case err != nil:
			return 0, err
		case n2 == 0:
			return 0, io.EOF
		}
		return n2, nil
	}
}

// Cancel makes a waiting Read, and every later one, return ErrCanceled.
func (r *Reader) Cancel() {
	r.canceled.Store(true)
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.closed {
		_, _ = r.cancelW.Write([]byte{0}) // a full pipe already means "cancelled"
	}
}

// Close releases the cancel pipe. It is safe to call more than once.
func (r *Reader) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}
	r.closed = true
	_ = r.cancelR.Close()
	_ = r.cancelW.Close()
}

// word is the element type of an FdSet's bit array, which differs by platform.
type word interface {
	~int32 | ~int64 | ~uint32 | ~uint64
}

func wordBits[W word](words []W) int {
	var w W
	return int(unsafe.Sizeof(w)) * 8
}

// setBit sets bit fd of the bit array words.
func setBit[W word](words []W, fd int) {
	bits := wordBits(words)
	words[fd/bits] |= 1 << (uint(fd) % uint(bits))
}

// hasBit reports whether bit fd of words is set.
func hasBit[W word](words []W, fd int) bool {
	bits := wordBits(words)
	return words[fd/bits]&(1<<(uint(fd)%uint(bits))) != 0
}
