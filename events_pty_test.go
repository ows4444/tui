//go:build darwin || linux

package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/ptytest"
)

// styledPTYModel records Msgs like ptyModel but draws a truecolor-styled
// view, so a test can see what colour profile actually reached the pty.
type styledPTYModel struct{ ptyModel }

func (m styledPTYModel) Update(msg Msg) (Model, Cmd) {
	next, cmd := m.ptyModel.Update(msg)
	return styledPTYModel{next.(ptyModel)}, cmd
}
func (m styledPTYModel) View() string {
	return "\x1b[1;38;2;255;0;0mhello\x1b[0m"
}

func msgsOf(res runResult) []Msg {
	switch m := res.model.(type) {
	case ptyModel:
		return m.msgs
	case styledPTYModel:
		return m.msgs
	}
	return nil
}

// TestRunFocusAndBackgroundDetectionOnAPty runs Program.Run on a real pty
// with focus reporting and background detection on, plays the terminal's
// part (a focus-in report, an OSC 11 answer), and checks the sequences the
// program wrote and the events it delivered.
func TestRunFocusAndBackgroundDetectionOnAPty(t *testing.T) {
	s := startPTY(t, ptyModel{}, 80, 24, WithFocusReporting(true), WithBackgroundDetection(time.Hour))

	s.waitFor(ansi.FocusReportingEnable)
	s.waitFor(ansi.QueryBackgroundColor)
	if n := strings.Count(s.out.String(), ansi.QueryBackgroundColor); n != 1 {
		t.Errorf("OSC 11 query written %d times, want 1", n)
	}

	s.send("\x1b[I")                         // focus in
	s.send("\x1b]11;rgb:1e1e/1e1e/1e1e\x07") // terminal's background answer
	s.send("\x1b[O")                         // focus out
	s.waitFor("frame:4")                     // resize + 3 events
	s.send("q")
	res := s.finish()
	if res.err != nil {
		t.Fatalf("Run error: %v", res.err)
	}

	var got []string
	for _, m := range msgsOf(res) {
		switch v := m.(type) {
		case FocusEvent:
			if v.Focused {
				got = append(got, "focus-in")
			} else {
				got = append(got, "focus-out")
			}
		case BackgroundColorEvent:
			if v != (BackgroundColorEvent{R: 30, G: 30, B: 30}) {
				t.Errorf("background = %+v, want {30 30 30}", v)
			}
			got = append(got, "background")
		}
	}
	if want := "focus-in,background,focus-out"; strings.Join(got, ",") != want {
		t.Errorf("events = %v, want %s", got, want)
	}

	out := s.out.String()
	if !strings.Contains(out, ansi.FocusReportingDisable) {
		t.Error("focus reporting was not disabled when Run returned")
	}
	if strings.LastIndex(out, ansi.FocusReportingDisable) < strings.LastIndex(out, ansi.FocusReportingEnable) {
		t.Error("focus reporting disable should come after enable")
	}
	after, err := ptytest.Termios(s.slave)
	if err != nil || after != s.before {
		t.Errorf("terminal not restored: err=%v after=%+v before=%+v", err, after, s.before)
	}
}

func TestRunBackgroundDetectionTimesOutOnAPty(t *testing.T) {
	s := startPTY(t, ptyModel{}, 80, 24, WithBackgroundDetection(80*time.Millisecond))
	s.waitFor(ansi.QueryBackgroundColor)
	s.waitFor("frame:2") // resize, then BackgroundUnknownMsg after the timeout
	s.send("q")
	res := s.finish()
	if res.err != nil {
		t.Fatalf("Run error: %v", res.err)
	}
	var unknown int
	for _, m := range msgsOf(res) {
		if _, ok := m.(BackgroundUnknownMsg); ok {
			unknown++
		}
	}
	if unknown != 1 {
		t.Errorf("got %d BackgroundUnknownMsg, want exactly 1", unknown)
	}
}

func TestRunWithoutEventOptionsWritesNoQueryOrFocusSequence(t *testing.T) {
	s := startPTY(t, ptyModel{}, 80, 24)
	s.waitFor("frame:1")
	s.send("q")
	if res := s.finish(); res.err != nil {
		t.Fatal(res.err)
	}
	out := s.out.String()
	if strings.Contains(out, "?1004") || strings.Contains(out, "]11;?") {
		t.Errorf("unrequested sequence written: %q", out)
	}
}

func TestRunColorProfileReachesThePty(t *testing.T) {
	cases := []struct {
		profile ansi.Profile
		want    string
		absent  string
	}{
		{ansi.TrueColor, "\x1b[1;38;2;255;0;0mhello", ""},
		{ansi.ANSI256, "\x1b[1;38;5;196mhello", "38;2"},
		{ansi.ANSI16, "\x1b[1;91mhello", "38;"},
		{ansi.NoColor, "\x1b[1mhello", "38;"},
	}
	for _, c := range cases {
		s := startPTY(t, styledPTYModel{}, 80, 24, WithColorProfile(c.profile))
		s.waitFor("hello")
		s.send("q")
		if res := s.finish(); res.err != nil {
			t.Fatal(res.err)
		}
		out := s.out.String()
		if !strings.Contains(out, c.want) {
			t.Errorf("profile %v: output %q lacks %q", c.profile, out, c.want)
		}
		if c.absent != "" && strings.Contains(out, c.absent) {
			t.Errorf("profile %v: output %q still contains %q", c.profile, out, c.absent)
		}
	}
}

func TestRunChordsOnAPty(t *testing.T) {
	s := startPTY(t, ptyModel{}, 80, 24, WithChords(ChordDef{Name: "top", Keys: []string{"g", "g"}}))
	s.waitFor("frame:1")
	s.send("g")
	s.send("g")
	s.waitFor("frame:2") // resize + one ChordMsg, not two Keys
	s.send("q")
	res := s.finish()
	if res.err != nil {
		t.Fatal(res.err)
	}
	var got []string
	for _, m := range msgsOf(res) {
		switch v := m.(type) {
		case ChordMsg:
			got = append(got, "chord:"+v.Name)
		case Key:
			got = append(got, v.String())
		}
	}
	if strings.Join(got, ",") != "chord:top,q" {
		t.Errorf("messages = %v, want [chord:top q]", got)
	}
}
