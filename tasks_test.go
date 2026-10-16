package tui

import (
	"context"
	"testing"
	"time"
)

func runTask(ctx context.Context, cmd Cmd) <-chan TaskDoneMsg {
	out := make(chan TaskDoneMsg, 1)
	go func() { out <- RunCmd(ctx, cmd).(TaskDoneMsg) }()
	return out
}

func wait(t *testing.T, ch <-chan TaskDoneMsg) TaskDoneMsg {
	t.Helper()
	select {
	case m := <-ch:
		return m
	case <-time.After(2 * time.Second):
		t.Fatal("task did not finish")
		return TaskDoneMsg{}
	}
}

func TestTaskDeliversItsResult(t *testing.T) {
	ts := NewTasks()
	cmd := ts.Start("k", func(context.Context) Msg { return "hello" })
	if !ts.Running("k") {
		t.Error("Running is false right after Start")
	}
	got := wait(t, runTask(context.Background(), cmd))
	if got.Key != "k" || got.Msg != "hello" || got.Canceled {
		t.Errorf("got %+v", got)
	}
	if ts.Running("k") {
		t.Error("Running is still true after the task returned")
	}
}

func TestStartingAKeyAgainCancelsTheEarlierRun(t *testing.T) {
	ts := NewTasks()
	started := make(chan struct{})
	first := runTask(context.Background(), ts.Start("search", func(ctx context.Context) Msg {
		close(started)
		<-ctx.Done()
		return "stale"
	}))
	<-started
	second := runTask(context.Background(), ts.Start("search", func(context.Context) Msg { return "fresh" }))

	if got := wait(t, first); !got.Canceled {
		t.Errorf("the superseded run is not marked Canceled: %+v", got)
	}
	if got := wait(t, second); got.Canceled || got.Msg != "fresh" {
		t.Errorf("the newer run: %+v", got)
	}
	if ts.Running("search") {
		t.Error("key still running after both returned")
	}
}

func TestCancelAndCancelAll(t *testing.T) {
	ts := NewTasks()
	block := func(ctx context.Context) Msg { <-ctx.Done(); return nil }
	a := runTask(context.Background(), ts.Start("a", block))
	b := runTask(context.Background(), ts.Start("b", block))
	ts.Cancel("a")
	if ts.Running("a") || !ts.Running("b") {
		t.Errorf("after Cancel(a): a=%v b=%v", ts.Running("a"), ts.Running("b"))
	}
	if got := wait(t, a); !got.Canceled {
		t.Errorf("a: %+v", got)
	}
	ts.CancelAll()
	if got := wait(t, b); !got.Canceled {
		t.Errorf("b: %+v", got)
	}
	ts.Cancel("missing") // no run, no panic
}

func TestTaskStopsWhenTheProgramContextEnds(t *testing.T) {
	ts := NewTasks()
	ctx, cancel := context.WithCancel(context.Background())
	done := runTask(ctx, ts.Start("k", func(c context.Context) Msg { <-c.Done(); return "late" }))
	cancel()
	if got := wait(t, done); got.Msg != "late" {
		t.Errorf("got %+v", got)
	}
}

func TestNilTaskFunctionGivesNilCmd(t *testing.T) {
	if NewTasks().Start("k", nil) != nil {
		t.Error("Start(nil) is not nil")
	}
}
