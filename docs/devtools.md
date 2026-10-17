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
| `tui prompt ...` | ask a question from a shell script |

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
