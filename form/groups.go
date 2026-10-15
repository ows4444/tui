package form

import (
	"strconv"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
)

// Group is a titled run of fields shown together by NewGroups.
type Group struct {
	// Title is shown above the group's fields.
	Title string
	// Description is one line of help under the title.
	Description string
	// Fields of the group. Names must be unique across every group.
	Fields []Field
	// Hide, when set, is asked with the current values of every field and
	// reports whether the group does not apply. A hidden group is skipped by
	// navigation and by Submit, and its values are left out of Values and
	// SubmittedMsg.Values.
	Hide func(values map[string]string) bool
}

// group is a Group once its fields are in the flat field list: it owns
// fields lo up to hi.
type group struct {
	title, desc string
	lo, hi      int
	hide        func(values map[string]string) bool
}

// NewGroups returns a form of the given groups, showing the first one that
// is not hidden. It is not focused: call Focus. A group with no fields is
// kept, and shows only its title.
func NewGroups(groups ...Group) Model {
	var all []Field
	spans := make([]group, len(groups))
	for i, g := range groups {
		spans[i] = group{title: g.Title, desc: g.Description, lo: len(all), hide: g.Hide}
		all = append(all, g.Fields...)
		spans[i].hi = len(all)
	}
	m := New(all...)
	if len(groups) == 0 {
		return m
	}
	m.groups = spans
	if v := m.visibleGroups(); len(v) > 0 {
		m.cur = v[0]
	}
	m.syncRing()
	return m
}

// span is the run of fields showing: fields lo up to hi.
func (m Model) span() (lo, hi int) {
	if len(m.groups) == 0 {
		return 0, len(m.fields)
	}
	g := m.groups[m.cur]
	return g.lo, g.hi
}

// allValues is every field's value, hidden groups included, for Hide.
func (m Model) allValues() map[string]string {
	v := make(map[string]string, len(m.fields))
	for i, f := range m.fields {
		v[f.Name] = m.inputs[i].value()
	}
	return v
}

// visibleGroups lists, in order, the groups that are not hidden.
func (m Model) visibleGroups() []int {
	var out []int
	var vals map[string]string
	for i, g := range m.groups {
		if g.hide != nil {
			if vals == nil {
				vals = m.allValues()
			}
			if g.hide(vals) {
				continue
			}
		}
		out = append(out, i)
	}
	return out
}

// groupOf is the group that owns field i, or -1.
func (m Model) groupOf(i int) int {
	for g, gr := range m.groups {
		if i >= gr.lo && i < gr.hi {
			return g
		}
	}
	return -1
}

// CurrentGroup is the index, among the groups given to NewGroups, of the
// group showing; 0 for a form built with New.
func (m Model) CurrentGroup() int { return m.cur }

// GroupCount is the number of groups given to NewGroups that are not hidden
// right now; 1 for a form built with New.
func (m Model) GroupCount() int {
	if len(m.groups) == 0 {
		return 1
	}
	return len(m.visibleGroups())
}

// syncRing makes the focus ring skip every field outside the group showing.
// It enables the group first so the ring never has to move focus through a
// run that is about to be disabled.
func (m *Model) syncRing() {
	lo, hi := m.span()
	for i := range m.fields {
		if i >= lo && i < hi {
			m.ring = m.ring.SetDisabled(i, false)
		}
	}
	for i := range m.fields {
		if i < lo || i >= hi {
			m.ring = m.ring.SetDisabled(i, true)
		}
	}
}

// goTo shows group g with focus on its first field, returning the Cmd the
// focused field needs.
func (m Model) goTo(g int) (Model, tui.Cmd) {
	m = m.clone()
	m.cur = g
	m.syncRing()
	lo, _ := m.span()
	m.ring = m.ring.Set(lo)
	if !m.focused {
		return m, nil
	}
	return m, m.ring.Sync(m.bindings()...)
}

// neighbour returns the nearest visible group after (dir 1) or before (dir
// -1) the one showing, or -1.
func (m Model) neighbour(dir int) int {
	v := m.visibleGroups()
	for i := range v {
		j := i
		if dir < 0 {
			j = len(v) - 1 - i
		}
		if (dir > 0 && v[j] > m.cur) || (dir < 0 && v[j] < m.cur) {
			return v[j]
		}
	}
	return -1
}

// checkSpan checks fields lo up to hi, returning the first invalid one or -1.
func (m *Model) checkSpan(lo, hi int) int {
	first := -1
	for i := lo; i < hi; i++ {
		m.errs[i] = m.check(i)
		if m.errs[i] != "" && first < 0 {
			first = i
		}
	}
	return first
}

// advance checks the group showing and, when it is valid, shows the next one.
func (m Model) advance() (Model, tui.Cmd) {
	next := m.neighbour(1)
	if next < 0 {
		return m, nil
	}
	m = m.clone()
	lo, hi := m.span()
	if first := m.checkSpan(lo, hi); first >= 0 {
		m.ring = m.ring.Set(first)
		if !m.focused {
			return m, nil
		}
		return m, m.ring.Sync(m.bindings()...)
	}
	return m.goTo(next)
}

// header returns the group's title line and description, or "" for each that
// is not shown, with the position among the visible groups.
func (m Model) header() (title, desc string, pos, total int) {
	if len(m.groups) == 0 {
		return "", "", 0, 0
	}
	v := m.visibleGroups()
	for i, g := range v {
		if g == m.cur {
			pos = i + 1
		}
	}
	g := m.groups[m.cur]
	return g.title, g.desc, pos, len(v)
}

// headerRows are the styled lines drawn above the group's fields.
func (m Model) headerRows() []string {
	title, desc, pos, total := m.header()
	if title == "" && desc == "" && total <= 1 {
		return nil
	}
	var rows []string
	t := ansi.NewStyle().Bold().Render(title)
	if total > 1 {
		step := "step " + strconv.Itoa(pos) + " of " + strconv.Itoa(total)
		if title == "" {
			t = ansi.NewStyle().Faint().Render(step)
		} else {
			t += ansi.NewStyle().Faint().Render("  " + step)
		}
	}
	if title != "" || total > 1 {
		rows = append(rows, t)
	}
	if desc != "" {
		rows = append(rows, ansi.NewStyle().Faint().Render(desc))
	}
	return rows
}

func contains(xs []int, x int) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
