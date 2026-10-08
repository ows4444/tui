// Package tui is a zero-dependency terminal UI framework built on the Elm
// Architecture (Model / Update / View), in the same spirit as Bubble Tea or
// Ink, using only the Go standard library.
//
// An application implements Model, hands it to NewProgram with the options it
// wants, and calls Run:
//
//	p := tui.NewProgram(myModel{}, tui.WithAltScreen(true))
//	if _, err := p.Run(); err != nil { ... }
//
// The Program reads keys, mouse events and resizes, calls Model.Update on each
// Msg, runs the Cmds Update returns, and redraws Model.View with a cell diff
// so only changed cells are written.
//
// Cmds returned together by Batch run concurrently and their messages reach
// Update in no particular order; use Sequence when order matters, which runs
// its Cmds one at a time and delivers their messages in order.
// WithMaxConcurrentCmds caps how many Cmds run at once.
//
// Package tui owns the event loop (stability: core). Terminal I/O is performed
// by package term and internal/termio, which tui drives. Package cellbuf, a
// retained grid of terminal cells for widgets that draw straight into cells,
// sits beside it (stability: stable-ish). The other packages are building
// blocks that tui and applications share, in layers that each import only the ones above them in this list (an import
// that points up is a test failure, see docs/architecture/overview.md):
//
// Primitives, with no dependency on the rest (stability: core):
//
//	ansi    styles, colour, escape encoding, width, graphemes, wrapping
//	layout  a Node tree with flex, grid and overlay layout
//	theme   colours, glyph sets and states as data
//	term    raw mode and terminal size
//	motion  springs, transitions and the reduced-motion preference
//
// Codecs and ports (core):
//
//	input   the byte-to-event decoder (keys, mouse, paste, focus)
//	keymap  key bindings for help text
//	hittest named click regions
//
// Input aliases (stability: core, by alias). The root package re-exports 14
// types from input as aliases: Key, KeyType, KeyAction, MouseEvent,
// MouseButton, MouseAction, PasteEvent, FocusEvent, ChordDef, ChordMsg,
// BackgroundColorEvent, PaletteColorEvent, ReplyEvent and BackgroundUnknownMsg.
// They are part of the root API, so each is exactly as stable as the input
// type it names: it changes only when input changes, never on its own.
//
// Renderers and test harness (core for tuitest, stable-ish for the rest):
//
//	widgets       stateless render helpers: Badge, Box, CodeBlock, ChatMessage, ...
//	widgets/chart Sparkline, BarChart, LineChart, HeatMap, Gauge
//	markdown      a CommonMark subset rendered to styled text
//	focus         a focus ring over components
//	tuitest       a headless harness that runs a model against a virtual screen
//
// Components, one package per stateful widget (textinput, viewport,
// datatable, ...), each a value-typed Model configured through exported
// fields. Most are stable-ish; the experimental ones are listed below and say so
// in their own package comment. The README lists every package by level.
//
// Experimental: appshell, streamtext, toolapproval, notificationcenter,
// commandpalette, errorretry, faces, imageview, clipboard, avatar, button.
//
// Terminals other than the local one (an SSH session, a test) plug in through
// the Terminal and Clock ports (WithTerminal, WithClock), plus ResizeNotifier
// for resizes; a wrapper model exposes the optional interfaces of the model it
// wraps with Unwrapper.
//
// docs/architecture/overview.md describes how the packages layer.
//
// Guides, a cookbook and screens of every example are at [tui.nizaami.com].
//
// [tui.nizaami.com]: https://tui.nizaami.com
package tui
