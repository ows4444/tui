package edit

import (
	"strings"
	"testing"
)

func ed(s string) *Editor { var e Editor; e.Set(s, 0); return &e }

// #70: undo after typing, deleting and pasting restores value and cursor.
func TestUndoRestoresValueAndCursor(t *testing.T) {
	cases := map[string]func(e *Editor){
		"insert":    func(e *Editor) { e.Insert("XY", 0) },
		"backspace": func(e *Editor) { e.Backspace() },
		"delete":    func(e *Editor) { e.Delete() },
		"word":      func(e *Editor) { e.DeleteWordBack() },
		"toStart":   func(e *Editor) { e.DeleteToStart() },
		"toEnd":     func(e *Editor) { e.DeleteToEnd() },
		"paste":     func(e *Editor) { e.Insert("pasted text 😀", 0) },
		"replace":   func(e *Editor) { e.SetAnchor(); e.SetCursor(9); e.ReplaceSelection("Q", 0) },
	}
	for name, op := range cases {
		e := ed("hello world")
		e.SetCursor(6)
		op(e)
		if e.String() == "hello world" {
			t.Fatalf("%s: op changed nothing", name)
		}
		after, afterCur := e.String(), e.Cursor()
		if !e.Undo() {
			t.Fatalf("%s: nothing to undo", name)
		}
		wantCur := 6
		if name == "replace" {
			wantCur = 9 // the cursor was at the selection's end
		}
		if e.String() != "hello world" || e.Cursor() != wantCur {
			t.Errorf("%s: undo = %q cursor %d", name, e.String(), e.Cursor())
		}
		if !e.Redo() || e.String() != after || e.Cursor() != afterCur {
			t.Errorf("%s: redo = %q cursor %d, want %q cursor %d", name, e.String(), e.Cursor(), after, afterCur)
		}
		if e.Redo() {
			t.Errorf("%s: second redo succeeded", name)
		}
	}
}

func TestTypingRunIsOneStep(t *testing.T) {
	e := ed("")
	for _, r := range "hello" {
		e.Insert(string(r), 0)
	}
	if e.HistoryLen() != 1 {
		t.Fatalf("typing run made %d steps", e.HistoryLen())
	}
	e.Insert(" ", 0)
	e.Insert("w", 0)
	e.Undo()
	if e.String() != "hello" {
		t.Fatalf("after one undo %q", e.String())
	}
	e.Undo()
	if e.String() != "" || e.Cursor() != 0 || e.Undo() {
		t.Fatalf("after two undos %q cursor %d", e.String(), e.Cursor())
	}
}

func TestNewEditClearsRedo(t *testing.T) {
	e := ed("ab")
	e.Backspace()
	e.Undo()
	e.Insert("z", 0)
	if e.CanRedo() {
		t.Fatal("redo survived a new edit")
	}
}

func TestSetAndResetForgetHistory(t *testing.T) {
	e := ed("ab")
	e.Backspace()
	e.Set("x", 0)
	if e.CanUndo() {
		t.Fatal("Set kept history")
	}
	e.Backspace()
	e.Reset()
	if e.CanUndo() || e.CanRedo() {
		t.Fatal("Reset kept history")
	}
}

// #71: history is bounded by a configurable step count.
func TestHistoryBounded(t *testing.T) {
	for _, limit := range []int{1, 3, 10} {
		e := ed(strings.Repeat("x", 100))
		e.SetHistoryLimit(limit)
		for i := 0; i < 50; i++ {
			e.Backspace()
		}
		if e.HistoryLen() != limit {
			t.Errorf("limit %d: HistoryLen = %d", limit, e.HistoryLen())
		}
		n := 0
		for e.Undo() {
			n++
		}
		if n != limit {
			t.Errorf("limit %d: %d undos", limit, n)
		}
		if got := len(e.String()); got != 50+limit {
			t.Errorf("limit %d: len after undos %d", limit, got)
		}
	}
}

func TestHistoryMemoryBounded(t *testing.T) {
	var h History
	for i := 0; i < 10000; i++ {
		h.Record(Step{Pos: i, Ins: "x"}, false, 5)
	}
	if h.undo.depth > 10 {
		t.Fatalf("chain holds %d nodes for limit 5", h.undo.depth)
	}
	if h.Len() != 5 {
		t.Fatalf("Len = %d", h.Len())
	}
}

func TestDefaultAndDisabledLimit(t *testing.T) {
	e := ed(strings.Repeat("x", 300))
	for i := 0; i < 200; i++ {
		e.Backspace()
	}
	if e.HistoryLen() != DefaultHistory {
		t.Fatalf("default bound = %d", e.HistoryLen())
	}
	e.SetHistoryLimit(-1)
	if e.CanUndo() {
		t.Fatal("disabled history kept steps")
	}
	e.Backspace()
	if e.CanUndo() {
		t.Fatal("disabled history recorded")
	}
}

func TestUndoCopyIsolation(t *testing.T) {
	e := ed("hello")
	e.Backspace()
	snap := *e
	e.Undo()
	e.Backspace()
	e.Backspace()
	if snap.String() != "hell" || !snap.CanUndo() {
		t.Fatalf("copy changed: %q", snap.String())
	}
	snap.Undo()
	if snap.String() != "hello" {
		t.Fatalf("copy undo = %q", snap.String())
	}
}

func TestUndoClusterBoundaries(t *testing.T) {
	e := ed("éa")
	e.SetCursor(1)
	e.Backspace()
	if e.String() != "a" {
		t.Fatalf("%q", e.String())
	}
	e.Undo()
	if e.String() != "éa" || e.Cursor() != 1 || e.Len() != 2 {
		t.Fatalf("undo = %q cursor %d len %d", e.String(), e.Cursor(), e.Len())
	}
}

func TestSelectionBasics(t *testing.T) {
	e := ed("hello world")
	e.SetCursor(0)
	e.SetAnchor()
	e.SetCursor(5)
	if lo, hi, ok := e.Selection(); !ok || lo != 0 || hi != 5 || e.SelectedText() != "hello" {
		t.Fatalf("selection %d %d %v %q", lo, hi, ok, e.SelectedText())
	}
	// #73: replacing the selection.
	e.ReplaceSelection("J", 0)
	if e.String() != "J world" || e.Cursor() != 1 {
		t.Fatalf("replace = %q cursor %d", e.String(), e.Cursor())
	}
	e.Undo()
	if e.String() != "hello world" || e.Cursor() != 5 {
		t.Fatalf("undo replace = %q cursor %d", e.String(), e.Cursor())
	}
	e.SelectAll()
	if e.SelectedText() != "hello world" {
		t.Fatal("SelectAll")
	}
	e.Backspace()
	if e.String() != "" {
		t.Fatalf("backspace over selection left %q", e.String())
	}
	e.Undo()
	if e.String() != "hello world" {
		t.Fatalf("undo = %q", e.String())
	}
}

func TestInsertKeepsLimitWithSelection(t *testing.T) {
	e := ed("abcd")
	e.SetCursor(1)
	e.SetAnchor()
	e.SetCursor(3)
	e.ReplaceSelection("XYZ", 4)
	if e.String() != "aXYd" {
		t.Fatalf("limit replace = %q", e.String())
	}
}

func TestCopyWritesOSC52(t *testing.T) {
	var got string
	Copy(func(s string) (int, error) { got += s; return len(s), nil }, "hi")
	if got != "\x1b]52;c;aGk=\x07" || Clip() != "hi" {
		t.Fatalf("got %q clip %q", got, Clip())
	}
}

// Recording a typing keystroke stays within the allocation budget.
func TestRecordAllocsAmortized(t *testing.T) {
	var h History
	i := 0
	n := testing.AllocsPerRun(1000, func() {
		h.Record(Step{Pos: i, Del: 1}, true, 0)
		i++
	})
	if n > 1.5 {
		t.Fatalf("%.2f allocs per coalesced record", n)
	}
	n = testing.AllocsPerRun(1000, func() {
		h.Record(Step{Pos: i, Ins: ""}, false, 0)
		i += 5
	})
	if n > 2.1 {
		t.Fatalf("%.2f allocs per record", n)
	}
}
