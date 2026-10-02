package ansi

import "testing"

func TestCursorMovement(t *testing.T) {
	for _, tc := range []struct{ got, want string }{
		{CursorPosition(3, 7), "\x1b[3;7H"},
		{CursorPosition(1, 1), "\x1b[1;1H"},
		{CursorUp(2), "\x1b[2A"},
		{CursorDown(4), "\x1b[4B"},
		{CursorForward(5), "\x1b[5C"},
		{CursorBack(6), "\x1b[6D"},
	} {
		if tc.got != tc.want {
			t.Errorf("got %q, want %q", tc.got, tc.want)
		}
	}
}

func TestOSC52Copy(t *testing.T) {
	if got, want := OSC52Copy("hello"), "\x1b]52;c;aGVsbG8=\x07"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got, want := OSC52Copy(""), "\x1b]52;c;\x07"; got != want {
		t.Errorf("empty: got %q, want %q", got, want)
	}
}

// otherColor implements Color without being one of the three built-in types.
type otherColor struct{}

func (otherColor) fgCode() string { return "F" }
func (otherColor) bgCode() string { return "B" }
func (otherColor) ulCode() string { return "U" }

func TestAppendSGRColorFallback(t *testing.T) {
	for _, tc := range []struct {
		slot int
		want string
	}{{slotFg, "pF"}, {slotBg, "pB"}, {slotUl, "pU"}} {
		if got := string(appendSGRColor([]byte("p"), otherColor{}, tc.slot)); got != tc.want {
			t.Errorf("slot %d: got %q, want %q", tc.slot, got, tc.want)
		}
	}
	if got := string(appendSGRColor(nil, Color256(9), slotBg)); got != "48;5;9" {
		t.Errorf("Color256 bg: got %q", got)
	}
	if got := string(appendSGRColor(nil, RGB{1, 2, 3}, slotFg)); got != "38;2;1;2;3" {
		t.Errorf("RGB fg: got %q", got)
	}
}

func TestParseColonColor(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want Color
		ok   bool
	}{
		{"5:200", Color256(200), true},
		{"5:0", Color256(0), true},
		{"5:255", Color256(255), true},
		{"5:256", nil, false},
		{"5:-1", nil, false},
		{"5:x", nil, false},
		{"2::1:2:3", RGB{1, 2, 3}, true},
		{"2:1:2:3", RGB{1, 2, 3}, true},
		{"2:1:2:300", nil, false},
		{"2:1:2:x", nil, false},
		{"2:1:2", nil, false},
		{"7:1", nil, false},
		{"", nil, false},
	} {
		got, ok := parseColonColor(tc.in)
		if ok != tc.ok || got != tc.want {
			t.Errorf("parseColonColor(%q) = %v, %v; want %v, %v", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

func TestRGBTo256(t *testing.T) {
	for _, tc := range []struct {
		in   RGB
		want Color256
	}{
		{RGB{0, 0, 0}, 16},
		{RGB{7, 7, 7}, 16},
		{RGB{255, 255, 255}, 231},
		{RGB{249, 249, 249}, 231},
		{RGB{8, 8, 8}, 232},
		{RGB{248, 248, 248}, 255},
		{RGB{255, 0, 0}, 196},
		{RGB{0, 255, 0}, 46},
		{RGB{0, 0, 255}, 21},
	} {
		if got := rgbTo256(tc.in); got != tc.want {
			t.Errorf("rgbTo256(%v) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestColor256RGB(t *testing.T) {
	for _, tc := range []struct {
		in   Color256
		want RGB
	}{
		{0, RGB{0, 0, 0}},
		{9, RGB{255, 0, 0}},
		{15, RGB{255, 255, 255}},
		{16, RGB{0, 0, 0}},
		{17, RGB{0, 0, 95}},
		{196, RGB{255, 0, 0}},
		{231, RGB{255, 255, 255}},
		{232, RGB{8, 8, 8}},
		{255, RGB{238, 238, 238}},
	} {
		if got := color256RGB(tc.in); got != tc.want {
			t.Errorf("color256RGB(%d) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestClamp8(t *testing.T) {
	for in, want := range map[int]uint8{-5: 0, 0: 0, 7: 7, 255: 255, 256: 255, 1000: 255} {
		if got := clamp8(in); got != want {
			t.Errorf("clamp8(%d) = %d, want %d", in, got, want)
		}
	}
}

func TestTrimLeftWidth(t *testing.T) {
	for _, tc := range []struct {
		name, in string
		w        int
		want     string
	}{
		{"zero", "abc", 0, "abc"},
		{"negative", "abc", -3, "abc"},
		{"plain", "abcdef", 2, "cdef"},
		{"all", "abc", 3, ""},
		{"beyond", "abc", 10, ""},
		{"wide whole", "世界x", 2, "界x"},
		{"wide split", "世界", 1, " 界"},
		{"style reapplied", "\x1b[31mabcd\x1b[0m", 2, "\x1b[31mcd\x1b[0m"},
		{"style closed before cut", "\x1b[31mab\x1b[0mcd", 3, "d"},
		{"style leading escapes only", "\x1b[1mab", 2, "\x1b[1m"},
	} {
		if got := TrimLeftWidth(tc.in, tc.w); got != tc.want {
			t.Errorf("%s: TrimLeftWidth(%q, %d) = %q, want %q", tc.name, tc.in, tc.w, got, tc.want)
		}
	}
}
