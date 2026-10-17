package keymap

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Overrides are the keys a user chose for actions, by action name: the
// Desc of a Binding. An action mapped to no keys is unbound.
type Overrides map[string][]string

// Parse reads overrides from JSON: an object from action name to a list of key
// names, or to one key name.
//
//	{"quit": ["q", "ctrl+c"], "scroll down": "j", "help": []}
//
// Every key is checked with ValidKey, so a typo is reported when the file is
// read and not when the key is pressed and nothing happens. A repeated key in
// one list is kept once. The file is the user's: a program loads it, applies it
// with Apply and shows Unknown names to the user.
func Parse(r io.Reader) (Overrides, error) {
	var raw map[string]json.RawMessage
	dec := json.NewDecoder(r)
	if err := dec.Decode(&raw); err != nil {
		return nil, fmt.Errorf("keymap: %w", err)
	}
	if dec.More() {
		return nil, errors.New("keymap: more than one JSON value")
	}
	out := make(Overrides, len(raw))
	for _, action := range sortedKeys(raw) {
		var keys []string
		if err := json.Unmarshal(raw[action], &keys); err != nil {
			var one string
			if err2 := json.Unmarshal(raw[action], &one); err2 != nil {
				return nil, fmt.Errorf("keymap: %q: want a key name or a list of key names", action)
			}
			keys = []string{one}
		}
		seen := map[string]bool{}
		clean := []string{}
		for _, k := range keys {
			if !ValidKey(k) {
				return nil, fmt.Errorf("keymap: %q: %s is not a key name", action, strconv.Quote(k))
			}
			if !seen[k] {
				seen[k] = true
				clean = append(clean, k)
			}
		}
		out[action] = clean
	}
	return out, nil
}

// Apply returns a copy of bindings in which each one whose Desc has an entry
// in overrides has its Keys replaced by that entry; its Scope and enabled
// state are kept. The second result lists, sorted, the names in overrides that
// match no binding, which is usually a typo or a binding of another version.
// The input slice is not changed.
//
// Apply works on a plain list, before the bindings go into a Registry, so Add
// still reports a conflict the new keys cause.
func Apply(bindings []Binding, overrides Overrides) ([]Binding, []string) {
	out := make([]Binding, len(bindings))
	used := map[string]bool{}
	for i, b := range bindings {
		if keys, ok := overrides[b.Desc]; ok {
			b.Keys = append([]string(nil), keys...)
			used[b.Desc] = true
		}
		out[i] = b
	}
	var unknown []string
	for name := range overrides {
		if !used[name] {
			unknown = append(unknown, name)
		}
	}
	sort.Strings(unknown)
	return out, unknown
}

// ValidKey reports whether name is a key name Matches can ever see: one
// character ("j", "?", "+"), a named key ("enter", "esc", "tab", "space",
// "backspace", "delete", "insert", "up", "down", "left", "right", "home",
// "end", "pgup", "pgdown", "f1" to "f24", the media keys), or any of those
// after "ctrl+", "alt+", "shift+" or "super+" prefixes. Names are lower case
// and have no spaces.
func ValidKey(name string) bool {
	for {
		rest := ""
		for _, m := range []string{"ctrl+", "alt+", "shift+", "super+"} {
			if strings.HasPrefix(name, m) && len(name) > len(m) {
				rest = name[len(m):]
				break
			}
		}
		if rest == "" {
			break
		}
		name = rest
	}
	if utf8.RuneCountInString(name) == 1 {
		r, _ := utf8.DecodeRuneInString(name)
		return r != ' ' && r != utf8.RuneError
	}
	if namedKeys[name] {
		return true
	}
	if strings.HasPrefix(name, "f") {
		n, err := strconv.Atoi(name[1:])
		return err == nil && n >= 1 && n <= 24 && name[1:] == strconv.Itoa(n)
	}
	return false
}

var namedKeys = func() map[string]bool {
	m := map[string]bool{}
	for _, n := range []string{
		"up", "down", "left", "right", "enter", "esc", "tab", "backspace", "delete", "insert",
		"space", "home", "end", "pgup", "pgdown",
		"media-play", "media-pause", "media-play-pause", "media-reverse", "media-stop",
		"media-fast-forward", "media-rewind", "media-next", "media-previous", "media-record",
		"volume-down", "volume-up", "mute",
	} {
		m[n] = true
	}
	return m
}()

func sortedKeys(m map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
