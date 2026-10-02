//go:build darwin || linux

package cancelreader

import (
	"os"
	"syscall"
	"testing"
)

func TestNewRejectsAClosedFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "x")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	if r, err := New(f); err == nil {
		r.Close()
		t.Fatal("New(closed file) succeeded, want an error")
	}
}

func TestNewRejectsDescriptorTooLargeForSelect(t *testing.T) {
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer pr.Close()
	defer pw.Close()
	fd := int(mustFd(pr))
	hi, _, errno := syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), syscall.F_DUPFD, uintptr(fdSetSize+10))
	if errno != 0 {
		t.Skipf("cannot allocate a descriptor >= %d: %v", fdSetSize, errno)
	}
	big := os.NewFile(hi, "big")
	defer big.Close()
	if r, err := New(big); err == nil {
		r.Close()
		t.Fatal("New(fd >= FD_SETSIZE) succeeded, want an error")
	}
}

func TestReadReportsAnUnderlyingReadError(t *testing.T) {
	r, pr, pw := newPipe(t)
	_ = pw
	// Close the read end behind the Reader's back: select then reports EBADF.
	pr.Close()
	buf := make([]byte, 4)
	if _, err := r.Read(buf); err == nil || err == ErrCanceled {
		t.Fatalf("Read on a closed descriptor = %v, want a syscall error", err)
	}
}

func TestReadReturnsTheSyscallErrorFromRead(t *testing.T) {
	// select reports a directory readable, but read(2) on it fails (EISDIR).
	d, err := os.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	r, err := New(d)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if _, err := r.Read(make([]byte, 4)); err == nil || err == ErrCanceled {
		t.Fatalf("Read on a directory = %v, want a syscall error", err)
	}
}

// select(2) can fail with EINTR or wake without either descriptor readable;
// Read must retry both and still deliver data that arrives afterwards.
func TestReadRetriesInterruptedAndSpuriousSelect(t *testing.T) {
	r, _, pw := newPipe(t)
	real := selectFn
	defer func() { selectFn = real }()
	calls := 0
	selectFn = func(n int, set *syscall.FdSet) error {
		calls++
		switch calls {
		case 1:
			return syscall.EINTR
		case 2:
			*set = syscall.FdSet{} // woke with nothing readable
			return nil
		}
		return real(n, set)
	}
	if _, err := pw.Write([]byte("ok")); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 4)
	n, err := r.Read(buf)
	if err != nil || string(buf[:n]) != "ok" {
		t.Fatalf("Read = %q, %v; want \"ok\", nil", buf[:n], err)
	}
	if calls < 3 {
		t.Errorf("select called %d times, want the EINTR and spurious wakeups retried", calls)
	}
}
