// Package form is a validating form: a column of single-line fields, each
// with a label and optional validators, and a Submit that checks them all.
//
// It composes textinput, passwordinput (for Secret fields) and focus.Route,
// so Tab and Shift+Tab move between fields and every other message goes to the
// focused field only. Submit runs every field's validators, shows each failing
// field's first error, moves focus to the first invalid field and only when
// all fields pass emits a SubmittedMsg. An error shown after a failed Submit
// is re-checked whenever its field is edited and clears once the value
// validates. Enter submits.
//
// A field is single-line text (masked when Secret), a choice from a list
// (FieldSelect) or a yes/no box (FieldCheckbox). Every field's value is a
// string in Values and SubmittedMsg.Values: a select's is its chosen option, a
// checkbox's is "true" or "false", so one Validator type checks them all.
//
// Model is a value type like the other widgets: Update and Submit return the
// new Model and leave the receiver unchanged.
package form

import (
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/focus"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/passwordinput"
	"github.com/ows4444/tui/textinput"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

// DefaultWidth is the visible width of a field whose Width is 0.
const DefaultWidth = 30

// Validator checks one field's value and returns an error describing what is
// wrong, or nil. The error's text is what the form shows.
type Validator func(value string) error

// FieldKind selects how a Field is drawn and edited.
type FieldKind int

const (
	// FieldText is a single-line text field, masked when Field.Secret is set.
	// It is the zero value, so a Field that sets no Kind is a text field.
	FieldText FieldKind = iota
	// FieldSelect is a choice from Field.Options: Left and Up choose the
	// previous option, Right and Down the next, stopping at either end. Its
	// value is the chosen option.
	FieldSelect
	// FieldCheckbox is a yes/no box: Space toggles it. Its value is "true" or
	// "false".
	FieldCheckbox
)

// Field describes one form field.
type Field struct {
	Name  string // key in Values and SubmittedMsg.Values; keep names unique
	Label string // shown before the input
	// Kind is how the field is drawn and edited; the zero value is FieldText.
	Kind FieldKind
	// Options are the choices of a FieldSelect, in order. Ignored otherwise.
	Options []string
	// Placeholder is shown while a text field is empty. Text fields only.
	Placeholder string
	// Value is the initial value: the text of a text field, the chosen option
	// of a select (an unknown or empty value chooses the first option), or
	// "true" (in any case) for a ticked checkbox; anything else is unticked.
	Value string
	// Secret masks a text field's value and never speaks it. Text fields only.
	Secret bool
	// Width is a text field's visible width; 0 = DefaultWidth. Text fields only.
	Width      int
	Validators []Validator
}

// SubmittedMsg is the message Submit's Cmd yields when every field is valid.
type SubmittedMsg struct {
	Values map[string]string
}

// fieldInput holds one field's widget: a passwordinput for a Secret text field,
// a textinput for other text, and plain state for a select or a checkbox.
type fieldInput struct {
	kind   FieldKind
	label  string
	text   textinput.Model
	secret passwordinput.Model
	isPass bool

	options []string // FieldSelect
	choice  int      // FieldSelect: index into options
	checked bool     // FieldCheckbox
	focused bool     // FieldSelect and FieldCheckbox; text widgets track their own
}

func (in *fieldInput) focus() tui.Cmd {
	switch in.kind {
	case FieldSelect, FieldCheckbox:
		in.focused = true
		return nil
	}
	if in.isPass {
		return in.secret.Focus()
	}
	return in.text.Focus()
}

func (in *fieldInput) blur() {
	switch in.kind {
	case FieldSelect, FieldCheckbox:
		in.focused = false
		return
	}
	if in.isPass {
		in.secret.Blur()
		return
	}
	in.text.Blur()
}

func (in *fieldInput) update(msg tui.Msg, km KeyMap) tui.Cmd {
	switch in.kind {
	case FieldSelect:
		switch {
		case keymap.Matches(msg, km.SelectPrev):
			if in.choice > 0 {
				in.choice--
			}
		case keymap.Matches(msg, km.SelectNext):
			if in.choice < len(in.options)-1 {
				in.choice++
			}
		}
		return nil
	case FieldCheckbox:
		if toggles(msg, km.Toggle) {
			in.checked = !in.checked
		}
		return nil
	}
	var cmd tui.Cmd
	if in.isPass {
		in.secret, cmd = in.secret.Update(msg)
	} else {
		in.text, cmd = in.text.Update(msg)
	}
	return cmd
}

// toggles reports whether msg triggers b. A binding that names "space" also
// takes a space sent as a plain rune, as before KeyMap existed.
func toggles(msg tui.Msg, b keymap.Binding) bool {
	if keymap.Matches(msg, b) {
		return true
	}
	k, ok := msg.(tui.Key)
	if !ok || !isSpace(k) {
		return false
	}
	for _, name := range b.Keys {
		if name == "space" {
			return true
		}
	}
	return false
}

// isSpace reports whether k is a plain space, however the terminal sent it.
func isSpace(k tui.Key) bool {
	if k.Type == tui.KeySpace {
		return true
	}
	return k.Type == tui.KeyRunes && k.Text == " " && !k.Mod.Alt()
}

func (in fieldInput) value() string {
	switch in.kind {
	case FieldSelect:
		if in.choice >= 0 && in.choice < len(in.options) {
			return in.options[in.choice]
		}
		return ""
	case FieldCheckbox:
		if in.checked {
			return "true"
		}
		return "false"
	}
	if in.isPass {
		return in.secret.Value()
	}
	return in.text.Value()
}

func (in fieldInput) view(th theme.Theme) string {
	switch in.kind {
	case FieldSelect:
		return in.selectView(th)
	case FieldCheckbox:
		return widgets.Checkbox(in.label, in.checked, in.focused, th)
	}
	if in.isPass {
		return in.secret.View()
	}
	return in.text.View()
}

// linearize is the field's plain-text description for accessible output: its
// label, what kind of field it is, ", focused" when it has focus, and its
// state. A select speaks its chosen option and position ("Pro, 2 of 3"), a
// checkbox "checked" or "unchecked"; text and secret fields speak as their own
// widgets do (a secret never its value).
func (in fieldInput) linearize() string {
	head := func(kind string) string {
		s := kind
		if in.label != "" {
			s = in.label + ", " + kind
		}
		if in.focused {
			s += ", focused"
		}
		return s
	}
	switch in.kind {
	case FieldSelect:
		if len(in.options) == 0 {
			return head("select") + ", no options"
		}
		return head("select") + ", " + in.value() + ", " + strconv.Itoa(in.choice+1) + " of " + strconv.Itoa(len(in.options))
	case FieldCheckbox:
		if in.checked {
			return head("checkbox") + ", checked"
		}
		return head("checkbox") + ", unchecked"
	}
	if in.isPass {
		return in.secret.Linearize()
	}
	return in.text.Linearize()
}

// layoutNode is the field's input as a layout node: the text widgets' own
// nodes, and for a select or a checkbox its one-row view, clipped or padded to
// the width it is given.
func (in fieldInput) layoutNode(th theme.Theme) layout.Node {
	switch in.kind {
	case FieldSelect, FieldCheckbox:
		return layout.Block(in.view(th))
	}
	if in.isPass {
		return in.secret.LayoutNode()
	}
	return in.text.LayoutNode()
}

// selectView draws "Label: < option >", the option in the focus colour while
// the field has focus.
func (in fieldInput) selectView(th theme.Theme) string {
	prompt := in.label + ": "
	if len(in.options) == 0 {
		return prompt + "(no options)"
	}
	choice := "< " + in.value() + " >"
	if in.focused {
		choice = th.ResolvedStates().Focus.Bold().Render(choice)
	}
	return prompt + choice
}

// Model is a form. Build one with New.
type Model struct {
	// Theme colors the error lines. New sets theme.DarkTheme().
	Theme theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
	// KeyMap holds the keys for submitting and for editing select and
	// checkbox fields. New fills it with DefaultKeyMap; a Model with a zero
	// KeyMap behaves as if it held DefaultKeyMap. Tab and Shift+Tab move
	// between fields through focus.Route and are not rebindable here.
	KeyMap KeyMap

	// Mouse, when true, makes a focused form's Update handle tui.MouseEvent: a
	// left click on a field focuses it; on a checkbox it also toggles it, on a
	// select it chooses the previous option (left of the option's middle) or
	// the next one (right of it). The wheel over a select chooses the previous
	// or next option. Off (the default) ignores the mouse.
	Mouse bool
	// Bounds is the screen rectangle where the app draws the form (its first
	// row is the first field); mouse events outside it are ignored.
	Bounds hittest.Rect

	fields  []Field
	inputs  []fieldInput
	errs    []string // current error per field, "" when none
	ring    focus.Ring
	focused bool
}

// KeyMap names the keys of each action of a Model.
type KeyMap struct {
	Submit     keymap.Binding // check every field and submit
	SelectPrev keymap.Binding // choose the previous option of a select
	SelectNext keymap.Binding // choose the next option of a select
	Toggle     keymap.Binding // tick or untick a checkbox
}

// DefaultKeyMap returns the keys a Model used before KeyMap existed.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Submit:     keymap.NewBinding("submit", "enter"),
		SelectPrev: keymap.NewBinding("previous option", "left", "up"),
		SelectNext: keymap.NewBinding("next option", "right", "down"),
		Toggle:     keymap.NewBinding("toggle", "space"),
	}
}

func (m Model) keys() KeyMap {
	km := m.KeyMap
	if len(km.Submit.Keys)+len(km.SelectPrev.Keys)+len(km.SelectNext.Keys)+len(km.Toggle.Keys) == 0 {
		return DefaultKeyMap()
	}
	return km
}

// Bindings returns the actions active for the field that has focus, with
// descriptions, for help text: Submit, the fixed Tab and Shift+Tab field
// movement, then the focused field's own keys (an option choice for a select,
// Toggle for a checkbox, the editing keys of its input for a text field).
func (m Model) Bindings() []keymap.Binding {
	km := m.keys()
	out := []keymap.Binding{
		km.Submit,
		keymap.NewBinding("next field", "tab"),
		keymap.NewBinding("previous field", "shift+tab"),
	}
	i := m.ring.Current()
	if i < 0 || i >= len(m.inputs) {
		return out
	}
	switch in := m.inputs[i]; in.kind {
	case FieldSelect:
		out = append(out, km.SelectPrev, km.SelectNext)
	case FieldCheckbox:
		out = append(out, km.Toggle)
	default:
		if in.isPass {
			out = append(out, in.secret.Bindings()...)
		} else {
			out = append(out, in.text.Bindings()...)
		}
	}
	return out
}

// New returns a form of the given fields. It is not focused: call Focus.
func New(fields ...Field) Model {
	m := Model{
		Theme:  theme.DarkTheme(),
		KeyMap: DefaultKeyMap(),
		fields: append([]Field(nil), fields...),
		inputs: make([]fieldInput, len(fields)),
		errs:   make([]string, len(fields)),
		ring:   focus.New(len(fields)),
	}
	for i, f := range fields {
		switch f.Kind {
		case FieldSelect:
			in := fieldInput{kind: FieldSelect, label: f.Label, options: append([]string(nil), f.Options...)}
			for j, o := range f.Options {
				if o == f.Value {
					in.choice = j
					break
				}
			}
			m.inputs[i] = in
			continue
		case FieldCheckbox:
			m.inputs[i] = fieldInput{kind: FieldCheckbox, label: f.Label, checked: strings.EqualFold(f.Value, "true")}
			continue
		}
		w := f.Width
		if w <= 0 {
			w = DefaultWidth
		}
		prompt := f.Label + ": "
		if f.Secret {
			p := passwordinput.New()
			p.Prompt, p.Placeholder, p.Width = prompt, f.Placeholder, w
			p.SetValue(f.Value)
			m.inputs[i] = fieldInput{secret: p, isPass: true}
			continue
		}
		t := textinput.New()
		t.Prompt, t.Placeholder, t.Width = prompt, f.Placeholder, w
		t.SetValue(f.Value)
		m.inputs[i] = fieldInput{text: t}
	}
	return m
}

// clone copies the slices a change would write through, so the receiver of
// Update or Submit is never modified.
func (m Model) clone() Model {
	m.inputs = append([]fieldInput(nil), m.inputs...)
	m.errs = append([]string(nil), m.errs...)
	return m
}

// bindings adapts each field's widget to focus.Field. It must be called on the
// Model that will be returned, since the closures write into its slices.
func (m *Model) bindings() []focus.Field {
	fs := make([]focus.Field, len(m.inputs))
	for i := range m.inputs {
		i := i
		fs[i] = focus.Field{
			Focus:  func() tui.Cmd { return m.inputs[i].focus() },
			Blur:   func() { m.inputs[i].blur() },
			Update: func(msg tui.Msg) tui.Cmd { return m.inputs[i].update(msg, m.keys()) },
		}
	}
	return fs
}

// Focus focuses the form, and the field that has focus within it, returning
// the Cmd that field needs (its cursor blink). Run the Cmd.
func (m *Model) Focus() tui.Cmd {
	m.inputs = append([]fieldInput(nil), m.inputs...)
	m.focused = true
	return m.ring.Sync(m.bindings()...)
}

// Blur removes focus from the form and every field.
func (m *Model) Blur() {
	m.inputs = append([]fieldInput(nil), m.inputs...)
	m.focused = false
	for i := range m.inputs {
		m.inputs[i].blur()
	}
}

// Focused reports whether the form has focus.
func (m Model) Focused() bool { return m.focused }

// Current returns the Name of the field that has focus within the form, or ""
// for a form with no fields.
func (m Model) Current() string {
	if i := m.ring.Current(); i >= 0 && i < len(m.fields) {
		return m.fields[i].Name
	}
	return ""
}

// Values returns each field's current value, keyed by Name. A Secret field's
// real value is included: it is what the application asked for.
func (m Model) Values() map[string]string {
	v := make(map[string]string, len(m.fields))
	for i, f := range m.fields {
		v[f.Name] = m.inputs[i].value()
	}
	return v
}

// Err returns the error currently shown for the field called name, or "".
func (m Model) Err(name string) string {
	for i, f := range m.fields {
		if f.Name == name {
			return m.errs[i]
		}
	}
	return ""
}

// check runs field i's validators in order and returns the first error's
// text, or "" when all pass.
func (m Model) check(i int) string {
	value := m.inputs[i].value()
	for _, v := range m.fields[i].Validators {
		if err := v(value); err != nil {
			return err.Error()
		}
	}
	return ""
}

// Update routes msg. Tab and Shift+Tab move focus between fields, Enter
// submits, and every other message goes to the focused field, after which
// that field's error, if one is showing, is re-checked. With Mouse on it also
// handles tui.MouseEvent (see Model.Mouse). A form without focus ignores all
// messages.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if !m.focused || len(m.inputs) == 0 {
		return m, nil
	}
	if ev, ok := msg.(tui.MouseEvent); ok {
		return m.updateMouse(ev)
	}
	if keymap.Matches(msg, m.keys().Submit) {
		return m.Submit()
	}
	m = m.clone()
	edited := m.ring.Current()
	ring, cmd := m.ring.Route(msg, m.bindings()...)
	m.ring = ring
	if edited >= 0 && m.errs[edited] != "" {
		m.errs[edited] = m.check(edited)
	}
	return m, cmd
}

// fieldAtRow maps a row of the rendered View (a field's line, or the error
// line under it) to the field it belongs to, or -1 below the last field.
func (m Model) fieldAtRow(row int) int {
	y := 0
	for i := range m.inputs {
		n := 1
		if m.errs[i] != "" {
			n = 2
		}
		if row < y+n {
			return i
		}
		y += n
	}
	return -1
}

// updateMouse applies a mouse press inside Bounds when Mouse is on.
func (m Model) updateMouse(ev tui.MouseEvent) (Model, tui.Cmd) {
	if !m.Mouse || !m.Bounds.Contains(ev.X, ev.Y) || ev.Action != tui.MouseActionPress {
		return m, nil
	}
	col, row := m.Bounds.Local(ev.X, ev.Y)
	i := m.fieldAtRow(row)
	if i < 0 {
		return m, nil
	}
	switch ev.Button {
	case tui.MouseButtonWheelUp, tui.MouseButtonWheelDown:
		if m.inputs[i].kind != FieldSelect {
			return m, nil
		}
		m = m.clone()
		m.inputs[i].step(ev.Button == tui.MouseButtonWheelDown)
		return m, nil
	case tui.MouseButtonLeft:
	default:
		return m, nil
	}
	m = m.clone()
	var cmd tui.Cmd
	if i != m.ring.Current() && m.ring.Enabled(i) {
		fs := m.bindings()
		if cur := m.ring.Current(); cur >= 0 {
			m.inputs[cur].blur()
		}
		m.ring = m.ring.Set(i)
		cmd = fs[i].Focus()
	}
	switch in := &m.inputs[i]; in.kind {
	case FieldCheckbox:
		in.checked = !in.checked
	case FieldSelect:
		in.step(col >= ansi.Width(in.label+": ")+(ansi.Width(in.value())+4)/2)
	}
	if m.errs[i] != "" {
		m.errs[i] = m.check(i)
	}
	return m, cmd
}

// step chooses the next (or previous) option of a select, stopping at the ends.
func (in *fieldInput) step(next bool) {
	if next && in.choice < len(in.options)-1 {
		in.choice++
	} else if !next && in.choice > 0 {
		in.choice--
	}
}

// Submit checks every field. If any fails, each failing field shows its first
// error, focus moves to the first invalid field and no SubmittedMsg is
// produced. If all pass, the returned Cmd yields a SubmittedMsg with the
// values.
func (m Model) Submit() (Model, tui.Cmd) {
	m = m.clone()
	first := -1
	for i := range m.fields {
		m.errs[i] = m.check(i)
		if m.errs[i] != "" && first < 0 {
			first = i
		}
	}
	if first < 0 {
		values := m.Values()
		return m, func() tui.Msg { return SubmittedMsg{Values: values} }
	}
	m.ring = m.ring.Set(first)
	if !m.focused {
		return m, nil
	}
	return m, m.ring.Sync(m.bindings()...)
}

// View renders each field on its own line, followed by its error, if any,
// on the next line.
func (m Model) View() string {
	errStyle := ansi.NewStyle().Foreground(m.themed().Error)
	var b strings.Builder
	for i := range m.inputs {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(m.inputs[i].view(m.themed()))
		if m.errs[i] != "" {
			b.WriteString("\n  " + errStyle.Render(m.errs[i]))
		}
	}
	return b.String()
}

// Compile-time proof that Model satisfies tui.Component[Model].
var _ tui.Component[Model] = Model{}
