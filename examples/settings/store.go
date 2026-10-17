package main

import (
	"os"
	"strconv"
	"strings"

	"github.com/ows4444/tui"
)

// The file format is one "key=value" line a setting, in catalogue order.
func encode(cats []category, v values) string {
	var b strings.Builder
	for _, c := range cats {
		for _, s := range c.items {
			b.WriteString(s.key + "=" + strconv.FormatFloat(v[s.key], 'f', -1, 64) + "\n")
		}
	}
	return b.String()
}

// decode reads what encode wrote over the defaults; unknown keys and values
// that are not numbers or are out of range are ignored, so a hand-edited file
// cannot break the screen.
func decode(cats []category, text string) values {
	v := defaults(cats)
	known := map[string]setting{}
	for _, c := range cats {
		for _, s := range c.items {
			known[s.key] = s
		}
	}
	for _, line := range strings.Split(text, "\n") {
		k, val, ok := strings.Cut(strings.TrimSpace(line), "=")
		s, isKnown := known[k]
		x, err := strconv.ParseFloat(val, 64)
		if ok && isKnown && err == nil && x == s.clamp(x) {
			v[k] = x
		}
	}
	return v
}

// fromFile starts from the settings in path, or the defaults if it is not
// there yet. Apply writes them back.
func fromFile(m model, path string) (model, error) {
	m.path = path
	b, err := os.ReadFile(path) // #nosec G304 -- the path is the user's own -file flag
	switch {
	case err == nil:
		m.saved = decode(m.cats, string(b))
		m.cur = m.saved
	case !os.IsNotExist(err):
		return m, err
	}
	return m, nil
}

// writeFile returns a Cmd that saves text to path off the Update path.
func writeFile(path, text string) tui.Cmd {
	return func() tui.Msg {
		_ = os.WriteFile(path, []byte(text), 0o600)
		return nil
	}
}
