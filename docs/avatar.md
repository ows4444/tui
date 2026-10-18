# Avatars and faces

A name goes in and a figure comes out, the same one every time. Use package
[avatar](../avatar) to stand for a person or an object a user has to
recognise again in the next list. Use package [faces](../faces) for a
ready-made animated character that shows a mood or a status. Pick by what
decides the picture: a name (avatar) or you (faces).

| | avatar | faces |
|---|---|---|
| Decided by | the name, hashed | an index or a name you choose |
| Looks | 10 silhouettes, 5 accessories, 4 eye styles, a colour | 50 Braille characters, 4 to 12 frames |
| Colour | follows the theme: 3:1 against the page | the theme's colours |
| Output | cells, PNG, SVG | cells |
| Use for | people, accounts, ids | spinners with a face, moods, empty states |

`faces.For(s)` and `faces.ColorFor(s)` pick a character and a colour for any
string, so a screen that wants only a stable mascot per name can use faces.

## A list with avatars

Make the Model where you draw it. The pictures drawn lately are kept for all
Models, so `New` for each row on each frame costs a lookup, not a drawing. The
avatar sits beside the name, so set `Decorative` and a screen reader hears the
name once.

```go
m := avatar.New(name)
m.Width, m.Height, m.Decorative = 4, 2, true
row := m.View() // one line per row of the avatar
```

`Example_list` in `avatar/guide_example_test.go`. `examples/routerapp` shows
one beside an open customer.

## A house style

`Hue` (a pointer: nil lets the name choose, so `0` is a real hue), `Tone`,
`Silhouette`, `Accessory` and `Eyes` pin the look. The name still chooses
everything else.

```go
blue := 250.0
m.Hue, m.Tone, m.Silhouette = &blue, avatar.ToneMid, avatar.SilhouetteRound
```

`Example_houseStyle`. `m.Shape()` reports the silhouette drawn, pinned or not.

## Expressions, blinking and idling

`Expression` is the pose the eyes hold (13 of them, each with eyebrows and a
mouth where the cells are big enough). `SetExpression` eases into one,
`React` pulls a face for a moment, `Blink` closes the eyes once and `StartIdle`
keeps the figure breathing and glancing. They return a Cmd: return it from your
`Update`, and pass the Msgs that follow to the avatar's `Update`. Call them on
the Model your own model keeps, not a copy. Under `motion.Reduced` they play
nothing.

For the mouse, set `X` and `Y` to where the avatar is drawn and send it the
Program's mouse events: while the pointer is over `Bounds` the idle loop runs
and the eyes follow it. The states and what moves between them are drawn in
`avatar/states.go`.

## Reading state, themes and layout

`Blinking`, `Idling` and `Reacting` say which animation is running. `SetTheme`
makes the Model a `tui.ThemeSetter`, so a root model can forward the Program's
theme; the body is held at 3:1 against the theme's page colour (or the plate),
and the eyes at 4.5:1 against the body. `LayoutNode` puts an avatar in a
`layout` tree at the size it is given. `DrawCells` draws the same picture as
`View` straight into a cell grid, for a model that is a `tui.CellDrawer`.
`Linearize` is what a screen reader hears.

## Outside the terminal

`SVG()` returns markup on a 100 by 100 view box and `PNG(size)` an image up to
`MaxPNGSize` pixels; both draw the pose now shown. `PNG` fits `imageview` for a
terminal that shows images, with `View` as the fallback.

## What is frozen

The mapping from a name to its silhouette, accessory, eyes and colour is pinned
by `avatar/testdata/vectors.json` and is not changed after v1. A name is put in
Unicode Normalization Form C, trimmed and lowercased with tables frozen at
Unicode 17, and cut to 256 bytes before it is hashed (`Raw` skips the case
folding). Pictures you cache outside the process stay valid.
