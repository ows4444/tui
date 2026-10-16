package tui

import (
	"context"
	"sync"
)

// TaskDoneMsg reports the end of one run of a task started with Tasks.Start.
//
// Msg is what the task function returned (nil if it returned nothing).
// Canceled is true when the run was superseded by a newer Start with the same
// key, or ended by Tasks.Cancel or Tasks.CancelAll; its Msg is then stale and
// should be ignored. Use Tasks.Running to know whether the key is still busy.
type TaskDoneMsg struct {
	Key      string
	Msg      Msg
	Canceled bool
}

// Tasks runs background work by key. Starting a key while an earlier run of it
// is still going cancels the earlier run first, so "search as the user types"
// is one Start per keystroke with no stale results to filter by hand.
//
// Create one with NewTasks and keep the pointer in the model. A Tasks is safe
// for use from the event loop and from the task functions themselves.
type Tasks struct {
	mu   sync.Mutex
	gen  uint64
	live map[string]*taskRun
}

type taskRun struct {
	id   uint64
	stop chan struct{} // closed to cancel this run
	once sync.Once
}

func (r *taskRun) cancel() { r.once.Do(func() { close(r.stop) }) }

// NewTasks returns an empty Tasks.
func NewTasks() *Tasks { return &Tasks{live: map[string]*taskRun{}} }

// Start returns a Cmd that runs fn on its own goroutine and delivers a
// TaskDoneMsg when fn returns. fn gets a context that is cancelled when this
// run is superseded or cancelled, and when the Program ends. A run that is
// already going under the same key is cancelled before this one starts, at the
// call to Start. A nil fn gives a nil Cmd.
func (t *Tasks) Start(key string, fn CtxCmd) Cmd {
	if fn == nil {
		return nil
	}
	t.mu.Lock()
	if old, ok := t.live[key]; ok {
		old.cancel()
	}
	t.gen++
	run := &taskRun{id: t.gen, stop: make(chan struct{})}
	t.live[key] = run
	t.mu.Unlock()

	return FromCtx(func(parent context.Context) Msg {
		ctx, cancel := context.WithCancel(parent)
		defer cancel()
		go func() {
			select {
			case <-run.stop:
				cancel()
			case <-ctx.Done():
			}
		}()
		msg := fn(ctx)

		t.mu.Lock()
		current := t.live[key] == run
		if current {
			delete(t.live, key)
		}
		t.mu.Unlock()
		select {
		case <-run.stop:
			current = false
		default:
		}
		return TaskDoneMsg{Key: key, Msg: msg, Canceled: !current}
	})
}

// Cancel cancels the run under key, if any. The run still delivers a
// TaskDoneMsg with Canceled set once its function returns.
func (t *Tasks) Cancel(key string) {
	t.mu.Lock()
	run, ok := t.live[key]
	delete(t.live, key)
	t.mu.Unlock()
	if ok {
		run.cancel()
	}
}

// CancelAll cancels every run.
func (t *Tasks) CancelAll() {
	t.mu.Lock()
	runs := make([]*taskRun, 0, len(t.live))
	for k, r := range t.live {
		runs = append(runs, r)
		delete(t.live, k)
	}
	t.mu.Unlock()
	for _, r := range runs {
		r.cancel()
	}
}

// Running reports whether a run is going under key: it was started and has
// neither returned nor been cancelled.
func (t *Tasks) Running(key string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	_, ok := t.live[key]
	return ok
}
