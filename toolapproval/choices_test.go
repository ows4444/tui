package toolapproval

import (
	"context"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/layout"
)

func rune1(s string) tui.Key { return tui.Key{Type: tui.KeyRunes, Text: s} }

// twoChoice is the prompt an app with no rules yet shows: Allow and Deny,
// answered with y and n, with the command as the body.
func twoChoice() Model {
	m := New("Bash", "wants to run", RiskLow)
	m.Choices = []Choice{ChoiceApprove, ChoiceDeny}
	m.Labels = map[Choice]string{ChoiceApprove: "Allow"}
	m.Body = "$ go test ./...\nin kaatib/backend"
	m.KeyMap.Approve = keymap.NewBinding("allow", "y")
	m.KeyMap.Deny = keymap.NewBinding("deny", "n")
	m.KeyMap.Always = keymap.NewBinding("always", "a")
	return m
}

func resolvedChoice(t *testing.T, cmd tui.Cmd) Choice {
	t.Helper()
	if cmd == nil {
		t.Fatal("no command")
	}
	r, ok := tui.RunCmd(context.Background(), cmd).(ResolvedMsg)
	if !ok {
		t.Fatalf("command produced %T, want ResolvedMsg", tui.RunCmd(context.Background(), cmd))
	}
	return r.Choice
}

func TestChoicesLimitWhatIsShownAndCycled(t *testing.T) {
	m := twoChoice()
	view := ansi.StripANSI(m.View())
	if strings.Contains(view, "Always Allow") || strings.Contains(view, "Approve") {
		t.Errorf("View shows a hidden option or the default label:\n%s", view)
	}
	if !strings.HasSuffix(view, "[Allow]   Deny ") {
		t.Errorf("options row = %q", view[strings.LastIndex(view, "\n")+1:])
	}
	m, _ = m.Update(key(tui.KeyRight))
	if m.Highlighted() != ChoiceDeny {
		t.Fatalf("Right: Highlighted() = %v, want Deny", m.Highlighted())
	}
	m, _ = m.Update(key(tui.KeyRight))
	if m.Highlighted() != ChoiceApprove {
		t.Fatalf("Right wraps over two options: Highlighted() = %v, want Approve", m.Highlighted())
	}
	m, _ = m.Update(key(tui.KeyLeft))
	if m.Highlighted() != ChoiceDeny {
		t.Fatalf("Left wraps over two options: Highlighted() = %v, want Deny", m.Highlighted())
	}
	_, cmd := m.Update(key(tui.KeyEnter))
	if got := resolvedChoice(t, cmd); got != ChoiceDeny {
		t.Errorf("Enter resolved %v, want Deny", got)
	}
}

func TestChoicesOrderAndBadValues(t *testing.T) {
	m := New("rm", "", RiskLow)
	m.Choices = []Choice{ChoiceDeny, Choice(7), ChoiceDeny, Choice(-1), ChoiceApprove}
	if got := ansi.StripANSI(m.View()); !strings.HasSuffix(got, " Deny   [Approve]") {
		t.Errorf("options row = %q, want Deny then the highlighted Approve", got[strings.LastIndex(got, "\n")+1:])
	}
	m.Choices = []Choice{Choice(9)}
	if got := m.Linearize(); !strings.Contains(got, "of 3") {
		t.Errorf("only invalid Choices should show all three:\n%s", got)
	}
}

func TestHighlightFallsToFirstShownOption(t *testing.T) {
	m := New("rm", "", RiskLow) // Approve highlighted
	m.Choices = []Choice{ChoiceDeny, ChoiceAlwaysAllow}
	if m.Highlighted() != ChoiceDeny {
		t.Fatalf("Highlighted() = %v, want the first shown option", m.Highlighted())
	}
	_, cmd := m.Update(key(tui.KeyEnter))
	if got := resolvedChoice(t, cmd); got != ChoiceDeny {
		t.Errorf("Enter resolved %v, want Deny", got)
	}
}

func TestDirectKeysResolveAtOnce(t *testing.T) {
	for k, want := range map[string]Choice{"y": ChoiceApprove, "n": ChoiceDeny} {
		m := twoChoice()
		m.Start()
		next, cmd := m.Update(rune1(k))
		if got := resolvedChoice(t, cmd); got != want {
			t.Errorf("%q resolved %v, want %v", k, got, want)
		}
		if next.Highlighted() != want {
			t.Errorf("%q: Highlighted() = %v, want %v", k, next.Highlighted(), want)
		}
		if _, again := next.Update(timeoutMsg{id: 1}); again != nil {
			t.Errorf("%q: the timeout resolved the prompt a second time", k)
		}
	}
}

func TestDirectKeyOfHiddenOptionDoesNothing(t *testing.T) {
	m := twoChoice() // Always is bound to "a" but not shown
	next, cmd := m.Update(rune1("a"))
	if cmd != nil {
		t.Fatal("a key for an option that is not shown resolved the prompt")
	}
	if next.Highlighted() != ChoiceApprove {
		t.Errorf("Highlighted() moved to %v", next.Highlighted())
	}
}

func TestDirectKeysAreUnboundByDefault(t *testing.T) {
	m := New("rm", "", RiskLow)
	for _, k := range []string{"y", "n", "a"} {
		if _, cmd := m.Update(rune1(k)); cmd != nil {
			t.Errorf("%q resolved a prompt that binds no direct key", k)
		}
	}
	for _, b := range []keymap.Binding{m.KeyMap.Approve, m.KeyMap.Deny, m.KeyMap.Always} {
		if len(b.Keys) != 0 || b.Desc == "" {
			t.Errorf("default direct binding %+v: want a description and no keys", b)
		}
	}
}

func TestDirectKeyOnStructLiteralKeepsDefaultNavigation(t *testing.T) {
	m := Model{KeyMap: KeyMap{Deny: keymap.Binding{Keys: []string{"n"}}}}
	moved, _ := m.Update(key(tui.KeyRight))
	if moved.Highlighted() != ChoiceDeny {
		t.Fatalf("Right: Highlighted() = %v, want Deny with default navigation", moved.Highlighted())
	}
	_, cmd := m.Update(rune1("n"))
	if got := resolvedChoice(t, cmd); got != ChoiceDeny {
		t.Errorf("n resolved %v, want Deny", got)
	}
	bs := m.Bindings()
	if len(bs) != 4 || bs[3].Desc != "deny" {
		t.Errorf("Bindings() = %+v, want navigation plus a described deny binding", bs)
	}
}

func TestBindingsListDirectKeysOfShownOptions(t *testing.T) {
	bs := twoChoice().Bindings()
	if len(bs) != 5 {
		t.Fatalf("Bindings() has %d entries, want 3 navigation and 2 direct", len(bs))
	}
	if bs[3].Desc != "allow" || bs[4].Desc != "deny" {
		t.Errorf("direct bindings = %+v, %+v", bs[3], bs[4])
	}
}

func TestBodySitsBetweenHeaderAndOptions(t *testing.T) {
	lines := strings.Split(ansi.StripANSI(twoChoice().View()), "\n")
	want := []string{"", "wants to run", "$ go test ./...", "in kaatib/backend", "[Allow]   Deny "}
	if len(lines) != len(want) {
		t.Fatalf("View has %d lines, want %d:\n%s", len(lines), len(want), strings.Join(lines, "\n"))
	}
	if !strings.HasPrefix(lines[0], "Bash") || !strings.Contains(lines[0], "LOW") {
		t.Errorf("line 0 = %q, want the tool name and its risk badge", lines[0])
	}
	for i := range want {
		if i == 0 {
			continue
		}
		if strings.TrimRight(lines[i], " ") != strings.TrimRight(want[i], " ") {
			t.Errorf("line %d = %q, want %q", i, lines[i], want[i])
		}
	}
	if got := ansi.StripANSI(New("ls", "", RiskLow).View()); strings.Count(got, "\n") != 1 {
		t.Errorf("no Body and no Description should give two lines:\n%s", got)
	}
}

func TestLinearizeWithChoicesLabelsAndBody(t *testing.T) {
	m := twoChoice()
	m.Body = "\x1b[1m$ go test ./...\x1b[0m"
	want := "Approval needed: Bash, low risk\n" +
		"wants to run\n" +
		"$ go test ./...\n" +
		"Allow, option 1 of 2, selected\n" +
		"Deny, option 2 of 2"
	if got := m.Linearize(); got != want {
		t.Errorf("Linearize =\n%s\nwant\n%s", got, want)
	}
}

// Options row is row 4 (name, description, two body lines): "[Allow]" 0..6,
// " Deny " 9..14.
func TestClickFollowsBodyLabelsAndChoices(t *testing.T) {
	base := twoChoice()
	base.Mouse = true
	base.Bounds = hittest.Rect{X: 2, Y: 1, W: 40, H: 5}
	for _, tc := range []struct {
		x    int
		want Choice
	}{{0, ChoiceApprove}, {6, ChoiceApprove}, {9, ChoiceDeny}, {14, ChoiceDeny}} {
		_, cmd := base.Update(ev(2+tc.x, 1+4, tui.MouseButtonLeft, tui.MouseActionPress))
		if got := resolvedChoice(t, cmd); got != tc.want {
			t.Errorf("x=%d resolved %v, want %v", tc.x, got, tc.want)
		}
	}
	for name, e := range map[string]tui.MouseEvent{
		"old options row": ev(2+1, 1+2, tui.MouseButtonLeft, tui.MouseActionPress),
		"past last label": ev(2+17, 1+4, tui.MouseButtonLeft, tui.MouseActionPress),
		"gap":             ev(2+7, 1+4, tui.MouseButtonLeft, tui.MouseActionPress),
	} {
		if _, cmd := base.Update(e); cmd != nil {
			t.Errorf("%s: produced a command", name)
		}
	}
}

func TestLayoutNodeWrapsBody(t *testing.T) {
	m := New("Bash", "", RiskLow)
	m.Body = "one two three four five six seven eight nine ten"
	got := ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 20, H: 6}))
	for _, l := range strings.Split(got, "\n") {
		if ansi.Width(l) > 20 {
			t.Errorf("line wider than 20 cells: %q", l)
		}
	}
	if !strings.Contains(got, "ten") {
		t.Errorf("wrapped body lost text:\n%s", got)
	}
}

func TestUpdateDoesNotChangeChoicesOrLabels(t *testing.T) {
	m := twoChoice()
	for _, k := range []tui.Msg{key(tui.KeyRight), key(tui.KeyLeft), rune1("y"), key(tui.KeyEnter)} {
		m, _ = m.Update(k)
	}
	if len(m.Choices) != 2 || m.Choices[0] != ChoiceApprove || m.Choices[1] != ChoiceDeny || m.Labels[ChoiceApprove] != "Allow" || len(m.Labels) != 1 {
		t.Errorf("Update changed the caller's Choices or Labels: %v %v", m.Choices, m.Labels)
	}
}
