package ansi

import "testing"

func TestTruncateDropsUnterminatedEscape(t *testing.T) {
	got := Truncate("00\x1b[", 3)
	if got != "00" || Width(got) != 2 {
		t.Errorf("Truncate = %q (width %d), want %q", got, Width(got), "00")
	}
}
