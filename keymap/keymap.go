// Package keymap is a registry of key bindings that an app fills once and
// that help widgets (helpscreen, widgets.KeyHints) read, so the help text
// cannot drift from a second hand-written list. Keys are plain strings such as
// "ctrl+c" or "enter", matched against a key event by Matches.
//
// # Widget convention
//
// A stateful widget that reacts to keys declares a KeyMap struct with one
// Binding field per action, a DefaultKeyMap constructor returning today's
// keys, a KeyMap field on its Model, and a Bindings method returning the
// bindings it currently honours (so help can list them). The widget's Update
// tests keys with Matches(msg, m.KeyMap.Down), never with a literal, so a
// caller rebinding a field changes the behaviour and the help together.
package keymap

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/ows4444/tui/input"
)

// NewBinding returns a Binding for desc triggered by keys.
func NewBinding(desc string, keys ...string) Binding {
	return Binding{Keys: keys, Desc: desc}
}

// Matches reports whether msg is a key press named by one of b's keys. Key
// names are those of input.Key.String ("j", "ctrl+c", "enter", "up"). Any
// other Msg (tui.KeyReleaseMsg included), a Key whose Action is KeyRelease, or
// a Binding with no keys, reports false. A repeat counts as a press.
func Matches(msg any, b Binding) bool {
	k, ok := msg.(input.Key)
	if !ok || k.Action == input.KeyRelease || b.disabled {
		return false
	}
	if k.Mod != input.ModNone {
		name := k.String() // a modified key is rare enough to allocate for
		for _, want := range b.Keys {
			if want == name {
				return true
			}
		}
		return false
	}
	// An unmodified key is the common case (every typed character), so its
	// name is built in a stack buffer and compared without allocating.
	var buf [32]byte
	name := appendPlainName(buf[:0], k)
	for _, want := range b.Keys {
		if want == string(name) {
			return true
		}
	}
	return false
}

// appendPlainName appends what input.Key.String returns for k when k has no
// modifiers. Only the two kinds that String builds from runes (and so
// allocates for) are spelled out here; the named keys return constants from
// String without allocating. A test keeps this in step with String.
func appendPlainName(dst []byte, k input.Key) []byte {
	switch k.Type {
	case input.KeyRunes:
		return append(dst, k.Text...)
	case input.KeyCtrl:
		if k.Code == ' ' {
			return append(dst, "ctrl+space"...)
		}
		dst = append(dst, "ctrl+"...)
		return utf8.AppendRune(dst, k.Code)
	}
	return append(dst, k.String()...)
}

// Binding is one action and the keys that trigger it.
type Binding struct {
	// Keys are the key names that trigger the action, e.g. "q", "ctrl+c".
	Keys []string
	// Desc is the human description, e.g. "quit".
	Desc string
	// Scope groups bindings that are active together ("" is the default
	// scope). Only bindings in the same scope can conflict.
	Scope string

	// disabled is stored inverted so the zero Binding is enabled.
	disabled bool
}

// SetEnabled enables or disables b. A disabled binding never matches in
// Matches and is left out of Registry.Bindings and Hints (so help screens do
// not list it). The zero Binding is enabled.
func (b *Binding) SetEnabled(on bool) { b.disabled = !on }

// Enabled reports whether b is enabled.
func (b Binding) Enabled() bool { return !b.disabled }

// Hint is a binding flattened for display: its keys joined with "/" and its
// description.
type Hint struct {
	Key  string
	Desc string
}

// Conflict reports a key already bound in a scope.
type Conflict struct {
	Scope string
	Key   string
	// Existing is the binding that already held Key; New is the one added
	// after it. New is still registered; the conflict is only reported.
	Existing Binding
	New      Binding
}

// String describes the conflict in one line, for logs and error text.
func (c Conflict) String() string {
	return fmt.Sprintf("key %q in scope %q bound to both %q and %q", c.Key, c.Scope, c.Existing.Desc, c.New.Desc)
}

// Registry holds bindings in registration order. The zero Registry is empty
// and ready to use; use it through a pointer.
type Registry struct {
	bindings  []Binding
	conflicts []Conflict
}

// Add registers b and returns any conflicts it creates with earlier bindings
// in the same scope, and a non-nil error when there are some. The binding is
// registered either way; Conflicts lists every conflict seen so far.
func (r *Registry) Add(b Binding) ([]Conflict, error) {
	b.Keys = append([]string(nil), b.Keys...)
	var found []Conflict
	seen := map[string]bool{}
	for _, k := range b.Keys {
		if seen[k] {
			continue
		}
		seen[k] = true
		for _, e := range r.bindings {
			if e.Scope != b.Scope {
				continue
			}
			if hasKey(e, k) {
				found = append(found, Conflict{Scope: b.Scope, Key: k, Existing: e, New: b})
				break
			}
		}
	}
	r.bindings = append(r.bindings, b)
	r.conflicts = append(r.conflicts, found...)
	if len(found) == 0 {
		return nil, nil
	}
	parts := make([]string, len(found))
	for i, c := range found {
		parts[i] = c.String()
	}
	return found, fmt.Errorf("keymap: %s", strings.Join(parts, "; "))
}

func hasKey(b Binding, k string) bool {
	for _, x := range b.Keys {
		if x == k {
			return true
		}
	}
	return false
}

// Conflicts returns every conflict reported by Add so far.
func (r *Registry) Conflicts() []Conflict { return append([]Conflict(nil), r.conflicts...) }

// SetEnabled enables or disables every registered binding in scope that holds
// key, and returns how many it changed. Disabled bindings are omitted from
// Bindings and Hints but still count for conflict detection.
func (r *Registry) SetEnabled(scope, key string, on bool) int {
	n := 0
	for i := range r.bindings {
		b := &r.bindings[i]
		if b.Scope == scope && hasKey(*b, key) && b.Enabled() != on {
			b.SetEnabled(on)
			n++
		}
	}
	return n
}

// Bindings returns the enabled bindings in scope, in registration order.
func (r *Registry) Bindings(scope string) []Binding {
	var out []Binding
	for _, b := range r.bindings {
		if b.Scope == scope && !b.disabled {
			out = append(out, b)
		}
	}
	return out
}

// Hints returns the bindings in scope as display hints, in registration order.
// Bindings with no keys or no description are skipped.
func (r *Registry) Hints(scope string) []Hint {
	var out []Hint
	for _, b := range r.Bindings(scope) {
		if len(b.Keys) == 0 || b.Desc == "" {
			continue
		}
		out = append(out, Hint{Key: strings.Join(b.Keys, "/"), Desc: b.Desc})
	}
	return out
}
