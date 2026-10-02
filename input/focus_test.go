package input

import "testing"

func TestFocusEvents(t *testing.T) {
	got := readAll(t, "\x1b[I\x1b[O", 2)
	want := []Event{FocusEvent{Focused: true}, FocusEvent{Focused: false}}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("event #%d = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func TestFocusSequenceDoesNotLeakBytes(t *testing.T) {
	got := readAll(t, "\x1b[Ia\x1b[Ob", 4)
	if got[0] != (FocusEvent{Focused: true}) || got[2] != (FocusEvent{Focused: false}) {
		t.Errorf("focus events = %#v, %#v", got[0], got[2])
	}
	for _, i := range []int{1, 3} {
		k, ok := got[i].(Key)
		want := "a"
		if i == 3 {
			want = "b"
		}
		if !ok || k.String() != want {
			t.Errorf("event #%d = %#v, want key %q", i, got[i], want)
		}
	}
}
