package tui

import (
	"context"
	"testing"
	"time"
)

type childMsg struct{ n int }
type parentMsg struct{ from childMsg }

func wrapChild(m Msg) Msg { return parentMsg{from: m.(childMsg)} }

func TestMapWrapsAPlainCmd(t *testing.T) {
	cmd := Map(func() Msg { return childMsg{1} }, wrapChild)
	if got := cmd(); got != (parentMsg{childMsg{1}}) {
		t.Errorf("got %#v", got)
	}
}

func TestMapNilCases(t *testing.T) {
	if Map(nil, wrapChild) != nil {
		t.Error("Map(nil) is not nil")
	}
	c := func() Msg { return childMsg{2} }
	if got := Map(c, nil)(); got != (childMsg{2}) {
		t.Errorf("Map(c, nil) changed the message: %#v", got)
	}
	if got := Map(func() Msg { return nil }, wrapChild)(); got != nil {
		t.Errorf("a nil message became %#v", got)
	}
	drop := Map(func() Msg { return childMsg{3} }, func(Msg) Msg { return nil })
	if got := drop(); got != nil {
		t.Errorf("f returning nil did not drop the message: %#v", got)
	}
}

func TestMapLeavesProgramInstructionsAlone(t *testing.T) {
	if _, ok := Map(Quit(), wrapChild)().(QuitMsg); !ok {
		t.Error("Quit was wrapped")
	}
	if _, ok := Map(Println("x"), wrapChild)().(printMsg); !ok {
		t.Error("Println was wrapped")
	}
	if _, ok := Map(EnterAltScreen(), wrapChild)().(modeMsg); !ok {
		t.Error("a mode Cmd was wrapped")
	}
}

func TestMapSeesThroughBatchAndSequence(t *testing.T) {
	a := func() Msg { return childMsg{1} }
	b := func() Msg { return childMsg{2} }
	batch := Map(Batch(a, b), wrapChild)().(batchMsg)
	if len(batch) != 2 || batch[0]() != (parentMsg{childMsg{1}}) || batch[1]() != (parentMsg{childMsg{2}}) {
		t.Errorf("batch not mapped: %v", batch)
	}
	seq := Map(Sequence(a, Quit()), wrapChild)().(sequenceMsg)
	if len(seq) != 2 || seq[0]() != (parentMsg{childMsg{1}}) {
		t.Errorf("sequence not mapped")
	}
	if _, ok := seq[1]().(QuitMsg); !ok {
		t.Error("Quit inside a sequence was wrapped")
	}
}

func TestMapSeesThroughContextCmds(t *testing.T) {
	cmd := Map(FromCtx(func(context.Context) Msg { return childMsg{4} }), wrapChild)
	if got := RunCmd(context.Background(), cmd); got != (parentMsg{childMsg{4}}) {
		t.Errorf("got %#v", got)
	}
	tick := Map(Tick(time.Millisecond, func(time.Time) Msg { return childMsg{5} }), wrapChild)
	if got := RunCmd(context.Background(), tick); got != (parentMsg{childMsg{5}}) {
		t.Errorf("Tick: got %#v", got)
	}
}

func TestMapSeesThroughEvery(t *testing.T) {
	cmd, cancel := Every(time.Millisecond, func(time.Time) Msg { return childMsg{6} })
	defer cancel()
	em := Map(cmd, wrapChild)().(everyMsg)
	if got := em.fn(time.Now()); got != (parentMsg{childMsg{6}}) {
		t.Errorf("got %#v", got)
	}
}

func TestMapAsWrapsOnlyThatType(t *testing.T) {
	type other struct{}
	wrap := func(c childMsg) Msg { return parentMsg{c} }
	if got := MapAs(func() Msg { return childMsg{7} }, wrap)(); got != (parentMsg{childMsg{7}}) {
		t.Errorf("T not wrapped: %#v", got)
	}
	if got := MapAs(func() Msg { return other{} }, wrap)(); got != (other{}) {
		t.Errorf("another type changed: %#v", got)
	}
	c := func() Msg { return childMsg{8} }
	if got := MapAs[childMsg](c, nil)(); got != (childMsg{8}) {
		t.Errorf("nil f changed the message: %#v", got)
	}
}

// mapModel starts a mapped Cmd in Init and records what Update receives.
type mapModel struct {
	got []Msg
}

func (m mapModel) Init() Cmd {
	return Map(Batch(
		Go(func(context.Context) Msg { return childMsg{9} }),
		func() Msg { return childMsg{10} },
	), wrapChild)
}

func (m mapModel) Update(msg Msg) (Model, Cmd) {
	if _, ok := msg.(parentMsg); ok {
		m.got = append(m.got, msg)
		if len(m.got) == 2 {
			return m, Quit()
		}
	}
	return m, nil
}

func (mapModel) View() string { return "" }

func TestMappedBatchReachesUpdateWrapped(t *testing.T) {
	_, _, wait := bgRun(t, mapModel{})
	got := wait().model.(mapModel).got
	if len(got) != 2 {
		t.Fatalf("Update received %d wrapped messages, want 2: %v", len(got), got)
	}
}
