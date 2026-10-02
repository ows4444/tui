package input

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

type pipeCase struct {
	name string
	make func(t *testing.T) (io.Reader, io.WriteCloser)
}

// The two read paths: io.Pipe has no read deadline (goroutine peek), os.Pipe
// does (deadline peek).
var pipes = []pipeCase{
	{"io.Pipe", func(t *testing.T) (io.Reader, io.WriteCloser) {
		r, w := io.Pipe()
		return r, w
	}},
	{"os.Pipe", func(t *testing.T) (io.Reader, io.WriteCloser) {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { r.Close() })
		return r, w
	}},
}

type result struct {
	ev  Event
	err error
	at  time.Time
}

func readAsync(rd *Reader) <-chan result {
	ch := make(chan result, 1)
	go func() {
		ev, err := rd.ReadEvent()
		ch <- result{ev, err, time.Now()}
	}()
	return ch
}

func wait(t *testing.T, ch <-chan result, d time.Duration) result {
	t.Helper()
	select {
	case r := <-ch:
		return r
	case <-time.After(d):
		t.Fatal("ReadEvent still blocked")
		return result{}
	}
}

// #13: a lost end marker ends the paste after the idle timeout, marked
// Incomplete. #14: a key typed afterwards is delivered promptly.
func TestPasteIdleTimeoutOnLostEndMarker(t *testing.T) {
	for _, pc := range pipes {
		t.Run(pc.name, func(t *testing.T) {
			r, w := pc.make(t)
			defer w.Close()
			rd := NewReader(r)
			start := time.Now()
			ch := readAsync(rd)
			if _, err := w.Write([]byte("\x1b[200~hello")); err != nil {
				t.Fatal(err)
			}
			res := wait(t, ch, 2*time.Second)
			if res.err != nil {
				t.Fatal(res.err)
			}
			if got, want := res.ev, (PasteEvent{Text: "hello", Incomplete: true}); got != want {
				t.Fatalf("got %#v, want %#v", got, want)
			}
			if d := res.at.Sub(start); d < PasteIdleTimeout {
				t.Errorf("paste ended after %v, before the %v idle timeout", d, PasteIdleTimeout)
			}

			ch = readAsync(rd)
			sent := time.Now()
			if _, err := w.Write([]byte("q")); err != nil {
				t.Fatal(err)
			}
			res = wait(t, ch, 2*time.Second)
			if k, ok := res.ev.(Key); !ok || k.Type != KeyRunes || k.Text != "q" {
				t.Fatalf("got %#v, want key q", res.ev)
			}
			if d := res.at.Sub(sent); d > 50*time.Millisecond {
				t.Errorf("key delivered %v after it was typed, want within 50ms", d)
			}
		})
	}
}

func TestPasteWithMarkerIsCompleteAndSlowPasteSurvives(t *testing.T) {
	r, w := io.Pipe()
	rd := NewReader(r)
	ch := readAsync(rd)
	go func() {
		w.Write([]byte("\x1b[200~ab"))
		time.Sleep(PasteIdleTimeout / 3) // slower than a burst, faster than the timeout
		w.Write([]byte("cd\x1b[201~"))
	}()
	res := wait(t, ch, 2*time.Second)
	if got, want := res.ev, (PasteEvent{Text: "abcd"}); got != want {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestPasteEndMarkerSplitAcrossWrites(t *testing.T) {
	r, w := io.Pipe()
	rd := NewReader(r)
	ch := readAsync(rd)
	go func() {
		w.Write([]byte("\x1b[200~ab\x1b[2"))
		time.Sleep(20 * time.Millisecond)
		w.Write([]byte("01~"))
	}()
	res := wait(t, ch, 2*time.Second)
	if got, want := res.ev, (PasteEvent{Text: "ab"}); got != want {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestPasteEscapeInsideContentIsKept(t *testing.T) {
	got := readAll(t, "\x1b[200~a\x1b[Ab\x1b[201~", 1)[0]
	if want := (PasteEvent{Text: "a\x1b[Ab"}); got != want {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

// #15: over 16 MiB, the first 16 MiB are delivered as one truncated event and
// the rest is discarded up to the end marker, so the next key still decodes.
func TestPasteOverCapIsTruncatedAndRestDiscarded(t *testing.T) {
	body := strings.Repeat("x", MaxPasteBytes) + strings.Repeat("y", 1000)
	in := "\x1b[200~" + body + "\x1b[201~z"
	rd := NewReader(bytes.NewReader([]byte(in)))
	ev, err := rd.ReadEvent()
	if err != nil {
		t.Fatal(err)
	}
	p, ok := ev.(PasteEvent)
	if !ok {
		t.Fatalf("got %#v", ev)
	}
	if !p.Truncated || p.Incomplete {
		t.Errorf("Truncated=%v Incomplete=%v, want true/false", p.Truncated, p.Incomplete)
	}
	if len(p.Text) != MaxPasteBytes || strings.Trim(p.Text, "x") != "" {
		t.Errorf("Text has %d bytes, want exactly the first %d", len(p.Text), MaxPasteBytes)
	}
	ev, err = rd.ReadEvent()
	if k, ok := ev.(Key); err != nil || !ok || k.Text != "z" {
		t.Fatalf("after the paste got %#v, %v; want key z", ev, err)
	}
}

func TestPasteExactlyAtCapIsNotTruncated(t *testing.T) {
	in := "\x1b[200~" + strings.Repeat("x", MaxPasteBytes) + "\x1b[201~"
	ev, _ := NewReader(bytes.NewReader([]byte(in))).ReadEvent()
	if p := ev.(PasteEvent); p.Truncated || len(p.Text) != MaxPasteBytes {
		t.Fatalf("Truncated=%v len=%d", p.Truncated, len(p.Text))
	}
}

func TestPasteCapNeverSplitsACharacter(t *testing.T) {
	// 3-byte runes: the cap falls inside one, so it must be dropped whole.
	in := "\x1b[200~" + strings.Repeat("世", MaxPasteBytes/3+2) + "\x1b[201~"
	ev, _ := NewReader(bytes.NewReader([]byte(in))).ReadEvent()
	p := ev.(PasteEvent)
	if !p.Truncated || len(p.Text) != MaxPasteBytes/3*3 || strings.Contains(p.Text, "�") {
		t.Fatalf("Truncated=%v len=%d", p.Truncated, len(p.Text))
	}
}
