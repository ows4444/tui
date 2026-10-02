package ansi

import "testing"

func TestWidthKeycap(t *testing.T) {
	SetClusterWidth(true)
	const k = "1️⃣"
	for _, tc := range []struct {
		name, in string
		want     int
	}{
		{"digit", k, 2},
		{"hash", "#️⃣", 2},
		{"star", "*️⃣", 2},
		{"after ascii run", "ab" + k, 4},
		{"between text", "a" + k + "b", 4},
		{"styled", "\x1b[1m" + k + "\x1b[0m", 2},
		{"two", k + k, 4},
		{"no vs16 stays narrow", "1⃣", 1},
		{"bare digit", "1", 1},
	} {
		if got := Width(tc.in); got != tc.want {
			t.Errorf("%s: Width = %d, want %d", tc.name, got, tc.want)
		}
		if got := plainWidth(StripANSI(tc.in)); got != tc.want {
			t.Errorf("%s: plainWidth = %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestKeycapTruncateAgreesWithWidth(t *testing.T) {
	SetClusterWidth(true)
	const k = "1️⃣"
	if got := Truncate("ab"+k+"cd", 4); got != "ab"+k {
		t.Errorf("Truncate = %q", got)
	}
	if got := Truncate("ab"+k, 3); got != "ab" {
		t.Errorf("Truncate to 3 = %q, want %q", got, "ab")
	}
	if got := TrimLeftWidth("ab"+k+"cd", 2); got != k+"cd" {
		t.Errorf("TrimLeftWidth = %q", got)
	}
	if got := TrimLeftWidth("ab"+k+"cd", 3); got != " cd" { // a cluster split by the trim is padded
		t.Errorf("TrimLeftWidth 3 = %q, want %q", got, " cd")
	}
}
