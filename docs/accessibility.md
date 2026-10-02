# Accessibility

`tui.WithAccessible(true)` switches a Program to accessible output: an
append-only plain-text transcript that a screen reader can follow, with no
escape sequences. It is off by default. `tui.WithAccessibleAuto()` turns it on
when `ACCESSIBLE=1` or `TERM=dumb`. Under `TERM=dumb` it is also on by default
for a Program given neither `WithAltScreen` nor `WithAccessible`.
`Program.Accessible` reports whether it is on.

## What accessible mode changes

- No alternate screen, even with `WithAltScreen(true)`, and no cursor
  movement, line clearing, synchronized output, colour or other SGR styling, or
  OSC sequences.
- Each frame writes only the lines that differ from the previous frame, once
  each, in order. Lines that disappear aren't erased.
- When a line grows (the new line starts with the old one), only the new
  suffix is written. `WithLinearizeFullLine()` writes the whole line again
  instead.
- When the new frame is the previous one scrolled by a fixed number of rows,
  lines already written aren't written again. Otherwise every changed line is
  written, even if the same text appears elsewhere.
- Reduced motion is on, and `WithReducedMotion(false)` can't turn it off.
- The inspector, the capability probe and background detection are off.
- A resize doesn't repaint anything.

The text comes from the root model's `Linearize()` if it implements
`Linearizer`, and otherwise from `View()` with escape sequences removed.
`Linearize` should return one self-contained line per item, with state in
words rather than glyphs, and no padding or box drawing. Composite models
delegate to their children. Every stateful widget package has a `Linearize`
method; `TestEveryStatefulWidgetHasLinearizeAndLayoutNode` in
`internal/archtest` fails for a widget package that lacks one.

The Program doesn't own the theme, so it can't drop borders by itself. When
`Program.Accessible()` is true, give your widgets `theme.Theme.Plain()`, which
clears the border.

## Announcements

`tui.Announce(text)` returns a Cmd that reports a change that isn't otherwise
visible, such as "saved". `tui.AnnounceWith(text, tui.Assertive)` marks one as
urgent; `Announce` is `tui.Polite`.

In accessible mode, an announcement is written to the transcript as its own
line, with escape sequences removed. Polite ones are de-duplicated: the same
text within one second is written once. Assertive ones are always written,
prefixed `Alert: `.

Outside accessible mode, `Announce` does nothing unless one of these is set:

- `WithAnnounceRegion(rows)` reserves `rows` rows below the View that show the
  latest announcements for `tui.AnnounceTTL` (3 seconds) each. Inline, the rows
  are added below the View. On the alternate screen, the View is cut to make
  room.
- With `WithCapabilityProbe`, if the terminal supports OSC 99, Assertive
  announcements are also sent as desktop notifications, with or without a
  region.

## Reduced motion

`Program.ReducedMotion` reports the preference. It is `motion.Detect()`, which
is reduced when `NO_ANIMATION` or `REDUCE_MOTION` is non-empty, unless
`WithReducedMotion` set it. Accessible mode forces it on.

The Program doesn't enforce the preference, because it can't tell which Cmds
animate or which parts of a View are decoration. Your app reads it and passes
it on:

- The animated widgets (`spinner`, `skeleton`, `loadingbar`, `faces`,
  `streamtext`, `textinput`, `textarea`, `drawer`) have a `Motion` field of
  type `motion.Preference`. Its zero value is `motion.Normal`, so the
  environment alone doesn't change them. Set it to `motion.Reduced` and the
  widget is at its final frame as soon as it starts, and schedules no tick
  (`TestReduceMotionEnvEveryAnimatedWidget`).
- `theme.Theme.Plain()` drops decorative borders.

## Colour independence

No status-bearing widget relies on colour alone. `TestColourIndependence` sets
`NO_COLOR=1`, renders every state of each listed widget (alerts, badges,
checkboxes, toggles, toasts, progress and more), removes the escape sequences,
and fails if two states of one widget give the same text. A new widget whose
meaning is a state has to be added to that test by hand. `NO_COLOR` itself is
covered in [capabilities.md](capabilities.md#colour-profile).

## The hardware cursor

By default the terminal cursor is hidden. A screen magnifier or an IME needs
it to sit on the focused field. A root model that implements `CursorPlacer`
or `CursorProvider` gets the real cursor placed at the cell it names after
each frame. `textinput` and `textarea` implement `CursorProvider`, so a root
model can forward the focused field's `CursorCell`. The Program checks only
the root model and its `Unwrap` chain, not fields inside it. The cursor isn't
shown in accessible mode.
