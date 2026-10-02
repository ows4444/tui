package edit

import "testing"

// Copies of an Editor must not observe later mutations of the original.
func TestCopyIsolation(t *testing.T) {
	ops := map[string]func(e *Editor){
		"Backspace":     func(e *Editor) { e.Backspace() },
		"Delete":        func(e *Editor) { e.Delete() },
		"DeleteWord":    func(e *Editor) { e.DeleteWordBack() },
		"DeleteToStart": func(e *Editor) { e.DeleteToStart() },
		"DeleteToEnd":   func(e *Editor) { e.DeleteToEnd() },
		"Insert":        func(e *Editor) { e.Insert("XY", 0) },
		"InsertLimit":   func(e *Editor) { e.Insert("XY", 11) },
		"Set":           func(e *Editor) { e.Set("zzz", 0) },
		"Reset":         func(e *Editor) { e.Reset() },
	}
	for name, op := range ops {
		var e Editor
		e.Set("hello world", 0)
		e.SetCursor(6)
		snap := e
		op(&e)
		if snap.String() != "hello world" || snap.Cursor() != 6 || snap.Width() != 11 {
			t.Errorf("%s: copy changed: %q cursor %d width %d", name, snap.String(), snap.Cursor(), snap.Width())
		}
	}
}
