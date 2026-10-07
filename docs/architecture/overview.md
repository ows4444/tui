# Architecture overview

tui is a terminal UI framework on the Elm Architecture: an application is a
Model with `Init`, `Update` and `View`, and the root package's Program turns
terminal input into messages for `Update` and turns `View` into the fewest
bytes that bring the screen up to date. The module imports only the Go
standard library. Everything else is built on that runtime in layers: text
and colour primitives below it, stateless renderers and stateful widgets
above it.

## Bird's-eye view

```text
terminal bytes
  -> input.Reader (reader goroutine)        decode keys, mouse, paste, focus, replies
  -> Program message queue (64)             also fed by Cmd goroutines, Send, timers, resize watcher
  -> event loop (one goroutine)             intercepts runtime messages, calls Model.Update,
                                            starts the returned Cmd on its own goroutine
  -> Model.View  or  CellDrawer.DrawCells
  -> frame transforms                       inspector, announce region, colour downgrade, fit, bidi
  -> frame renderer                         internal/render cell diff, or the line renderer
  -> output writer (one mutex)              wrapped in synchronized output
```

Terminal control (size, raw mode, output VT processing) doesn't go through
the byte stream. It goes through the `Terminal` port, implemented by
`internal/termio`, which calls package `term`.

## Entry points

- `tui.NewProgram(model, opts...)` and `Program.Run`: the API every app uses.
- `Model`, `Msg`, `Cmd` (`model.go`): the application contract.
- `tuitest`: runs a Model headless against a virtual screen, for tests.
- `examples/`: one runnable program per directory.

## Components

### Program and event loop (root package)

`program.go` builds the Program and owns startup and terminal restore.
`loop.go` is the event loop. `options.go` holds the `ProgramOption`s, and
`cmds.go`, `cmds_runtime.go` and `cmds_ctx.go` the Cmds. The loop is the only
goroutine that calls `Init`, `Update` and `View`, and the only one that
touches frame state. Other goroutines (input reader, Cmds, signal handler,
resize watcher) only put messages on the queue. A goroutine that needs the
terminal restored (a signal, a panicking Cmd) asks the loop, and falls back to
a fixed restore sequence after 500ms (`program_restore.go`).

Runtime messages such as `Batch`, `Sequence`, `Println`, `Suspend` and mode
changes are handled by the loop and never reach `Update`. Lifecycle,
messages and options: [program.md](../program.md).

### Input

Package `input` decodes bytes into `Key`, `MouseEvent`, `PasteEvent`,
`FocusEvent` and terminal replies. The root package re-exports these types as
aliases. `program_input.go` runs the reader goroutine. It takes capability-probe
and clipboard replies out of the stream, and drops repeated focus events,
before anything reaches the queue. Details: [input.md](../input.md).

### Rendering

`frame.go` turns a View into lines and tracks the live region (how many rows,
where the cursor is). `frame_render.go` defines the two frame renderers: the
cell renderer (default, `internal/render`) and the line renderer, which is
also the fallback. `celldraw.go` is the direct path for a `CellDrawer` root,
drawing into a `cellbuf.Buffer`. `internal/render` does no I/O: it returns
bytes and the Program writes them. Details: [rendering.md](../rendering.md).

### Terminal port

`tui.Terminal` is the one terminal interface; `internal/termio` holds
`OSTerminal` and a `Fake` for tests (`TestSinglePortDefinition`). Only `term`
and `internal/termio` may import `term` (`TestOnlyTermioImportsTerm`).
`WithTerminal` swaps the port for a remote session.

### Capabilities and accessibility

`capabilities.go` runs the optional startup probe. The wire format and reply
parser live in `internal/capprobe`, which does no I/O. `accessible.go` and
`dumbterm.go` implement accessible mode and announcements, with their state in
`internal/announce`. See [capabilities.md](../capabilities.md) and
[accessibility.md](../accessibility.md).

### Diagnostics

`framelog.go` (`WithFrameLog`), `inspector.go` (`WithInspector`) and
`recorder.go` (`WithRecorder`, read back by `tuitest.Replay`). They only
observe: the frame log and the inspector's timing run after the frame has
been written.

## Package layers

`internal/archtest` enforces import direction: a package may import only
packages in lower tiers, or in its own tier, except that components can't
import each other. `TestImportDirection` fails on any import that points up.

| Tier | Packages |
| --- | --- |
| 0 | `ansi`, `layout`, `theme`, `motion`, `term`; `internal/basetypes`, `internal/a11y`, `internal/bidi`, `internal/fsutil`, `internal/highlight`, `internal/ptytest`, `internal/boxdraw` |
| 1 | `input`, `keymap` |
| 2 | `hittest`, `cellbuf`; `internal/render`, `internal/termio`, `internal/capprobe`, `internal/announce`, `internal/braille`, `internal/cancelreader`, `internal/edit`, `internal/vtscreen` |
| 3 | the root package `tui` |
| 4 | `widgets`, `widgets/chart`, `markdown` (stateless, may not import the root); `focus`, `tuitest`; test support `internal/cellcheck`, `internal/testutil` |
| 5 | every other package: the stateful components (`textinput`, `viewport`, `datatable`, ...) |

Other rules in the same package:

- Every package in the module, examples and tools included, imports only the
  standard library and the module, and `go.mod` requires nothing
  (`TestLibraryImportsStdlibOnly`).
- A component imports another component only where `composition` in
  `layers_test.go` allows it (for example `form` imports `textinput` and
  `passwordinput`).
- `widgets`, `widgets/chart` and `markdown` import neither the root runtime nor
  `focus` or `tuitest` (`TestStatelessKitsDoNotImportRuntime`). The root
  imports no component (`TestRootImportsNoComponent`).
- No non-test file imports `internal/cellcheck` or `internal/testutil`
  (`TestTestHelpersStayOutOfProductionCode`).
- No library code calls `ansi.SetClusterWidth`: width settings for one
  terminal travel as an `ansi.Measurer` (`TestNoGlobalWidthToggle`).
- Every public package has a runnable Example (`TestEveryPublicPackageHasExample`).
- Every stateful widget has `Linearize` and `LayoutNode` methods
  (`TestEveryStatefulWidgetHasLinearizeAndLayoutNode`).

`.golangci.yml` mirrors the import rules for editors. It is generated from the
same tables: `ARCHTEST_UPDATE=1 go test ./internal/archtest -run TestGolangciMirror`.

## Cross-cutting concerns

**Concurrency.** Model code runs on the loop goroutine only. Each Cmd runs on
its own goroutine (`WithMaxConcurrentCmds` caps them). `Program.Context`
is cancelled when Run returns. Every send to the queue also watches for
shutdown, so a Cmd that finishes after Run returns doesn't block delivering
its message. All terminal writes
share one mutex. Once the terminal is restored, later writes are dropped.

**Terminal restore.** Every exit path restores the terminal exactly once: a
normal return, an error, a panic, SIGTERM/SIGHUP, and a panicking Cmd
goroutine. Modes are switched on in a fixed order and off in reverse
(`enterModes` and `leaveModes` in `program.go`).

**Platforms.** Linux, macOS, the BSDs and Windows. Building for any other
`GOOS` fails on purpose with an import error naming the platform as
unsupported (`platform_unsupported.go`). Resizes come from SIGWINCH on Unix
and from console events, with polling as a fallback, on Windows.
