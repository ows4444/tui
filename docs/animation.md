# Animation

Animation in tui is a value changing over time that your `Update` reads on a
tick. The `motion` package supplies the maths (easing, tweens, springs,
timelines); `tui.Tick` and `tui.Every` supply the ticks; nothing here owns a
goroutine or a timer. Every piece honours reduced motion.

## A value over time: Tween

A [`motion.Tween`](https://pkg.go.dev/github.com/ows4444/tui/motion#Tween)
turns elapsed time into a value between `From` and `To`, shaped by an easing:

```go
tw := motion.Tween{From: 0, To: 40, Duration: 300 * time.Millisecond, Ease: motion.OutCubic}

// On each tick: elapsed is time.Since(start).
width := int(tw.At(elapsed))
if tw.Done(elapsed) { /* stop ticking */ }
```

Keep the start time in your model, start a ticker when the animation begins and
cancel it when `Done` is true:

```go
tick, stop := tui.Every(motion.FrameInterval(motion.DefaultFPS), func(t time.Time) tui.Msg { return frameMsg(t) })
```

`Every` returns the Cmd and a cancel func; after cancel returns, no further tick
reaches `Update`.

## Easing

An `Ease` maps progress in `[0, 1]` to eased progress and returns 0 at 0 and 1 at
1. There are 31, `Linear` and the Quad, Cubic, Quart, Quint, Sine, Expo, Circ,
Back, Elastic and Bounce families, each as `In`, `Out` and `InOut`
(`motion.InOutQuart`, `motion.OutBounce`). Back and Elastic overshoot 1 on the
way. To let a config file or a theme choose one, use `motion.ByName("out-cubic")`;
`motion.EaseNames()` lists the names. `tui easings [name]` plots the curves in
the terminal.

## Following a moving target: Spring

A `motion.Spring` moves `Value` toward `Target` as a damped oscillator, so a
target that changes mid-flight keeps its speed instead of restarting:

```go
sp := motion.NewSpring(0, 180) // value 0, stiffness 180
sp.Damping = 2 * math.Sqrt(180) // critical damping: no overshoot
sp.SetTarget(24)
// On each tick: sp.Step(sinceLastTick); draw sp.Value; stop when sp.Done().
```

Less damping rings around the target, more is sluggish. `Step` solves the motion
in closed form, so a long frame cannot blow it up.

## Sequences: Timeline

A `motion.Timeline` runs several tweens in order or overlapping, driven by a
time you pass in (a `motion.Clock`, or a fake one in a test):

```go
tl := &motion.Timeline{}
tl.Then(fadeIn, func(v float64) { m.alpha = v }).
	Then(slide, func(v float64) { m.x = int(v) }).
	With(grow, func(v float64) { m.h = int(v) }) // starts with slide
tl.Start(now)
// On each tick: if !tl.Update(now) { /* still running */ }
```

## Reduced motion

`Tween`, `Spring` and `Timeline` carry a `Motion` preference. With
`motion.Reduced` they skip to the end value at once. The preference comes from
`motion.Detect()` (the `NO_ANIMATION` and `REDUCE_MOTION` variables) or, inside a
Program, `program.ReducedMotion()` (set with `tui.WithReducedMotion`, or
`tui.WithEnvironment` for a program with its own environment: use
`motion.DetectEnv(program.Getenv)`). Widgets that animate (`spinner`,
`skeleton`, `streamtext`) are stopped by the app when it is set. See
[Accessibility](accessibility.md).

## Animating a screen region

`motion` moves numbers; what the number does is up to your `View`: a width, an
offset, a colour from `ansi.RGB.Mix`. For a fade, mix the colour of each style
with the background by the eased value; for a slide, pass the offset to
`layout.Absolute` or `layout.Anchor` inside a `layout.Stack`.

## Effects: motion/fx

For the usual cases there is a ready-made set in the experimental package
`motion/fx`: `Fade`, `Dissolve`, `Slide` and `Sweep`. An `Effect` rewrites a
cell grid that already holds the finished picture, as a function of a progress
`p` from 0 (nothing shows) to 1 (the picture as it is). `fx.Wrap` puts one
around any `layout.Node`, `fx.Out` runs one backwards for a panel that closes,
and `fx.Progress` turns elapsed time into `p` (eased, and 1 at once under
`motion.Reduced`):

```go
p := fx.Progress(time.Since(m.shownAt), 250*time.Millisecond, motion.OutCubic, m.motion)
panel := fx.Wrap(m.panel.LayoutNode(), fx.Sweep{Soft: 4}, p)
```

Drive `p` with the same tick as a `Tween`. At `p == 1` the node is drawn
unchanged. `Fade` blends toward the colour in `Fade.To`, which should be the
background behind the region; a cell with no background of its own is taken to
be that colour already.

