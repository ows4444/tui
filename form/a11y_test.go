package form

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/textinput"
)

var (
	_ tui.Linearizer = Model{}
	_ layout.Node    = Model{}.LayoutNode()
)

const secretValue = "hunter2-correct-horse"

// assertNoSecret is the check every accessible or layout output of a form
// holding secretValue must pass.
func assertNoSecret(t testing.TB, what, out string) {
	t.Helper()
	for _, leak := range []string{"hunter2", "correct-horse", secretValue} {
		if strings.Contains(ansi.StripANSI(out), leak) {
			t.Errorf("%s leaked the secret (%q): %q", what, leak, out)
		}
	}
}

func filledSignup() Model {
	m := signup()
	m = typed(m, "Ada")
	m = tab(m)
	m = typed(m, "not-an-email")
	m = tab(m)
	m = typed(m, secretValue)
	m, _ = m.Submit() // email is invalid, so errors are showing
	return m
}

// Linearize speaks each field's label, its value and its error, and never a
// secret's value.
func TestLinearizeSpeaksLabelValueAndErrorButNotTheSecret(t *testing.T) {
	m := filledSignup()
	got := m.Linearize()
	lines := strings.Split(got, "\n")
	if len(lines) != 3 {
		t.Fatalf("%d lines, want one per field: %q", len(lines), got)
	}
	if !strings.Contains(lines[0], "Name") || !strings.Contains(lines[0], "Ada") {
		t.Errorf("name line = %q, want the label and value", lines[0])
	}
	if !strings.Contains(lines[1], "Email") || !strings.Contains(lines[1], "not-an-email") ||
		!strings.Contains(lines[1], "error: "+m.Err("email")) {
		t.Errorf("email line = %q, want label, value and its error", lines[1])
	}
	if !strings.Contains(lines[2], "Password") || !strings.Contains(lines[2], "characters entered") {
		t.Errorf("password line = %q, want the label and a character count", lines[2])
	}
	assertNoSecret(t, "Linearize", got)
}

func TestLinearizeMarksTheFocusedFieldAndOmitsErrorsUntilSubmit(t *testing.T) {
	m := signup()
	m = typed(m, "x")
	lines := strings.Split(m.Linearize(), "\n")
	if !strings.Contains(lines[0], "focused") || strings.Contains(lines[1], "focused") {
		t.Errorf("focus marks wrong: %q", lines)
	}
	if strings.Contains(m.Linearize(), "error") {
		t.Errorf("an error was spoken before Submit: %q", m.Linearize())
	}
}

// The secret is only safe because the field is a passwordinput, whose
// Linearize overrides the promoted one. A text field holding the same value
// would speak it, and assertNoSecret catches that.
func TestTheSecretCheckWouldCatchALeakingLinearize(t *testing.T) {
	leaky := textinput.New()
	leaky.Prompt = "Password: "
	leaky.SetValue(secretValue)
	inner := &leakCheck{}
	assertNoSecret(inner, "text field Linearize", leaky.Linearize())
	if !inner.failed {
		t.Fatal("assertNoSecret did not flag a text field that speaks its value")
	}

	// The form's Secret field goes through passwordinput, which does not.
	assertNoSecret(t, "form Linearize", filledSignup().Linearize())
}

// leakCheck records a failure instead of failing the real test.
type leakCheck struct {
	testing.TB
	failed bool
}

func (l *leakCheck) Helper()               {}
func (l *leakCheck) Errorf(string, ...any) { l.failed = true }

func TestLayoutNodeRendersExactlyTheRequestedSize(t *testing.T) {
	m := filledSignup()
	sizes := []layout.Size{
		{W: 5, H: 3}, {W: 12, H: 1}, {W: 20, H: 2}, {W: 40, H: 10}, {W: 40, H: 5},
		{W: 80, H: 24}, {W: 300, H: 80}, {W: 5, H: 80}, {W: 300, H: 3},
	}
	for _, s := range sizes {
		lines := strings.Split(m.LayoutNode().Render(s), "\n")
		if len(lines) != s.H {
			t.Fatalf("%v: %d rows, want %d", s, len(lines), s.H)
		}
		for i, l := range lines {
			if w := ansi.Width(l); w != s.W {
				t.Errorf("%v: row %d has width %d (%q)", s, i, w, l)
			}
		}
	}
	if out := m.LayoutNode().Render(layout.Size{}); out != "" {
		t.Errorf("zero size rendered %q", out)
	}
}

func TestLayoutNodeNeverDrawsTheSecret(t *testing.T) {
	m := filledSignup()
	for _, s := range []layout.Size{{W: 40, H: 10}, {W: 30, H: 4}, {W: 300, H: 80}} {
		assertNoSecret(t, "LayoutNode", m.LayoutNode().Render(s))
	}
}

func TestLayoutNodeMeasuresRowsForFieldsAndErrors(t *testing.T) {
	m := signup()
	if got := m.LayoutNode().Measure(layout.Unconstrained()); got.H != 3 {
		t.Errorf("3 fields without errors measure %d rows, want 3", got.H)
	}
	m, _ = m.Submit() // all three fail Required
	if got := m.LayoutNode().Measure(layout.Unconstrained()); got.H != 6 {
		t.Errorf("3 fields with errors measure %d rows, want 6", got.H)
	}
}

// When the form is taller than the space, the window follows the focused
// field instead of clipping to the top.
func TestLayoutNodeKeepsTheFocusedFieldVisible(t *testing.T) {
	m := signup() // 3 rows, no errors
	for i, want := range []string{"Name", "Email", "Password"} {
		out := ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 40, H: 1}))
		if !strings.Contains(out, want) {
			t.Errorf("focus on field %d: a 1-row window shows %q, want %q", i, out, want)
		}
		m = tab(m)
	}
}

func TestLayoutNodeDoesNotChangeTheModel(t *testing.T) {
	m := filledSignup()
	before := m.View()
	_ = m.LayoutNode().Render(layout.Size{W: 20, H: 2})
	if m.View() != before {
		t.Error("rendering the layout node changed the model")
	}
}
