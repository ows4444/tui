package tuitest

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/termio"
)

// replayWait bounds each wait for the model to catch up with the recording. It
// only fires when the replay has diverged or stalled; a healthy replay never
// waits on it.
const replayWait = 5 * time.Second

// sidecarLine mirrors one line of the sidecar WithRecorderSidecar writes.
type sidecarLine struct {
	K   string `json:"k"`
	V   int    `json:"v"`
	W   int    `json:"w"`
	H   int    `json:"h"`
	B   string `json:"b"`
	ID  *int   `json:"id"`
	D   int64  `json:"d"`
	N   int    `json:"n"`
	Sha string `json:"sha"`
}

// observer is the sidecar writer of the replayed Program: it counts what the
// replay has done so far so the driver can wait for it.
type observer struct {
	mu   sync.Mutex
	shas []string
	upd  atomic.Int64
	out  atomic.Int64
}

func (o *observer) Write(b []byte) (int, error) {
	var l sidecarLine
	if json.Unmarshal(b, &l) != nil {
		return len(b), nil
	}
	switch l.K {
	case "upd":
		o.upd.Add(1)
	case "out":
		o.mu.Lock()
		o.shas = append(o.shas, l.Sha)
		o.mu.Unlock()
		o.out.Add(1)
	}
	return len(b), nil
}

// capture keeps a copy of each write to the replayed terminal.
type capture struct {
	mu     sync.Mutex
	frames []string
}

func (c *capture) Write(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil // the recorder does not count empty writes either
	}
	c.mu.Lock()
	c.frames = append(c.frames, string(b))
	c.mu.Unlock()
	return len(b), nil
}

// step is one thing the replay drives, with how far the model must already
// have got before it is driven.
type step struct {
	line     sidecarLine
	upd, out int64
}

// Replay re-runs a recording made with tui.WithRecorder and
// tui.WithRecorderSidecar against newModel and fails t if any write to the
// terminal differs from the recording, byte for byte. cast is the asciicast
// file (nil to check only against the sidecar's hashes) and sidecar the file
// written by WithRecorderSidecar.
//
// The replay feeds the recorded input bytes, terminal sizes and Every ticks
// in their recorded order, waiting for the model to catch up at each step, so
// it does not depend on the wall clock: Every runs on a FakeClock that fires
// exactly the recorded ticks. opts must be the options the recorded Program had
// (alt screen, colour profile, accessibility, and so on), except for the
// recorder options; the harness sets a true-colour profile, which opts may
// override. newModel must return a model in the state the recording began with.
// Replay feeds each recorded input read whole, so it cannot reproduce a
// recording in which a tick or resize was handled between two keys that
// arrived in one read; such a recording fails with a note saying so.
// Messages from Program.Send and Cmd results that depend on the outside world
// are not recorded, so a model that relies on them does not replay.
func Replay(t TB, newModel func() tui.Model, cast, sidecar io.Reader, opts ...tui.ProgramOption) {
	t.Helper()
	hdr, steps, wantShas, err := readSidecar(sidecar)
	if err != nil {
		t.Fatalf("tuitest: replay: reading sidecar: %v", err)
		return
	}
	var wantFrames []string
	if cast != nil {
		wantFrames, err = readCast(cast)
		if err != nil {
			t.Fatalf("tuitest: replay: reading cast: %v", err)
			return
		}
		if len(wantFrames) != len(wantShas) {
			t.Fatalf("tuitest: replay: the cast has %d output events but the sidecar %d: they are not from one recording", len(wantFrames), len(wantShas))
			return
		}
	}
	r := startReplay(t, newModel, hdr, steps, opts)
	if r.drive() {
		r.finish(wantFrames, wantShas)
	}
}

// replayRun is a Program being driven through a recording.
type replayRun struct {
	t     TB
	steps []step
	pw    *io.PipeWriter
	done  chan struct{}
	term  *termio.Fake
	clock *FakeClock
	obs   *observer
	out   *capture
	// hint is added to a failure when the recording cannot be driven in order.
	hint    string
	pending [][]byte // chunks read in the recording, not yet fed
}

// startReplay starts newModel() in a Program fed by a pipe, on a fake clock and
// terminal, and watched by an observer of its updates and writes.
func startReplay(t TB, newModel func() tui.Model, hdr sidecarLine, steps []step, opts []tui.ProgramOption) *replayRun {
	pr, pw := io.Pipe()
	r := &replayRun{
		t: t, steps: steps, pw: pw, done: make(chan struct{}),
		clock: NewFakeClock(), term: &termio.Fake{W: hdr.W, H: hdr.H, SizeKnown: true},
		obs: &observer{}, out: &capture{},
	}
	all := append([]tui.ProgramOption{
		tui.WithAltScreen(true),
		tui.WithColorProfile(ansi.TrueColor),
	}, opts...)
	all = append(all,
		tui.WithInput(pr),
		tui.WithOutput(r.out),
		tui.WithErrOutput(io.Discard),
		tui.WithTerminal(r.term),
		tui.WithClock(r.clock),
		tui.WithRecorderSidecar(r.obs),
	)
	prog := tui.NewProgram(newModel(), all...)
	go func() {
		defer close(r.done)
		_, _ = prog.Run()
		_ = pr.Close()
	}()
	// Replay feeds each recorded read whole. A recording in which a tick or a
	// resize was handled between two messages of one read cannot be driven in
	// that order, so a failure of such a recording says so.
	if i := interleavedRead(steps); i >= 0 {
		r.hint = fmt.Sprintf("\n  note: at step %d the recording handled a tick or resize between two messages that came from one input read. "+
			"Replay feeds a read whole and cannot reproduce that order; record again, or send those keys in separate writes", i)
	}
	return r
}

func (r *replayRun) fail(format string, args ...any) {
	_ = r.pw.Close()
	r.t.Fatalf("tuitest: replay: "+format, args...)
}

// waitFor waits until the program has made upd updates and out writes.
func (r *replayRun) waitFor(upd, out int64) bool {
	deadline := time.Now().Add(replayWait)
	for r.obs.upd.Load() < upd || r.obs.out.Load() < out {
		select {
		case <-r.done:
			return r.obs.upd.Load() >= upd && r.obs.out.Load() >= out
		default:
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(50 * time.Microsecond)
	}
	return true
}

// flush feeds the input read so far.
func (r *replayRun) flush(i int) bool {
	for _, b := range r.pending {
		if _, err := r.pw.Write(b); err != nil {
			r.fail("the program stopped reading input at step %d", i)
			return false
		}
	}
	r.pending = nil
	return true
}

func (r *replayRun) stalled(i int, st step, wantUpd int64) {
	select {
	case <-r.done:
		r.fail("the program exited at step %d (%s) before the recording did", i, st.line.K)
	default:
		r.fail("stalled before step %d (%s): the model made %d updates and %d writes, the recording %d and %d%s",
			i, st.line.K, r.obs.upd.Load(), r.obs.out.Load(), wantUpd, st.out, r.hint)
	}
}

// drive plays the recorded steps, in order, and ends the input. It reports
// false after a failure.
func (r *replayRun) drive() bool {
	skipUpd := false // the next upd belongs to a tick or size just driven
	for i, st := range r.steps {
		switch st.line.K {
		case "in":
			b, err := base64.StdEncoding.DecodeString(st.line.B)
			if err != nil {
				r.fail("step %d: bad input bytes: %v", i, err)
				return false
			}
			r.pending = append(r.pending, b)
		case "upd":
			if skipUpd {
				skipUpd = false
				continue
			}
			// A message the loop took that is not a tick or size: it came from
			// input read before it, so feed that now.
			if !r.flush(i) {
				return false
			}
			if !r.waitFor(st.upd+1, st.out) {
				r.stalled(i, st, st.upd+1)
				return false
			}
		case "size", "tick":
			if !r.tickOrSize(i, st) {
				return false
			}
			skipUpd = true
		}
	}
	if !r.flush(len(r.steps)) {
		return false
	}
	// Let the last event finish, then end the input: EOF quits the program.
	r.waitFor(totalUpd(r.steps), 0)
	_ = r.pw.Close()
	select {
	case <-r.done:
		return true
	case <-time.After(replayWait):
		r.t.Fatalf("tuitest: replay: the program did not exit after the recorded input ended")
		return false
	}
}

// tickOrSize drives a recorded resize or Every tick and waits for the loop to
// take it.
func (r *replayRun) tickOrSize(i int, st step) bool {
	if !r.waitFor(st.upd, st.out) {
		r.stalled(i, st, st.upd)
		return false
	}
	if st.line.K == "size" {
		r.term.Resize(st.line.W, st.line.H)
	} else if st.line.ID == nil || !r.clock.fire(*st.line.ID, r.done) {
		r.fail("step %d: the recording ticked Every #%v but the replayed model has no such ticker", i, st.line.ID)
		return false
	}
	// Let the loop take it before anything else is fed, so the two cannot swap
	// places on the way in.
	if !r.waitFor(st.upd+1, st.out) {
		r.stalled(i, st, st.upd+1)
		return false
	}
	return true
}

// finish compares what the replay wrote with the recording.
func (r *replayRun) finish(wantFrames, wantShas []string) {
	r.out.mu.Lock()
	got := append([]string(nil), r.out.frames...)
	r.out.mu.Unlock()
	for i := 0; i < len(got) && i < len(wantShas); i++ {
		sum := sha256.Sum256([]byte(got[i]))
		if hex.EncodeToString(sum[:]) != wantShas[i] {
			want := "(sha256 " + wantShas[i][:12] + ")"
			if wantFrames != nil {
				want = fmt.Sprintf("%q", clip(wantFrames[i]))
			}
			r.t.Errorf("tuitest: replay: write %d differs from the recording:\n  want %s\n  got  %q%s", i, want, clip(got[i]), r.hint)
			return
		}
		if wantFrames != nil && string([]rune(got[i])) != wantFrames[i] {
			r.t.Errorf("tuitest: replay: write %d differs from the cast:\n  want %q\n  got  %q%s", i, clip(wantFrames[i]), clip(got[i]), r.hint)
			return
		}
	}
	if len(got) != len(wantShas) {
		r.t.Errorf("tuitest: replay: the replay made %d writes to the terminal, the recording %d%s", len(got), len(wantShas), r.hint)
	}
}

// interleavedRead returns the index of the first step where the recording
// handled a message that came from input already read, after a tick or resize
// that followed that read, or -1. It mirrors the driver loop: an "upd" that is
// not the tick's or resize's own, with no new input recorded since the last
// one, comes from the read fed before the tick.
func interleavedRead(steps []step) int {
	fed, ticked, skip, pending := false, false, false, 0
	for i, st := range steps {
		switch st.line.K {
		case "in":
			pending++
		case "upd":
			switch {
			case skip:
				skip = false
			case pending > 0:
				pending, fed, ticked = 0, true, false
			case fed && ticked:
				return i
			}
		case "size", "tick":
			skip, ticked = true, true
		}
	}
	return -1
}

func totalUpd(steps []step) int64 {
	var n int64
	for _, s := range steps {
		if s.line.K == "upd" {
			n = s.upd + 1
		}
	}
	return n
}

func clip(s string) string {
	const limit = 200
	if len(s) > limit {
		return s[:limit] + "..."
	}
	return s
}

// readSidecar parses the sidecar into the steps to drive, with the counts of
// updates and writes the recording had made before each, and the SHA-256 of
// every write.
func readSidecar(r io.Reader) (hdr sidecarLine, steps []step, shas []string, err error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64<<10), 64<<20)
	var upd, out int64
	for sc.Scan() {
		var l sidecarLine
		if err := json.Unmarshal(sc.Bytes(), &l); err != nil {
			return hdr, nil, nil, fmt.Errorf("bad line %q: %w", clip(sc.Text()), err)
		}
		switch l.K {
		case "hdr":
			if l.V != 1 {
				return hdr, nil, nil, fmt.Errorf("sidecar version %d is not supported", l.V)
			}
			hdr = l
		case "out":
			out++
			shas = append(shas, l.Sha)
		case "in", "size", "tick", "upd":
			steps = append(steps, step{line: l, upd: upd, out: out})
			if l.K == "upd" {
				upd++
			}
		}
	}
	if err := sc.Err(); err != nil {
		return hdr, nil, nil, err
	}
	if hdr.K != "hdr" || hdr.W <= 0 || hdr.H <= 0 {
		return hdr, nil, nil, fmt.Errorf("no header line")
	}
	return hdr, steps, shas, nil
}

// readCast returns the data of each "o" event of an asciicast v2 file.
func readCast(r io.Reader) ([]string, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64<<10), 64<<20)
	var frames []string
	first := true
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		if first {
			first = false
			var h struct {
				Version int `json:"version"`
			}
			if err := json.Unmarshal(line, &h); err != nil || h.Version != 2 {
				return nil, fmt.Errorf("not an asciicast v2 header")
			}
			continue
		}
		var ev []json.RawMessage
		if err := json.Unmarshal(line, &ev); err != nil || len(ev) != 3 {
			return nil, fmt.Errorf("bad event %q", clip(string(line)))
		}
		var kind, data string
		if json.Unmarshal(ev[1], &kind) != nil || json.Unmarshal(ev[2], &data) != nil {
			return nil, fmt.Errorf("bad event %q", clip(string(line)))
		}
		if kind == "o" {
			frames = append(frames, data)
		}
	}
	if first {
		return nil, fmt.Errorf("empty file")
	}
	return frames, sc.Err()
}
