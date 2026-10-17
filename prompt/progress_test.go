package prompt

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
)

func quiet(out *bytes.Buffer) Config {
	return Config{Program: []tui.ProgramOption{tui.WithInput(blockingReader{}), tui.WithOutput(out), tui.WithReducedMotion(true)}}
}

func TestProgressReportsAndReturnsTheError(t *testing.T) {
	boom := errors.New("boom")
	out := &bytes.Buffer{}
	err := Progress("Copying", 10, func(ctx context.Context, report func(int)) error {
		report(4)
		time.Sleep(250 * time.Millisecond) // long enough for a redraw tick
		return boom
	}, quiet(out))
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
	if s := ansi.StripANSI(out.String()); !strings.Contains(s, "Copying") || !strings.Contains(s, "40%") {
		t.Fatalf("the bar did not show 40%%:\n%s", s)
	}
}

func TestProgressSucceedsAtOneHundredPercent(t *testing.T) {
	out := &bytes.Buffer{}
	if err := Progress("Work", 3, func(context.Context, func(int)) error { return nil }, quiet(out)); err != nil {
		t.Fatal(err)
	}
	if s := ansi.StripANSI(out.String()); !strings.Contains(s, "100%") {
		t.Fatalf("a finished run should end on 100%%:\n%s", s)
	}
}

func TestTrackRunsEveryItemInOrder(t *testing.T) {
	var seen []string
	err := Track("Files", []string{"a", "b", "c"}, func(_ context.Context, it string) error {
		seen = append(seen, it)
		return nil
	}, quiet(&bytes.Buffer{}))
	if err != nil || strings.Join(seen, "") != "abc" {
		t.Fatalf("err %v seen %v", err, seen)
	}
}

func TestTrackStopsAtTheFirstError(t *testing.T) {
	boom := errors.New("boom")
	var n int
	err := Track("Files", []int{1, 2, 3, 4}, func(_ context.Context, it int) error {
		n++
		if it == 2 {
			return boom
		}
		return nil
	}, quiet(&bytes.Buffer{}))
	if !errors.Is(err, boom) || n != 2 {
		t.Fatalf("err %v after %d items", err, n)
	}
}

func TestProgressCancelStopsTheFunction(t *testing.T) {
	cfg := Config{Program: []tui.ProgramOption{
		tui.WithInput(&oneThenBlock{data: "\x03"}), tui.WithOutput(&bytes.Buffer{}), tui.WithReducedMotion(true),
	}}
	err := Progress("Work", 5, func(ctx context.Context, _ func(int)) error {
		<-ctx.Done()
		return ctx.Err()
	}, cfg)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v; want context.Canceled", err)
	}
}

func TestTrackWithNoItems(t *testing.T) {
	if err := Track("None", []int(nil), func(context.Context, int) error { return errors.New("never") }, quiet(&bytes.Buffer{})); err != nil {
		t.Fatalf("err = %v", err)
	}
}
