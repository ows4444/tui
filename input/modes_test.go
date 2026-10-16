package input

import "testing"

func TestColorSchemeReports(t *testing.T) {
	got := readAll(t, "\x1b[?997;1n\x1b[?997;2n", 2)
	want := []Event{ColorSchemeEvent{Dark: true}, ColorSchemeEvent{Dark: false}}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("event #%d = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func TestOtherPrivateReplyIsNotAColorScheme(t *testing.T) {
	got := readAll(t, "\x1b[?997;3na", 2)
	if _, ok := got[0].(ColorSchemeEvent); ok {
		t.Errorf("unknown scheme value decoded as %#v", got[0])
	}
	if k, ok := got[1].(Key); !ok || k.String() != "a" {
		t.Errorf("byte after the reply = %#v, want key a", got[1])
	}
}

func TestInBandResizeReports(t *testing.T) {
	got := readAll(t, "\x1b[48;24;80;480;800t\x1b[48;30;100t", 2)
	want := []Event{
		InBandResizeEvent{Rows: 24, Cols: 80, HeightPx: 480, WidthPx: 800},
		InBandResizeEvent{Rows: 30, Cols: 100},
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("event #%d = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func TestOtherWindowOpsAreNotResizeReports(t *testing.T) {
	// ESC[8;24;80t is a window-manipulation command, and ESC[48t lacks a size.
	for _, in := range []string{"\x1b[8;24;80t", "\x1b[48t", "\x1b[48;5t"} {
		got := readAll(t, in, 1)
		if _, ok := got[0].(InBandResizeEvent); ok {
			t.Errorf("%q decoded as %#v", in, got[0])
		}
	}
}
