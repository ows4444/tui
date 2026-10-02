//go:build !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !windows

package cancelreader

import (
	"os"
	"time"
)

// Reader falls back to read deadlines where select(2) isn't available. This
// is what Program did before this package existed: best effort, and
// unverified on these platforms.
type Reader struct{ f *os.File }

// New returns a Reader for f.
func New(f *os.File) (*Reader, error) { return &Reader{f: f}, nil }

// Read reads from the underlying file.
func (r *Reader) Read(p []byte) (int, error) { return r.f.Read(p) }

// Cancel makes a waiting Read fail by expiring the file's read deadline.
func (r *Reader) Cancel() { _ = r.f.SetReadDeadline(time.Now()) }

// Close does nothing here.
func (r *Reader) Close() {}
