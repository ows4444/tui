package tuitest_test

import (
	"encoding/base64"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/tuitest"
)

// mutateSidecar returns side with fn applied to each line; fn returns the lines
// to put in its place (none drops the line).
func mutateSidecar(side string, fn func(line string) []string) string {
	var out []string
	for _, l := range strings.Split(strings.TrimRight(side, "\n"), "\n") {
		out = append(out, fn(l)...)
	}
	return strings.Join(out, "\n") + "\n"
}

func replayFails(t *testing.T, cast, side, want string) {
	t.Helper()
	ftb := &replayTB{}
	var c *strings.Reader
	if cast != "" {
		c = strings.NewReader(cast)
	}
	if c == nil {
		tuitest.Replay(ftb, func() tui.Model { return replayModel{} }, nil, strings.NewReader(side))
	} else {
		tuitest.Replay(ftb, func() tui.Model { return replayModel{} }, c, strings.NewReader(side))
	}
	if len(ftb.errs) == 0 || !strings.Contains(strings.Join(ftb.errs, "\n"), want) {
		t.Fatalf("want a failure containing %q, got %q", want, ftb.errs)
	}
}

func TestReplayRejectsABadSidecar(t *testing.T) {
	cast, _ := record(t)
	for name, tc := range map[string]struct{ side, want string }{
		"not json":   {"{nope\n", "bad line"},
		"version":    {`{"k":"hdr","v":2,"w":10,"h":3}` + "\n", "not supported"},
		"no header":  {`{"k":"upd"}` + "\n", "no header"},
		"zero size":  {`{"k":"hdr","v":1,"w":0,"h":0}` + "\n", "no header"},
		"empty file": {"", "no header"},
	} {
		t.Run(name, func(t *testing.T) { replayFails(t, cast, tc.side, tc.want) })
	}
}

func TestReplayRejectsABadCast(t *testing.T) {
	_, side := record(t)
	for name, tc := range map[string]struct{ cast, want string }{
		"not asciicast": {"{\"version\":1}\n", "not an asciicast v2"},
		"bad header":    {"garbage\n", "not an asciicast v2"},
		"bad event":     {"{\"version\":2}\n[1,2]\n", "bad event"},
		"bad kinds":     {"{\"version\":2}\n[1,2,3]\n", "bad event"},
		"empty":         {"\n\n", "empty file"},
		"count":         {"{\"version\":2}\n[0.1,\"o\",\"x\"]\n", "not from one recording"},
	} {
		t.Run(name, func(t *testing.T) { replayFails(t, tc.cast, side, tc.want) })
	}
}

func TestReplayWithoutACastChecksTheSidecarHashes(t *testing.T) {
	_, side := record(t)
	tuitest.Replay(t, func() tui.Model { return replayModel{} }, nil, strings.NewReader(side))
	replayFailsNoCast := func(m replayModel) []string {
		ftb := &replayTB{}
		tuitest.Replay(ftb, func() tui.Model { return m }, nil, strings.NewReader(side))
		return ftb.errs
	}
	if errs := replayFailsNoCast(replayModel{skew: true}); len(errs) == 0 || !strings.Contains(errs[0], "sha256") {
		t.Errorf("a changed model must fail on the hash, got %q", errs)
	}
}

func TestReplayNoticesADifferentFrameAgainstTheCast(t *testing.T) {
	cast, side := record(t)
	// Change one frame of the cast but not the sidecar: the sidecar's hash
	// still matches the replay, so the cast comparison catches it.
	edited := strings.Replace(cast, "ticks=0", "ticks=9", 1)
	if edited == cast {
		t.Skip("the recording has no ticks=0 frame")
	}
	replayFails(t, edited, side, "differs")
}

func TestReplayReportsAMissingOutputAndAnUnknownTicker(t *testing.T) {
	cast, side := record(t)
	lines := strings.Split(strings.TrimRight(side, "\n"), "\n")
	last := -1
	for i, l := range lines {
		if strings.Contains(l, `"k":"out"`) {
			last = i
		}
	}
	fewer := strings.Join(append(lines[:last:last], lines[last+1:]...), "\n") + "\n"
	replayFails(t, "", fewer, "writes to the terminal")

	badTick := mutateSidecar(side, func(l string) []string {
		if strings.Contains(l, `"k":"tick"`) {
			return []string{`{"k":"tick","id":99,"upd":0}`}
		}
		return []string{l}
	})
	replayFails(t, cast, badTick, "no such ticker")
}

func TestReplayReportsBadInputBytes(t *testing.T) {
	_, side := record(t)
	bad := mutateSidecar(side, func(l string) []string {
		if strings.Contains(l, `"k":"in"`) {
			return []string{`{"k":"in","b":"!!not base64!!"}`}
		}
		return []string{l}
	})
	replayFails(t, "", bad, "bad input bytes")
	if _, err := base64.StdEncoding.DecodeString("!!not base64!!"); err == nil {
		t.Fatal("test premise: the bytes must be invalid base64")
	}
}

func TestReplayStallsWhenTheRecordingHadMoreUpdates(t *testing.T) {
	if testing.Short() {
		t.Skip("waits for the replay timeout")
	}
	_, side := record(t)
	// One more update than the model can ever make, before its last step.
	extra := mutateSidecar(side, func(l string) []string {
		if strings.Contains(l, `"k":"hdr"`) {
			return []string{l, strings.Repeat(`{"k":"upd"}`+"\n", 400) + `{"k":"upd"}`}
		}
		return []string{l}
	})
	start := time.Now()
	replayFails(t, "", extra, "stalled")
	if time.Since(start) > 30*time.Second {
		t.Error("the stall took too long to report")
	}
}

func TestSessionKeysAfterCloseAndPasteAfterCloseDoNotHang(t *testing.T) {
	s := tuitest.New(echo{}, 30, 6)
	s.Close()
	s.Close() // idempotent
	s.Keys("a")
	s.Paste("x")
	s.Send(customMsg("y"))
	if !s.Done() {
		t.Error("a closed session must be done")
	}
}

func TestKeysEncodeAltCtrlAndLiteralText(t *testing.T) {
	s := newSession(t)
	s.Keys("alt+x", "ctrl+a", "ctrl+", "hello")
	if got := s.Screen()[0]; !strings.HasPrefix(got, "typed:") {
		t.Fatalf("screen = %q", got)
	}
}

func TestGoldenUpdateEnvWritesIntoANewDirectory(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("TUITEST_UPDATE", "1")
	s := newSession(t)
	s.Keys("z")
	ftb := &replayTB{}
	s.Golden(ftb, "fresh")
	if len(ftb.errs) != 0 {
		t.Fatalf("update failed: %q", ftb.errs)
	}
	if b, err := os.ReadFile("testdata/fresh.golden"); err != nil || !strings.HasPrefix(string(b), "typed:z") {
		t.Fatalf("golden = %q, %v", b, err)
	}
}

func TestAdvanceWithoutAFakeClockDoesNothing(t *testing.T) {
	s := newSession(t)
	s.Advance(time.Hour) // no FakeClock: returns at once
}
