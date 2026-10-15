package form

import (
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/multiselect"
	"github.com/ows4444/tui/textarea"
	"github.com/ows4444/tui/theme"
)

// defaultAreaHeight is the rows of a FieldTextArea whose Height is 0.
const defaultAreaHeight = 4

// newMultiInput builds the widget of a FieldMultiSelect: the options as a
// list, with those named in the comma-separated Field.Value ticked.
func newMultiInput(f Field) fieldInput {
	ms := multiselect.NewStrings(f.Options...)
	ticked := map[string]bool{}
	for _, v := range strings.Split(f.Value, ",") {
		ticked[strings.TrimSpace(v)] = true
	}
	for i, o := range f.Options {
		if ticked[o] {
			ms.Toggle(i)
		}
	}
	return fieldInput{kind: FieldMultiSelect, label: f.Label, options: append([]string(nil), f.Options...), multi: ms}
}

// newAreaInput builds the widget of a FieldTextArea.
func newAreaInput(f Field) fieldInput {
	a := textarea.New()
	a.Width = f.Width
	if a.Width <= 0 {
		a.Width = DefaultWidth
	}
	a.Height = f.Height
	if a.Height <= 0 {
		a.Height = defaultAreaHeight
	}
	a.SoftWrap = true
	a.Placeholder = f.Placeholder
	a.SetValue(f.Value)
	return fieldInput{kind: FieldTextArea, label: f.Label, area: a}
}

// multiValue is the ticked options in list order, joined with ",".
func (in fieldInput) multiValue() string {
	var out []string
	for _, i := range in.multi.SelectedIndexes() {
		if i >= 0 && i < len(in.options) {
			out = append(out, in.options[i])
		}
	}
	return strings.Join(out, ",")
}

// Selected returns the ticked options of the FieldMultiSelect called name, in
// list order; nil when there is no such field, it is not a multi-select, or
// nothing is ticked.
func Selected(m Model, name string) []string {
	i := m.index(name)
	if i < 0 || m.fields[i].Kind != FieldMultiSelect {
		return nil
	}
	var out []string
	for _, j := range m.inputs[i].multi.SelectedIndexes() {
		if j >= 0 && j < len(m.inputs[i].options) {
			out = append(out, m.inputs[i].options[j])
		}
	}
	return out
}

// blockView draws a multi-row field: its label on a line, and the widget
// under it indented by two columns.
func (in fieldInput) blockView() string {
	var body string
	if in.kind == FieldMultiSelect {
		body = in.multi.View()
	} else {
		body = in.area.View()
	}
	head := in.label + ":"
	if in.focused {
		head = ansi.NewStyle().Bold().Render(head)
	}
	lines := strings.Split(body, "\n")
	if in.kind == FieldTextArea {
		// A text area draws as many rows as it has lines, up to its Height;
		// keep its box that tall so the fields under it do not move as you type.
		for len(lines) < in.area.Height {
			lines = append(lines, "")
		}
	}
	for i := range lines {
		lines[i] = "  " + lines[i]
	}
	return head + "\n" + strings.Join(lines, "\n")
}

// blockLinearize is the spoken form of a multi-row field.
func (in fieldInput) blockLinearize(head func(kind string) string) string {
	if in.kind == FieldMultiSelect {
		return head("multi-select") + "\n" + in.multi.Linearize()
	}
	n := in.area.LineCount()
	s := head("text area") + ", " + strconv.Itoa(n) + " line"
	if n != 1 {
		s += "s"
	}
	if v := in.area.Value(); v != "" {
		s += "\n" + v
	}
	return s
}

// rows is how many lines the field takes: its widget and its description.
func (in fieldInput) rows(th theme.Theme) int {
	n := strings.Count(in.view(th), "\n") + 1
	if in.desc != "" {
		n++
	}
	return n
}

// descLine is the field's description as it is drawn, or "".
func (in fieldInput) descLine() string {
	if in.desc == "" {
		return ""
	}
	return "  " + ansi.NewStyle().Faint().Render(ansi.Sanitize(in.desc))
}

// inArea reports whether the field with focus is a text area, where Enter adds
// a line instead of submitting.
func (m Model) inArea() bool {
	i := m.ring.Current()
	return i >= 0 && i < len(m.inputs) && m.inputs[i].kind == FieldTextArea
}

// isEnter reports whether msg is the Enter key.
func isEnter(msg tui.Msg) bool {
	k, ok := msg.(tui.Key)
	return ok && k.Type == tui.KeyEnter
}
