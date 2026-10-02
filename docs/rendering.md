# Rendering

After each `Update`, the Program calls `View` and writes only the difference
from the frame already on screen. The default renderer is the cell renderer.
It parses each frame into a grid of cells and rewrites only the cells that
changed. `WithCellRenderer(false)` selects the line renderer, which rewrites
every row that changed.

You don't call the renderer yourself. This page covers what it does with a
View, when it skips a frame, and the tools for seeing what it wrote.

## From View to terminal

For each frame the Program:

1. Calls `View` (or, for a `CellDrawer` root, `DrawCells`; see below).
2. Draws the inspector overlay (`WithInspector`) and the announcement rows
   (`WithAnnounceRegion`) over or under the View, if they're on.
3. Rewrites colours to the Program's colour profile, and flattens styled
   underlines when the capability probe found no support
   ([capabilities.md](capabilities.md)).
4. Splits the View into lines. With `WithBidi(true)` it reorders each line
   for display.
5. Fits the lines to the terminal: expands tabs, cuts lines wider than the
   terminal, and on the alternate screen drops rows below the last one.
   Inline, rows that no longer fit go into scrollback once
   ([program.md](program.md#inline-or-alternate-screen)).
6. Diffs against the previous frame, with relative cursor movement from the
   top of the live region.
7. Wraps the write in synchronized output (mode 2026), so the terminal paints
   the frame at once. A terminal without mode 2026 ignores the brackets. With
   `WithCapabilityProbe`, the brackets are left out when the terminal reports
   no support.
8. Shows the hardware cursor at the cell a `CursorPlacer` or `CursorProvider`
   asked for, if any. Otherwise the cursor stays hidden.

All terminal writes (frames, `Println`, mode changes, the restore sequence) go
through one mutex, and nothing is written after the terminal has been
restored.

## When a frame is skipped or forced

- If `View` returns the same string as the last frame, nothing is written.
  If the cursor position changed, only the cursor moves. A `CellDrawer` root
  has no string to compare, so it is diffed every time.
- Without `WithMaxFPS`, messages that don't come from input (ticks, Cmd
  results, `Send`) are drawn at most 60 times a second. Keys, mouse, paste,
  focus and chords draw at once. `Update` still runs for every message, and a
  frame the cap skipped is drawn when the interval ends. `WithMaxFPS(n)` with
  `n > 0` caps every message, input included. `WithMaxFPS(0)` draws every
  message. `WithRecorder` and `WithRecorderSidecar` turn the cap off.
- A resize, a `Suspend` or Ctrl+Z resume, and `ClearScreen` repaint the whole
  frame instead of diffing, because the terminal may have cropped or re-wrapped
  what was there.
- `Println` and the quit frame are never delayed by the cap.

## Cell renderer and line renderer

The two renderers produce the same screen. `TestCellRendererEquivalentToLineRenderer`
and `FuzzRendererEquivalence` (in the root package) drive both with the same
frames and compare every cell, its style and the cursor row.
There is one difference: in the cell renderer each row's style stands alone, so
a style that a View row leaves open doesn't carry into the next row.

The cell renderer falls back in two ways:

- **A row it can't represent** (a control character, an unknown escape or
  SGR code) is drawn alone with the line strategy. The rest of the frame stays
  a cell frame.
- **A whole frame** goes to the line renderer in a few cases. One is a View
  taller than the terminal. The frame log names the reason.

The current godoc of `WithCellRenderer` lists tabs, control characters and
non-SGR escapes as whole-frame fallbacks. In the current code, tabs are
expanded before rendering, and the other two trigger only the per-row
fallback (`TestCellRendererFallsBackForUnsupportedRowOnly`).

## Drawing cells directly

A root Model that implements `CellDrawer` draws each frame straight into a
`cellbuf.Buffer` the size of the terminal, so no View string is built or
parsed. `DrawView` and `DrawChild` compose children that only have a View.
`View` is still required and should show the same screen, because the Program
uses it whenever the direct path is unavailable:

- with `WithCellRenderer(false)` or in accessible mode
- while an inspector pane is showing, or with an announcement region
- when the colour profile is below TrueColor, or the probe found no styled
  underlines
- when the terminal size is unknown
- once an inline frame has scrolled rows into scrollback

`examples/canvas` is a `CellDrawer` widget.

## Frame budget

Tests in the root package hold the cell renderer to a budget for a steady-state
200x60 frame of styled rows that all change (`BenchmarkFrame200x60Styled`):

| Test | Limit |
| --- | --- |
| `TestFrame200x60StyledAllocsCriterion34` | at most 7 allocations per frame |
| `TestFrame300x80StyledAllocs` | at most 8 allocations per frame at 300x80 |
| `TestFrame200x60StyledTime` | at most 60µs per frame; runs only with `TUI_TIMING_TESTS` set |

The allocation tests skip under the race detector.

## Seeing what was drawn

### Frame log

`WithFrameLog(w)` writes one line per frame to `w`:

```text
n=7 t=1042 kind=diff rows=24 changed=2 bytes=118 us=61
```

`kind` is `diff`, `full` (no previous frame: the first frame, after a resize,
`Suspend` or `Println`), `accessible`, or `fallback`, which adds
`reason=<slug>`. A frame with rows drawn by the per-row fallback adds
`fallback_rows=<row>:<slug>,...`. The log doesn't change what reaches the
terminal, and write errors are ignored. Don't point it at the terminal's own
output. The `WithFrameLog` godoc defines each field.

### Inspector

`WithInspector(tui.InspectorKeys{})` turns on four overlay panes, each toggled
by a key that never reaches `Update`:

| Key | Pane |
| --- | --- |
| F12 | Info panel: named layout rectangles (`LayoutInspector`), the focused id (`FocusInspector`), the last frame's bytes and draw time |
| F11 | Outlines of every named layout rectangle |
| F10 | The last 200 messages `Update` received, with how long each took. Recorded only while the pane is on |
| F9 | A `%#v` dump of the root Model |

Set a field of `InspectorKeys` to another key name, or to `tui.InspectorOff`
to leave that pane out. The inspector is ignored in accessible mode.

### Recording and replay

`WithRecorder(w)` writes the session as an asciicast v2 file that plays in
asciinema. `WithRecorderSidecar(w)` writes the input bytes, sizes, `Every`
ticks and a hash of every write. `tuitest.Replay` reads the pair and fails a
test if any write differs. Messages from `Send`, and Cmd results that depend on
the outside world, aren't recorded, so a model that needs them won't replay.
The recording contains input as typed, passwords included.

### Logging

The terminal belongs to the Program, so don't log to stdout or stderr while it
runs. `tui.LogToFile(path, prefix)` points the standard `log` package at a file
(created, or appended to) and returns it for you to close.
