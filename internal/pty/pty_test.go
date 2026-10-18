//go:build darwin || linux

package pty

import (
	"os"
	"syscall"
	"testing"
	"unsafe"
)

func TestOpenGivesAPairThatTalks(t *testing.T) {
	master, slave, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer slave.Close()
	if _, err := master.WriteString("x"); err != nil {
		t.Fatal(err)
	}
	if err := SetSize(master, 90, 31); err != nil {
		t.Fatal(err)
	}
	var ws struct{ Row, Col, Xpixel, Ypixel uint16 }
	if err := ioctl(slave.Fd(), syscall.TIOCGWINSZ, unsafe.Pointer(&ws)); err != nil { // #nosec G103 -- ioctl requires unsafe.Pointer
		t.Fatal(err)
	}
	if ws.Col != 90 || ws.Row != 31 {
		t.Errorf("the slave reports %dx%d, want 90x31", ws.Col, ws.Row)
	}
}

func TestSetSizeReportsErrors(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "plain")
	if err != nil {
		t.Fatal(err)
	}
	if err := SetSize(f, 80, 24); err == nil {
		t.Error("a regular file took a window size")
	}
	f.Close()
	if err := SetSize(f, 80, 24); err == nil {
		t.Error("a closed file took a window size")
	}
}
