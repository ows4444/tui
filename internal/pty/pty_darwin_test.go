//go:build darwin

package pty

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenReportsEachFailure(t *testing.T) {
	defer func(p string) { ptmxPath = p }(ptmxPath)

	ptmxPath = filepath.Join(t.TempDir(), "missing")
	if _, _, err := Open(); err == nil {
		t.Error("a missing ptmx opened")
	}
	// A device that is not a pseudo-terminal refuses the grant request.
	ptmxPath = os.DevNull
	if _, _, err := Open(); err == nil {
		t.Error("/dev/null opened as a ptmx")
	}
}
