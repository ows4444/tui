package tui

import (
	"sync"
	"testing"
	"time"
)

type sendOrdMsg struct{ g, n int }

// sendRecModel records every sendOrdMsg it sees and quits after want of them.
type sendRecModel struct {
	mu   *sync.Mutex
	got  *[]sendOrdMsg
	want int
}

func (sendRecModel) Init() Cmd { return nil }
func (m sendRecModel) Update(msg Msg) (Model, Cmd) {
	if s, ok := msg.(sendOrdMsg); ok {
		m.mu.Lock()
		*m.got = append(*m.got, s)
		n := len(*m.got)
		m.mu.Unlock()
		if n == m.want {
			return m, Quit()
		}
	}
	return m, nil
}
func (sendRecModel) View() string { return "x" }

func startSendLoop(t *testing.T, p *Program) chan runResult {
	t.Helper()
	ch := make(chan runResult, 1)
	go func() {
		m, err := p.runLoop()
		ch <- runResult{m, err}
	}()
	deadline := time.Now().Add(2 * time.Second)
	for {
		// Queued messages may end the loop before this poll sees it running.
		if _, ended := p.loopState(); p.running() || ended {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("loop never started")
		}
		time.Sleep(time.Millisecond)
	}
	return ch
}

func newSendProgram(t *testing.T, m Model) *Program {
	pr, pw := mustPipe(t)
	t.Cleanup(func() { pr.Close(); pw.Close() })
	dn := mustDevNull(t)
	t.Cleanup(func() { dn.Close() })
	return NewProgram(m, WithInput(pr), WithOutput(dn))
}

// #23: per-goroutine order preserved, from concurrent senders (more than the
// channel buffer so Send must wait for room rather than drop).
func TestSendOrderPerGoroutine(t *testing.T) {
	const gs, per = 4, 100
	var mu sync.Mutex
	var got []sendOrdMsg
	p := newSendProgram(t, sendRecModel{&mu, &got, gs * per})
	ch := startSendLoop(t, p)
	var wg sync.WaitGroup
	for g := 0; g < gs; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; n < per; n++ {
				p.Send(sendOrdMsg{g, n})
			}
		}()
	}
	wg.Wait()
	select {
	case r := <-ch:
		if r.err != nil {
			t.Fatal(r.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
	next := map[int]int{}
	for _, s := range got {
		if s.n != next[s.g] {
			t.Fatalf("goroutine %d: got %d want %d", s.g, s.n, next[s.g])
		}
		next[s.g]++
	}
	if len(got) != gs*per {
		t.Fatalf("got %d msgs", len(got))
	}
}

// #24: before Run and after return, Send and Quit neither block nor panic.
func TestSendBeforeAndAfterRun(t *testing.T) {
	var mu sync.Mutex
	var got []sendOrdMsg
	p := newSendProgram(t, sendRecModel{&mu, &got, 1})
	fin := make(chan struct{})
	go func() {
		for i := 0; i < 10; i++ {
			p.Send(sendOrdMsg{})
		}
		p.Quit()
		close(fin)
	}()
	select {
	case <-fin:
	case <-time.After(2 * time.Second):
		t.Fatal("Send blocked before Run")
	}
	ch := startSendLoop(t, p)
	p.Send(sendOrdMsg{0, 0}) // quits the model
	<-ch
	fin = make(chan struct{})
	go func() {
		for i := 0; i < 1000; i++ {
			p.Send(sendOrdMsg{})
		}
		p.Quit()
		close(fin)
	}()
	select {
	case <-fin:
	case <-time.After(2 * time.Second):
		t.Fatal("Send blocked after Run")
	}
	// Pre-Run sends are now queued and delivered; the first sendOrdMsg
	// quits the model, so at least one was seen and the loop ended.
	if len(got) < 1 {
		t.Fatalf("no messages reached Update: %d", len(got))
	}
}

// #47: Send before Run is delivered to Update once Run starts.
func TestSendBeforeRunDelivered(t *testing.T) {
	var mu sync.Mutex
	var got []sendOrdMsg
	p := newSendProgram(t, sendRecModel{&mu, &got, 2})
	p.Send(sendOrdMsg{0, 0})
	p.Send(sendOrdMsg{0, 1})
	ch := startSendLoop(t, p)
	select {
	case r := <-ch:
		if r.err != nil {
			t.Fatal(r.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
	mu.Lock()
	defer mu.Unlock()
	if len(got) != 2 || got[0].n != 0 || got[1].n != 1 {
		t.Fatalf("got %v", got)
	}
}

// #47: TrySend returns false without blocking when the queue is full.
func TestTrySendFullQueue(t *testing.T) {
	p := newSendProgram(t, sendIdleModel{})
	n := cap(p.msgs)
	for i := 0; i < n; i++ {
		if !p.TrySend(sendOrdMsg{0, i}) {
			t.Fatalf("TrySend %d returned false with room", i)
		}
	}
	res := make(chan bool, 1)
	go func() { res <- p.TrySend(sendOrdMsg{}) }()
	select {
	case ok := <-res:
		if ok {
			t.Fatal("TrySend on full queue returned true")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("TrySend blocked")
	}
	if p.TrySend(nil) {
		t.Fatal("TrySend(nil) returned true")
	}
}

type sendIdleModel struct{}

func (sendIdleModel) Init() Cmd                 { return nil }
func (m sendIdleModel) Update(Msg) (Model, Cmd) { return m, nil }
func (sendIdleModel) View() string              { return "tick" }

// #25 and #26: Quit from another goroutine ends the loop with nil, racing
// Send calls and rendering.
func TestQuitConcurrentWithSendAndRender(t *testing.T) {
	p := newSendProgram(t, sendIdleModel{})
	ch := startSendLoop(t, p)
	var wg sync.WaitGroup
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				p.Send(i)
			}
		}()
	}
	wg.Add(1)
	go func() { defer wg.Done(); time.Sleep(5 * time.Millisecond); p.Quit(); p.Quit() }()
	select {
	case r := <-ch:
		if r.err != nil {
			t.Fatalf("err %v", r.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Quit did not end loop")
	}
	wg.Wait()
}
