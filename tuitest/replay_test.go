package tuitest_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/termio"
	"github.com/ows4444/tui/tuitest"
)

// replayModel ticks on an Every, collects typed keys and shows its size.
type replayModel struct {
	ticks int
	typed string
	w, h  int
	skew  bool // the model under replay differs from the recorded one
}

type rtick struct{}

func (m replayModel) Init() tui.Cmd {
	c, _ := tui.Every(3*time.Millisecond, func(time.Time) tui.Msg { return rtick{} })
	return c
}

func (m replayModel) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch v := msg.(type) {
	case rtick:
		m.ticks++
	case tui.ResizeMsg:
		m.w, m.h = v.Width, v.Height
	case tui.Key:
		if v.Action != tui.KeyRelease {
			m.typed += v.String()
		}
	}
	return m, nil
}

func (m replayModel) View() string {
	extra := ""
	if m.skew {
		extra = "!"
	}
	return fmt.Sprintf("ticks=%d typed=%q size=%dx%d%s", m.ticks, m.typed, m.w, m.h, extra)
}

type lockedBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *lockedBuf) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *lockedBuf) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}

// record runs a real session with a real clock and returns the cast and sidecar.
func record(t *testing.T) (cast, side string) {
	t.Helper()
	pr, pw := io.Pipe()
	var c, s, out lockedBuf
	term := &termio.Fake{W: 40, H: 5, SizeKnown: true}
	p := tui.NewProgram(replayModel{}, tui.WithAltScreen(true), tui.WithColorProfile(ansi.TrueColor),
		tui.WithInput(pr), tui.WithOutput(&out), tui.WithErrOutput(io.Discard),
		tui.WithTerminal(term), tui.WithRecorder(&c), tui.WithRecorderSidecar(&s))
	done := make(chan struct{})
	go func() { defer close(done); _, _ = p.Run(); _ = pr.Close() }()
	nap := func() { time.Sleep(40 * time.Millisecond) }
	nap()
	_, _ = pw.Write([]byte("a"))
	nap()
	term.Resize(30, 4)
	nap()
	_, _ = pw.Write([]byte("bc"))
	nap()
	_ = pw.Close()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("recorded program did not exit")
	}
	return c.String(), s.String()
}

type replayTB struct{ errs []string }

func (f *replayTB) Helper() {}
func (f *replayTB) Errorf(format string, args ...any) {
	f.errs = append(f.errs, fmt.Sprintf(format, args...))
}
func (f *replayTB) Fatalf(format string, args ...any) {
	f.errs = append(f.errs, fmt.Sprintf(format, args...))
}

func opts() []tui.ProgramOption { return nil }

// Criterion #114: replaying a recorded session reproduces every frame
// byte-for-byte.
func TestReplayReproducesRecording(t *testing.T) {
	cast, side := record(t)
	if !strings.Contains(side, `"k":"tick"`) || !strings.Contains(side, `"k":"in"`) || !strings.Contains(side, `"k":"size"`) {
		t.Fatalf("sidecar lacks ticks, input or sizes:\n%s", side)
	}
	tuitest.Replay(t, func() tui.Model { return replayModel{} }, strings.NewReader(cast), strings.NewReader(side), opts()...)
}

func TestReplayDetectsADifferentModel(t *testing.T) {
	cast, side := record(t)
	ftb := &replayTB{}
	tuitest.Replay(ftb, func() tui.Model { return replayModel{skew: true} }, strings.NewReader(cast), strings.NewReader(side))
	if len(ftb.errs) == 0 || !strings.Contains(ftb.errs[0], "differs") {
		t.Fatalf("a changed model replayed clean: %v", ftb.errs)
	}
}

// Criterion #115: the recording is an asciicast v2 file: a header object, then
// [seconds, "o"|"i", string] events with non-decreasing times.
func TestRecordingIsAsciicastV2(t *testing.T) {
	cast, _ := record(t)
	lines := strings.Split(strings.TrimRight(cast, "\n"), "\n")
	var hdr struct {
		Version       int `json:"version"`
		Width, Height int
		W             int `json:"width"`
		H             int `json:"height"`
		Timestamp     int64
	}
	if err := json.Unmarshal([]byte(lines[0]), &hdr); err != nil || hdr.Version != 2 || hdr.W != 40 || hdr.H != 5 {
		t.Fatalf("bad header %q: %v", lines[0], err)
	}
	prev := 0.0
	outs := 0
	for _, l := range lines[1:] {
		var ev []any
		if err := json.Unmarshal([]byte(l), &ev); err != nil || len(ev) != 3 {
			t.Fatalf("bad event %q: %v", l, err)
		}
		at, ok := ev[0].(float64)
		kind, _ := ev[1].(string)
		_, isStr := ev[2].(string)
		if !ok || at < prev || (kind != "o" && kind != "i") || !isStr {
			t.Fatalf("bad event %q", l)
		}
		prev = at
		if kind == "o" {
			outs++
		}
	}
	if outs == 0 {
		t.Fatal("no output events")
	}
}
