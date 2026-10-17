package form

import (
	"strconv"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/filepicker"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/multiselect"
	"github.com/ows4444/tui/passwordinput"
	"github.com/ows4444/tui/textarea"
	"github.com/ows4444/tui/textinput"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

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

	multi multiselect.Model // FieldMultiSelect
	area  textarea.Model    // FieldTextArea
	desc  string            // Field.Description

	picker   filepicker.Model // FieldFile: the browser, valid while browsing
	browsing bool             // FieldFile: the browser is open
	exts     []string         // FieldFile: Field.Options
}

func (in *fieldInput) focus() tui.Cmd {
	switch in.kind {
	case FieldSelect, FieldCheckbox, FieldMultiSelect, FieldConfirm:
		in.focused = true
		return nil
	case FieldNote:
		return nil
	case FieldTextArea:
		in.focused = true
		return in.area.Focus()
	}
	if in.isPass {
		return in.secret.Focus()
	}
	return in.text.Focus()
}

func (in *fieldInput) blur() {
	switch in.kind {
	case FieldSelect, FieldCheckbox, FieldMultiSelect, FieldConfirm:
		in.focused = false
		return
	case FieldNote:
		return
	case FieldFile:
		in.browsing = false
	case FieldTextArea:
		in.focused = false
		in.area.Blur()
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
	case FieldConfirm:
		in.updateConfirm(msg, km)
		return nil
	case FieldNote:
		return nil
	case FieldFile:
		if cmd, handled := in.updateFile(msg, km); handled {
			return cmd
		}
	case FieldMultiSelect:
		in.multi, _ = in.multi.Update(msg)
		return nil
	case FieldTextArea:
		var cmd tui.Cmd
		in.area, cmd = in.area.Update(msg)
		return cmd
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
	case FieldCheckbox, FieldConfirm:
		if in.checked {
			return "true"
		}
		return "false"
	case FieldNote:
		return ""
	case FieldMultiSelect:
		return in.multiValue()
	case FieldTextArea:
		return in.area.Value()
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
	case FieldConfirm:
		return in.confirmView(th)
	case FieldNote:
		return ansi.NewStyle().Faint().Render(ansi.Sanitize(in.label))
	case FieldFile:
		return in.fileView()
	case FieldMultiSelect, FieldTextArea:
		return in.blockView()
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
	case FieldConfirm:
		if in.checked {
			return head("confirm") + ", yes"
		}
		return head("confirm") + ", no"
	case FieldNote:
		return "note, " + in.label
	case FieldFile:
		s := in.text.Linearize()
		if in.browsing {
			s += ", browsing\n" + in.picker.Linearize()
		}
		return s
	case FieldMultiSelect, FieldTextArea:
		return in.blockLinearize(head)
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
	case FieldSelect, FieldCheckbox, FieldMultiSelect, FieldTextArea, FieldConfirm, FieldNote:
		return layout.Block(in.view(th))
	case FieldFile:
		if in.browsing {
			return layout.Block(in.view(th))
		}
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
