package ansi

import "testing"

// A family emoji is one cluster of width 2, or five runes (2+2+2+2+0 = 8 with
// the ZWJs) counted on their own.
const family = "👨‍👩‍👧"

func TestMeasurerZeroFollowsTheProcessSetting(t *testing.T) {
	t.Cleanup(func() { SetClusterWidth(true) })
	var m Measurer
	for _, on := range []bool{true, false} {
		SetClusterWidth(on)
		if m.Clusters() != on {
			t.Fatalf("zero Measurer Clusters() = %v with the process setting %v", m.Clusters(), on)
		}
		if got, want := m.Width(family), Width(family); got != want {
			t.Fatalf("setting %v: Measurer.Width = %d, Width = %d", on, got, want)
		}
	}
}

func TestMeasurersDisagreeWithoutTouchingEachOtherOrTheDefault(t *testing.T) {
	t.Cleanup(func() { SetClusterWidth(true) })
	SetClusterWidth(true)
	on, off := ClusterMeasurer(true), ClusterMeasurer(false)
	if !on.Clusters() || off.Clusters() {
		t.Fatalf("Clusters: on=%v off=%v", on.Clusters(), off.Clusters())
	}
	if w := on.Width(family); w != 2 {
		t.Fatalf("clusters on: Width = %d, want 2", w)
	}
	if w := off.Width(family); w <= 2 {
		t.Fatalf("clusters off: Width = %d, want more than 2 (each rune counted)", w)
	}
	// Using a pinned Measurer never changes the process setting.
	if w := Width(family); w != 2 {
		t.Fatalf("package Width = %d after using Measurers, want 2", w)
	}
	// And a pinned Measurer ignores a later change to the process setting.
	SetClusterWidth(false)
	if w := on.Width(family); w != 2 {
		t.Fatalf("pinned-on Measurer followed the process setting: Width = %d", w)
	}
}

func TestMeasurerTruncateAndTrimLeftUseItsOwnSetting(t *testing.T) {
	t.Cleanup(func() { SetClusterWidth(true) })
	SetClusterWidth(true)
	s := family + "abc"
	if got := ClusterMeasurer(true).Truncate(s, 3); got != family+"a" {
		t.Fatalf("clusters on: Truncate = %q, want the family and one letter", got)
	}
	if got := ClusterMeasurer(false).Truncate(s, 3); got == family+"a" {
		t.Fatalf("clusters off: Truncate kept the whole cluster: %q", got)
	}
	if got := ClusterMeasurer(true).TrimLeftWidth(s, 2); got != "abc" {
		t.Fatalf("clusters on: TrimLeftWidth = %q, want abc", got)
	}
}

func TestMeasurerIsComparable(t *testing.T) {
	if (Measurer{}) != (Measurer{}) || ClusterMeasurer(true) == ClusterMeasurer(false) || ClusterMeasurer(true) == (Measurer{}) {
		t.Fatal("Measurer equality is wrong")
	}
}

// The package-level functions are the zero Measurer: same answers on a corpus.
func TestMeasurerMatchesPackageFunctions(t *testing.T) {
	for _, s := range []string{"", "abc", "日本語", family, "\x1b[1mbold\x1b[0m", "éx", "🇯🇵🇺🇸", "a\x1b]8;;http://x\x07link\x1b]8;;\x07"} {
		var m Measurer
		if m.Width(s) != Width(s) || m.Truncate(s, 2) != Truncate(s, 2) || m.TrimLeftWidth(s, 1) != TrimLeftWidth(s, 1) {
			t.Fatalf("Measurer{} disagrees with the package functions on %q", s)
		}
	}
}
