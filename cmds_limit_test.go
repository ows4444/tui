package tui

import (
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestMaxConcurrentCmdsLimitsAndOrders(t *testing.T) {
	const n, total = 2, 6
	p := NewProgram(recorderModel{}, WithMaxConcurrentCmds(n))
	p.msgs = make(chan Msg, total)
	done := make(chan struct{})
	defer close(done)

	var cur, peak atomic.Int32
	gate := make(chan struct{})
	var mu sync.Mutex
	var started []int
	for i := 0; i < total; i++ {
		i := i
		p.spawn(func() Msg {
			c := cur.Add(1)
			for {
				pk := peak.Load()
				if c <= pk || peak.CompareAndSwap(pk, c) {
					break
				}
			}
			mu.Lock()
			started = append(started, i)
			mu.Unlock()
			<-gate
			cur.Add(-1)
			return i
		}, done)
	}
	time.Sleep(50 * time.Millisecond)
	if got := cur.Load(); got != n {
		t.Fatalf("running = %d before release, want %d", got, n)
	}
	for i := 0; i < total; i++ {
		gate <- struct{}{}
		// One at a time, so the next queued Cmd's start is observable in order.
		time.Sleep(20 * time.Millisecond)
	}
	for i := 0; i < total; i++ {
		select {
		case <-p.msgs:
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for Cmd results")
		}
	}
	if pk := peak.Load(); pk > n {
		t.Errorf("peak concurrency %d > %d", pk, n)
	}
	mu.Lock()
	defer mu.Unlock()
	// The first n start in any order; the queued rest must follow in order.
	for i := n; i < total; i++ {
		if started[i] != i {
			t.Fatalf("start order = %v, want queued Cmds in arrival order", started)
		}
	}
}

func TestDocDocumentsBatchUnorderedSequenceOrdered(t *testing.T) {
	src, err := os.ReadFile("doc.go")
	if err != nil {
		t.Fatal(err)
	}
	s := string(src)
	for _, want := range []string{"Batch", "no particular order", "Sequence", "in order"} {
		if !strings.Contains(s, want) {
			t.Errorf("doc.go missing %q", want)
		}
	}
}
