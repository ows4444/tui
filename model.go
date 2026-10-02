package tui

// Msg is any value delivered to Model.Update: a key press, a resize, a
// timer tick, or an application-defined event returned by a Cmd.
type Msg any

// Cmd is a unit of work run outside the render loop (I/O, timers, ...)
// whose result is fed back into Update as a Msg. Returning nil means
// "no side effect."
//
// A Cmd made by FromCtx, Tick or a motion wait needs the Program's context,
// so calling it directly does not wait: it returns an internal message that
// the Program recognises and runs. Hand such a Cmd to the Program (return it
// from Init or Update, or put it in Batch or Sequence), or call RunCmd, as a
// test does; do not wrap its result in a Msg of your own, which hides it from
// the Program. Printed, the internal message says so.
type Cmd func() Msg

// Model is the Elm-architecture contract every application implements.
type Model interface {
	// Init runs once, before the first render, to kick off any initial I/O.
	Init() Cmd
	// Update handles one Msg and returns the next Model state plus an
	// optional Cmd to run.
	Update(Msg) (Model, Cmd)
	// View renders the current state as a plain string; embedded ANSI
	// styling (via the ansi package) is fine, but there should be no
	// cursor-movement codes — the renderer owns cursor positioning.
	View() string
}

// CursorPlacer is an optional interface for a Model that wants the real
// terminal cursor shown, for an IME or a screen magnifier to anchor to. After
// each frame the Program asks it for a cell: x and y are 0-based, relative to
// the top-left of the View, and ok false keeps the cursor hidden, as it is for
// a Model that does not implement this. A position outside the View is clamped
// to it. The cursor is drawn by the terminal, so the View should not also draw
// a fake one at the same cell if the app wants only one.
type CursorPlacer interface {
	CursorPos() (x, y int, ok bool)
}

// CursorProvider is the optional interface of a focused widget, such as
// textinput or textarea, that knows its own cursor cell. A Model (or a model
// in its Unwrap chain) implementing it gets the hardware cursor placed with
// the same rules as CursorPlacer, so an IME anchors without app wiring. ok is
// false while the widget is not focused. CursorPlacer takes precedence when a
// model has both.
type CursorProvider interface {
	CursorCell() (x, y int, ok bool)
}

// Component is the documented contract for an embeddable widget: one
// composed inside another Model or Component, rather than run directly as
// a Program's root. Every stateful leaf widget in this library (textinput,
// viewport, datatable, and the rest) follows it structurally already; this
// interface just gives that existing convention a name and a place to
// assert conformance, instead of leaving it as something only visible by
// reading each widget's source.
//
// Two differences from Model:
//
//  1. No Init. A parent's own Init is responsible for initializing or
//     composing its children — an embeddable widget's "starting state" is
//     just whatever its constructor (conventionally New) returns.
//  2. Update returns T, the widget's own concrete type, not Component
//     itself. Go has no covariant return types, so a method that returned
//     Component literally couldn't also return the widget's own type for
//     callers that need to read a field back out of it (as every widget's
//     Update already does: `m, cmd := w.Update(msg)` keeps m as the
//     concrete widget type). Parameterizing on T keeps that natural,
//     already-universal style, while still making the shape explicit.
//
// A widget package proves it satisfies this contract by declaring, once,
// something like:
//
//	var _ tui.Component[Model] = Model{}
//
// which fails to compile the moment Update or View's signature drifts from
// this shape — real verification, not just a comment asserting it.
type Component[T any] interface {
	Update(Msg) (T, Cmd)
	View() string
}

// Overlay is the documented contract for a widget that renders by
// compositing onto an existing frame rather than standing alone —
// dialog.Model, drawer.Model, helpscreen.Model, popover.Model, and
// toast.Model all follow it structurally already, independently arrived
// at before this interface existed; Overlay just gives that convention a
// name and a place to assert conformance, the same way Component did for
// the plain embeddable-widget shape.
//
// Update returns T, not Overlay itself, for the same covariant-return-type
// reason Component's doc comment explains.
//
// Show and Hide (which flip Open's backing state) are deliberately NOT
// part of this interface, for two reasons: every implementation uses a
// pointer receiver for them (*Model, so they can mutate in place), which
// puts them outside a value type's method set and therefore outside an
// interface satisfied by a value the way Open/Update/Render are; and
// toast.Model.Show additionally returns a Cmd (it kicks off an
// auto-dismiss timer) where every other implementation's Show returns
// nothing — a real, legitimate difference this interface doesn't try to
// paper over by forcing a shape that doesn't fit. Open/Update/Render are
// this contract's actual common shape: enough for a caller to check
// visibility and composite an overlay onto a base frame generically,
// without knowing which concrete overlay it's holding.
//
// A widget package proves it satisfies this contract by declaring, once,
// something like:
//
//	var _ tui.Overlay[Model] = Model{}
type Overlay[T any] interface {
	Open() bool
	Update(Msg) (T, Cmd)
	Render(base string) string
}

// Linearizer is an optional interface for a root Model. When the Program
// runs with WithAccessible(true) it renders Linearize() instead of View():
// plain text, one self-contained line per item, state spoken in words
// rather than glyphs, no alignment padding or box drawing. A Model that
// doesn't implement it falls back to View() with ANSI sequences stripped.
// Composite models delegate to their children (e.g. datatable.Model and
// treeview.Model each have a Linearize method).
type Linearizer interface {
	Linearize() string
}
