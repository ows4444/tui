# Terminal capabilities

A Program learns about the terminal in two ways. At `NewProgram`, it reads the
colour profile from the environment. Optionally, at startup, it can query the
terminal itself with `WithCapabilityProbe`. Every other protocol the library
can use (mouse, focus reporting, the kitty keyboard protocol, background
detection) is off until an option asks for it, so a Program behaves the same
whichever terminal it runs on.

## Colour profile

Without `WithColorProfile`, `NewProgram` calls `ansi.DetectColorProfileFor` on
the output. The first rule that matches wins:

1. `NO_COLOR` non-empty: `ansi.NoColor`.
2. Unless `CLICOLOR_FORCE` is non-empty and not `0`: an output that isn't a
   terminal, or `CLICOLOR=0`, gives `ansi.NoColor`.
3. `TERM=dumb`: `ansi.NoColor`.
4. `COLORTERM` is `truecolor` or `24bit`, or `WT_SESSION` is set: `ansi.TrueColor`.
5. `TERM_PROGRAM` is `iTerm.app`, `WezTerm`, `vscode`, `ghostty` or `Hyper`:
   `ansi.TrueColor`. `Apple_Terminal`: `ansi.ANSI256`.
6. `TERM` is `xterm-kitty`, `xterm-ghostty`, `alacritty` or `wezterm`:
   `ansi.TrueColor`. A `TERM` containing `256color`: `ansi.ANSI256`. An empty
   `TERM`: `ansi.NoColor`, or `ansi.ANSI16` on Windows. Anything else:
   `ansi.ANSI16`.

The Program rewrites every colour it writes to the nearest one the profile can
show, in frames and in `Println`/`Eprintln` text, so widgets need no changes.
Under `ansi.NoColor` all colour is removed, and bold, underline and the other
attributes stay. `Program.ColorProfile` reports the profile in use.

Truecolor terminals that set none of these variables (some SSH sessions, tmux
with `TERM=screen`) are detected at a lower depth. Pin the profile with
`WithColorProfile(ansi.TrueColor)`. Any `WithColorProfile`, even
`ansi.TrueColor`, overrides `NO_COLOR` (`TestExplicitColorProfileOverridesNoColor`).
An empty `NO_COLOR` counts as unset.

`examples/probe` shows the detected profile, focus reports and the background
colour for the terminal you run it in.

## Dumb terminals

With `TERM=dumb`:

- The colour profile is `ansi.NoColor`.
- A Program given neither `WithAltScreen` nor `WithAccessible` runs in
  accessible mode, which writes no escape sequences at all
  ([accessibility.md](accessibility.md)). An explicit `WithAltScreen(true)` or
  `WithAccessible(false)` is honoured (`TestDumbTermDefaults`).
- `theme.DetectGlyphs`, if the app calls it, returns the ASCII glyph set
  ([theming.md](theming.md)).

## The capability probe

`WithCapabilityProbe(timeout)` makes Run send these queries at startup:
DECRQM for modes 2026 and 2027, the kitty keyboard and kitty graphics queries,
the kitty notification query (OSC 99), and XTVERSION, followed by DA1. The DA1
reply ends the probe. If DA1 doesn't arrive within `timeout`, every
capability is false. A non-positive timeout means
`tui.DefaultCapabilityProbeTimeout` (500ms). The probe is off by default and
ignored in accessible mode.

When the probe ends, `Update` receives one `CapabilitiesMsg`, and
`Program.Capabilities` returns the same `Capabilities` value.
`Program.CapabilitiesKnown` tells "nothing supported" from "not finished yet".
Replies to the probe's queries don't reach `Update`.

What the Program does with the answer:

| Capability | Effect |
| --- | --- |
| `SyncOutput` false | Frames are written without the mode 2026 brackets |
| `StyledUnderline` false | Styled underlines (`SGR 4:n`) are written as plain underline. The value is inferred from the kitty replies or XTVERSION, not queried |
| `GraphemeClusters` | When true, the Program sets mode 2027 (reset on exit) and measures grapheme clusters as one unit; otherwise it measures per codepoint |
| `Notifications` | `Assertive` announcements are also sent as desktop notifications |
| `KittyKeyboard`, `KittyGraphics`, `Sixel`, `XTVersion` | Reported only |

Before the probe finishes, and without it, the Program uses synchronized
output and styled underlines as the View asks.

## Grapheme cluster width

`ansi.Width` and the other package-level width functions count a grapheme
cluster (a ZWJ emoji, a flag, a letter with combining marks) as one unit by
default. `TUI_NO_CLUSTERS` set to any non-empty value, or
`ansi.SetClusterWidth(false)`, counts each rune on its own instead. That
setting is process-wide.

Each Program also has its own `ansi.Measurer`, returned by `Program.Measurer`
and carried in every `ResizeMsg`. It follows the process-wide setting until
the probe decides. The probe then sends a second `ResizeMsg` with the new
`Measurer` and repaints. A component that measures text for one terminal
should use `msg.Measurer` rather than `ansi.Width`, so two Programs in one
process (an SSH server, parallel tests) don't measure for each other. No
library code calls `ansi.SetClusterWidth` (`TestNoGlobalWidthToggle` in
`internal/archtest`).

## Background colour

`WithBackgroundDetection(timeout)` asks the terminal for its background colour
(OSC 11) and for palette colours 0 to 15 (OSC 4) at startup. The reply comes as
a `BackgroundColorEvent`, and each palette reply as a `PaletteColorEvent`. If
nothing arrives within `timeout`, `Update` gets one `BackgroundUnknownMsg`.
`WithTheme` uses the same query, with a 300ms timeout unless
`WithBackgroundDetection` set one, to pick a light or dark theme
([theming.md](theming.md)). Neither query is sent in accessible mode.

## Environment variables

Library code reads only these:

| Variable | Read by | Effect |
| --- | --- | --- |
| `NO_COLOR` | colour detection | Non-empty: no colour |
| `CLICOLOR_FORCE`, `CLICOLOR` | colour detection | Force colour on a non-terminal; `CLICOLOR=0` turns it off |
| `COLORTERM`, `WT_SESSION`, `TERM_PROGRAM`, `TERM` | colour detection | Pick the depth (see above) |
| `TERM=dumb` | `NewProgram`, `WithAccessibleAuto`, glyph detection | Accessible mode by default; ASCII glyphs |
| `ACCESSIBLE=1` | `WithAccessibleAuto` | Accessible mode |
| `NO_ANIMATION`, `REDUCE_MOTION` | `motion.Detect` | Non-empty: reduced motion ([accessibility.md](accessibility.md)) |
| `TUI_NO_CLUSTERS` | `ansi` | Non-empty: measure per rune |
| `LC_ALL`, `LC_CTYPE`, `LANG`, `TUI_NERD_FONT` | `theme.DetectGlyphs` | Glyph set ([theming.md](theming.md)) |

`WithRecorder` also copies `TERM` into the asciicast header.
