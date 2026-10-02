package tui

import "github.com/ows4444/tui/keymap"

// BindingsProvider is implemented by a root Model that can say which key
// bindings are active right now. A composite model returns the bindings of the
// widget that has focus (and its own), so the list follows focus and follows a
// widget's KeyMap when the app rebinds it; a widget's own Bindings method has
// this shape, so a model that embeds one directly already satisfies it.
type BindingsProvider interface {
	Bindings() []keymap.Binding
}

// publishKeymap records m, the model as of the frame being drawn, for Keymap.
func (p *Program) publishKeymap() {
	m := p.model
	p.keymapModel.Store(&m)
}

// Keymap returns a registry of the key bindings the root model reports through
// BindingsProvider, as of the last frame drawn (or of the initial model before
// the first). It is built fresh on each call, so it always reflects the
// focused widget's current KeyMap; a model that is not a BindingsProvider
// gives an empty registry. The registry is the list helpscreen.FromRegistry
// and widgets.KeyHints read, so help text cannot drift from behaviour. It is
// safe to call from any goroutine, provided the model's Bindings is a pure
// function of the model, as a value-typed Model's is.
func (p *Program) Keymap() *keymap.Registry {
	r := &keymap.Registry{}
	m := p.keymapModel.Load()
	if m == nil {
		return r
	}
	if bp, ok := find[BindingsProvider](*m); ok {
		for _, b := range bp.Bindings() {
			_, _ = r.Add(b) // conflicts are kept on the registry, see Conflicts
		}
	}
	return r
}
