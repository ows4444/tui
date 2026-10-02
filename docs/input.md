# Input, key bindings, focus and mouse

The Program decodes terminal bytes with an
[`input.Reader`](https://pkg.go.dev/github.com/ows4444/tui/input#Reader) and
passes each event to `Update` as a message: `tui.Key`, `tui.MouseEvent`,
`tui.PasteEvent`, `tui.FocusEvent`, and the others listed below. These root
types are aliases of the `input` types, so `tui.Key` and `input.Key` are the
same type. The `KeyType`, `MouseButton` and `MouseAction` constants are
re-exported too (`tui.KeyEnter`, `tui.MouseButtonLeft`). The modifier bits are
not: use `input.ModCtrl`, `input.ModShift` and so on.

```go
func (c counter) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	if k, ok := msg.(tui.Key); ok && k.Type == tui.KeyRunes {
		switch k.Text {
		case "+":
			c.n++
		case "q":
			return c, tui.Quit()
		}
	}
	return c, nil
}
```

From the `counter` model of `ExampleNewProgram` in `example_test.go`.

Program options (`WithMouse`, `WithKeyboard`, ...) are covered in
[Program](program.md). This page covers handling the events.

## Keys

A [`Key`](https://pkg.go.dev/github.com/ows4444/tui/input#Key) has a `Type`.
Printable input is `KeyRunes`, with the typed text in `Key.Text`. Ctrl+C is
`KeyCtrlC`. Other Ctrl+letter keys are `KeyCtrl`, with the letter in
`Key.Code`. Modifiers on arrows, Home/End, PgUp/PgDn and Alt+rune arrive in
`Key.Mod`. `Key.String()` gives the name used everywhere else in the library
(`"a"`, `"up"`, `"ctrl+right"`, `"enter"`):

```go
rd := input.NewReader(strings.NewReader("a\x1b[A\x1b[1;5C"))
for {
	ev, err := rd.ReadEvent()
	if err != nil {
		break
	}
	fmt.Println(ev.(input.Key))
}
// Output:
// a
// up
// ctrl+right
```

From `ExampleReader` in `input/example_test.go`. The Reader is a plain decoder
over an `io.Reader`, so it needs no TTY.

A lone ESC waits `input.DefaultEscTimeout` (30ms) for the rest of a sequence
before it becomes `KeyEsc`. Change this with `tui.WithEscTimeout`.

The legacy encoding cannot tell Shift+Enter from Enter.
`tui.WithKittyKeyboard(true)` turns on the kitty keyboard protocol's
"disambiguate" flag, and terminals without the protocol ignore it.
`tui.WithKeyboard(tui.KeyboardReportEvents)` also reports repeats and releases.
They arrive as `tui.KeyRepeatMsg` and `tui.KeyReleaseMsg`, never as `Key`, so
`case tui.Key` still sees only presses.

## Paste and terminal focus

Bracketed paste is on by default. A paste arrives as one `tui.PasteEvent` with
the whole text, not as keys, so a pasted newline is not an Enter press. One
event holds at most `input.MaxPasteBytes` (16 MiB). `Truncated` is set if the
paste was longer. If the end marker does not arrive within
`input.PasteIdleTimeout` (250ms) of the last byte, the event carries what was
received and `Incomplete` is set. `tui.WithBracketedPaste(false)` turns this
off.

`tui.WithFocusReporting(true)` sends a `tui.FocusEvent{Focused: bool}` when the
terminal window gains or loses focus. Only changes are delivered. This is the
window's focus, not focus between widgets, which is covered
[below](#focus-between-widgets).

## Key bindings

[`keymap`](https://pkg.go.dev/github.com/ows4444/tui/keymap) keeps a widget's
keys and its help text in one place. A `keymap.Binding` is a description plus
key names (the `Key.String()` forms). `keymap.Matches(msg, b)` reports whether
`msg` is a press of one of them. Releases and disabled bindings never match.
Repeats count as presses.

Widgets that react to keys follow one convention. The package has a `KeyMap`
struct with one `Binding` per action, `DefaultKeyMap()`, a `KeyMap` field on
`Model`, and a `Bindings()` method that returns the bindings in effect. To
rebind a widget, change its field, for example `m.KeyMap.Down`. The behaviour
and the help text change together. The packages that follow it are marked in
the [widget catalog](widgets.md#component-packages).

A `keymap.Registry` collects bindings for help screens and reports two actions
bound to the same key in one scope:

```go
var r keymap.Registry
r.Add(keymap.NewBinding("quit", "q"))
conflicts, _ := r.Add(keymap.NewBinding("close", "q"))
fmt.Println(len(conflicts))
for _, h := range r.Hints("") {
	fmt.Println(h.Key, h.Desc)
}
// Output:
// 1
// q quit
// q close
```

From `Example` in `keymap/example_test.go`.

To build help from what is actually bound, implement `tui.BindingsProvider` on
your root model by returning the focused widget's `Bindings()` and your own.
`(*tui.Program).Keymap()` then returns a fresh `Registry` built from it, as of
the last frame drawn. Pass that registry to `helpscreen.FromRegistry` or
`widgets.HintsFromKeymap`. Build the help screen when it opens, so it follows
the focus.

## Chords

`tui.WithChords(defs...)` recognises multi-key sequences such as `g g` or
`ctrl+x ctrl+s`. Each is a `tui.ChordDef{Name, Keys}`, with `Keys` in
`Key.String()` form. When the keys complete a chord, `Update` receives one
`tui.ChordMsg{Name}` instead of the keys. A key that cannot start or continue a
chord is delivered at once. A prefix that turns out not to be a chord is
delivered as its original keys, in order. So is a prefix still waiting when the
timeout passes, even if no other key is typed. A key that can start a chord is
therefore held back for up to the timeout: `input.DefaultChordTimeout` (500ms),
changed with `tui.WithChordTimeout`. `chords_test.go` covers each case.

Outside a Program, `input.NewChordMatcher` does the same matching. You call
`Feed` for each `Key` and `Expire` from a timer. The repository has no Example
for chords.

## Focus between widgets

A [`focus.Ring`](https://pkg.go.dev/github.com/ows4444/tui/focus#Ring) tracks
which of n items has focus. Tab and Shift+Tab move it, wrapping at the ends and
skipping disabled items. Like the widget Models it is a value: every method
returns a new Ring. `Ring.Route` does the whole dispatch. On Tab and Shift+Tab it
blurs the old item and focuses the new one. It sends every other message to the
focused item only.

```go
func (m *model) fields() []focus.Field {
	return []focus.Field{
		focus.Bind(&m.name),
		focus.Bind(&m.password.Model), // passwordinput embeds a textinput.Model
		focus.Bind(&m.bio),
	}
}
```

```go
// Route does the whole dispatch: Tab and Shift+Tab (the input reader
// decodes both ESC [ Z and the kitty form to a Tab key with the Shift
// modifier) blur the old widget and focus the next one, wrapping in both
// directions; every other message goes to the focused widget only.
var cmd tui.Cmd
m.ring, cmd = m.ring.Route(msg, m.fields()...)
return m, cmd
```

From `examples/focus/main.go`, the reference program for focus.

- `focus.Bind(&w)` works for any widget whose pointer has `Focus() tui.Cmd` and
  `Blur()` (`focus.Focusable`). Bind fields of the model value you are about to
  return. The pointers must point into that copy.
- Call `ring.Sync(fields...)` once at start-up and return its Cmd from `Init`.
  Call it again after `Set` or `SetDisabled`.
- `Route` does not broadcast. Forward messages every widget needs, such as
  blink ticks for an unfocused field, yourself.
- `ring.Push(n)` opens a scope that traps Tab inside a modal. `Pop` restores
  the previous scope and its focused item.
- `ring.WithOrder(focus.LayoutOrder(root, size, names...))` makes Tab follow
  screen position rather than index order (`ExampleLayoutOrder` in
  `focus/order_test.go`).
- `ring.RouteAuto(msg, focus.Zones(root, size, names...), fields...)` also
  focuses the item under a left click. It also lets an item keep Tab for itself
  through `Field.Consumes`. `examples/settings` uses it.

## Mouse and hit-testing

Mouse reporting is off by default. Turn it on with
`tui.WithMouse(tui.MouseClick)` (press and release),
`tui.MouseCellMotion` (also drags) or `tui.MouseAllMotion` (all movement).
`tui.EnableMouse(mode)` returns a Cmd that switches it at run time. A
`tui.MouseEvent` has 0-indexed cell coordinates `X`, `Y`, a `Button` (the wheel
is four buttons), an `Action` and a `Mod`.

[`hittest`](https://pkg.go.dev/github.com/ows4444/tui/hittest) answers which
region a cell is in. A `hittest.Map[ID]` is an ordered set of rectangles. Later
additions win where they overlap. Take the rectangles from the layout rather
than computing them:

```go
ui := layout.Row(1,
	layout.FlexChild{Node: layout.Named("list", layout.Block("one\ntwo\nthree")), Basis: 8},
	layout.FlexChild{Node: layout.Named("detail", layout.Block("details")), Grow: 1},
)
size := layout.Size{W: 30, H: 5}

var m hittest.Map[string]
for _, p := range layout.Rects(ui, size) {
	if p.Name != "" {
		m = m.Add(p.Name, p.Rect)
	}
}

for _, click := range [][2]int{{2, 1}, {20, 4}} {
	if h, ok := m.At(click[0], click[1]); ok {
		fmt.Printf("click %v -> %s at %d,%d\n", click, h.ID, h.LX, h.LY)
	}
}
// Output:
// click [2 1] -> list at 2,1
// click [20 4] -> detail at 11,4
```

From `Example_fromLayout` in `hittest/layout_test.go`.

`hittest.HitMap(root, size)` builds the same map in one call, keyed by name.
`Map.AtEvent(ev)` tests a `MouseEvent` directly, and you decide which actions
count. `layout.Rects` coordinates are relative to the root's top-left. If the
root is not drawn at the screen origin, add its offset. In inline mode
(`WithAltScreen(false)`) mouse coordinates are absolute in the terminal, so
offset your rectangles by where the live region starts.

## Clipboard

[`clipboard.Model`](https://pkg.go.dev/github.com/ows4444/tui/clipboard) is a
button that copies its `Text` to the system clipboard with an OSC 52 sequence
on Enter or Space, then shows "Copied!" for `Timeout` (2 seconds from `New`).
With `Mouse` set, a left click inside `Bounds` activates it too. The package is
experimental.

```go
m := clipboard.New("secret-token", "Copy")
var sent string
m.Write = func(s string) (int, error) { sent = s; return len(s), nil }
m, _ = m.Update(tui.Key{Type: tui.KeyEnter})
fmt.Println(m.Copied(), strings.HasPrefix(sent, "\x1b]52;"))
// Output:
// true true
```

From `Example` in `clipboard/example_test.go`. `Write` defaults to
`os.Stdout.WriteString`. The example replaces it to capture the bytes.

OSC 52 is write-only and best-effort, and the program cannot detect failure.
The terminal must support it. Under tmux, `set-clipboard` must be `on` or
`external`. If it is `off`, the button still shows "Copied!" but nothing reaches
the clipboard.
