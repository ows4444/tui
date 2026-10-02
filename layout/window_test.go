package layout

import (
	"fmt"
	"testing"
)

func TestWindowKeepsTheCursorVisibleAndClamps(t *testing.T) {
	lines := make([]string, 20)
	for i := range lines {
		lines[i] = fmt.Sprint(i)
	}
	for _, h := range []int{1, 3, 4, 7, 19} {
		for cursor := -2; cursor < 23; cursor++ {
			got := Window(lines, cursor, h)
			if len(got) != h {
				t.Fatalf("h=%d cursor=%d: %d lines", h, cursor, len(got))
			}
			want := cursor
			if want < 0 {
				want = 0
			}
			if want > 19 {
				want = 19
			}
			found := false
			for _, l := range got {
				if l == fmt.Sprint(want) {
					found = true
				}
			}
			if !found {
				t.Errorf("h=%d cursor=%d: %v does not contain the (clamped) cursor line", h, cursor, got)
			}
		}
	}
	if got := Window(lines[:3], 1, 10); len(got) != 3 {
		t.Errorf("fitting lines changed: %v", got)
	}
	if got := Window(lines, 5, 0); len(got) != 20 {
		t.Errorf("h<=0 should return lines unchanged")
	}
}
