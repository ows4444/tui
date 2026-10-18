# Development tools

## The tui command

`go run github.com/ows4444/tui/cmd/tui` (or `go install` it) has:

| Command | What it does |
|---|---|
| `tui new [-module path] dir` | scaffold a program with a headless test |
| `tui keys` | show what the terminal sends: each key decoded (type, text, modifiers, press or release), mouse events with click counts, paste, focus. Ctrl+Q quits |
| `tui keys check file.json` | lint a key-binding file (`keymap.Parse`) |
| `tui log [-f] [-n lines] [-grep text] file` | read and follow the file `tui.LogToFile` writes |
| `tui theme check [-min ratio] file.json` | parse a theme file and list roles below the contrast ratio |
| `tui theme list` | the built-in theme names |
| `tui colors [preset]` | swatches of a preset's roles with their contrast |
| `tui borders` | a box in each border preset |
| `tui easings [name]` | plots the easing curves |
| `tui demo [-list] [widget]` | try the widgets live: a filterable list of 27, the selected one working on the right with its code below, and a key to change the theme |
| `tui prompt ...` | ask a question from a shell script |
| `tui dev [-poll d] [-watch dir]... [package] [-- args]` | build a program, run it on the terminal, and build and run it again whenever the `.go`, `go.mod` or `.tmpl` files of its module change |

`tui demo` is the quickest way to see what tui has without writing any code:
`go run github.com/ows4444/tui/cmd/tui@latest demo`. Type to narrow the list,
Enter (or Tab, or Right) gives the keys to the widget, Tab or Esc takes them back,
Ctrl+T changes the theme, Ctrl+C quits. The line under the widget shows the last
message it sent, which is what your `Update` would receive. `tui demo -list` prints
the widgets and `tui demo Dropdown` starts on one. It needs a terminal of at
least 60x14.

`tui dev` watches the module in the working directory (add `-watch dir` for other
directories, such as a module you `replace`), polls every 400 ms (`-poll`), and
needs no tools beyond `go`. A change stops the running program with SIGTERM so it
can restore the terminal, and kills it after two seconds if it does not; then it
resets the terminal modes and builds again. A build that fails prints the
compiler's errors and waits for the next change. When the program exits by
itself, `tui dev` waits for a change too. Ctrl+C stops it.

```
tui dev ./examples/workspace
tui dev ./examples/pkgs
```

## Seeing your app

- `tui.LogToFile(path, prefix)` redirects the standard logger to a file, so
  `log.Printf` is safe while the Program owns the screen. Tail it from another
  terminal with `tui log -f path`.
- `tui.WithInspector(tui.InspectorKeys{})` adds overlays (F12 for the panel, F11 for the layout outlines, and keys for the messages and the model): the
  named layout rectangles, the focused id, the last frame's size and time, the
  recent messages, a model dump.
- `tui.WithFrameLog(w)` writes one line per frame (`kind=diff rows=24 changed=2
  bytes=118 us=61`) to find a flicker or a slow frame.
- `tui.WithScreenshot(key, save)` hands `save` a `tui.Screenshot` (the rows as
  text and with their colours) when the key is pressed; `tui.ScreenshotCmd`
  delivers one to `Update`. `tuitest.SVGOf(shot.ANSI, w, h, opts)` draws it as
  an SVG for a bug report or the docs.
- `tui.WithRecorder` writes an asciicast of the session.

## Testing

`tuitest.New(model, w, h)` runs a model headless: `Keys`, `Click`, `ClickText`,
`Find`, `CursorOn`, `WaitForText`, `Golden`, `SVG`. See [Testing](testing.md).
