package form

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/filepicker"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/theme"
)

// browserRows is how many entries a FieldFile's list shows at once.
const browserRows = 6

// updateConfirm handles a key on a FieldConfirm: Y and N choose, SelectPrev
// (Left) chooses Yes, SelectNext (Right) chooses No, Toggle flips.
func (in *fieldInput) updateConfirm(msg tui.Msg, km KeyMap) {
	k, isKey := msg.(tui.Key)
	switch {
	case isKey && k.Type == tui.KeyRunes && !k.Mod.Ctrl() && !k.Mod.Alt() && (k.Text == "y" || k.Text == "Y"):
		in.checked = true
	case isKey && k.Type == tui.KeyRunes && !k.Mod.Ctrl() && !k.Mod.Alt() && (k.Text == "n" || k.Text == "N"):
		in.checked = false
	case keymap.Matches(msg, km.SelectPrev):
		in.checked = true
	case keymap.Matches(msg, km.SelectNext):
		in.checked = false
	case toggles(msg, km.Toggle):
		in.checked = !in.checked
	}
}

// confirmView draws "Label: [Yes]  No" or "Label:  Yes  [No]", the chosen one
// in the focus colour while the field has focus.
func (in fieldInput) confirmView(th theme.Theme) string {
	yes, no := " Yes ", " No "
	if in.checked {
		yes = "[Yes]"
	} else {
		no = "[No]"
	}
	chosen := yes
	if !in.checked {
		chosen = no
	}
	if in.focused {
		styled := th.ResolvedStates().Focus.Bold().Render(chosen)
		if in.checked {
			yes = styled
		} else {
			no = styled
		}
	}
	return in.label + ": " + yes + " " + no
}

// updateFile handles the keys of a FieldFile. It reports handled when the key
// was the field's own: the browse key, or any key while the list is open.
func (in *fieldInput) updateFile(msg tui.Msg, km KeyMap) (tui.Cmd, bool) {
	if !in.browsing {
		if !keymap.Matches(msg, km.Browse) {
			return nil, false
		}
		dir := "."
		if p := strings.TrimSpace(in.text.Value()); p != "" {
			if st, err := os.Stat(p); err == nil && st.IsDir() {
				dir = p
			} else if d := filepath.Dir(p); d != "" {
				if st, err := os.Stat(d); err == nil && st.IsDir() {
					dir = d
				}
			}
		}
		pk := filepicker.New(dir)
		pk.Extensions, pk.Height = in.exts, browserRows
		in.picker, in.browsing = pk.Reload(), true
		return nil, true
	}
	if k, ok := msg.(tui.Key); ok && k.Type == tui.KeyEsc {
		in.browsing = false
		return nil, true
	}
	next, cmd := in.picker.Update(msg)
	in.picker = next
	if cmd != nil {
		// The picker answers a confirmed file with a SelectedMsg from its Cmd;
		// reading it here keeps the choice in the same Update.
		if sel, ok := cmd().(filepicker.SelectedMsg); ok {
			in.text.SetValue(sel.Path)
			in.browsing = false
		}
	}
	return nil, true
}

// fileView is the path field, and under it the file list while it is open.
func (in fieldInput) fileView() string {
	if !in.browsing {
		return in.text.View()
	}
	lines := strings.Split(in.picker.View(), "\n")
	for i := range lines {
		lines[i] = "  " + lines[i]
	}
	head := ansi.NewStyle().Faint().Render("  " + ansi.Sanitize(in.picker.Dir) + " (Enter picks, Esc closes)")
	return in.text.View() + "\n" + head + "\n" + strings.Join(lines, "\n")
}

// browsingFile reports whether the field with focus is a FieldFile whose list
// is open.
func (m Model) browsingFile() bool {
	i := m.ring.Current()
	return i >= 0 && i < len(m.inputs) && m.inputs[i].kind == FieldFile && m.inputs[i].browsing
}

// skipNotes takes the notes out of the focus order, and puts focus on the
// first field that can have it.
func (m *Model) skipNotes() {
	first := -1
	for i, f := range m.fields {
		if f.Kind == FieldNote {
			m.ring = m.ring.SetDisabled(i, true)
		} else if first < 0 {
			first = i
		}
	}
	if first >= 0 {
		m.ring = m.ring.Set(first)
	}
}
