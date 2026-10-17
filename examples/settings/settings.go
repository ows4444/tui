package main

import (
	"fmt"
	"strconv"
)

// kind is how a setting is shown and changed.
type kind int

const (
	toggle kind = iota // on or off
	choice             // one of a list, by index
	slide              // a number between two ends
)

// setting is one line of the settings list. Its value is a float64 for every
// kind: 0 or 1 for a toggle, the index for a choice, the number for a slider.
type setting struct {
	key   string
	label string
	help  string // one line, shown under the list for the selected setting
	kind  kind
	def   float64

	opts          []string // choice
	min, max, inc float64  // slide
	unit          string   // slide, shown after the number
	needs         string   // key of a toggle this setting depends on, or ""
}

type category struct {
	name  string
	items []setting
}

// values maps a setting's key to its value.
type values map[string]float64

func catalogue() []category {
	return []category{
		{"General", []setting{
			{key: "lang", label: "Language", help: "The language of menus and messages.", kind: choice,
				opts: []string{"English", "Deutsch", "Español", "Français"}},
			{key: "updates", label: "Check for updates", help: "Look for a new version when the app starts.", kind: toggle, def: 1},
			{key: "session", label: "Restore last session", help: "Reopen the files you had open.", kind: toggle, def: 1},
			{key: "telemetry", label: "Share usage data", help: "Off by default. Nothing is sent in this example either way.", kind: toggle},
		}},
		{"Appearance", []setting{
			{key: "theme", label: "Theme", help: "Auto follows the terminal.", kind: choice, def: 2,
				opts: []string{"Dark", "Light", "Auto"}},
			{key: "font", label: "Font size", help: "Text size in points.", kind: slide, def: 14, min: 8, max: 32, inc: 1, unit: " pt"},
			{key: "spacing", label: "Line spacing", help: "Space between lines, as a multiple of the font size.", kind: slide, def: 1.4, min: 1, max: 2, inc: 0.1, unit: "x"},
			{key: "motion", label: "Animations", help: "Fades and slides. Turn off to reduce motion.", kind: toggle, def: 1},
		}},
		{"Editor", []setting{
			{key: "tab", label: "Tab width", help: "Spaces a tab stands for.", kind: choice, def: 1, opts: []string{"2", "4", "8"}},
			{key: "wrap", label: "Word wrap", help: "Wrap long lines instead of scrolling sideways.", kind: toggle},
			{key: "autosave", label: "Autosave delay", help: "Seconds after the last change. 0 turns autosave off.", kind: slide, def: 10, min: 0, max: 60, inc: 5, unit: " s"},
			{key: "fmt", label: "Format on save", help: "Tidy the file each time it is saved.", kind: toggle, def: 1},
		}},
		{"Notifications", []setting{
			{key: "sound", label: "Sound", help: "Play a sound for a new notification.", kind: toggle, def: 1},
			{key: "volume", label: "Volume", help: "How loud the sound is.", kind: slide, def: 60, min: 0, max: 100, inc: 5, unit: "%", needs: "sound"},
			{key: "banner", label: "Desktop banners", help: "Show a banner as well as the list entry.", kind: toggle, def: 1},
			{key: "dnd", label: "Quiet hours", help: "No banners or sounds during these hours.", kind: choice,
				opts: []string{"Never", "22:00 to 07:00", "20:00 to 08:00"}},
		}},
	}
}

func defaults(cats []category) values {
	v := values{}
	for _, c := range cats {
		for _, s := range c.items {
			v[s.key] = s.def
		}
	}
	return v
}

func (v values) with(key string, x float64) values {
	out := make(values, len(v))
	for k, y := range v {
		out[k] = y
	}
	out[key] = x
	return out
}

// clamp keeps x inside the setting's range and on its step.
func (s setting) clamp(x float64) float64 {
	switch s.kind {
	case toggle:
		if x >= 0.5 {
			return 1
		}
		return 0
	case choice:
		return max(0, min(x, float64(len(s.opts)-1)))
	}
	x = max(s.min, min(x, s.max))
	steps := float64(int((x-s.min)/s.inc + 0.5))
	return float64(int((s.min+steps*s.inc)*1000+0.5)) / 1000
}

// step moves the value one notch, wrapping a choice round.
func (s setting) step(x float64, dir int) float64 {
	switch s.kind {
	case toggle:
		return 1 - x
	case choice:
		n := len(s.opts)
		return float64((int(x) + dir + n) % n)
	}
	return s.clamp(x + float64(dir)*s.inc)
}

// text is the value as shown.
func (s setting) text(x float64) string {
	switch s.kind {
	case toggle:
		if x == 1 {
			return "On"
		}
		return "Off"
	case choice:
		return s.opts[int(x)]
	}
	if s.inc < 1 {
		return strconv.FormatFloat(x, 'f', 1, 64) + s.unit
	}
	return fmt.Sprintf("%d%s", int(x), s.unit)
}

// enabled is false while the toggle a setting needs is off.
func (s setting) enabled(v values) bool { return s.needs == "" || v[s.needs] == 1 }
