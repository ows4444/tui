package input

import "testing"

// Criteria : parser correctness gaps.

func TestFunctionKeysTilde(t *testing.T) { // #10
	codes := map[string]KeyType{
		"15": KeyF5, "17": KeyF6, "18": KeyF7, "19": KeyF8, "20": KeyF9,
		"21": KeyF10, "23": KeyF11, "24": KeyF12,
		"25": KeyF13, "26": KeyF14, "28": KeyF15, "29": KeyF16,
		"31": KeyF17, "32": KeyF18, "33": KeyF19, "34": KeyF20,
	}
	for c, want := range codes {
		got := readAllKeys(t, "\x1b["+c+"~", 1)[0]
		if got.Type != want {
			t.Errorf("ESC[%s~ = %v, want type %d", c, got, want)
		}
	}
	if k := readAllKeys(t, "\x1b[15;5~", 1)[0]; k.Type != KeyF5 || k.Mod != ModCtrl {
		t.Errorf("ESC[15;5~ = %#v", k)
	}
	// Kitty functional codepoints for F13-F24 (57376..57387).
	for i := 0; i < 12; i++ {
		s := "\x1b[" + itoa(57376+i) + "u"
		if k := readAllKeys(t, s, 1)[0]; k.Type != KeyF13+KeyType(i) {
			t.Errorf("%q = %#v, want F%d", s, k, 13+i)
		}
	}
	if KeyF24-KeyF5 != 19 || KeyF5 <= KeyUnknown {
		t.Errorf("F-key enum not contiguous/appended")
	}
	if got := (Key{Type: KeyF24}).String(); got != "f24" {
		t.Errorf("String = %q", got)
	}
	if got := (Key{Type: KeyF5, Mod: ModShift}).String(); got != "shift+f5" {
		t.Errorf("String = %q", got)
	}
}

func itoa(n int) string {
	s := ""
	for ; n > 0; n /= 10 {
		s = string(rune('0'+n%10)) + s
	}
	return s
}

func TestInsertKey(t *testing.T) { // #11
	if k := readAllKeys(t, "\x1b[2~", 1)[0]; k.Type != KeyInsert {
		t.Errorf("got %#v", k)
	}
	if got := (Key{Type: KeyInsert}).String(); got != "insert" {
		t.Errorf("String = %q", got)
	}
}

func TestCSIConsumedThroughFinal(t *testing.T) { // #12
	for _, s := range []string{
		"\x1b[?62;22c", "\x1b[>0;95;0c", "\x1b[=1c", "\x1b[?2026;2$y",
		"\x1b[0 q", "\x1b[!p", "\x1b[<1;2;3t", "\x1b[?1;2c",
	} {
		keys := readAllKeys(t, s+"x", 2)
		if keys[0].Type != KeyUnknown {
			t.Errorf("%q first = %#v, want KeyUnknown", s, keys[0])
		}
		if keys[1].Type != KeyRunes || keys[1].Text != "x" {
			t.Errorf("%q leaked: second = %#v", s, keys[1])
		}
	}
}

func TestX10MouseConsumed(t *testing.T) {
	keys := readAllKeys(t, "\x1b[M !!x", 2)
	if keys[0].Type != KeyUnknown || keys[1].Text != "x" {
		t.Errorf("got %#v", keys)
	}
}

func TestAltEnterBackspaceTab(t *testing.T) { // #13
	tests := []struct {
		in   string
		want Key
	}{
		{"\x1b\r", Key{Type: KeyEnter, Mod: ModAlt}},
		{"\x1b\x7f", Key{Type: KeyBackspace, Mod: ModAlt}},
		{"\x1b\x08", Key{Type: KeyBackspace, Mod: ModAlt}},
		{"\x1b\t", Key{Type: KeyTab, Mod: ModAlt}},
		{"\x1b\x1b", Key{Type: KeyEsc, Mod: ModAlt}},
	}
	for _, tt := range tests {
		got := readAllKeys(t, tt.in, 1)[0]
		if got.Type != tt.want.Type || got.Mod != tt.want.Mod {
			t.Errorf("%q = %#v, want %#v", tt.in, got, tt.want)
		}
	}
}

func TestControlBytes(t *testing.T) { // #14
	tests := []struct{ in, str string }{
		{"\x00", "ctrl+space"}, {"\x1c", "ctrl+\\"}, {"\x1d", "ctrl+]"},
		{"\x1e", "ctrl+^"}, {"\x1f", "ctrl+_"},
	}
	for _, tt := range tests {
		k := readAllKeys(t, tt.in, 1)[0]
		if k.Type != KeyCtrl || k.String() != tt.str {
			t.Errorf("%q = %#v (%q), want %q", tt.in, k, k.String(), tt.str)
		}
	}
}

func TestInvalidUTF8(t *testing.T) { // #15
	for _, s := range []string{"\xff", "\x80", "\xc3(", "\xe2\x82(", "\xf0\x9f(", "\xc0\x80", "\xed\xa0\x80"} {
		keys := readAllKeys(t, s+"z", 2)
		if keys[0].Type != KeyRunes || len([]rune(keys[0].Text)) != 1 || keys[0].Code != '�' {
			t.Errorf("%q first = %#v, want U+FFFD", s, keys[0])
		}
		if last := keys[len(keys)-1]; last.Text != "z" && len(s) < 2 {
			t.Errorf("%q: last = %#v", s, last)
		}
	}
	// Truncated sequence must not swallow the following ASCII byte.
	keys := readAllKeys(t, "\xc3z", 2)
	if keys[1].Code != 'z' {
		t.Errorf("swallowed: %#v", keys)
	}
}

func TestKittyPUA(t *testing.T) { // #16
	for _, s := range []string{"\x1b[57344u", "\x1b[57414;5u", "\x1b[63743u", "\x1b[983040u"} {
		k := readAllKeys(t, s, 1)[0]
		if k.Type == KeyRunes {
			t.Errorf("%q emitted KeyRunes: %#v", s, k)
		}
	}
}
