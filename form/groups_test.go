package form

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/layout"
)

func enter(m Model) (Model, tui.Cmd) { return m.Update(tui.Key{Type: tui.KeyEnter}) }

func pgdn(m Model) Model { m, _ = m.Update(tui.Key{Type: tui.KeyPgDown}); return m }

func pgup(m Model) Model { m, _ = m.Update(tui.Key{Type: tui.KeyPgUp}); return m }

func wizardForm() Model {
	m := NewGroups(
		Group{Title: "Account", Description: "Who you are", Fields: []Field{
			{Name: "name", Label: "Name", Validators: []Validator{Required()}},
			{Name: "email", Label: "Email", Validators: []Validator{Required(), Email()}},
		}},
		Group{Title: "Plan", Fields: []Field{
			{Name: "plan", Label: "Plan", Kind: FieldSelect, Options: []string{"Free", "Pro"}},
		}},
		Group{Title: "Billing", Fields: []Field{
			{Name: "card", Label: "Card", Validators: []Validator{Required()}},
		}, Hide: func(v map[string]string) bool { return v["plan"] != "Pro" }},
		Group{Title: "Finish", Fields: []Field{
			{Name: "terms", Label: "Accept terms", Kind: FieldCheckbox},
		}},
	)
	m.Focus()
	return m
}

func TestGroupsShowOnlyTheFirstGroup(t *testing.T) {
	m := wizardForm()
	v := plain(m)
	for _, want := range []string{"Account", "step 1 of 3", "Who you are", "Name", "Email"} {
		if !strings.Contains(v, want) {
			t.Errorf("view lacks %q:\n%s", want, v)
		}
	}
	for _, not := range []string{"Plan", "Card", "Accept terms"} {
		if strings.Contains(v, not) {
			t.Errorf("view shows %q of another group:\n%s", not, v)
		}
	}
	if m.CurrentGroup() != 0 || m.GroupCount() != 3 || m.Current() != "name" {
		t.Errorf("group %d of %d, field %q", m.CurrentGroup(), m.GroupCount(), m.Current())
	}
}

func TestTabStaysInsideTheGroup(t *testing.T) {
	m := wizardForm()
	m = tab(m)
	if m.Current() != "email" {
		t.Fatalf("after Tab: %q", m.Current())
	}
	m = tab(m) // wraps to the first field of the same group
	if m.Current() != "name" || m.CurrentGroup() != 0 {
		t.Errorf("after second Tab: %q in group %d", m.Current(), m.CurrentGroup())
	}
}

func TestEnterChecksTheGroupBeforeMovingOn(t *testing.T) {
	m := wizardForm()
	m, cmd := enter(m)
	if cmd != nil && func() bool { _, ok := cmd().(SubmittedMsg); return ok }() {
		t.Fatal("Enter on the first group submitted the form")
	}
	if m.CurrentGroup() != 0 || m.Err("name") == "" {
		t.Errorf("invalid group: moved to %d, name error %q", m.CurrentGroup(), m.Err("name"))
	}
	if m.Err("card") != "" {
		t.Error("a field of another group was checked")
	}
	m = typed(m, "Ada")
	m = tab(m)
	m = typed(m, "ada@example.com")
	m, _ = enter(m)
	if m.CurrentGroup() != 1 || m.Current() != "plan" {
		t.Errorf("valid group: now on group %d field %q", m.CurrentGroup(), m.Current())
	}
	if v := plain(m); !strings.Contains(v, "step 2 of 3") || strings.Contains(v, "Email") {
		t.Errorf("view:\n%s", v)
	}
}

func TestPageKeysMoveBetweenGroups(t *testing.T) {
	m := wizardForm()
	m = pgdn(m) // invalid: stays, shows the error
	if m.CurrentGroup() != 0 || m.Err("name") == "" {
		t.Errorf("PgDn on an invalid group moved to %d", m.CurrentGroup())
	}
	m = typed(m, "Ada")
	m = tab(m)
	m = typed(m, "ada@example.com")
	m = pgdn(m)
	if m.CurrentGroup() != 1 {
		t.Fatalf("PgDn on a valid group: %d", m.CurrentGroup())
	}
	m = pgup(m)
	if m.CurrentGroup() != 0 || m.Current() != "name" {
		t.Errorf("PgUp: group %d field %q", m.CurrentGroup(), m.Current())
	}
	m = pgup(m) // already first
	if m.CurrentGroup() != 0 {
		t.Errorf("PgUp on the first group: %d", m.CurrentGroup())
	}
}

func fillAccount(m Model) Model {
	m = typed(m, "Ada")
	m = tab(m)
	m = typed(m, "ada@example.com")
	m, _ = enter(m)
	return m
}

func TestHiddenGroupIsSkippedAndItsValuesLeftOut(t *testing.T) {
	m := fillAccount(wizardForm()) // now on Plan, Free: Billing is hidden
	if m.GroupCount() != 3 {
		t.Fatalf("GroupCount = %d, want 3 (Billing hidden)", m.GroupCount())
	}
	m, _ = enter(m)
	if m.CurrentGroup() != 3 {
		t.Fatalf("Enter on Plan went to group %d, want 3 (Finish)", m.CurrentGroup())
	}
	if !strings.Contains(plain(m), "step 3 of 3") {
		t.Errorf("view:\n%s", plain(m))
	}
	_, cmd := enter(m) // last group: submit
	got, ok := cmd().(SubmittedMsg)
	if !ok {
		t.Fatalf("Enter on the last group produced %T", cmd())
	}
	if _, has := got.Values["card"]; has {
		t.Errorf("hidden group's field is in the values: %v", got.Values)
	}
	for _, k := range []string{"name", "email", "plan", "terms"} {
		if _, has := got.Values[k]; !has {
			t.Errorf("Values lacks %q: %v", k, got.Values)
		}
	}
}

func TestShownGroupIsCheckedAndReached(t *testing.T) {
	m := fillAccount(wizardForm())
	m, _ = m.Update(tui.Key{Type: tui.KeyRight}) // Free -> Pro: Billing applies
	if m.GroupCount() != 4 {
		t.Fatalf("GroupCount = %d, want 4", m.GroupCount())
	}
	m, _ = enter(m)
	if m.CurrentGroup() != 2 || m.Current() != "card" {
		t.Fatalf("Enter on Plan went to group %d field %q", m.CurrentGroup(), m.Current())
	}
}

func TestSubmitJumpsToTheGroupWithTheFirstError(t *testing.T) {
	m := wizardForm()
	m, cmd := m.Submit()
	if cmd != nil {
		if _, ok := cmd().(SubmittedMsg); ok {
			t.Fatal("an invalid form submitted")
		}
	}
	if m.Err("name") == "" || m.Err("email") == "" {
		t.Error("Submit did not check the account group")
	}
	m = fillAccount(m)
	m, _ = m.Update(tui.Key{Type: tui.KeyRight}) // Pro
	m, _ = enter(m)                              // Billing, card empty
	m, _ = enter(m)                              // invalid: stays
	m = pgup(m)
	m = pgup(m) // back on Account, now valid
	m, cmd = m.Submit()
	if m.CurrentGroup() != 2 || m.Current() != "card" || cmd != nil && func() bool { _, ok := cmd().(SubmittedMsg); return ok }() {
		t.Errorf("Submit left group %d field %q, want Billing's card", m.CurrentGroup(), m.Current())
	}
}

func TestEditingCanHideTheGroupShowing(t *testing.T) {
	m := NewGroups(
		Group{Fields: []Field{{Name: "mode", Label: "Mode", Kind: FieldSelect, Options: []string{"on", "off"}}}},
		Group{Title: "Extra", Fields: []Field{{Name: "x", Label: "X", Value: "1"}},
			Hide: func(v map[string]string) bool { return v["x"] == "gone" }},
		Group{Title: "Last", Fields: []Field{{Name: "y", Label: "Y"}}},
	)
	m.Focus()
	m, _ = enter(m) // to Extra
	if m.CurrentGroup() != 1 {
		t.Fatalf("group %d", m.CurrentGroup())
	}
	m = typed(m, "gone") // x becomes "1gone", not hidden
	if m.CurrentGroup() != 1 {
		t.Fatalf("group %d after typing", m.CurrentGroup())
	}
	for range 5 {
		m, _ = m.Update(tui.Key{Type: tui.KeyBackspace})
	}
	m = typed(m, "gone")
	if m.CurrentGroup() != 2 || m.Current() != "y" {
		t.Errorf("hid the group showing: now group %d field %q", m.CurrentGroup(), m.Current())
	}
}

func TestGroupedLinearize(t *testing.T) {
	m := wizardForm()
	got := m.Linearize()
	if !strings.HasPrefix(got, "Group 1 of 3, Account\nWho you are\nName, ") {
		t.Errorf("Linearize =\n%s", got)
	}
	if strings.Contains(got, "Plan") {
		t.Errorf("another group is spoken:\n%s", got)
	}
}

func TestGroupedMouseClickAccountsForTheHeader(t *testing.T) {
	m := wizardForm()
	m.Mouse = true
	m.Bounds = hittest.Rect{X: 0, Y: 0, W: 40, H: 8}
	// Rows: title, description, Name, Email.
	m, _ = m.Update(tui.MouseEvent{X: 3, Y: 3, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if m.Current() != "email" {
		t.Errorf("click on row 3 focused %q, want email", m.Current())
	}
}

func TestGroupedLayoutNodeDrawsTheHeader(t *testing.T) {
	m := wizardForm()
	sz := m.LayoutNode().Measure(layout.Unconstrained())
	if sz.H != 4 { // title, description, two fields
		t.Errorf("Measure H = %d, want 4", sz.H)
	}
	out := strings.Split(plainNode(m, layout.Size{W: 40, H: 4}), "\n")
	if len(out) != 4 || !strings.Contains(out[0], "Account") || !strings.Contains(out[2], "Name") {
		t.Errorf("Render:\n%s", strings.Join(out, "\n"))
	}
}

func TestNewStaysAnUngroupedForm(t *testing.T) {
	m := signup()
	if v := plain(m); strings.Contains(v, "step") || m.GroupCount() != 1 || m.CurrentGroup() != 0 {
		t.Errorf("New form changed:\n%s", v)
	}
	if len(NewGroups().fields) != 0 {
		t.Error("NewGroups() with no groups has fields")
	}
}

func plainNode(m Model, s layout.Size) string {
	return ansi.StripANSI(m.LayoutNode().Render(s))
}
