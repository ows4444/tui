//go:build linux

package pty

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenReportsEachFailure(t *testing.T) {
	defer func(m, p string) { ptmxPath, ptsDir = m, p }(ptmxPath, ptsDir)

	ptmxPath = filepath.Join(t.TempDir(), "missing")
	if _, _, err := Open(); err == nil {
		t.Error("a missing ptmx opened")
	}

	// A device that is not a pseudo-terminal refuses the unlock request.
	ptmxPath = os.DevNull
	if _, _, err := Open(); err == nil {
		t.Error("/dev/null opened as a ptmx")
	}

	// A real ptmx with a slave directory that does not hold the slave.
	ptmxPath, ptsDir = "/dev/ptmx", t.TempDir()+"/"
	if _, _, err := Open(); err == nil {
		t.Error("a missing slave opened")
	}
}
