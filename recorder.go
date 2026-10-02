package tui

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"sync"
	"time"
)

// WithRecorder records the session to w as an asciicast v2 file
// (https://docs.asciinema.org/manual/asciicast/v2/): a header line, then one
// JSON event per line, "o" for each write to the terminal (a frame, or a mode
// change) and "i" for each chunk of input bytes read. The file plays in
// asciinema unmodified. Pair it with WithRecorderSidecar and tuitest.Replay to
// reproduce the session byte-for-byte in a test.
//
// Recording turns the frame cap off (as WithMaxFPS(0)): a capped Program
// paces non-input frames by the wall clock, so what it draws would not be a
// function of the messages it received. Input bytes are recorded as typed, so
// a recording of a password prompt contains the password. A write error on w
// stops the recording quietly and never fails the Program.
func WithRecorder(w io.Writer) ProgramOption {
	return func(p *Program) { p.recCast = w }
}

// WithRecorderSidecar records, to w, the facts Replay needs that an asciicast
// cannot hold: the exact input bytes (base64), terminal sizes, the Every tickers
// that fired, a mark per message handed to Update, and a SHA-256 of each write
// to the terminal. It is JSON lines, written one whole line per Write call. It
// is independent of WithRecorder, but the pair is what tuitest.Replay reads.
// Like WithRecorder it turns the frame cap off.
//
// Only what flows through the Program's own seams is recorded: input bytes,
// sizes and Every ticks. Messages from Program.Send, or Cmd results that depend
// on something outside the model (the network, the wall clock), are not, so a
// model that needs them does not replay.
func WithRecorderSidecar(w io.Writer) ProgramOption {
	return func(p *Program) { p.recSide = w }
}

// sidecarLine is one line of the sidecar. K is "hdr", "in", "size", "tick",
// "upd" or "out".
type sidecarLine struct {
	K   string  `json:"k"`
	V   int     `json:"v,omitempty"`   // hdr: format version
	T   float64 `json:"t,omitempty"`   // seconds since the recording began
	W   int     `json:"w,omitempty"`   // hdr, size
	H   int     `json:"h,omitempty"`   // hdr, size
	B   string  `json:"b,omitempty"`   // in: base64 of the bytes
	ID  *int    `json:"id,omitempty"`  // tick: the ticker's registration order
	D   int64   `json:"d,omitempty"`   // tick: its period in nanoseconds
	N   int     `json:"n,omitempty"`   // out: length in bytes
	Sha string  `json:"sha,omitempty"` // out: hex SHA-256
}

// recorder writes the cast and the sidecar. Its methods are safe on a nil
// receiver (no recording) and for concurrent use.
type recorder struct {
	mu    sync.Mutex
	cast  io.Writer
	side  io.Writer
	start time.Time
	begun bool
	buf   bytes.Buffer
}

func newRecorder(cast, side io.Writer) *recorder {
	if cast == nil && side == nil {
		return nil
	}
	return &recorder{cast: cast, side: side}
}

// line encodes v as one JSON line and writes it, whole, to dst.
func (r *recorder) line(dstp *io.Writer, v any) {
	dst := *dstp
	if dst == nil {
		return
	}
	r.buf.Reset()
	enc := json.NewEncoder(&r.buf)
	enc.SetEscapeHTML(false)
	if enc.Encode(v) != nil {
		return
	}
	if _, err := dst.Write(r.buf.Bytes()); err != nil {
		// Stop writing to a broken destination rather than retry every event.
		*dstp = nil
	}
}

func (r *recorder) elapsed() float64 { return time.Since(r.start).Seconds() }

// begin writes the headers once, with the starting size.
func (r *recorder) begin(w, h int) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.beginLocked(w, h)
}

func (r *recorder) beginLocked(w, h int) {
	if r.begun {
		return
	}
	r.begun = true
	r.start = time.Now()
	hdr := map[string]any{
		"version":   2,
		"width":     w,
		"height":    h,
		"timestamp": r.start.Unix(),
		"env":       map[string]string{"TERM": termName()},
	}
	r.line(&r.cast, hdr)
	r.line(&r.side, sidecarLine{K: "hdr", V: 1, W: w, H: h})
}

func termName() string {
	if t := os.Getenv("TERM"); t != "" {
		return t
	}
	return "xterm-256color"
}

// event writes one asciicast event.
func (r *recorder) event(kind string, b []byte) {
	r.line(&r.cast, []any{round6(r.elapsed()), kind, string(b)})
}

func round6(f float64) float64 { return float64(int64(f*1e6)) / 1e6 }

// out records one write to the terminal.
func (r *recorder) out(b []byte) {
	if r == nil || len(b) == 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.beginLocked(80, 24)
	r.event("o", b)
	sum := sha256.Sum256(b)
	r.line(&r.side, sidecarLine{K: "out", T: round6(r.elapsed()), N: len(b), Sha: hex.EncodeToString(sum[:])})
}

// in records bytes read from the input.
func (r *recorder) in(b []byte) {
	if r == nil || len(b) == 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.beginLocked(80, 24)
	r.event("i", b)
	r.line(&r.side, sidecarLine{K: "in", T: round6(r.elapsed()), B: base64.StdEncoding.EncodeToString(b)})
}

// size records a terminal size the loop handed to Update.
func (r *recorder) size(w, h int) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.beginLocked(w, h)
	r.line(&r.side, sidecarLine{K: "size", T: round6(r.elapsed()), W: w, H: h})
}

// tick records an Every tick of the id'th ticker, period d.
func (r *recorder) tick(id int, d time.Duration) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.beginLocked(80, 24)
	r.line(&r.side, sidecarLine{K: "tick", T: round6(r.elapsed()), ID: &id, D: int64(d)})
}

// upd records that Update returned for one message from the queue.
func (r *recorder) upd() {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.beginLocked(80, 24)
	r.line(&r.side, sidecarLine{K: "upd", T: round6(r.elapsed())})
}

// recWriter forwards to the real output and records what was written.
type recWriter struct {
	w io.Writer
	r *recorder
}

func (rw recWriter) Write(b []byte) (int, error) {
	n, err := rw.w.Write(b)
	rw.r.out(b[:n])
	return n, err
}

// recReader forwards reads and records the bytes returned.
type recReader struct {
	r   io.Reader
	rec *recorder
}

func (rr recReader) Read(b []byte) (int, error) {
	n, err := rr.r.Read(b)
	if n > 0 {
		rr.rec.in(b[:n])
	}
	return n, err
}

// setupRecorder runs at the end of NewProgram, after every option.
func (p *Program) setupRecorder() {
	p.rec = newRecorder(p.recCast, p.recSide)
	if p.rec == nil {
		return
	}
	// Pin the terminal to the real output before it is wrapped, so size and
	// raw-mode calls still reach the file.
	p.term = p.terminal()
	p.output = recWriter{w: p.output, r: p.rec}
	p.maxFPS, p.fpsSet = 0, true
}
