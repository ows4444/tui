package termio

import "sync"

// Fake is a Terminal for tests: it answers from its fields and counts calls.
// The zero value is not a terminal, has no size, and succeeds at everything.
type Fake struct {
	Interactive bool
	W, H        int // reported by Size when SizeKnown
	SizeKnown   bool
	RawErr      error // returned by MakeRaw
	VTErr       error // returned by EnableOutputVT
	NoVT        bool  // EnableOutputVT returns a nil restore, like a non-console output

	mu                   sync.Mutex
	raw, unraw, vt, unvt int
	lastMouse            bool
	resizes              chan struct{}
}

func (f *Fake) IsTerminal() bool { return f.Interactive }

func (f *Fake) Size() (int, int, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.W, f.H, f.SizeKnown
}

// Resizes implements tui.ResizeNotifier: it receives a value for each Resize.
func (f *Fake) Resizes() <-chan struct{} { return f.resizeChan() }

func (f *Fake) resizeChan() chan struct{} {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.resizes == nil {
		f.resizes = make(chan struct{}, 1)
	}
	return f.resizes
}

// Resize changes the size Size reports to w x h and notifies a Program
// watching Resizes, like a remote client resizing its window.
func (f *Fake) Resize(w, h int) {
	f.mu.Lock()
	f.W, f.H, f.SizeKnown = w, h, true
	f.mu.Unlock()
	select {
	case f.resizeChan() <- struct{}{}:
	default: // one is already pending; it will read the latest size
	}
}

// SetSize changes the size Size reports to w x h without notifying anyone:
// for a caller that delivers the tui.ResizeMsg itself.
func (f *Fake) SetSize(w, h int) {
	f.mu.Lock()
	f.W, f.H, f.SizeKnown = w, h, true
	f.mu.Unlock()
}

func (f *Fake) MakeRaw(mouse bool) (func() error, error) {
	if f.RawErr != nil {
		return nil, f.RawErr
	}
	f.mu.Lock()
	f.raw++
	f.lastMouse = mouse
	f.mu.Unlock()
	return func() error { f.mu.Lock(); f.unraw++; f.mu.Unlock(); return nil }, nil
}

func (f *Fake) EnableOutputVT() (func() error, error) {
	if f.VTErr != nil {
		return nil, f.VTErr
	}
	if f.NoVT {
		return nil, nil
	}
	f.mu.Lock()
	f.vt++
	f.mu.Unlock()
	return func() error { f.mu.Lock(); f.unvt++; f.mu.Unlock(); return nil }, nil
}

// Calls returns how many times raw mode was entered and left, and how many
// times output VT processing was enabled and restored.
func (f *Fake) Calls() (raw, unraw, vt, unvt int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.raw, f.unraw, f.vt, f.unvt
}

// LastMouse is the mouse argument of the latest MakeRaw.
func (f *Fake) LastMouse() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.lastMouse
}
