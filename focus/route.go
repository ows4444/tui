package focus

import "github.com/ows4444/tui"

// Focusable is what a widget offers so a Ring can move focus onto and off it:
// Focus gives it focus (returning any Cmd it needs, such as a cursor blink)
// and Blur takes focus away. textinput, passwordinput, textarea and the other
// input widgets satisfy it through a pointer to their Model.
type Focusable interface {
	Focus() tui.Cmd
	Blur()
}

// Field is one ring item as the Ring drives it: how to focus it, blur it, and
// hand it a message. Build one with Bind, or fill the funcs yourself for a
// widget that does not follow the Model conventions. Nil funcs are skipped.
type Field struct {
	Focus  func() tui.Cmd
	Blur   func()
	Update func(tui.Msg) tui.Cmd
	// Consumes reports whether the item wants msg for itself, so RouteAuto
	// gives it the message instead of moving focus (a Tab that indents, a
	// completion popup). Nil means the item never consumes anything.
	Consumes func(tui.Msg) bool
}

// Bind returns the Field for the widget p points to, typically a field of
// your model:
//
//	fields := []focus.Field{focus.Bind(&m.name), focus.Bind(&m.bio)}
//
// Focus and Blur are p's methods, and Update runs p's Update and stores the
// new Model back through p, returning its Cmd. Bind the fields of the model
// value you are about to return, inside Update (models are values, so the
// pointer must point into the copy you keep).
func Bind[M interface {
	Update(tui.Msg) (M, tui.Cmd)
}, P interface {
	*M
	Focusable
}](p P) Field {
	return Field{
		Focus: p.Focus,
		Blur:  p.Blur,
		Update: func(msg tui.Msg) tui.Cmd {
			next, cmd := (*p).Update(msg)
			*p = next
			return cmd
		},
	}
}

// Route is the whole focus dispatch for a screen of several widgets. On Tab
// and Shift+Tab it moves focus to the next or previous enabled item, calls
// Blur on the item that had it and Focus on the item that gets it, and returns
// the Focus Cmd. Every other message goes to the focused item's Update only
// and its Cmd is returned; a message the focused item does not want costs
// nothing. Route never touches a disabled item or an index outside fields.
//
//	ring, cmd := m.ring.Route(msg, fields...)
//	m.ring = ring
//	return m, cmd
//
// fields must have one entry per ring item, in order. Messages that every
// widget needs, such as cursor blink ticks for an unfocused field, are not
// broadcast: forward those yourself.
func (r Ring) Route(msg tui.Msg, fields ...Field) (Ring, tui.Cmd) {
	if next, moved := r.Update(msg); moved {
		if next.cur != r.cur {
			blur(fields, r.cur)
			return next, focusField(fields, next.cur)
		}
		return next, nil
	}
	if f, ok := fieldAt(fields, r.Current()); ok && f.Update != nil {
		return r, f.Update(msg)
	}
	return r, nil
}

// Sync makes the fields agree with the Ring: it blurs every item but the
// focused one and focuses that one, returning its Focus Cmd. Call it once at
// start-up (from the initial model, returning the Cmd from Init) and after
// SetDisabled or Set changes which item has focus.
func (r Ring) Sync(fields ...Field) tui.Cmd {
	cur := r.Current()
	for i := range fields {
		if i != cur {
			blur(fields, i)
		}
	}
	return focusField(fields, cur)
}

func fieldAt(fields []Field, i int) (Field, bool) {
	if i < 0 || i >= len(fields) {
		return Field{}, false
	}
	return fields[i], true
}

func blur(fields []Field, i int) {
	if f, ok := fieldAt(fields, i); ok && f.Blur != nil {
		f.Blur()
	}
}

func focusField(fields []Field, i int) tui.Cmd {
	if f, ok := fieldAt(fields, i); ok && f.Focus != nil {
		return f.Focus()
	}
	return nil
}
