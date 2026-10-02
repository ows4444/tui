package announce

import (
	"strings"
	"testing"
	"time"
)

type fake struct {
	now   time.Time
	fires []func()
}

func (f *fake) region(rows int) *Region {
	return &Region{
		Rows:  rows,
		Now:   func() time.Time { return f.now },
		After: func(_ time.Duration, fn func()) { f.fires = append(f.fires, fn) },
	}
}

func TestRegionOffLeavesTheViewAlone(t *testing.T) {
	var r Region
	r.Show("x", func() {}) // never reached in the runtime with Rows 0, but must not panic
	if got := r.Compose("a\nb", false, 10); got != "a\nb" {
		t.Fatalf("Compose = %q", got)
	}
}

func TestComposePadsAndKeepsTheNewest(t *testing.T) {
	f := &fake{now: time.Unix(1, 0)}
	r := f.region(2)
	if got := r.Compose("v", false, 0); got != "v\n\n" {
		t.Fatalf("empty region = %q, want two blank rows", got)
	}
	for _, s := range []string{"one", "two", "three"} {
		r.Show(s, func() {})
	}
	if got := r.Compose("v", false, 0); got != "v\ntwo\nthree" {
		t.Fatalf("Compose = %q", got)
	}
	if r.Len() != 2 {
		t.Fatalf("Len = %d, want 2 (older items are dropped)", r.Len())
	}
}

func TestComposeCutsAnAlternateScreenViewToLeaveRoom(t *testing.T) {
	f := &fake{now: time.Unix(1, 0)}
	r := f.region(1)
	r.Show("note", func() {})
	if got := r.Compose("1\n2\n3\n4", true, 3); got != "1\n2\nnote" {
		t.Fatalf("alt screen: %q", got)
	}
	if got := r.Compose("1\n2\n3\n4", false, 3); got != "1\n2\n3\n4\nnote" {
		t.Fatalf("inline: %q", got)
	}
	if got := r.Compose("1\n2", true, 1); got != "1\n2\nnote" {
		t.Fatalf("height no larger than rows must not cut: %q", got)
	}
}

func TestShowArmsAnExpiryAfterTheTTL(t *testing.T) {
	f := &fake{now: time.Unix(1000, 0)}
	r := f.region(1)
	expired := 0
	r.Show("hi", func() { expired++ })
	if len(f.fires) != 1 {
		t.Fatalf("timers = %d, want 1", len(f.fires))
	}
	f.now = f.now.Add(TTL - time.Millisecond)
	if r.Prune() || r.Len() != 1 {
		t.Fatal("pruned before the TTL")
	}
	f.now = f.now.Add(2 * time.Millisecond)
	if !r.Prune() || r.Len() != 0 {
		t.Fatal("not pruned after the TTL")
	}
	if r.Prune() {
		t.Fatal("Prune reported a change with nothing to drop")
	}
	f.fires[0]()
	if expired != 1 {
		t.Fatal("expire callback did not run")
	}
}

func TestShowUsesTheWallClockTimerByDefault(t *testing.T) {
	r := Region{Rows: 1}
	done := make(chan struct{})
	r.Show("x", func() { close(done) })
	if r.Len() != 1 {
		t.Fatal("not shown")
	}
	// The real timer would take TTL; only check it was armed without panicking.
	select {
	case <-done:
		t.Fatal("expired immediately")
	default:
	}
}

func TestNotifyNumbersAndSanitises(t *testing.T) {
	var r Region
	if got := r.Notify("saved"); got != "\x1b]99;i=1:u=2;saved\x1b\\" {
		t.Fatalf("first = %q", got)
	}
	got := r.Notify("a\x1b[31mred\x1b[0m\x07\nb\x1b]8;;http://x\x07link\x1b]8;;\x07")
	if !strings.HasPrefix(got, "\x1b]99;i=2:u=2;") || !strings.HasSuffix(got, "\x1b\\") {
		t.Fatalf("second = %q", got)
	}
	body := strings.TrimSuffix(strings.TrimPrefix(got, "\x1b]99;i=2:u=2;"), "\x1b\\")
	if strings.ContainsAny(body, "\x1b\x07\n") || !strings.Contains(body, "red") || !strings.Contains(body, "link") || strings.Contains(body, "http") {
		t.Fatalf("body = %q", body)
	}
}

func TestPoliteDedupWindow(t *testing.T) {
	var q Queue
	t0 := time.Unix(10, 0)
	if !q.Polite("saved", "saved", t0) {
		t.Fatal("first announcement rejected")
	}
	if q.Polite("saved", "saved", t0.Add(PoliteDedupWindow-time.Millisecond)) {
		t.Fatal("repeat inside the window accepted")
	}
	if !q.Polite("other", "other", t0.Add(10*time.Millisecond)) {
		t.Fatal("different text rejected")
	}
	if !q.Polite("saved", "saved", t0.Add(PoliteDedupWindow+time.Second)) {
		t.Fatal("repeat after the window rejected")
	}
	if got := q.Take(); strings.Join(got, ",") != "saved,other,saved" {
		t.Fatalf("Take = %v", got)
	}
	if q.Pending() || len(q.Take()) != 0 {
		t.Fatal("Take did not empty the queue")
	}
}

// Repeats are compared on the text as sent, not the stripped line, as before.
func TestDedupComparesTheRawText(t *testing.T) {
	var q Queue
	now := time.Unix(1, 0)
	q.Polite("\x1b[1mhi\x1b[0m", "hi", now)
	if !q.Polite("hi", "hi", now.Add(time.Millisecond)) {
		t.Fatal("plain text was taken for a repeat of the styled text")
	}
}

func TestAssertiveDropsPendingAndResetsDedup(t *testing.T) {
	var q Queue
	now := time.Unix(1, 0)
	q.Polite("a", "a", now)
	q.Assertive()
	if q.Pending() {
		t.Fatal("pending polite lines survived an assertive announcement")
	}
	if !q.Polite("a", "a", now.Add(time.Millisecond)) {
		t.Fatal("de-duplication was not reset by the assertive announcement")
	}
}

func TestStripOSCAndStrip(t *testing.T) {
	for in, want := range map[string]string{
		"plain":                              "plain",
		"\x1b]8;;http://x\x07go\x1b]8;;\x07": "go",
		"\x1b]8;;http://x\x1b\\go":           "go",
		"keep\x1b]0;unterminated":            "keep",
	} {
		if got := StripOSC(in); got != want {
			t.Errorf("StripOSC(%q) = %q, want %q", in, got, want)
		}
	}
	if got := Strip("\x1b[1;31mred\x1b[0m \x1b]8;;u\x07link\x1b]8;;\x07"); got != "red link" {
		t.Fatalf("Strip = %q", got)
	}
	if got := Strip("no escapes"); got != "no escapes" {
		t.Fatalf("Strip = %q", got)
	}
}
