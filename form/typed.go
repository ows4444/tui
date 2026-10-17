package form

import (
	"slices"
	"strconv"
	"strings"
)

// Option is one choice of a SelectOf field: the Label shown and the Value the
// program reads back.
type Option[T any] struct {
	Label string
	Value T
}

// SelectOf returns a FieldSelect whose choices carry a value of any type T.
// The field's string value, in Values and SubmittedMsg.Values, is the chosen
// Label; read the typed value with Value[T]. Set Field.Value to a Label to
// start on that choice.
func SelectOf[T any](name, label string, options ...Option[T]) Field {
	labels := make([]string, len(options))
	for i, o := range options {
		labels[i] = o.Label
	}
	return Field{
		Name:    name,
		Label:   label,
		Kind:    FieldSelect,
		Options: labels,
		choices: slices.Clone(options),
	}
}

// Value returns the typed value of the option chosen in the SelectOf[T] field
// called name. ok is false when there is no such field, the field was not
// built by SelectOf[T] (a different T counts), its group is hidden, or it has
// no option.
func Value[T any](m Model, name string) (v T, ok bool) {
	i := m.index(name)
	if i < 0 || m.hidden(i) {
		return v, false
	}
	opts, isOf := m.fields[i].choices.([]Option[T])
	if !isOf {
		return v, false
	}
	c := m.inputs[i].choice
	if c < 0 || c >= len(opts) {
		return v, false
	}
	return opts[c].Value, true
}

// Bool returns whether the checkbox or confirm field called name is on; false
// when there is no such field or it is neither.
func Bool(m Model, name string) bool {
	i := m.index(name)
	return i >= 0 && (m.fields[i].Kind == FieldCheckbox || m.fields[i].Kind == FieldConfirm) && m.inputs[i].checked
}

// Int returns the value of the field called name as an integer, trimming
// spaces. It returns an error when there is no such field or the text is not
// an integer, so pair it with a validator that has already said so.
func Int(m Model, name string) (int, error) {
	i := m.index(name)
	if i < 0 {
		return 0, &unknownFieldError{name}
	}
	return strconv.Atoi(strings.TrimSpace(m.inputs[i].value()))
}

type unknownFieldError struct{ name string }

func (e *unknownFieldError) Error() string { return "form: no field named " + strconv.Quote(e.name) }

// index is the position of the field called name, or -1.
func (m Model) index(name string) int {
	for i, f := range m.fields {
		if f.Name == name {
			return i
		}
	}
	return -1
}

// hidden reports whether field i belongs to a hidden group.
func (m Model) hidden(i int) bool {
	if len(m.groups) == 0 {
		return false
	}
	g := m.groupOf(i)
	return g < 0 || !contains(m.visibleGroups(), g)
}

// refreshOptions asks every OptionsFunc for its options and updates the
// selects whose options changed. It leaves the receiver's slices alone: a
// change is made on a copy.
func (m Model) refreshOptions() Model {
	var vals map[string]string
	cloned := false
	for i, f := range m.fields {
		if f.OptionsFunc == nil || f.Kind != FieldSelect {
			continue
		}
		if vals == nil {
			vals = m.allValues()
		}
		opts := f.OptionsFunc(vals)
		in := m.inputs[i]
		if slices.Equal(opts, in.options) {
			continue
		}
		if !cloned {
			m = m.clone()
			cloned = true
		}
		kept := in.value()
		in.options = slices.Clone(opts)
		in.choice = 0
		if j := slices.Index(opts, kept); j >= 0 {
			in.choice = j
		}
		m.inputs[i] = in
		vals = m.allValues() // a later field may depend on this one
	}
	return m
}
