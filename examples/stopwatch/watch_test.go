package main

import (
	"testing"
	"time"
)

var t0 = time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)

func TestWatchCountsFromTheClockNotFromTicks(t *testing.T) {
	w := watch{}.start(t0)
	if got := w.elapsed(t0.Add(1500 * time.Millisecond)); got != 1500*time.Millisecond {
		t.Fatalf("%v", got)
	}
	w = w.stop(t0.Add(2 * time.Second))
	if w.running || w.elapsed(t0.Add(time.Hour)) != 2*time.Second {
		t.Fatal("a stopped watch kept counting")
	}
	w = w.start(t0.Add(10 * time.Second))
	if got := w.elapsed(t0.Add(13 * time.Second)); got != 5*time.Second {
		t.Fatalf("resumed: %v", got)
	}
	if again := w.start(t0.Add(20 * time.Second)); again.since != w.since {
		t.Fatal("starting a running watch restarted it")
	}
	if w.stop(t0.Add(13*time.Second)).stop(t0.Add(30*time.Second)).elapsed(t0) != 5*time.Second {
		t.Fatal("stopping twice changed the time")
	}
}

func TestLapsRecordLengthAndTotal(t *testing.T) {
	w := watch{}.start(t0)
	if len(w.lap(t0).laps) != 0 {
		t.Fatal("a lap at zero was recorded")
	}
	w = w.lap(t0.Add(1230 * time.Millisecond)).lap(t0.Add(3230 * time.Millisecond)).lap(t0.Add(3730 * time.Millisecond))
	want := []lap{{1230 * time.Millisecond, 1230 * time.Millisecond}, {2 * time.Second, 3230 * time.Millisecond}, {500 * time.Millisecond, 3730 * time.Millisecond}}
	for i, l := range want {
		if w.laps[i] != l {
			t.Errorf("lap %d = %+v, want %+v", i+1, w.laps[i], l)
		}
	}
	fast, slow := w.extremes()
	if fast != 2 || slow != 1 {
		t.Fatalf("fast %d slow %d", fast, slow)
	}
	if f, s := (watch{}).lap(t0).extremes(); f != -1 || s != -1 {
		t.Fatal("no laps have no extremes")
	}
	one := watch{laps: []lap{{time.Second, time.Second}}}
	if f, s := one.extremes(); f != -1 || s != -1 {
		t.Fatal("one lap is neither fastest nor slowest")
	}
	if len(w.reset().laps) != 0 || w.reset().running {
		t.Fatal("reset left something")
	}
}

func TestLapsDoNotShareStorage(t *testing.T) {
	a := watch{}.start(t0).lap(t0.Add(time.Second))
	b := a.lap(t0.Add(2 * time.Second))
	c := a.lap(t0.Add(3 * time.Second))
	if b.laps[1].total == c.laps[1].total || len(a.laps) != 1 {
		t.Fatal("two laps taken from the same watch overwrote each other")
	}
}

func TestStampFormat(t *testing.T) {
	for in, want := range map[time.Duration]string{0: "00:00.00", 1230 * time.Millisecond: "00:01.23", 59990 * time.Millisecond: "00:59.99", 61 * time.Second: "01:01.00", time.Hour + 2*time.Second + 50*time.Millisecond: "1:00:02.05", -time.Second: "00:00.00"} {
		if got := stamp(in); got != want {
			t.Errorf("stamp(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestBigDigits(t *testing.T) {
	rows := big("12:30.5")
	if len(rows) != 3 || rows[0] == "" || big("x")[0] != "" {
		t.Fatalf("%q", rows)
	}
	for _, r := range "0123456789:." {
		if _, ok := glyphs[r]; !ok {
			t.Errorf("no glyph for %q", r)
		}
	}
	if bigWidth("00:00.00") != 4*6+2*1+3 && bigWidth("00:00.00") < 20 {
		t.Fatalf("width %d", bigWidth("00:00.00"))
	}
}
