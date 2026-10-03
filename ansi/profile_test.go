package ansi

import (
	"bytes"
	"io"
	"os"
	"runtime"
	"testing"
)

func TestDetectColorProfileFor(t *testing.T) {
	tty, err := os.Open(os.DevNull) // character device
	if err != nil {
		t.Fatal(err)
	}
	defer tty.Close()
	file, err := os.CreateTemp(t.TempDir(), "out") // regular file: not a TTY
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var buf bytes.Buffer
	cases := []struct {
		name string
		out  io.Writer
		goos string
		vars map[string]string
		want Profile
	}{
		{"#51 WT_SESSION empty TERM", tty, "linux", map[string]string{"WT_SESSION": "x"}, TrueColor},
		{"#52 xterm-kitty", tty, "linux", map[string]string{"TERM": "xterm-kitty"}, TrueColor},
		{"#52 xterm-ghostty", tty, "linux", map[string]string{"TERM": "xterm-ghostty"}, TrueColor},
		{"#53 force without tty", &buf, "linux", map[string]string{"CLICOLOR_FORCE": "1", "TERM": "xterm-256color"}, ANSI256},
		{"#53 force on regular file", file, "linux", map[string]string{"CLICOLOR_FORCE": "1", "COLORTERM": "truecolor"}, TrueColor},
		{"#53 NO_COLOR beats force", &buf, "linux", map[string]string{"CLICOLOR_FORCE": "1", "NO_COLOR": "1", "TERM": "xterm"}, NoColor},
		{"#53 force 0 is not force", &buf, "linux", map[string]string{"CLICOLOR_FORCE": "0", "TERM": "xterm-256color"}, NoColor},
		{"#55 NO_COLOR beats COLORTERM", tty, "linux", map[string]string{"NO_COLOR": "1", "COLORTERM": "truecolor", "TERM": "xterm-256color"}, NoColor},
		{"#55 NO_COLOR beats TERM_PROGRAM and WT_SESSION", tty, "linux", map[string]string{"NO_COLOR": "x", "TERM_PROGRAM": "iTerm.app", "WT_SESSION": "1"}, NoColor},
		{"#55 empty NO_COLOR is ignored", tty, "linux", map[string]string{"NO_COLOR": "", "COLORTERM": "24bit"}, TrueColor},
		{"#54 buffer no force", &buf, "linux", map[string]string{"COLORTERM": "truecolor", "TERM": "xterm-256color"}, NoColor},
		{"#54 regular file no force", file, "linux", map[string]string{"COLORTERM": "truecolor"}, NoColor},
		{"#54 nil out", nil, "linux", map[string]string{"COLORTERM": "truecolor"}, NoColor},
		{"tty truecolor", tty, "linux", map[string]string{"COLORTERM": "truecolor"}, TrueColor},
		{"CLICOLOR=0 on tty", tty, "linux", map[string]string{"CLICOLOR": "0", "TERM": "xterm"}, NoColor},
		{"TERM_PROGRAM iTerm", tty, "linux", map[string]string{"TERM_PROGRAM": "iTerm.app", "TERM": "xterm"}, TrueColor},
		{"TERM_PROGRAM Apple_Terminal", tty, "linux", map[string]string{"TERM_PROGRAM": "Apple_Terminal", "TERM": "xterm"}, ANSI256},
		{"dumb tty", tty, "linux", map[string]string{"TERM": "dumb"}, NoColor},
		{"windows empty TERM", tty, "windows", map[string]string{}, ANSI16},
		{"unix empty TERM", tty, "linux", map[string]string{}, NoColor},
	}
	for _, c := range cases {
		if got := detectProfile(isTerminal(c.out), c.goos, env(c.vars)); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
		if c.goos == runtime.GOOS || c.goos == "linux" && runtime.GOOS != "windows" {
			if got := DetectColorProfileFor(c.out, env(c.vars)); got != c.want {
				t.Errorf("%s (public): got %v, want %v", c.name, got, c.want)
			}
		}
	}
}

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func TestDetectColorProfile(t *testing.T) {
	// An empty TERM is ANSI16 on Windows and NoColor elsewhere.
	unset := NoColor
	if runtime.GOOS == "windows" {
		unset = ANSI16
	}
	cases := []struct {
		name string
		env  map[string]string
		want Profile
	}{
		{"NO_COLOR wins over truecolor", map[string]string{"NO_COLOR": "1", "COLORTERM": "truecolor", "TERM": "xterm-256color"}, NoColor},
		{"empty NO_COLOR ignored", map[string]string{"NO_COLOR": "", "TERM": "xterm"}, ANSI16},
		{"COLORTERM truecolor", map[string]string{"COLORTERM": "truecolor", "TERM": "xterm"}, TrueColor},
		{"COLORTERM 24bit", map[string]string{"COLORTERM": "24bit", "TERM": "xterm"}, TrueColor},
		{"TERM 256color", map[string]string{"TERM": "xterm-256color"}, ANSI256},
		{"TERM plain xterm", map[string]string{"TERM": "xterm"}, ANSI16},
		{"TERM dumb", map[string]string{"TERM": "dumb"}, NoColor},
		{"TERM unset", map[string]string{}, unset},
	}
	for _, c := range cases {
		if got := DetectColorProfileEnv(env(c.env)); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestDetectColorProfileReadsProcessEnv(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if DetectColorProfile() != NoColor {
		t.Fatal("DetectColorProfile ignored NO_COLOR")
	}
}

func TestDowngrade(t *testing.T) {
	if got := Downgrade(RGB{255, 0, 0}, TrueColor); got != (RGB{255, 0, 0}) {
		t.Errorf("TrueColor changed RGB: %v", got)
	}
	if got := Downgrade(RGB{255, 0, 0}, ANSI256); got != Color256(196) {
		t.Errorf("RGB red -> 256 = %v, want 196", got)
	}
	if got := Downgrade(RGB{255, 0, 0}, ANSI16); got != BrightRed {
		t.Errorf("RGB red -> 16 = %v, want BrightRed", got)
	}
	if got := Downgrade(Color256(196), ANSI16); got != BrightRed {
		t.Errorf("256 red -> 16 = %v, want BrightRed", got)
	}
	if got := Downgrade(RGB{128, 128, 128}, ANSI256); got != Color256(232+(128-8)*24/247) {
		t.Errorf("gray -> 256 = %v", got)
	}
	if got := Downgrade(Red, ANSI16); got != Red {
		t.Errorf("BasicColor changed: %v", got)
	}
	if Downgrade(Red, NoColor) != nil || Downgrade(nil, ANSI16) != nil {
		t.Error("NoColor/nil should yield nil")
	}
}

func TestQueryBackgroundColor(t *testing.T) {
	if QueryBackgroundColor != "\x1b]11;?\x1b\\" {
		t.Errorf("QueryBackgroundColor = %q", QueryBackgroundColor)
	}
}
