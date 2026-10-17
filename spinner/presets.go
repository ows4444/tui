package spinner

import (
	"slices"
	"strings"
	"time"

	"github.com/ows4444/tui/theme"
)

// Preset is a named animation: its frames, how long each stays and the frames
// to use on an ASCII-only theme. The frames come from the widely copied
// cli-spinners set, so a name that is known elsewhere means the same here.
type Preset struct {
	// Name is how Lookup finds the preset.
	Name string
	// Frames are the animation, every one the same width.
	Frames []string
	// Interval is how long each frame stays.
	Interval time.Duration
	// ASCII is what an ASCII-only theme draws instead: frames that use only
	// 7-bit characters. It is nil when Frames already are.
	ASCII []string
}

func f(s string) []string { return strings.Split(s, "|") }

// presets are in the order Presets returns them.
var presets = []Preset{
	{Name: "dots", Frames: f("⠋|⠙|⠹|⠸|⠼|⠴|⠦|⠧|⠇|⠏"), Interval: 80 * time.Millisecond},
	{Name: "dots2", Frames: f("⣾|⣽|⣻|⢿|⡿|⣟|⣯|⣷"), Interval: 80 * time.Millisecond},
	{Name: "line", Frames: []string{"-", `\`, "|", "/"}, Interval: 130 * time.Millisecond},
	{Name: "pipe", Frames: f("┤|┘|┴|└|├|┌|┬|┐"), Interval: 100 * time.Millisecond},
	{Name: "simpleDots", Frames: f(".  |.. |...|   "), Interval: 400 * time.Millisecond},
	{Name: "star", Frames: f("✶|✸|✹|✺|✹|✷"), Interval: 70 * time.Millisecond},
	{Name: "growVertical", Frames: f("▁|▃|▄|▅|▆|▇|▆|▅|▄|▃"), Interval: 120 * time.Millisecond},
	{Name: "growHorizontal", Frames: f("▏|▎|▍|▌|▋|▊|▉|▊|▋|▌|▍|▎"), Interval: 120 * time.Millisecond},
	{Name: "balloon", Frames: f(" |.|o|O|@|*| "), Interval: 140 * time.Millisecond},
	{Name: "arc", Frames: f("◜|◠|◝|◞|◡|◟"), Interval: 100 * time.Millisecond},
	{Name: "circle", Frames: f("◡|⊙|◠"), Interval: 120 * time.Millisecond},
	{Name: "bounce", Frames: f("⠁|⠂|⠄|⠂"), Interval: 120 * time.Millisecond},
	{Name: "arrow", Frames: f("←|↖|↑|↗|→|↘|↓|↙"), Interval: 100 * time.Millisecond},
	{Name: "boxBounce", Frames: f("▖|▘|▝|▗"), Interval: 120 * time.Millisecond},
	{Name: "triangle", Frames: f("◢|◣|◤|◥"), Interval: 50 * time.Millisecond},
	{Name: "toggle", Frames: f("⊶|⊷"), Interval: 250 * time.Millisecond},
	{Name: "bouncingBar", Frames: f("[    ]|[=   ]|[==  ]|[=== ]|[====]|[ ===]|[  ==]|[   =]|[    ]|[   =]|[  ==]|[ ===]|[====]|[=== ]|[==  ]|[=   ]"), Interval: 80 * time.Millisecond},
	{Name: "ascii", Frames: []string{"-", `\`, "|", "/"}, Interval: 100 * time.Millisecond},
}

func init() {
	// Presets whose frames are not 7-bit ASCII fall back to the plain spinner.
	for i := range presets {
		if !isASCII(presets[i].Frames) {
			presets[i].ASCII = []string{"-", `\`, "|", "/"}
		}
	}
}

func isASCII(frames []string) bool {
	for _, s := range frames {
		for _, r := range s {
			if r > 0x7e {
				return false
			}
		}
	}
	return true
}

// copyOf returns p with its slices copied, so a caller cannot change the set.
func copyOf(p Preset) Preset {
	p.Frames = slices.Clone(p.Frames)
	p.ASCII = slices.Clone(p.ASCII)
	return p
}

// Presets returns the named animations, as copies the caller may change.
func Presets() []Preset {
	out := make([]Preset, len(presets))
	for i, p := range presets {
		out[i] = copyOf(p)
	}
	return out
}

// Lookup returns the preset called name, ignoring case, and whether there is
// one.
func Lookup(name string) (Preset, bool) {
	for _, p := range presets {
		if strings.EqualFold(p.Name, name) {
			return copyOf(p), true
		}
	}
	return Preset{}, false
}

// WithPreset returns m set to play p: its frames and interval, starting from
// the first frame. On an ASCII-only theme (theme.Theme.ASCII) the preset's
// ASCII frames are drawn instead.
func (m Model) WithPreset(p Preset) Model {
	p = copyOf(p)
	m.Frames, m.Interval, m.ascii, m.frame = p.Frames, p.Interval, p.ASCII, 0
	return m
}

// frames are the frames View and DrawCells draw: the stock frames follow the
// theme's glyph set, a preset falls back to its ASCII frames on an ASCII
// theme, and anything else is drawn as given.
func (m Model) frames() []string {
	g := m.themed().GlyphSet()
	if isDefaultFrames(m.Frames) {
		return g.SpinnerFrames()
	}
	if m.ascii != nil && g.Spinner == theme.ASCIIGlyphSet().Spinner {
		return m.ascii
	}
	return m.Frames
}
