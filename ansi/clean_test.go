package ansi

import (
	"strings"
	"testing"
)

func TestCleanStripsEscapesUnlessRaw(t *testing.T) {
	evil := "a\x1b]52;c;ZXZpbA==\x07b\x1b[2Jc\td"
	if got := Clean(false, evil); strings.ContainsRune(got, 0x1b) || got != "abc     d" {
		t.Errorf("Clean = %q", got)
	}
	if got := Clean(true, evil); got != evil {
		t.Errorf("Clean(raw) = %q, want unchanged", got)
	}
	got := CleanAll(false, []string{"x\x1b[2J", "y"})
	if len(got) != 2 || got[0] != "x" || got[1] != "y" {
		t.Errorf("CleanAll = %q", got)
	}
}
