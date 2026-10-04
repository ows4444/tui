package capprobe

import "testing"

func observe(t *testing.T, p *Parser, kind byte, data string) (consumed, done bool, res Result) {
	t.Helper()
	return p.Observe(kind, data)
}

func TestDECRPMValues(t *testing.T) {
	for _, tc := range []struct {
		val       string
		supported bool
	}{{"0", false}, {"1", true}, {"2", true}, {"3", true}, {"4", false}} {
		var p Parser
		for _, mode := range []string{"2026", "2027"} {
			if c, d, _ := observe(t, &p, '[', "?"+mode+";"+tc.val+"$y"); !c || d {
				t.Fatalf("mode %s val %s: consumed=%v done=%v", mode, tc.val, c, d)
			}
		}
		_, done, res := observe(t, &p, '[', "?62;22c")
		if !done {
			t.Fatal("DA1 did not end the probe")
		}
		if res.SyncOutput != tc.supported || res.GraphemeClusters != tc.supported {
			t.Errorf("val %s: %+v, want both %v", tc.val, res, tc.supported)
		}
	}
}

func TestMalformedDECRPMIsConsumedButIgnored(t *testing.T) {
	for _, d := range []string{"?2026$y", "?x;1$y", "?2026;x$y", "?;$y"} {
		var p Parser
		if c, done, _ := observe(t, &p, '[', d); !c || done {
			t.Errorf("%q: consumed=%v done=%v", d, c, done)
		}
		if _, _, res := observe(t, &p, '[', "?62c"); res != (Result{}) {
			t.Errorf("%q changed the result: %+v", d, res)
		}
	}
}

func TestKittyKeyboardGraphicsNotifyXTVersion(t *testing.T) {
	var p Parser
	for _, r := range []struct {
		kind byte
		data string
	}{
		{'[', "?0u"},
		{'_', "Gi=31;OK"},
		{']', "99;i=32:p=?;p=title"},
		{'P', ">|WezTerm 2024"},
	} {
		if c, d, _ := observe(t, &p, r.kind, r.data); !c || d {
			t.Fatalf("%c %q: consumed=%v done=%v", r.kind, r.data, c, d)
		}
	}
	_, done, res := observe(t, &p, '[', "?62;4c")
	want := Result{KittyKeyboard: true, KittyGraphics: true, Sixel: true, Notifications: true, XTVersion: "WezTerm 2024", StyledUnderline: true, InlineImages: true}
	if !done || res != want {
		t.Fatalf("done=%v res=%+v, want %+v", done, res, want)
	}
}

func TestKittyGraphicsWithoutOKIsFalse(t *testing.T) {
	var p Parser
	observe(t, &p, '_', "Gi=31;ENOTSUPPORTED:no")
	if _, _, res := observe(t, &p, '[', "?62c"); res.KittyGraphics {
		t.Fatalf("KittyGraphics true for an error reply: %+v", res)
	}
}

func TestRepliesThatAreNotTheProbesAreNotConsumed(t *testing.T) {
	var p Parser
	for _, r := range []struct {
		kind byte
		data string
	}{
		{'[', "1;2R"},        // a cursor position report
		{'[', "?2026;1x"},    // not DECRPM
		{'P', "1+r5463"},     // XTGETTCAP, not XTVERSION
		{']', "52;c;aGk="},   // a clipboard reply
		{']', "99;i=7:p=?;"}, // a notification reply for another id
		{'_', "Gi=7;OK"},     // kitty graphics for another id
		{'X', "anything"},    // SOS
	} {
		if c, d, _ := observe(t, &p, r.kind, r.data); c || d {
			t.Errorf("%c %q: consumed=%v done=%v, want neither", r.kind, r.data, c, d)
		}
	}
}

func TestInferStyledUnderline(t *testing.T) {
	for _, tc := range []struct {
		r    Result
		want bool
	}{
		{Result{}, false},
		{Result{KittyKeyboard: true}, true},
		{Result{KittyGraphics: true}, true},
		{Result{XTVersion: "kitty(0.35.2)"}, true},
		{Result{XTVersion: "Ghostty 1.0"}, true},
		{Result{XTVersion: "xterm(388)"}, false},
		{Result{XTVersion: "tmux 3.4"}, false},
	} {
		if got := InferStyledUnderline(tc.r); got != tc.want {
			t.Errorf("%+v: %v, want %v", tc.r, got, tc.want)
		}
	}
}

func TestQueriesEndWithDA1(t *testing.T) {
	if got := Queries[len(Queries)-len(QueryDA1):]; got != QueryDA1 {
		t.Fatalf("Queries ends with %q, want the DA1 sentinel last", got)
	}
}

// DA1 lists the terminal's feature codes; 4 is Sixel graphics.
func TestDA1AttributeFourMeansSixel(t *testing.T) {
	for _, c := range []struct {
		reply string
		want  bool
	}{
		{"?62;4c", true},
		{"?64;1;2;4;6;9;15;22c", true},
		{"?4c", true},
		{"?62;22c", false},
		{"?62;14;24c", false}, // 14 and 24 are not 4
		{"?62c", false},
		{"?c", false},
	} {
		var p Parser
		if _, done, res := observe(t, &p, '[', c.reply); !done || res.Sixel != c.want {
			t.Errorf("%q: done=%v Sixel=%v, want %v", c.reply, done, res.Sixel, c.want)
		}
	}
}

// Inline images are inferred from the terminal's name: iTerm2 and WezTerm
// draw them, and no other terminal is assumed to.
func TestInferInlineImages(t *testing.T) {
	for version, want := range map[string]bool{
		"iTerm2 3.5.10":         true,
		"ITERM2 3.6":            true,
		"WezTerm 20240203":      true,
		"kitty(0.35.2)":         false,
		"ghostty 1.0":           false,
		"XTerm(390)":            false,
		"":                      false,
		"not iTerm2":            false,
		"tmux 3.4":              false,
		"Alacritty 0.13":        false,
		"foot(1.16.2)":          false,
		"Konsole 23.08":         false,
		"vte(7600)":             false,
		"contour 0.4":           false,
		"mintty 3.7":            false,
		"Apple_Terminal 455":    false,
		"WarpTerminal":          false,
		"iterm":                 false,
		"wez":                   false,
		" iTerm2 3.5":           false,
		"iTerm2":                true,
		"wezterm":               true,
		"WezTerm":               true,
		"iTerm2 3.5.0beta1":     true,
		"wezterm 20240203-1100": true,
	} {
		if got := InferInlineImages(Result{XTVersion: version}); got != want {
			t.Errorf("InferInlineImages(%q) = %v, want %v", version, got, want)
		}
	}
	// A kitty protocol says nothing about inline images.
	if InferInlineImages(Result{KittyGraphics: true, KittyKeyboard: true, Sixel: true}) {
		t.Error("a terminal with no name was taken to draw inline images")
	}
}
