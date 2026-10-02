// Package widgets holds small, stateless, presentational render helpers —
// Divider, Badge, StatusIndicator, KeyHint, ProgressBar, CodeBlock, DiffView, TokenCounter — bundled into one
// package rather than one apiece since each is a handful of lines with no
// internal state.
//
// This is a deliberately different shape from textinput/viewport: those
// are interactive (they implement Update(tui.Msg) (Model, tui.Cmd) and
// hold state like a cursor or a scroll offset), so they depend on tui and
// live in their own packages. Everything here is a pure function of its
// arguments to a string, needs no Msg handling, and never imports tui or a
// stateful component package: it sits below the components, not beside them.
package widgets

//
// What belongs here: a function that returns a string for a view to place, is
// a function of its arguments alone, and is chrome or content rather than a
// data plot: labels and containers (Badge, Box, Card, Alert), separators,
// progress and status, text formatting (CodeBlock, DiffView, Gradient, BigText),
// and the chat and agent helpers (ChatMessage, TokenCounter, UsageMonitor).
// Charts that turn numbers into marks live in widgets/chart. Anything that
// needs Update, a cursor or a scroll offset is a component in its own package.
