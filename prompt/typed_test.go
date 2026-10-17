package prompt

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestIntReturnsANumberAndRefusesTheRest(t *testing.T) {
	o, out := drive("abc\r\x15\r\x1542\r")
	got, err := Int("Count:", o)
	if err != nil || got != 42 {
		t.Fatalf("Int = %d, %v", got, err)
	}
	if n := strings.Count(out.String(), "enter a whole number"); n < 1 {
		t.Errorf("the refusal was never shown: %q", out.String())
	}
	o, _ = drive("-7\r")
	if got, err := Int("Offset:", o); err != nil || got != -7 {
		t.Errorf("negative: %d, %v", got, err)
	}
}

func TestFloatDurationAndDefaults(t *testing.T) {
	o, _ := drive("x\r\x152.5\r")
	if got, err := Float("Ratio:", o); err != nil || got != 2.5 {
		t.Errorf("Float = %v, %v", got, err)
	}
	o, out := drive("soon\r\x151h30m\r")
	if got, err := Duration("Timeout:", o); err != nil || got != 90*time.Minute {
		t.Errorf("Duration = %v, %v", got, err)
	} else if !strings.Contains(out.String(), "such as 90s") {
		t.Errorf("no hint for a bad duration: %q", out.String())
	}
	o, _ = drive("\r")
	o.Default = "8080"
	if got, err := Int("Port:", o); err != nil || got != 8080 {
		t.Errorf("a default answer: %d, %v", got, err)
	}
}

func TestParseGivesTheCallerATypedValueAndRunsValidateAfterIt(t *testing.T) {
	calls := 0
	parse := func(s string) (uint16, error) {
		calls++
		n, err := strconv.ParseUint(s, 10, 16)
		return uint16(n), err
	}
	o, _ := drive("99999\r\x1580\r")
	o.Validate = func(s string) error {
		if s == "80" {
			return errors.New("port 80 is taken")
		}
		return nil
	}
	o2, out := drive("99999\r\x1580\r\x158080\r")
	o2.Validate = o.Validate
	got, err := Parse("Port:", parse, o2)
	if err != nil || got != 8080 {
		t.Fatalf("Parse = %d, %v", got, err)
	}
	if !strings.Contains(out.String(), "port 80 is taken") || !strings.Contains(out.String(), "value out of range") {
		t.Errorf("a refusal was not shown: %q", out.String())
	}
	if calls != 3 {
		t.Errorf("parse ran %d times for three answers, want 3 (the accepted one is not parsed again)", calls)
	}
}

func TestParseCancelAndZeroValue(t *testing.T) {
	o, _ := drive("12\x03")
	got, err := Int("N:", o)
	if !errors.Is(err, ErrCanceled) || got != 0 {
		t.Errorf("Ctrl+C: %d, %v", got, err)
	}
}

func TestPathExpandsHomeCleansAndChecksExistence(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if err := os.WriteFile(filepath.Join(home, "notes.txt"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	o, out := drive("~/missing.txt\r\x15~/notes.txt\r")
	got, err := Path("File:", true, o)
	if err != nil || got != filepath.Join(home, "notes.txt") {
		t.Fatalf("Path = %q, %v", got, err)
	}
	if !strings.Contains(out.String(), "no such file or directory") {
		t.Errorf("a missing file was not refused: %q", out.String())
	}
	o, _ = drive("\r\x15./a/../b//c.txt\r")
	if got, err := Path("Out:", false, o); err != nil || got != filepath.Join("b", "c.txt") {
		t.Errorf("a new path: %q, %v", got, err)
	}
	if expandHome("~user/x") != "~user/x" || expandHome("a~/b") != "a~/b" {
		t.Error("only a leading ~ or ~/ is the home directory")
	}
	if expandHome("~") != home {
		t.Errorf("~ = %q", expandHome("~"))
	}
}

func TestParseTrimsTheAnswerBeforeParsing(t *testing.T) {
	var seen []string
	o, _ := drive("  42  \r")
	got, err := Parse("N:", func(s string) (int, error) {
		seen = append(seen, s)
		return strconv.Atoi(s)
	}, o)
	if err != nil || got != 42 {
		t.Fatalf("Parse = %d, %v", got, err)
	}
	for _, s := range seen {
		if s != "42" {
			t.Errorf("parse was given %q, want the trimmed answer", s)
		}
	}
}
