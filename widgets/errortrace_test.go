package widgets

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

func plain(s string) string { return ansi.StripANSI(s) }

func TestErrorTraceShowsEachLayerAndItsType(t *testing.T) {
	cause := &fs.PathError{Op: "open", Path: "/etc/x", Err: errors.New("permission denied")}
	err := fmt.Errorf("load config: %w", fmt.Errorf("read settings: %w", cause))
	got := plain(ErrorTrace(err))
	// PathError.Unwrap gives the inner error, so it has one more level.
	want := "✗ load config (*fmt.wrapError)\n" +
		"└── read settings (*fmt.wrapError)\n" +
		"    └── open /etc/x (*fs.PathError)\n" +
		"        └── permission denied (*errors.errorString)"
	if got != want {
		t.Errorf("ErrorTrace =\n%s\nwant\n%s", got, want)
	}
}

func TestErrorTraceJoinShowsMembersAsSiblings(t *testing.T) {
	err := errors.Join(errors.New("disk full"), fmt.Errorf("sync: %w", errors.New("timeout")))
	got := plain(ErrorTrace(err))
	want := "✗ 2 errors (*errors.joinError)\n" +
		"├── disk full (*errors.errorString)\n" +
		"└── sync (*fmt.wrapError)\n" +
		"    └── timeout (*errors.errorString)"
	if got != want {
		t.Errorf("ErrorTrace =\n%s\nwant\n%s", got, want)
	}
}

func TestErrorTraceNilPlainAndSanitised(t *testing.T) {
	if ErrorTrace(nil) != "" {
		t.Error("a nil error is not empty")
	}
	if got := plain(ErrorTrace(errors.New("boom"))); got != "✗ boom (*errors.errorString)" {
		t.Errorf("plain error = %q", got)
	}
	got := ErrorTrace(errors.New("bad \x1b[31minput\x1b[0m"))
	if strings.Contains(got, "\x1b[31m") {
		t.Errorf("an escape from the message reached the output: %q", got)
	}
}

type selfWrap struct{}

func (e *selfWrap) Error() string { return "loop" }
func (e *selfWrap) Unwrap() error { return e }

func TestErrorTraceStopsOnAChainThatWrapsItself(t *testing.T) {
	got := ErrorTrace(&selfWrap{})
	if n := strings.Count(plain(got), "(*widgets.selfWrap)"); n < 2 || n > maxTraceDepth+2 {
		t.Errorf("%d levels shown for a self-wrapping error", n)
	}
}

func TestErrorTraceASCII(t *testing.T) {
	got := plain(ErrorTraceWith(fmt.Errorf("a: %w", errors.New("b")), theme.DarkTheme().ASCII()))
	if want := "x a (*fmt.wrapError)\n`-- b (*errors.errorString)"; got != want {
		t.Errorf("ASCII trace =\n%s\nwant\n%s", got, want)
	}
}
