package ansi

import (
	"io"
	"os"
	"runtime"
	"strings"
)

// Profile is the color depth a terminal can display.
type Profile uint8

const (
	// NoColor means no color output at all (NO_COLOR set, or TERM=dumb).
	NoColor Profile = iota
	// ANSI16 is the 16 standard ANSI colors.
	ANSI16
	// ANSI256 is the 256-color xterm palette.
	ANSI256
	// TrueColor is 24-bit color.
	TrueColor
)

// DetectColorProfile reports the color depth of standard output. It is
// DetectColorProfileFor(os.Stdout, os.Getenv): when stdout is not a
// terminal the result is NoColor unless CLICOLOR_FORCE is set.
func DetectColorProfile() Profile { return DetectColorProfileFor(os.Stdout, os.Getenv) }

// DetectColorProfileEnv is the environment-only detection: it behaves as
// DetectColorProfileFor with a terminal output, so it does no TTY check.
func DetectColorProfileEnv(getenv func(string) string) Profile {
	return detectProfile(true, runtime.GOOS, getenv)
}

// DetectColorProfileFor reports the color depth for out, reading the
// environment through getenv. Precedence: NO_COLOR (non-empty) gives
// NoColor; CLICOLOR_FORCE (non-empty, not "0") enables color even when out
// is not a terminal; otherwise a non-terminal out or CLICOLOR=0 gives
// NoColor. Then TERM=dumb gives NoColor, and COLORTERM=truecolor|24bit,
// WT_SESSION, TERM_PROGRAM and the TERM name pick the depth. An empty TERM
// is ANSI16 on Windows and NoColor elsewhere.
func DetectColorProfileFor(out io.Writer, getenv func(string) string) Profile {
	return detectProfile(isTerminal(out), runtime.GOOS, getenv)
}

// isTerminal reports whether w is an *os.File attached to a character device.
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok || f == nil {
		return false
	}
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

func detectProfile(tty bool, goos string, getenv func(string) string) Profile {
	if getenv("NO_COLOR") != "" {
		return NoColor
	}
	if f := getenv("CLICOLOR_FORCE"); f == "" || f == "0" {
		if !tty || getenv("CLICOLOR") == "0" {
			return NoColor
		}
	}
	term := strings.ToLower(getenv("TERM"))
	if term == "dumb" {
		return NoColor
	}
	if ct := strings.ToLower(getenv("COLORTERM")); ct == "truecolor" || ct == "24bit" {
		return TrueColor
	}
	if getenv("WT_SESSION") != "" {
		return TrueColor
	}
	switch getenv("TERM_PROGRAM") {
	case "iTerm.app", "WezTerm", "vscode", "ghostty", "Hyper":
		return TrueColor
	case "Apple_Terminal":
		return ANSI256
	}
	switch {
	case term == "xterm-kitty" || term == "xterm-ghostty" || term == "alacritty" || term == "wezterm":
		return TrueColor
	case strings.Contains(term, "256color"):
		return ANSI256
	case term == "":
		if goos == "windows" {
			return ANSI16
		}
		return NoColor
	}
	return ANSI16
}

// Downgrade returns c converted to the nearest color p can display. It
// returns nil for NoColor (a nil Color means "unset" to Style), and c
// unchanged when p already covers it or c is nil.
func Downgrade(c Color, p Profile) Color {
	if c == nil {
		return nil
	}
	if p == NoColor {
		return nil
	}
	switch v := c.(type) {
	case RGB:
		switch p {
		case ANSI256:
			return rgbTo256(v)
		case ANSI16:
			return rgbTo16(v)
		}
	case Color256:
		if p == ANSI16 {
			return rgbTo16(color256RGB(v))
		}
	}
	return c
}

func rgbTo256(c RGB) Color256 {
	if c.R == c.G && c.G == c.B { // grayscale ramp 232-255 (or cube ends)
		switch {
		case c.R < 8:
			return 16
		case c.R > 248:
			return 231
		}
		return Color256(clamp8(232 + (int(c.R)-8)*24/247))
	}
	q := func(v uint8) int { return (int(v)*5 + 127) / 255 }
	return Color256(clamp8(16 + 36*q(c.R) + 6*q(c.G) + q(c.B)))
}

func color256RGB(c Color256) RGB {
	n := int(c)
	switch {
	case n < 16:
		return ansi16RGB[n]
	case n >= 232:
		v := clamp8(8 + (n-232)*10)
		return RGB{v, v, v}
	}
	n -= 16
	level := func(i int) uint8 {
		if i == 0 {
			return 0
		}
		return clamp8(55 + 40*i)
	}
	return RGB{level(n / 36), level(n / 6 % 6), level(n % 6)}
}

// ansi16RGB approximates the standard palette (xterm defaults).
var ansi16RGB = [16]RGB{
	{0, 0, 0}, {205, 0, 0}, {0, 205, 0}, {205, 205, 0},
	{0, 0, 238}, {205, 0, 205}, {0, 205, 205}, {229, 229, 229},
	{127, 127, 127}, {255, 0, 0}, {0, 255, 0}, {255, 255, 0},
	{92, 92, 255}, {255, 0, 255}, {0, 255, 255}, {255, 255, 255},
}

func rgbTo16(c RGB) BasicColor {
	best, bestD := 0, int(^uint(0)>>1)
	for i, p := range ansi16RGB {
		dr, dg, db := int(c.R)-int(p.R), int(c.G)-int(p.G), int(c.B)-int(p.B)
		if d := dr*dr + dg*dg + db*db; d < bestD {
			best, bestD = i, d
		}
	}
	return BasicColor(best)
}

// clamp8 converts v to uint8, saturating at 0 and 255.
func clamp8(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}
