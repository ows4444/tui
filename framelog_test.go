package tui

import (
	"bytes"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// frameLine is one parsed line of the frame log.
type frameLine struct {
	n, t                     int
	kind                     string
	rows, changed, bytes, us int
}

var frameLineRE = regexp.MustCompile(`^n=(\d+) t=(\d+) kind=(diff|full|accessible) rows=(\d+) changed=(\d+) bytes=(\d+) us=(\d+)$`)

func parseFrameLog(t *testing.T, log string) []frameLine {
	t.Helper()
	var out []frameLine
	if log == "" {
		return out
	}
	for _, line := range strings.Split(strings.TrimSuffix(log, "\n"), "\n") {
		m := frameLineRE.FindStringSubmatch(line)
		if m == nil {
			t.Fatalf("frame log line %q does not match the documented format", line)
		}
		atoi := func(s string) int { v, _ := strconv.Atoi(s); return v }
		out = append(out, frameLine{atoi(m[1]), atoi(m[2]), m[3], atoi(m[4]), atoi(m[5]), atoi(m[6]), atoi(m[7])})
	}
	return out
}

// logProgram is a Program drawing model into out (a plain writer, so 80x24),
// logging frames to log.
func logProgram(m Model, out, log *bytes.Buffer) *Program {
	return NewProgram(m, WithOutput(out), WithFrameLog(log))
}

// The log has exactly one line per rendered frame, with the fields in order.
func TestFrameLogWritesOneLinePerFrame(t *testing.T) {
	var out, log bytes.Buffer
	p := logProgram(staticModel{view: "one\ntwo\nthree"}, &out, &log)
	for i := 0; i < 4; i++ {
		p.render()
	}
	lines := parseFrameLog(t, log.String())
	if len(lines) != 4 {
		t.Fatalf("%d log lines for 4 frames: %q", len(lines), log.String())
	}
	for i, l := range lines {
		if l.n != i+1 {
			t.Errorf("line %d has n=%d, want %d", i, l.n, i+1)
		}
		if l.rows != 3 {
			t.Errorf("line %d has rows=%d, want 3", i, l.rows)
		}
		if l.bytes <= 0 && l.changed > 0 {
			t.Errorf("line %d changed %d rows but reports bytes=%d", i, l.changed, l.bytes)
		}
	}
}

// bytes is what the frame wrote to the terminal, so the values add up to the
// output.
func TestFrameLogBytesAddUpToTheTerminalOutput(t *testing.T) {
	var out, log bytes.Buffer
	m := &swapModel{views: []string{"a\nb", "a\nc", "x\ny\nz"}}
	p := logProgram(m, &out, &log)
	for range m.views {
		p.render()
		m.next()
	}
	total := 0
	for _, l := range parseFrameLog(t, log.String()) {
		total += l.bytes
	}
	if total != out.Len() {
		t.Errorf("logged bytes sum to %d, the terminal received %d", total, out.Len())
	}
}

// swapModel shows views[i], advancing when next is called.
type swapModel struct {
	views []string
	i     int
}

func (m *swapModel) Init() Cmd               { return nil }
func (m *swapModel) Update(Msg) (Model, Cmd) { return m, nil }
func (m *swapModel) View() string            { return m.views[m.i] }
func (m *swapModel) next()                   { m.i = (m.i + 1) % len(m.views) }

// changed is the number of rows rewritten; an unchanged frame reports 0.
func TestFrameLogChangedCountsRewrittenRows(t *testing.T) {
	var out, log bytes.Buffer
	m := &swapModel{views: []string{
		"r0\nr1\nr2\nr3\nr4",      // first frame: all 5 written
		"r0\nr1\nr2\nr3\nr4",      // identical: 0
		"r0\nX1\nr2\nX3\nr4",      // two rows differ: 2
		"r0\nX1",                  // shrinks: r2..r4 are cleared, 3 rewritten
		"r0\nX1\nnew\nnew2\nnew3", // grows: 3 new rows
	}}
	p := logProgram(m, &out, &log)
	want := []struct{ rows, changed int }{{5, 5}, {5, 0}, {5, 2}, {5, 3}, {5, 3}}
	for range m.views {
		p.render()
		m.next()
	}
	lines := parseFrameLog(t, log.String())
	if len(lines) != len(want) {
		t.Fatalf("%d lines, want %d", len(lines), len(want))
	}
	for i, w := range want {
		if lines[i].rows != w.rows || lines[i].changed != w.changed {
			t.Errorf("frame %d: rows=%d changed=%d, want rows=%d changed=%d", i+1, lines[i].rows, lines[i].changed, w.rows, w.changed)
		}
	}
}

// The first frame, a repaint after a resize and the frame after committed text
// (which resets the diff, as Suspend does) are kind=full; the rest are diffs.
func TestFrameLogMarksFullRepaints(t *testing.T) {
	var out, log bytes.Buffer
	p := logProgram(staticModel{view: "a\nb"}, &out, &log)
	p.render()       // first frame
	p.render()       // diff
	p.repaintFresh() // resize
	p.render()       // diff again
	p.printLines("committed", &out)
	p.render() // no previous frame after a commit: full
	var kinds []string
	for _, l := range parseFrameLog(t, log.String()) {
		kinds = append(kinds, l.kind)
	}
	want := []string{"full", "diff", "full", "diff", "full"}
	if strings.Join(kinds, ",") != strings.Join(want, ",") {
		t.Errorf("kinds = %v, want %v", kinds, want)
	}
}

// Through the real event loop: a ResizeMsg makes the next frame a full repaint.
func TestFrameLogMarksTheFrameAfterAResizeAsFull(t *testing.T) {
	pr, pw := mustPipe(t)
	defer pr.Close()
	defer pw.Close()
	var out, log bytes.Buffer
	p := NewProgram(printModel{view: "v", quitAfter: 3}, WithInput(pr), WithOutput(&out), WithFrameLog(&log))
	done := make(chan struct{})
	go func() { p.runLoop(); close(done) }()
	p.msgs <- Key{Type: KeyRunes, Text: "a", Code: 'a'}
	p.msgs <- ResizeMsg{Width: 100, Height: 30}
	p.msgs <- Key{Type: KeyRunes, Text: "b", Code: 'b'}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("runLoop did not return")
	}
	lines := parseFrameLog(t, log.String())
	full := 0
	for _, l := range lines {
		if l.kind == "full" {
			full++
		}
	}
	if full < 2 {
		t.Errorf("want at least two full frames (the first and the one after the resize), got %d in %+v", full, lines)
	}
}

// Logging changes nothing on the terminal.
func TestFrameLogDoesNotChangeTheTerminalOutput(t *testing.T) {
	run := func(withLog bool) string {
		var out, log bytes.Buffer
		m := &swapModel{views: []string{"a\nb", "a\nc", "x", "x\ny\nz"}}
		opts := []ProgramOption{WithOutput(&out)}
		if withLog {
			opts = append(opts, WithFrameLog(&log))
		}
		p := NewProgram(m, opts...)
		for range m.views {
			p.render()
			m.next()
		}
		p.repaintFresh()
		p.printLines("text", &out)
		p.render()
		return out.String()
	}
	if without, with := run(false), run(true); without != with {
		t.Errorf("terminal output differs with a frame log:\nwithout %q\nwith    %q", without, with)
	}
}

type failingWriter struct{ calls int }

func (f *failingWriter) Write([]byte) (int, error) { f.calls++; return 0, errors.New("disk full") }

// A log that cannot be written must not stop or alter rendering.
func TestFrameLogWriteErrorIsIgnored(t *testing.T) {
	var out, ref bytes.Buffer
	bad := &failingWriter{}
	m := &swapModel{views: []string{"a", "b", "c"}}
	p := NewProgram(m, WithOutput(&out), WithFrameLog(bad))
	q := NewProgram(&swapModel{views: []string{"a", "b", "c"}}, WithOutput(&ref))
	for i := 0; i < 3; i++ {
		p.render()
		q.render()
		m.next()
		q.model.(*swapModel).next()
	}
	if bad.calls != 3 {
		t.Errorf("the failing log was written %d times, want 3", bad.calls)
	}
	if out.String() != ref.String() {
		t.Error("a failing frame log changed the terminal output")
	}
}

// Without the option nothing is logged and no frame state is kept.
func TestNoFrameLogByDefault(t *testing.T) {
	var out bytes.Buffer
	p := NewProgram(staticModel{view: "x"}, WithOutput(&out))
	p.render()
	if p.frameN != 0 {
		t.Errorf("frameN = %d without WithFrameLog, want 0", p.frameN)
	}
}

// The frame after a Suspend returns is a full repaint: Suspend resets the diff
// state, so there is nothing to diff against.
func TestFrameLogMarksTheFrameAfterSuspendAsFull(t *testing.T) {
	out, _ := captureOutput(t)
	var log bytes.Buffer
	p := NewProgram(staticModel{view: "v"}, WithOutput(out), WithFrameLog(&log))
	p.render() // first frame: full
	p.render() // diff
	if err := p.suspend(func() error { return nil }); err != nil {
		t.Fatalf("suspend: %v", err)
	}
	p.render() // after the suspend: full
	var kinds []string
	for _, l := range parseFrameLog(t, log.String()) {
		kinds = append(kinds, l.kind)
	}
	if got, want := strings.Join(kinds, ","), "full,diff,full"; got != want {
		t.Errorf("kinds = %s, want %s", got, want)
	}
}

// In accessible mode each transcript write is one log line with
// kind=accessible; changed is the number of lines appended.
func TestFrameLogAccessibleTranscriptWrites(t *testing.T) {
	var out, log bytes.Buffer
	m := &swapModel{views: []string{"one\ntwo", "one\ntwo", "one\nTWO\nthree"}}
	p := NewProgram(m, WithAccessible(true), WithOutput(&out), WithFrameLog(&log))
	for range m.views {
		p.render()
		m.next()
	}
	lines := parseFrameLog(t, log.String())
	want := []struct{ rows, changed int }{{2, 2}, {2, 0}, {3, 2}} // first: both lines; repeat: none; then TWO and three
	if len(lines) != len(want) {
		t.Fatalf("%d log lines for %d transcript writes: %q", len(lines), len(want), log.String())
	}
	total := 0
	for i, w := range want {
		l := lines[i]
		if l.kind != "accessible" || l.rows != w.rows || l.changed != w.changed {
			t.Errorf("frame %d = %+v, want kind=accessible rows=%d changed=%d", i+1, l, w.rows, w.changed)
		}
		total += l.bytes
	}
	if total != out.Len() {
		t.Errorf("logged bytes sum to %d, the transcript is %d bytes", total, out.Len())
	}
}

// Nothing the log writes reaches the terminal output, and the log holds only
// frame lines: with separate writers, neither stream contains the other.
func TestFrameLogNeverReachesTheTerminal(t *testing.T) {
	var out, log bytes.Buffer
	p := logProgram(&swapModel{views: []string{"hello\nworld"}}, &out, &log)
	p.render()
	p.repaintFresh()
	if strings.Contains(out.String(), "kind=") || strings.Contains(out.String(), "n=1") {
		t.Errorf("the terminal output contains frame-log text: %q", out.String())
	}
	if strings.Contains(log.String(), "hello") || strings.Contains(log.String(), "\x1b") {
		t.Errorf("the frame log contains screen content or escape codes: %q", log.String())
	}
}

// Every line ends in a newline and the log is plain text, so it can be tailed.
func TestFrameLogIsPlainLines(t *testing.T) {
	var out, log bytes.Buffer
	p := logProgram(staticModel{view: "x"}, &out, &log)
	p.render()
	p.render()
	if s := log.String(); !strings.HasSuffix(s, "\n") || strings.Count(s, "\n") != 2 {
		t.Errorf("log = %q, want two newline-terminated lines", s)
	}
}
