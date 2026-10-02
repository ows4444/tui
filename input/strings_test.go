package input

import (
	"io"
	"strings"
	"testing"
	"time"
)

// Criterion 1: Alt+] then ordinary keys delivers Alt+] within the ESC timeout
// and then each following key.
func TestAltBracketTimedThenKeys(t *testing.T) {
	pr, pw := io.Pipe()
	defer pw.Close()
	rd := NewReader(pr)
	rd.SetEscTimeout(20 * time.Millisecond)
	go func() {
		_, _ = pw.Write([]byte("\x1b]"))
		time.Sleep(200 * time.Millisecond)
		_, _ = pw.Write([]byte("2hello\r"))
	}()
	start := time.Now()
	ev, err := rd.ReadEvent()
	if err != nil {
		t.Fatal(err)
	}
	if k, ok := ev.(Key); !ok || k.Type != KeyRunes || k.Code != ']' || !k.Mod.Alt() {
		t.Fatalf("first = %#v, want Alt+]", ev)
	}
	if d := time.Since(start); d > 150*time.Millisecond {
		t.Errorf("Alt+] took %v, want within the ESC timeout", d)
	}
	for i, want := range "2hello" {
		ev, err := rd.ReadEvent()
		if err != nil {
			t.Fatal(err)
		}
		if k, ok := ev.(Key); !ok || k.Type != KeyRunes || k.Code != want {
			t.Fatalf("key %d = %#v, want %q", i, ev, want)
		}
	}
	if ev, _ := rd.ReadEvent(); !isEnter(ev) {
		t.Errorf("last = %#v, want Enter", ev)
	}
}

// Criterion 2: DCS/SOS/PM/APC terminated by ST or BEL deliver no Key events.
func TestStringSequencesDeliverNoKeys(t *testing.T) {
	for _, kind := range []byte{'P', 'X', '^', '_'} {
		for _, term := range []string{"\x07", "\x1b\\"} {
			in := "\x1b" + string(kind) + "1+r5443=787465726d" + term + "z"
			got := readAll(t, in, 2)
			r, ok := got[0].(ReplyEvent)
			if !ok || r.Kind != kind || r.Data != "1+r5443=787465726d" {
				t.Errorf("%q first = %#v, want ReplyEvent", in, got[0])
			}
			if k, ok := got[1].(Key); !ok || k.String() != "z" {
				t.Errorf("%q second = %#v, want z", in, got[1])
			}
		}
	}
}

// Criterion 3: an OSC longer than 256 bytes leaks no Key events; past the
// 64 KiB bound the tail is still discarded through the terminator.
func TestLongOSCDeliversNoKeys(t *testing.T) {
	for _, n := range []int{300, maxStringLen + 1000} {
		for _, term := range []string{"\x07", "\x1b\\"} {
			in := "\x1b]52;c;" + strings.Repeat("A", n) + term + "z"
			got := readAll(t, in, 2)
			if _, ok := got[0].(ReplyEvent); !ok {
				t.Errorf("n=%d first = %T, want ReplyEvent", n, got[0])
			} else if l := len(got[0].(ReplyEvent).Data); l > maxStringLen {
				t.Errorf("n=%d data len %d exceeds bound", n, l)
			}
			if k, ok := got[1].(Key); !ok || k.String() != "z" {
				t.Errorf("n=%d second = %#v, want z", n, got[1])
			}
		}
	}
}

func isEnter(ev Event) bool {
	k, ok := ev.(Key)
	return ok && k.Type == KeyEnter
}
